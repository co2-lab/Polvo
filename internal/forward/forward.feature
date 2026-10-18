# language: en
# @anchors
#   code: FRFTR
#   ref: FRWRD
#   updated_at: 2026-10-18
Feature: Os sockets encaminhados entre duas máquinas, pelo nó

  @FRWRD-B01 @unit-level
  Scenario: O texto de um socket
    Given os textos "unix:/tmp/s", "maquina:80", ":8080" e "8080"
    When se leem
    Then são o Unix "/tmp/s", e o TCP "maquina:80", "127.0.0.1:8080" e "127.0.0.1:8080"

  @FRWRD-B02 @unit-level
  Scenario: Daqui para lá
    Given um nó, e na máquina dele um socket que devolve o que recebe
    When se encaminha uma porta desta máquina para ele e alguém se liga a ela e escreve "ola"
    Then lê "ola" de volta

  @FRWRD-B03 @unit-level
  Scenario: De lá para cá
    Given um nó, e nesta máquina um socket Unix que devolve o que recebe
    When se põe o nó a escutar numa porta encaminhada para ele, e alguém se liga a ela na máquina do nó e escreve "ola"
    Then lê "ola" de volta

  @FRWRD-B04 @unit-level
  Scenario: Acabar
    Given um encaminhamento daqui para lá
    When se fecha
    Then a porta deixa de aceitar ligações

  @FRWRD-E01 @unit-level
  Scenario: Um texto que não é de um socket
    Given os textos "maquina:porta" e "70000"
    When se leem
    Then são erros com o texto

  @FRWRD-E02 @unit-level
  Scenario: O nó recusa encaminhar
    Given um nó que não deixa esta chave encaminhar
    When se pede o Remote
    Then é um erro, e nada fica a escutar

  @FRWRD-E03 @unit-level
  Scenario: Um destino que não responde
    Given um encaminhamento daqui para lá, para um socket que não responde
    When alguém se liga duas vezes à porta
    Then cada ligação fecha-se, e a porta continua a aceitar
