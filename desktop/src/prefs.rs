// @anchors
//   code: PRIMP
//   ref: PRFSE

//! The desktop preferences the window follows (PRFSE): the title bar buttons, the theme and the
//! monospace font.

/// A title bar button.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Button {
    Close,
    Minimize,
    Maximize,
}

/// The preferences of the desktop.
#[derive(Clone, Debug, PartialEq)]
pub struct Prefs {
    pub left: Vec<Button>,
    pub right: Vec<Button>,
    pub dark: bool,
    pub font: String,
    pub font_px: f32,
}

/// The button layout when the desktop gives none (PRFSE-B04).
pub const DEFAULT_LAYOUT: &str = ":minimize,maximize,close";
/// The monospace font when the desktop gives none (PRFSE-B04, PRFSE-E01).
pub const DEFAULT_FONT: &str = "DejaVu Sans Mono";
/// The size of the default font, in points.
pub const DEFAULT_POINTS: f32 = 11.;

/// Reads the preferences of this desktop: the GNOME keys through `gsettings`, and `POLVO_FONT`.
pub fn read() -> Prefs {
    let font = std::env::var("POLVO_FONT").ok();
    read_with(gsetting, font.as_deref())
}

/// Reads the preferences through `get`, which gives the value of a GNOME key or None, and `font`,
/// the name that replaces the desktop's font.
pub fn read_with(get: impl Fn(&str) -> Option<String>, font: Option<&str>) -> Prefs {
    // PRFSE-B01, PRFSE-B04
    let layout = get("button-layout").unwrap_or_else(|| DEFAULT_LAYOUT.into());
    let (left, right) = layout.split_once(':').unwrap_or(("", &layout));
    // PRFSE-B02: no scheme at all is the dark theme (PRFSE-B04)
    let dark = get("color-scheme").is_none_or(|s| s == "prefer-dark");
    // PRFSE-B03
    let mono = get("monospace-font-name");
    let parsed = mono.as_deref().and_then(|f| {
        let (name, size) = f.trim().rsplit_once(' ')?;
        let points = size.parse::<f32>().ok().filter(|p| *p > 0.)?;
        Some((name.to_string(), points))
    });
    let (name, points) = match (parsed, &mono) {
        (Some(font), _) => font,
        // PRFSE-E01: a font without a valid size is the default font, and the user is told why.
        (None, Some(f)) => {
            eprintln!("polvo: the monospace font {f:?} has no valid size; using {DEFAULT_FONT} {DEFAULT_POINTS}");
            (DEFAULT_FONT.into(), DEFAULT_POINTS)
        }
        // PRFSE-B04
        (None, None) => (DEFAULT_FONT.into(), DEFAULT_POINTS),
    };
    Prefs {
        left: buttons(left),
        right: buttons(right),
        dark,
        font: font.filter(|f| !f.is_empty()).map_or(name, str::to_string),
        font_px: (points * 96. / 72.).round(),
    }
}

fn buttons(side: &str) -> Vec<Button> {
    side.split(',')
        .filter_map(|b| match b.trim() {
            "close" => Some(Button::Close),
            "minimize" => Some(Button::Minimize),
            "maximize" => Some(Button::Maximize),
            _ => None,
        })
        .collect()
}

/// A GNOME key through `gsettings`: None when there is no `gsettings` or no such key.
fn gsetting(key: &str) -> Option<String> {
    let schema = match key {
        "button-layout" => "org.gnome.desktop.wm.preferences",
        _ => "org.gnome.desktop.interface",
    };
    let out = std::process::Command::new("gsettings").args(["get", schema, key]).output().ok()?;
    if !out.status.success() {
        return None;
    }
    Some(String::from_utf8_lossy(&out.stdout).trim().trim_matches('\'').to_string())
}

#[cfg(test)]
#[path = "prefs_test.rs"]
mod tests;
