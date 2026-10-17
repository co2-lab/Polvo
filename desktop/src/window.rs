// @anchors
//   code: WNIMW
//   ref: WNDWI

//! The window of Polvo (WNDWI): the title bar the desktop's way, and below it the dock of the
//! terminals.

use std::cell::RefCell;
use std::collections::HashMap;
use std::rc::Rc;
use std::time::Duration;

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::dock::{
    DockArea, DockEvent, DockLayout, DockPlacement, DockSkin, PanelId, PanelStyle, panel_handle, register_panel,
};
use gpui_kit::component::{Icon, IconName, InteractiveElementExt as _, Sizable as _};
use gpui_kit::{
    App, AppContext as _, Bounds, Context, Entity, EntityId, Focusable as _, FontWeight, Hsla, IntoElement, MouseButton, ParentElement as _,
    Render, Styled as _, Subscription, Task, Window, div, prelude::*, px,
};

use crate::layout::{LAYOUT_VERSION, LayoutFile}; // @dep: DLIDC
use crate::panel::{TerminalEvent, TerminalPanel}; // @dep: TPITR
use crate::prefs::{Button, Prefs}; // @dep: PRIMP

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

/// The time after the last change of the dock when the organisation is saved (WNDWI-B12).
pub const SAVE_DELAY: Duration = Duration::from_millis(500);

/// The window: the title bar and the dock.
pub struct PolvoWindow {
    prefs: Prefs,
    dock: Entity<DockArea>,
    new_terminal: NewTerminal,
    layout: Option<LayoutFile>,
    /// The terminals in the dock, in the order they were opened.
    terminals: Vec<Entity<TerminalPanel>>,
    /// What the window hears from each terminal, by the terminal's entity.
    heard: HashMap<EntityId, Subscription>,
    /// What the window hears from the dock.
    dock_heard: Option<Subscription>,
    saving: Option<Task<()>>,
    moving: bool,
}

impl PolvoWindow {
    /// The window, with the dock as `layout` kept it, or with one terminal; the terminals are made with
    /// `new_terminal`.
    pub fn new(
        prefs: Prefs,
        new_terminal: NewTerminal,
        layout: Option<LayoutFile>,
        window: &mut Window,
        cx: &mut Context<Self>,
    ) -> Self {
        // WNDWI-B09
        let (dock, skin) = DockSkin::dock_area("polvo", Some(LAYOUT_VERSION), window, cx);
        skin.set_panel_style(PanelStyle::TabBar, cx);
        skin.set_close_button_visible(true, cx);
        let mut this = PolvoWindow {
            prefs,
            dock: dock.clone(),
            new_terminal,
            layout,
            terminals: Vec::new(),
            heard: HashMap::new(),
            dock_heard: None,
            saving: None,
            moving: false,
        };
        // WNDWI-B12: the kept organisation, each terminal with a new shell.
        let built: Rc<RefCell<Vec<Entity<TerminalPanel>>>> = Rc::default();
        {
            let (make, prefs, built) = (this.new_terminal.clone(), this.prefs.clone(), built.clone());
            register_panel(cx, "terminal", move |_, window, cx| {
                let panel = match make(&prefs, window, cx) {
                    Ok(panel) => panel,
                    // WNDWI-E04: a restored terminal that does not start says why in its tab.
                    Err(e) => match TerminalPanel::failed(&e.to_string(), "", &prefs, window, cx) {
                        Ok(panel) => panel,
                        Err(e) => panic!("polvo: not even an empty terminal could be made: {e}"),
                    },
                };
                built.borrow_mut().push(panel.clone());
                panel_handle(panel)
            });
        }
        if let Some(state) = this.layout.as_ref().and_then(LayoutFile::read) {
            if let Err(e) = dock.update(cx, |dock, cx| dock.load(state, window, cx)) {
                eprintln!("polvo: the layout could not be restored ({e}); starting with the default");
            }
        }
        for panel in built.borrow_mut().drain(..) {
            this.watch(panel, window, cx);
        }
        if this.terminals.is_empty() {
            match (this.new_terminal)(&this.prefs, window, cx) {
                Ok(first) => {
                    let layout = DockLayout::tabs().panel_view(panel_handle(first.clone()), cx);
                    dock.update(cx, |dock, cx| dock.set_center(layout, window, cx));
                    this.watch(first, window, cx);
                }
                Err(e) => {
                    // WNDWI-E04
                    eprintln!("polvo: the shell did not start: {e}");
                    window.defer(cx, |window, _| window.remove_window());
                }
            }
        }
        let sub = cx.subscribe_in(&dock, window, |this, _, ev: &DockEvent, window, cx| {
            if let DockEvent::LayoutChanged = ev {
                this.layout_changed(window, cx);
            }
        });
        this.dock_heard = Some(sub);
        this
    }

    /// Hears the end of a terminal's shell (WNDWI-B11).
    fn watch(&mut self, panel: Entity<TerminalPanel>, window: &mut Window, cx: &mut Context<Self>) {
        let sub = cx.subscribe_in(&panel, window, |this, panel, ev: &TerminalEvent, window, cx| match ev {
            TerminalEvent::Exited => this.close_terminal(panel.clone(), window, cx),
        });
        self.heard.insert(panel.entity_id(), sub);
        self.terminals.push(panel);
    }

    /// Takes the terminal out of the dock (WNDWI-B11).
    fn close_terminal(&mut self, panel: Entity<TerminalPanel>, window: &mut Window, cx: &mut Context<Self>) {
        self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        self.layout_changed(window, cx);
    }

    /// After any change of the dock: the terminals whose tab left are let go, so their shells end
    /// (WNDWI-B13); the window closes with the last (WNDWI-B11); and the organisation is saved, once the
    /// changes stop (WNDWI-B12).
    fn layout_changed(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let dock = self.dock.read(cx);
        let gone: Vec<Entity<TerminalPanel>> =
            self.terminals.iter().filter(|t| dock.panel(PanelId::from(t.entity_id())).is_none()).cloned().collect();
        for panel in &gone {
            self.heard.remove(&panel.entity_id());
        }
        self.terminals.retain(|t| !gone.contains(t));
        if self.terminals.is_empty() {
            window.remove_window();
            return;
        }
        if self.layout.is_some() {
            self.saving = Some(cx.spawn_in(window, async move |this, cx| {
                cx.background_executor().timer(SAVE_DELAY).await;
                let _ = this.update(cx, |this, cx| this.save(cx));
            }));
        }
        cx.notify();
    }

    /// Writes the organisation of the dock (WNDWI-B12).
    fn save(&mut self, cx: &mut Context<Self>) {
        let Some(file) = &self.layout else { return };
        let state = self.dock.read(cx).dump(cx);
        if let Err(e) = file.write(&state) {
            // WNDWI-E03
            eprintln!("polvo: the layout was not saved in {}: {e}", file.path().display());
        }
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

/// Opens the window of Polvo with the user's `shell`, in `cwd`, following `prefs`, with the dock as
/// the installation's layout file kept it.
pub fn open(shell: String, cwd: Option<std::path::PathBuf>, prefs: Prefs, cx: &mut App) {
    let bounds = Bounds::centered(None, gpui_kit::size(px(1000.), px(640.)), cx);
    let options = gpui_kit::WindowOptions {
        window_bounds: Some(gpui_kit::WindowBounds::Windowed(bounds)),
        // GNOME on Wayland draws no frame: the window draws its own title bar and border.
        window_decorations: Some(gpui_kit::WindowDecorations::Client),
        app_id: Some("polvo".into()),
        ..gpui_kit::component::TitleBar::window_options()
    };
    let new_terminal: NewTerminal =
        Rc::new(move |prefs, window, cx| TerminalPanel::spawn(&shell, cwd.as_deref(), prefs, window, cx));
    let opened = gpui_kit::open_window(options, cx, |window, cx| {
        cx.new(|cx| PolvoWindow::new(prefs, new_terminal, LayoutFile::default(), window, cx))
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
