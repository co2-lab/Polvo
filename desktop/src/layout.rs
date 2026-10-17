// @anchors
//   code: DLIDC
//   ref: DCLYD

//! The organisation of the dock, kept in a file and brought back (DCLYD).

use std::path::{Path, PathBuf};

pub use gpui_kit::component::dock::DockAreaState;

/// The version of the organisation (DCLYD-E01).
pub const LAYOUT_VERSION: usize = 1;

/// The file of the organisation.
#[derive(Clone, Debug)]
pub struct LayoutFile {
    path: PathBuf,
}

impl LayoutFile {
    /// The file of this installation: `polvo/layout.json` in the user's configuration directory
    /// (DCLYD-B01); none when the system gives no such directory.
    pub fn default() -> Option<LayoutFile> {
        Some(LayoutFile::at(dirs::config_dir()?.join("polvo").join("layout.json")))
    }

    /// The file at `path`.
    pub fn at(path: PathBuf) -> LayoutFile {
        LayoutFile { path }
    }

    /// Where the file is.
    pub fn path(&self) -> &Path {
        &self.path
    }

    /// The organisation kept in the file (DCLYD-B03): none with no file, and none with a warning when
    /// the file is not one or is of another version (DCLYD-E01).
    pub fn read(&self) -> Option<DockAreaState> {
        let text = std::fs::read_to_string(&self.path).ok()?;
        let state: DockAreaState = match serde_json::from_str(&text) {
            Ok(state) => state,
            Err(e) => {
                eprintln!("polvo: {} is not a layout ({e}); starting with the default", self.path.display());
                return None;
            }
        };
        if state.version != Some(LAYOUT_VERSION) {
            eprintln!(
                "polvo: {} is a layout of version {:?}, not {LAYOUT_VERSION}; starting with the default",
                self.path.display(),
                state.version
            );
            return None;
        }
        Some(state)
    }

    /// Writes the organisation: the directory when it is missing, the file beside it, then its name
    /// at once (DCLYD-B02). The error when they cannot be written (DCLYD-E02).
    pub fn write(&self, state: &DockAreaState) -> std::io::Result<()> {
        let text = serde_json::to_string_pretty(state).map_err(std::io::Error::other)?;
        write_at_once(&self.path, &text)
    }
}

/// Writes text to path at once: the directory when it is missing, the file beside it, then its name
/// (DCLYD-B02). The settings file is written the same way.
pub(crate) fn write_at_once(path: &Path, text: &str) -> std::io::Result<()> {
    if let Some(dir) = path.parent() {
        std::fs::create_dir_all(dir)?;
    }
    let beside = path.with_extension("json.tmp");
    std::fs::write(&beside, text)?;
    std::fs::rename(&beside, path)
}

#[cfg(test)]
#[path = "layout_test.rs"]
mod tests;
