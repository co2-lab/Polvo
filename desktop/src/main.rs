// @anchors
//   code: DSMND
//   ref: JNNTJ

//! The native window of Polvo: the terminal of this machine, in a GPUI window (plan JNNTJ).

/// A test named by its scenario, the code and the title as the feature has them (the Rust
/// counterpart of Go's `t.Run("CODE: title", …)`).
#[cfg(test)]
macro_rules! scenario {
    ($name:ident, $title:literal, $body:block) => {
        #[doc = $title]
        #[test]
        fn $name() $body
    };
    // A test of the window: it runs in GPUI's headless test platform, with its context.
    ($name:ident, $title:literal, |$cx:ident| $body:block) => {
        #[doc = $title]
        #[gpui_kit::test]
        fn $name($cx: &mut gpui_kit::TestAppContext) $body
    };
}

mod keys; // @dep: KYIMK
mod prefs; // @dep: PRIMP
mod screen; // @dep: SCIMS
mod window; // @dep: WNIMW

use screen::{Screen, Size};

fn main() {
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/sh".into());
    let home = std::env::var_os("HOME").map(std::path::PathBuf::from);
    // The window fits the terminal to its size once it is laid out.
    let size = Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 };
    let (screen, events) = match Screen::spawn(&[shell], home.as_deref(), size) {
        Ok(s) => s,
        Err(e) => {
            eprintln!("polvo: the shell did not start: {e}");
            std::process::exit(1);
        }
    };
    let prefs = prefs::read();
    gpui_kit::application().with_assets(gpui_kit::assets::Assets).run(move |cx| {
        gpui_kit::init(cx);
        if prefs.dark {
            gpui_kit::component::Theme::change(gpui_kit::component::ThemeMode::Dark, None, cx);
        }
        window::open(screen, events, prefs, cx);
        cx.activate(true);
    });
}
