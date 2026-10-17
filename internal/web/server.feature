# language: en
# @anchors
#   code: WBFTS
#   ref: WEBSR
#   updated_at: 2026-10-17
#   layer: feature

@WEBSR
Feature: Server

  @WEBSR-B01 @unit-level
  Scenario: A raiz dá o index.html sem segredo
    Given um servidor com o ficheiro da UI "index.html" com "<h1>polvo</h1>"
    When o browser pede "/" sem segredo
    Then a resposta é 200 com "<h1>polvo</h1>"

  @WEBSR-B01 @unit-level
  Scenario: Os ficheiros embutidos da UI servem-se
    Given um servidor com os ficheiros embutidos no executável
    When o browser pede "/", "/app.js", "/style.css" e "/xterm.css"
    Then as quatro respostas são 200 com o ficheiro inteiro

  @WEBSR-B02 @unit-level
  Scenario: A chamada fs.list com o segredo devolve as entradas do nó
    Given um servidor e um nó com a pasta que tem o ficheiro "a.txt"
    When o browser chama fs.list dessa pasta com o segredo
    Then o resultado tem a entrada "a.txt"

  @WEBSR-B02 @unit-level
  Scenario: Uma recusa do nó volta como objeto de erro
    Given um servidor e um nó
    When o browser chama fs.read de "/nao/existe" com o segredo
    Then a resposta tem o erro com o código -32000 e "/nao/existe"

  @WEBSR-B03 @unit-level
  Scenario: Sem o segredo, ou com outro, a chamada é recusada
    Given um servidor e um nó
    When o browser chama fs.list sem o segredo e depois com o segredo "errado"
    Then as duas respostas são 401

  @WEBSR-B04 @unit-level
  Scenario: O terminal por WebSocket corre um comando e termina com o código 4
    Given um servidor e um nó
    When o browser abre o terminal de 30 por 100 com o segredo, muda para 40 por 120 e escreve "stty size; exit 4"
    Then a saída tem "40 120" e a última mensagem diz o código 4

  @WEBSR-B05 @unit-level
  Scenario: O WebSocket de outra origem é recusado
    Given um servidor e um nó
    When um browser na origem "http://mal.exemplo" abre o terminal com o segredo
    Then a ligação é recusada com 403

  @WEBSR-B06 @unit-level
  Scenario: Um pedido para um nome que não é local é recusado
    Given um servidor
    When chega um pedido com o Host "polvo.mal.exemplo"
    Then a resposta é 403

  @WEBSR-E01 @unit-level
  Scenario: Um nó que não responde volta como objeto de erro
    Given um servidor
    When o browser chama fs.list no nó "127.0.0.1:1" com o segredo
    Then a resposta tem o erro com "127.0.0.1:1"

  @WEBSR-E02 @unit-level
  Scenario: Um GET à chamada e um corpo que não é JSON são recusados
    Given um servidor
    When o browser faz GET a "/api/call" e um POST com "isto nao e json", os dois com o segredo
    Then as respostas são 405 e 400
