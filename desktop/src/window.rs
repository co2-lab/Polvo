// @anchors
//   code: WNIMW
//   ref: WNDWI

//! The window of Polvo (WNDWI): the title bar the desktop's way, and below it the dock of the
//! terminals.

use std::rc::Rc;

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::dock::{DockArea, DockLayout, DockPlacement, DockSkin, PanelStyle, panel_handle};
use gpui_kit::component::{Icon, IconName, InteractiveElementExt as _, Sizable as _};
use gpui_kit::{
    App, AppContext as _, Bounds, Context, Entity, Focusable as _, FontWeight, Hsla, IntoElement, MouseButton, ParentElement as _,
    Render, Styled as _, Subscription, Window, div, prelude::*, px,
};

use crate::panel::{TerminalEvent, TerminalPanel}; // @dep: TPITR
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::screen::{Screen, Size}; // @dep: SCIMS

/// The height of the title bar (WNDWI-B01), GNOME's (libadwaita).
pub const TITLE_H: f32 = 38.;

/// How the window makes a new terminal (WNDWI-B10): the user's shell in the application, terminals
/// with no shell in the tests.
pub type NewTerminal = Rc<dyn Fn(&Prefs, &mut Window, &mut App) -> std::io::Result<Entity<TerminalPanel>>>;

/// The colours of the window for the theme (WNDWI-B07).
#[derive(Clone, Copy, Debug)]
pub struct Colors {
    pub bar: Hsla,
    pub fg: Hsla,
    pub button: Hsla,
    pub button_hover: Hsla,
}

/// The window: the title bar and the dock.
pub struct PolvoWindow {
    prefs: Prefs,
    dock: Entity<DockArea>,
    new_terminal: NewTerminal,
    /// The terminals in the dock, in the order they were opened.
    terminals: Vec<Entity<TerminalPanel>>,
    subscriptions: Vec<Subscription>,
    moving: bool,
}

impl PolvoWindow {
    /// The window with `first` in the dock, making the next terminals with `new_terminal`.
    pub fn new(
        prefs: Prefs,
        first: Entity<TerminalPanel>,
        new_terminal: NewTerminal,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) -> Self {
        // WNDWI-B09
        let (dock, skin) = DockSkin::dock_area("polvo", Some(1), window, cx);
        skin.set_panel_style(PanelStyle::TabBar, cx);
        let layout = DockLayout::tabs().panel_view(panel_handle(first.clone()), cx);
        dock.update(cx, |dock, cx| dock.set_center(layout, window, cx));
        let mut this = PolvoWindow { prefs, dock, new_terminal, terminals: Vec::new(), subscriptions: Vec::new(), moving: false };
        this.watch(first, window, cx);
        this
    }

    /// Hears the end of a terminal's shell (WNDWI-B11).
    fn watch(&mut self, panel: Entity<TerminalPanel>, window: &mut Window, cx: &mut Context<Self>) {
        let sub = cx.subscribe_in(&panel, window, |this, panel, ev: &TerminalEvent, window, cx| match ev {
            TerminalEvent::Exited => this.close_terminal(panel.clone(), window, cx),
        });
        self.subscriptions.push(sub);
        self.terminals.push(panel);
    }

    /// Takes the terminal out of the dock; the window closes with the last (WNDWI-B11).
    fn close_terminal(&mut self, panel: Entity<TerminalPanel>, window: &mut Window, cx: &mut Context<Self>) {
        self.terminals.retain(|t| t != &panel);
        self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        if self.terminals.is_empty() {
            window.remove_window();
        }
        cx.notify();
    }

    /// Opens a new terminal in the dock, with the focus (WNDWI-B10).
    pub fn open_terminal(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let panel = match (self.new_terminal)(&self.prefs, window, cx) {
            Ok(panel) => panel,
            Err(e) => {
                // WNDWI-E02
                eprintln!("polvo: the new terminal did not start: {e}");
                return;
            }
        };
        self.dock.update(cx, |dock, cx| dock.add_panel(panel.clone(), DockPlacement::Center, None, window, cx));
        let focus = panel.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        self.watch(panel, window, cx);
        cx.notify();
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

    /// The title bar the desktop's way (WNDWI-B01, WNDWI-B02), with the new terminal button on the side
    /// the window's buttons are not (WNDWI-B10).
    fn title_bar(&self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let c = self.colors();
        let supported = window.window_controls();
        let maximized = window.is_maximized();
        let round = move |id: &'static str, icon: IconName| {
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
                .child(Icon::new(icon).xsmall())
        };
        let button = move |b: Button| {
            let (id, icon) = match b {
                Button::Close => ("close", IconName::WindowClose),
                Button::Minimize => ("minimize", IconName::WindowMinimize),
                Button::Maximize if maximized => ("maximize", IconName::WindowRestore),
                Button::Maximize => ("maximize", IconName::WindowMaximize),
            };
            round(id, icon).on_click(move |_, window, cx| {
                cx.stop_propagation();
                match b {
                    Button::Close => window.remove_window(),
                    Button::Minimize => window.minimize_window(),
                    Button::Maximize => window.zoom_window(),
                }
            })
        };
        let new_terminal = round("new-terminal", IconName::Plus).on_click(cx.listener(|this, _, window, cx| {
            cx.stop_propagation();
            this.open_terminal(window, cx);
        }));
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
        let (mut left, mut right) = (side(&self.prefs.left), side(&self.prefs.right));
        if self.prefs.left.is_empty() {
            left = left.child(new_terminal);
        } else {
            right = right.child(new_terminal);
        }
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
            .child(left)
            .child(right)
    }
}

impl Render for PolvoWindow {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        gpui_kit::component::window_border().shadow_size(px(0.)).child(
            div()
                .flex()
                .flex_col()
                .size_full()
                .child(self.title_bar(window, cx))
                .child(div().flex_1().min_h_0().w_full().child(self.dock.clone())),
        )
    }
}

/// Opens the window of Polvo with the user's `shell`, in `cwd`, following `prefs`.
pub fn open(shell: String, cwd: Option<std::path::PathBuf>, prefs: Prefs, cx: &mut App) {
    // TRPNT-E01: no first terminal, no window.
    let size = Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 };
    let (screen, events) = match Screen::spawn(&[shell.clone()], cwd.as_deref(), size) {
        Ok(started) => started,
        Err(e) => {
            eprintln!("polvo: the shell did not start: {e}");
            cx.quit();
            return;
        }
    };
    let bounds = Bounds::centered(None, gpui_kit::size(px(1000.), px(640.)), cx);
    let options = gpui_kit::WindowOptions {
        window_bounds: Some(gpui_kit::WindowBounds::Windowed(bounds)),
        // GNOME on Wayland draws no frame: the window draws its own title bar and border.
        window_decorations: Some(gpui_kit::WindowDecorations::Client),
        app_id: Some("polvo".into()),
        ..gpui_kit::component::TitleBar::window_options()
    };
    let first_shell = shell.clone();
    let new_terminal: NewTerminal =
        Rc::new(move |prefs, window, cx| TerminalPanel::spawn(&shell, cwd.as_deref(), prefs, window, cx));
    let opened = gpui_kit::open_window(options, cx, |window, cx| {
        let first = cx.new(|cx| TerminalPanel::new(screen, Some(events), &first_shell, &prefs, window, cx));
        cx.new(|cx| PolvoWindow::new(prefs, first, new_terminal, window, cx))
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
