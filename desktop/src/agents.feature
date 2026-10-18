# language: en
# @anchors
#   code: AGFSS
#   ref: AGNTS
#   updated_at: 2026-10-18
#   layer: feature

@AGNTS
Feature: Agents

  @AGNTS-B01 @unit-level
  Scenario: Os agentes de todas as máquinas
    Given uma instalação onde o agente "a" tem o trabalho "migrar" com os passos "ler" feito e "escrever" em curso
    When a aba se mostra
    Then mostra o agente com "migrar", 1 de 2 passos feitos, e cada passo com o estado

  @AGNTS-B02 @unit-level
  Scenario: Os últimos estados
    Given o agente com dez notas no registo
    When a aba se mostra
    Then mostra as oito mais novas, a mais nova primeiro

  @AGNTS-B03#01 @unit-level
  Scenario: Perguntar uma pergunta comum
    Given a aba com o agente "a"
    When o utilizador carrega em "Em que ponto estás?"
    Then o agente tem essa pergunta no nó

  @AGNTS-B03#02 @unit-level
  Scenario: Perguntar uma pergunta escrita
    Given a aba com o agente "a"
    When o utilizador escreve "quanto falta?" e carrega em perguntar
    Then o agente tem essa pergunta no nó

  @AGNTS-B04 @unit-level
  Scenario: As perguntas e as respostas
    Given o agente com uma pergunta respondida "a meio" e outra falhada "sem sessão"
    When a aba se mostra
    Then mostra as duas, com "a meio" e com "sem sessão"

  @AGNTS-B05 @unit-level
  Scenario: O feed acompanha as mudanças
    Given um feed que já leu o progresso
    When um agente deixa uma nota, e o feed lê outra vez
    Then quem o ouve recebe a nota

  @AGNTS-B06 @unit-level
  Scenario: A marca de um agente numa sessão
    Given os agentes das sessões "s1" (com um passo em curso), "s2" (com uma pergunta por responder), "s3" (com um passo bloqueado) e "s4" (terminado)
    When o feed lê o progresso
    Then as marcas são a trabalhar, à espera, bloqueado e terminado

  @AGNTS-E01 @unit-level
  Scenario: Uma máquina que não responde
    Given uma instalação com outra máquina juntada que entretanto fechou
    When a aba se mostra
    Then a outra máquina diz porquê, e os agentes desta continuam

  @AGNTS-E02 @unit-level
  Scenario: Perguntar a um agente de uma máquina que fechou
    Given a aba com o agente "a" de outra máquina, que entretanto fechou
    When o utilizador carrega numa pergunta comum
    Then a mensagem junto do agente diz porquê
