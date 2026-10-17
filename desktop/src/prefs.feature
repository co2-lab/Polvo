# language: en
# @anchors
#   code: PRFTP
#   ref: PRFSE
#   updated_at: 2026-10-17
#   layer: feature

@PRFSE
Feature: Prefs

  @PRFSE-B01#01 @unit-level
  Scenario: Os botões seguem a disposição do sistema
    Given a disposição "close,minimize,maximize:appmenu"
    When a janela lê as preferências
    Then à esquerda estão fechar, minimizar e maximizar, por esta ordem, e à direita nenhum

  @PRFSE-B01#02 @unit-level
  Scenario: Uma disposição sem dois pontos põe os botões à direita
    Given a disposição "minimize,close"
    When a janela lê as preferências
    Then à esquerda não há botões, e à direita estão minimizar e fechar

  @PRFSE-B02 @unit-level
  Scenario: O tema segue o esquema de cores
    Given os esquemas "prefer-dark", "prefer-light" e "default"
    When a janela lê as preferências de cada um
    Then o primeiro dá o tema escuro e os outros dois o claro

  @PRFSE-B03#01 @unit-level
  Scenario: A fonte segue a fonte monoespaçada do sistema
    Given a fonte monoespaçada "Ubuntu Sans Mono 11"
    When a janela lê as preferências
    Then a fonte é "Ubuntu Sans Mono" com 15 píxeis

  @PRFSE-B03#02 @unit-level
  Scenario: A variável POLVO_FONT substitui o nome da fonte
    Given a fonte monoespaçada "Ubuntu Sans Mono 11" e o POLVO_FONT "Fira Code"
    When a janela lê as preferências
    Then a fonte é "Fira Code" com 15 píxeis

  @PRFSE-B04 @unit-level
  Scenario: Sem preferências, os valores por omissão
    Given um sistema que não dá nenhuma das chaves
    When a janela lê as preferências
    Then os botões são minimizar, maximizar e fechar à direita, o tema é escuro e a fonte é "DejaVu Sans Mono" com 15 píxeis

  @PRFSE-E01 @unit-level
  Scenario: Uma fonte sem tamanho válido dá a fonte por omissão
    Given as fontes monoespaçadas "Ubuntu Sans Mono", "Ubuntu Sans Mono onze" e "Ubuntu Sans Mono 0"
    When a janela lê as preferências de cada uma
    Then cada uma dá a fonte "DejaVu Sans Mono" com 15 píxeis
