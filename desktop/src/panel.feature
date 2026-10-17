# language: en
# @anchors
#   code: TPFTR
#   ref: TRPNT
#   updated_at: 2026-10-17
#   layer: feature

@TRPNT
Feature: Panel

  @TRPNT-B01 @unit-level
  Scenario: O teclado vai para a shell
    Given um painel com um terminal
    When o utilizador escreve "a", carrega em Enter e em Ctrl+C
    Then a shell recebe "a", "\r" e 0x03

  @TRPNT-B02#01 @unit-level
  Scenario: Ctrl+Shift+C copia a seleção e Ctrl+Shift+V cola
    Given um painel com "abc def" no terminal e "def" selecionado
    When o utilizador carrega em Ctrl+Shift+C, e depois em Ctrl+Shift+V
    Then a área de transferência tem "def", a shell recebe "def", e não recebe nenhum dos atalhos

  @TRPNT-B02#02 @unit-level
  Scenario: Selecionar copia para a seleção primária e o botão do meio cola
    Given um painel com "abc def" no terminal
    When o utilizador arrasta o rato sobre "abc" e depois carrega no botão do meio
    Then a seleção primária tem "abc" e a shell recebe "abc"

  @TRPNT-B03 @unit-level
  Scenario: O rato seleciona e a roda faz scroll
    Given um painel com 100 linhas escritas no terminal
    When o utilizador sobe a roda um passo, e depois arrasta sobre a primeira linha à vista
    Then a vista subiu três linhas, e a seleção é o texto dessa linha

  @TRPNT-B04 @unit-level
  Scenario: O terminal enche o painel
    Given um painel de 800 por 600 píxeis
    When o painel é desenhado
    Then as linhas e as colunas do terminal são o que cabe da área do painel em células da fonte

  @TRPNT-B05 @unit-level
  Scenario: A fonte segue as preferências
    Given as preferências da fonte "Fira Code" de 15 píxeis
    When o painel abre
    Then o terminal usa a "Fira Code" de 15 píxeis

  @TRPNT-B06 @unit-level
  Scenario: A aba diz o título da shell
    Given um painel da shell "/bin/bash"
    When a shell ainda não deu título, e depois dá o título "~/projeto"
    Then a aba diz "bash", e depois "~/projeto"

  @TRPNT-B07 @unit-level
  Scenario: O fim da shell avisa
    Given um painel com uma shell
    When chega o fim da shell
    Then o painel avisa que a shell terminou

  @TRPNT-B08 @unit-level
  Scenario: Clicar dá o foco
    Given dois painéis lado a lado, com o foco no primeiro
    When o utilizador carrega no segundo e escreve "x"
    Then só a shell do segundo recebe "x"

  @TRPNT-E01 @unit-level
  Scenario: Uma shell que não arranca não faz painel
    Given o SHELL "/nao/existe/shell"
    When a janela pede um terminal
    Then o pedido dá o erro com "/nao/existe/shell"
