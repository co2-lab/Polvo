// @anchors
//   code: EDIMP
//   ref: EDTRS

//! The code editor, as a tab of the dock (EDTRS): gpui-component's editor, empty or with a file.

use std::path::{Path, PathBuf};

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent, PanelInfo, PanelState};
use gpui_kit::component::input::{Editor, EditorState, InputEvent};
use gpui_kit::{
    App, Context, Entity, EventEmitter, FocusHandle, Focusable, IntoElement, KeyDownEvent, ParentElement as _, Render,
    Role, Styled as _, Subscription, Window, div, prelude::*, px,
};

use crate::keymap::{Action, Keymap}; // @dep: KYIMA

/// The largest file the editor opens (EDTRS-E01).
pub const MAX_OPEN: u64 = 16 << 20;

/// The title of an editor with no file (EDTRS-B01).
const UNTITLED: &str = "Sem título";

/// The language the editor highlights a file in, by its extension (EDTRS-B02): a name the highlighter
/// knows, or plain text.
pub fn language_for(path: &Path) -> &'static str {
    let name = path.file_name().and_then(|n| n.to_str()).unwrap_or_default();
    if name == "Makefile" || name == "makefile" || name == "GNUmakefile" {
        return "make";
    }
    if name == "CMakeLists.txt" {
        return "cmake";
    }
    let ext = path.extension().and_then(|e| e.to_str()).unwrap_or_default().to_lowercase();
    match ext.as_str() {
        "rs" => "rust",
        "go" => "go",
        "py" | "pyi" => "python",
        "js" | "mjs" | "cjs" | "jsx" => "javascript",
        "ts" | "mts" | "cts" => "typescript",
        "tsx" => "tsx",
        "json" | "jsonc" => "json",
        "toml" => "toml",
        "yaml" | "yml" => "yaml",
        "md" | "markdown" | "mdx" => "markdown",
        "sh" | "bash" | "zsh" => "bash",
        "c" | "h" => "c",
        "cc" | "cpp" | "cxx" | "hpp" | "hh" | "hxx" => "cpp",
        "cs" => "csharp",
        "css" | "scss" => "css",
        "html" | "htm" => "html",
        "java" => "java",
        "kt" | "kts" => "kotlin",
        "lua" => "lua",
        "php" => "php",
        "proto" => "proto",
        "rb" => "ruby",
        "scala" => "scala",
        "sql" => "sql",
        "svelte" => "svelte",
        "swift" => "swift",
        "zig" => "zig",
        "ex" | "exs" => "elixir",
        "graphql" | "gql" => "graphql",
        "diff" | "patch" => "diff",
        "mk" => "make",
        _ => "plaintext",
    }
}

/// The text of a file the editor can open, or why it cannot (EDTRS-E01).
fn read_text(path: &Path) -> Result<String, String> {
    let shown = path.display();
    let size = std::fs::metadata(path).map_err(|e| format!("Não foi possível abrir {shown}: {e}"))?.len();
    if size > MAX_OPEN {
        return Err(format!("{shown} tem {} MiB; o editor abre até {} MiB.", size >> 20, MAX_OPEN >> 20));
    }
    let bytes = std::fs::read(path).map_err(|e| format!("Não foi possível abrir {shown}: {e}"))?;
    String::from_utf8(bytes).map_err(|_| format!("{shown} não é texto em UTF-8."))
}

/// The code editor, as a tab of the dock.
pub struct EditorPanel {
    focus: FocusHandle,
    /// The editor, unless the file could not be opened.
    state: Option<Entity<EditorState>>,
    path: Option<PathBuf>,
    modified: bool,
    /// Why the file could not be opened (EDTRS-E01).
    error: Option<String>,
    /// Why the file could not be saved (EDTRS-E02).
    message: Option<String>,
    _heard: Option<Subscription>,
}

impl EditorPanel {
    /// An editor with no text and no file (EDTRS-B01).
    pub fn empty(window: &mut Window, cx: &mut Context<Self>) -> Self {
        let state = cx.new(|cx| EditorState::new(window, cx).language("plaintext").line_number(true));
        Self::with(Some(state), None, None, cx)
    }

    /// An editor with the file at `path` (EDTRS-B02), or the reason it cannot open it (EDTRS-E01).
    pub fn open(path: PathBuf, window: &mut Window, cx: &mut Context<Self>) -> Self {
        match read_text(&path) {
            Ok(text) => {
                let language = language_for(&path);
                let state = cx.new(|cx| EditorState::new(window, cx).language(language).line_number(true).default_value(text));
                Self::with(Some(state), Some(path), None, cx)
            }
            // EDTRS-E01
            Err(reason) => Self::with(None, Some(path), Some(reason), cx),
        }
    }

    fn with(state: Option<Entity<EditorState>>, path: Option<PathBuf>, error: Option<String>, cx: &mut Context<Self>) -> Self {
        // EDTRS-B03
        let heard = state.as_ref().map(|s| {
            cx.subscribe(s, |this, _, ev: &InputEvent, cx| {
                if let InputEvent::Change = ev {
                    this.modified = true;
                    cx.notify();
                }
            })
        });
        EditorPanel { focus: cx.focus_handle(), state, path, modified: false, error, message: None, _heard: heard }
    }

    /// The file of the editor, if it has one.
    pub fn path(&self) -> Option<&Path> {
        self.path.as_deref()
    }

    /// The title of the tab: the file's name, with the mark while there are changes to save (EDTRS-B03).
    pub fn title_text(&self) -> String {
        let name = self.path.as_ref().and_then(|p| p.file_name()).map_or(UNTITLED.to_string(), |n| n.to_string_lossy().into_owned());
        if self.modified { format!("● {name}") } else { name }
    }

    /// The text in the editor; none when the file could not be opened.
    pub fn text(&self, cx: &App) -> Option<String> {
        self.state.as_ref().map(|s| s.read(cx).value().to_string())
    }

    /// The language the editor highlights in.
    pub fn language(&self, cx: &App) -> Option<String> {
        self.state.as_ref().map(|s| s.read(cx).language_name().to_string())
    }

    /// Saves the text in the editor's file, asking where first when it has none (EDTRS-B04).
    pub fn save(&mut self, window: &mut Window, cx: &mut Context<Self>) {
        if self.state.is_none() {
            return;
        }
        if let Some(path) = self.path.clone() {
            self.write(path, cx);
            return;
        }
        let dir = dirs::home_dir().unwrap_or_else(|| PathBuf::from("."));
        let asked = cx.prompt_for_new_path(&dir, None);
        cx.spawn_in(window, async move |this, cx| {
            // a cancelled dialog saves nothing
            let Ok(Ok(Some(path))) = asked.await else { return };
            let _ = this.update(cx, |this, cx| {
                if let Some(state) = &this.state {
                    let language = language_for(&path);
                    state.update(cx, |s, cx| s.set_highlighter(language, cx));
                }
                this.path = Some(path.clone());
                this.write(path, cx);
            });
        })
        .detach();
    }

    fn write(&mut self, path: PathBuf, cx: &mut Context<Self>) {
        let Some(text) = self.text(cx) else { return };
        if let Err(e) = std::fs::write(&path, text) {
            // EDTRS-E02: the mark stays
            eprintln!("polvo: {} was not saved: {e}", path.display());
            self.tell(format!("{} não foi gravado: {e}", path.display()));
        } else {
            self.modified = false;
            self.message = None;
        }
        cx.notify();
    }

    /// Shows the user why something they did failed.
    fn tell(&mut self, message: String) {
        self.message = Some(message);
    }

    fn key_down(&mut self, ev: &KeyDownEvent, window: &mut Window, cx: &mut Context<Self>) {
        let keymap = cx.try_global::<Keymap>().cloned().unwrap_or_default();
        if keymap.action_for(&ev.keystroke.unparse()) == Some(Action::Save) {
            cx.stop_propagation();
            self.save(window, cx);
        }
    }
}

impl EventEmitter<PanelEvent> for EditorPanel {}

impl Focusable for EditorPanel {
    fn focus_handle(&self, cx: &App) -> FocusHandle {
        match &self.state {
            Some(state) => state.read(cx).focus_handle(cx),
            None => self.focus.clone(),
        }
    }
}

impl BasePanel for EditorPanel {
    fn panel_name(&self) -> &'static str {
        "editor"
    }

    /// The file goes with the organisation of the dock, for the editor to come back with it.
    fn dump(&self, _: &App) -> PanelState {
        let mut state = PanelState::new(self.panel_name());
        let path = self.path.as_ref().map(|p| p.to_string_lossy().into_owned());
        state.info = PanelInfo::panel(serde_json::json!({ "path": path }));
        state
    }
}

impl Panel for EditorPanel {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        self.title_text()
    }
}

impl Render for EditorPanel {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let theme = cx.theme();
        let (bg, fg, danger) = (theme.background, theme.foreground, theme.danger);
        let root = div().id("editor").test_support().size_full().flex().flex_col().bg(bg).text_color(fg);
        let Some(state) = &self.state else {
            let reason = self.error.clone().unwrap_or_default();
            return root.track_focus(&self.focus).child(
                div()
                    .id("editor-error")
                    .role(Role::Status)
                    .aria_label(reason.clone())
                    .test_support()
                    .p(px(16.))
                    .text_color(danger)
                    .child(reason),
            );
        };
        let message = self.message.clone().map(|m| {
            div()
                .id("editor-message")
                .role(Role::Status)
                .aria_label(m.clone())
                .test_support()
                .px(px(12.))
                .py(px(4.))
                .text_color(danger)
                .child(m)
        });
        root.on_key_down(cx.listener(Self::key_down))
            .child(div().flex_1().min_h_0().child(Editor::new(state).h_full()))
            .children(message)
    }
}

#[cfg(test)]
#[path = "editor_test.rs"]
mod tests;
