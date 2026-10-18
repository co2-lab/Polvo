// @anchors
//   code: FLIMP
//   ref: FLTRE

//! The file tree of a folder, as a tab of the dock (FLTRE): each folder is read when it is expanded, from this
//! machine or from the node of another (FLTRE-B05).

use std::ops::Range;
use std::path::{Path, PathBuf};

use crate::remote::Station; // @dep: RMIMP
use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent, PanelInfo, PanelState};
use gpui_kit::component::{Icon, IconName, Sizable as _};
use gpui_kit::{
    App, Context, EventEmitter, FocusHandle, Focusable, IntoElement, KeyDownEvent, ParentElement as _, Render, Role,
    ScrollStrategy, SharedString, Styled as _, UniformListScrollHandle, Window, div, prelude::*, px, uniform_list,
};

/// Where the files are: this machine, or the node of a machine (FLTRE-B05).
#[derive(Clone)]
pub enum FileSource {
    Local,
    Node { station: Station, machine: String },
}

impl FileSource {
    /// The machine of a node's files, or none for this machine's.
    pub fn machine(&self) -> Option<&str> {
        match self {
            FileSource::Local => None,
            FileSource::Node { machine, .. } => Some(machine),
        }
    }
}

/// What the tree tells the window.
#[derive(Clone, Debug, PartialEq)]
pub enum FileTreeEvent {
    /// The user opened this file (FLTRE-B03).
    Open(PathBuf),
}

/// The indentation of each level.
const INDENT: f32 = 16.;

#[derive(Clone, Debug)]
enum Kind {
    Folder { expanded: bool },
    File,
    /// Why the folder above could not be read (FLTRE-E01), in the row's name.
    Error,
    /// The folder above is being read from the node (FLTRE-B05); the row's path is the folder's.
    Loading,
}

/// A line of the tree.
#[derive(Clone, Debug)]
struct Row {
    path: PathBuf,
    name: String,
    depth: usize,
    kind: Kind,
}

/// The rows of a folder's content, at `depth` (FLTRE-B01), or the line with why it cannot be read
/// (FLTRE-E01).
fn read_folder(path: &Path, depth: usize) -> Vec<Row> {
    let entries = match std::fs::read_dir(path) {
        Ok(entries) => entries,
        Err(e) => {
            // FLTRE-E01
            return vec![Row { path: path.to_path_buf(), name: e.to_string(), depth, kind: Kind::Error }];
        }
    };
    let rows: Vec<Row> = entries
        .filter_map(Result::ok)
        .map(|e| {
            let path = e.path();
            // a link to a folder is a folder
            let folder = std::fs::metadata(&path).is_ok_and(|m| m.is_dir());
            let kind = if folder { Kind::Folder { expanded: false } } else { Kind::File };
            Row { name: e.file_name().to_string_lossy().into_owned(), path, depth, kind }
        })
        .collect();
    sorted(rows)
}

/// The rows of a folder of a node's machine, read by the node (FLTRE-B05), or the line with why not.
fn read_node_folder(station: &Station, machine: &str, path: &Path, depth: usize) -> Vec<Row> {
    match station.list_dir(machine, &path.to_string_lossy()) {
        Ok(entries) => sorted(
            entries
                .into_iter()
                .map(|e| {
                    let kind = if e.dir { Kind::Folder { expanded: false } } else { Kind::File };
                    Row { path: path.join(&e.name), name: e.name, depth, kind }
                })
                .collect(),
        ),
        // FLTRE-E01
        Err(e) => vec![Row { path: path.to_path_buf(), name: e.0, depth, kind: Kind::Error }],
    }
}

/// Folders first, then by name without case.
fn sorted(mut rows: Vec<Row>) -> Vec<Row> {
    rows.sort_by(|a, b| {
        let folder = |r: &Row| matches!(r.kind, Kind::Folder { .. });
        folder(b).cmp(&folder(a)).then_with(|| a.name.to_lowercase().cmp(&b.name.to_lowercase())).then_with(|| a.name.cmp(&b.name))
    });
    rows
}

/// The file tree of a folder.
pub struct FileTree {
    focus: FocusHandle,
    source: FileSource,
    root: PathBuf,
    rows: Vec<Row>,
    selected: usize,
    scroll: UniformListScrollHandle,
}

impl FileTree {
    /// The tree of `root` of this machine, with its content shown (FLTRE-B01).
    #[cfg(test)]
    pub fn new(root: PathBuf, window: &mut Window, cx: &mut Context<Self>) -> Self {
        Self::on(FileSource::Local, root, window, cx)
    }

    /// The tree of `root` of the machine of `source` (FLTRE-B05).
    pub fn on(source: FileSource, root: PathBuf, _: &mut Window, cx: &mut Context<Self>) -> Self {
        let mut tree = FileTree {
            focus: cx.focus_handle(),
            source,
            root: root.clone(),
            rows: Vec::new(),
            selected: 0,
            scroll: UniformListScrollHandle::new(),
        };
        tree.read(0, root, 0, cx);
        tree
    }

    /// Where the files of the tree are.
    pub fn source(&self) -> &FileSource {
        &self.source
    }

    /// Puts the content of the folder at `path` at `at`: at once from this machine, or a line "a ler…" until
    /// the node answers (FLTRE-B05).
    fn read(&mut self, at: usize, path: PathBuf, depth: usize, cx: &mut Context<Self>) {
        let (station, machine) = match &self.source {
            FileSource::Local => {
                let content = read_folder(&path, depth);
                self.rows.splice(at..at, content);
                return;
            }
            FileSource::Node { station, machine } => (station.clone(), machine.clone()),
        };
        self.rows.insert(at, Row { path: path.clone(), name: "a ler…".into(), depth, kind: Kind::Loading });
        let task = cx.background_executor().spawn(async move {
            let content = read_node_folder(&station, &machine, &path, depth);
            (path, content)
        });
        // a closed tree no longer takes the answer
        cx.spawn(async move |this, cx| {
            let (path, content) = task.await;
            let _ = this.update(cx, |this, cx| {
                // the folder may have been collapsed meanwhile, and then there is no line to fill
                if let Some(i) = this.rows.iter().position(|r| matches!(r.kind, Kind::Loading) && r.path == path) {
                    this.rows.splice(i..i + 1, content);
                    cx.notify();
                }
            });
        })
        .detach();
    }

    /// The folder of the tree.
    #[cfg(test)]
    pub fn root(&self) -> &Path {
        &self.root
    }

    #[cfg(test)]
    /// The lines of the tree as text: two spaces a level, folders with "/", and "! " before a reason.
    pub fn listing(&self) -> Vec<String> {
        self.rows
            .iter()
            .map(|r| {
                let indent = "  ".repeat(r.depth);
                match &r.kind {
                    Kind::Folder { .. } => format!("{indent}{}/", r.name),
                    Kind::File => format!("{indent}{}", r.name),
                    Kind::Error => format!("{indent}! {}", r.name),
                    Kind::Loading => format!("{indent}… a ler"),
                }
            })
            .collect()
    }

    /// The name of the chosen line.
    #[cfg(test)]
    pub fn chosen(&self) -> String {
        self.rows.get(self.selected).map(|r| r.name.clone()).unwrap_or_default()
    }

    /// The end of the rows inside the folder at `i`.
    fn end_of(&self, i: usize) -> usize {
        let depth = self.rows[i].depth;
        let mut end = i + 1;
        while end < self.rows.len() && self.rows[end].depth > depth {
            end += 1;
        }
        end
    }

    /// FLTRE-B02: reads the folder again each time it is expanded.
    fn expand(&mut self, i: usize, cx: &mut Context<Self>) {
        let Some(Row { kind: Kind::Folder { expanded: false }, path, depth, .. }) = self.rows.get(i).cloned() else { return };
        self.rows[i].kind = Kind::Folder { expanded: true };
        self.read(i + 1, path, depth + 1, cx);
    }

    /// FLTRE-B02
    fn collapse(&mut self, i: usize) {
        if !matches!(self.rows.get(i).map(|r| &r.kind), Some(Kind::Folder { expanded: true })) {
            return;
        }
        let end = self.end_of(i);
        self.rows.drain(i + 1..end);
        self.rows[i].kind = Kind::Folder { expanded: false };
        if self.selected > i && self.selected < end {
            self.selected = i;
        }
    }

    /// What a click or the Enter does to the line at `i`: a folder expands or collapses, a file opens.
    fn activate(&mut self, i: usize, cx: &mut Context<Self>) {
        let Some(row) = self.rows.get(i) else { return };
        match row.kind {
            Kind::Folder { expanded: false } => self.expand(i, cx),
            Kind::Folder { expanded: true } => self.collapse(i),
            // FLTRE-B03
            Kind::File => cx.emit(FileTreeEvent::Open(row.path.clone())),
            Kind::Error | Kind::Loading => {}
        }
    }

    fn select(&mut self, i: usize) {
        self.selected = i.min(self.rows.len().saturating_sub(1));
        self.scroll.scroll_to_item(self.selected, ScrollStrategy::Center);
    }

    /// FLTRE-B02, FLTRE-B03, FLTRE-B04
    fn key_down(&mut self, ev: &KeyDownEvent, _: &mut Window, cx: &mut Context<Self>) {
        let ks = &ev.keystroke;
        if ks.modifiers.modified() || self.rows.is_empty() {
            return;
        }
        let i = self.selected;
        match ks.key.as_str() {
            "down" => self.select(i + 1),
            "up" => self.select(i.saturating_sub(1)),
            "right" => self.expand(i, cx),
            "enter" => self.activate(i, cx),
            "left" => match self.rows[i].kind {
                Kind::Folder { expanded: true } => self.collapse(i),
                _ => {
                    // FLTRE-B04: the folder it is in
                    let depth = self.rows[i].depth;
                    if let Some(parent) = (0..i).rev().find(|&j| self.rows[j].depth < depth) {
                        self.select(parent);
                    }
                }
            },
            _ => return,
        }
        cx.stop_propagation();
        cx.notify();
    }

    fn render_rows(&mut self, range: Range<usize>, _: &mut Window, cx: &mut Context<Self>) -> Vec<impl IntoElement + use<>> {
        let theme = cx.theme();
        let (active, muted, danger) = (theme.list_active, theme.muted_foreground, theme.danger);
        range
            .filter_map(|i| self.rows.get(i).map(|r| (i, r.clone())))
            .map(|(i, row)| {
                let rel = row.path.strip_prefix(&self.root).unwrap_or(&row.path).to_string_lossy().into_owned();
                let selected = i == self.selected;
                let (icon, expanded) = match row.kind {
                    Kind::Folder { expanded } => (Some(if expanded { IconName::FolderOpen } else { IconName::Folder }), Some(expanded)),
                    Kind::File => (Some(IconName::File), None),
                    Kind::Error | Kind::Loading => (None, None),
                };
                let id = match row.kind {
                    Kind::Error => SharedString::from(format!("error-{rel}")),
                    Kind::Loading => SharedString::from(format!("loading-{rel}")),
                    _ => SharedString::from(format!("entry-{rel}")),
                };
                div()
                    .id(id)
                    .role(Role::TreeItem)
                    .aria_label(row.name.clone())
                    .aria_selected(selected)
                    .aria_level(row.depth + 1)
                    .when_some(expanded, |d, e| d.aria_expanded(e))
                    .test_support()
                    .h(px(24.))
                    .flex()
                    .items_center()
                    .gap(px(6.))
                    .pl(px(8. + INDENT * row.depth as f32))
                    .pr(px(8.))
                    .whitespace_nowrap()
                    .cursor_pointer()
                    .when(selected, |d| d.bg(active))
                    .when(matches!(row.kind, Kind::Error), |d| d.text_color(danger))
                    .when(matches!(row.kind, Kind::Loading), |d| d.text_color(muted))
                    .on_click(cx.listener(move |this, _, window, cx| {
                        window.focus(&this.focus, cx);
                        this.selected = i;
                        this.activate(i, cx);
                        cx.notify();
                    }))
                    .children(icon.map(|icon| Icon::new(icon).small().text_color(muted)))
                    .child(row.name.clone())
            })
            .collect()
    }
}

impl EventEmitter<FileTreeEvent> for FileTree {}
impl EventEmitter<PanelEvent> for FileTree {}

impl Focusable for FileTree {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl BasePanel for FileTree {
    fn panel_name(&self) -> &'static str {
        "files"
    }

    /// The folder goes with the organisation of the dock, for the tree to come back with it.
    fn dump(&self, _: &App) -> PanelState {
        let mut state = PanelState::new(self.panel_name());
        let mut info = serde_json::json!({ "root": self.root.to_string_lossy() });
        // FLTRE-B05: the machine goes with the folder
        if let Some(machine) = self.source.machine() {
            info["machine"] = machine.into();
        }
        state.info = PanelInfo::panel(info);
        state
    }
}

impl Panel for FileTree {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        let name = self.root.file_name().map_or_else(|| self.root.to_string_lossy().into_owned(), |n| n.to_string_lossy().into_owned());
        match self.source.machine() {
            // FLTRE-B05: the title says the machine
            Some(machine) => format!("{machine}: {name}"),
            None => name,
        }
    }
}

impl Render for FileTree {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let theme = cx.theme();
        let (bg, fg) = (theme.background, theme.foreground);
        div()
            .id("files")
            .test_support()
            .size_full()
            .bg(bg)
            .text_color(fg)
            .track_focus(&self.focus)
            .on_key_down(cx.listener(Self::key_down))
            .child(uniform_list("files-rows", self.rows.len(), cx.processor(Self::render_rows)).track_scroll(&self.scroll).size_full())
    }
}

#[cfg(test)]
#[path = "files_test.rs"]
mod tests;
