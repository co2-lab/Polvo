# language: en
# @anchors
#   code: CNFTR
#   ref: CNTRL
#   updated_at: 2026-10-18
#   layer: feature

@CNTRL
Feature: Control

  @CNTRL-B01 @unit-level
  Scenario: fs.read devolve o conteúdo e o tamanho
    Given o ficheiro "a.txt" com o texto "ola polvo"
    When o controlo recebe o pedido fs.read com o caminho dele
    Then a resposta traz o conteúdo "ola polvo" em base64 e o tamanho 9

  @CNTRL-B02 @unit-level
  Scenario: fs.write cria o ficheiro com 0644 e depois substitui-o
    Given uma pasta vazia
    When o controlo recebe fs.write de "b.txt" com "um" e depois com "dois"
    Then o ficheiro "b.txt" tem "dois" e as permissões 0644

  @CNTRL-B03 @unit-level
  Scenario: fs.list devolve as entradas por ordem de nome
    Given uma pasta com o ficheiro "z.txt" de 3 bytes, a pasta "m" e o ficheiro "a.txt" de 1 byte
    When o controlo recebe o pedido fs.list com o caminho da pasta
    Then as entradas são "a.txt" (1 byte), "m" (pasta) e "z.txt" (3 bytes), por esta ordem

  @CNTRL-B04 @unit-level
  Scenario: proc.run devolve o código, a saída e o erro separados
    Given o programa "sh -c 'cat; echo erro >&2; exit 5'" com a entrada "entrada"
    When o controlo recebe o pedido proc.run
    Then a resposta traz o código 5, a saída "entrada" e o erro "erro"

  @CNTRL-B04 @unit-level
  Scenario: proc.run corre na pasta pedida
    Given o programa "pwd" e a pasta "/tmp"
    When o controlo recebe o pedido proc.run
    Then a saída é "/tmp"

  @CNTRL-B05 @unit-level
  Scenario: Um sleep lento não atrasa um fs.list que chega depois
    Given o pedido 1 proc.run de "sleep 2" e o pedido 2 fs.list de uma pasta
    When o controlo recebe os dois por esta ordem
    Then a primeira resposta a chegar é a do pedido 2

  @CNTRL-B06 @unit-level
  Scenario: Uma notificação fs.read não tem resposta
    Given a notificação fs.read de um ficheiro e depois o pedido 7 fs.list
    When o controlo recebe as duas
    Then a única resposta é a do pedido 7

  @CNTRL-E01 @unit-level
  Scenario: Um método desconhecido dá método não encontrado
    Given o pedido 1 com o método "fs.apagatudo"
    When o controlo o recebe
    Then a resposta é o erro -32601

  @CNTRL-E02 @unit-level
  Scenario: Um fs.write com base64 inválido dá parâmetros inválidos
    Given o pedido 1 fs.write com o caminho "c.txt" e o conteúdo "!!!"
    When o controlo o recebe
    Then a resposta é o erro -32602 e o ficheiro "c.txt" não existe

  @CNTRL-E03 @unit-level
  Scenario: Ler um ficheiro que não existe dá erro de operação
    Given o pedido 1 fs.read do caminho "/nao/existe"
    When o controlo o recebe
    Then a resposta é o erro -32000 com uma mensagem que contém "/nao/existe"

  @CNTRL-E04 @unit-level
  Scenario: Um ficheiro maior do que 8 MiB não é lido
    Given um ficheiro de 8 MiB e um byte
    When o controlo recebe o pedido fs.read dele
    Then a resposta é o erro -32000 com uma mensagem que contém "too large"

  @CNTRL-E05 @unit-level
  Scenario: Uma linha que não é mensagem tem resposta de erro e a sessão continua
    Given a linha "isto nao e json" e depois o pedido 2 fs.list de uma pasta
    When o controlo recebe as duas
    Then a primeira resposta é o erro -32700 e a segunda é a do pedido 2

  @CNTRL-B07#01 @unit-level
  Scenario: Criar, listar e terminar uma sessão pelo controlo
    Given um nó com sessões
    When o controlo cria uma sessão de "sleep 5", lista as sessões, e termina-a
    Then a lista tem a sessão com o programa, e depois de terminada já não a tem

  @CNTRL-B07#02 @unit-level
  Scenario: O dono trava uma sessão
    Given um nó com uma sessão
    When o dono a trava
    Then a lista diz que está travada

  @CNTRL-B07#04 @unit-level
  Scenario: A lista diz a cada ligação se pode escrever
    Given um nó com uma sessão livre e uma travada
    When quem só vê, quem controla e o dono pedem a lista
    Then quem só vê não escreve em nenhuma, quem controla escreve na livre, e o dono nas duas

  @CNTRL-B07#03 @unit-level
  Scenario: Sem sessões, os pedidos de sessão dão erro
    Given o controlo de um nó sem sessões
    When pede a lista das sessões
    Then dá erro de operação

  @CNTRL-B08 @CNTRL-E06 @unit-level
  Scenario: O que cada permissão pode fazer
    Given um nó com sessões, uma ligação que só vê e outra que controla
    When a que só vê lê um ficheiro, lista as sessões, escreve um ficheiro, corre um processo e cria uma sessão, e a que controla trava uma sessão
    Then ler e listar dão resultado, e escrever, correr, criar e travar dão erro de permissão


  @CNTRL-B09#01 @unit-level
  Scenario: O agente reporta e quem acompanha lê
    Given o controlo de um nó com o quadro do progresso
    When o agente começa um trabalho com o passo "ler", passa-o a feito, deixa uma nota e termina
    Then o progress.get dá o trabalho terminado com o passo feito, e o progress.list tem-no

  @CNTRL-B09#02 @unit-level
  Scenario: Esperar pelo progresso
    Given o controlo de um nó com o quadro do progresso, e uma espera desde a versão atual
    When o agente deixa uma nota
    Then a espera responde com a nota

  @CNTRL-B09#03 @unit-level
  Scenario: Quem só vê não reporta, e sem quadro há erro
    Given o controlo de um nó com o quadro, para quem só vê, e o de um nó sem quadro
    When quem só vê reporta, e se pede a lista ao nó sem quadro
    Then o primeiro é recusado pela permissão, e o segundo dá erro de operação

  @CNTRL-B10#01 @unit-level
  Scenario: Perguntar ao claude pela conversa dele
    Given o trabalho de um agente com a conversa "abc" do claude, e um claude que responde "a meio"
    When se pergunta "onde vais?"
    Then o pedido dá o id da pergunta, e a pergunta fica respondida com "a meio"

  @CNTRL-B10#02 @unit-level
  Scenario: Perguntar ao agente de uma sessão, que responde pela ferramenta
    Given o trabalho do agente da sessão de uma shell do nó, sem conversa
    When se pergunta "onde vais?", e o agente responde "a meio" pelo op answer
    Then a sessão recebe a pergunta com o id, e a pergunta fica respondida com "a meio"

  @CNTRL-B10#03 @unit-level
  Scenario: Sem conversa nem sessão, a pergunta falha
    Given o trabalho de um agente sem conversa e que não é de uma sessão
    When se pergunta "onde vais?"
    Then a pergunta fica falhada com o motivo
