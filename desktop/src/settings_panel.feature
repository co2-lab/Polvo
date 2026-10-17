# language: en
# @anchors
#   code: SPFST
#   ref: STPNS
#   updated_at: 2026-10-17
#   layer: feature

@STPNS
Feature: Settings panel

  @STPNS-B01 @unit-level
  Scenario: As secções, com os atalhos abertos
    Given a aba da configuração aberta
    When é desenhada
    Then a secção dos atalhos está aberta

  @STPNS-B02 @unit-level
  Scenario: Cada ação com o atalho dela
    Given a aba da configuração aberta, com o mapa de omissão
    When é desenhada
    Then cada ação mostra o atalho de omissão

  @STPNS-B03 @unit-level
  Scenario: Mudar um atalho
    Given a aba da configuração com o ficheiro numa pasta vazia
    When o utilizador carrega no atalho da aba nova e prime Ctrl+Alt+N
    Then o mapa abre a aba nova com "ctrl-alt-n", a aba mostra "ctrl-alt-n", e o ficheiro tem-no

  @STPNS-B04 @unit-level
  Scenario: Cancelar, e os modificadores sozinhos
    Given a aba da configuração
    When o utilizador carrega no atalho da aba nova, prime só o Shift, e depois o Esc
    Then o atalho da aba nova continua "ctrl-shift-t"

  @STPNS-E01 @unit-level
  Scenario: O atalho de outra ação é recusado
    Given a aba da configuração
    When o utilizador carrega no atalho da aba nova e prime Ctrl+Shift+W
    Then a mensagem diz "Fechar a aba", e a aba nova continua "ctrl-shift-t"

  @STPNS-E02 @unit-level
  Scenario: Uma configuração que não se grava avisa
    Given a aba da configuração com o ficheiro dentro de um caminho que é um ficheiro
    When o utilizador muda o atalho da aba nova para Ctrl+Alt+N
    Then a mensagem diz que não gravou, e o mapa abre a aba nova com "ctrl-alt-n"
