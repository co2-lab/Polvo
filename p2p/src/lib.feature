# language: en
# @anchors
#   code: P2FTR
#   ref: PTRNS
#   updated_at: 2026-10-18
Feature: O transporte entre redes

  @PTRNS-B01 @unit-level
  Scenario: O endpoint de uma instalação
    Given o segredo de uma chave, sem relay
    When se abre o endpoint
    Then o id é a chave pública dela, e o endereço tem a chave e pelo menos um endereço direto

  @PTRNS-B02 @unit-level
  Scenario: Ligar e aceitar
    Given duas instalações sem relay
    When uma se liga à outra pelo endereço dela, escreve "ola" e acaba, e a outra devolve o que leu
    Then a primeira lê "ola" e o fim

  @PTRNS-B03 @unit-level
  Scenario: A chave do outro lado
    Given duas instalações sem relay
    When uma se liga à outra, e a outra aceita; e depois uma liga-se ao endereço da outra com uma terceira chave
    Then cada lado sabe a chave do outro, e a ligação com a chave errada falha

  @PTRNS-B04 @unit-level
  Scenario: A API C
    Given duas instalações abertas pela API C
    When uma se liga à outra, escreve "ola" e diz que acabou; a outra lê e devolve; e depois fecha-se o endpoint que aceita
    Then a primeira lê "ola" e o fim, e um accept à espera dá 0

  @PTRNS-E01 @unit-level
  Scenario: Um endereço que não é
    Given uma instalação
    When se liga a "isto não é um endereço"
    Then é um erro

  @PTRNS-E02 @unit-level
  Scenario: Um relay que não é URL
    Given o relay "não é url"
    When se abre o endpoint
    Then é um erro

  @PTRNS-B05 @unit-level
  Scenario: A porta de cada rede
    Given uma porta UDP livre
    When se abre um endpoint nela, e outro na mesma
    Then o primeiro escuta nela, e o segundo noutra
