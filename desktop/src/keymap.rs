// @anchors
//   code: KYIMA
//   ref: KYMPE

//! The window's actions and the shortcut of each (KYMPE).

use std::collections::BTreeMap;

/// An action of the window that has a shortcut.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Action {
    NewTab,
    CloseTab,
    Copy,
    Paste,
    Settings,
    Save,
}

impl Action {
    pub const ALL: [Action; 6] = [Action::NewTab, Action::CloseTab, Action::Copy, Action::Paste, Action::Settings, Action::Save];

    /// The id the action is saved under.
    pub fn id(self) -> &'static str {
        match self {
            Action::NewTab => "new-tab",
            Action::CloseTab => "close-tab",
            Action::Copy => "copy",
            Action::Paste => "paste",
            Action::Settings => "settings",
            Action::Save => "save",
        }
    }

    /// The name the user reads.
    pub fn label(self) -> &'static str {
        match self {
            Action::NewTab => "Aba nova",
            Action::CloseTab => "Fechar a aba",
            Action::Copy => "Copiar",
            Action::Paste => "Colar",
            Action::Settings => "Configuração",
            Action::Save => "Gravar",
        }
    }

    pub fn from_id(id: &str) -> Option<Action> {
        Action::ALL.into_iter().find(|a| a.id() == id)
    }

    /// The default shortcut (KYMPE-B01).
    fn default_shortcut(self) -> &'static str {
        DEFAULTS.iter().find(|(a, _)| *a == self).map_or("", |(_, keystroke)| keystroke)
    }
}

impl gpui_kit::Global for Keymap {}

/// Why a shortcut was refused.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum KeymapError {
    /// The shortcut is another action's (KYMPE-E01).
    Taken(Action),
    /// It is not a shortcut (KYMPE-E02).
    Invalid(String),
}

/// The shortcut of each action.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Keymap {
    shortcuts: Vec<(Action, String)>,
}

/// The default shortcuts (KYMPE-B01): those of the Linux terminals; Ctrl with no Shift is the shell's.
pub const DEFAULTS: [(Action, &str); 6] = [
    (Action::NewTab, "ctrl-shift-t"),
    (Action::CloseTab, "ctrl-shift-w"),
    (Action::Copy, "ctrl-shift-c"),
    (Action::Paste, "ctrl-shift-v"),
    (Action::Settings, "ctrl-,"),
    (Action::Save, "ctrl-s"),
];

const MODIFIERS: [&str; 4] = ["ctrl", "alt", "shift", "cmd"];

/// The one form of a shortcut (KYMPE-B02): the modifiers in a fixed order, lower case; an error when
/// it is not a shortcut (KYMPE-E02).
pub fn normalize(keystroke: &str) -> Result<String, String> {
    let s = keystroke;
    let lower = s.trim().to_lowercase();
    // the key itself may be a hyphen: `ctrl--`
    let (mods, key) = match lower.strip_suffix("--") {
        Some(rest) => (rest, "-".to_string()),
        None => match lower.rsplit_once('-') {
            Some((m, k)) => (m, k.to_string()),
            None => ("", lower.clone()),
        },
    };
    if key.is_empty() {
        return Err(format!("{s:?} has no key"));
    }
    let mut have = [false; 4];
    for m in mods.split('-').filter(|m| !m.is_empty()) {
        let m = match m {
            "super" | "win" | "meta" => "cmd",
            "control" => "ctrl",
            "option" => "alt",
            m => m,
        };
        match MODIFIERS.iter().position(|x| *x == m) {
            Some(i) => have[i] = true,
            None => return Err(format!("{s:?}: unknown modifier {m:?}")),
        }
    }
    if MODIFIERS.contains(&key.as_str()) {
        return Err(format!("{s:?} has only modifiers"));
    }
    let mut out: Vec<&str> = MODIFIERS.iter().zip(have).filter(|(_, h)| *h).map(|(m, _)| *m).collect();
    out.push(&key);
    Ok(out.join("-"))
}

impl Default for Keymap {
    fn default() -> Self {
        Keymap { shortcuts: Action::ALL.iter().map(|a| (*a, a.default_shortcut().to_string())).collect() }
    }
}

impl Keymap {
    /// The shortcut of an action.
    pub fn shortcut(&self, a: Action) -> &str {
        self.shortcuts.iter().find(|(x, _)| *x == a).map_or("", |(_, s)| s.as_str())
    }

    /// The action a shortcut does, if any (KYMPE-B03).
    pub fn action_for(&self, keystroke: &str) -> Option<Action> {
        let keystroke = normalize(keystroke).ok()?;
        self.shortcuts.iter().find(|(_, s)| *s == keystroke).map(|(a, _)| *a)
    }

    /// Gives an action a new shortcut (KYMPE-B04).
    pub fn set(&mut self, a: Action, keystroke: &str) -> Result<(), KeymapError> {
        let keystroke = match normalize(keystroke) {
            Ok(k) => k,
            // KYMPE-E02
            Err(e) => return Err(KeymapError::Invalid(e)),
        };
        if let Some(other) = self.action_for(&keystroke).filter(|o| *o != a) {
            // KYMPE-E01
            return Err(KeymapError::Taken(other));
        }
        if let Some(entry) = self.shortcuts.iter_mut().find(|(x, _)| *x == a) {
            entry.1 = keystroke;
        }
        Ok(())
    }

    /// The shortcuts that differ from the defaults, by action id (KYMPE-B05).
    pub fn overrides(&self) -> BTreeMap<String, String> {
        self.shortcuts
            .iter()
            .filter(|(a, s)| s != a.default_shortcut())
            .map(|(a, s)| (a.id().to_string(), s.clone()))
            .collect()
    }

    /// The defaults with the saved changes; a change that does not fit is left out with a warning
    /// (KYMPE-E03).
    pub fn with_overrides(overrides: &BTreeMap<String, String>) -> Keymap {
        let mut k = Keymap::default();
        for (id, ks) in overrides {
            let Some(a) = Action::from_id(id) else {
                eprintln!("polvo: the shortcut of an unknown action {id:?} was left out");
                continue;
            };
            if let Err(e) = k.set(a, ks) {
                // KYMPE-E03
                eprintln!("polvo: the shortcut {ks:?} of {id} was left out: {e:?}");
            }
        }
        k
    }
}

#[cfg(test)]
#[path = "keymap_test.rs"]
mod tests;
