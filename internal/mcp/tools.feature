# language: en
# @anchors
#   code: MCFTT
#   ref: MCPTL
#   updated_at: 2026-10-17
#   layer: feature

@MCPTL
Feature: Tools

  @MCPTL-B01 @unit-level
  Scenario: polvo_run devolve o código 3, a saída e o erro
    Given um nó a correr
    When o agente corre "echo saida; echo erro >&2; exit 3" no nó
    Then o resultado não é uma falha e tem "exit code: 3", "saida" na secção da saída e "erro" na do erro

  @MCPTL-B01 @unit-level
  Scenario: polvo_run com pasta e entrada
    Given um nó a correr e a pasta "/tmp"
    When o agente corre "pwd; cat" nessa pasta com a entrada "entrada"
    Then a secção da saída tem "/tmp" e "entrada"

  @MCPTL-B02 @unit-level
  Scenario: polvo_read_file devolve o texto, e um ficheiro binário é uma falha
    Given um nó com o ficheiro de texto "a.txt" com "ola" e o ficheiro "b.bin" com bytes que não são UTF-8
    When o agente lê os dois
    Then a leitura de "a.txt" dá "ola", e a de "b.bin" é uma falha com o tamanho

  @MCPTL-B03 @unit-level
  Scenario: polvo_write_file escreve e diz quanto escreveu
    Given um nó e uma pasta vazia nele
    When o agente escreve "conteúdo" em "c.txt"
    Then o resultado diz 9 bytes e o caminho, e o ficheiro tem "conteúdo"

  @MCPTL-B04 @unit-level
  Scenario: polvo_list mostra as pastas com barra e os ficheiros com o tamanho
    Given uma pasta do nó com a pasta "d" e o ficheiro "f.txt" de 3 bytes
    When o agente lista a pasta
    Then a lista é "d/" e depois "f.txt (3 bytes)"

  @MCPTL-B05 @unit-level
  Scenario: Duas chamadas ao mesmo nó usam a mesma ligação
    Given um nó a correr
    When o agente corre dois comandos no nó
    Then o conjunto tem uma só ligação, a mesma nas duas chamadas

  @MCPTL-B06 @unit-level
  Scenario: A ligação caída é refeita na chamada seguinte
    Given um nó a correr e uma chamada já feita
    When a ligação cai e o agente corre outro comando
    Then o comando corre por uma ligação nova

  @MCPTL-E01 @unit-level
  Scenario: Sem o comando, polvo_run é uma falha
    Given um nó a correr
    When o agente chama polvo_run sem o comando
    Then o resultado é uma falha que nomeia o "command"

  @MCPTL-E02 @unit-level
  Scenario: Um nó que não responde é uma falha com o endereço
    Given o endereço "127.0.0.1:1"
    When o agente corre um comando nele
    Then o resultado é uma falha que contém "127.0.0.1:1"

  @MCPTL-E03 @unit-level
  Scenario: Ler um ficheiro que não existe é uma falha com a mensagem do nó
    Given um nó a correr
    When o agente lê "/nao/existe"
    Then o resultado é uma falha que contém "/nao/existe"
