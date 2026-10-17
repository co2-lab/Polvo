// @anchors
//   code: KYTSA
//   ref: KYMPE

use super::*;
use std::collections::BTreeMap;

scenario!(kympe_b01_os_atalhos_por_omissao, "KYMPE-B01: Os atalhos por omissão", {
    let k = Keymap::default();
    let got: Vec<&str> = Action::ALL.iter().map(|a| k.shortcut(*a)).collect();
    assert_eq!(got, vec!["ctrl-shift-t", "ctrl-shift-w", "ctrl-shift-c", "ctrl-shift-v", "ctrl-,", "ctrl-s"]);
});

scenario!(kympe_b02_a_forma_unica_de_um_atalho, "KYMPE-B02: A forma única de um atalho", {
    let got: Vec<String> = ["shift-ctrl-T", "CTRL-SHIFT-t", "super-k"].iter().map(|s| normalize(s).unwrap()).collect();
    assert_eq!(got, vec!["ctrl-shift-t", "ctrl-shift-t", "cmd-k"]);
});

scenario!(kympe_b03_a_acao_de_um_atalho, "KYMPE-B03: A ação de um atalho", {
    let k = Keymap::default();
    assert_eq!((k.action_for("shift-ctrl-t"), k.action_for("ctrl-x")), (Some(Action::NewTab), None));
});

scenario!(kympe_b04_mudar_um_atalho, "KYMPE-B04: Mudar um atalho", {
    let mut k = Keymap::default();
    k.set(Action::NewTab, "ctrl-alt-n").unwrap();
    assert_eq!((k.action_for("ctrl-alt-n"), k.action_for("ctrl-shift-t")), (Some(Action::NewTab), None));
});

scenario!(kympe_b05_gravar_e_aplicar_o_que_mudou, "KYMPE-B05: Gravar e aplicar o que mudou", {
    let mut k = Keymap::default();
    k.set(Action::NewTab, "ctrl-alt-n").unwrap();
    let changed = k.overrides();
    let want: BTreeMap<String, String> = [("new-tab".to_string(), "ctrl-alt-n".to_string())].into();
    assert_eq!(changed, want);
    assert_eq!(Keymap::with_overrides(&changed).action_for("ctrl-alt-n"), Some(Action::NewTab));
});

scenario!(kympe_e01_o_atalho_de_outra_acao_e_recusado, "KYMPE-E01: O atalho de outra ação é recusado", {
    let mut k = Keymap::default();
    assert_eq!(k.set(Action::NewTab, "ctrl-shift-w"), Err(KeymapError::Taken(Action::CloseTab)));
    assert_eq!(k.shortcut(Action::NewTab), "ctrl-shift-t");
});

scenario!(kympe_e02_o_que_nao_e_um_atalho_e_recusado, "KYMPE-E02: O que não é um atalho é recusado", {
    assert!(["", "ctrl-shift", "hyper-k"].iter().all(|s| normalize(s).is_err()));
});

scenario!(kympe_e03_um_atalho_gravado_que_nao_serve_e_ignorado, "KYMPE-E03: Um atalho gravado que não serve é ignorado", {
    let saved: BTreeMap<String, String> = [
        ("voar".to_string(), "ctrl-k".to_string()),
        ("new-tab".to_string(), "hyper-k".to_string()),
        ("copy".to_string(), "ctrl-shift-v".to_string()),
    ]
    .into();
    assert_eq!(Keymap::with_overrides(&saved), Keymap::default());
});
