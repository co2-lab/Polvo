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
    // The same, for a test that waits on the window.
    ($name:ident, $title:literal, async |$cx:ident| $body:block) => {
        #[doc = $title]
        #[gpui_kit::test]
        async fn $name($cx: &mut gpui_kit::TestAppContext) $body
    };
}

mod editor; // @dep: EDIMP
mod keymap; // @dep: KYIMA
mod keys; // @dep: KYIMK
mod layout; // @dep: DLIDC
mod new_tab; // @dep: NTIMP
mod panel; // @dep: TPITR
mod prefs; // @dep: PRIMP
mod screen; // @dep: SCIMS
mod settings; // @dep: STIMS
mod settings_panel; // @dep: SPIST
mod window; // @dep: WNIMW

fn main() {
    let shell = std::env::var("SHELL").unwrap_or_else(|_| panel::SHELL_FALLBACK.into());
    let home = std::env::var_os("HOME").map(std::path::PathBuf::from);
    let prefs = prefs::read();
    gpui_kit::application().with_assets(gpui_kit::assets::Assets).run(move |cx| {
        gpui_kit::init(cx);
        // CEACN-W01: the shortcuts the user changed, and the file the settings tab saves to.
        if let Some(file) = settings::SettingsFile::default() {
            cx.set_global(keymap::Keymap::with_overrides(&file.read().keys));
            cx.set_global(file);
        }
        if prefs.dark {
            gpui_kit::component::Theme::change(gpui_kit::component::ThemeMode::Dark, None, cx);
        }
        window::open(shell, home, prefs, cx);
        cx.activate(true);
    });
}
