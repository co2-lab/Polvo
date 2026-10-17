# language: en
# @anchors
#   code: WNFTW
#   ref: WNDWI
#   updated_at: 2026-10-17
#   layer: feature

@WNDWI
Feature: Window

  @WNDWI-B01#01 @unit-level
  Scenario: Os botões à esquerda pela ordem do sistema
    Given as preferências com os botões fechar, minimizar e maximizar à esquerda
    When a janela abre
    Then fechar, minimizar e maximizar estão por esta ordem à esquerda do título "Polvo"

  @WNDWI-B01#02 @unit-level
  Scenario: Os botões à direita pela ordem do sistema
    Given as preferências com os botões minimizar, maximizar e fechar à direita
    When a janela abre
    Then minimizar, maximizar e fechar estão por esta ordem à direita do título, que fica ao centro

  @WNDWI-B02 @unit-level
  Scenario: Fechar fecha a janela
    Given uma janela aberta
    When o utilizador carrega em fechar
    Then a janela deixa de existir

  @WNDWI-B03 @unit-level
  Scenario: O teclado vai para a shell
    Given uma janela com um terminal
    When o utilizador escreve "a", carrega em Enter e em Ctrl+C
    Then a shell recebe "a", "\r" e 0x03

  @WNDWI-B04#01 @unit-level
  Scenario: Ctrl+Shift+C copia a seleção e Ctrl+Shift+V cola
    Given uma janela com "abc def" no terminal e "def" selecionado
    When o utilizador carrega em Ctrl+Shift+C, e depois em Ctrl+Shift+V
    Then a área de transferência tem "def", a shell recebe "def", e não recebe nenhum dos atalhos

  @WNDWI-B04#02 @unit-level
  Scenario: Selecionar copia para a seleção primária e o botão do meio cola
    Given uma janela com "abc def" no terminal
    When o utilizador arrasta o rato sobre "abc" e depois carrega no botão do meio
    Then a seleção primária tem "abc" e a shell recebe "abc"

  @WNDWI-B05 @unit-level
  Scenario: O rato seleciona e a roda faz scroll
    Given uma janela com 100 linhas escritas no terminal
    When o utilizador sobe a roda um passo, e depois arrasta sobre a primeira linha à vista
    Then a vista subiu três linhas, e a seleção é o texto dessa linha

  @WNDWI-B06 @unit-level
  Scenario: O terminal enche a janela
    Given uma janela de 800 por 600 píxeis
    When a janela é desenhada
    Then as linhas e as colunas do terminal são o que cabe da área do terminal em células da fonte

  @WNDWI-B07 @unit-level
  Scenario: O tema e a fonte seguem as preferências
    Given as preferências do tema claro e da fonte "Fira Code" de 15 píxeis
    When a janela abre
    Then a barra é clara e o terminal usa a "Fira Code" de 15 píxeis

  @WNDWI-B08 @unit-level
  Scenario: O fim da shell fecha a janela
    Given uma janela com uma shell que termina logo
    When chega o fim da shell
    Then a janela deixa de existir
