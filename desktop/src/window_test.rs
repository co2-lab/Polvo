// @anchors
//   code: WNTSW
//   ref: WNDWI

use super::*;
use crate::prefs::{Button, Prefs}; // @dep: PRIMP
use crate::screen::tests::Sink; // @dep: SCIMS
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
                let first = new_terminal(&prefs, window, cx).unwrap();
                cx.new(|cx| PolvoWindow::new(prefs, first, new_terminal, window, cx))
            })
        })
        .unwrap();
    frame(cx, handle);
    frame(cx, handle);
    Opened { handle, view, made }
}

fn open_gnome(cx: &mut TestAppContext) -> Opened {
    open_with(cx, gnome_left(), usize::MAX)
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
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
    cx.update_window(o.handle, |_, window, cx| window.click("new-terminal", cx)).unwrap();
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
    cx.update_window(o.handle, |_, window, cx| window.click("new-terminal", cx)).unwrap();
    frame(cx, o.handle);
    cx.update_window(o.handle, |_, window, cx| window.press("x", cx)).unwrap();
    assert_eq!(terminals(cx, &o), 2);
    let m = o.made.borrow();
    assert_eq!((m.sinks[0].take(), m.sinks[1].take()), (Vec::new(), b"x".to_vec()));
});

scenario!(wndwi_b11_o_fim_de_uma_shell_tira_a_aba, "WNDWI-B11: O fim de uma shell tira a aba, e a última fecha a janela", |cx| {
    let o = open_gnome(cx);
    cx.update_window(o.handle, |_, window, cx| window.click("new-terminal", cx)).unwrap();
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
    cx.update_window(o.handle, |_, window, cx| window.click("new-terminal", cx)).unwrap();
    frame(cx, o.handle);
    assert_eq!(terminals(cx, &o), 1);
});
