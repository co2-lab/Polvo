// @anchors
//   code: NTIMP
//   ref: NWTBS

//! The chooser of the new tab's kind (NWTBS): over the dock, the kinds of tab the window can open.

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::{
    App, Context, EventEmitter, FocusHandle, Focusable, FontWeight, IntoElement, KeyDownEvent, ParentElement as _,
    Render, Role, SharedString, Styled as _, Window, div, prelude::*, px,
};

/// A kind of tab the window can open.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct TabKind {
    pub id: &'static str,
    pub label: &'static str,
}

/// What the chooser tells the window.
#[derive(Clone, Debug, PartialEq)]
pub enum ChooserEvent {
    /// The user chose this kind.
    Chosen(TabKind),
    /// The user closed the chooser without choosing.
    Dismissed,
}

/// The chooser of the new tab's kind.
pub struct NewTabChooser {
    focus: FocusHandle,
    kinds: Vec<TabKind>,
    selected: usize,
}

impl NewTabChooser {
    /// The chooser of `kinds`, with the first chosen and the focus (NWTBS-B01).
    pub fn new(kinds: Vec<TabKind>, window: &mut Window, cx: &mut Context<Self>) -> Self {
        let focus = cx.focus_handle();
        window.focus(&focus, cx);
        NewTabChooser { focus, kinds, selected: 0 }
    }

    fn choose(&mut self, i: usize, cx: &mut Context<Self>) {
        if let Some(kind) = self.kinds.get(i) {
            cx.emit(ChooserEvent::Chosen(*kind));
        }
    }

    /// NWTBS-B02, NWTBS-B04
    fn key_down(&mut self, ev: &KeyDownEvent, _: &mut Window, cx: &mut Context<Self>) {
        let ks = &ev.keystroke;
        if ks.modifiers.modified() {
            return;
        }
        cx.stop_propagation();
        match ks.key.as_str() {
            "down" => self.selected = (self.selected + 1).min(self.kinds.len().saturating_sub(1)),
            "up" => self.selected = self.selected.saturating_sub(1),
            "enter" => self.choose(self.selected, cx),
            "escape" => cx.emit(ChooserEvent::Dismissed),
            key => match key.parse::<usize>() {
                Ok(n) if n >= 1 => self.choose(n - 1, cx),
                _ => {}
            },
        }
        cx.notify();
    }
}

impl EventEmitter<ChooserEvent> for NewTabChooser {}

impl Focusable for NewTabChooser {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl Render for NewTabChooser {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let theme = cx.theme();
        let (active, muted, popover, border, fg) = (theme.list_active, theme.muted_foreground, theme.popover, theme.border, theme.foreground);
        let rows: Vec<_> = self
            .kinds
            .iter()
            .enumerate()
            .map(|(i, kind)| {
                let selected = i == self.selected;
                div()
                    .id(SharedString::from(format!("kind-{}", kind.id)))
                    .role(Role::ListItem)
                    .aria_label(format!("{} {}", i + 1, kind.label))
                    .aria_selected(selected)
                    .test_support()
                    .flex()
                    .gap(px(12.))
                    .px(px(12.))
                    .py(px(8.))
                    .rounded(px(6.))
                    .cursor_pointer()
                    .when(selected, |d| d.bg(active))
                    // NWTBS-B03
                    .on_click(cx.listener(move |this, _, _, cx| this.choose(i, cx)))
                    .child(div().text_color(muted).child(format!("{}", i + 1)))
                    .child(kind.label)
            })
            .collect();
        div()
            .id("new-tab")
            .test_support()
            .track_focus(&self.focus)
            .on_key_down(cx.listener(Self::key_down))
            .w(px(320.))
            .p(px(8.))
            .rounded(px(10.))
            .border_1()
            .border_color(border)
            .bg(popover)
            .text_color(fg)
            .shadow_lg()
            .child(div().px(px(12.)).py(px(6.)).font_weight(FontWeight::BOLD).child("Aba nova"))
            .children(rows)
    }
}

#[cfg(test)]
#[path = "new_tab_test.rs"]
mod tests;
