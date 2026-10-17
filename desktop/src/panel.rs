// @anchors
//   code: TPITR
//   ref: TRPNT

//! A terminal as a panel of the dock (TRPNT): it draws the screen and gives it the keyboard, the
//! mouse, what is pasted and its size.

use std::cell::Cell;
use std::path::Path;
use std::rc::Rc;
use std::time::Instant;

use futures::StreamExt;
use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent};
use gpui_kit::{
    App, AppContext as _, Bounds, ClipboardItem, Context, Entity, EventEmitter, Focusable, Font, FocusHandle, FontWeight,
    Hsla, IntoElement, KeyDownEvent, MouseButton, MouseDownEvent, MouseMoveEvent, MouseUpEvent, ParentElement as _,
    Pixels, Point, Render, Rgba, ScrollDelta, ScrollWheelEvent, SharedString, Styled as _, Task, Window, canvas, div,
    prelude::*, px,
};

use crate::keymap::{Action, Keymap}; // @dep: KYIMA
use crate::keys::{Modifiers, translate}; // @dep: KYIMK
use crate::prefs::Prefs; // @dep: PRIMP
use crate::screen::{Event, Events, Frame, Rgb, Screen, Size}; // @dep: SCIMS

/// The margin around the terminal (TRPNT-B04).
pub const PADDING: f32 = 8.;
/// The lines a step of the wheel scrolls (TRPNT-B03).
pub const SCROLL_LINES: isize = 3;
/// The shell when `SHELL` is not set (TRPNT-E01).
pub const SHELL_FALLBACK: &str = "/bin/sh";

/// What the panel tells the window.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum TerminalEvent {
    /// The shell ended (TRPNT-B07).
    Exited,
}

/// A terminal in the dock.
pub struct TerminalPanel {
    screen: Screen,
    shell: String,
    focus: FocusHandle,
    font: Font,
    font_px: f32,
    line_px: f32,
    cell_w: f32,
    /// Where the terminal was laid out in the last frame (its padding included).
    area: Rc<Cell<Bounds<Pixels>>>,
    born: Instant,
    selecting: bool,
    scroll_acc: f32,
    _listening: Option<Task<()>>,
}

impl TerminalPanel {
    /// The panel of `screen` (the shell's `events` when there is a shell), running `shell`, with the
    /// font of `prefs`.
    pub fn new(
        screen: Screen,
        events: Option<Events>,
        shell: &str,
        prefs: &Prefs,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) -> Self {
        // The task that reads the shell lives as long as the panel: dropping the panel drops it, and the
        // shell's channel with it.
        let listening = events.map(|mut events| {
            cx.spawn_in(window, async move |this, cx| {
                while let Some(ev) = events.next().await {
                    let more = this.update(cx, |panel, cx| match ev {
                        Event::Output(bytes) => {
                            panel.screen.feed(&bytes);
                            cx.notify();
                            true
                        }
                        // TRPNT-B07
                        Event::Exit => {
                            cx.emit(TerminalEvent::Exited);
                            false
                        }
                    });
                    if !matches!(more, Ok(true)) {
                        break;
                    }
                }
            })
        });
        let focus = cx.focus_handle();
        window.focus(&focus, cx);
        // TRPNT-B05
        let font = gpui_kit::font(prefs.font.clone());
        let font_px = prefs.font_px;
        let font_id = window.text_system().resolve_font(&font);
        let cell_w = window.text_system().advance(font_id, px(font_px), 'm').map_or(font_px * 0.6, |a| f32::from(a.width));
        TerminalPanel {
            screen,
            shell: shell.to_string(),
            focus,
            font,
            font_px,
            line_px: (font_px * 1.3).round(),
            cell_w,
            area: Rc::new(Cell::new(Bounds::default())),
            born: Instant::now(),
            selecting: false,
            scroll_acc: 0.,
            _listening: listening,
        }
    }

    /// Starts `shell` in `cwd` and makes its panel; the error when it does not start (TRPNT-E01).
    pub fn spawn(
        shell: &str,
        cwd: Option<&Path>,
        prefs: &Prefs,
        window: &mut Window,
        cx: &mut App,
    ) -> std::io::Result<Entity<TerminalPanel>> {
        // The panel fits the terminal to its size once it is laid out.
        let size = Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 };
        let (screen, events) = match Screen::spawn(&[shell.to_string()], cwd, size) {
            Ok(started) => started,
            // TRPNT-E01: the error goes back to whoever asked for the terminal.
            Err(e) => return Err(e),
        };
        Ok(cx.new(|cx| TerminalPanel::new(screen, Some(events), shell, prefs, window, cx)))
    }

    /// A panel with no shell that shows why `shell` did not start (TRPNT-B09).
    /// The error when not even a terminal with no shell can be made (libghostty-vt out of memory).
    pub fn failed(
        reason: &str,
        shell: &str,
        prefs: &Prefs,
        window: &mut Window,
        cx: &mut App,
    ) -> Result<Entity<TerminalPanel>, String> {
        let size = Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 };
        let mut screen = Screen::new(size, Box::new(std::io::sink()))?;
        screen.feed(format!("polvo: the shell did not start: {reason}\r\n").as_bytes());
        Ok(cx.new(|cx| TerminalPanel::new(screen, None, shell, prefs, window, cx)))
    }

    /// The title of the tab: the shell's title for the terminal, or the shell's name (TRPNT-B06).
    pub fn tab_title(&self) -> String {
        let title = self.screen.title();
        if !title.is_empty() {
            return title;
        }
        Path::new(&self.shell).file_name().map_or(self.shell.clone(), |n| n.to_string_lossy().into_owned())
    }

    /// Where the panel was laid out in the last frame (TRPNT-B04).
    #[cfg_attr(not(test), expect(dead_code, reason = "the window's tests read where each terminal is"))]
    pub fn laid_out(&self) -> Bounds<Pixels> {
        self.area.get()
    }

    /// The size of the terminal that fits the area it was laid out in (TRPNT-B04).
    fn fit(&mut self) {
        let area = self.area.get();
        if area.size.width <= px(0.) || area.size.height <= px(0.) {
            return;
        }
        let cols = ((f32::from(area.size.width) - 2. * PADDING) / self.cell_w).floor().max(1.) as u16;
        let rows = ((f32::from(area.size.height) - 2. * PADDING) / self.line_px).floor().max(1.) as u16;
        self.screen.resize(Size { rows, cols, cell_w: self.cell_w.round() as u32, cell_h: self.line_px as u32 });
    }

    /// A window position, in the pixels of the terminal's grid: from its corner, and in the whole
    /// pixels of a cell the terminal counts in (the font's cell is fractional).
    fn local(&self, p: Point<Pixels>) -> (f64, f64) {
        let o = self.area.get().origin;
        let size = self.screen.size();
        let x = (f32::from(p.x - o.x) - PADDING) / self.cell_w * size.cell_w as f32;
        let y = (f32::from(p.y - o.y) - PADDING) / self.line_px * size.cell_h as f32;
        (f64::from(x), f64::from(y))
    }

    fn key_down(&mut self, ev: &KeyDownEvent, _: &mut Window, cx: &mut Context<Self>) {
        let ks = &ev.keystroke;
        let m = ks.modifiers;
        let action = cx.try_global::<Keymap>().cloned().unwrap_or_default().action_for(&ks.unparse());
        match action {
            // TRPNT-B02: the copy and paste shortcuts are the panel's, not the shell's.
            Some(Action::Copy) => {
                if let Some(text) = self.screen.selection_text() {
                    cx.write_to_clipboard(ClipboardItem::new_string(text));
                }
            }
            Some(Action::Paste) => {
                if let Some(text) = cx.read_from_clipboard().and_then(|i| i.text()) {
                    self.screen.paste(&text);
                }
            }
            // TRPNT-B01: the window's actions go on to the window
            Some(_) => return,
            None => {
                let mods = Modifiers { control: m.control, alt: m.alt, shift: m.shift };
                self.screen.key(&translate(&ks.key, ks.key_char.as_deref(), mods));
            }
        }
        cx.stop_propagation();
        cx.notify();
    }

    fn mouse_down(&mut self, ev: &MouseDownEvent, window: &mut Window, cx: &mut Context<Self>) {
        // TRPNT-B08
        window.focus(&self.focus, cx);
        // TRPNT-B03
        let (x, y) = self.local(ev.position);
        self.screen.press(x, y, self.born.elapsed());
        self.selecting = true;
        cx.notify();
    }

    fn mouse_move(&mut self, ev: &MouseMoveEvent, _: &mut Window, cx: &mut Context<Self>) {
        if self.selecting && ev.pressed_button == Some(MouseButton::Left) {
            let (x, y) = self.local(ev.position);
            self.screen.drag(x, y);
            cx.notify();
        }
    }

    fn mouse_up(&mut self, ev: &MouseUpEvent, _: &mut Window, cx: &mut Context<Self>) {
        if !self.selecting {
            return;
        }
        self.selecting = false;
        let (x, y) = self.local(ev.position);
        self.screen.release(x, y);
        // TRPNT-B02: a selection goes to the primary selection too.
        #[cfg(any(target_os = "linux", target_os = "freebsd"))]
        if let Some(text) = self.screen.selection_text() {
            cx.write_to_primary(ClipboardItem::new_string(text));
        }
        cx.notify();
    }

    // TRPNT-B02: the middle button pastes the primary selection.
    fn middle_down(&mut self, _: &MouseDownEvent, window: &mut Window, cx: &mut Context<Self>) {
        window.focus(&self.focus, cx);
        #[cfg(any(target_os = "linux", target_os = "freebsd"))]
        if let Some(text) = cx.read_from_primary().and_then(|i| i.text()) {
            self.screen.paste(&text);
        }
        cx.notify();
    }

    // TRPNT-B03
    fn wheel(&mut self, ev: &ScrollWheelEvent, _: &mut Window, cx: &mut Context<Self>) {
        self.scroll_acc += match ev.delta {
            ScrollDelta::Lines(d) => d.y * SCROLL_LINES as f32,
            ScrollDelta::Pixels(d) => f32::from(d.y) / self.line_px,
        };
        let lines = self.scroll_acc.trunc();
        self.scroll_acc -= lines;
        if lines != 0. {
            self.screen.scroll(-(lines as isize));
            cx.notify();
        }
    }

    /// The terminal's lines as runs of colour.
    fn lines(&self, frame: &Frame) -> Vec<gpui_kit::AnyElement> {
        let line_px = self.line_px;
        frame
            .lines
            .iter()
            .map(|runs| {
                div()
                    .flex()
                    .h(px(line_px))
                    .children(runs.iter().map(|r| {
                        let d = div().bg(hsla(r.bg)).text_color(hsla(r.fg)).child(SharedString::from(r.text.clone()));
                        let d = if r.bold { d.font_weight(FontWeight::BOLD) } else { d };
                        let d = if r.italic { d.italic() } else { d };
                        if r.underline { d.underline() } else { d }
                    }))
                    .into_any_element()
            })
            .collect()
    }
}

fn hsla((r, g, b): Rgb) -> Hsla {
    Rgba { r: f32::from(r) / 255., g: f32::from(g) / 255., b: f32::from(b) / 255., a: 1. }.into()
}

impl EventEmitter<TerminalEvent> for TerminalPanel {}
impl EventEmitter<PanelEvent> for TerminalPanel {}

impl Focusable for TerminalPanel {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl BasePanel for TerminalPanel {
    fn panel_name(&self) -> &'static str {
        "terminal"
    }
}

impl Panel for TerminalPanel {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        self.tab_title()
    }
}

impl Render for TerminalPanel {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        self.fit();
        let frame = self.screen.frame();
        let area = self.area.clone();
        div()
            .id("terminal")
            .test_support()
            .relative()
            .size_full()
            .p(px(PADDING))
            .bg(hsla(frame.bg))
            .font(self.font.clone())
            .text_size(px(self.font_px))
            .line_height(px(self.line_px))
            .whitespace_nowrap()
            .overflow_hidden()
            .track_focus(&self.focus)
            .on_key_down(cx.listener(Self::key_down))
            .on_mouse_down(MouseButton::Left, cx.listener(Self::mouse_down))
            .on_mouse_down(MouseButton::Middle, cx.listener(Self::middle_down))
            .on_mouse_move(cx.listener(Self::mouse_move))
            .on_mouse_up(MouseButton::Left, cx.listener(Self::mouse_up))
            .on_scroll_wheel(cx.listener(Self::wheel))
            .child(canvas(move |bounds, _, _| area.set(bounds), |_, _, _, _| {}).absolute().top_0().left_0().size_full())
            .children(self.lines(&frame))
    }
}

#[cfg(test)]
#[path = "panel_test.rs"]
mod tests;
