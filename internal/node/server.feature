# language: en
# @anchors
#   code: SRFTR
#   ref: SRVRE
#   updated_at: 2026-10-17
#   layer: feature

@SRVRE
Feature: Server

  @SRVRE-B01 @unit-level
  Scenario: A chave autorizada entra, a outra chave e a palavra-passe não
    Given um nó com a chave A na lista de chaves autorizadas
    When se liga com a chave A, com a chave B e com a palavra-passe "segredo"
    Then a ligação com a chave A abre uma sessão, e as outras duas são recusadas

  @SRVRE-B02 @unit-level
  Scenario: A shell interativa corre num terminal de 30 por 100
    Given um nó e uma sessão que pediu um terminal de 30 linhas por 100 colunas
    When a sessão pede a shell e escreve "stty size; exit" e um fim de linha
    Then a saída contém "30 100"

  @SRVRE-B03 @unit-level
  Scenario: A mudança de janela para 40 por 120 chega à shell
    Given um nó e uma sessão com uma shell num terminal de 24 por 80
    When a sessão muda a janela para 40 por 120 e escreve "stty size; exit" e um fim de linha
    Then a saída contém "40 120"

  @SRVRE-B04 @unit-level
  Scenario: O comando avulso separa a saída do erro e recebe a entrada
    Given um nó e uma sessão sem terminal com a entrada "entrada"
    When a sessão corre "echo saida; echo erro >&2; cat"
    Then a saída é "saida" e "entrada", e o erro é "erro"

  @SRVRE-B05 @unit-level
  Scenario: O comando avulso com terminal vê o terminal
    Given um nó e uma sessão que pediu um terminal
    When a sessão corre "tty >/dev/null && echo tem-tty"
    Then a saída contém "tem-tty"

  @SRVRE-B06 @unit-level
  Scenario: O código de saída 7 chega a quem ligou
    Given um nó e uma sessão sem terminal
    When a sessão corre "exit 7"
    Then a sessão termina com o código de saída 7

  @SRVRE-B06 @unit-level
  Scenario: O sinal que matou o comando chega a quem ligou
    Given um nó e uma sessão sem terminal
    When a sessão corre "kill -9 $$"
    Then a sessão termina com o sinal "KILL"

  @SRVRE-B07 @unit-level
  Scenario: O subsistema polvo chega ao controlo e o sftp é recusado
    Given um nó cujo controlo responde "controlo:" seguido do que recebe
    When uma sessão pede o subsistema "polvo" e escreve "oi", e outra pede o subsistema "sftp"
    Then a primeira lê "controlo:oi", e o pedido do "sftp" é recusado

  @SRVRE-B08 @unit-level
  Scenario: O reencaminhamento de portas e um pedido desconhecido são recusados
    Given um nó e uma ligação aberta
    When a ligação pede um reencaminhamento para "127.0.0.1:1" e uma sessão envia o pedido "xpto"
    Then o reencaminhamento é recusado com o motivo de tipo desconhecido, o pedido "xpto" é recusado, e a sessão ainda corre "echo ainda"

  @SRVRE-E01 @unit-level
  Scenario: Sem a identidade do nó o nó não serve
    Given uma configuração sem a identidade do nó
    When o nó tenta servir
    Then o nó recusa com erro

  @SRVRE-E02 @unit-level
  Scenario: Uma shell que não existe dá 127 e a mensagem no erro
    Given um nó cuja shell é "nao-existe-polvo"
    When uma sessão sem terminal corre "echo x"
    Then a sessão termina com o código de saída 127 e o erro contém "nao-existe-polvo"
