# language: en
# @anchors
#   code: MQFST
#   ref: MQNAS
#   updated_at: 2026-10-18
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

  @MQNAS-B07 @unit-level
  Scenario: As máquinas da rede
    Given a aba, e a procura deu uma máquina desconhecida e uma conhecida que não está nas máquinas
    When a aba se mostra
    Then a primeira tem o botão de emparelhar, e a segunda o de juntar

  @MQNAS-B08 @unit-level
  Scenario: Emparelhar
    Given a aba, e a procura deu outra instalação que não conhece esta
    When o utilizador carrega em emparelhar, escreve o código que a outra mostra, e confirma
    Then a outra passa a estar nas máquinas

  @MQNAS-B09 @unit-level
  Scenario: Os pedidos de emparelhamento desta máquina
    Given a aba de uma instalação a escutar
    When outra instalação começa a emparelhar com ela, e passa um segundo
    Then a aba mostra o pedido com o código

  @MQNAS-E03 @unit-level
  Scenario: Um código errado
    Given a aba a emparelhar com outra instalação
    When o utilizador confirma um código errado
    Then a mensagem diz porquê, e as máquinas ficam como estavam

  @MQNAS-B10 @unit-level
  Scenario: Os ficheiros de uma máquina
    Given a aba das máquinas
    When o utilizador carrega em "Ficheiros" desta máquina
    Then a aba pede a árvore dos ficheiros dela

  @MQNAS-B11 @unit-level
  Scenario: Deixar outras máquinas ligarem-se
    Given a aba das máquinas de uma instalação que não escuta
    When o utilizador carrega em "Deixar outras máquinas ligarem-se", e depois no mesmo botão outra vez
    Then o nó escuta entre os dois cliques, o botão diz "Deixar de aceitar outras máquinas", e no fim não escuta

  @MQNAS-B12 @unit-level
  Scenario: As máquinas de outras redes
    Given duas instalações com o transporte, a outra a aceitar e a autorizar esta
    When o utilizador escreve no campo de juntar o endereço entre redes da outra e carrega em juntar
    Then a aba diz que esta máquina se alcança de outras redes, e a outra junta-se com o nome "p2p:" e a chave
