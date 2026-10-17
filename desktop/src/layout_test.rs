// @anchors
//   code: DLTDC
//   ref: DCLYD

use super::*;

fn state() -> DockAreaState {
    DockAreaState { version: Some(LAYOUT_VERSION), ..DockAreaState::default() }
}

scenario!(dclyd_b01_o_ficheiro_fica_na_pasta_de_configuracao, "DCLYD-B01: O ficheiro fica na pasta de configuração do sistema", {
    let file = LayoutFile::default().expect("a configuration directory");
    let want = dirs::config_dir().unwrap().join("polvo").join("layout.json");
    assert_eq!(file.path(), want.as_path());
});

scenario!(dclyd_b02_gravar_cria_a_pasta_e_nao_deixa_ficheiros_a_meio, "DCLYD-B02: Gravar cria a pasta e não deixa ficheiros a meio", {
    let dir = tempfile::tempdir().unwrap();
    let file = LayoutFile::at(dir.path().join("novo").join("layout.json"));
    file.write(&state()).unwrap();
    let names: Vec<String> = std::fs::read_dir(dir.path().join("novo"))
        .unwrap()
        .map(|e| e.unwrap().file_name().to_string_lossy().into_owned())
        .collect();
    assert_eq!(names, vec!["layout.json".to_string()]);
});

scenario!(dclyd_b03_ler_devolve_a_organizacao_guardada, "DCLYD-B03: Ler devolve a organização guardada", {
    let dir = tempfile::tempdir().unwrap();
    let file = LayoutFile::at(dir.path().join("layout.json"));
    file.write(&state()).unwrap();
    assert_eq!(file.read(), Some(state()));
    assert_eq!(LayoutFile::at(dir.path().join("vazia").join("layout.json")).read(), None);
});

scenario!(dclyd_e01_um_ficheiro_estragado_ou_de_outra_versao, "DCLYD-E01: Um ficheiro estragado ou de outra versão não dá organização", {
    let dir = tempfile::tempdir().unwrap();
    let broken = dir.path().join("estragado.json");
    std::fs::write(&broken, "isto não é json").unwrap();
    let old = LayoutFile::at(dir.path().join("antigo.json"));
    old.write(&DockAreaState { version: Some(99), ..DockAreaState::default() }).unwrap();
    assert_eq!((LayoutFile::at(broken).read(), old.read()), (None, None));
});

scenario!(dclyd_e02_uma_pasta_que_nao_se_escreve_da_o_erro, "DCLYD-E02: Uma pasta que não se escreve dá o erro", {
    let dir = tempfile::tempdir().unwrap();
    let not_a_dir = dir.path().join("ficheiro");
    std::fs::write(&not_a_dir, "").unwrap();
    assert!(LayoutFile::at(not_a_dir.join("layout.json")).write(&state()).is_err());
});
