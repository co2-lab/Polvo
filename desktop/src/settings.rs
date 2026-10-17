// @anchors
//   code: STIMS
//   ref: STTNG

//! The user's settings, kept in a file (STTNG).

use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

use crate::layout::write_at_once; // @dep: DLIDC

/// What the user changed.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Settings {
    /// The shortcuts that differ from the defaults, by action id.
    pub keys: BTreeMap<String, String>,
}

/// The file of the settings; the application's, as a global, is where the settings tab saves.
#[derive(Clone, Debug)]
pub struct SettingsFile {
    path: PathBuf,
}

impl gpui_kit::Global for SettingsFile {}

impl SettingsFile {
    /// The file of this installation: `polvo/settings.json` in the user's configuration directory
    /// (STTNG-B01); none when the system gives no such directory.
    pub fn default() -> Option<SettingsFile> {
        Some(SettingsFile::at(dirs::config_dir()?.join("polvo").join("settings.json")))
    }

    /// The file at `path`.
    pub fn at(path: PathBuf) -> SettingsFile {
        SettingsFile { path }
    }

    /// Where the file is.
    pub fn path(&self) -> &Path {
        &self.path
    }

    /// The settings kept in the file (STTNG-B02); with no file, none changed; with a file that is not
    /// settings, none changed and a warning (STTNG-E01).
    pub fn read(&self) -> Settings {
        let Ok(text) = std::fs::read_to_string(&self.path) else { return Settings::default() };
        match parse(&text) {
            Ok(keys) => Settings { keys },
            Err(e) => {
                // STTNG-E01
                eprintln!("polvo: {} is not settings ({e}); starting with the defaults", self.path.display());
                Settings::default()
            }
        }
    }

    /// Writes the settings at once (STTNG-B03); the error when they cannot be written (STTNG-E02).
    pub fn write(&self, s: &Settings) -> std::io::Result<()> {
        let text = serde_json::to_string_pretty(&serde_json::json!({ "keys": s.keys })).map_err(std::io::Error::other)?;
        if let Err(e) = write_at_once(&self.path, &text) {
            // STTNG-E02: the error goes back to whoever saved, and the old file stays.
            return Err(e);
        }
        Ok(())
    }
}

/// The shortcuts of a settings file's text.
fn parse(text: &str) -> Result<BTreeMap<String, String>, String> {
    let v: serde_json::Value = serde_json::from_str(text).map_err(|e| e.to_string())?;
    let keys = v.get("keys").and_then(|k| k.as_object()).ok_or("no \"keys\" object")?;
    keys.iter()
        .map(|(k, v)| v.as_str().map(|s| (k.clone(), s.to_string())).ok_or(format!("the shortcut of {k} is not text")))
        .collect()
}

#[cfg(test)]
#[path = "settings_test.rs"]
mod tests;
