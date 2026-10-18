# language: en
# @anchors
#   code: CLFSQ
#   ref: CLDAG
#   updated_at: 2026-10-18
#   layer: feature

@CLDAG
Feature: Claude

  @CLDAG-B01#01 @unit-level
  Scenario: A conversa do id que o claude dá
    Given a pasta do projeto de "/p" com as conversas "a" e "b", a "b" mais recente, e o modelo "m-a" na "a"
    When se procura a conversa de "/p" com o id "a"
    Then é a "a", na pasta "/p", com o modelo "m-a"

  @CLDAG-B01#02 @unit-level
  Scenario: Sem id, a mais recente
    Given a pasta do projeto de "/p" com as conversas "a" e "b", a "b" mais recente
    When se procura a conversa de "/p" sem id
    Then é a "b"

  @CLDAG-B02 @unit-level
  Scenario: Perguntar
    Given um claude que responde "a meio" em JSON e guarda os argumentos
    When se pergunta "onde vais?" à conversa "a" de "/p" com o modelo "m-a"
    Then a resposta é "a meio", e o claude correu com --resume a, --fork-session, --permission-mode plan e --model m-a

  @CLDAG-E01 @unit-level
  Scenario: Uma pasta sem conversas
    Given uma pasta sem conversas do claude
    When se procura a conversa dela
    Then dá o erro com a pasta

  @CLDAG-E02 @unit-level
  Scenario: Um claude que falha
    Given um claude que responde com erro "sem saldo"
    When se pergunta à conversa
    Then dá o erro com "sem saldo"
