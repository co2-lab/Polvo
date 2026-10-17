# language: en
# @anchors
#   code: APFTR
#   ref: APPUI
#   updated_at: 2026-10-17
#   layer: feature

@APPUI
Feature: App

  @APPUI-B01 @unit-level
  Scenario: O segredo lido do fragmento sai da barra de endereço
    Given o endereço "http://127.0.0.1:8080/#t=abc"
    When a janela arranca
    Then as chamadas levam o segredo "abc" e o endereço fica sem o fragmento

  @APPUI-B02 @unit-level
  Scenario: O nó acrescentado fica lembrado e escolhê-lo lista a raiz
    Given uma janela pronta com o armazenamento vazio
    When o utilizador acrescenta o nó "10.0.0.1:2222"
    Then o armazenamento guarda "10.0.0.1:2222", e a janela pede a lista de "/" a esse nó

  @APPUI-B03 @unit-level
  Scenario: A árvore mostra a pasta de cima, as pastas e os ficheiros por ordem
    Given um nó cuja pasta "/home" tem o ficheiro "z.txt" e as pastas "b" e "a"
    When a janela abre a pasta "/home"
    Then a árvore mostra "..", "a/", "b/" e "z.txt", por esta ordem

  @APPUI-B03 @unit-level
  Scenario: Clicar numa pasta abre-a
    Given a árvore da pasta "/home" com a pasta "a"
    When o utilizador clica em "a/"
    Then a janela pede a lista de "/home/a"

  @APPUI-B04 @unit-level
  Scenario: Abrir um ficheiro põe o texto no editor
    Given um nó com o ficheiro "/home/z.txt" com "ola polvo"
    When a janela abre o ficheiro
    Then o editor tem "ola polvo" e o título é "/home/z.txt"

  @APPUI-B05 @unit-level
  Scenario: Alterar põe a marca e gravar escreve, tira a marca e diz gravado
    Given o ficheiro "/home/z.txt" aberto
    When o editor muda para "novo texto" e o utilizador grava
    Then a marca aparece depois da alteração, a janela escreve "novo texto" em "/home/z.txt", a marca sai e a barra diz "Gravado: /home/z.txt"

  @APPUI-B06 @unit-level
  Scenario: O terminal envia a entrada e o tamanho, mostra a saída e o fim
    Given uma janela pronta com o nó "10.0.0.1:2222"
    When o utilizador abre um terminal de 24 por 80, escreve "ls", o terminal muda para 30 por 100, chega a saída "ola" e chega o fim com o código 0
    Then o WebSocket é o do terminal com o segredo, o nó e 24 por 80, as mensagens enviadas são a entrada "ls" e o tamanho 30 por 100, e o terminal mostra "ola" e "[terminou: código 0]"

  @APPUI-B07 @unit-level
  Scenario: A própria máquina vem primeiro e já escolhida, sem ficar guardada
    Given uma janela pronta com o armazenamento vazio
    When a janela arranca, abre um terminal e o utilizador acrescenta o nó "10.0.0.1:2222"
    Then o primeiro nó é "esta máquina", escolhido, a janela pede a lista de "/" ao nó "local", o terminal abre no nó "local", e o armazenamento guarda só "10.0.0.1:2222"

  @APPUI-E01 @unit-level
  Scenario: Uma chamada que falha mostra o erro na barra
    Given um nó que responde ao fs.read com o erro "open /x: no such file or directory"
    When a janela abre o ficheiro "/x"
    Then a barra de estado diz "Erro: open /x: no such file or directory"

  @APPUI-E02 @unit-level
  Scenario: Sem o segredo só aparece a mensagem
    Given o endereço "http://127.0.0.1:8080/" sem fragmento
    When a janela arranca
    Then a página mostra "Abra o Polvo pelo endereço que o comando polvo mostrou (falta o segredo da sessão)." e não mostra os nós

  @APPUI-S01 @unit-level
  Scenario: O estado sem segredo esconde a janela
    Given o endereço sem o segredo
    When a janela arranca
    Then a lista dos nós, a árvore, o editor e os terminais não aparecem

  @APPUI-S02 @unit-level
  Scenario: O estado pronto mostra a lista dos nós e a barra de estado
    Given o endereço com o segredo
    When a janela arranca
    Then a lista dos nós, o campo para acrescentar e a barra de estado aparecem

  @APPUI-M01 @unit-level
  Scenario: A mensagem sem segredo
    Given o endereço sem o segredo
    When a janela arranca
    Then o texto é "Abra o Polvo pelo endereço que o comando polvo mostrou (falta o segredo da sessão)."

  @APPUI-M02 @unit-level
  Scenario: A mensagem de erro
    Given uma chamada que falha com "sem rede"
    When a janela mostra o erro
    Then o texto é "Erro: sem rede"

  @APPUI-M03 @unit-level
  Scenario: A mensagem de gravado
    Given o ficheiro "/a.txt" gravado
    When a janela o confirma
    Then o texto é "Gravado: /a.txt"

  @APPUI-M04 @unit-level
  Scenario: A mensagem de fim do terminal
    Given um terminal cuja shell terminou com o código 3
    When chega o fim
    Then o terminal mostra "[terminou: código 3]"

  @APPUI-M05 @unit-level
  Scenario: O nome da própria máquina
    Given a lista dos nós
    When aparece a própria máquina
    Then o nome dela é "esta máquina"
