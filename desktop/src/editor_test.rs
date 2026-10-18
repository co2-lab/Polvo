// @anchors
//   code: EDTST
//   ref: EDTRS

use super::*;
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::path::PathBuf;

struct Opened {
    handle: AnyWindowHandle,
    editor: Entity<EditorPanel>,
}

fn open_with(cx: &mut TestAppContext, path: Option<PathBuf>) -> Opened {
    cx.update(gpui_kit::init);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let (handle, editor) = cx
        .update(|cx| {
            gpui_kit::open_window(options, cx, |window, cx| {
                cx.new(|cx| match path {
                    Some(path) => EditorPanel::open(path, window, cx),
                    None => EditorPanel::empty(window, cx),
                })
            })
        })
        .unwrap();
    frame(cx, handle);
    Opened { handle, editor }
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

/// Types `text` in the editor, with the focus.
fn type_in(cx: &mut TestAppContext, o: &Opened, text: &str) {
    let editor = o.editor.clone();
    cx.update_window(o.handle, |_, window, cx| {
        let focus = editor.read(cx).focus_handle(cx);
        window.focus(&focus, cx);
        window.render_frame(cx);
        window.input(text, cx);
    })
    .unwrap();
    frame(cx, o.handle);
}

fn press(cx: &mut TestAppContext, o: &Opened, key: &str) {
    cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    cx.run_until_parked();
    frame(cx, o.handle);
}

fn title(cx: &mut TestAppContext, o: &Opened) -> String {
    o.editor.read_with(cx, |e, _| e.title_text())
}

fn text(cx: &mut TestAppContext, o: &Opened) -> Option<String> {
    o.editor.read_with(cx, |e, cx| e.text(cx))
}

/// A file with `text` in a new folder.
fn file(name: &str, text: &str) -> (tempfile::TempDir, PathBuf) {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join(name);
    std::fs::write(&path, text).unwrap();
    (dir, path)
}

scenario!(edtrs_b01_o_editor_vazio, "EDTRS-B01: O editor vazio", |cx| {
    let o = open_with(cx, None);
    let path = o.editor.read_with(cx, |e, _| e.path().map(PathBuf::from));
    assert_eq!((text(cx, &o), path, title(cx, &o)), (Some(String::new()), None, "Sem título".to_string()));
});

scenario!(edtrs_b02_o_editor_com_um_ficheiro, "EDTRS-B02: O editor com um ficheiro", |cx| {
    let (_dir, path) = file("main.rs", "fn main() {}");
    let o = open_with(cx, Some(path));
    let language = o.editor.read_with(cx, |e, cx| e.language(cx));
    assert_eq!((text(cx, &o), language, title(cx, &o)), (Some("fn main() {}".to_string()), Some("rust".to_string()), "main.rs".to_string()));
});

scenario!(edtrs_b03_a_marca_de_alterado, "EDTRS-B03: A marca de alterado", |cx| {
    let (_dir, path) = file("notas.txt", "");
    let o = open_with(cx, Some(path));
    type_in(cx, &o, "x");
    assert_eq!(title(cx, &o), "● notas.txt");
});

scenario!(edtrs_b04_01_gravar_o_ficheiro, "EDTRS-B04#01: Gravar o ficheiro", |cx| {
    let (_dir, path) = file("notas.txt", "");
    let o = open_with(cx, Some(path.clone()));
    type_in(cx, &o, "ola");
    press(cx, &o, "ctrl-s");
    assert_eq!((std::fs::read_to_string(&path).unwrap(), title(cx, &o)), ("ola".to_string(), "notas.txt".to_string()));
});

scenario!(edtrs_b04_02_gravar_o_editor_vazio_pergunta_onde, "EDTRS-B04#02: Gravar o editor vazio pergunta onde", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let target = dir.path().join("novo.py");
    let o = open_with(cx, None);
    type_in(cx, &o, "ola");
    press(cx, &o, "ctrl-s");
    let chosen = target.clone();
    cx.simulate_new_path_selection(move |_| Some(chosen));
    cx.run_until_parked();
    let language = o.editor.read_with(cx, |e, cx| e.language(cx));
    assert_eq!(
        (std::fs::read_to_string(&target).unwrap(), title(cx, &o), language),
        ("ola".to_string(), "novo.py".to_string(), Some("python".to_string()))
    );
});

scenario!(edtrs_b04_03_cancelar_o_dialogo_nao_grava, "EDTRS-B04#03: Cancelar o diálogo não grava", |cx| {
    let o = open_with(cx, None);
    type_in(cx, &o, "ola");
    press(cx, &o, "ctrl-s");
    cx.simulate_new_path_selection(|_| None);
    cx.run_until_parked();
    let path = o.editor.read_with(cx, |e, _| e.path().map(PathBuf::from));
    assert_eq!((path, title(cx, &o)), (None, "● Sem título".to_string()));
});

scenario!(edtrs_e01_um_ficheiro_que_nao_e_texto, "EDTRS-E01: Um ficheiro que não é texto", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("imagem.bin");
    std::fs::write(&path, [0xff, 0xfe, 0x00, 0x80]).unwrap();
    let o = open_with(cx, Some(path));
    let reason = cx.update_window(o.handle, |_, window, _| window.find("editor-error").label().map(str::to_string)).unwrap();
    assert!(reason.as_deref().is_some_and(|r| r.contains("UTF-8")), "{reason:?}");
    assert_eq!(text(cx, &o), None);
});

scenario!(edtrs_e02_um_ficheiro_que_nao_se_grava, "EDTRS-E02: Um ficheiro que não se grava", |cx| {
    let (dir, path) = file("notas.txt", "");
    let o = open_with(cx, Some(path));
    drop(dir);
    type_in(cx, &o, "x");
    press(cx, &o, "ctrl-s");
    let message = cx.update_window(o.handle, |_, window, _| window.try_find("editor-message").and_then(|m| m.label().map(str::to_string))).unwrap();
    assert!(message.is_some_and(|m| m.contains("não foi gravado")));
    assert_eq!(title(cx, &o), "● notas.txt");
});

scenario!(edtrs_b05_o_ficheiro_de_outra_maquina, "EDTRS-B05: O ficheiro de outra máquina", |cx| {
    let (_dir, path) = file("notas.md", "olá");
    let home = tempfile::tempdir().unwrap();
    let st = crate::remote::Station::open(home.path(), "", "/bin/sh").unwrap();
    let source = FileSource::Node { station: st, machine: crate::remote::LOCAL.into() };
    cx.update(gpui_kit::init);
    let (handle, editor) = cx
        .update(|cx| {
            let p = path.clone();
            gpui_kit::open_window(WindowOptions::default(), cx, |window, cx| cx.new(|cx| EditorPanel::open_on(source, p, window, cx)))
        })
        .unwrap();
    let o = Opened { handle, editor };
    frame(cx, o.handle);
    let waiting = cx.update_window(o.handle, |_, window, _| window.find("editor-error").label().map(str::to_string)).unwrap();
    cx.run_until_parked();
    let shown = text(cx, &o);
    type_in(cx, &o, "x");
    press(cx, &o, "ctrl-s");
    assert_eq!(
        (waiting, shown, std::fs::read_to_string(&path).unwrap(), title(cx, &o)),
        (Some("a abrir…".to_string()), Some("olá".to_string()), "xolá".to_string(), "local: notas.md".to_string())
    );
});

/// Asks the editor to close: whether it closed at once, and the closes it announced after `answer`.
fn close_with(cx: &mut TestAppContext, o: &Opened, answer: Option<&str>) -> (bool, Vec<EditorEvent>) {
    let closed: std::rc::Rc<std::cell::RefCell<Vec<EditorEvent>>> = Default::default();
    {
        let closed = closed.clone();
        cx.update(|cx| cx.subscribe(&o.editor, move |_, ev: &EditorEvent, _| closed.borrow_mut().push(ev.clone())).detach());
    }
    let editor = o.editor.clone();
    let at_once = cx.update_window(o.handle, |_, window, cx| editor.update(cx, |e, cx| e.ask_to_close(window, cx))).unwrap();
    if let Some(answer) = answer {
        assert!(cx.has_pending_prompt(), "want the question");
        cx.simulate_prompt_answer(answer);
        cx.run_until_parked();
    }
    let got = closed.borrow().clone();
    (at_once, got)
}

scenario!(edtrs_b06_01_descartar_ao_fechar, "EDTRS-B06#01: Descartar as alterações ao fechar", |cx| {
    let (_dir, path) = file("notas.md", "olá");
    let o = open_with(cx, Some(path.clone()));
    type_in(cx, &o, "x");
    let got = close_with(cx, &o, Some("Descartar"));
    assert_eq!((got, std::fs::read_to_string(&path).unwrap()), ((false, vec![EditorEvent::Close]), "olá".to_string()));
});

scenario!(edtrs_b06_02_gravar_ao_fechar, "EDTRS-B06#02: Gravar as alterações ao fechar", |cx| {
    let (_dir, path) = file("notas.md", "olá");
    let o = open_with(cx, Some(path.clone()));
    type_in(cx, &o, "x");
    let got = close_with(cx, &o, Some("Gravar"));
    assert_eq!((got, std::fs::read_to_string(&path).unwrap()), ((false, vec![EditorEvent::Close]), "xolá".to_string()));
});

scenario!(edtrs_b06_03_cancelar_e_fechar_sem_alteracoes, "EDTRS-B06#03: Cancelar o fecho, e fechar sem alterações", |cx| {
    let (_dir, path) = file("notas.md", "olá");
    let o = open_with(cx, Some(path.clone()));
    type_in(cx, &o, "x");
    let cancelled = close_with(cx, &o, Some("Cancelar"));
    let marked = title(cx, &o);
    let (_d2, other) = file("outro.md", "nada");
    let clean = open_with(cx, Some(other));
    let at_once = close_with(cx, &clean, None);
    assert_eq!((cancelled, marked, at_once), ((false, vec![]), "● notas.md".to_string(), (true, vec![])));
});
