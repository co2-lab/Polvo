# language: en
# @anchors
#   code: MCFTS
#   ref: MCPSR
#   updated_at: 2026-10-17
#   layer: feature

@MCPSR
Feature: Server

  @MCPSR-B01 @unit-level
  Scenario: A inicialização devolve a versão pedida, as ferramentas como capacidade e o servidor
    Given um servidor "polvo" na versão "0.1.0"
    When o cliente inicializa com a versão "2025-03-26" e depois com a versão "1999-01-01"
    Then a primeira resposta tem a versão "2025-03-26", a segunda a "2025-06-18", e as duas a capacidade das ferramentas e o servidor "polvo" "0.1.0"

  @MCPSR-B02 @unit-level
  Scenario: A lista tem as duas ferramentas pela ordem do servidor
    Given um servidor com as ferramentas "eco" e "falha"
    When o cliente pede a lista das ferramentas
    Then a lista tem "eco" e depois "falha", cada uma com a descrição e o esquema

  @MCPSR-B03 @unit-level
  Scenario: A ferramenta eco devolve o texto, e a ferramenta que falha devolve um resultado de erro
    Given um servidor com a ferramenta "eco", que devolve o texto que recebe, e a "falha", que falha com "sem rede"
    When o cliente chama "eco" com o texto "ola" e depois "falha"
    Then o primeiro resultado é o texto "ola" sem erro, e o segundo é o texto "sem rede" marcado como erro

  @MCPSR-B04 @unit-level
  Scenario: O ping tem resposta vazia
    Given um servidor
    When o cliente envia o ping
    Then a resposta é um objeto vazio

  @MCPSR-B05 @unit-level
  Scenario: A notificação de inicializado não tem resposta
    Given um servidor
    When o cliente envia a notificação de inicializado e depois o ping 9
    Then a única resposta é a do ping 9

  @MCPSR-B06 @unit-level
  Scenario: Uma ferramenta lenta não atrasa o ping que chega depois
    Given um servidor com a ferramenta "lenta", que demora meio segundo
    When o cliente chama "lenta" como pedido 1 e envia o ping 2
    Then a primeira resposta a chegar é a do ping 2

  @MCPSR-B07 @unit-level
  Scenario: O fim da entrada acaba o servidor depois da resposta em curso
    Given um servidor com a ferramenta "lenta"
    When o cliente chama "lenta" e fecha a entrada
    Then a resposta da "lenta" chega e o servidor termina sem erro

  @MCPSR-E01 @unit-level
  Scenario: Um método desconhecido dá -32601
    Given um servidor
    When o cliente pede "resources/list"
    Then a resposta é o erro -32601

  @MCPSR-E02 @unit-level
  Scenario: Uma ferramenta que não existe dá -32602
    Given um servidor
    When o cliente chama a ferramenta "nao-existe"
    Then a resposta é o erro -32602

  @MCPSR-E03 @unit-level
  Scenario: Uma linha que não é JSON tem resposta de erro e o servidor continua
    Given um servidor
    When o cliente envia "isto nao e json" e depois o ping 3
    Then a primeira resposta é o erro -32700 com o identificador nulo e a segunda é a do ping 3
