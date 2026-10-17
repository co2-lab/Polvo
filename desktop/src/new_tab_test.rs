// @anchors
//   code: NTTST
//   ref: NWTBS

use super::*;
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::cell::RefCell;
use std::rc::Rc;

const ONE: TabKind = TabKind { id: "um", label: "Um" };
const TWO: TabKind = TabKind { id: "dois", label: "Dois" };

struct Opened {
    handle: AnyWindowHandle,
    chooser: Entity<NewTabChooser>,
    heard: Rc<RefCell<Vec<ChooserEvent>>>,
}

fn open(cx: &mut TestAppContext) -> Opened {
    cx.update(gpui_kit::init);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(800.), px(600.)) })),
        ..WindowOptions::default()
    };
    let (handle, chooser) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| NewTabChooser::new(vec![ONE, TWO], window, cx))))
        .unwrap();
    let heard: Rc<RefCell<Vec<ChooserEvent>>> = Rc::default();
    {
        let heard = heard.clone();
        cx.update(|cx| cx.subscribe(&chooser, move |_, ev: &ChooserEvent, _| heard.borrow_mut().push(ev.clone())).detach());
    }
    frame(cx, handle);
    Opened { handle, chooser, heard }
}

fn frame(cx: &mut TestAppContext, handle: AnyWindowHandle) {
    cx.update_window(handle, |_, window, cx| window.render_frame(cx)).unwrap();
}

fn press(cx: &mut TestAppContext, o: &Opened, key: &str) {
    cx.update_window(o.handle, |_, window, cx| window.press(key, cx)).unwrap();
    frame(cx, o.handle);
}

fn heard(o: &Opened) -> Vec<ChooserEvent> {
    o.heard.borrow().clone()
}

scenario!(nwtbs_b01_os_tipos_com_o_primeiro_escolhido, "NWTBS-B01: Os tipos, com o primeiro escolhido", |cx| {
    let o = open(cx);
    let (one, two, focused) = cx
        .update_window(o.handle, |_, window, cx| {
            let one = window.find("kind-um");
            let two = window.find("kind-dois");
            let focused = o.chooser.read(cx).focus_handle(cx).is_focused(window);
            ((one.label().map(str::to_string), one.selected()), (two.label().map(str::to_string), two.selected(), one.bounds().origin.y < two.bounds().origin.y), focused)
        })
        .unwrap();
    assert_eq!(one, (Some("1 Um".to_string()), Some(true)));
    assert_eq!(two, (Some("2 Dois".to_string()), Some(false), true));
    assert!(focused, "the chooser has the focus");
});

scenario!(nwtbs_b02_01_as_setas_e_o_enter, "NWTBS-B02#01: As setas e o Enter", |cx| {
    let o = open(cx);
    for key in ["down", "down", "up", "enter"] {
        press(cx, &o, key);
    }
    assert_eq!(heard(&o), vec![ChooserEvent::Chosen(ONE)]);
});

scenario!(nwtbs_b02_02_a_tecla_do_numero, "NWTBS-B02#02: A tecla do número", |cx| {
    let o = open(cx);
    press(cx, &o, "2");
    assert_eq!(heard(&o), vec![ChooserEvent::Chosen(TWO)]);
});

scenario!(nwtbs_b03_escolher_com_o_rato, "NWTBS-B03: Escolher com o rato", |cx| {
    let o = open(cx);
    cx.update_window(o.handle, |_, window, cx| window.click("kind-dois", cx)).unwrap();
    assert_eq!(heard(&o), vec![ChooserEvent::Chosen(TWO)]);
});

scenario!(nwtbs_b04_fechar_sem_escolher, "NWTBS-B04: Fechar sem escolher", |cx| {
    let o = open(cx);
    press(cx, &o, "escape");
    assert_eq!(heard(&o), vec![ChooserEvent::Dismissed]);
});
