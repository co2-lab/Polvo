// @anchors
//   code: SPTST
//   ref: STPNS

use super::*;
use crate::keymap::{Action, Keymap}; // @dep: KYIMA
use crate::settings::SettingsFile; // @dep: STIMS
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, SharedString, TestAppContext, WindowBounds, WindowOptions, point, px, size};

struct Opened {
    handle: AnyWindowHandle,
    panel: Entity<SettingsPanel>,
}

fn open_with(cx: &mut TestAppContext, file: Option<SettingsFile>) -> Opened {
    cx.update(|cx| {
        gpui_kit::init(cx);
        cx.set_global(Keymap::default());
    });
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let (handle, panel) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| SettingsPanel::new(file, window, cx))))
        .unwrap();
    frame(cx, handle);
    Opened { handle, panel }
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

fn press(cx: &mut TestAppContext, o: &Opened, key: &str) {
    cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    frame(cx, o.handle);
}

fn click(cx: &mut TestAppContext, o: &Opened, id: &'static str) {
    cx.update_window(o.handle, |_, window, cx| window.click(id, cx)).unwrap();
    frame(cx, o.handle);
}

fn new_tab_shortcut(cx: &mut TestAppContext) -> String {
    cx.update(|cx| cx.global::<Keymap>().shortcut(Action::NewTab).to_string())
}

/// The shortcut the panel shows for an action.
fn shown(cx: &mut TestAppContext, o: &Opened, a: Action) -> String {
    let id = format!("shortcut-{}", a.id());
    cx.update_window(o.handle, |_, window, _| window.find(SharedString::from(id)).value().unwrap_or_default().to_string()).unwrap()
}

/// The message the panel shows, if any.
fn message(cx: &mut TestAppContext, o: &Opened) -> String {
    cx.update_window(o.handle, |_, window, _| window.try_find("keys-message").and_then(|m| m.label().map(str::to_string)))
        .unwrap()
        .unwrap_or_default()
}

scenario!(stpns_b01_as_seccoes_com_os_atalhos_abertos, "STPNS-B01: As secções, com os atalhos abertos", |cx| {
    let o = open_with(cx, None);
    let selected = cx.update_window(o.handle, |_, window, _| window.find("section-keys").selected()).unwrap();
    assert_eq!(selected, Some(true));
});

scenario!(stpns_b02_cada_acao_com_o_atalho_dela, "STPNS-B02: Cada ação com o atalho dela", |cx| {
    let o = open_with(cx, None);
    let all: Vec<String> = Action::ALL.into_iter().map(|a| shown(cx, &o, a)).collect();
    let defaults: Vec<String> = Action::ALL.into_iter().map(|a| Keymap::default().shortcut(a).to_string()).collect();
    assert_eq!(all, defaults);
});

scenario!(stpns_b03_mudar_um_atalho, "STPNS-B03: Mudar um atalho", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let file = SettingsFile::at(dir.path().join("settings.json"));
    let o = open_with(cx, Some(file.clone()));
    click(cx, &o, "shortcut-new-tab");
    press(cx, &o, "ctrl-alt-n");
    let now = shown(cx, &o, Action::NewTab);
    assert_eq!((new_tab_shortcut(cx), now.as_str()), ("ctrl-alt-n".to_string(), "ctrl-alt-n"));
    assert_eq!(file.read().keys.get("new-tab").map(String::as_str), Some("ctrl-alt-n"));
});

scenario!(stpns_b04_cancelar_e_os_modificadores_sozinhos, "STPNS-B04: Cancelar, e os modificadores sozinhos", |cx| {
    let o = open_with(cx, None);
    click(cx, &o, "shortcut-new-tab");
    press(cx, &o, "shift");
    let listening = o.panel.read_with(cx, |p, _| p.listening());
    press(cx, &o, "escape");
    assert_eq!((listening, o.panel.read_with(cx, |p, _| p.listening()), new_tab_shortcut(cx)), (Some(Action::NewTab), None, "ctrl-shift-t".to_string()));
});

scenario!(stpns_e01_o_atalho_de_outra_acao_e_recusado, "STPNS-E01: O atalho de outra ação é recusado", |cx| {
    let o = open_with(cx, None);
    click(cx, &o, "shortcut-new-tab");
    press(cx, &o, "ctrl-shift-w");
    let message = message(cx, &o);
    assert!(message.contains("Fechar a aba"), "{message}");
    assert_eq!(new_tab_shortcut(cx), "ctrl-shift-t");
});

scenario!(stpns_e02_uma_configuracao_que_nao_se_grava_avisa, "STPNS-E02: Uma configuração que não se grava avisa", |cx| {
    let dir = tempfile::tempdir().unwrap();
    let not_a_dir = dir.path().join("ficheiro");
    std::fs::write(&not_a_dir, "").unwrap();
    let o = open_with(cx, Some(SettingsFile::at(not_a_dir.join("settings.json"))));
    click(cx, &o, "shortcut-new-tab");
    press(cx, &o, "ctrl-alt-n");
    let message = message(cx, &o);
    assert!(message.contains("não foi gravada"), "{message}");
    assert_eq!(new_tab_shortcut(cx), "ctrl-alt-n");
});
