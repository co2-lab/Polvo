// @anchors
//   code: GTIMP
//   ref: GITRS

//! The git information of a folder, as a tab of the dock (GITRS), read from the user's own git.

use std::path::{Path, PathBuf};
use std::process::Command;

use gpui_kit::base::TestSupportExt as _;
use gpui_kit::component::ActiveTheme as _;
use gpui_kit::component::dock::{BasePanel, Panel, PanelEvent, PanelInfo, PanelState};
use gpui_kit::component::{Icon, Sizable as _};
use gpui_kit::{
    App, Context, EventEmitter, FocusHandle, Focusable, FontWeight, IntoElement, ParentElement as _, Render, Role,
    SharedString, Styled as _, Task, Window, div, prelude::*, px,
};

/// The git the tab reads from: the user's, the one of the terminal.
pub const GIT: &str = "git";

/// The commits the tab shows (GITRS-B03).
pub const LOG_LIMIT: usize = 30;

/// Where the repository is (GITRS-B01).
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Head {
    /// On a branch, with how many commits it is ahead of and behind its upstream, when it has one.
    Branch { name: String, upstream: Option<(u32, u32)> },
    /// On no branch: the short hash of the commit.
    Detached(String),
}

/// The groups of changed files (GITRS-B02), in the order they are shown.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Group {
    Conflict,
    Staged,
    Changed,
    Untracked,
}

impl Group {
    const ALL: [Group; 4] = [Group::Conflict, Group::Staged, Group::Changed, Group::Untracked];

    fn id(self) -> &'static str {
        match self {
            Group::Conflict => "conflict",
            Group::Staged => "staged",
            Group::Changed => "changed",
            Group::Untracked => "untracked",
        }
    }

    fn label(self) -> &'static str {
        match self {
            Group::Conflict => "Em conflito",
            Group::Staged => "Preparados",
            Group::Changed => "Alterados",
            Group::Untracked => "Não seguidos",
        }
    }
}

/// A changed file, by its path in the repository's folder.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Change {
    pub group: Group,
    pub path: String,
}

/// A commit of the branch (GITRS-B03).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Commit {
    pub hash: String,
    pub subject: String,
    pub author: String,
    pub date: String,
}

/// The state of a repository.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Repo {
    pub head: Head,
    pub changes: Vec<Change>,
    pub commits: Vec<Commit>,
}

/// Why the repository could not be read.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GitError {
    /// The folder is in no repository (GITRS-E01).
    NotARepo,
    /// Git is missing or failed, with what it said (GITRS-E02).
    Failed(String),
}

/// Runs git in `root`, in English for the messages to be known, without taking the index's lock.
fn run(program: &Path, root: &Path, args: &[&str]) -> Result<Vec<u8>, GitError> {
    let out = Command::new(program)
        .args(args)
        .current_dir(root)
        .env("LC_ALL", "C")
        .env("GIT_OPTIONAL_LOCKS", "0")
        .output()
        // GITRS-E02
        .map_err(|e| GitError::Failed(format!("{}: {e}", program.display())))?;
    if out.status.success() {
        return Ok(out.stdout);
    }
    let said = String::from_utf8_lossy(&out.stderr).trim().to_string();
    if said.contains("not a git repository") {
        // GITRS-E01
        return Err(GitError::NotARepo);
    }
    // GITRS-E02
    Err(GitError::Failed(format!("{} {}: {said}", program.display(), args.join(" "))))
}

/// The state of the repository of `root`, as `program` (the user's git) gives it.
pub fn read_repo(program: &Path, root: &Path) -> Result<Repo, GitError> {
    let status = run(program, root, &["status", "--porcelain=v2", "--branch", "-z"])?;
    let status = String::from_utf8_lossy(&status);
    let (mut oid, mut branch, mut ab) = (String::new(), String::new(), None);
    let mut changes = Vec::new();
    let mut records = status.split('\0').filter(|r| !r.is_empty());
    while let Some(record) = records.next() {
        let mut add = |group, path: &str| changes.push(Change { group, path: path.to_string() });
        let xy = |r: &str| r.split(' ').nth(1).unwrap_or("..").to_string();
        if let Some(v) = record.strip_prefix("# branch.oid ") {
            oid = v.to_string();
        } else if let Some(v) = record.strip_prefix("# branch.head ") {
            branch = v.to_string();
        } else if let Some(v) = record.strip_prefix("# branch.ab ") {
            let n = |s: Option<&str>| s.and_then(|s| s[1..].parse().ok()).unwrap_or(0);
            let mut parts = v.split(' ');
            ab = Some((n(parts.next()), n(parts.next())));
        } else if let Some(path) = record.strip_prefix("? ") {
            add(Group::Untracked, path);
        } else if record.starts_with("u ") {
            add(Group::Conflict, record.splitn(11, ' ').nth(10).unwrap_or_default());
        } else if record.starts_with("1 ") || record.starts_with("2 ") {
            let fields = if record.starts_with("1 ") { 9 } else { 10 };
            let path = record.splitn(fields, ' ').nth(fields - 1).unwrap_or_default().to_string();
            let xy = xy(record);
            if !xy.starts_with('.') {
                add(Group::Staged, &path);
            }
            if !xy.ends_with('.') {
                add(Group::Changed, &path);
            }
            if record.starts_with("2 ") {
                // the original path of a rename
                records.next();
            }
        }
    }
    changes.sort_by_key(|c| Group::ALL.iter().position(|g| *g == c.group));
    let commits = if oid == "(initial)" {
        Vec::new()
    } else {
        let format = format!("--format=%h%x1f%s%x1f%an%x1f%ad");
        let log = run(program, root, &["log", "-n", &LOG_LIMIT.to_string(), &format, "--date=short", "-z"])?;
        String::from_utf8_lossy(&log)
            .split('\0')
            .filter(|r| !r.is_empty())
            .map(|r| {
                let mut f = r.split('\x1f').map(str::to_string);
                let mut next = || f.next().unwrap_or_default();
                Commit { hash: next(), subject: next(), author: next(), date: next() }
            })
            .collect()
    };
    let head = if branch == "(detached)" {
        Head::Detached(commits.first().map(|c| c.hash.clone()).unwrap_or(oid))
    } else {
        Head::Branch { name: branch, upstream: ab }
    };
    Ok(Repo { head, changes, commits })
}

/// What the tab tells the window.
#[derive(Clone, Debug, PartialEq)]
pub enum GitEvent {
    /// The user opened this changed file (GITRS-B05).
    Open(PathBuf),
}

/// The git tab of a folder.
pub struct GitPanel {
    focus: FocusHandle,
    root: PathBuf,
    program: PathBuf,
    /// What was read, once it was.
    read: Option<Result<Repo, GitError>>,
    _reading: Option<Task<()>>,
}

impl GitPanel {
    /// The tab of `root`, read with the user's git.
    pub fn new(root: PathBuf, window: &mut Window, cx: &mut Context<Self>) -> Self {
        Self::with_program(root, PathBuf::from(GIT), window, cx)
    }

    /// The tab of `root`, read with `program`.
    pub fn with_program(root: PathBuf, program: PathBuf, _: &mut Window, cx: &mut Context<Self>) -> Self {
        let mut this = GitPanel { focus: cx.focus_handle(), root, program, read: None, _reading: None };
        this.refresh(cx);
        this
    }

    /// The folder of the tab.
    pub fn root(&self) -> &Path {
        &self.root
    }

    /// Reads the repository again, away from the window's thread (GITRS-B04).
    pub fn refresh(&mut self, cx: &mut Context<Self>) {
        let (program, root) = (self.program.clone(), self.root.clone());
        let reading = cx.background_executor().spawn(async move { read_repo(&program, &root) });
        self._reading = Some(cx.spawn(async move |this, cx| {
            let read = reading.await;
            let _ = this.update(cx, |this, cx| {
                if let Err(GitError::Failed(e)) = &read {
                    // GITRS-E02
                    eprintln!("polvo: git: {e}");
                }
                this.read = Some(read);
                cx.notify();
            });
        }));
    }

    /// GITRS-B05: a changed file that exists opens.
    fn open(&mut self, path: &str, cx: &mut Context<Self>) {
        let path = self.root.join(path);
        if path.is_file() {
            cx.emit(GitEvent::Open(path));
        }
    }

    fn head_line(head: &Head) -> String {
        match head {
            Head::Branch { name, upstream: Some((ahead, behind)) } => format!("{name}  ↑{ahead} ↓{behind}"),
            Head::Branch { name, upstream: None } => name.clone(),
            Head::Detached(hash) => format!("HEAD destacada em {hash}"),
        }
    }
}

impl EventEmitter<GitEvent> for GitPanel {}
impl EventEmitter<PanelEvent> for GitPanel {}

impl Focusable for GitPanel {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus.clone()
    }
}

impl BasePanel for GitPanel {
    fn panel_name(&self) -> &'static str {
        "git"
    }

    /// The folder goes with the organisation of the dock, for the tab to come back with it.
    fn dump(&self, _: &App) -> PanelState {
        let mut state = PanelState::new(self.panel_name());
        state.info = PanelInfo::panel(serde_json::json!({ "root": self.root.to_string_lossy() }));
        state
    }
}

impl Panel for GitPanel {
    fn title(&mut self, _: &mut Window, _: &mut Context<Self>) -> impl IntoElement {
        let name = self.root.file_name().map_or_else(|| self.root.to_string_lossy().into_owned(), |n| n.to_string_lossy().into_owned());
        format!("Git · {name}")
    }
}

impl Render for GitPanel {
    fn render(&mut self, _: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let theme = cx.theme();
        let (bg, fg, muted, danger, hover, border) =
            (theme.background, theme.foreground, theme.muted_foreground, theme.danger, theme.list_hover, theme.border);
        let heading = |text: &'static str| div().pt(px(12.)).pb(px(4.)).font_weight(FontWeight::BOLD).child(text);
        let status = |text: String| {
            div().id("git-message").role(Role::Status).aria_label(text.clone()).test_support().py(px(8.)).child(text)
        };
        // GITRS-B04
        let refresh = div()
            .id("git-refresh")
            .role(Role::Button)
            .aria_label("Atualizar")
            .test_support()
            .p(px(4.))
            .rounded(px(6.))
            .cursor_pointer()
            .hover(move |s| s.bg(hover))
            .on_click(cx.listener(|this, _, _, cx| this.refresh(cx)))
            .child(Icon::new(gpui_kit::assets::IconName::RefreshCw).small());
        let mut body = div().id("git-body").flex_1().min_h_0().overflow_y_scroll().px(px(12.));
        match &self.read {
            None => body = body.child(div().text_color(muted).child("A ler o repositório…")),
            // GITRS-E01
            Some(Err(GitError::NotARepo)) => body = body.child(status(format!("{} não é de um repositório git.", self.root.display()))),
            // GITRS-E02
            Some(Err(GitError::Failed(e))) => body = body.child(status(e.clone()).text_color(danger)),
            Some(Ok(repo)) => {
                // GITRS-B01
                body = body.child(heading("Ramo")).child(div().id("git-head").test_support().child(Self::head_line(&repo.head)));
                // GITRS-B02
                body = body.child(heading("Alterações"));
                if repo.changes.is_empty() {
                    body = body.child(div().text_color(muted).child("Sem alterações."));
                }
                for group in Group::ALL {
                    let files: Vec<&Change> = repo.changes.iter().filter(|c| c.group == group).collect();
                    if files.is_empty() {
                        continue;
                    }
                    body = body.child(div().pt(px(4.)).text_color(muted).child(format!("{} ({})", group.label(), files.len())));
                    for change in files {
                        let path = change.path.clone();
                        body = body.child(
                            div()
                                .id(SharedString::from(format!("change-{}-{}", group.id(), change.path)))
                                .role(Role::ListItem)
                                .aria_label(change.path.clone())
                                .test_support()
                                .pl(px(12.))
                                .rounded(px(4.))
                                .cursor_pointer()
                                .hover(move |s| s.bg(hover))
                                .on_click(cx.listener(move |this, _, _, cx| this.open(&path, cx)))
                                .child(change.path.clone()),
                        );
                    }
                }
                // GITRS-B03
                body = body.child(heading("Últimos commits"));
                for c in &repo.commits {
                    body = body.child(
                        div()
                            .flex()
                            .gap(px(8.))
                            .py(px(2.))
                            .child(div().font_family("monospace").text_color(muted).child(c.hash.clone()))
                            .child(div().flex_1().overflow_hidden().whitespace_nowrap().child(c.subject.clone()))
                            .child(div().text_color(muted).child(format!("{} · {}", c.author, c.date))),
                    );
                }
            }
        }
        div()
            .id("git")
            .test_support()
            .size_full()
            .flex()
            .flex_col()
            .bg(bg)
            .text_color(fg)
            .track_focus(&self.focus)
            .child(
                div()
                    .flex()
                    .items_center()
                    .justify_between()
                    .px(px(12.))
                    .py(px(6.))
                    .border_b_1()
                    .border_color(border)
                    .child(div().text_color(muted).child(self.root.display().to_string()))
                    .child(refresh),
            )
            .child(body)
    }
}

#[cfg(test)]
#[path = "git_test.rs"]
mod tests;
