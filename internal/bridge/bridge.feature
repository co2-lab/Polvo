# language: en
# @anchors
#   code: BRFST
#   ref: BRDGE
#   updated_at: 2026-10-17
#   layer: feature

@BRDGE
Feature: Bridge

  @BRDGE-B01 @unit-level
  Scenario: Abrir e fechar uma estação
    Given duas estações abertas
    When a primeira fecha
    Then os números eram diferentes, e o da primeira deixou de valer

  @BRDGE-B02 @unit-level
  Scenario: Os pedidos sobre uma estação
    Given uma estação
    When pede o estado, as máquinas, cria uma sessão local, trava-a, lista-as e termina-a
    Then o estado não escuta e dá a chave da instalação, as máquinas são a local, e a sessão aparece travada antes de terminar

  @BRDGE-B03 @unit-level
  Scenario: Um terminal ligado a uma sessão
    Given uma sessão local de uma shell
    When um terminal se liga, escreve "echo $((40+2))", muda o tamanho e larga
    Then lê "42", e a sessão continua na lista

  @BRDGE-E01 @unit-level
  Scenario: Um número que não vale
    Given uma estação
    When se lê do número dela como se fosse um terminal, e se pede algo ao número 0
    Then os dois dão o erro do número

  @BRDGE-E02 @unit-level
  Scenario: Um pedido que não se percebe
    Given uma estação
    When pede "nao json" e o op "voar"
    Then os dois dão erro

  @BRDGE-E03 @unit-level
  Scenario: O erro da máquina chega a quem chamou
    Given uma estação
    When pede as sessões da máquina "nenhuma"
    Then o erro diz a máquina
