// @anchors
//   code: RMTST
//   ref: RMTEC

use super::*;
use std::time::{Duration, Instant};

fn open() -> (tempfile::TempDir, Station) {
    let dir = tempfile::tempdir().unwrap();
    let st = Station::open(dir.path(), "", "/bin/sh").unwrap();
    (dir, st)
}

/// Reads the terminal until it has `want`, or a while passes.
fn read_for(term: &RemoteTerm, want: &str) -> String {
    let mut got = Vec::new();
    let mut buf = [0u8; 4096];
    let end = Instant::now() + Duration::from_secs(10);
    while !String::from_utf8_lossy(&got).contains(want) && Instant::now() < end {
        match term.read(&mut buf) {
            Ok(0) | Err(_) => break,
            Ok(n) => got.extend_from_slice(&buf[..n]),
        }
    }
    String::from_utf8_lossy(&got).into_owned()
}

scenario!(rmtec_b01_abrir_a_instalacao, "RMTEC-B01: Abrir a instalação", {
    let (_dir, st) = open();
    let names: Vec<String> = st.machines().unwrap().into_iter().map(|m| m.name).collect();
    assert_eq!((names, st.status().unwrap().listen), (vec![LOCAL.to_string()], String::new()));
});

scenario!(rmtec_b02_as_sessoes_de_uma_maquina, "RMTEC-B02: As sessões de uma máquina", {
    let (_dir, st) = open();
    let id = st.new_session(LOCAL, 24, 80).unwrap();
    st.lock(LOCAL, &id, true).unwrap();
    let list = st.sessions(LOCAL).unwrap();
    st.close_session(LOCAL, &id).unwrap();
    let end = Instant::now() + Duration::from_secs(5);
    let mut after = st.sessions(LOCAL).unwrap();
    while !after.is_empty() && Instant::now() < end {
        std::thread::sleep(Duration::from_millis(20));
        after = st.sessions(LOCAL).unwrap();
    }
    assert_eq!(list.len(), 1);
    assert_eq!((list[0].id.as_str(), list[0].locked, list[0].writable, after.len()), (id.as_str(), true, true, 0));
});

scenario!(rmtec_b03_um_terminal_de_uma_sessao, "RMTEC-B03: Um terminal de uma sessão", {
    let (_dir, st) = open();
    let id = st.new_session(LOCAL, 24, 80).unwrap();
    let term = st.attach(LOCAL, &id).unwrap();
    // a mark only the result has: the echo of the line and the prompt do not make it
    term.write(b"echo marca-$((40+2))\n").unwrap();
    let got = read_for(&term, "marca-42");
    term.resize(30, 100).unwrap();
    drop(term);
    assert!(got.contains("marca-42"), "{got:?}");
    assert_eq!(st.sessions(LOCAL).unwrap().len(), 1);
});

scenario!(rmtec_b04_juntar_e_esquecer_uma_maquina, "RMTEC-B04: Juntar e esquecer uma máquina", {
    let (_dir, first) = open();
    let other_dir = tempfile::tempdir().unwrap();
    let other = Station::open(other_dir.path(), "127.0.0.1:0", "/bin/sh").unwrap();
    std::fs::write(other_dir.path().join("authorized_keys"), first.status().unwrap().key).unwrap();
    let id = other.new_session(LOCAL, 24, 80).unwrap();
    let addr = other.status().unwrap().listen;
    first.add(&addr).unwrap();
    let names: Vec<String> = first.machines().unwrap().into_iter().map(|m| m.name).collect();
    let theirs: Vec<String> = first.sessions(&addr).unwrap().into_iter().map(|s| s.id).collect();
    first.remove(&addr).unwrap();
    let after: Vec<String> = first.machines().unwrap().into_iter().map(|m| m.name).collect();
    assert_eq!((names, theirs, after), (vec![LOCAL.to_string(), addr.clone()], vec![id], vec![LOCAL.to_string()]));
});

scenario!(rmtec_e01_o_erro_da_biblioteca, "RMTEC-E01: O erro da biblioteca", {
    let (_dir, st) = open();
    let err = st.sessions("nenhuma").unwrap_err();
    assert!(err.0.contains("nenhuma"), "{err:?}");
});

scenario!(rmtec_b05_o_progresso_dos_agentes, "RMTEC-B05: O progresso dos agentes", {
    let dir = tempfile::tempdir().unwrap();
    std::fs::create_dir_all(dir.path().join("progress")).unwrap();
    let job = r#"{"agent":"a","goal":"migrar","steps":[{"id":"1","title":"ler","state":"doing"}],"log":[{"at":"2026-10-18T10:00:00Z","kind":"start","text":"migrar"}],"finished":false}"#;
    std::fs::write(dir.path().join("progress").join("a.json"), job).unwrap();
    let st = Station::open(dir.path(), "", "/bin/sh").unwrap();
    let all = st.progress_all(&std::collections::BTreeMap::new(), 1000).unwrap();
    let q = st.ask(LOCAL, "a", "onde vais?").unwrap();
    assert_eq!(all.len(), 1);
    let agent = &all[0].agents[0];
    assert_eq!((all[0].name.as_str(), agent.agent.as_str(), agent.goal.as_str()), (LOCAL, "a", "migrar"));
    assert_eq!((agent.steps[0].title.as_str(), agent.steps[0].state.as_str(), agent.log.len()), ("ler", "doing", 1));
    assert!(!q.is_empty());
});
