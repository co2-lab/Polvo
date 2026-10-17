# language: en
# @anchors
#   code: PTFTR
#   ref: PTYXX
#   updated_at: 2026-10-17
#   layer: feature

@PTYXX
Feature: Pty

  @PTYXX-B01 @unit-level
  Scenario: A shell arranca num terminal de 30 linhas por 100 colunas
    Given o programa "sh -c 'tty >/dev/null && echo tem-tty; stty size'" e o tamanho 30 por 100
    When a sessão arranca e é lida até ao fim
    Then a leitura contém "tem-tty" e "30 100"

  @PTYXX-B02 @unit-level
  Scenario: O texto escrito na sessão volta pela leitura
    Given o programa "sh -c 'read l; echo eco:$l'" a correr numa sessão
    When a sessão escreve "ola" e um fim de linha e é lida até ao fim
    Then a leitura contém "eco:ola"

  @PTYXX-B03 @unit-level
  Scenario: O redimensionamento para 40 por 120 chega ao processo
    Given o programa "sh -c 'read l; stty size'" numa sessão de 24 por 80
    When a sessão muda para 40 por 120, escreve um fim de linha e é lida até ao fim
    Then a leitura contém "40 120"

  @PTYXX-B04 @unit-level
  Scenario: O código de saída 3 volta pelo fim do processo
    Given o programa "sh -c 'exit 3'" numa sessão
    When a sessão é lida até ao fim e espera pelo processo
    Then o código de saída é 3

  @PTYXX-B04 @unit-level
  Scenario: Um processo morto por um sinal devolve -1 e o nome do sinal
    Given o programa "sleep 30" numa sessão
    When o processo recebe o sinal de matar e a sessão espera por ele
    Then o código de saída é -1 e o sinal é "killed"

  @PTYXX-B05 @unit-level
  Scenario: A leitura acaba com o fim normal depois do processo
    Given o programa "echo fim" numa sessão
    When a sessão é lida até ao fim
    Then a leitura contém "fim" e acaba sem erro

  @PTYXX-B06 @unit-level
  Scenario: Fechar a sessão termina a shell
    Given o programa "sleep 30" numa sessão
    When a sessão é fechada e espera pelo processo
    Then o processo terminou por um sinal

  @PTYXX-E01 @unit-level
  Scenario: Um programa que não existe falha com o nome dele
    Given o programa "nao-existe-polvo"
    When a sessão tenta arrancar
    Then o arranque falha com um erro que contém "nao-existe-polvo"

  @PTYXX-B07 @unit-level
  Scenario: O processo arranca na pasta e com o ambiente pedidos
    Given uma pasta e a variável "POLVO_TESTE=sim"
    When a sessão arranca "sh -c 'pwd; echo $POLVO_TESTE'" nessa pasta com essa variável
    Then a leitura contém a pasta e "sim"

  @PTYXX-E02 @unit-level
  Scenario: Sem programa o arranque falha
    Given uma lista de programa vazia
    When a sessão tenta arrancar
    Then o arranque falha com erro

  @PTYXX-I01 @unit-level
  Scenario: O tamanho visto é o do último redimensionamento
    Given o programa "sh -c 'stty size; read l; stty size'" numa sessão de 10 por 20
    When a sessão muda para 11 por 21 depois da primeira linha, escreve um fim de linha e é lida até ao fim
    Then a leitura contém "10 20" e depois "11 21"
