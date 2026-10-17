# language: en
# @anchors
#   code: NTFST
#   ref: NWTBS
#   updated_at: 2026-10-17
#   layer: feature

@NWTBS
Feature: New tab

  @NWTBS-B01 @unit-level
  Scenario: Os tipos, com o primeiro escolhido
    Given o escolhedor com dois tipos
    When é desenhado
    Then mostra os dois pela ordem, o primeiro escolhido, e tem o foco

  @NWTBS-B02#01 @unit-level
  Scenario: As setas e o Enter
    Given o escolhedor com dois tipos
    When o utilizador prime a seta para baixo duas vezes, a seta para cima uma, e o Enter
    Then escolhe o primeiro

  @NWTBS-B02#02 @unit-level
  Scenario: A tecla do número
    Given o escolhedor com dois tipos
    When o utilizador prime o 2
    Then escolhe o segundo

  @NWTBS-B03 @unit-level
  Scenario: Escolher com o rato
    Given o escolhedor com dois tipos
    When o utilizador carrega no segundo
    Then escolhe o segundo

  @NWTBS-B04 @unit-level
  Scenario: Fechar sem escolher
    Given o escolhedor com dois tipos
    When o utilizador prime o Esc
    Then fecha sem escolher
