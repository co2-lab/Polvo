// @anchors
//   code: RMIMP
//   ref: RMTEC

//! The machines and their terminal sessions, through Polvo's library (RMTEC): the Go bridge linked into
//! this executable as a C library (`libpolvo`).

use std::ffi::{CStr, CString, c_char, c_int};
use std::path::Path;
use std::sync::Arc;

use serde_json::{Value, json};

/// The name of this window's machine in the library.
pub const LOCAL: &str = "local";

unsafe extern "C" {
    fn polvo_open(dir: *const c_char, listen: *const c_char, shell: *const c_char, err: *mut *mut c_char) -> i64;
    fn polvo_close(h: i64);
    fn polvo_call(h: i64, req: *const c_char, err: *mut *mut c_char) -> *mut c_char;
    fn polvo_attach(h: i64, machine: *const c_char, id: *const c_char, err: *mut *mut c_char) -> i64;
    fn polvo_read(t: i64, buf: *mut u8, n: c_int, err: *mut *mut c_char) -> c_int;
    fn polvo_write(t: i64, buf: *const u8, n: c_int, err: *mut *mut c_char) -> c_int;
    fn polvo_resize(t: i64, rows: u16, cols: u16, err: *mut *mut c_char) -> c_int;
    fn polvo_detach(t: i64);
    fn polvo_free(p: *mut c_char);
}

/// A refusal of the library, with its text (RMTEC-E01).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct RemoteError(pub String);

impl std::fmt::Display for RemoteError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

/// A machine the window reaches.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Machine {
    pub name: String,
    pub local: bool,
}

/// A live session of a machine (RMTEC-B02).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SessionInfo {
    pub id: String,
    pub argv: Vec<String>,
    pub rows: u16,
    pub cols: u16,
    pub viewers: u32,
    pub locked: bool,
    /// Whether this window may type in it.
    pub writable: bool,
}

/// Where this machine's node listens, or why it does not.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Status {
    pub listen: String,
    pub listen_error: Option<String>,
    /// This installation's key, as the line another machine adds to its `authorized_keys` (RMTEC-B04).
    pub key: String,
}

fn c(s: &str) -> CString {
    CString::new(s.replace('\0', "")).unwrap_or_default()
}

/// Takes a text the library gave, freeing it.
unsafe fn take(p: *mut c_char) -> String {
    if p.is_null() {
        return String::new();
    }
    // SAFETY: the library gives NUL-terminated UTF-8 that this side frees once.
    let s = unsafe { CStr::from_ptr(p) }.to_string_lossy().into_owned();
    unsafe { polvo_free(p) };
    s
}

/// The text the library gave, or its error when it gave none (RMTEC-E01).
fn answered(out: *mut c_char, err: *mut c_char) -> Result<String, RemoteError> {
    if out.is_null() {
        return Err(RemoteError(unsafe { take(err) }));
    }
    Ok(unsafe { take(out) })
}

/// The installation, with this machine's node (RMTEC-B01); the clones share it, and the last one closes it.
#[derive(Clone)]
pub struct Station {
    h: Arc<Handle>,
}

struct Handle(i64);

impl Drop for Handle {
    fn drop(&mut self) {
        // SAFETY: the number is the library's, closed once.
        unsafe { polvo_close(self.0) }
    }
}

impl Station {
    /// Opens the installation of `dir`, listening on `listen` (empty: not), with `shell` for new sessions.
    pub fn open(dir: &Path, listen: &str, shell: &str) -> Result<Station, RemoteError> {
        let (d, l, s) = (c(&dir.to_string_lossy()), c(listen), c(shell));
        let mut err = std::ptr::null_mut();
        // SAFETY: NUL-terminated texts, and an out-parameter the library fills or leaves null.
        let h = unsafe { polvo_open(d.as_ptr(), l.as_ptr(), s.as_ptr(), &mut err) };
        if h == 0 {
            return Err(RemoteError(unsafe { take(err) }));
        }
        Ok(Station { h: Arc::new(Handle(h)) })
    }

    fn call(&self, req: Value) -> Result<Value, RemoteError> {
        let r = c(&req.to_string());
        let mut err = std::ptr::null_mut();
        // SAFETY: as in `open`.
        let out = unsafe { polvo_call(self.h.0, r.as_ptr(), &mut err) };
        let text = match answered(out, err) {
            // RMTEC-E01: the library's text goes to whoever asked
            Err(e) => return Err(e),
            Ok(text) => text,
        };
        serde_json::from_str(&text).map_err(|e| RemoteError(format!("polvo: the library answered {text:?}: {e}")))
    }

    /// Where this machine's node listens (RMTEC-B01).
    pub fn status(&self) -> Result<Status, RemoteError> {
        let v = self.call(json!({ "op": "status" }))?;
        Ok(Status {
            listen: v["listen"].as_str().unwrap_or_default().to_string(),
            listen_error: v["listen_error"].as_str().map(str::to_string),
            key: v["key"].as_str().unwrap_or_default().to_string(),
        })
    }

    /// This window's machine first, then the known ones (RMTEC-B01).
    pub fn machines(&self) -> Result<Vec<Machine>, RemoteError> {
        let v = self.call(json!({ "op": "machines" }))?;
        Ok(v["machines"]
            .as_array()
            .into_iter()
            .flatten()
            .map(|m| Machine { name: m["name"].as_str().unwrap_or_default().to_string(), local: m["local"].as_bool().unwrap_or(false) })
            .collect())
    }

    /// Connects to the machine at `addr` and keeps it (RMTEC-B04).
    pub fn add(&self, addr: &str) -> Result<(), RemoteError> {
        self.call(json!({ "op": "add", "addr": addr })).map(|_| ())
    }

    /// Forgets a machine (RMTEC-B04).
    pub fn remove(&self, name: &str) -> Result<(), RemoteError> {
        self.call(json!({ "op": "remove", "name": name })).map(|_| ())
    }

    /// The live sessions of a machine (RMTEC-B02).
    pub fn sessions(&self, machine: &str) -> Result<Vec<SessionInfo>, RemoteError> {
        let v = self.call(json!({ "op": "sessions", "machine": machine }))?;
        Ok(v["sessions"]
            .as_array()
            .into_iter()
            .flatten()
            .map(|s| SessionInfo {
                id: s["id"].as_str().unwrap_or_default().to_string(),
                argv: s["argv"].as_array().into_iter().flatten().filter_map(|a| a.as_str().map(str::to_string)).collect(),
                rows: s["rows"].as_u64().unwrap_or(0) as u16,
                cols: s["cols"].as_u64().unwrap_or(0) as u16,
                viewers: s["viewers"].as_u64().unwrap_or(0) as u32,
                locked: s["locked"].as_bool().unwrap_or(false),
                writable: s["writable"].as_bool().unwrap_or(false),
            })
            .collect())
    }

    /// Makes a session of the installation's shell and gives its id (RMTEC-B02).
    pub fn new_session(&self, machine: &str, rows: u16, cols: u16) -> Result<String, RemoteError> {
        let v = self.call(json!({ "op": "new", "machine": machine, "rows": rows, "cols": cols }))?;
        Ok(v["id"].as_str().unwrap_or_default().to_string())
    }

    /// Locks a session for viewing only, or unlocks it (RMTEC-B02).
    pub fn lock(&self, machine: &str, id: &str, locked: bool) -> Result<(), RemoteError> {
        self.call(json!({ "op": "lock", "machine": machine, "id": id, "locked": locked })).map(|_| ())
    }

    /// Ends a session (RMTEC-B02).
    pub fn close_session(&self, machine: &str, id: &str) -> Result<(), RemoteError> {
        self.call(json!({ "op": "close", "machine": machine, "id": id })).map(|_| ())
    }

    /// A terminal of a session (RMTEC-B03).
    pub fn attach(&self, machine: &str, id: &str) -> Result<RemoteTerm, RemoteError> {
        let (m, i) = (c(machine), c(id));
        let mut err = std::ptr::null_mut();
        // SAFETY: as in `open`.
        let t = unsafe { polvo_attach(self.h.0, m.as_ptr(), i.as_ptr(), &mut err) };
        if t == 0 {
            return Err(RemoteError(unsafe { take(err) }));
        }
        Ok(RemoteTerm { t, _station: self.clone() })
    }
}

/// A terminal of a session: read, written and resized like a pty; dropping it leaves the session
/// alive (RMTEC-B03). Its calls may come from any thread.
pub struct RemoteTerm {
    t: i64,
    _station: Station,
}

impl RemoteTerm {
    /// What the session shows, as it comes; zero at the end.
    pub fn read(&self, buf: &mut [u8]) -> Result<usize, RemoteError> {
        let mut err = std::ptr::null_mut();
        let n = buf.len().min(c_int::MAX as usize) as c_int;
        // SAFETY: the library writes at most `n` bytes into `buf`.
        let k = unsafe { polvo_read(self.t, buf.as_mut_ptr(), n, &mut err) };
        if k < 0 {
            return Err(RemoteError(unsafe { take(err) }));
        }
        Ok(k as usize)
    }

    /// Types `buf` in the session.
    pub fn write(&self, buf: &[u8]) -> Result<usize, RemoteError> {
        let mut err = std::ptr::null_mut();
        let n = buf.len().min(c_int::MAX as usize) as c_int;
        // SAFETY: the library reads `n` bytes of `buf`.
        let k = unsafe { polvo_write(self.t, buf.as_ptr(), n, &mut err) };
        if k < 0 {
            return Err(RemoteError(unsafe { take(err) }));
        }
        Ok(k as usize)
    }

    /// Changes the session's size.
    pub fn resize(&self, rows: u16, cols: u16) -> Result<(), RemoteError> {
        let mut err = std::ptr::null_mut();
        // SAFETY: as in `open`.
        if unsafe { polvo_resize(self.t, rows, cols, &mut err) } < 0 {
            return Err(RemoteError(unsafe { take(err) }));
        }
        Ok(())
    }
}

impl Drop for RemoteTerm {
    fn drop(&mut self) {
        // SAFETY: the number is the library's, let go once.
        unsafe { polvo_detach(self.t) }
    }
}

#[cfg(test)]
#[path = "remote_test.rs"]
mod tests;
