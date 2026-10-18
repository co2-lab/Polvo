# language: en
# @anchors
#   code: PRFST
#   ref: PRGRS
#   updated_at: 2026-10-18
#   layer: feature

@PRGRS
Feature: Progress

  @PRGRS-B01 @unit-level
  Scenario: Começar um trabalho
    Given um quadro onde o agente "claude-1" já tinha um trabalho
    When começa outro com o objetivo "migrar" e os passos "ler" e "escrever"
    Then o trabalho dele é o novo, com os dois passos por fazer e o começo no registo

  @PRGRS-B02 @unit-level
  Scenario: O estado de um passo
    Given o trabalho do agente com o passo "ler"
    When o passo passa a em curso com a nota "a ler o código"
    Then o passo está em curso, e o registo tem a mudança com a nota

  @PRGRS-B03 @unit-level
  Scenario: Uma nota
    Given o trabalho do agente
    When deixa a nota "encontrei um teste partido"
    Then o registo tem a nota, e os passos não mudam

  @PRGRS-B04 @unit-level
  Scenario: Terminar
    Given o trabalho do agente
    When termina com o resumo "feito"
    Then o trabalho está terminado, e o registo acaba com o resumo

  @PRGRS-B05 @unit-level
  Scenario: Ler o progresso
    Given os agentes "a" e "b", com o "a" mudado por último
    When se lista o quadro
    Then vem o "a" e depois o "b", e o Get dá o de cada um

  @PRGRS-B06#01 @unit-level
  Scenario: Esperar quando o quadro já mudou
    Given um quadro na versão 3
    When se espera desde a versão 2
    Then a espera volta logo, com a versão 3

  @PRGRS-B06#02 @unit-level
  Scenario: Esperar pela próxima mudança
    Given um quadro, e uma espera desde a versão atual
    When um agente deixa uma nota
    Then a espera volta com a nota

  @PRGRS-B06#03 @unit-level
  Scenario: O prazo da espera
    Given um quadro sem mudanças
    When se espera 100 ms desde a versão atual
    Then a espera volta com a mesma versão

  @PRGRS-B07 @unit-level
  Scenario: O progresso fica gravado
    Given um quadro com o trabalho do agente, com um passo feito
    When um quadro novo abre na mesma pasta
    Then tem o mesmo trabalho

  @PRGRS-B08 @unit-level
  Scenario: O registo tem um limite
    Given o trabalho do agente
    When deixa mais notas do que o limite
    Then o registo tem o limite, e a última nota é a mais nova

  @PRGRS-E01 @unit-level
  Scenario: Um nome de agente que não é aceite
    Given um quadro
    When "a/b", o vazio e um nome maior do que o limite começam um trabalho
    Then os três dão erro, e o quadro fica vazio

  @PRGRS-E02 @unit-level
  Scenario: Reportar sem trabalho
    Given um quadro
    When o agente "x" deixa uma nota sem ter começado
    Then dá ErrNoJob

  @PRGRS-E03 @unit-level
  Scenario: Um passo que não é do trabalho
    Given o trabalho do agente com o passo "ler"
    When muda o passo "voar"
    Then dá ErrNoStep

  @PRGRS-E04 @unit-level
  Scenario: Um estado que não existe
    Given o trabalho do agente com o passo "ler"
    When o passo passa a "talvez"
    Then dá o erro com "talvez", e o passo fica por fazer

  @PRGRS-E05 @unit-level
  Scenario: Um trabalho que não se grava
    Given um quadro cuja pasta deixou de existir
    When o agente começa um trabalho
    Then dá o erro, e o quadro tem o trabalho

  @PRGRS-E06 @unit-level
  Scenario: Um ficheiro que não é um trabalho
    Given uma pasta com um trabalho e um ficheiro estragado
    When um quadro abre nela
    Then tem o trabalho, e não o estragado
