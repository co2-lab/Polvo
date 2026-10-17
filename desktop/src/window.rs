// @anchors
//   code: WNIMW
//   ref: WNDWI

//! The window of Polvo (WNDWI): the title bar the desktop's way, and below it the dock of the
//! terminals.

use std::cell::{Cell, RefCell};
use std::collections::HashMap;
use std::rc::Rc;
use std::sync::Arc;
use std::time::Duration;

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::base::ResizeHandleContext;
use gpui_kit::component::dock::{
    BasePanelView, DockArea, DockAreaRenderer, DockContext, DockEvent, DockLayout, DockPlacement, DockSkin, DropIndicator,
    NodeId, PanelId, PanelState, PanelStyle, TabGroupContext, TabGroupRenderer, panel_handle, register_panel,
};
use gpui_kit::component::{Icon, IconName, InteractiveElementExt as _, Sizable as _};
use gpui_kit::{
    AnyElement, AnyView, App, AppContext as _, Axis, Bounds, Context, Div, Entity, EntityId, Pixels, Stateful, canvas, Focusable as _, FocusHandle, FontWeight, Hsla, IntoElement, KeyDownEvent, MouseButton, ParentElement as _,
    Render, Styled as _, Subscription, Task, Window, div, prelude::*, px,
};

use crate::keymap::{Action, Keymap}; // @dep: KYIMA
use crate::layout::{LAYOUT_VERSION, LayoutFile}; // @dep: DLIDC
use crate::new_tab::{ChooserEvent, NewTabChooser, TabKind}; // @dep: NTIMP
use crate::panel::{TerminalEvent, TerminalPanel}; // @dep: TPITR
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::settings::SettingsFile; // @dep: STIMS
use crate::settings_panel::SettingsPanel; // @dep: SPIST

/// The height of the title bar (WNDWI-B01), GNOME's (libadwaita).
pub const TITLE_H: f32 = 38.;

/// How the window makes a new terminal (WNDWI-B10): the user's shell in the application, terminals
/// with no shell in the tests.
pub type NewTerminal = Rc<dyn Fn(&Prefs, &mut Window, &mut App) -> std::io::Result<Entity<TerminalPanel>>>;

/// The terminal, the kind of tab the window always opens (WNDWI-B10).
pub const TERMINAL: TabKind = TabKind { id: "terminal", label: "Terminal" };

/// The kinds of tab the window can open (WNDWI-B10).
pub const KINDS: [TabKind; 1] = [TERMINAL];

/// The chooser of the new tab, while it is open (WNDWI-B17).
struct Choosing {
    chooser: Entity<NewTabChooser>,
    /// What had the focus before it, to give it back.
    before: Option<FocusHandle>,
    _heard: Subscription,
}

/// The colours of the window for the theme (WNDWI-B07).
#[derive(Clone, Copy, Debug)]
pub struct Colors {
    pub bar: Hsla,
    pub fg: Hsla,
    pub button: Hsla,
    pub button_hover: Hsla,
}

/// The height of the tab bar before it is first measured (WNDWI-B14).
const TAB_BAR_H: f32 = 32.;

/// The dock's look, the gpui-component skin, with the tabs below each group (WNDWI-B14). The dock lays a
/// group out as its tab bar above its content; this renderer draws the bar at the group's bottom edge
/// instead, and keeps the content clear of it.
struct BottomTabs {
    skin: Rc<DockSkin>,
    groups: Rc<dyn TabGroupRenderer>,
}

impl BottomTabs {
    fn new(skin: Rc<DockSkin>) -> Self {
        let inner = skin.tab_group_renderer();
        BottomTabs { skin, groups: Rc::new(BottomTabGroup { inner, bar: Rc::new(Cell::new(px(TAB_BAR_H))) }) }
    }
}

impl DockAreaRenderer for BottomTabs {
    fn frame(&self, window: &mut Window, cx: &mut App) -> Stateful<Div> {
        self.skin.frame(window, cx)
    }

    fn split_frame(&self, node: NodeId, axis: Axis, window: &mut Window, cx: &mut App) -> Stateful<Div> {
        self.skin.split_frame(node, axis, window, cx)
    }

    fn center_frame(&self, window: &mut Window, cx: &mut App) -> Stateful<Div> {
        self.skin.center_frame(window, cx)
    }

    fn render_split_handle(&self, handle: &ResizeHandleContext, window: &mut Window, cx: &mut App) -> Option<AnyElement> {
        self.skin.render_split_handle(handle, window, cx)
    }

    fn render_dock(&self, dock: &DockContext, content: AnyElement, window: &mut Window, cx: &mut App) -> AnyElement {
        self.skin.render_dock(dock, content, window, cx)
    }

    fn build_placeholder(&self, state: &PanelState, window: &mut Window, cx: &mut App) -> Option<Arc<dyn BasePanelView>> {
        self.skin.build_placeholder(state, window, cx)
    }

    fn tab_group_renderer(&self) -> Rc<dyn TabGroupRenderer> {
        self.groups.clone()
    }
}

/// A group of tabs with its bar at the bottom: the bar, measured as it is drawn, and the content above it.
struct BottomTabGroup {
    inner: Rc<dyn TabGroupRenderer>,
    bar: Rc<Cell<Pixels>>,
}

impl TabGroupRenderer for BottomTabGroup {
    fn frame(&self, group: &TabGroupContext, window: &mut Window, cx: &mut App) -> Stateful<Div> {
        self.inner.frame(group, window, cx).relative()
    }

    fn content_frame(&self, group: &TabGroupContext, window: &mut Window, cx: &mut App) -> Stateful<Div> {
        self.inner.content_frame(group, window, cx).mb(self.bar.get())
    }

    fn render_tab_bar(&self, group: &TabGroupContext, window: &mut Window, cx: &mut App) -> AnyElement {
        let bar = self.bar.clone();
        div()
            .absolute()
            .left_0()
            .right_0()
            .bottom_0()
            .child(self.inner.render_tab_bar(group, window, cx))
            .child(canvas(move |bounds, _, _| bar.set(bounds.size.height), |_, _, _, _| {}).absolute().top_0().left_0().size_full())
            .into_any_element()
    }

    fn render_active_panel(&self, panel: AnyView, group: &TabGroupContext, window: &mut Window, cx: &mut App) -> AnyElement {
        self.inner.render_active_panel(panel, group, window, cx)
    }

    fn render_drop_indicator(&self, indicator: DropIndicator, window: &mut Window, cx: &mut App) -> Option<AnyElement> {
        self.inner.render_drop_indicator(indicator, window, cx)
    }

    fn render_empty(&self, group: &TabGroupContext, window: &mut Window, cx: &mut App) -> Option<AnyElement> {
        self.inner.render_empty(group, window, cx)
    }
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
    /// The settings tab, when it is in the dock (WNDWI-B15).
    settings: Option<Entity<SettingsPanel>>,
    /// What the window hears from each terminal, by the terminal's entity.
    heard: HashMap<EntityId, Subscription>,
    /// What the window hears from the dock.
    dock_heard: Option<Subscription>,
    saving: Option<Task<()>>,
    moving: bool,
    choosing: Option<Choosing>,
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
        let mut skin = None;
        let dock = cx.new(|cx| {
            let this = DockSkin::new(cx);
            skin = Some(this.clone());
            DockArea::new("polvo", Some(LAYOUT_VERSION), window, cx).with_renderer(Rc::new(BottomTabs::new(this)))
        });
        if let Some(skin) = skin {
            skin.set_panel_style(PanelStyle::TabBar, cx);
            skin.set_close_button_visible(true, cx);
        }
        let mut this = PolvoWindow {
            prefs,
            dock: dock.clone(),
            new_terminal,
            layout,
            terminals: Vec::new(),
            settings: None,
            heard: HashMap::new(),
            dock_heard: None,
            saving: None,
            moving: false,
            choosing: None,
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
        // WNDWI-B15: the settings tab comes back too.
        let restored: Rc<RefCell<Option<Entity<SettingsPanel>>>> = Rc::default();
        {
            let restored = restored.clone();
            register_panel(cx, "settings", move |_, window, cx| {
                let panel = cx.new(|cx| SettingsPanel::new(cx.try_global::<SettingsFile>().cloned(), window, cx));
                *restored.borrow_mut() = Some(panel.clone());
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
        this.settings = restored.borrow_mut().take();
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
    /// (WNDWI-B13); the window closes with the last tab (WNDWI-B11, WNDWI-B16); and the organisation is
    /// saved, once the changes stop (WNDWI-B12).
    fn layout_changed(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let dock = self.dock.read(cx);
        let gone: Vec<Entity<TerminalPanel>> =
            self.terminals.iter().filter(|t| dock.panel(PanelId::from(t.entity_id())).is_none()).cloned().collect();
        if self.settings.as_ref().is_some_and(|s| dock.panel(PanelId::from(s.entity_id())).is_none()) {
            self.settings = None;
        }
        for panel in &gone {
            self.heard.remove(&panel.entity_id());
        }
        self.terminals.retain(|t| !gone.contains(t));
        if self.terminals.is_empty() && self.settings.is_none() {
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

    /// The chooser of the new tab, while it is open.
    #[cfg(test)]
    pub fn chooser(&self) -> Option<&Entity<NewTabChooser>> {
        self.choosing.as_ref().map(|c| &c.chooser)
    }

    /// Opens the chooser of the new tab's kind (WNDWI-B10), or keeps the one open (WNDWI-B17).
    pub fn new_tab(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if let Some(c) = &self.choosing {
            let focus = c.chooser.read(cx).focus_handle(cx);
            window.focus(&focus, cx);
            return;
        }
        let before = window.focused(cx);
        let chooser = cx.new(|cx| NewTabChooser::new(KINDS.to_vec(), window, cx));
        let heard = cx.subscribe_in(&chooser, window, |this, _, ev: &ChooserEvent, window, cx| {
            let before = this.choosing.take().and_then(|c| c.before);
            match ev {
                ChooserEvent::Chosen(kind) => this.open_kind(*kind, window, cx),
                // WNDWI-B17
                ChooserEvent::Dismissed => {
                    if let Some(focus) = before {
                        window.focus(&focus, cx);
                    }
                }
            }
            cx.notify();
        });
        self.choosing = Some(Choosing { chooser, before, _heard: heard });
        cx.notify();
    }

    /// Opens a tab of the kind chosen (WNDWI-B10).
    fn open_kind(&mut self, kind: TabKind, window: &mut Window, cx: &mut Context<Self>) {
        if kind == TERMINAL {
            self.open_terminal(window, cx);
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

    /// Opens the settings tab with the focus, or shows the one already open (WNDWI-B15).
    pub fn open_settings(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let panel = match &self.settings {
            Some(panel) => {
                let id = PanelId::from(panel.entity_id());
                self.dock.update(cx, |dock, cx| dock.select_panel(id, window, cx));
                panel.clone()
            }
            None => {
                let file = cx.try_global::<SettingsFile>().cloned();
                let panel = cx.new(|cx| SettingsPanel::new(file, window, cx));
                self.dock.update(cx, |dock, cx| dock.add_panel(panel.clone(), DockPlacement::Center, None, window, cx));
                self.settings = Some(panel.clone());
                panel
            }
        };
        let focus = panel.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        cx.notify();
    }

    /// Closes the tab that has the focus (WNDWI-B16).
    fn close_focused(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if let Some(panel) = self.terminals.iter().find(|t| t.read(cx).focus_handle(cx).contains_focused(window, cx)).cloned() {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else if let Some(panel) = self.settings.clone().filter(|s| s.read(cx).focus_handle(cx).contains_focused(window, cx)) {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else {
            return;
        }
        self.layout_changed(window, cx);
    }

    /// The window's shortcuts, from any tab (WNDWI-B16): the tabs leave the window's actions to it.
    fn key_down(&mut self, ev: &KeyDownEvent, window: &mut Window, cx: &mut Context<Self>) {
        let keymap = cx.try_global::<Keymap>().cloned().unwrap_or_default();
        match keymap.action_for(&ev.keystroke.unparse()) {
            Some(Action::NewTab) => self.new_tab(window, cx),
            Some(Action::CloseTab) => self.close_focused(window, cx),
            Some(Action::Settings) => self.open_settings(window, cx),
            Some(Action::Copy | Action::Paste) | None => return,
        }
        cx.stop_propagation();
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
        let new_tab = round("new-tab", IconName::Plus).on_click(cx.listener(|this, _, window, cx| {
            cx.stop_propagation();
            this.new_tab(window, cx);
        }));
        // WNDWI-B15
        let settings = round("settings", IconName::Settings).on_click(cx.listener(|this, _, window, cx| {
            cx.stop_propagation();
            this.open_settings(window, cx);
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
            left = left.child(new_tab).child(settings);
        } else {
            right = right.child(settings).child(new_tab);
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
                .relative()
                .on_key_down(cx.listener(Self::key_down))
                .child(self.title_bar(window, cx))
                .child(div().flex_1().min_h_0().w_full().child(self.dock.clone()))
                .children(self.choosing.as_ref().map(|c| {
                    let chooser = c.chooser.clone();
                    // WNDWI-B17: a click outside the chooser closes it
                    div()
                        .id("new-tab-backdrop")
                        .test_support()
                        .absolute()
                        .top_0()
                        .left_0()
                        .size_full()
                        .flex()
                        .justify_center()
                        .items_start()
                        .pt(px(TITLE_H + 60.))
                        .bg(gpui_kit::hsla(0., 0., 0., 0.25))
                        .on_mouse_down(MouseButton::Left, move |_, _, cx| {
                            chooser.update(cx, |_, cx| cx.emit(ChooserEvent::Dismissed));
                        })
                        .child(div().on_mouse_down(MouseButton::Left, |_, _, cx| cx.stop_propagation()).child(c.chooser.clone()))
                })),
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
