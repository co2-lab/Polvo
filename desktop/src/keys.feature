# language: en
# @anchors
#   code: KYFTK
#   ref: KYSEA
#   updated_at: 2026-10-17
#   layer: feature

@KYSEA
Feature: Keys

  @KYSEA-B01 @unit-level
  Scenario: As letras e os algarismos são as teclas deles
    Given as teclas "a", "z", "0" e "9"
    When a janela as traduz
    Then são as teclas A, Z, Digit0 e Digit9

  @KYSEA-B02 @unit-level
  Scenario: As teclas com nome são as teclas delas
    Given as teclas "enter", "backspace", "up", "pagedown", "f1" e "f12"
    When a janela as traduz
    Then são as teclas Enter, Backspace, ArrowUp, PageDown, F1 e F12

  @KYSEA-B03 @unit-level
  Scenario: A pontuação é a tecla física dela
    Given as teclas "-", "/" e "`"
    When a janela as traduz
    Then são as teclas Minus, Slash e Backquote

  @KYSEA-B04 @unit-level
  Scenario: Os modificadores passam
    Given a tecla "c" com Ctrl, Alt e Shift
    When a janela a traduz
    Then os modificadores são Ctrl, Alt e Shift

  @KYSEA-B05 @unit-level
  Scenario: O texto é o carácter sem modificar
    Given a tecla "a" que escreve "á", a tecla "c" com Ctrl e sem carácter, e a tecla "enter"
    When a janela as traduz
    Then os textos são "á", "c" e nenhum

  @KYSEA-B06 @unit-level
  Scenario: Os bytes vêm da libghostty-vt
    Given um terminal novo
    When a janela codifica Enter, Ctrl+C, a seta para cima, e a seta para cima depois de o terminal ligar o modo das setas da aplicação
    Then os bytes são "\r", 0x03, "ESC [ A" e "ESC O A"

  @KYSEA-B07 @unit-level
  Scenario: Uma tecla desconhecida passa o carácter
    Given a tecla "ç" que escreve "ç"
    When a janela a traduz e a codifica
    Then a tecla é a desconhecida e os bytes são "ç"
