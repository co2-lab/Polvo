# language: en
# @anchors
#   code: CDFTC
#   ref: CDCOE
#   updated_at: 2026-10-17
#   layer: feature

@CDCOE
Feature: Codec

  @CDCOE-B01 @unit-level
  Scenario: Duas mensagens em duas linhas, com uma linha vazia no meio, chegam pela ordem
    Given um fluxo com a linha {"jsonrpc":"2.0","method":"a"}, uma linha vazia e a linha {"jsonrpc":"2.0","method":"b"}
    When o leitor lê o fluxo até ao fim
    Then o leitor entrega a notificação "a" e depois a notificação "b", e mais nenhuma mensagem

  @CDCOE-B02 @unit-level
  Scenario: O pedido, a notificação e a resposta distinguem-se pelos campos
    Given um fluxo com um pedido com o identificador 1 e o método "fs.read", uma notificação "pty.output" e uma resposta ao identificador 1 com o resultado "ok"
    When o leitor lê as três mensagens
    Then a primeira é um pedido, a segunda é uma notificação e a terceira é uma resposta com o resultado "ok"

  @CDCOE-B03 @unit-level
  Scenario: O identificador 7 e o identificador "7" são diferentes
    Given um pedido escrito com o identificador inteiro 7 e outro com o identificador de texto "7"
    When o leitor lê os dois pedidos
    Then o primeiro identificador é o inteiro 7, o segundo é o texto "7", e os dois não são iguais

  @CDCOE-B04 @unit-level
  Scenario: Cem escritores ao mesmo tempo produzem cem linhas inteiras
    Given cem tarefas que escrevem cada uma uma notificação com um parâmetro de 4 KiB no mesmo fluxo
    When as cem tarefas escrevem ao mesmo tempo
    Then o leitor lê exatamente cem notificações, todas com o parâmetro inteiro

  @CDCOE-B05 @unit-level
  Scenario: Um lote de dois pedidos é recusado como pedido inválido
    Given um fluxo com a linha [{"jsonrpc":"2.0","id":1,"method":"a"},{"jsonrpc":"2.0","id":2,"method":"b"}]
    When o leitor lê a linha
    Then o leitor dá o erro de pedido inválido com o código -32600

  @CDCOE-E01 @unit-level
  Scenario: Uma linha que não é JSON dá erro de parse e a seguinte ainda é lida
    Given um fluxo com a linha {"jsonrpc": e depois a linha {"jsonrpc":"2.0","method":"b"}
    When o leitor lê duas vezes
    Then a primeira leitura dá o erro de parse com o código -32700 e a segunda entrega a notificação "b"

  @CDCOE-E02 @unit-level
  Scenario: Um objeto sem método nem resultado dá pedido inválido com o identificador lido
    Given um fluxo com a linha {"jsonrpc":"2.0","id":9} e depois a linha {"jsonrpc":"2.0","method":"b"}
    When o leitor lê duas vezes
    Then a primeira leitura dá o erro de pedido inválido com o código -32600 e o identificador 9, e a segunda entrega a notificação "b"

  @CDCOE-E02 @unit-level
  Scenario: Uma mensagem sem a versão 2.0 dá pedido inválido
    Given um fluxo com a linha {"jsonrpc":"1.0","id":3,"method":"a"}
    When o leitor lê a linha
    Then o leitor dá o erro de pedido inválido com o código -32600 e o identificador 3

  @CDCOE-E03 @unit-level
  Scenario: Uma linha maior do que o limite é descartada e a seguinte ainda é lida
    Given um leitor com o limite de 64 bytes e um fluxo com uma linha de 200 bytes e depois a linha {"jsonrpc":"2.0","method":"b"}
    When o leitor lê duas vezes
    Then a primeira leitura dá o erro de mensagem grande demais e a segunda entrega a notificação "b"

  @CDCOE-E04 @unit-level
  Scenario: O fluxo que acaba a meio de uma linha dá fim inesperado
    Given um fluxo que termina com {"jsonrpc":"2.0","met sem fim de linha
    When o leitor lê
    Then o leitor dá o erro de fim inesperado do fluxo

  @CDCOE-E04 @unit-level
  Scenario: O fluxo que acaba entre linhas dá o fim normal
    Given um fluxo com a linha {"jsonrpc":"2.0","method":"a"} e o fim depois do fim de linha
    When o leitor lê duas vezes
    Then a primeira leitura entrega a notificação "a" e a segunda dá o fim normal do fluxo

  @CDCOE-E05 @unit-level
  Scenario: Uma resposta com resultado e erro ao mesmo tempo não é escrita
    Given uma resposta ao identificador 1 com o resultado "ok" e com um erro
    When o escritor tenta escrevê-la
    Then a escrita é recusada e o fluxo continua vazio

  @CDCOE-I01 @unit-level
  Scenario: Escrever e ler devolve a mesma mensagem
    Given um pedido com o identificador "abc", o método "fs.write" e os parâmetros {"path":"/tmp/x","data":"oi"}, e uma resposta de erro ao identificador 5 com o código -32601 e a mensagem "method not found"
    When o escritor as escreve e o leitor as lê
    Then o leitor entrega o mesmo pedido e a mesma resposta de erro, campo a campo
