# language: en
# @anchors
#   code: STFST
#   ref: STATN
#   updated_at: 2026-10-18
#   layer: feature

@STATN
Feature: Station

  @STATN-B01 @unit-level
  Scenario: A máquina desta janela
    Given uma Station aberta numa pasta vazia, sem escutar
    When cria uma sessão no nó local, liga-se a ela e escreve "echo $((40+2))", e trava-a
    Then lê "42", a sessão aparece travada, e depois do Close não há sessões

  @STATN-B02 @unit-level
  Scenario: As outras máquinas espelham esta
    Given duas Stations, a primeira a escutar e com a chave da segunda autorizada
    When a primeira cria uma sessão e a segunda junta a primeira
    Then a segunda vê a sessão da primeira

  @STATN-B03 @unit-level
  Scenario: Juntar uma máquina
    Given a segunda Station juntou a primeira
    When a segunda fecha e abre outra vez
    Then as máquinas são a local e a primeira

  @STATN-B04 @unit-level
  Scenario: Esquecer uma máquina
    Given a segunda Station juntou a primeira
    When a esquece, fecha e abre outra vez
    Then as máquinas são só a local

  @STATN-B05 @unit-level
  Scenario: Ligar outra vez quando caiu
    Given a segunda Station ligada à primeira
    When a primeira fecha e abre outra vez no mesmo endereço
    Then o Node seguinte da segunda chega às sessões da primeira

  @STATN-E01 @unit-level
  Scenario: Um endereço ocupado
    Given uma Station a escutar num endereço
    When outra abre a escutar no mesmo endereço
    Then a outra abre, o ListenError diz porquê, e o nó local dela funciona

  @STATN-E02 @unit-level
  Scenario: Uma máquina que não responde
    Given uma Station
    When junta um endereço onde ninguém escuta
    Then o Add dá o erro com o endereço, e as máquinas são só a local

  @STATN-B06 @unit-level
  Scenario: O progresso dos agentes desta máquina
    Given uma Station onde um agente começou um trabalho pelo controlo do nó local
    When a Station fecha e abre outra vez na mesma pasta
    Then a lista do progresso tem o trabalho do agente

  @STATN-B07 @unit-level
  Scenario: As sessões sabem onde está o nó
    Given uma Station sem escutar
    When uma sessão local escreve o POLVO_NODE, e a instalação se liga a esse endereço
    Then a ligação entra e lista a sessão

  @STATN-B08 @unit-level
  Scenario: O nó desta máquina emparelha
    Given uma Station a escutar, e outra instalação que ela não conhece
    When a outra começa a emparelhar
    Then a primeira mostra o pedido com o código, e depois de emparelhar já não

  @STATN-B09#01 @unit-level
  Scenario: Anunciar-se na rede
    Given uma Station a escutar, anunciada no grupo de teste
    When se procura no grupo
    Then encontra-a, com a porta onde escuta e a impressão digital da chave do nó

  @STATN-B10 @unit-level
  Scenario: Procurar as máquinas da rede
    Given duas Stations a escutar no grupo de teste, a segunda já com a primeira conhecida
    When a segunda procura
    Then encontra a primeira, conhecida, e não se encontra a si

  @STATN-B11 @unit-level
  Scenario: Emparelhar com uma máquina
    Given uma Station a escutar, e outra instalação que ela não conhece
    When a outra emparelha com o código que a primeira mostra
    Then a outra tem a primeira nas máquinas, e lista as sessões dela

  @STATN-B09#02 @unit-level
  Scenario: No loopback não se anuncia
    Given uma Station a escutar no loopback, com o grupo de teste
    When se procura no grupo
    Then não encontra nada
