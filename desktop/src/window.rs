// @anchors
//   code: WNIMW
//   ref: WNDWI

//! The window of Polvo (WNDWI): the title bar the desktop's way, and below it the dock of the
//! terminals.

use std::cell::{Cell, RefCell};
use std::collections::HashMap;
use std::path::PathBuf;
use std::rc::Rc;
use std::sync::Arc;
use std::time::Duration;

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::base::ResizeHandleContext;
use gpui_kit::component::dock::{
    BasePanelView, DockArea, DockAreaRenderer, DockContext, DockEvent, DockLayout, DockPlacement, DockSkin, DropIndicator,
    NodeId, PanelId, PanelInfo, PanelState, PanelStyle, TabGroupContext, TabGroupRenderer, panel_handle, register_panel,
};
use gpui_kit::component::{Icon, IconName, InteractiveElementExt as _, Sizable as _};
use gpui_kit::{
    AnyElement, AnyView, App, AppContext as _, Axis, Bounds, Context, Div, Entity, EntityId, Pixels, Stateful, canvas, Focusable as _, FocusHandle, FontWeight, Hsla, IntoElement, KeyDownEvent, MouseButton, ParentElement as _, PathPromptOptions,
    Render, Styled as _, Subscription, Task, Window, div, prelude::*, px,
};

use crate::editor::EditorPanel; // @dep: EDIMP
use crate::files::{FileTree, FileTreeEvent}; // @dep: FLIMP
use crate::git::{GitEvent, GitPanel}; // @dep: GTIMP
use crate::keymap::{Action, Keymap}; // @dep: KYIMA
use crate::layout::{LAYOUT_VERSION, LayoutFile}; // @dep: DLIDC
use crate::new_tab::{ChooserEvent, NewTabChooser, TabKind}; // @dep: NTIMP
use crate::nodes::{NodesEvent, NodesPanel}; // @dep: MQIMP
use crate::panel::{SessionTab, TerminalEvent, TerminalPanel}; // @dep: TPITR
use crate::remote::{RemoteError, Station}; // @dep: RMIMP
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

/// The file tree of the window's folder (WNDWI-B18).
pub const FILES: TabKind = TabKind { id: "files", label: "Árvore de ficheiros" };

/// The git information of the window's folder (WNDWI-B20).
pub const GIT_INFO: TabKind = TabKind { id: "git", label: "Informação de git" };

/// An editor with no file (WNDWI-B19).
pub const EDITOR: TabKind = TabKind { id: "editor", label: "Editor vazio" };

/// A file of the computer, chosen in the system's dialog, in an editor (WNDWI-B19).
pub const OPEN_FILE: TabKind = TabKind { id: "open-file", label: "Abrir um ficheiro…" };

/// The kinds of tab the window can open (WNDWI-B10).
/// The machines and their sessions, with the installation open (WNDWI-B21).
pub const MACHINES: TabKind = TabKind { id: "machines", label: "Máquinas" };

/// The kinds of tab the window can open (WNDWI-B10).
pub const KINDS: [TabKind; 6] = [TERMINAL, FILES, GIT_INFO, EDITOR, OPEN_FILE, MACHINES];

/// How the window opens the panel of a session (WNDWI-B22): connected in the application, with no
/// channel in the tests.
pub type OpenSession = Rc<dyn Fn(SessionTab, &Prefs, &mut Window, &mut App) -> Result<Entity<TerminalPanel>, RemoteError>>;

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
    /// The file trees in the dock (WNDWI-B18).
    trees: Vec<Entity<FileTree>>,
    /// The git tabs in the dock (WNDWI-B20).
    gits: Vec<Entity<GitPanel>>,
    /// The editors in the dock (WNDWI-B19).
    editors: Vec<Entity<EditorPanel>>,
    /// The window's folder (WNDWI-B18).
    root: PathBuf,
    /// The installation, when it opened (WNDWI-B21).
    station: Option<Station>,
    open_session: OpenSession,
    /// The machines tab, when it is in the dock (WNDWI-B21).
    machines: Option<Entity<NodesPanel>>,
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
        root: PathBuf,
        station: Option<Station>,
        open_session: OpenSession,
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
            trees: Vec::new(),
            gits: Vec::new(),
            editors: Vec::new(),
            root,
            station,
            open_session,
            machines: None,
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
        // WNDWI-B12: the trees with their folder, the editors with their file.
        let restored_trees: Rc<RefCell<Vec<Entity<FileTree>>>> = Rc::default();
        {
            let (restored, fallback) = (restored_trees.clone(), this.root.clone());
            register_panel(cx, "files", move |ctx, window, cx| {
                let root = kept(ctx.info(), "root").map_or_else(|| fallback.clone(), PathBuf::from);
                let tree = cx.new(|cx| FileTree::new(root, window, cx));
                restored.borrow_mut().push(tree.clone());
                panel_handle(tree)
            });
        }
        // WNDWI-B21, WNDWI-B22: the machines tab, and the sessions the tabs joined
        let restored_machines: Rc<RefCell<Option<Entity<NodesPanel>>>> = Rc::default();
        if let Some(station) = this.station.clone() {
            let restored = restored_machines.clone();
            let st = station.clone();
            register_panel(cx, "machines", move |_, window, cx| {
                let panel = cx.new(|cx| NodesPanel::new(st.clone(), window, cx));
                *restored.borrow_mut() = Some(panel.clone());
                panel_handle(panel)
            });
            let (open, prefs, built) = (this.open_session.clone(), this.prefs.clone(), built.clone());
            register_panel(cx, "session", move |ctx, window, cx| {
                let (machine, id) = (kept(ctx.info(), "machine").unwrap_or_default(), kept(ctx.info(), "id").unwrap_or_default());
                let writable = station.sessions(&machine).ok().and_then(|l| l.into_iter().find(|s| s.id == id)).map(|s| s.writable);
                let tab = SessionTab { station: station.clone(), machine: machine.clone(), id: id.clone(), owned: false, writable: writable.unwrap_or(false) };
                let panel = match writable.map(|_| open(tab, &prefs, window, cx)) {
                    Some(Ok(panel)) => panel,
                    // WNDWI-B22: a session that is gone says why
                    Some(Err(e)) => failed_panel(&format!("the session {id} of {machine} did not connect: {e}"), &prefs, window, cx),
                    None => failed_panel(&format!("the session {id} of {machine} is gone"), &prefs, window, cx),
                };
                built.borrow_mut().push(panel.clone());
                panel_handle(panel)
            });
        }
        let restored_gits: Rc<RefCell<Vec<Entity<GitPanel>>>> = Rc::default();
        {
            let (restored, fallback) = (restored_gits.clone(), this.root.clone());
            register_panel(cx, "git", move |ctx, window, cx| {
                let root = kept(ctx.info(), "root").map_or_else(|| fallback.clone(), PathBuf::from);
                let tab = cx.new(|cx| GitPanel::new(root, window, cx));
                restored.borrow_mut().push(tab.clone());
                panel_handle(tab)
            });
        }
        let restored_editors: Rc<RefCell<Vec<Entity<EditorPanel>>>> = Rc::default();
        {
            let restored = restored_editors.clone();
            register_panel(cx, "editor", move |ctx, window, cx| {
                let editor = match kept(ctx.info(), "path") {
                    Some(path) => cx.new(|cx| EditorPanel::open(PathBuf::from(path), window, cx)),
                    None => cx.new(|cx| EditorPanel::empty(window, cx)),
                };
                restored.borrow_mut().push(editor.clone());
                panel_handle(editor)
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
        for tree in restored_trees.borrow_mut().drain(..) {
            this.watch_tree(tree, window, cx);
        }
        if let Some(panel) = restored_machines.borrow_mut().take() {
            this.watch_machines(panel, window, cx);
        }
        for tab in restored_gits.borrow_mut().drain(..) {
            this.watch_git(tab, window, cx);
        }
        this.editors.extend(restored_editors.borrow_mut().drain(..));
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
        let in_dock = |id: EntityId| dock.panel(PanelId::from(id)).is_some();
        if self.settings.as_ref().is_some_and(|s| !in_dock(s.entity_id())) {
            self.settings = None;
        }
        if let Some(id) = self.machines.as_ref().map(|m| m.entity_id()).filter(|id| !in_dock(*id)) {
            self.machines = None;
            self.heard.remove(&id);
        }
        let trees_gone: Vec<EntityId> = self
            .trees
            .iter()
            .map(|t| t.entity_id())
            .chain(self.gits.iter().map(|g| g.entity_id()))
            .filter(|id| !in_dock(*id))
            .collect();
        self.trees.retain(|t| !trees_gone.contains(&t.entity_id()));
        self.gits.retain(|g| !trees_gone.contains(&g.entity_id()));
        self.editors.retain(|e| in_dock(e.entity_id()));
        for id in trees_gone {
            self.heard.remove(&id);
        }
        for panel in &gone {
            self.heard.remove(&panel.entity_id());
        }
        self.terminals.retain(|t| !gone.contains(t));
        if self.terminals.is_empty()
            && self.settings.is_none()
            && self.machines.is_none()
            && self.trees.is_empty()
            && self.gits.is_empty()
            && self.editors.is_empty()
        {
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
        // WNDWI-B21: the machines only with the installation
        let kinds: Vec<TabKind> = KINDS.into_iter().filter(|k| *k != MACHINES || self.station.is_some()).collect();
        let chooser = cx.new(|cx| NewTabChooser::new(kinds, window, cx));
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
        } else if kind == FILES {
            self.open_tree(window, cx);
        } else if kind == GIT_INFO {
            self.open_git(window, cx);
        } else if kind == EDITOR {
            let editor = cx.new(|cx| EditorPanel::empty(window, cx));
            self.show_editor(editor, window, cx);
        } else if kind == OPEN_FILE {
            self.ask_file(window, cx);
        } else if kind == MACHINES {
            self.open_machines(window, cx);
        }
    }

    /// Opens the machines tab with the focus, or shows the one open (WNDWI-B21).
    fn open_machines(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let Some(station) = self.station.clone() else { return };
        let panel = match &self.machines {
            Some(panel) => {
                let id = PanelId::from(panel.entity_id());
                self.dock.update(cx, |dock, cx| dock.select_panel(id, window, cx));
                panel.clone()
            }
            None => {
                let panel = cx.new(|cx| NodesPanel::new(station, window, cx));
                self.dock.update(cx, |dock, cx| dock.add_panel(panel.clone(), DockPlacement::Center, None, window, cx));
                self.watch_machines(panel.clone(), window, cx);
                panel
            }
        };
        let focus = panel.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        cx.notify();
    }

    /// Hears the sessions the machines tab opens (WNDWI-B22).
    fn watch_machines(&mut self, panel: Entity<NodesPanel>, window: &mut Window, cx: &mut Context<Self>) {
        let sub = cx.subscribe_in(&panel, window, |this, _, ev: &NodesEvent, window, cx| match ev {
            NodesEvent::Open(tab) => this.open_session_tab(tab.clone(), window, cx),
        });
        self.heard.insert(panel.entity_id(), sub);
        self.machines = Some(panel);
    }

    /// Opens a session in a terminal tab, with the focus; one that does not connect says why (WNDWI-B22).
    fn open_session_tab(&mut self, tab: SessionTab, window: &mut Window, cx: &mut Context<Self>) {
        let (machine, id) = (tab.machine.clone(), tab.id.clone());
        let panel = match (self.open_session)(tab, &self.prefs, window, cx) {
            Ok(panel) => panel,
            Err(e) => failed_panel(&format!("the session {id} of {machine} did not connect: {e}"), &self.prefs, window, cx),
        };
        self.dock.update(cx, |dock, cx| dock.add_panel(panel.clone(), DockPlacement::Center, None, window, cx));
        let focus = panel.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        self.watch(panel, window, cx);
        cx.notify();
    }

    /// Opens the file tree of the window's folder, with the focus (WNDWI-B18).
    fn open_tree(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let root = self.root.clone();
        let tree = cx.new(|cx| FileTree::new(root, window, cx));
        self.dock.update(cx, |dock, cx| dock.add_panel(tree.clone(), DockPlacement::Center, None, window, cx));
        let focus = tree.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        self.watch_tree(tree, window, cx);
        cx.notify();
    }

    /// Hears the files the tree opens (WNDWI-B18).
    fn watch_tree(&mut self, tree: Entity<FileTree>, window: &mut Window, cx: &mut Context<Self>) {
        let sub = cx.subscribe_in(&tree, window, |this, _, ev: &FileTreeEvent, window, cx| match ev {
            FileTreeEvent::Open(path) => this.open_file(path.clone(), window, cx),
        });
        self.heard.insert(tree.entity_id(), sub);
        self.trees.push(tree);
    }

    /// Opens the git tab of the window's folder, with the focus (WNDWI-B20).
    fn open_git(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let root = self.root.clone();
        let tab = cx.new(|cx| GitPanel::new(root, window, cx));
        self.dock.update(cx, |dock, cx| dock.add_panel(tab.clone(), DockPlacement::Center, None, window, cx));
        let focus = tab.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        self.watch_git(tab, window, cx);
        cx.notify();
    }

    /// Hears the changed files the git tab opens (WNDWI-B20).
    fn watch_git(&mut self, tab: Entity<GitPanel>, window: &mut Window, cx: &mut Context<Self>) {
        let sub = cx.subscribe_in(&tab, window, |this, _, ev: &GitEvent, window, cx| match ev {
            GitEvent::Open(path) => this.open_file(path.clone(), window, cx),
        });
        self.heard.insert(tab.entity_id(), sub);
        self.gits.push(tab);
    }

    /// Asks for a file in the system's dialog and opens it (WNDWI-B19); a cancelled dialog opens nothing.
    fn ask_file(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        let asked = cx.prompt_for_paths(PathPromptOptions { files: true, directories: false, multiple: false, prompt: None });
        cx.spawn_in(window, async move |this, cx| {
            let Ok(Ok(Some(paths))) = asked.await else { return };
            let Some(path) = paths.into_iter().next() else { return };
            let _ = this.update_in(cx, |this, window, cx| this.open_file(path, window, cx));
        })
        .detach();
    }

    /// Opens the file in an editor, or shows the editor that has it (WNDWI-B18, WNDWI-B19).
    fn open_file(&mut self, path: PathBuf, window: &mut Window, cx: &mut Context<Self>) {
        if let Some(editor) = self.editors.iter().find(|e| e.read(cx).path() == Some(path.as_path())).cloned() {
            let id = PanelId::from(editor.entity_id());
            self.dock.update(cx, |dock, cx| dock.select_panel(id, window, cx));
            let focus = editor.read(cx).focus_handle(cx);
            window.focus(&focus, cx);
            return;
        }
        let editor = cx.new(|cx| EditorPanel::open(path, window, cx));
        self.show_editor(editor, window, cx);
    }

    fn show_editor(&mut self, editor: Entity<EditorPanel>, window: &mut Window, cx: &mut Context<Self>) {
        self.dock.update(cx, |dock, cx| dock.add_panel(editor.clone(), DockPlacement::Center, None, window, cx));
        let focus = editor.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        self.editors.push(editor);
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
        let focused = |f: FocusHandle| f.contains_focused(window, cx);
        if let Some(panel) = self.terminals.iter().find(|t| focused(t.read(cx).focus_handle(cx))).cloned() {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else if let Some(panel) = self.settings.clone().filter(|s| focused(s.read(cx).focus_handle(cx))) {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else if let Some(panel) = self.trees.iter().find(|t| focused(t.read(cx).focus_handle(cx))).cloned() {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else if let Some(panel) = self.machines.clone().filter(|m| focused(m.read(cx).focus_handle(cx))) {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else if let Some(panel) = self.gits.iter().find(|g| focused(g.read(cx).focus_handle(cx))).cloned() {
            self.dock.update(cx, |dock, cx| dock.remove_panel(panel, window, cx));
        } else if let Some(panel) = self.editors.iter().find(|e| focused(e.read(cx).focus_handle(cx))).cloned() {
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
            Some(Action::Copy | Action::Paste | Action::Save) | None => return,
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

/// A terminal panel that says why it has no shell (WNDWI-B22, WNDWI-E04).
fn failed_panel(reason: &str, prefs: &Prefs, window: &mut Window, cx: &mut App) -> Entity<TerminalPanel> {
    match TerminalPanel::failed(reason, "", prefs, window, cx) {
        Ok(panel) => panel,
        Err(e) => panic!("polvo: not even an empty terminal could be made: {e}"),
    }
}

/// A text the dock kept for a panel (WNDWI-B12).
fn kept(info: &PanelInfo, key: &str) -> Option<String> {
    match info {
        PanelInfo::Panel(v) => v.get(key).and_then(|x| x.as_str()).map(str::to_string),
        _ => None,
    }
}

/// Opens the window of Polvo with the user's `shell`, in `cwd`, following `prefs`, with the dock as
/// the installation's layout file kept it.
pub fn open(shell: String, cwd: Option<std::path::PathBuf>, prefs: Prefs, station: Option<Station>, cx: &mut App) {
    let bounds = Bounds::centered(None, gpui_kit::size(px(1000.), px(640.)), cx);
    let options = gpui_kit::WindowOptions {
        window_bounds: Some(gpui_kit::WindowBounds::Windowed(bounds)),
        // GNOME on Wayland draws no frame: the window draws its own title bar and border.
        window_decorations: Some(gpui_kit::WindowDecorations::Client),
        app_id: Some("polvo".into()),
        ..gpui_kit::component::TitleBar::window_options()
    };
    let root = cwd.clone().unwrap_or_else(|| PathBuf::from("."));
    let new_terminal: NewTerminal = match station.clone() {
        // WNDWI-R0008: a session of this machine, for the others to mirror it
        Some(st) => Rc::new(move |prefs, window, cx| {
            let id = st.new_session(crate::remote::LOCAL, 24, 80).map_err(|e| std::io::Error::other(e.0))?;
            let tab = SessionTab { station: st.clone(), machine: crate::remote::LOCAL.into(), id, owned: true, writable: true };
            TerminalPanel::session(tab, prefs, window, cx).map_err(|e| std::io::Error::other(e.0))
        }),
        None => Rc::new(move |prefs, window, cx| TerminalPanel::spawn(&shell, cwd.as_deref(), prefs, window, cx)),
    };
    let open_session: OpenSession = Rc::new(TerminalPanel::session);
    let opened = gpui_kit::open_window(options, cx, |window, cx| {
        cx.new(|cx| PolvoWindow::new(prefs, new_terminal, LayoutFile::default(), root, station, open_session, window, cx))
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
