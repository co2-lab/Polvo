// @anchors
//   code: AGTSS
//   ref: AGNTS

use super::*;
use crate::remote::LOCAL; // @dep: RMIMP
use gpui_kit::test::TestWindowExt as _;
use gpui_kit::{AnyWindowHandle, Bounds, Entity, SharedString, TestAppContext, WindowBounds, WindowOptions, point, px, size};
use std::cell::RefCell;
use std::path::Path;
use std::rc::Rc;

/// The job of an agent, as the node keeps it in the installation's folder.
fn job(dir: &Path, agent: &str, json: &str) {
    let folder = dir.join("progress");
    std::fs::create_dir_all(&folder).unwrap();
    std::fs::write(folder.join(format!("{agent}.json")), json).unwrap();
}

fn entry(kind: &str, text: &str, i: usize) -> String {
    format!(r#"{{"at":"2026-10-18T10:00:{i:02}Z","kind":"{kind}","text":"{text}"}}"#)
}

/// An installation with these jobs.
fn installation(jobs: &[(&str, String)]) -> (tempfile::TempDir, Station) {
    let dir = tempfile::tempdir().unwrap();
    for (agent, json) in jobs {
        job(dir.path(), agent, json);
    }
    let st = Station::open(dir.path(), "", "/bin/sh").unwrap();
    (dir, st)
}

fn migrar() -> String {
    format!(
        r#"{{"agent":"a","goal":"migrar","steps":[{{"id":"1","title":"ler","state":"done"}},{{"id":"2","title":"escrever","state":"doing"}}],"log":[{}],"finished":false}}"#,
        entry("start", "migrar", 0)
    )
}

struct Opened {
    handle: AnyWindowHandle,
    feed: Entity<AgentsFeed>,
}

fn open(cx: &mut TestAppContext, st: &Station) -> Opened {
    cx.update(gpui_kit::init);
    let options = WindowOptions {
        window_bounds: Some(WindowBounds::Windowed(Bounds { origin: point(px(0.), px(0.)), size: size(px(900.), px(900.)) })),
        ..WindowOptions::default()
    };
    let feed = cx.update(|cx| cx.new(|cx| AgentsFeed::new(st.clone(), false, cx)));
    let f = feed.clone();
    let (handle, _panel) = cx
        .update(|cx| gpui_kit::open_window(options, cx, |window, cx| cx.new(|cx| AgentsPanel::new(f, window, cx))))
        .unwrap();
    let o = Opened { handle, feed };
    settle(cx, &o);
    o
}

fn settle(cx: &mut TestAppContext, o: &Opened) {
    for _ in 0..3 {
        cx.run_until_parked();
        cx.update_window(o.handle, |_, window, cx| window.render_frame(cx)).unwrap();
    }
}

fn label(cx: &mut TestAppContext, o: &Opened, id: String) -> Option<String> {
    let id = SharedString::from(id);
    cx.update_window(o.handle, |_, window, _| window.try_find(id).map(|e| e.label().unwrap_or_default().to_string())).unwrap()
}

fn value(cx: &mut TestAppContext, o: &Opened, id: String) -> Option<String> {
    let id = SharedString::from(id);
    cx.update_window(o.handle, |_, window, _| window.try_find(id).and_then(|e| e.value().map(str::to_string))).unwrap()
}

fn click(cx: &mut TestAppContext, o: &Opened, id: String) {
    let id = SharedString::from(id);
    cx.update_window(o.handle, |_, window, cx| window.click(id, cx)).unwrap();
    settle(cx, o);
}

/// The texts of the questions the node has for an agent of this machine.
fn asked(st: &Station, agent: &str) -> Vec<String> {
    let all = st.progress_all(&Default::default(), 0).unwrap();
    all[0].agents.iter().filter(|a| a.agent == agent).flat_map(|a| a.questions.iter().map(|q| q.text.clone())).collect()
}

scenario!(agnts_b01_os_agentes_de_todas_as_maquinas, "AGNTS-B01: Os agentes de todas as máquinas", |cx| {
    let (_d, st) = installation(&[("a", migrar())]);
    let o = open(cx, &st);
    let head = value(cx, &o, format!("agent-{LOCAL}-a"));
    let steps = (value(cx, &o, "step-a-1".into()), value(cx, &o, "step-a-2".into()));
    assert_eq!(head.as_deref(), Some("migrar · 1/2"));
    assert_eq!(steps, (Some("done".to_string()), Some("doing".to_string())));
});

scenario!(agnts_b02_os_ultimos_estados, "AGNTS-B02: Os últimos estados", |cx| {
    let notes: Vec<String> = (1..=10).map(|i| entry("note", &format!("nota {i}"), i)).collect();
    let json = format!(r#"{{"agent":"a","goal":"g","steps":[],"log":[{},{}],"finished":false}}"#, entry("start", "g", 0), notes.join(","));
    let (_d, st) = installation(&[("a", json)]);
    let o = open(cx, &st);
    let shown: Vec<Option<String>> = (0..9).map(|i| label(cx, &o, format!("entry-a-{i}"))).collect();
    assert_eq!(shown[0].as_deref(), Some("nota 10"));
    assert_eq!(shown[7].as_deref(), Some("nota 3"));
    assert_eq!(shown[8], None);
});

scenario!(agnts_b03_01_perguntar_uma_pergunta_comum, "AGNTS-B03#01: Perguntar uma pergunta comum", |cx| {
    let (_d, st) = installation(&[("a", migrar())]);
    let o = open(cx, &st);
    click(cx, &o, "ask-a-0".into());
    assert_eq!(asked(&st, "a"), vec![COMMON_QUESTIONS[0].to_string()]);
});

scenario!(agnts_b03_02_perguntar_uma_pergunta_escrita, "AGNTS-B03#02: Perguntar uma pergunta escrita", |cx| {
    let (_d, st) = installation(&[("a", migrar())]);
    let o = open(cx, &st);
    click(cx, &o, "ask-input-a".into());
    cx.update_window(o.handle, |_, window, cx| window.input("quanto falta?", cx)).unwrap();
    click(cx, &o, "ask-send-a".into());
    assert_eq!(asked(&st, "a"), vec!["quanto falta?".to_string()]);
});

scenario!(agnts_b04_as_perguntas_e_as_respostas, "AGNTS-B04: As perguntas e as respostas", |cx| {
    let json = format!(
        r#"{{"agent":"a","goal":"g","steps":[],"log":[{}],"finished":false,"questions":[{{"id":"q-1","text":"onde?","state":"answered","answer":"a meio"}},{{"id":"q-2","text":"e?","state":"failed","answer":"sem sessão"}}]}}"#,
        entry("start", "g", 0)
    );
    let (_d, st) = installation(&[("a", json)]);
    let o = open(cx, &st);
    let (a, b) = (label(cx, &o, "question-a-q-1".into()), label(cx, &o, "question-a-q-2".into()));
    assert!(a.as_deref().is_some_and(|t| t.contains("a meio")), "{a:?}");
    assert!(b.as_deref().is_some_and(|t| t.contains("sem sessão")), "{b:?}");
});

scenario!(agnts_b05_o_feed_acompanha_as_mudancas, "AGNTS-B05: O feed acompanha as mudanças", |cx| {
    let (_d, st) = installation(&[("a", migrar())]);
    let o = open(cx, &st);
    let heard = Rc::new(RefCell::new(0));
    {
        let heard = heard.clone();
        cx.update(|cx| cx.subscribe(&o.feed, move |_, _: &FeedEvent, _| *heard.borrow_mut() += 1).detach());
    }
    st.ask(LOCAL, "a", "uma nota pela pergunta").unwrap();
    o.feed.update(cx, |f, cx| f.refresh(cx));
    settle(cx, &o);
    let questions = o.feed.read_with(cx, |f, _| f.machines()[0].agents[0].questions.len());
    assert_eq!((*heard.borrow() > 0, questions), (true, 1));
});

scenario!(agnts_b06_a_marca_de_um_agente_numa_sessao, "AGNTS-B06: A marca de um agente numa sessão", |cx| {
    let start = entry("start", "g", 0);
    let mk = |steps: &str, questions: &str, finished: bool| {
        format!(r#"{{"agent":"x","goal":"g","steps":[{steps}],"log":[{start}],"finished":{finished},"questions":[{questions}]}}"#)
    };
    let doing = r#"{"id":"1","title":"t","state":"doing"}"#;
    let blocked = r#"{"id":"1","title":"t","state":"blocked"}"#;
    let pending = r#"{"id":"q-1","text":"?","state":"pending"}"#;
    let jobs = [
        ("session-s1", mk(doing, "", false).replace(r#""agent":"x""#, r#""agent":"session-s1""#)),
        ("session-s2", mk(doing, pending, false).replace(r#""agent":"x""#, r#""agent":"session-s2""#)),
        ("session-s3", mk(blocked, "", false).replace(r#""agent":"x""#, r#""agent":"session-s3""#)),
        ("session-s4", mk(doing, "", true).replace(r#""agent":"x""#, r#""agent":"session-s4""#)),
    ];
    let (_d, st) = installation(&jobs.iter().map(|(a, j)| (*a, j.clone())).collect::<Vec<_>>());
    let o = open(cx, &st);
    let marks = o.feed.read_with(cx, |f, _| f.marks());
    let at = |s: &str| marks.get(&(LOCAL.to_string(), s.to_string())).copied();
    assert_eq!(
        (at("s1"), at("s2"), at("s3"), at("s4")),
        (Some(AgentMark::Working), Some(AgentMark::Waiting), Some(AgentMark::Blocked), Some(AgentMark::Finished))
    );
});

/// Another installation, listening, that lets `me` in, with the job of "a".
fn other_with_a(me: &Station) -> (tempfile::TempDir, Station, String) {
    let dir = tempfile::tempdir().unwrap();
    job(dir.path(), "a", &migrar());
    let other = Station::open(dir.path(), "127.0.0.1:0", "/bin/sh").unwrap();
    std::fs::write(dir.path().join("authorized_keys"), me.status().unwrap().key).unwrap();
    let addr = other.status().unwrap().listen;
    (dir, other, addr)
}

scenario!(agnts_e01_uma_maquina_que_nao_responde, "AGNTS-E01: Uma máquina que não responde", |cx| {
    let (_d, me) = installation(&[("mine", migrar().replace(r#""agent":"a""#, r#""agent":"mine""#))]);
    let (_od, other, addr) = other_with_a(&me);
    me.add(&addr).unwrap();
    drop(other);
    let o = open(cx, &me);
    let why = label(cx, &o, format!("machine-error-{addr}"));
    assert!(why.is_some_and(|w| !w.is_empty()));
    assert!(value(cx, &o, format!("agent-{LOCAL}-mine")).is_some());
});

scenario!(agnts_e02_perguntar_a_um_agente_de_uma_maquina_que_fechou, "AGNTS-E02: Perguntar a um agente de uma máquina que fechou", |cx| {
    let (_d, me) = installation(&[]);
    let (_od, other, addr) = other_with_a(&me);
    me.add(&addr).unwrap();
    let o = open(cx, &me);
    drop(other);
    std::thread::sleep(std::time::Duration::from_millis(200));
    click(cx, &o, format!("ask-{addr}/a-0"));
    let message = label(cx, &o, format!("ask-message-{addr}/a"));
    assert!(message.is_some_and(|m| !m.is_empty()));
});
