# language: en
# @anchors
#   code: PNFST
#   ref: PRNGS
#   updated_at: 2026-10-18
#   layer: feature

@PRNGS
Feature: Pairing

  @PRNGS-B01 @unit-level
  Scenario: O pedido
    Given um cliente com o nome "portatil" a emparelhar com um nó
    When o protocolo chega ao código
    Then o nó mostra o pedido de "portatil", com a impressão digital da chave dele e um código de seis dígitos

  @PRNGS-B02 @unit-level
  Scenario: O código é o mesmo nos dois lados
    Given duas chaves e dois números
    When cada lado calcula o código com as chaves que vê
    Then os dois são iguais, e com outra chave do nó o do cliente é outro

  @PRNGS-B03 @unit-level
  Scenario: Emparelhar
    Given um cliente a emparelhar com um nó
    When o utilizador escreve no cliente o código que o nó mostra
    Then os dois acabam bem, e a chave do cliente está nas chaves autorizadas do nó com o nome dele

  @PRNGS-B04 @unit-level
  Scenario: Um intermediário
    Given um cliente que vê uma chave do nó diferente da que o nó tem
    When o utilizador escreve no cliente o código que o nó mostra
    Then o cliente desiste com ErrWrongCode, e o nó não junta nada

  @PRNGS-E01 @unit-level
  Scenario: Um código errado
    Given um cliente cuja conferência é saltada, a emparelhar
    When chega ao nó um código que não é o dele
    Then o nó dá ErrWrongCode, e não junta nada

  @PRNGS-E02 @unit-level
  Scenario: Um número que não bate com o compromisso
    Given um cliente que revela um número diferente do que comprometeu
    When o nó o recebe
    Then o nó desiste com o erro, e não junta nada

  @PRNGS-E03 @unit-level
  Scenario: Demasiadas falhas
    Given um nó que já teve cinco falhas no último minuto
    When chega outro pedido
    Then o nó recusa-o com ErrTooMany

  @PRNGS-B05 @unit-level
  Scenario: O fim de um pedido
    Given um nó que avisa o fim dos pedidos
    When um emparelhamento acaba bem, e outro com um código errado
    Then o nó avisa os dois, o primeiro sem erro e o segundo com ErrWrongCode
