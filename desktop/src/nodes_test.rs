// @anchors
//   code: MQTST
//   ref: MQNAS

use super::*;
use crate::remote::LOCAL; // @dep: RMIMP
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, SharedString, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::cell::RefCell;
use std::rc::Rc;

struct Opened {
    handle: AnyWindowHandle,
    asked: Rc<RefCell<Vec<SessionTab>>>,
    _panel: Entity<NodesPanel>,
}

/// An installation in a new folder; listening when `listen` is not empty.
fn installation(listen: &str) -> (tempfile::TempDir, Station) {
    let dir = tempfile::tempdir().unwrap();
    let st = Station::open(dir.path(), listen, "/bin/sh").unwrap();
    (dir, st)
}

/// Another installation, listening, that lets `me` in.
fn other_for(me: &Station) -> (tempfile::TempDir, Station, String) {
    let (dir, other) = installation("127.0.0.1:0");
    std::fs::write(dir.path().join("authorized_keys"), me.status().unwrap().key).unwrap();
    let addr = other.status().unwrap().listen;
    (dir, other, addr)
}

fn open(cx: &mut TestAppContext, st: &Station) -> Opened {
    cx.update(gpui_kit::init);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(900.), px(700.)) })),
        ..WindowOptions::default()
    };
    let st = st.clone();
    let (handle, panel) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| NodesPanel::new(st, window, cx))))
        .unwrap();
    let asked: Rc<RefCell<Vec<SessionTab>>> = Rc::default();
    {
        let asked = asked.clone();
        cx.update(|cx| {
            cx.subscribe(&panel, move |_, ev: &NodesEvent, _| match ev {
                NodesEvent::Open(tab) => asked.borrow_mut().push(tab.clone()),
            })
            .detach()
        });
    }
    let o = Opened { handle, asked, _panel: panel };
    settle(cx, &o);
    o
}

fn settle(cx: &mut TestAppContext, o: &Opened) {
    for _ in 0..3 {
        cx.run_until_parked();
        cx.update_window(o.handle, |_, window, cx| window.render_frame(cx)).unwrap();
    }
}

fn click(cx: &mut TestAppContext, o: &Opened, id: String) {
    let id = SharedString::from(id);
    cx.update_window(o.handle, |_, window, cx| window.click(id, cx)).unwrap();
    settle(cx, o);
}

fn shown(cx: &mut TestAppContext, o: &Opened, id: String) -> Option<String> {
    let id = SharedString::from(id);
    cx.update_window(o.handle, |_, window, _| window.try_find(id).map(|e| e.label().unwrap_or_default().to_string())).unwrap()
}

scenario!(mqnas_b01_as_maquinas_e_as_sessoes, "MQNAS-B01: As máquinas e as sessões delas", |cx| {
    let (_d, me) = installation("");
    let mine = me.new_session(LOCAL, 24, 80).unwrap();
    let (_od, other, addr) = other_for(&me);
    let theirs = other.new_session(LOCAL, 24, 80).unwrap();
    me.add(&addr).unwrap();
    let o = open(cx, &me);
    let key = shown(cx, &o, "machine-key".into());
    let (a, b) = (shown(cx, &o, format!("machine-{LOCAL}")), shown(cx, &o, format!("machine-{addr}")));
    let ys = cx
        .update_window(o.handle, |_, window, _| {
            [format!("session-{LOCAL}-{mine}"), format!("machine-{addr}"), format!("session-{addr}-{theirs}")]
                .map(|id| window.find(SharedString::from(id)).bounds().origin.y)
        })
        .unwrap();
    assert_eq!(key, Some(me.status().unwrap().key));
    assert!(a.is_some() && b.is_some(), "{a:?} {b:?}");
    assert!(ys[0] < ys[1] && ys[1] < ys[2], "this machine and its session first: {ys:?}");
});

scenario!(mqnas_b02_atualizar, "MQNAS-B02: Atualizar", |cx| {
    let (_d, me) = installation("");
    let o = open(cx, &me);
    let id = me.new_session(LOCAL, 24, 80).unwrap();
    let before = shown(cx, &o, format!("session-{LOCAL}-{id}"));
    click(cx, &o, "nodes-refresh".into());
    assert_eq!((before, shown(cx, &o, format!("session-{LOCAL}-{id}")).is_some()), (None, true));
});

scenario!(mqnas_b03_abrir_uma_sessao, "MQNAS-B03: Abrir uma sessão", |cx| {
    let (_d, me) = installation("");
    let id = me.new_session(LOCAL, 24, 80).unwrap();
    let o = open(cx, &me);
    click(cx, &o, format!("session-{LOCAL}-{id}"));
    let asked: Vec<(String, String, bool, bool)> =
        o.asked.borrow().iter().map(|t| (t.machine.clone(), t.id.clone(), t.owned, t.writable)).collect();
    assert_eq!(asked, vec![(LOCAL.to_string(), id, false, true)]);
});

scenario!(mqnas_b04_uma_sessao_nova, "MQNAS-B04: Uma sessão nova", |cx| {
    let (_d, me) = installation("");
    let o = open(cx, &me);
    click(cx, &o, format!("new-{LOCAL}"));
    let made = me.sessions(LOCAL).unwrap();
    let asked: Vec<(String, bool)> = o.asked.borrow().iter().map(|t| (t.id.clone(), t.owned)).collect();
    assert_eq!(made.len(), 1);
    assert_eq!(asked, vec![(made[0].id.clone(), true)]);
});

scenario!(mqnas_b05_travar_as_sessoes_desta_maquina, "MQNAS-B05: Travar as sessões desta máquina", |cx| {
    let (_d, me) = installation("");
    let id = me.new_session(LOCAL, 24, 80).unwrap();
    let o = open(cx, &me);
    let before = shown(cx, &o, format!("lock-{id}"));
    click(cx, &o, format!("lock-{id}"));
    let after = shown(cx, &o, format!("lock-{id}"));
    assert!(me.sessions(LOCAL).unwrap()[0].locked);
    assert_eq!((before.as_deref(), after.as_deref()), (Some("Travar"), Some("Destravar")));
});

scenario!(mqnas_b06_juntar_e_esquecer_uma_maquina, "MQNAS-B06: Juntar e esquecer uma máquina", |cx| {
    let (_d, me) = installation("");
    let (_od, _other, addr) = other_for(&me);
    let o = open(cx, &me);
    click(cx, &o, "join-addr".into());
    cx.update_window(o.handle, |_, window, cx| window.input(&addr, cx)).unwrap();
    click(cx, &o, "join".into());
    let joined = shown(cx, &o, format!("machine-{addr}")).is_some();
    click(cx, &o, format!("remove-{addr}"));
    let names: Vec<String> = me.machines().unwrap().into_iter().map(|m| m.name).collect();
    assert_eq!((joined, shown(cx, &o, format!("machine-{addr}")), names), (true, None, vec![LOCAL.to_string()]));
});

scenario!(mqnas_e01_uma_maquina_que_nao_responde, "MQNAS-E01: Uma máquina que não responde", |cx| {
    let (_d, me) = installation("");
    let mine = me.new_session(LOCAL, 24, 80).unwrap();
    let (_od, other, addr) = other_for(&me);
    me.add(&addr).unwrap();
    drop(other);
    let o = open(cx, &me);
    click(cx, &o, "nodes-refresh".into());
    let why = shown(cx, &o, format!("machine-error-{addr}"));
    assert!(why.is_some_and(|w| !w.is_empty()));
    assert!(shown(cx, &o, format!("session-{LOCAL}-{mine}")).is_some());
});

scenario!(mqnas_e02_juntar_uma_maquina_que_nao_responde, "MQNAS-E02: Juntar uma máquina que não responde", |cx| {
    let (_d, me) = installation("");
    let free = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
    let addr = free.local_addr().unwrap().to_string();
    drop(free);
    let o = open(cx, &me);
    click(cx, &o, "join-addr".into());
    cx.update_window(o.handle, |_, window, cx| window.input(&addr, cx)).unwrap();
    click(cx, &o, "join".into());
    let message = shown(cx, &o, "join-message".into()).unwrap_or_default();
    assert!(message.contains(&addr), "{message}");
    assert_eq!(me.machines().unwrap().len(), 1);
});
