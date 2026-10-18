# language: en
# @anchors
#   code: SSFTS
#   ref: SSSNS
#   updated_at: 2026-10-18
#   layer: feature

@SSSNS
Feature: Sessions

  @SSSNS-B01 @unit-level
  Scenario: Criar uma sessão
    Given uma pasta
    When o nó cria duas sessões de uma shell nessa pasta, e uma escreve o TERM e a pasta onde está
    Then as duas têm ids diferentes, e a shell escreve "xterm-256color" e a pasta

  @SSSNS-B02 @unit-level
  Scenario: Listar as sessões
    Given duas sessões, a segunda com um espectador
    When o nó lista as sessões
    Then estão pela ordem em que nasceram, com o programa, o tamanho, os espectadores e destravadas

  @SSSNS-B03 @unit-level
  Scenario: Quem se liga recebe o ecrã atual e depois a saída ao vivo
    Given uma sessão onde a shell já escreveu "marca-1"
    When um segundo espectador se liga e a shell escreve 40+2
    Then o segundo espectador recebe "marca-1" do ecrã atual e o 42 uma vez só

  @SSSNS-B04 @unit-level
  Scenario: A saída chega a todos, e o ecrã acompanha sem espectadores
    Given uma sessão sem espectadores cuja shell escreve "tarde" depois de um momento
    When dois espectadores se ligam depois disso
    Then os dois recebem "tarde" do ecrã atual

  @SSSNS-B05 @unit-level
  Scenario: Controlar é escrever e mudar o tamanho, e vale o último tamanho
    Given uma sessão com dois espectadores que controlam
    When o primeiro muda o tamanho para 30 por 100, o segundo para 20 por 60, e o primeiro escreve "stty size"
    Then a shell escreve "20 60"

  @SSSNS-B06 @SSSNS-E03 @unit-level
  Scenario: Só ver não escreve
    Given uma sessão com um espectador que só vê
    When ele escreve e muda o tamanho
    Then as duas coisas são recusadas por ser só de ver

  @SSSNS-B07 @unit-level
  Scenario: A sessão travada só deixa ver, menos ao dono
    Given uma sessão travada, com um espectador que controla e o dono
    When os dois escrevem, e depois a sessão é destravada e o espectador escreve outra vez
    Then o espectador é recusado, o dono escreve, e destravada o espectador escreve

  @SSSNS-B08 @unit-level
  Scenario: O fim do programa acaba a sessão
    Given uma sessão cuja shell escreve "fim" e sai com 3
    When o espectador lê até ao fim
    Then lê "fim" e o código 3, a sessão sai da lista, e ligar-se a ela já não dá

  @SSSNS-B09 @unit-level
  Scenario: Terminar uma sessão
    Given uma sessão de um programa que espera 30 segundos, com um espectador
    When o nó a termina
    Then o espectador chega ao fim em poucos segundos e a sessão sai da lista

  @SSSNS-B10 @unit-level
  Scenario: As respostas do ecrã do nó não chegam ao programa
    Given uma sessão cujo programa pergunta os atributos do terminal e lê o que lhe chega
    When ninguém responde do lado de um espectador
    Then o programa não recebe nada

  @SSSNS-E01 @unit-level
  Scenario: Um programa que não arranca não faz sessão
    Given o programa "/nao/existe/shell"
    When o nó cria a sessão dele
    Then o erro diz "/nao/existe/shell" e a lista continua vazia

  @SSSNS-E02 @unit-level
  Scenario: Um id que não é de nenhuma sessão
    Given um nó sem a sessão "nenhuma"
    When se liga a ela, a trava e a termina
    Then as três dão que a sessão não existe

  @SSSNS-E04 @unit-level
  Scenario: Um espectador que fica para trás é desligado
    Given uma sessão com o limite de 4 KiB por ler, com um espectador que lê e outro que não lê
    When o programa escreve 9 KB em três rajadas de 3 KB, com pausas
    Then o que lê recebe tudo, e o outro é desligado por ficar para trás

  @SSSNS-B11 @unit-level
  Scenario: O ambiente de uma sessão
    Given um Manager com o ambiente comum POLVO_NODE=127.0.0.1:2222
    When cria uma sessão que escreve o POLVO_SESSION e o POLVO_NODE
    Then a sessão escreve o id dela e 127.0.0.1:2222

  @SSSNS-B12 @unit-level
  Scenario: O ecrã em texto
    Given uma sessão que escreve "ola" a vermelho e depois "mundo"
    When se pede o ecrã dela
    Then o texto tem "ola" e "mundo" em linhas, sem as sequências das cores

  @SSSNS-B13 @unit-level
  Scenario: Escrever sem se ligar
    Given uma shell numa sessão
    When se escreve "echo marca-$((40+2))" com a permissão de controlar, e com a de só ver
    Then o ecrã mostra "marca-42", e a de só ver é recusada
