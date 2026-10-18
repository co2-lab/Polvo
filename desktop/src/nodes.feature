# language: en
# @anchors
#   code: MQFST
#   ref: MQNAS
#   updated_at: 2026-10-17
#   layer: feature

@MQNAS
Feature: Nodes

  @MQNAS-B01 @unit-level
  Scenario: As máquinas e as sessões delas
    Given uma instalação com uma sessão local, e outra máquina juntada com uma sessão
    When a aba se mostra
    Then mostra esta máquina com a chave e a sua sessão, e depois a outra com a sua

  @MQNAS-B02 @unit-level
  Scenario: Atualizar
    Given a aba de uma instalação sem sessões
    When nasce uma sessão local e o utilizador carrega em atualizar
    Then a aba mostra a sessão

  @MQNAS-B03 @unit-level
  Scenario: Abrir uma sessão
    Given a aba com uma sessão local
    When o utilizador carrega nela
    Then a aba pede um terminal ligado a ela, sem ser o dono, que escreve nela

  @MQNAS-B04 @unit-level
  Scenario: Uma sessão nova
    Given a aba de uma instalação sem sessões
    When o utilizador carrega na sessão nova desta máquina
    Then esta máquina tem uma sessão, e a aba pede um terminal dono dela

  @MQNAS-B05 @unit-level
  Scenario: Travar as sessões desta máquina
    Given a aba com uma sessão local
    When o utilizador a trava
    Then a sessão está travada, e a aba mostra-o

  @MQNAS-B06 @unit-level
  Scenario: Juntar e esquecer uma máquina
    Given a aba e outra instalação que a autoriza
    When o utilizador escreve o endereço dela e junta-a, e depois esquece-a
    Then a aba mostra a outra máquina, e no fim só esta

  @MQNAS-E01 @unit-level
  Scenario: Uma máquina que não responde
    Given a aba com outra máquina juntada que entretanto fechou
    When a aba atualiza
    Then a outra máquina diz porquê, e esta máquina continua com as suas sessões

  @MQNAS-E02 @unit-level
  Scenario: Juntar uma máquina que não responde
    Given a aba
    When o utilizador junta um endereço onde ninguém escuta
    Then a mensagem diz porquê, e só há esta máquina
