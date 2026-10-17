// @anchors
//   code: SPIST
//   ref: STPNS

//! The settings tab of the dock (STPNS): the sections, and the keyboard shortcuts of the window.

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent};
use gpui_kit::{
    App, Context, EventEmitter, FocusHandle, Focusable, FontWeight, IntoElement, KeyDownEvent, ParentElement as _,
    Render, Role, SharedString, Styled as _, Window, div, prelude::*, px,
};

use crate::keymap::{Action, Keymap, KeymapError}; // @dep: KYIMA
use crate::settings::{Settings, SettingsFile}; // @dep: STIMS

/// The keys that only modify another (STPNS-B04).
const MODIFIER_KEYS: [&str; 6] = ["ctrl", "alt", "shift", "cmd", "platform", "function"];

/// The settings of the window, as a tab of the dock.
pub struct SettingsPanel {
    focus: FocusHandle,
    file: Option<SettingsFile>,
    /// The action whose new shortcut is being pressed.
    recording: Option<Action>,
    message: Option<String>,
}

impl SettingsPanel {
    /// The panel, saving to `file` (none: the changes last until the window closes).
    pub fn new(file: Option<SettingsFile>, _: &mut Window, cx: &mut Context<Self>) -> Self {
        SettingsPanel { focus: cx.focus_handle(), file, recording: None, message: None }
    }

    /// The action whose new shortcut the panel waits for.
    pub fn listening(&self) -> Option<Action> {
        self.recording
    }

    /// Shows the user why something they did failed.
    fn tell(&mut self, message: String) {
        self.message = Some(message);
    }

    fn keymap(cx: &App) -> Keymap {
        cx.try_global::<Keymap>().cloned().unwrap_or_default()
    }

    /// STPNS-B03: listen for the action's new shortcut.
    fn listen(&mut self, a: Action, window: &mut Window, cx: &mut Context<Self>) {
        self.recording = Some(a);
        self.message = None;
        window.focus(&self.focus, cx);
        cx.notify();
    }

    fn key_down(&mut self, ev: &KeyDownEvent, _: &mut Window, cx: &mut Context<Self>) {
        let Some(a) = self.recording else { return };
        cx.stop_propagation();
        let ks = &ev.keystroke;
        if MODIFIER_KEYS.contains(&ks.key.as_str()) {
            // STPNS-B04: still listening, for the key
            return;
        }
        if ks.key == "escape" && !ks.modifiers.modified() {
            // STPNS-B04
            self.recording = None;
            cx.notify();
            return;
        }
        let mut keymap = Self::keymap(cx);
        let refused = keymap.set(a, &ks.unparse());
        if let Err(e) = refused {
            self.tell(match e {
                // STPNS-E01: nothing changes
                KeymapError::Taken(other) => format!("{} já é o atalho de «{}».", ks.unparse(), other.label()),
                KeymapError::Invalid(e) => format!("Não é um atalho: {e}"),
            });
            cx.notify();
            return;
        }
        self.recording = None;
        self.message = None;
        let overrides = keymap.overrides();
        cx.set_global(keymap);
        if let Some(file) = &self.file {
            if let Err(e) = file.write(&Settings { keys: overrides }) {
                // STPNS-E02: the shortcut stays for this session
                eprintln!("polvo: the settings were not saved in {}: {e}", file.path().display());
                self.tell(format!("A configuração não foi gravada em {}: {e}", file.path().display()));
            }
        }
        cx.notify();
    }

    /// STPNS-B02: an action with its shortcut, which listens for a new one when clicked.
    fn row(&self, a: Action, keymap: &Keymap, cx: &mut Context<Self>) -> impl IntoElement {
        let theme = cx.theme();
        let listening = self.listening() == Some(a);
        let shown: SharedString = if listening { "Prima as teclas…".into() } else { keymap.shortcut(a).to_string().into() };
        div()
            .flex()
            .items_center()
            .justify_between()
            .py(px(6.))
            .child(a.label())
            .child(
                div()
                    .id(SharedString::from(format!("shortcut-{}", a.id())))
                    .role(Role::Button)
                    .aria_label(a.label())
                    .aria_value(shown.clone())
                    .test_support()
                    .px(px(10.))
                    .py(px(4.))
                    .rounded(px(6.))
                    .border_1()
                    .border_color(if listening { theme.ring } else { theme.border })
                    .bg(theme.secondary)
                    .font_family("monospace")
                    .cursor_pointer()
                    .on_click(cx.listener(move |this, _, window, cx| this.listen(a, window, cx)))
                    .child(shown),
            )
    }
}

impl EventEmitter<PanelEvent> for SettingsPanel {}

impl Focusable for SettingsPanel {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl BasePanel for SettingsPanel {
    fn panel_name(&self) -> &'static str {
        "settings"
    }
}

impl Panel for SettingsPanel {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        "Configuração"
    }
}

impl Render for SettingsPanel {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let keymap = Self::keymap(cx);
        let rows: Vec<_> = Action::ALL.into_iter().map(|a| self.row(a, &keymap, cx).into_any_element()).collect();
        let theme = cx.theme();
        // STPNS-B01: the sections on the left, the keyboard shortcuts open
        let sections = div().w(px(200.)).flex_shrink_0().p(px(8.)).bg(theme.sidebar).child(
            div()
                .id("section-keys")
                .role(Role::Tab)
                .aria_label("Atalhos de teclado")
                .aria_selected(true)
                .test_support()
                .px(px(10.))
                .py(px(6.))
                .rounded(px(6.))
                .bg(theme.list_active)
                .child("Atalhos de teclado"),
        );
        let message = self.message.clone().map(|m| {
            div()
                .id("keys-message")
                .role(Role::Status)
                .aria_label(m.clone())
                .test_support()
                .mt(px(12.))
                .text_color(theme.danger)
                .child(m)
        });
        div()
            .id("settings")
            .test_support()
            .size_full()
            .flex()
            .bg(theme.background)
            .text_color(theme.foreground)
            .track_focus(&self.focus)
            .on_key_down(cx.listener(Self::key_down))
            .child(sections)
            .child(
                div()
                    .flex_1()
                    .p(px(16.))
                    .child(div().font_weight(FontWeight::BOLD).pb(px(8.)).child("Atalhos de teclado"))
                    .children(rows)
                    .children(message),
            )
    }
}

#[cfg(test)]
#[path = "settings_panel_test.rs"]
mod tests;
