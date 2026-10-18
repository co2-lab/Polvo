// @anchors
//   code: P2FFI
//   ref: PTRNS

//! The C API of the transport (PTRNS-B04), for the node in Go (cgo): endpoints and streams are numbers,
//! every call blocks until it is done (Go calls each from its own goroutine), and the text a call gives
//! back is freed with `polvo_p2p_free`.

use std::collections::HashMap;
use std::ffi::{CStr, CString, c_char};
use std::sync::atomic::{AtomicI64, Ordering};
use std::sync::{Arc, LazyLock, Mutex};

use iroh::endpoint::{Connection, RecvStream, SendStream};
use tokio::runtime::Runtime;

use crate::{CLOSE_WAIT, Peer, Relay};

static RUNTIME: LazyLock<Runtime> =
    LazyLock::new(|| tokio::runtime::Builder::new_multi_thread().enable_all().build().expect("p2p: no runtime"));

static NEXT: AtomicI64 = AtomicI64::new(1);

/// A stream's two halves apart, for a read and a write to run at once.
struct Halves {
    send: Arc<tokio::sync::Mutex<SendStream>>,
    recv: Arc<tokio::sync::Mutex<RecvStream>>,
    // the stream lives while its connection does
    conn: Connection,
}

static PEERS: LazyLock<Mutex<HashMap<i64, Peer>>> = LazyLock::new(Default::default);
static STREAMS: LazyLock<Mutex<HashMap<i64, Halves>>> = LazyLock::new(Default::default);

fn next() -> i64 {
    NEXT.fetch_add(1, Ordering::Relaxed)
}

/// Gives `text` to C, for `polvo_p2p_free`.
fn give(text: String) -> *mut c_char {
    CString::new(text.replace('\0', "")).unwrap_or_default().into_raw()
}

/// Puts `why` in `err`, when there is where; the failure's number.
unsafe fn fail(err: *mut *mut c_char, why: String) -> i64 {
    if !err.is_null() {
        // SAFETY: the caller gives a place for one pointer, or null.
        unsafe { *err = give(why) };
    }
    -1
}

unsafe fn text<'a>(s: *const c_char) -> &'a str {
    if s.is_null() {
        return "";
    }
    // SAFETY: the caller gives a NUL-terminated string that lives during the call.
    unsafe { CStr::from_ptr(s) }.to_str().unwrap_or_default()
}

fn keep(s: crate::Stream) -> (i64, String) {
    let n = next();
    let halves = Halves {
        send: Arc::new(tokio::sync::Mutex::new(s.send)),
        recv: Arc::new(tokio::sync::Mutex::new(s.recv)),
        conn: s._conn,
    };
    STREAMS.lock().unwrap().insert(n, halves);
    (n, s.remote)
}

/// Opens the endpoint of the key whose secret is the 32 bytes at `seed`, with the relay "default", ""
/// (none) or a URL; its number, or -1 and why in `err`.
///
/// # Safety
/// `seed` points to 32 bytes; `relay` is a C string or null; `err` is null or a place for a pointer.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn polvo_p2p_open(seed: *const u8, relay: *const c_char, err: *mut *mut c_char) -> i64 {
    if seed.is_null() {
        return unsafe { fail(err, "p2p: no key".into()) };
    }
    let mut key = [0u8; 32];
    // SAFETY: the caller gives 32 bytes.
    key.copy_from_slice(unsafe { std::slice::from_raw_parts(seed, 32) });
    let relay = Relay::parse(unsafe { text(relay) });
    match RUNTIME.block_on(Peer::bind(key, relay)) {
        Ok(peer) => {
            let n = next();
            PEERS.lock().unwrap().insert(n, peer);
            n
        }
        Err(e) => unsafe { fail(err, e) },
    }
}

fn peer(h: i64) -> Option<Peer> {
    PEERS.lock().unwrap().get(&h).cloned()
}

/// Where the endpoint is reached, as `Peer::addr` gives it; null for an endpoint that is not open.
#[unsafe(no_mangle)]
pub extern "C" fn polvo_p2p_addr(h: i64) -> *mut c_char {
    peer(h).map_or(std::ptr::null_mut(), |p| give(p.addr()))
}

/// The key of the endpoint; null for an endpoint that is not open.
#[unsafe(no_mangle)]
pub extern "C" fn polvo_p2p_id(h: i64) -> *mut c_char {
    peer(h).map_or(std::ptr::null_mut(), |p| give(p.id()))
}

/// Waits for the next connection of another installation: the stream's number and the other's key in
/// `remote`; 0 when the endpoint closed; -1 and why in `err`.
///
/// # Safety
/// `remote` and `err` are null or places for a pointer.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn polvo_p2p_accept(h: i64, remote: *mut *mut c_char, err: *mut *mut c_char) -> i64 {
    let Some(p) = peer(h) else { return 0 };
    match RUNTIME.block_on(p.accept()) {
        None => 0,
        Some(Ok(s)) => {
            let (n, who) = keep(s);
            if !remote.is_null() {
                // SAFETY: a place for one pointer.
                unsafe { *remote = give(who) };
            }
            n
        }
        Some(Err(e)) => unsafe { fail(err, e) },
    }
}

/// Connects to the installation at `addr`: the stream's number, or -1 and why in `err`.
///
/// # Safety
/// `addr` is a C string; `err` is null or a place for a pointer.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn polvo_p2p_connect(h: i64, addr: *const c_char, err: *mut *mut c_char) -> i64 {
    let Some(p) = peer(h) else { return unsafe { fail(err, "p2p: the endpoint is closed".into()) } };
    let addr = unsafe { text(addr) }.to_string();
    match RUNTIME.block_on(p.connect(&addr)) {
        Ok(s) => keep(s).0,
        Err(e) => unsafe { fail(err, e) },
    }
}

fn halves(s: i64) -> Option<(Arc<tokio::sync::Mutex<SendStream>>, Arc<tokio::sync::Mutex<RecvStream>>)> {
    STREAMS.lock().unwrap().get(&s).map(|h| (h.send.clone(), h.recv.clone()))
}

/// Reads up to `len` bytes into `buf`: how many, 0 at the end, -1 when the stream failed or is closed.
///
/// # Safety
/// `buf` points to `len` writable bytes.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn polvo_p2p_read(s: i64, buf: *mut u8, len: usize) -> i64 {
    let Some((_, recv)) = halves(s) else { return -1 };
    // SAFETY: the caller gives `len` writable bytes.
    let out = unsafe { std::slice::from_raw_parts_mut(buf, len) };
    match RUNTIME.block_on(async { recv.lock().await.read(out).await }) {
        Ok(Some(n)) => n as i64,
        Ok(None) => 0,
        Err(_) => -1,
    }
}

/// Writes the `len` bytes at `buf`: how many, or -1 when the stream failed or is closed.
///
/// # Safety
/// `buf` points to `len` readable bytes.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn polvo_p2p_write(s: i64, buf: *const u8, len: usize) -> i64 {
    let Some((send, _)) = halves(s) else { return -1 };
    // SAFETY: the caller gives `len` readable bytes.
    let data = unsafe { std::slice::from_raw_parts(buf, len) };
    match RUNTIME.block_on(async { send.lock().await.write_all(data).await }) {
        Ok(()) => len as i64,
        Err(_) => -1,
    }
}

/// Says that nothing more is written: the other side reads the end.
#[unsafe(no_mangle)]
pub extern "C" fn polvo_p2p_close_write(s: i64) {
    if let Some((send, _)) = halves(s) {
        RUNTIME.block_on(async {
            let _ = send.lock().await.finish();
        });
    }
}

/// Ends the stream both ways; its number no longer counts.
#[unsafe(no_mangle)]
pub extern "C" fn polvo_p2p_close(s: i64) {
    let Some(h) = STREAMS.lock().unwrap().remove(&s) else { return };
    RUNTIME.block_on(async {
        let _ = h.send.lock().await.finish();
        let _ = h.recv.lock().await.stop(0u32.into());
    });
    // the connection closes once the other side has what was written, or after a while
    RUNTIME.spawn(async move {
        let _ = tokio::time::timeout(CLOSE_WAIT, async { h.send.lock().await.stopped().await }).await;
        h.conn.close(0u32.into(), b"closed");
    });
}

/// Ends the endpoint and its connections; its number no longer counts, and a waiting accept gives 0.
#[unsafe(no_mangle)]
pub extern "C" fn polvo_p2p_shutdown(h: i64) {
    let Some(p) = PEERS.lock().unwrap().remove(&h) else { return };
    RUNTIME.block_on(p.close());
}

/// Frees text this API gave.
///
/// # Safety
/// `s` is null or a pointer this API gave, freed once.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn polvo_p2p_free(s: *mut c_char) {
    if !s.is_null() {
        // SAFETY: given by `give`, freed once.
        drop(unsafe { CString::from_raw(s) });
    }
}
