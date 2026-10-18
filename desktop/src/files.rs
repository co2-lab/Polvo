// @anchors
//   code: FLIMP
//   ref: FLTRE

//! The file tree of a folder, as a tab of the dock (FLTRE): each folder is read when it is expanded.

use std::ops::Range;
use std::path::{Path, PathBuf};

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent, PanelInfo, PanelState};
use gpui_kit::component::{Icon, IconName, Sizable as _};
use gpui_kit::{
    App, Context, EventEmitter, FocusHandle, Focusable, IntoElement, KeyDownEvent, ParentElement as _, Render, Role,
    ScrollStrategy, SharedString, Styled as _, UniformListScrollHandle, Window, div, prelude::*, px, uniform_list,
};

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
    let mut rows: Vec<Row> = entries
        .filter_map(Result::ok)
        .map(|e| {
            let path = e.path();
            // a link to a folder is a folder
            let folder = std::fs::metadata(&path).is_ok_and(|m| m.is_dir());
            let kind = if folder { Kind::Folder { expanded: false } } else { Kind::File };
            Row { name: e.file_name().to_string_lossy().into_owned(), path, depth, kind }
        })
        .collect();
    rows.sort_by(|a, b| {
        let folder = |r: &Row| matches!(r.kind, Kind::Folder { .. });
        folder(b).cmp(&folder(a)).then_with(|| a.name.to_lowercase().cmp(&b.name.to_lowercase())).then_with(|| a.name.cmp(&b.name))
    });
    rows
}

/// The file tree of a folder.
pub struct FileTree {
    focus: FocusHandle,
    root: PathBuf,
    rows: Vec<Row>,
    selected: usize,
    scroll: UniformListScrollHandle,
}

impl FileTree {
    /// The tree of `root`, with its content shown (FLTRE-B01).
    pub fn new(root: PathBuf, _: &mut Window, cx: &mut Context<Self>) -> Self {
        let rows = read_folder(&root, 0);
        FileTree { focus: cx.focus_handle(), root, rows, selected: 0, scroll: UniformListScrollHandle::new() }
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
    fn expand(&mut self, i: usize) {
        let Some(Row { kind: Kind::Folder { expanded: false }, path, depth, .. }) = self.rows.get(i).cloned() else { return };
        self.rows[i].kind = Kind::Folder { expanded: true };
        let content = read_folder(&path, depth + 1);
        self.rows.splice(i + 1..i + 1, content);
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
            Kind::Folder { expanded: false } => self.expand(i),
            Kind::Folder { expanded: true } => self.collapse(i),
            // FLTRE-B03
            Kind::File => cx.emit(FileTreeEvent::Open(row.path.clone())),
            Kind::Error => {}
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
            "right" => self.expand(i),
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
                    Kind::Error => (None, None),
                };
                let id = match row.kind {
                    Kind::Error => SharedString::from(format!("error-{rel}")),
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
        state.info = PanelInfo::panel(serde_json::json!({ "root": self.root.to_string_lossy() }));
        state
    }
}

impl Panel for FileTree {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        self.root.file_name().map_or_else(|| self.root.to_string_lossy().into_owned(), |n| n.to_string_lossy().into_owned())
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
