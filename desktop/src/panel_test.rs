// @anchors
//   code: TPTTR
//   ref: TRPNT

use super::*;
use crate::keymap::{Action, Keymap}; // @dep: KYIMA
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::screen::tests::Sink; // @dep: SCIMS
use crate::screen::{Event, Screen, Size}; // @dep: SCIMS
use futures::channel::mpsc;
use gpui_kit::component::dock::{DockLayout, DockSkin, panel_handle};
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{
    AnyWindowHandle, Bounds, Entity, Modifiers, MouseButton, MouseDownEvent, MouseUpEvent, PlatformInput, ScrollDelta,
    TestAppContext, WindowBounds, WindowOptions, point, px, size,
};
use std::cell::RefCell;
use std::rc::Rc;

fn prefs() -> Prefs {
    Prefs { left: vec![Button::Close], right: vec![], dark: true, font: "DejaVu Sans Mono".into(), font_px: 15. }
}

fn options() -> WindowOptions {
    WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    }
}

fn offline() -> (Screen, Sink) {
    let sink = Sink::default();
    (Screen::new(Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 }, Box::new(sink.clone())).unwrap(), sink)
}

struct Opened {
    handle: AnyWindowHandle,
    panel: Entity<TerminalPanel>,
    sink: Sink,
}

fn open_with(cx: &mut TestAppContext, prefs: Prefs, events: Option<Events>) -> Opened {
    cx.update(gpui_kit::init);
    let (screen, sink) = offline();
    let (handle, panel) = cx
        .update(|cx| {
            gpui_kit::open_window(options(), cx, |window, cx| {
                cx.new(|cx| TerminalPanel::new(screen, events, "/bin/bash", &prefs, window, cx))
            })
        })
        .unwrap();
    frame(cx, handle);
    frame(cx, handle);
    Opened { handle, panel, sink }
}

fn open_panel(cx: &mut TestAppContext) -> Opened {
    open_with(cx, prefs(), None)
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

fn feed(cx: &mut TestAppContext, o: &Opened, data: &[u8]) {
    o.panel.update(cx, |p, cx| {
        p.screen.feed(data);
        cx.notify();
    });
    frame(cx, o.handle);
}

/// The window position of the middle of a cell of the terminal.
fn cell(cx: &mut TestAppContext, o: &Opened, col: u16, row: u16) -> gpui_kit::Point<gpui_kit::Pixels> {
    let origin = cx.update_window(o.handle, |_, window, _| window.find("terminal").bounds().origin).unwrap();
    let (cw, ch) = o.panel.read_with(cx, |p, _| (p.cell_w, p.line_px));
    point(origin.x + px(PADDING) + px(cw * (f32::from(col) + 0.5)), origin.y + px(PADDING) + px(ch * (f32::from(row) + 0.5)))
}

/// A drag over a row of the terminal, from the start of one cell to the end of another.
fn drag_over(cx: &mut TestAppContext, o: &Opened, from: u16, to: u16, row: u16) {
    let cw = o.panel.read_with(cx, |p, _| p.cell_w);
    let a = cell(cx, o, from, row) - point(px(cw * 0.4), px(0.));
    let b = cell(cx, o, to, row) + point(px(cw * 0.4), px(0.));
    cx.update_window(o.handle, |_, window, cx| window.drag(a, b, cx)).unwrap();
}

scenario!(trpnt_b01_o_teclado_vai_para_a_shell, "TRPNT-B01: O teclado vai para a shell", |cx| {
    let o = open_panel(cx);
    for key in ["a", "enter", "ctrl-c"] {
        cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    }
    assert_eq!(o.sink.take(), b"a\r\x03".to_vec());
});

scenario!(trpnt_b02_01_copiar_e_colar, "TRPNT-B02#01: Ctrl+Shift+C copia a seleção e Ctrl+Shift+V cola", |cx| {
    let o = open_panel(cx);
    feed(cx, &o, b"abc def\r\n");
    drag_over(cx, &o, 4, 6, 0);
    o.sink.take();
    cx.update_window(o.handle, |_, window, cx| window.press("ctrl-shift-c", cx)).unwrap();
    let copied = cx.update(|cx| cx.read_from_clipboard().and_then(|i| i.text()));
    assert_eq!(copied.as_deref(), Some("def"));
    cx.update_window(o.handle, |_, window, cx| window.press("ctrl-shift-v", cx)).unwrap();
    assert_eq!(o.sink.take(), b"def".to_vec());
});

scenario!(trpnt_b02_03_copiar_segue_o_mapa_de_atalhos, "TRPNT-B02#03: Copiar segue o mapa de atalhos, e as ações da janela não vão para a shell", |cx| {
    let o = open_panel(cx);
    let mut keymap = Keymap::default();
    keymap.set(Action::Copy, "ctrl-alt-c").unwrap();
    cx.update(|cx| cx.set_global(keymap));
    feed(cx, &o, b"abc def\r\n");
    drag_over(cx, &o, 4, 6, 0);
    o.sink.take();
    for key in ["ctrl-alt-c", "ctrl-shift-t"] {
        cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    }
    let copied = cx.update(|cx| cx.read_from_clipboard().and_then(|i| i.text()));
    assert_eq!((copied.as_deref(), o.sink.take()), (Some("def"), Vec::new()));
});

scenario!(trpnt_b02_02_a_selecao_primaria, "TRPNT-B02#02: Selecionar copia para a seleção primária e o botão do meio cola", |cx| {
    let o = open_panel(cx);
    feed(cx, &o, b"abc def\r\n");
    drag_over(cx, &o, 0, 2, 0);
    let primary = cx.update(|cx| cx.read_from_primary().and_then(|i| i.text()));
    assert_eq!(primary.as_deref(), Some("abc"));
    o.sink.take();
    let at = cell(cx, &o, 5, 3);
    cx.update_window(o.handle, |_, window, cx| {
        let down = MouseDownEvent { button: MouseButton::Middle, position: at, modifiers: Modifiers::default(), click_count: 1, first_mouse: false };
        window.dispatch_event(PlatformInput::MouseDown(down), cx);
        let up = MouseUpEvent { button: MouseButton::Middle, position: at, modifiers: Modifiers::default(), click_count: 1 };
        window.dispatch_event(PlatformInput::MouseUp(up), cx);
    })
    .unwrap();
    assert_eq!(o.sink.take(), b"abc".to_vec());
});

scenario!(trpnt_b03_o_rato_seleciona_e_a_roda_faz_scroll, "TRPNT-B03: O rato seleciona e a roda faz scroll", |cx| {
    let o = open_panel(cx);
    let lines: Vec<String> = (1..=100).map(|n| format!("linha {n}")).collect();
    feed(cx, &o, lines.join("\r\n").as_bytes());
    let top = |cx: &mut TestAppContext| o.panel.update(cx, |p, _| p.screen.text().lines().next().unwrap_or("").to_string());
    let before = top(cx);
    cx.update_window(o.handle, |_, window, cx| window.scroll("terminal", ScrollDelta::Lines(point(0., 1.)), cx)).unwrap();
    frame(cx, o.handle);
    let after = top(cx);
    let n = |s: &str| s.trim_start_matches("linha ").parse::<i32>().unwrap();
    assert_eq!(n(&before) - n(&after), SCROLL_LINES as i32, "{before} -> {after}");
    drag_over(cx, &o, 0, 7, 0);
    let selected = o.panel.read_with(cx, |p, _| p.screen.selection_text());
    assert_eq!(selected.as_deref(), Some(after.as_str()));
});

scenario!(trpnt_b04_o_terminal_enche_o_painel, "TRPNT-B04: O terminal enche o painel", |cx| {
    let o = open_panel(cx);
    let area = cx.update_window(o.handle, |_, window, _| window.find("terminal").bounds().size).unwrap();
    let (size, cw, lh) = o.panel.read_with(cx, |p, _| (p.screen.size(), p.cell_w, p.line_px));
    let want_cols = ((f32::from(area.width) - 2. * PADDING) / cw).floor() as u16;
    let want_rows = ((f32::from(area.height) - 2. * PADDING) / lh).floor() as u16;
    assert_eq!((size.cols, size.rows), (want_cols, want_rows));
});

scenario!(trpnt_b05_a_fonte_segue_as_preferencias, "TRPNT-B05: A fonte segue as preferências", |cx| {
    let o = open_with(cx, Prefs { font: "Fira Code".into(), font_px: 15., ..prefs() }, None);
    let (family, font_px) = o.panel.read_with(cx, |p, _| (p.font.family.to_string(), p.font_px));
    assert_eq!((family.as_str(), font_px), ("Fira Code", 15.));
});

scenario!(trpnt_b06_a_aba_diz_o_titulo_da_shell, "TRPNT-B06: A aba diz o título da shell", |cx| {
    let o = open_panel(cx);
    assert_eq!(o.panel.read_with(cx, |p, _| p.tab_title()), "bash");
    feed(cx, &o, b"\x1b]2;~/projeto\x07");
    assert_eq!(o.panel.read_with(cx, |p, _| p.tab_title()), "~/projeto");
});

scenario!(trpnt_b07_o_fim_da_shell_avisa, "TRPNT-B07: O fim da shell avisa", |cx| {
    let (tx, rx) = mpsc::unbounded();
    let o = open_with(cx, prefs(), Some(rx));
    let heard = Rc::new(RefCell::new(Vec::new()));
    let h = heard.clone();
    let _sub = cx.update(|cx| cx.subscribe(&o.panel, move |_, ev: &TerminalEvent, _| h.borrow_mut().push(*ev)));
    tx.unbounded_send(Event::Exit).unwrap();
    cx.run_until_parked();
    assert_eq!(*heard.borrow(), vec![TerminalEvent::Exited]);
});

scenario!(trpnt_b08_clicar_da_o_foco, "TRPNT-B08: Clicar dá o foco", |cx| {
    cx.update(gpui_kit::init);
    let ((s1, sink1), (s2, sink2)) = (offline(), offline());
    let panels: Rc<RefCell<Vec<Entity<TerminalPanel>>>> = Rc::default();
    let p = panels.clone();
    let (handle, _) = cx
        .update(|cx| {
            gpui_kit::open_window(options(), cx, |window, cx| {
                let (area, _) = DockSkin::dock_area("panels", None, window, cx);
                let a = cx.new(|cx| TerminalPanel::new(s1, None, "/bin/sh", &prefs(), window, cx));
                let b = cx.new(|cx| TerminalPanel::new(s2, None, "/bin/sh", &prefs(), window, cx));
                p.borrow_mut().extend([a.clone(), b.clone()]);
                let layout = DockLayout::h_split()
                    .child(DockLayout::tabs().panel_view(panel_handle(a), cx), None)
                    .child(DockLayout::tabs().panel_view(panel_handle(b), cx), None);
                area.update(cx, |area, cx| area.set_center(layout, window, cx));
                // The focus starts on the first panel.
                let first = p.borrow()[0].read(cx).focus_handle(cx);
                window.focus(&first, cx);
                area
            })
        })
        .unwrap();
    frame(cx, handle);
    frame(cx, handle);
    assert_eq!(panels.borrow().len(), 2);
    // The middle of the right half of the window: the second panel.
    let at = point(px(600.), px(300.));
    cx.update_window(handle, |_, window, cx| {
        let down = MouseDownEvent { button: MouseButton::Left, position: at, modifiers: Modifiers::default(), click_count: 1, first_mouse: false };
        window.dispatch_event(PlatformInput::MouseDown(down), cx);
        let up = MouseUpEvent { button: MouseButton::Left, position: at, modifiers: Modifiers::default(), click_count: 1 };
        window.dispatch_event(PlatformInput::MouseUp(up), cx);
        window.press("x", cx);
    })
    .unwrap();
    assert_eq!((sink1.take(), sink2.take()), (Vec::new(), b"x".to_vec()));
});

scenario!(trpnt_b09_um_terminal_que_nao_arrancou_diz_porque, "TRPNT-B09: Um terminal que não arrancou diz porquê", |cx| {
    cx.update(gpui_kit::init);
    let (handle, panel) = cx
        .update(|cx| {
            gpui_kit::open_window(options(), cx, |window, cx| {
                TerminalPanel::failed("/nao/existe/shell: no such file", "/bin/sh", &prefs(), window, cx).unwrap()
            })
        })
        .unwrap();
    frame(cx, handle);
    let text = panel.update(cx, |p, _| p.screen.text());
    assert!(text.contains("/nao/existe/shell: no such file"), "{text}");
});

scenario!(trpnt_e01_uma_shell_que_nao_arranca_nao_faz_painel, "TRPNT-E01: Uma shell que não arranca não faz painel", |cx| {
    cx.update(gpui_kit::init);
    let result = Rc::new(RefCell::new(None));
    let r = result.clone();
    let _ = cx.update(|cx| {
        gpui_kit::open_window(options(), cx, |window, cx| {
            *r.borrow_mut() = Some(TerminalPanel::spawn("/nao/existe/shell", None, &prefs(), window, cx).map(|_| ()));
            cx.new(|_| Blank)
        })
    });
    let err = result.borrow_mut().take().expect("the spawn ran").expect_err("an error");
    assert!(err.to_string().contains("/nao/existe/shell"), "{err}");
});

/// A view with nothing in it, for the window of a test that needs only its context.
struct Blank;

impl gpui_kit::Render for Blank {
    fn render(&mut self, _: &mut gpui_kit::Window, _: &mut gpui_kit::Context<Self>) -> impl gpui_kit::IntoElement {
        gpui_kit::div()
    }
}
