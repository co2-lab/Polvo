// @anchors
//   code: KYIMK
//   ref: KYSEA

//! The key the window receives, translated for the terminal (KYSEA).

pub use libghostty_vt::key::{Key, Mods};
use libghostty_vt::Terminal;
use libghostty_vt::key::{Action, Encoder, Event};

/// The modifiers held with a key.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Modifiers {
    pub control: bool,
    pub alt: bool,
    pub shift: bool,
}

/// A key translated for libghostty-vt: the physical key, the modifiers and the unmodified text.
#[derive(Clone, Debug, PartialEq)]
pub struct Input {
    pub key: Key,
    pub mods: Mods,
    pub text: Option<String>,
}

/// Translates the key the window names, with the character it types and the modifiers.
pub fn translate(key: &str, key_char: Option<&str>, mods: Modifiers) -> Input {
    // KYSEA-B04
    let mut m = Mods::empty();
    if mods.shift {
        m |= Mods::SHIFT;
    }
    if mods.control {
        m |= Mods::CTRL;
    }
    if mods.alt {
        m |= Mods::ALT;
    }
    let k = key_of(key);
    // KYSEA-B05: the character it types; without one, the key's own single character; a named key
    // has none.
    let text = match key_char {
        Some(c) if !c.is_empty() && !c.chars().any(char::is_control) => Some(c.to_string()),
        _ if key.chars().count() == 1 => Some(key.to_string()),
        _ => None,
    };
    Input { key: k, mods: m, text }
}

fn key_of(name: &str) -> Key {
    let b = name.as_bytes();
    // KYSEA-B01
    if b.len() == 1 && b[0].is_ascii_lowercase() {
        return Key::try_from(Key::A as u32 + u32::from(b[0] - b'a')).unwrap_or(Key::Unidentified);
    }
    if b.len() == 1 && b[0].is_ascii_digit() {
        return Key::try_from(Key::Digit0 as u32 + u32::from(b[0] - b'0')).unwrap_or(Key::Unidentified);
    }
    match name {
        // KYSEA-B02
        "enter" => Key::Enter,
        "backspace" => Key::Backspace,
        "tab" => Key::Tab,
        "escape" => Key::Escape,
        "space" => Key::Space,
        "up" => Key::ArrowUp,
        "down" => Key::ArrowDown,
        "left" => Key::ArrowLeft,
        "right" => Key::ArrowRight,
        "home" => Key::Home,
        "end" => Key::End,
        "pageup" => Key::PageUp,
        "pagedown" => Key::PageDown,
        "delete" => Key::Delete,
        "insert" => Key::Insert,
        // KYSEA-B03
        "-" => Key::Minus,
        "=" => Key::Equal,
        "[" => Key::BracketLeft,
        "]" => Key::BracketRight,
        "\\" => Key::Backslash,
        ";" => Key::Semicolon,
        "'" => Key::Quote,
        "," => Key::Comma,
        "." => Key::Period,
        "/" => Key::Slash,
        "`" => Key::Backquote,
        // KYSEA-B07
        f => function_key(f).unwrap_or(Key::Unidentified),
    }
}

fn function_key(name: &str) -> Option<Key> {
    let n: u32 = name.strip_prefix('f')?.parse().ok()?;
    if !(1..=12).contains(&n) {
        return None;
    }
    Key::try_from(Key::F1 as u32 + n - 1).ok()
}

/// The bytes libghostty-vt makes for `input` in the terminal's current state (KYSEA-B06).
pub fn encode(input: &Input, enc: &mut Encoder<'static>, term: &Terminal<'static, 'static>) -> Vec<u8> {
    let mut out = Vec::with_capacity(16);
    let mut ev = match Event::new() {
        Ok(ev) => ev,
        // KYSEA-E01
        Err(e) => {
            eprintln!("polvo: the key {:?} was not sent: {e}", input.key);
            return out;
        }
    };
    ev.set_action(Action::Press).set_key(input.key).set_mods(input.mods).set_utf8(input.text.clone());
    enc.set_options_from_terminal(term);
    if let Err(e) = enc.encode_to_vec(&ev, &mut out) {
        // KYSEA-E01
        eprintln!("polvo: the key {:?} was not sent: {e}", input.key);
        out.clear();
    }
    out
}

#[cfg(test)]
#[path = "keys_test.rs"]
mod tests;
