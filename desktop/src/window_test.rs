// @anchors
//   code: WNTSW
//   ref: WNDWI

use super::*;
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::screen::tests::Sink; // @dep: SCIMS
use crate::screen::{Event, Screen, Size}; // @dep: SCIMS
use futures::channel::mpsc;
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{
    AnyWindowHandle, Bounds, Entity, Modifiers, MouseButton, MouseDownEvent, MouseUpEvent,
    PlatformInput, ScrollDelta, TestAppContext, WindowBounds, WindowOptions, point, px, size,
};

fn prefs(left: &[Button], right: &[Button]) -> Prefs {
    Prefs { left: left.to_vec(), right: right.to_vec(), dark: true, font: "DejaVu Sans Mono".into(), font_px: 15. }
}

fn gnome_left() -> Prefs {
    prefs(&[Button::Close, Button::Minimize, Button::Maximize], &[])
}

struct Opened {
    handle: AnyWindowHandle,
    view: Entity<TerminalView>,
    sink: Sink,
}

fn open_with(cx: &mut TestAppContext, prefs: Prefs, events: Option<Events>) -> Opened {
    cx.update(gpui_kit::init);
    let sink = Sink::default();
    let screen = Screen::new(Size { rows: 24, cols: 80, cell_w: 9, cell_h: 20 }, Box::new(sink.clone())).unwrap();
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let (handle, view) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| TerminalView::new(screen, events, prefs, window, cx))))
        .unwrap();
    frame(cx, handle);
    frame(cx, handle);
    Opened { handle, view, sink }
}

fn open_gnome(cx: &mut TestAppContext) -> Opened {
    open_with(cx, gnome_left(), None)
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

fn x_of(cx: &mut TestAppContext, handle: AnyWindowHandle, id: &'static str) -> f32 {
    cx.update_window(handle, |_, window, _| f32::from(window.find(id).bounds().origin.x)).unwrap()
}

fn feed(cx: &mut TestAppContext, o: &Opened, data: &[u8]) {
    o.view.update(cx, |v, cx| {
        v.screen.feed(data);
        cx.notify();
    });
    frame(cx, o.handle);
}

/// The window position of the middle of a cell of the terminal.
fn cell(cx: &mut TestAppContext, o: &Opened, col: u16, row: u16) -> gpui_kit::Point<gpui_kit::Pixels> {
    let origin = cx.update_window(o.handle, |_, window, _| window.find("terminal").bounds().origin).unwrap();
    let (cw, ch) = o.view.read_with(cx, |v, _| (v.cell_w, v.line_px));
    point(
        origin.x + px(PADDING) + px(cw * (f32::from(col) + 0.5)),
        origin.y + px(PADDING) + px(ch * (f32::from(row) + 0.5)),
    )
}

/// A drag over a row of the terminal, from the start of one cell to the end of another.
fn drag_over(cx: &mut TestAppContext, o: &Opened, from: u16, to: u16, row: u16) {
    let cw = o.view.read_with(cx, |v, _| v.cell_w);
    let a = cell(cx, o, from, row) - point(px(cw * 0.4), px(0.));
    let b = cell(cx, o, to, row) + point(px(cw * 0.4), px(0.));
    cx.update_window(o.handle, |_, window, cx| window.drag(a, b, cx)).unwrap();
}

scenario!(wndwi_b01_01_os_botoes_a_esquerda, "WNDWI-B01#01: Os botões à esquerda pela ordem do sistema", |cx| {
    let o = open_gnome(cx);
    let xs: Vec<f32> = ["close", "minimize", "maximize", "title"].into_iter().map(|id| x_of(cx, o.handle, id)).collect();
    assert!(xs.windows(2).all(|w| w[0] < w[1]), "{xs:?}");
});

scenario!(wndwi_b01_02_os_botoes_a_direita, "WNDWI-B01#02: Os botões à direita pela ordem do sistema", |cx| {
    let o = open_with(cx, prefs(&[], &[Button::Minimize, Button::Maximize, Button::Close]), None);
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

scenario!(wndwi_b03_o_teclado_vai_para_a_shell, "WNDWI-B03: O teclado vai para a shell", |cx| {
    let o = open_gnome(cx);
    for key in ["a", "enter", "ctrl-c"] {
        cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    }
    assert_eq!(o.sink.take(), b"a\r\x03".to_vec());
});

scenario!(wndwi_b04_01_copiar_e_colar, "WNDWI-B04#01: Ctrl+Shift+C copia a seleção e Ctrl+Shift+V cola", |cx| {
    let o = open_gnome(cx);
    feed(cx, &o, b"abc def\r\n");
    drag_over(cx, &o, 4, 6, 0);
    o.sink.take();
    cx.update_window(o.handle, |_, window, cx| window.press("ctrl-shift-c", cx)).unwrap();
    let copied = cx.update(|cx| cx.read_from_clipboard().and_then(|i| i.text()));
    assert_eq!(copied.as_deref(), Some("def"));
    cx.update_window(o.handle, |_, window, cx| window.press("ctrl-shift-v", cx)).unwrap();
    assert_eq!(o.sink.take(), b"def".to_vec());
});

scenario!(wndwi_b04_02_a_selecao_primaria, "WNDWI-B04#02: Selecionar copia para a seleção primária e o botão do meio cola", |cx| {
    let o = open_gnome(cx);
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

scenario!(wndwi_b05_o_rato_seleciona_e_a_roda_faz_scroll, "WNDWI-B05: O rato seleciona e a roda faz scroll", |cx| {
    let o = open_gnome(cx);
    let lines: Vec<String> = (1..=100).map(|n| format!("linha {n}")).collect();
    feed(cx, &o, lines.join("\r\n").as_bytes());
    let top = |cx: &mut TestAppContext| o.view.update(cx, |v, _| v.screen.text().lines().next().unwrap_or("").to_string());
    let before = top(cx);
    cx.update_window(o.handle, |_, window, cx| window.scroll("terminal", ScrollDelta::Lines(point(0., 1.)), cx)).unwrap();
    frame(cx, o.handle);
    let after = top(cx);
    let n = |s: &str| s.trim_start_matches("linha ").parse::<i32>().unwrap();
    assert_eq!(n(&before) - n(&after), SCROLL_LINES as i32, "{before} -> {after}");
    drag_over(cx, &o, 0, 7, 0);
    let selected = o.view.read_with(cx, |v, _| v.screen.selection_text());
    assert_eq!(selected.as_deref(), Some(after.as_str()));
});

scenario!(wndwi_b06_o_terminal_enche_a_janela, "WNDWI-B06: O terminal enche a janela", |cx| {
    let o = open_gnome(cx);
    let area = cx.update_window(o.handle, |_, window, _| window.find("terminal").bounds().size).unwrap();
    let (size, cw, lh) = o.view.read_with(cx, |v, _| (v.screen.size(), v.cell_w, v.line_px));
    let want_cols = ((f32::from(area.width) - 2. * PADDING) / cw).floor() as u16;
    let want_rows = ((f32::from(area.height) - 2. * PADDING) / lh).floor() as u16;
    assert_eq!((size.cols, size.rows), (want_cols, want_rows));
    assert!(f32::from(area.height) < 600. - TITLE_H + 1., "the terminal is below the title bar: {area:?}");
});

scenario!(wndwi_b07_o_tema_e_a_fonte_seguem_as_preferencias, "WNDWI-B07: O tema e a fonte seguem as preferências", |cx| {
    let light = Prefs { dark: false, font: "Fira Code".into(), font_px: 15., ..gnome_left() };
    let o = open_with(cx, light, None);
    let (bar, family, font_px) = o.view.read_with(cx, |v, _| (v.colors().bar, v.font.family.to_string(), v.font_px));
    assert!(bar.l > 0.5, "a light title bar: {bar:?}");
    assert_eq!((family.as_str(), font_px), ("Fira Code", 15.));
});

scenario!(wndwi_b08_o_fim_da_shell_fecha_a_janela, "WNDWI-B08: O fim da shell fecha a janela", |cx| {
    let (tx, rx) = mpsc::unbounded();
    let _o = open_with(cx, gnome_left(), Some(rx));
    tx.unbounded_send(Event::Exit).unwrap();
    cx.run_until_parked();
    assert!(cx.update(|cx| cx.windows().is_empty()));
});
