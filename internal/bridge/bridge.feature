# language: en
# @anchors
#   code: BRFST
#   ref: BRDGE
#   updated_at: 2026-10-18
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
    When um terminal se liga, escreve "echo marca-$((40+2))", muda o tamanho e larga
    Then lê "marca-42", e a sessão continua na lista

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

  @BRDGE-B04#01 @unit-level
  Scenario: O progresso de uma máquina, e perguntar a um agente
    Given uma estação onde um agente começou um trabalho
    When pede o progresso desta máquina e pergunta ao agente "onde vais?"
    Then o progresso tem o agente, e a pergunta tem um id

  @BRDGE-B04#02 @unit-level
  Scenario: Esperar pela mudança de qualquer máquina
    Given uma estação, e a versão que a janela já tem dela
    When um agente começa um trabalho enquanto a janela espera
    Then a espera volta com o agente

  @BRDGE-B04#03 @unit-level
  Scenario: Uma máquina que a janela não conhece volta logo
    Given uma estação
    When a janela espera sem versões
    Then a espera volta logo, com esta máquina
