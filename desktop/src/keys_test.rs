// @anchors
//   code: KYTSK
//   ref: KYSEA

use super::*;
use libghostty_vt::{Terminal, TerminalOptions};

fn plain(key: &str) -> Input {
    translate(key, None, Modifiers::default())
}

fn bytes(input: &Input, term: &Terminal<'static, 'static>) -> Vec<u8> {
    let mut enc = libghostty_vt::key::Encoder::new().unwrap();
    encode(input, &mut enc, term)
}

fn terminal() -> Terminal<'static, 'static> {
    Terminal::new(TerminalOptions { cols: 80, rows: 24, max_scrollback: 0 }).unwrap()
}

scenario!(kysea_b01_as_letras_e_os_algarismos_sao_as_teclas_deles, "KYSEA-B01: As letras e os algarismos são as teclas deles", {
    let keys: Vec<Key> = ["a", "z", "0", "9"].iter().map(|k| plain(k).key).collect();
    assert_eq!(keys, vec![Key::A, Key::Z, Key::Digit0, Key::Digit9]);
});

scenario!(kysea_b02_as_teclas_com_nome_sao_as_teclas_delas, "KYSEA-B02: As teclas com nome são as teclas delas", {
    let keys: Vec<Key> = ["enter", "backspace", "up", "pagedown", "f1", "f12"].iter().map(|k| plain(k).key).collect();
    assert_eq!(keys, vec![Key::Enter, Key::Backspace, Key::ArrowUp, Key::PageDown, Key::F1, Key::F12]);
});

scenario!(kysea_b03_a_pontuacao_e_a_tecla_fisica_dela, "KYSEA-B03: A pontuação é a tecla física dela", {
    let keys: Vec<Key> = ["-", "/", "`"].iter().map(|k| plain(k).key).collect();
    assert_eq!(keys, vec![Key::Minus, Key::Slash, Key::Backquote]);
});

scenario!(kysea_b04_os_modificadores_passam, "KYSEA-B04: Os modificadores passam", {
    let i = translate("c", None, Modifiers { control: true, alt: true, shift: true });
    assert_eq!(i.mods, Mods::CTRL | Mods::ALT | Mods::SHIFT);
});

scenario!(kysea_b05_o_texto_e_o_caracter_sem_modificar, "KYSEA-B05: O texto é o carácter sem modificar", {
    let a = translate("a", Some("á"), Modifiers::default());
    let c = translate("c", None, Modifiers { control: true, ..Default::default() });
    let enter = translate("enter", Some("\r"), Modifiers::default());
    assert_eq!((a.text.as_deref(), c.text.as_deref(), enter.text.as_deref()), (Some("á"), Some("c"), None));
});

scenario!(kysea_b06_os_bytes_vem_da_libghostty_vt, "KYSEA-B06: Os bytes vêm da libghostty-vt", {
    let mut term = terminal();
    let enter = bytes(&translate("enter", Some("\r"), Modifiers::default()), &term);
    let ctrl_c = bytes(&translate("c", None, Modifiers { control: true, ..Default::default() }), &term);
    let up = bytes(&plain("up"), &term);
    term.vt_write(b"\x1b[?1h");
    let up_app = bytes(&plain("up"), &term);
    assert_eq!(
        (enter, ctrl_c, up, up_app),
        (b"\r".to_vec(), vec![0x03], b"\x1b[A".to_vec(), b"\x1bOA".to_vec())
    );
});

scenario!(kysea_b07_uma_tecla_desconhecida_passa_o_caracter, "KYSEA-B07: Uma tecla desconhecida passa o carácter", {
    let i = translate("ç", Some("ç"), Modifiers::default());
    assert_eq!(i.key, Key::Unidentified);
    assert_eq!(bytes(&i, &terminal()), "ç".as_bytes().to_vec());
});
