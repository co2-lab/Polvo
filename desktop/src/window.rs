// @anchors
//   code: WNIMW
//   ref: WNDWI

//! The GPUI window of the terminal (WNDWI): the title bar the desktop's way, and the terminal.

use std::cell::Cell;
use std::rc::Rc;
use std::time::Instant;

use futures::StreamExt;
use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::{Icon, IconName, InteractiveElementExt as _, Sizable as _};
use gpui_kit::{
    App, Bounds, ClipboardItem, Context, Focusable, Font, FocusHandle, FontWeight, Hsla, IntoElement, KeyDownEvent,
    MouseButton, MouseDownEvent, MouseMoveEvent, MouseUpEvent, ParentElement as _, Pixels, Point, Render, Rgba,
    ScrollDelta, ScrollWheelEvent, SharedString, Styled as _, Window, canvas, div, prelude::*, px,
};

use crate::keys::{Modifiers, translate}; // @dep: KYIMK
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::screen::{Event, Events, Frame, Rgb, Screen, Size}; // @dep: SCIMS

/// The height of the title bar (WNDWI-B01), GNOME's (libadwaita).
pub const TITLE_H: f32 = 38.;
/// The margin around the terminal (WNDWI-B06).
pub const PADDING: f32 = 8.;
/// The lines a step of the wheel scrolls (WNDWI-B05).
pub const SCROLL_LINES: isize = 3;

/// The colours of the window for the theme (WNDWI-B07).
#[derive(Clone, Copy, Debug)]
pub struct Colors {
    pub bar: Hsla,
    pub fg: Hsla,
    pub button: Hsla,
    pub button_hover: Hsla,
}

/// The window of one terminal.
pub struct TerminalView {
    screen: Screen,
    prefs: Prefs,
    focus: FocusHandle,
    font: Font,
    font_px: f32,
    line_px: f32,
    cell_w: f32,
    /// Where the terminal was laid out in the last frame (its padding included).
    area: Rc<Cell<Bounds<Pixels>>>,
    born: Instant,
    selecting: bool,
    moving: bool,
    scroll_acc: f32,
}

impl TerminalView {
    /// The window of `screen`, with the shell's `events` when there is a shell, following `prefs`.
    pub fn new(screen: Screen, events: Option<Events>, prefs: Prefs, window: &mut Window, cx: &mut Context<Self>) -> Self {
        if let Some(mut events) = events {
            cx.spawn_in(window, async move |this, cx| {
                while let Some(ev) = events.next().await {
                    let more = this.update_in(cx, |view, window, cx| match ev {
                        Event::Output(bytes) => {
                            view.screen.feed(&bytes);
                            cx.notify();
                            true
                        }
                        // WNDWI-B08
                        Event::Exit => {
                            window.remove_window();
                            false
                        }
                    });
                    if !matches!(more, Ok(true)) {
                        break;
                    }
                }
            })
            .detach();
        }
        let focus = cx.focus_handle();
        window.focus(&focus, cx);
        let font = gpui_kit::font(prefs.font.clone());
        let font_px = prefs.font_px;
        let font_id = window.text_system().resolve_font(&font);
        let cell_w = window.text_system().advance(font_id, px(font_px), 'm').map_or(font_px * 0.6, |a| f32::from(a.width));
        TerminalView {
            screen,
            prefs,
            focus,
            font,
            font_px,
            line_px: (font_px * 1.3).round(),
            cell_w,
            area: Rc::new(Cell::new(Bounds::default())),
            born: Instant::now(),
            selecting: false,
            moving: false,
            scroll_acc: 0.,
        }
    }

    /// The colours of the theme the preferences ask for (WNDWI-B07).
    pub fn colors(&self) -> Colors {
        let rgb = |c: u32| Hsla::from(gpui_kit::rgb(c));
        if self.prefs.dark {
            Colors {
                bar: rgb(0x2e2e32),
                fg: rgb(0xffffff),
                button: gpui_kit::hsla(0., 0., 1., 0.10),
                button_hover: gpui_kit::hsla(0., 0., 1., 0.18),
            }
        } else {
            Colors {
                bar: rgb(0xebebed),
                fg: rgb(0x1e1e1e),
                button: gpui_kit::hsla(0., 0., 0., 0.08),
                button_hover: gpui_kit::hsla(0., 0., 0., 0.14),
            }
        }
    }

    /// The size of the terminal that fits the area it was laid out in (WNDWI-B06).
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
        // WNDWI-B04: the copy and paste shortcuts are the window's, not the shell's.
        if m.control && m.shift && ks.key == "c" {
            if let Some(text) = self.screen.selection_text() {
                cx.write_to_clipboard(ClipboardItem::new_string(text));
            }
        } else if m.control && m.shift && ks.key == "v" {
            if let Some(text) = cx.read_from_clipboard().and_then(|i| i.text()) {
                self.screen.paste(&text);
            }
        } else {
            // WNDWI-B03
            let mods = Modifiers { control: m.control, alt: m.alt, shift: m.shift };
            self.screen.key(&translate(&ks.key, ks.key_char.as_deref(), mods));
        }
        cx.stop_propagation();
        cx.notify();
    }

    // WNDWI-B05
    fn mouse_down(&mut self, ev: &MouseDownEvent, window: &mut Window, cx: &mut Context<Self>) {
        window.focus(&self.focus, cx);
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
        // WNDWI-B04: a selection goes to the primary selection too.
        #[cfg(any(target_os = "linux", target_os = "freebsd"))]
        if let Some(text) = self.screen.selection_text() {
            cx.write_to_primary(ClipboardItem::new_string(text));
        }
        cx.notify();
    }

    // WNDWI-B04: the middle button pastes the primary selection.
    fn middle_down(&mut self, _: &MouseDownEvent, _: &mut Window, cx: &mut Context<Self>) {
        #[cfg(any(target_os = "linux", target_os = "freebsd"))]
        if let Some(text) = cx.read_from_primary().and_then(|i| i.text()) {
            self.screen.paste(&text);
        }
        cx.notify();
    }

    // WNDWI-B05
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

    /// The title bar the desktop's way (WNDWI-B01, WNDWI-B02).
    fn title_bar(&self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let c = self.colors();
        let supported = window.window_controls();
        let maximized = window.is_maximized();
        let button = move |b: Button| {
            let (id, icon) = match b {
                Button::Close => ("close", IconName::WindowClose),
                Button::Minimize => ("minimize", IconName::WindowMinimize),
                Button::Maximize if maximized => ("maximize", IconName::WindowRestore),
                Button::Maximize => ("maximize", IconName::WindowMaximize),
            };
            div()
                .id(id)
                .test_support()
                .size(px(24.))
                .rounded_full()
                .flex()
                .items_center()
                .justify_center()
                .bg(c.button)
                .hover(move |s| s.bg(c.button_hover))
                .text_color(c.fg)
                .on_mouse_down(MouseButton::Left, |_, window, cx| {
                    window.prevent_default();
                    cx.stop_propagation();
                })
                .on_click(move |_, window, cx| {
                    cx.stop_propagation();
                    match b {
                        Button::Close => window.remove_window(),
                        Button::Minimize => window.minimize_window(),
                        Button::Maximize => window.zoom_window(),
                    }
                })
                .child(Icon::new(icon).xsmall())
        };
        let side = move |buttons: &[Button]| {
            div().flex().items_center().gap(px(12.)).children(
                buttons
                    .iter()
                    .copied()
                    .filter(|b| match b {
                        Button::Minimize => supported.minimize,
                        Button::Maximize => supported.maximize,
                        Button::Close => true,
                    })
                    .map(button),
            )
        };
        div()
            .id("title-bar")
            .relative()
            .flex()
            .flex_shrink_0()
            .items_center()
            .justify_between()
            .h(px(TITLE_H))
            .px(px(8.))
            .bg(c.bar)
            .text_color(c.fg)
            .on_double_click(|_, window, _| window.zoom_window())
            // After `on_double_click`, which only the stateful element has.
            .test_support()
            .on_mouse_down(MouseButton::Left, cx.listener(|v, _, _, _| v.moving = true))
            .on_mouse_up(MouseButton::Left, cx.listener(|v, _, _, _| v.moving = false))
            .on_mouse_move(cx.listener(|v, _, window, _| {
                if v.moving {
                    v.moving = false;
                    window.start_window_move();
                }
            }))
            .on_mouse_down(MouseButton::Right, |ev, window, _| window.show_window_menu(ev.position))
            .child(
                div()
                    .absolute()
                    .left_0()
                    .right_0()
                    .flex()
                    .justify_center()
                    .child(div().id("title").test_support().font_weight(FontWeight::BOLD).child("Polvo")),
            )
            .child(side(&self.prefs.left))
            .child(side(&self.prefs.right))
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

impl Focusable for TerminalView {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl Render for TerminalView {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        self.fit();
        let frame = self.screen.frame();
        let area = self.area.clone();
        let terminal = div()
            .id("terminal")
            .test_support()
            .relative()
            .flex_1()
            .w_full()
            .min_h_0()
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
            .children(self.lines(&frame));
        gpui_kit::component::window_border().shadow_size(px(0.)).child(
            div().flex().flex_col().size_full().child(self.title_bar(window, cx)).child(terminal),
        )
    }
}

/// Opens the window of `screen` (the shell's events by `events`), following `prefs`.
pub fn open(screen: Screen, events: Events, prefs: Prefs, cx: &mut App) {
    let bounds = Bounds::centered(None, gpui_kit::size(px(1000.), px(640.)), cx);
    let options = gpui_kit::WindowOptions {
        window_bounds: Some(gpui_kit::WindowBounds::Windowed(bounds)),
        // GNOME on Wayland draws no frame: the window draws its own title bar and border.
        window_decorations: Some(gpui_kit::WindowDecorations::Client),
        app_id: Some("polvo".into()),
        ..gpui_kit::component::TitleBar::window_options()
    };
    let opened = gpui_kit::open_window(options, cx, |window, cx| {
        cx.new(|cx| TerminalView::new(screen, Some(events), prefs, window, cx))
    });
    if let Err(e) = opened {
        // WNDWI-E01
        eprintln!("polvo: the window did not open: {e}");
        cx.quit();
    }
}

#[cfg(test)]
#[path = "window_test.rs"]
mod tests;
