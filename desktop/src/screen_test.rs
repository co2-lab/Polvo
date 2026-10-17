// @anchors
//   code: SCTSS
//   ref: SCRNE

use super::*;
use crate::keys::{Modifiers, translate}; // @dep: KYIMK
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

/// What the terminal writes to the shell, kept for the test to read.
#[derive(Clone, Default)]
struct Sink(Arc<Mutex<Vec<u8>>>);

impl std::io::Write for Sink {
    fn write(&mut self, b: &[u8]) -> std::io::Result<usize> {
        self.0.lock().unwrap().extend_from_slice(b);
        Ok(b.len())
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

impl Sink {
    fn take(&self) -> Vec<u8> {
        std::mem::take(&mut self.0.lock().unwrap())
    }
}

const CELL_W: f64 = 10.;
const CELL_H: f64 = 20.;

fn size(rows: u16, cols: u16) -> Size {
    Size { rows, cols, cell_w: CELL_W as u32, cell_h: CELL_H as u32 }
}

fn offline(rows: u16, cols: u16) -> (Screen, Sink) {
    let sink = Sink::default();
    (Screen::new(size(rows, cols), Box::new(sink.clone())).unwrap(), sink)
}

fn sh(script: &str) -> Vec<String> {
    vec!["/bin/sh".into(), "-c".into(), script.into()]
}

/// Feeds the shell's output until the screen shows `want`, for up to ten seconds.
fn wait_for(screen: &mut Screen, rx: &mut Events, want: &str) -> bool {
    let end = Instant::now() + Duration::from_secs(10);
    while Instant::now() < end {
        if screen.text().contains(want) {
            return true;
        }
        match rx.try_next() {
            Ok(Some(Event::Output(b))) => screen.feed(&b),
            Ok(Some(Event::Exit)) | Ok(None) => return screen.text().contains(want),
            Err(_) => std::thread::sleep(Duration::from_millis(10)),
        }
    }
    false
}

/// The middle of a cell, in pixels from the grid's corner.
fn at(col: u16) -> (f64, f64) {
    (f64::from(col) * CELL_W + CELL_W / 2., CELL_H / 2.)
}

fn click(s: &mut Screen, col: u16, ms: u64) {
    let (x, y) = at(col);
    s.press(x, y, Duration::from_millis(ms));
    s.release(x, y);
}

fn run_with<'f>(frame: &'f Frame, text: &str) -> &'f Run {
    frame.lines[0].iter().find(|r| r.text == text).unwrap_or_else(|| panic!("no run {text:?} in {:?}", frame.lines[0]))
}

scenario!(scrne_b01_a_shell_corre_num_pty_do_tamanho_do_terminal, "SCRNE-B01: A shell corre num pty do tamanho do terminal", {
    let (mut s, mut rx) = Screen::spawn(&sh("echo $TERM $COLORTERM; stty size; sleep 1"), None, size(24, 80)).unwrap();
    assert!(wait_for(&mut s, &mut rx, "24 80"), "{}", s.text());
    assert!(s.text().contains("xterm-256color truecolor"), "{}", s.text());
});

scenario!(scrne_b02_a_saida_da_shell_e_o_ecra, "SCRNE-B02: A saída da shell é o ecrã", {
    let (mut s, _) = offline(5, 20);
    s.feed(b"ola\r\n\x1b[31mmundo\x1b[0m\rM\r\n");
    let text = s.text();
    let lines: Vec<&str> = text.lines().collect();
    assert_eq!(&lines[..2], &["ola", "Mundo"]);
    let frame = s.frame();
    let undo = frame.lines[1].iter().find(|r| r.text.starts_with("undo")).expect("the red run");
    assert_eq!(undo.fg, frame.palette[1]);
});

scenario!(scrne_b03_as_respostas_do_terminal_voltam_a_shell, "SCRNE-B03: As respostas do terminal voltam à shell", {
    let (mut s, sink) = offline(5, 20);
    s.feed(b"ab\x1b[6n");
    assert_eq!(sink.take(), b"\x1b[1;3R".to_vec());
});

scenario!(scrne_b04_o_tamanho_muda_o_terminal_e_o_pty, "SCRNE-B04: O tamanho muda o terminal e o pty", {
    let (mut s, mut rx) = Screen::spawn(&sh("read x; stty size; printf '%050d\\n' 0; sleep 1"), None, size(24, 80)).unwrap();
    s.resize(size(10, 40));
    s.key(&translate("enter", Some("\r"), Modifiers::default()));
    assert!(wait_for(&mut s, &mut rx, "10 40"), "{}", s.text());
    assert!(wait_for(&mut s, &mut rx, &"0".repeat(40)), "{}", s.text());
    let text = s.text();
    assert!(text.lines().any(|l| l == "0".repeat(40)) && text.lines().any(|l| l == "0".repeat(10)), "{text}");
});

scenario!(scrne_b05_o_scroll_mostra_o_historico_e_escrever_volta_ao_fim, "SCRNE-B05: O scroll mostra o histórico, e escrever volta ao fim", {
    let (mut s, _) = offline(5, 20);
    let lines: Vec<String> = (1..=20).map(|n| n.to_string()).collect();
    s.feed(lines.join("\r\n").as_bytes());
    s.scroll(-10);
    assert_eq!(s.text().lines().collect::<Vec<_>>(), vec!["6", "7", "8", "9", "10"]);
    s.key(&translate("a", Some("a"), Modifiers::default()));
    assert_eq!(s.text().lines().collect::<Vec<_>>(), vec!["16", "17", "18", "19", "20"]);
});

scenario!(scrne_b06_01_carregar_e_arrastar_seleciona, "SCRNE-B06#01: Carregar e arrastar seleciona de uma célula à outra", {
    let (mut s, _) = offline(5, 20);
    s.feed(b"abc def ghi");
    // Ghostty's rule: a cell is in once the pointer is past its middle.
    let (x0, y) = (1. * CELL_W + 1., CELL_H / 2.);
    let x1 = 5. * CELL_W + CELL_W - 1.;
    s.press(x0, y, Duration::ZERO);
    s.drag(x1, y);
    s.release(x1, y);
    assert_eq!(s.selection_text().as_deref(), Some("bc de"));
});

scenario!(scrne_b06_02_dois_cliques_a_palavra_e_tres_a_linha, "SCRNE-B06#02: Dois cliques selecionam a palavra e três a linha", {
    let (mut s, _) = offline(5, 20);
    s.feed(b"abc def ghi");
    click(&mut s, 5, 0);
    click(&mut s, 5, 100);
    assert_eq!(s.selection_text().as_deref(), Some("def"));
    click(&mut s, 5, 200);
    assert_eq!(s.selection_text().as_deref(), Some("abc def ghi"));
});

scenario!(scrne_b07_colar_respeita_o_modo_do_terminal, "SCRNE-B07: Colar respeita o modo do terminal", {
    let (mut s, sink) = offline(5, 20);
    s.paste("a\nb");
    assert_eq!(sink.take(), b"a\rb".to_vec());
    s.feed(b"\x1b[?2004h");
    s.paste("a\nb");
    assert_eq!(sink.take(), b"\x1b[200~a\nb\x1b[201~".to_vec());
});

scenario!(scrne_b08_o_desenho_mostra_o_cursor_e_a_selecao, "SCRNE-B08: O desenho mostra o cursor e a seleção", {
    let (mut s, _) = offline(5, 20);
    s.feed(b"abc");
    let y = CELL_H / 2.;
    s.press(CELL_W + 1., y, Duration::ZERO);
    s.drag(2. * CELL_W - 1., y);
    s.release(2. * CELL_W - 1., y);
    let frame = s.frame();
    let (a, b) = (run_with(&frame, "a"), run_with(&frame, "b"));
    assert_eq!((a.fg, a.bg), (frame.fg, frame.bg));
    assert_eq!((b.fg, b.bg), (frame.bg, frame.fg));
    let cursor = &frame.lines[0][3];
    assert_eq!((cursor.text.as_str(), cursor.fg, cursor.bg), (" ", frame.bg, frame.fg));
});

scenario!(scrne_b09_o_fim_da_shell_chega_pelo_canal, "SCRNE-B09: O fim da shell chega pelo canal", {
    let (_s, mut rx) = Screen::spawn(&sh("echo adeus"), None, size(5, 20)).unwrap();
    let mut out = Vec::new();
    let end = Instant::now() + Duration::from_secs(10);
    let mut exited = false;
    while Instant::now() < end && !exited {
        match rx.try_next() {
            Ok(Some(Event::Output(b))) => out.extend(b),
            Ok(Some(Event::Exit)) => exited = true,
            Ok(None) => break,
            Err(_) => std::thread::sleep(Duration::from_millis(10)),
        }
    }
    assert!(String::from_utf8_lossy(&out).contains("adeus"));
    assert!(exited);
});

scenario!(scrne_e01_uma_shell_que_nao_arranca_da_o_erro, "SCRNE-E01: Uma shell que não arranca dá o erro", {
    let err = Screen::spawn(&["/nao/existe/shell".to_string()], None, size(5, 20)).err().expect("an error");
    assert!(err.to_string().contains("/nao/existe/shell"), "{err}");
});
