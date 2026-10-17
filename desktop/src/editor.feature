# language: en
# @anchors
#   code: EDFST
#   ref: EDTRS
#   updated_at: 2026-10-17
#   layer: feature

@EDTRS
Feature: Editor

  @EDTRS-B01 @unit-level
  Scenario: O editor vazio
    Given um editor vazio
    When é desenhado
    Then não tem texto nem ficheiro, e o título é "Sem título"

  @EDTRS-B02 @unit-level
  Scenario: O editor com um ficheiro
    Given o ficheiro "main.rs" com "fn main() {}"
    When o editor o abre
    Then mostra "fn main() {}" realçado em rust, e o título é "main.rs"

  @EDTRS-B03 @unit-level
  Scenario: A marca de alterado
    Given um editor com o ficheiro "notas.txt"
    When o utilizador escreve "x"
    Then o título é "● notas.txt"

  @EDTRS-B04#01 @unit-level
  Scenario: Gravar o ficheiro
    Given um editor com o ficheiro "notas.txt" alterado
    When o utilizador prime Ctrl+S
    Then o ficheiro tem o texto novo, e o título perde a marca

  @EDTRS-B04#02 @unit-level
  Scenario: Gravar o editor vazio pergunta onde
    Given um editor vazio onde o utilizador escreveu "ola"
    When o utilizador prime Ctrl+S e escolhe "novo.py" no diálogo
    Then "novo.py" tem "ola", o título é "novo.py", e a linguagem é python

  @EDTRS-B04#03 @unit-level
  Scenario: Cancelar o diálogo não grava
    Given um editor vazio onde o utilizador escreveu "ola"
    When o utilizador prime Ctrl+S e cancela o diálogo
    Then o editor continua sem ficheiro e com a marca

  @EDTRS-E01 @unit-level
  Scenario: Um ficheiro que não é texto
    Given um ficheiro com bytes que não são UTF-8
    When o editor o abre
    Then a aba mostra o motivo, e não há editor

  @EDTRS-E02 @unit-level
  Scenario: Um ficheiro que não se grava
    Given um editor com um ficheiro cuja pasta deixou de existir, alterado
    When o utilizador prime Ctrl+S
    Then a mensagem diz porquê, e a marca fica
