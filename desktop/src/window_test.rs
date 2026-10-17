// @anchors
//   code: WNTSW
//   ref: WNDWI

use super::*;
use crate::layout::{DockAreaState, LayoutFile}; // @dep: DLIDC
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::screen::tests::Sink; // @dep: SCIMS
use crate::keymap::{Action, Keymap}; // @dep: KYIMA
use crate::screen::{Event, Screen, Size}; // @dep: SCIMS
use futures::channel::mpsc::{self, UnboundedSender};
use gpui_kit::test::{TestAppContextExt as _, TestWindowExt as _};
use gpui_kit::{AnyWindowHandle, Bounds, Entity, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::cell::RefCell;
use std::rc::Rc;
use std::time::Duration;

fn prefs(left: &[Button], right: &[Button]) -> Prefs {
    Prefs { left: left.to_vec(), right: right.to_vec(), dark: true, font: "DejaVu Sans Mono".into(), font_px: 15. }
}

fn gnome_left() -> Prefs {
    prefs(&[Button::Close, Button::Minimize, Button::Maximize], &[])
}

/// The terminals the window made: what each writes to its shell, and how to end each shell.
#[derive(Default)]
struct Made {
    sinks: Vec<Sink>,
    ends: Vec<UnboundedSender<Event>>,
}

/// Terminals with no shell: the first `ok` of them start, the others fail.
fn offline_terminals(made: Rc<RefCell<Made>>, ok: usize) -> NewTerminal {
    Rc::new(move |prefs: &Prefs, window: &mut gpui_kit::Window, cx: &mut gpui_kit::App| {
        let mut m = made.borrow_mut();
        if m.sinks.len() >= ok {
            return Err(std::io::Error::other("/nao/existe/shell: no such file"));
        }
        let sink = Sink::default();
        let screen = Screen::new(Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 }, Box::new(sink.clone())).unwrap();
        let (tx, rx) = mpsc::unbounded();
        m.sinks.push(sink);
        m.ends.push(tx);
        Ok(cx.new(|cx| TerminalPanel::new(screen, Some(rx), "/bin/sh", prefs, window, cx)))
    })
}

struct Opened {
    handle: AnyWindowHandle,
    view: Entity<PolvoWindow>,
    made: Rc<RefCell<Made>>,
}

fn open_with(cx: &mut TestAppContext, prefs: Prefs, ok: usize) -> Opened {
    open_full(cx, prefs, ok, None)
}

fn open_full(cx: &mut TestAppContext, prefs: Prefs, ok: usize, layout: Option<LayoutFile>) -> Opened {
    open_in(cx, prefs, ok, layout, std::env::temp_dir())
}

fn open_in(cx: &mut TestAppContext, prefs: Prefs, ok: usize, layout: Option<LayoutFile>, root: std::path::PathBuf) -> Opened {
    cx.update(gpui_kit::init);
    let made = Rc::new(RefCell::new(Made::default()));
    let new_terminal = offline_terminals(made.clone(), ok);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let (handle, view) = cx
        .update(|cx| {
            gpui_kit::open_window(options, cx, |window, cx| {
                cx.new(|cx| PolvoWindow::new(prefs, new_terminal, layout, root, window, cx))
            })
        })
        .unwrap();
    cx.run_until_parked();
    if !cx.update(|cx| cx.windows().is_empty()) {
        frame(cx, handle);
        frame(cx, handle);
    }
    Opened { handle, view, made }
}

fn open_gnome(cx: &mut TestAppContext) -> Opened {
    open_with(cx, gnome_left(), usize::MAX)
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

/// Asks for a new tab with the "+" and chooses the terminal (WNDWI-B10).
fn new_terminal(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.click("new-tab", cx)).unwrap();
    frame(cx, handle);
    cx.update_window(handle, |_, window, cx| window.press("enter", cx)).unwrap();
}

fn x_of(cx: &mut TestAppContext, handle: AnyWindowHandle, id: &'static str) -> f32 {
    cx.update_window(handle, |_, window, _| f32::from(window.find(id).bounds().origin.x)).unwrap()
}

fn terminals(cx: &mut TestAppContext, o: &Opened) -> usize {
    o.view.read_with(cx, |v, _| v.terminals.len())
}

scenario!(wndwi_b01_01_os_botoes_a_esquerda, "WNDWI-B01#01: Os botões à esquerda pela ordem do sistema", |cx| {
    let o = open_gnome(cx);
    let xs: Vec<f32> = ["close", "minimize", "maximize", "title"].into_iter().map(|id| x_of(cx, o.handle, id)).collect();
    assert!(xs.windows(2).all(|w| w[0] < w[1]), "{xs:?}");
});

scenario!(wndwi_b01_02_os_botoes_a_direita, "WNDWI-B01#02: Os botões à direita pela ordem do sistema", |cx| {
    let o = open_with(cx, prefs(&[], &[Button::Minimize, Button::Maximize, Button::Close]), usize::MAX);
    let xs: Vec<f32> = ["title", "minimize", "maximize", "close"].into_iter().map(|id| x_of(cx, o.handle, id)).collect();
    assert!(xs.windows(2).all(|w| w[0] < w[1]), "{xs:?}");
    let title = cx.update_window(o.handle, |_, window, _| window.find("title").bounds()).unwrap();
    let middle = f32::from(title.origin.x + title.size.width / 2.);
    assert!((middle - 400.).abs() < 2., "the title is centred: {middle}");
});

scenario!(wndwi_b02_fechar_fecha_a_janela, "WNDWI-B02: Fechar fecha a janela", |cx| {
    let o = open_gnome(cx);
    cx.update_window(o.handle, |_, window, cx| window.click("close", cx)).unwrap();
    cx.run_until_parked();
    assert!(cx.update(|cx| cx.windows().is_empty()));
});

scenario!(wndwi_b07_o_tema_da_barra_segue_as_preferencias, "WNDWI-B07: O tema da barra segue as preferências", |cx| {
    let o = open_with(cx, Prefs { dark: false, ..gnome_left() }, usize::MAX);
    let bar = o.view.read_with(cx, |v, _| v.colors().bar);
    assert!(bar.l > 0.5, "a light title bar: {bar:?}");
});

scenario!(wndwi_b09_arrastar_a_aba_para_a_borda_divide_a_doca, "WNDWI-B09: Arrastar a aba para a borda divide a doca", async |cx| {
    let o = open_gnome(cx);
    new_terminal(cx, o.handle);
    frame(cx, o.handle);
    cx.update_window(o.handle, |_, window, cx| {
        let visible = window.find("terminal").bounds();
        let from = window.within("tab-bar").find(1usize).bounds().center();
        let to = point(visible.right() - px(10.), visible.center().y);
        window.drag(from, to, cx);
    })
    .unwrap();
    let panels = o.view.read_with(cx, |v, _| v.terminals.clone());
    let side_by_side = move |cx: &gpui_kit::App| {
        let a = panels[0].read(cx).laid_out();
        let b = panels[1].read(cx).laid_out();
        a.size.width > px(0.) && b.size.width > px(0.) && b.left() >= a.right()
    };
    let check = side_by_side.clone();
    cx.wait_for(o.handle, Duration::from_secs(1), move |_, cx| check(cx)).await;
    assert!(cx.update(|cx| side_by_side(cx)), "the two terminals are side by side, the second on the right");
});

scenario!(wndwi_b10_o_botao_de_terminal_novo, "WNDWI-B10: O botão de terminal novo", |cx| {
    let o = open_gnome(cx);
    new_terminal(cx, o.handle);
    frame(cx, o.handle);
    cx.update_window(o.handle, |_, window, cx| window.press("x", cx)).unwrap();
    assert_eq!(terminals(cx, &o), 2);
    let m = o.made.borrow();
    assert_eq!((m.sinks[0].take(), m.sinks[1].take()), (Vec::new(), b"x".to_vec()));
});

scenario!(wndwi_b11_o_fim_de_uma_shell_tira_a_aba, "WNDWI-B11: O fim de uma shell tira a aba, e a última fecha a janela", |cx| {
    let o = open_gnome(cx);
    new_terminal(cx, o.handle);
    frame(cx, o.handle);
    let ends = o.made.borrow().ends.clone();
    ends[0].unbounded_send(Event::Exit).unwrap();
    cx.run_until_parked();
    assert_eq!(terminals(cx, &o), 1);
    assert!(!cx.update(|cx| cx.windows().is_empty()));
    ends[1].unbounded_send(Event::Exit).unwrap();
    cx.run_until_parked();
    assert!(cx.update(|cx| cx.windows().is_empty()));
});

scenario!(wndwi_e02_um_terminal_novo_que_nao_arranca, "WNDWI-E02: Um terminal novo que não arranca deixa a doca como estava", |cx| {
    let o = open_with(cx, gnome_left(), 1);
    new_terminal(cx, o.handle);
    frame(cx, o.handle);
    assert_eq!(terminals(cx, &o), 1);
});

/// The terminals in a kept organisation.
fn terminals_in(state: &DockAreaState) -> usize {
    fn count(p: &gpui_kit::component::dock::PanelState) -> usize {
        usize::from(p.panel_name == "terminal") + p.children.iter().map(count).sum::<usize>()
    }
    count(&state.center)
}

/// Lets the organisation be saved: the time after the last change of the dock.
fn settle(cx: &mut TestAppContext) {
    cx.executor().advance_clock(SAVE_DELAY + Duration::from_millis(100));
    cx.run_until_parked();
}

scenario!(wndwi_b12_01_cada_mudanca_grava_a_organizacao, "WNDWI-B12#01: Cada mudança da doca grava a organização", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let file = LayoutFile::at(dir.path().join("layout.json"));
    let o = open_full(cx, gnome_left(), usize::MAX, Some(file.clone()));
    new_terminal(cx, o.handle);
    settle(cx);
    let kept = file.read().expect("the organisation was saved");
    assert_eq!(terminals_in(&kept), 2);
});

scenario!(wndwi_b12_02_ao_abrir_a_organizacao_volta, "WNDWI-B12#02: Ao abrir, a organização volta", async |cx| {
    let dir = tempfile::tempdir().unwrap();
    let file = LayoutFile::at(dir.path().join("layout.json"));
    let first = open_full(cx, gnome_left(), usize::MAX, Some(file.clone()));
    new_terminal(cx, first.handle);
    frame(cx, first.handle);
    cx.update_window(first.handle, |_, window, cx| {
        let visible = window.find("terminal").bounds();
        let from = window.within("tab-bar").find(1usize).bounds().center();
        window.drag(from, point(visible.right() - px(10.), visible.center().y), cx);
    })
    .unwrap();
    settle(cx);
    cx.update_window(first.handle, |_, window, _| window.remove_window()).unwrap();
    cx.run_until_parked();

    let again = open_full(cx, gnome_left(), usize::MAX, Some(file));
    assert_eq!(again.made.borrow().sinks.len(), 2, "each terminal starts a new shell");
    let panels = again.view.read_with(cx, |v, _| v.terminals.clone());
    assert_eq!(panels.len(), 2);
    let side_by_side = move |cx: &gpui_kit::App| {
        let (a, b) = (panels[0].read(cx).laid_out(), panels[1].read(cx).laid_out());
        a.size.width > px(0.) && b.size.width > px(0.) && (b.left() >= a.right() || a.left() >= b.right())
    };
    let check = side_by_side.clone();
    cx.wait_for(again.handle, Duration::from_secs(1), move |_, cx| check(cx)).await;
    assert!(cx.update(|cx| side_by_side(cx)), "the two terminals are side by side again");
});

scenario!(wndwi_e03_uma_organizacao_que_nao_se_grava, "WNDWI-E03: Uma organização que não se grava deixa a janela a funcionar", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let not_a_dir = dir.path().join("ficheiro");
    std::fs::write(&not_a_dir, "").unwrap();
    let o = open_full(cx, gnome_left(), usize::MAX, Some(LayoutFile::at(not_a_dir.join("layout.json"))));
    new_terminal(cx, o.handle);
    settle(cx);
    assert!(!cx.update(|cx| cx.windows().is_empty()));
    assert_eq!(terminals(cx, &o), 2);
});

scenario!(wndwi_e04_sem_nenhum_terminal_a_janela_fecha, "WNDWI-E04: Sem nenhum terminal que arranque, a janela fecha", |cx| {
    let _o = open_with(cx, gnome_left(), 0);
    assert!(cx.update(|cx| cx.windows().is_empty()));
});

scenario!(wndwi_b13_fechar_a_aba_termina_a_shell, "WNDWI-B13: Fechar a aba termina a shell", |cx| {
    let o = open_gnome(cx);
    new_terminal(cx, o.handle);
    frame(cx, o.handle);
    let (panels, dock) = o.view.read_with(cx, |v, _| (v.terminals.clone(), v.dock.clone()));
    let ends = o.made.borrow().ends.clone();
    cx.update_window(o.handle, |_, window, cx| dock.update(cx, |d, cx| d.remove_panel(panels[0].clone(), window, cx))).unwrap();
    drop(panels);
    cx.run_until_parked();
    assert!(ends[0].is_closed(), "the first shell lost its terminal");
    assert!(!ends[1].is_closed());
    let last = o.view.read_with(cx, |v, _| v.terminals[0].clone());
    cx.update_window(o.handle, |_, window, cx| dock.update(cx, |d, cx| d.remove_panel(last, window, cx))).unwrap();
    cx.run_until_parked();
    assert!(cx.update(|cx| cx.windows().is_empty()));
});

scenario!(wndwi_b14_as_abas_ficam_em_baixo, "WNDWI-B14: As abas ficam em baixo", |cx| {
    let o = open_gnome(cx);
    new_terminal(cx, o.handle);
    frame(cx, o.handle);
    frame(cx, o.handle);
    let (terminal, tab) = cx
        .update_window(o.handle, |_, window, _| (window.find("terminal").bounds(), window.within("tab-bar").find(0usize).bounds()))
        .unwrap();
    assert!(tab.top() >= terminal.bottom() - px(1.), "the tab {tab:?} is below the terminal {terminal:?}");
});

/// The settings tab, when the window has one.
fn settings_tab(cx: &mut TestAppContext, o: &Opened) -> Option<Entity<SettingsPanel>> {
    o.view.read_with(cx, |v, _| v.settings.clone())
}

/// The settings tabs in the dock.
fn settings_in_dock(cx: &mut TestAppContext, o: &Opened) -> usize {
    let state = o.view.read_with(cx, |v, cx| v.dock.read(cx).dump(cx));
    fn count(p: &gpui_kit::component::dock::PanelState) -> usize {
        usize::from(p.panel_name == "settings") + p.children.iter().map(count).sum::<usize>()
    }
    count(&state.center)
}

fn has_focus<P: gpui_kit::Focusable>(cx: &mut TestAppContext, o: &Opened, panel: &Entity<P>) -> bool {
    cx.update_window(o.handle, |_, window, cx| panel.read(cx).focus_handle(cx).is_focused(window)).unwrap()
}

fn press(cx: &mut TestAppContext, o: &Opened, key: &str) {
    cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    cx.run_until_parked();
    frame(cx, o.handle);
}

fn click(cx: &mut TestAppContext, o: &Opened, id: &'static str) {
    cx.update_window(o.handle, |_, window, cx| window.click(id, cx)).unwrap();
    cx.run_until_parked();
    frame(cx, o.handle);
}

scenario!(wndwi_b15_01_o_botao_de_configuracao, "WNDWI-B15#01: O botão de configuração abre a configuração", |cx| {
    let o = open_gnome(cx);
    click(cx, &o, "settings");
    let tab = settings_tab(cx, &o).expect("the settings tab");
    assert_eq!((settings_in_dock(cx, &o), has_focus(cx, &o, &tab)), (1, true));
});

scenario!(wndwi_b15_02_a_configuracao_so_abre_uma_vez, "WNDWI-B15#02: A configuração só abre uma vez", |cx| {
    let o = open_gnome(cx);
    click(cx, &o, "settings");
    let first = settings_tab(cx, &o).expect("the settings tab");
    new_terminal(cx, o.handle);
    click(cx, &o, "settings");
    let again = settings_tab(cx, &o).expect("the settings tab");
    assert_eq!((settings_in_dock(cx, &o), again == first, has_focus(cx, &o, &again)), (1, true, true));
});

scenario!(wndwi_b16_01_o_atalho_de_aba_nova, "WNDWI-B16#01: O atalho de aba nova abre o escolhedor", |cx| {
    let o = open_gnome(cx);
    click(cx, &o, "terminal");
    press(cx, &o, "ctrl-shift-t");
    press(cx, &o, "enter");
    assert_eq!((terminals(cx, &o), o.made.borrow().sinks[0].take()), (2, Vec::new()));
});

scenario!(wndwi_b16_02_o_atalho_de_fechar, "WNDWI-B16#02: O atalho de fechar fecha a aba com o foco", |cx| {
    let o = open_gnome(cx);
    let first = o.view.read_with(cx, |v, _| v.terminals[0].clone());
    new_terminal(cx, o.handle);
    press(cx, &o, "ctrl-shift-w");
    let left = o.view.read_with(cx, |v, _| v.terminals.clone());
    assert_eq!(left, vec![first]);
});

scenario!(wndwi_b16_03_os_atalhos_seguem_o_mapa, "WNDWI-B16#03: Os atalhos seguem o mapa da aplicação", |cx| {
    let o = open_gnome(cx);
    cx.update(|cx| {
        let mut keymap = Keymap::default();
        keymap.set(Action::Settings, "ctrl-alt-s").unwrap();
        cx.set_global(keymap);
    });
    click(cx, &o, "terminal");
    press(cx, &o, "ctrl-alt-s");
    assert_eq!(settings_in_dock(cx, &o), 1);
});

fn chooser(cx: &mut TestAppContext, o: &Opened) -> Option<gpui_kit::EntityId> {
    o.view.read_with(cx, |v, _| v.chooser().map(|c| c.entity_id()))
}

scenario!(wndwi_b17_01_o_esc_fecha_o_escolhedor, "WNDWI-B17#01: O Esc fecha o escolhedor e devolve o foco", |cx| {
    let o = open_gnome(cx);
    click(cx, &o, "terminal");
    click(cx, &o, "new-tab");
    let opened = chooser(cx, &o).is_some();
    press(cx, &o, "escape");
    let first = o.view.read_with(cx, |v, _| v.terminals[0].clone());
    assert_eq!((opened, chooser(cx, &o), terminals(cx, &o), has_focus(cx, &o, &first)), (true, None, 1, true));
});

scenario!(wndwi_b17_02_so_um_escolhedor, "WNDWI-B17#02: Só um escolhedor de cada vez", |cx| {
    let o = open_gnome(cx);
    click(cx, &o, "new-tab");
    let before = chooser(cx, &o);
    press(cx, &o, "ctrl-shift-t");
    assert!(before.is_some());
    assert_eq!(chooser(cx, &o), before);
});

scenario!(wndwi_b17_03_carregar_fora_fecha_o_escolhedor, "WNDWI-B17#03: Carregar fora do escolhedor fecha-o", |cx| {
    let o = open_gnome(cx);
    click(cx, &o, "new-tab");
    click(cx, &o, "new-tab-backdrop");
    assert_eq!((chooser(cx, &o), terminals(cx, &o)), (None, 1));
});

/// Chooses the kind numbered `n` in the chooser the "+" opens.
fn choose_kind(cx: &mut TestAppContext, o: &Opened, n: usize) {
    click(cx, o, "new-tab");
    press(cx, o, &n.to_string());
}

fn editors(cx: &mut TestAppContext, o: &Opened) -> Vec<Option<std::path::PathBuf>> {
    o.view.read_with(cx, |v, cx| v.editors.iter().map(|e| e.read(cx).path().map(std::path::PathBuf::from)).collect())
}

/// A folder with a "notas.txt".
fn with_notes() -> tempfile::TempDir {
    let dir = tempfile::tempdir().unwrap();
    std::fs::write(dir.path().join("notas.txt"), "ola").unwrap();
    dir
}

scenario!(wndwi_b18_01_a_arvore_abre_a_pasta_da_janela, "WNDWI-B18#01: A árvore de ficheiros abre a pasta da janela", |cx| {
    let dir = with_notes();
    let o = open_in(cx, gnome_left(), usize::MAX, None, dir.path().to_path_buf());
    choose_kind(cx, &o, 2);
    let tree = o.view.read_with(cx, |v, _| v.trees.first().cloned()).expect("the tree");
    let (root, listing) = tree.read_with(cx, |t, _| (t.root().to_path_buf(), t.listing()));
    assert_eq!((root, listing, has_focus(cx, &o, &tree)), (dir.path().to_path_buf(), vec!["notas.txt".to_string()], true));
});

scenario!(wndwi_b18_02_um_ficheiro_da_arvore_abre_num_editor, "WNDWI-B18#02: Um ficheiro aberto na árvore abre num editor, uma vez", |cx| {
    let dir = with_notes();
    let o = open_in(cx, gnome_left(), usize::MAX, None, dir.path().to_path_buf());
    choose_kind(cx, &o, 2);
    click(cx, &o, "entry-notas.txt");
    let tree = o.view.read_with(cx, |v, _| v.trees[0].clone());
    let id = PanelId::from(tree.entity_id());
    let dock = o.view.read_with(cx, |v, _| v.dock.clone());
    cx.update_window(o.handle, |_, window, cx| dock.update(cx, |d, cx| d.select_panel(id, window, cx))).unwrap();
    frame(cx, o.handle);
    click(cx, &o, "entry-notas.txt");
    assert_eq!(editors(cx, &o), vec![Some(dir.path().join("notas.txt"))]);
});

scenario!(wndwi_b19_01_o_editor_vazio, "WNDWI-B19#01: O editor vazio", |cx| {
    let o = open_gnome(cx);
    choose_kind(cx, &o, 3);
    let editor = o.view.read_with(cx, |v, _| v.editors[0].clone());
    assert_eq!((editors(cx, &o), has_focus(cx, &o, &editor)), (vec![None], true));
});

scenario!(wndwi_b19_02_abrir_um_ficheiro_do_computador, "WNDWI-B19#02: Abrir um ficheiro do computador", |cx| {
    let dir = with_notes();
    let o = open_gnome(cx);
    choose_kind(cx, &o, 4);
    let chosen = dir.path().join("notas.txt");
    cx.simulate_path_prompt_response(move |_| Some(vec![chosen]));
    cx.run_until_parked();
    assert_eq!(editors(cx, &o), vec![Some(dir.path().join("notas.txt"))]);
});

scenario!(wndwi_b19_03_cancelar_o_dialogo, "WNDWI-B19#03: Cancelar o diálogo deixa a doca como estava", |cx| {
    let o = open_gnome(cx);
    choose_kind(cx, &o, 4);
    cx.simulate_path_prompt_response(|_| None);
    cx.run_until_parked();
    assert_eq!((editors(cx, &o), terminals(cx, &o)), (Vec::new(), 1));
});

scenario!(wndwi_b12_03_a_arvore_e_o_editor_voltam, "WNDWI-B12#03: A árvore e o editor voltam com a pasta e o ficheiro", |cx| {
    let dir = with_notes();
    let file = LayoutFile::at(dir.path().join("layout.json"));
    let first = open_in(cx, gnome_left(), usize::MAX, Some(file.clone()), dir.path().to_path_buf());
    choose_kind(cx, &first, 2);
    click(cx, &first, "entry-notas.txt");
    settle(cx);
    cx.update_window(first.handle, |_, window, _| window.remove_window()).unwrap();
    cx.run_until_parked();

    let again = open_in(cx, gnome_left(), usize::MAX, Some(file), std::env::temp_dir());
    let roots = again.view.read_with(cx, |v, cx| v.trees.iter().map(|t| t.read(cx).root().to_path_buf()).collect::<Vec<_>>());
    assert_eq!((roots, editors(cx, &again)), (vec![dir.path().to_path_buf()], vec![Some(dir.path().join("notas.txt"))]));
});
