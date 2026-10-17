# language: en
# @anchors
#   code: CLFEA
#   ref: CLIXX
#   updated_at: 2026-10-17
#   layer: feature

@CLIXX
Feature: Cli

  @CLIXX-B01 @unit-level
  Scenario: keygen cria a identidade e mostra a mesma linha duas vezes
    Given uma pasta de configuração vazia
    When o polvo keygen corre duas vezes nela
    Then as duas saídas são a mesma linha "ssh-ed25519 …" e o código de saída é 0

  @CLIXX-B02 @unit-level
  Scenario: node escuta no endereço pedido e mostra a impressão digital
    Given uma pasta de configuração do nó com a chave do desktop autorizada
    When o polvo node corre com --listen 127.0.0.1:0
    Then a saída tem uma linha com o endereço em que escuta e a impressão digital "SHA256:" da chave do nó

  @CLIXX-B03 @unit-level
  Scenario: connect sem terminal corre a shell com 24 por 80 e devolve o código dela
    Given um nó a correr e a entrada "stty size; exit 4" sem terminal
    When o polvo connect liga-se ao nó
    Then a saída contém "24 80" e o código de saída é 4

  @CLIXX-B04 @unit-level
  Scenario: o terminal local fica sem eco durante a ligação e volta ao que era
    Given um nó a correr e a entrada ligada a um terminal com eco
    When o polvo connect liga-se, a shell recebe "stty size" e depois "exit 5"
    Then durante a ligação o terminal local está sem eco, e depois tem eco outra vez

  @CLIXX-B05 @unit-level
  Scenario: a janela local de 33 por 99 passa a 44 por 111 e a shell vê-o
    Given um nó a correr e um terminal local de 33 por 99
    When o polvo connect liga-se, a janela local muda para 44 por 111 e a shell recebe "stty size"
    Then a saída contém "33 99" e depois "44 111"

  @CLIXX-B06 @unit-level
  Scenario: mcp inicializa, lista as quatro ferramentas e corre um comando num nó
    Given um nó a correr que autoriza a identidade da pasta de configuração
    When o polvo mcp recebe a inicialização, a lista das ferramentas e a chamada polvo_run de "echo do-agente" no nó, e o stdin acaba
    Then o servidor é "polvo", a lista tem polvo_run, polvo_read_file, polvo_write_file e polvo_list, o resultado tem "do-agente", e o código de saída é 0

  @CLIXX-B07 @unit-level
  Scenario: o terminal local da UI corre a shell do SHELL
    Given o SHELL é um programa que escreve "shell-do-SHELL" e sai com 3
    When o browser abre o terminal do nó "local" na UI
    Then a saída tem "shell-do-SHELL" e a última mensagem diz o código 3

  @CLIXX-B07 @unit-level
  Scenario: ui escreve o endereço com o segredo, abre o browser e serve a UI e as chamadas
    Given um nó a correr que autoriza a identidade da pasta de configuração
    When o polvo ui corre com --listen 127.0.0.1:0
    Then a saída tem o endereço "http://127.0.0.1:…/#t=…", o browser é aberto nesse endereço, a raiz dá a página da UI, e a chamada fs.list ao nó com o segredo devolve as entradas

  @CLIXX-B07 @unit-level
  Scenario: polvo sem subcomando é a UI, e --no-browser não abre o browser
    Given uma pasta de configuração
    When o polvo corre sem subcomando com --no-browser
    Then a saída tem o endereço da UI e o browser não é aberto

  @CLIXX-B07 @unit-level
  Scenario: ui recusa escutar num endereço que não é local
    Given uma pasta de configuração
    When o polvo ui corre com --listen 0.0.0.0:0
    Then a saída de erro começa por "polvo: " e o código de saída é 1

  @CLIXX-E01 @unit-level
  Scenario: um subcomando desconhecido mostra o uso e dá 2
    Given os argumentos "voar"
    When o polvo corre
    Then a saída de erro contém "usage" e o código de saída é 2

  @CLIXX-E01 @unit-level
  Scenario: connect sem endereço mostra o uso e dá 2
    Given os argumentos "connect"
    When o polvo corre
    Then a saída de erro contém "usage" e o código de saída é 2

  @CLIXX-E02 @unit-level
  Scenario: connect a um endereço sem nó dá o erro e 1
    Given os argumentos "connect 127.0.0.1:1"
    When o polvo corre
    Then a saída de erro começa por "polvo: " e o código de saída é 1
