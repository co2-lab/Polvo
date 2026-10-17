// @anchors
//   code: GTTST
//   ref: GITRS

use super::*;
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, SharedString, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::cell::RefCell;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::rc::Rc;

/// Runs git in `dir` as a test user, and fails the test when git does.
fn git(dir: &Path, args: &[&str]) -> String {
    let out = Command::new("git")
        .args(["-c", "user.name=Teste", "-c", "user.email=teste@exemplo", "-c", "init.defaultBranch=main", "-c", "commit.gpgsign=false"])
        .args(args)
        .current_dir(dir)
        .env("LC_ALL", "C")
        .output()
        .unwrap();
    assert!(out.status.success(), "git {args:?}: {}", String::from_utf8_lossy(&out.stderr));
    String::from_utf8_lossy(&out.stdout).trim().to_string()
}

fn write(dir: &Path, name: &str, text: &str) {
    std::fs::write(dir.join(name), text).unwrap();
}

/// A repository with one commit of "a.txt" and "b.txt".
fn repo() -> tempfile::TempDir {
    let dir = tempfile::tempdir().unwrap();
    git(dir.path(), &["init", "-q"]);
    write(dir.path(), "a.txt", "a\n");
    write(dir.path(), "b.txt", "b\n");
    git(dir.path(), &["add", "."]);
    git(dir.path(), &["commit", "-q", "-m", "um"]);
    dir
}

fn read(dir: &Path) -> Repo {
    read_repo(Path::new(GIT), dir).unwrap()
}

fn paths(repo: &Repo, group: Group) -> Vec<String> {
    repo.changes.iter().filter(|c| c.group == group).map(|c| c.path.clone()).collect()
}

scenario!(gitrs_b01_01_o_ramo, "GITRS-B01#01: O ramo", {
    let dir = repo();
    git(dir.path(), &["checkout", "-q", "-b", "trabalho"]);
    assert_eq!(read(dir.path()).head, Head::Branch { name: "trabalho".into(), upstream: None });
});

scenario!(gitrs_b01_02_a_frente_e_atras, "GITRS-B01#02: À frente e atrás do ramo a montante", {
    let origin = repo();
    let parent = tempfile::tempdir().unwrap();
    git(parent.path(), &["clone", "-q", origin.path().to_str().unwrap(), "clone"]);
    let clone = parent.path().join("clone");
    write(origin.path(), "o.txt", "o\n");
    git(origin.path(), &["add", "."]);
    git(origin.path(), &["commit", "-q", "-m", "da origem"]);
    git(&clone, &["fetch", "-q"]);
    write(&clone, "l.txt", "l\n");
    git(&clone, &["add", "."]);
    git(&clone, &["commit", "-q", "-m", "local"]);
    assert_eq!(read(&clone).head, Head::Branch { name: "main".into(), upstream: Some((1, 1)) });
});

scenario!(gitrs_b01_03_a_head_destacada, "GITRS-B01#03: A HEAD destacada", {
    let dir = repo();
    let first = git(dir.path(), &["rev-parse", "--short", "HEAD"]);
    write(dir.path(), "a.txt", "a2\n");
    git(dir.path(), &["commit", "-q", "-am", "dois"]);
    git(dir.path(), &["checkout", "-q", "--detach", &first]);
    assert_eq!(read(dir.path()).head, Head::Detached(first));
});

scenario!(gitrs_b02_01_os_ficheiros_com_alteracoes, "GITRS-B02#01: Os ficheiros com alterações", {
    let dir = repo();
    write(dir.path(), "a.txt", "a2\n");
    git(dir.path(), &["add", "a.txt"]);
    write(dir.path(), "a.txt", "a3\n");
    write(dir.path(), "b.txt", "b2\n");
    write(dir.path(), "c.txt", "c\n");
    let r = read(dir.path());
    assert_eq!(
        (paths(&r, Group::Staged), paths(&r, Group::Changed), paths(&r, Group::Untracked)),
        (vec!["a.txt".to_string()], vec!["a.txt".to_string(), "b.txt".into()], vec!["c.txt".to_string()])
    );
});

scenario!(gitrs_b02_02_sem_alteracoes, "GITRS-B02#02: Sem alterações", {
    let dir = repo();
    assert!(read(dir.path()).changes.is_empty());
});

scenario!(gitrs_b02_03_um_conflito, "GITRS-B02#03: Um conflito", {
    let dir = repo();
    git(dir.path(), &["checkout", "-q", "-b", "outro"]);
    write(dir.path(), "a.txt", "outro\n");
    git(dir.path(), &["commit", "-q", "-am", "outro"]);
    git(dir.path(), &["checkout", "-q", "main"]);
    write(dir.path(), "a.txt", "main\n");
    git(dir.path(), &["commit", "-q", "-am", "main"]);
    let merge = Command::new("git").args(["-c", "user.name=T", "-c", "user.email=t@e", "merge", "-q", "outro"]).current_dir(dir.path()).output().unwrap();
    assert!(!merge.status.success(), "the merge conflicts");
    assert_eq!(paths(&read(dir.path()), Group::Conflict), vec!["a.txt".to_string()]);
});

scenario!(gitrs_b03_os_ultimos_commits, "GITRS-B03: Os últimos commits", {
    let dir = repo();
    write(dir.path(), "a.txt", "a2\n");
    git(dir.path(), &["commit", "-q", "-am", "dois"]);
    let hash = git(dir.path(), &["rev-parse", "--short", "HEAD"]);
    let commits = read(dir.path()).commits;
    let got: Vec<(&str, &str)> = commits.iter().map(|c| (c.subject.as_str(), c.author.as_str())).collect();
    assert_eq!(got, vec![("dois", "Teste"), ("um", "Teste")]);
    assert_eq!(commits[0].hash, hash);
});

struct Opened {
    handle: AnyWindowHandle,
    panel: Entity<GitPanel>,
    opened: Rc<RefCell<Vec<PathBuf>>>,
}

fn open(cx: &mut TestAppContext, root: &Path, program: &str) -> Opened {
    cx.update(gpui_kit::init);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let (root, program) = (root.to_path_buf(), PathBuf::from(program));
    let (handle, panel) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| GitPanel::with_program(root, program, window, cx))))
        .unwrap();
    let opened: Rc<RefCell<Vec<PathBuf>>> = Rc::default();
    {
        let opened = opened.clone();
        cx.update(|cx| {
            cx.subscribe(&panel, move |_, ev: &GitEvent, _| match ev {
                GitEvent::Open(path) => opened.borrow_mut().push(path.clone()),
            })
            .detach()
        });
    }
    let o = Opened { handle, panel, opened };
    settle(cx, &o);
    o
}

fn settle(cx: &mut TestAppContext, o: &Opened) {
    cx.run_until_parked();
    cx.update_window(o.handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

fn click(cx: &mut TestAppContext, o: &Opened, id: &str) {
    let id = SharedString::from(id.to_string());
    cx.update_window(o.handle, |_, window, cx| window.click(id, cx)).unwrap();
    settle(cx, o);
}

fn message(cx: &mut TestAppContext, o: &Opened) -> String {
    cx.update_window(o.handle, |_, window, _| window.try_find("git-message").and_then(|m| m.label().map(str::to_string)))
        .unwrap()
        .unwrap_or_default()
}

scenario!(gitrs_b04_atualizar, "GITRS-B04: Atualizar", |cx| {
    let dir = repo();
    let o = open(cx, dir.path(), GIT);
    let before = cx.update_window(o.handle, |_, window, _| window.try_find("change-untracked-novo.txt").is_some()).unwrap();
    write(dir.path(), "novo.txt", "n\n");
    click(cx, &o, "git-refresh");
    let after = cx.update_window(o.handle, |_, window, _| window.try_find("change-untracked-novo.txt").is_some()).unwrap();
    assert_eq!((before, after), (false, true));
});

scenario!(gitrs_b05_abrir_um_ficheiro_alterado, "GITRS-B05: Abrir um ficheiro alterado", |cx| {
    let dir = repo();
    write(dir.path(), "b.txt", "b2\n");
    let o = open(cx, dir.path(), GIT);
    click(cx, &o, "change-changed-b.txt");
    assert_eq!(o.opened.borrow().clone(), vec![dir.path().join("b.txt")]);
});

scenario!(gitrs_e01_uma_pasta_fora_de_um_repositorio, "GITRS-E01: Uma pasta fora de um repositório", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let o = open(cx, dir.path(), GIT);
    let m = message(cx, &o);
    assert!(m.contains("não é de um repositório git"), "{m}");
    let _ = o.panel.read_with(cx, |p, _| p.root().to_path_buf());
});

scenario!(gitrs_e02_um_git_que_falha, "GITRS-E02: Um git que falha", |cx| {
    let dir = repo();
    let o = open(cx, dir.path(), "/nao/existe/git");
    let m = message(cx, &o);
    assert!(m.contains("/nao/existe/git"), "{m}");
});
