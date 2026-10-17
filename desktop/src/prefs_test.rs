// @anchors
//   code: PRTSP
//   ref: PRFSE

use super::*;

fn prefs(keys: &[(&str, &str)], font: Option<&str>) -> Prefs {
    read_with(|key| keys.iter().find(|(k, _)| *k == key).map(|(_, v)| v.to_string()), font)
}

scenario!(prfse_b01_os_botoes_seguem_a_disposicao_do_sistema, "PRFSE-B01#01: Os botões seguem a disposição do sistema", {
    let p = prefs(&[("button-layout", "close,minimize,maximize:appmenu")], None);
    assert_eq!(p.left, vec![Button::Close, Button::Minimize, Button::Maximize]);
    assert_eq!(p.right, vec![]);
});

scenario!(prfse_b01_uma_disposicao_sem_dois_pontos_poe_os_botoes_a_direita, "PRFSE-B01#02: Uma disposição sem dois pontos põe os botões à direita", {
    let p = prefs(&[("button-layout", "minimize,close")], None);
    assert_eq!(p.left, vec![]);
    assert_eq!(p.right, vec![Button::Minimize, Button::Close]);
});

scenario!(prfse_b02_o_tema_segue_o_esquema_de_cores, "PRFSE-B02: O tema segue o esquema de cores", {
    let dark = |scheme| prefs(&[("color-scheme", scheme)], None).dark;
    assert_eq!((dark("prefer-dark"), dark("prefer-light"), dark("default")), (true, false, false));
});

scenario!(prfse_b03_a_fonte_segue_a_fonte_monoespacada_do_sistema, "PRFSE-B03#01: A fonte segue a fonte monoespaçada do sistema", {
    let p = prefs(&[("monospace-font-name", "Ubuntu Sans Mono 11")], None);
    assert_eq!((p.font.as_str(), p.font_px), ("Ubuntu Sans Mono", 15.));
});

scenario!(prfse_b03_a_variavel_polvo_font_substitui_o_nome_da_fonte, "PRFSE-B03#02: A variável POLVO_FONT substitui o nome da fonte", {
    let p = prefs(&[("monospace-font-name", "Ubuntu Sans Mono 11")], Some("Fira Code"));
    assert_eq!((p.font.as_str(), p.font_px), ("Fira Code", 15.));
});

scenario!(prfse_b04_sem_preferencias_os_valores_por_omissao, "PRFSE-B04: Sem preferências, os valores por omissão", {
    let p = prefs(&[], None);
    assert_eq!(p.left, vec![]);
    assert_eq!(p.right, vec![Button::Minimize, Button::Maximize, Button::Close]);
    assert!(p.dark);
    assert_eq!((p.font.as_str(), p.font_px), ("DejaVu Sans Mono", 15.));
});

scenario!(prfse_e01_uma_fonte_sem_tamanho_valido_da_a_fonte_por_omissao, "PRFSE-E01: Uma fonte sem tamanho válido dá a fonte por omissão", {
    for bad in ["Ubuntu Sans Mono", "Ubuntu Sans Mono onze", "Ubuntu Sans Mono 0"] {
        let p = prefs(&[("monospace-font-name", bad)], None);
        assert_eq!((p.font.as_str(), p.font_px), ("DejaVu Sans Mono", 15.), "{bad}");
    }});
