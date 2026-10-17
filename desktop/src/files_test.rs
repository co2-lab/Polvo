// @anchors
//   code: FLTST
//   ref: FLTRE

use super::*;
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, SharedString, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::cell::RefCell;
use std::path::{Path, PathBuf};
use std::rc::Rc;

struct Opened {
    handle: AnyWindowHandle,
    tree: Entity<FileTree>,
    opened: Rc<RefCell<Vec<PathBuf>>>,
}

/// A folder with these files and folders (a name ending in "/" is a folder).
fn folder(entries: &[&str]) -> tempfile::TempDir {
    let dir = tempfile::tempdir().unwrap();
    for e in entries {
        let path = dir.path().join(e.trim_end_matches('/'));
        if e.ends_with('/') {
            std::fs::create_dir_all(&path).unwrap();
        } else {
            std::fs::create_dir_all(path.parent().unwrap()).unwrap();
            std::fs::write(&path, "").unwrap();
        }
    }
    dir
}

fn open(cx: &mut TestAppContext, root: &Path) -> Opened {
    cx.update(gpui_kit::init);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let root = root.to_path_buf();
    let (handle, tree) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| FileTree::new(root, window, cx))))
        .unwrap();
    let opened: Rc<RefCell<Vec<PathBuf>>> = Rc::default();
    {
        let opened = opened.clone();
        cx.update(|cx| {
            cx.subscribe(&tree, move |_, ev: &FileTreeEvent, _| match ev {
                FileTreeEvent::Open(path) => opened.borrow_mut().push(path.clone()),
            })
            .detach()
        });
    }
    let o = Opened { handle, tree, opened };
    let tree = o.tree.clone();
    cx.update_window(o.handle, |_, window, cx| {
        let focus = tree.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
    })
    .unwrap();
    frame(cx, &o);
    o
}

fn frame(cx: &mut TestAppContext, o: &Opened) {
    cx.update_window(o.handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

fn press(cx: &mut TestAppContext, o: &Opened, key: &str) {
    cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    frame(cx, o);
}

fn click(cx: &mut TestAppContext, o: &Opened, entry: &str) {
    let id = SharedString::from(format!("entry-{entry}"));
    cx.update_window(o.handle, |_, window, cx| window.click(id, cx)).unwrap();
    frame(cx, o);
}

fn listing(cx: &mut TestAppContext, o: &Opened) -> Vec<String> {
    o.tree.read_with(cx, |t, _| t.listing())
}

fn chosen(cx: &mut TestAppContext, o: &Opened) -> String {
    o.tree.read_with(cx, |t, _| t.chosen())
}

scenario!(fltre_b01_o_conteudo_da_pasta, "FLTRE-B01: O conteúdo da pasta", |cx| {
    let dir = folder(&["b.txt", "A.md", "zeta/", "Alfa/"]);
    let o = open(cx, dir.path());
    let selected = cx.update_window(o.handle, |_, window, _| window.find("entry-Alfa").selected()).unwrap();
    assert_eq!((listing(cx, &o), selected), (vec!["Alfa/".to_string(), "zeta/".into(), "A.md".into(), "b.txt".into()], Some(true)));
});

scenario!(fltre_b02_01_carregar_numa_pasta, "FLTRE-B02#01: Carregar numa pasta expande-a e recolhe-a", |cx| {
    let dir = folder(&["src/main.rs"]);
    let o = open(cx, dir.path());
    click(cx, &o, "src");
    let expanded = listing(cx, &o);
    click(cx, &o, "src");
    assert_eq!((expanded, listing(cx, &o)), (vec!["src/".to_string(), "  main.rs".into()], vec!["src/".to_string()]));
});

scenario!(fltre_b02_02_as_setas_expandem_e_recolhem, "FLTRE-B02#02: As setas expandem e recolhem", |cx| {
    let dir = folder(&["src/main.rs"]);
    let o = open(cx, dir.path());
    press(cx, &o, "right");
    let expanded = listing(cx, &o);
    press(cx, &o, "left");
    assert_eq!((expanded, listing(cx, &o)), (vec!["src/".to_string(), "  main.rs".into()], vec!["src/".to_string()]));
});

scenario!(fltre_b02_03_expandir_le_a_pasta_outra_vez, "FLTRE-B02#03: Expandir lê a pasta outra vez", |cx| {
    let dir = folder(&["src/main.rs"]);
    let o = open(cx, dir.path());
    click(cx, &o, "src");
    click(cx, &o, "src");
    std::fs::write(dir.path().join("src/novo.rs"), "").unwrap();
    click(cx, &o, "src");
    assert_eq!(listing(cx, &o), vec!["src/".to_string(), "  main.rs".into(), "  novo.rs".into()]);
});

scenario!(fltre_b03_01_carregar_num_ficheiro_abre_o, "FLTRE-B03#01: Carregar num ficheiro abre-o", |cx| {
    let dir = folder(&["notas.txt"]);
    let o = open(cx, dir.path());
    click(cx, &o, "notas.txt");
    assert_eq!(o.opened.borrow().clone(), vec![dir.path().join("notas.txt")]);
});

scenario!(fltre_b03_02_o_enter_abre_o_ficheiro_escolhido, "FLTRE-B03#02: O Enter abre o ficheiro escolhido", |cx| {
    let dir = folder(&["src/", "notas.txt"]);
    let o = open(cx, dir.path());
    press(cx, &o, "down");
    press(cx, &o, "enter");
    assert_eq!(o.opened.borrow().clone(), vec![dir.path().join("notas.txt")]);
});

scenario!(fltre_b04_andar_com_o_teclado, "FLTRE-B04: Andar com o teclado", |cx| {
    let dir = folder(&["src/main.rs"]);
    let o = open(cx, dir.path());
    press(cx, &o, "right");
    for _ in 0..3 {
        press(cx, &o, "down");
    }
    let after_down = chosen(cx, &o);
    press(cx, &o, "left");
    assert_eq!((after_down, chosen(cx, &o)), ("main.rs".to_string(), "src".to_string()));
});

scenario!(fltre_e01_uma_pasta_que_nao_se_le, "FLTRE-E01: Uma pasta que não se lê", |cx| {
    let dir = folder(&["fechada/", "notas.txt"]);
    let o = open(cx, dir.path());
    std::fs::remove_dir(dir.path().join("fechada")).unwrap();
    click(cx, &o, "fechada");
    let rows = listing(cx, &o);
    assert_eq!(rows.len(), 3, "{rows:?}");
    assert!(rows[1].starts_with("  ! "), "{rows:?}");
    click(cx, &o, "notas.txt");
    assert_eq!(o.opened.borrow().clone(), vec![dir.path().join("notas.txt")]);
});
