# language: en
# @anchors
#   code: SCFTS
#   ref: SCRNE
#   updated_at: 2026-10-17
#   layer: feature

@SCRNE
Feature: Screen

  @SCRNE-B01 @unit-level
  Scenario: A shell corre num pty do tamanho do terminal
    Given um terminal de 24 linhas por 80 colunas
    When a shell escreve o TERM, o COLORTERM e o tamanho do pty
    Then o ecrã mostra "xterm-256color truecolor" e "24 80"

  @SCRNE-B02 @unit-level
  Scenario: A saída da shell é o ecrã
    Given um terminal
    When a shell escreve "ola", muda de linha, escreve "mundo" a vermelho e volta o cursor ao início para escrever "M"
    Then o ecrã mostra "ola" e "Mundo", e o troço "undo" é vermelho

  @SCRNE-B03 @unit-level
  Scenario: As respostas do terminal voltam à shell
    Given um terminal com o cursor na terceira coluna da primeira linha
    When a shell pergunta a posição do cursor
    Then o terminal escreve para a shell "ESC [ 1 ; 3 R"

  @SCRNE-B04 @unit-level
  Scenario: O tamanho muda o terminal e o pty
    Given um terminal de 24 por 80 com uma shell
    When o terminal passa a 10 linhas por 40 colunas e a shell escreve o tamanho e uma linha de 50 letras
    Then o ecrã mostra "10 40" e a linha de 50 letras quebrada em 40 e 10

  @SCRNE-B05 @unit-level
  Scenario: O scroll mostra o histórico, e escrever volta ao fim
    Given um terminal de 5 linhas onde a shell escreveu as linhas de 1 a 20
    When a vista sobe 10 linhas, e depois chega uma tecla
    Then a vista mostra as linhas de 6 a 10, e depois as do fim

  @SCRNE-B06#01 @unit-level
  Scenario: Carregar e arrastar seleciona de uma célula à outra
    Given um terminal com "abc def ghi" na primeira linha
    When o rato carrega na segunda célula, arrasta até à sexta e solta
    Then o texto selecionado é "bc de"

  @SCRNE-B06#02 @unit-level
  Scenario: Dois cliques selecionam a palavra e três a linha
    Given um terminal com "abc def ghi" na primeira linha
    When o rato clica duas vezes em "def", e depois três vezes
    Then o texto selecionado é "def", e depois "abc def ghi"

  @SCRNE-B07 @unit-level
  Scenario: Colar respeita o modo do terminal
    Given um terminal
    When cola "a\nb" sem o modo de colar entre parênteses, e depois com ele ligado
    Then a shell recebe "a\rb", e depois "ESC [ 200 ~ a\nb ESC [ 201 ~"

  @SCRNE-B08 @unit-level
  Scenario: O desenho mostra o cursor e a seleção
    Given um terminal com "abc" e o cursor depois do "c"
    When a "b" está selecionada e o terminal é desenhado
    Then o troço "b" e a célula do cursor têm a frente e o fundo trocados, e o "a" e o "c" não

  @SCRNE-B09 @unit-level
  Scenario: O fim da shell chega pelo canal
    Given uma shell que escreve "adeus" e termina
    When o terminal lê o canal
    Then chega a saída com "adeus" e depois o fim

  @SCRNE-B10 @unit-level
  Scenario: O título do terminal é o que a shell lhe dá
    Given um terminal novo
    When a shell põe o título "~/projeto" com a sequência OSC 2
    Then o título é "~/projeto", e antes dela era vazio

  @SCRNE-B11 @unit-level
  Scenario: Largar o terminal termina a shell
    Given uma shell que espera 30 segundos
    When o terminal deixa de existir
    Then o canal dá o fim em poucos segundos

  @SCRNE-E01 @unit-level
  Scenario: Uma shell que não arranca dá o erro
    Given o programa "/nao/existe/shell"
    When o terminal o arranca
    Then o erro diz "/nao/existe/shell"

  @SCRNE-B12#01 @unit-level
  Scenario: O terminal de uma sessão mostra-a e escreve nela
    Given uma sessão da shell na máquina local
    When um terminal se liga a ela e cola "echo marca-$((40+2))" com um Enter
    Then o ecrã mostra "marca-42"

  @SCRNE-B12#02 @unit-level
  Scenario: O tamanho do terminal é o da sessão
    Given o terminal de uma sessão de 24 por 80
    When muda para 30 por 100
    Then a sessão passa a ter 100 colunas

  @SCRNE-B12#03 @unit-level
  Scenario: Largar o terminal deixa a sessão
    Given o terminal de uma sessão
    When o terminal deixa de existir
    Then o canal dá o fim, e a sessão continua na lista
