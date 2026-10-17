# language: en
# @anchors
#   code: KYFTA
#   ref: KYMPE
#   updated_at: 2026-10-17
#   layer: feature

@KYMPE
Feature: Keymap

  @KYMPE-B01 @unit-level
  Scenario: Os atalhos por omissão
    Given um mapa sem nada mudado
    When se pede o atalho de cada ação
    Then aba nova é "ctrl-shift-t", fechar aba "ctrl-shift-w", copiar "ctrl-shift-c", colar "ctrl-shift-v" e configuração "ctrl-,"

  @KYMPE-B02 @unit-level
  Scenario: A forma única de um atalho
    Given os atalhos "shift-ctrl-T", "CTRL-SHIFT-t" e "super-k"
    When se pede a forma única de cada um
    Then são "ctrl-shift-t", "ctrl-shift-t" e "cmd-k"

  @KYMPE-B03 @unit-level
  Scenario: A ação de um atalho
    Given um mapa sem nada mudado
    When se pede a ação de "shift-ctrl-t" e de "ctrl-x"
    Then a primeira é aba nova e a segunda nenhuma

  @KYMPE-B04 @unit-level
  Scenario: Mudar um atalho
    Given um mapa sem nada mudado
    When a aba nova passa a "ctrl-alt-n"
    Then "ctrl-alt-n" abre uma aba nova e "ctrl-shift-t" já não faz nada

  @KYMPE-B05 @unit-level
  Scenario: Gravar e aplicar o que mudou
    Given um mapa onde a aba nova passou a "ctrl-alt-n"
    When se pede o que mudou e se aplica a um mapa novo
    Then o que mudou é só "new-tab" com "ctrl-alt-n", e o mapa novo abre a aba nova com "ctrl-alt-n"

  @KYMPE-E01 @unit-level
  Scenario: O atalho de outra ação é recusado
    Given um mapa sem nada mudado
    When a aba nova passa a "ctrl-shift-w"
    Then é recusado por ser de fechar aba, e a aba nova continua em "ctrl-shift-t"

  @KYMPE-E02 @unit-level
  Scenario: O que não é um atalho é recusado
    Given os atalhos "", "ctrl-shift" e "hyper-k"
    When se pede a forma única de cada um
    Then os três são recusados

  @KYMPE-E03 @unit-level
  Scenario: Um atalho gravado que não serve é ignorado
    Given o que mudou com uma ação "voar", a aba nova com "hyper-k", e copiar com "ctrl-shift-v"
    When se aplica a um mapa novo
    Then os atalhos ficam os de omissão
