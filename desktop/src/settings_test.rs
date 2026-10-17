// @anchors
//   code: STTSS
//   ref: STTNG

use super::*;

fn changed() -> Settings {
    Settings { keys: [("new-tab".to_string(), "ctrl-alt-n".to_string())].into() }
}

scenario!(sttng_b01_o_ficheiro_fica_na_pasta_de_configuracao, "STTNG-B01: O ficheiro fica na pasta de configuração do sistema", {
    let file = SettingsFile::default().expect("a configuration directory");
    assert_eq!(file.path(), dirs::config_dir().unwrap().join("polvo").join("settings.json").as_path());
});

scenario!(sttng_b02_ler_devolve_o_que_foi_gravado, "STTNG-B02: Ler devolve o que foi gravado", {
    let dir = tempfile::tempdir().unwrap();
    let file = SettingsFile::at(dir.path().join("settings.json"));
    file.write(&changed()).unwrap();
    assert_eq!(file.read(), changed());
    assert_eq!(SettingsFile::at(dir.path().join("vazia").join("settings.json")).read(), Settings::default());
});

scenario!(sttng_b03_gravar_cria_a_pasta_e_nao_deixa_ficheiros_a_meio, "STTNG-B03: Gravar cria a pasta e não deixa ficheiros a meio", {
    let dir = tempfile::tempdir().unwrap();
    SettingsFile::at(dir.path().join("nova").join("settings.json")).write(&changed()).unwrap();
    let names: Vec<String> = std::fs::read_dir(dir.path().join("nova")).unwrap().map(|e| e.unwrap().file_name().to_string_lossy().into_owned()).collect();
    assert_eq!(names, vec!["settings.json".to_string()]);
});

scenario!(sttng_e01_um_ficheiro_estragado_nao_muda_nada, "STTNG-E01: Um ficheiro estragado não muda nada", {
    let dir = tempfile::tempdir().unwrap();
    let (a, b) = (dir.path().join("a.json"), dir.path().join("b.json"));
    std::fs::write(&a, "isto não é json").unwrap();
    std::fs::write(&b, r#"{"keys": ["ctrl-k"]}"#).unwrap();
    assert_eq!((SettingsFile::at(a).read(), SettingsFile::at(b).read()), (Settings::default(), Settings::default()));
});

scenario!(sttng_e02_uma_pasta_que_nao_se_escreve_da_o_erro, "STTNG-E02: Uma pasta que não se escreve dá o erro", {
    let dir = tempfile::tempdir().unwrap();
    let not_a_dir = dir.path().join("ficheiro");
    std::fs::write(&not_a_dir, "").unwrap();
    assert!(SettingsFile::at(not_a_dir.join("settings.json")).write(&changed()).is_err());
});
