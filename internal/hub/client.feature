# language: en
# @anchors
#   code: CLFTR
#   ref: CLNTI
#   updated_at: 2026-10-17
#   layer: feature

@CLNTI
Feature: Client

  @CLNTI-B01#01 @unit-level
  Scenario: O primeiro contacto regista o nó, a segunda ligação entra, e um nó com outra chave é recusado
    Given um nó com a chave N e uma lista de nós conhecidos vazia
    When o desktop se liga duas vezes ao nó e depois a outro nó com a chave M no mesmo endereço
    Then as duas primeiras ligações entram e a terceira é recusada com o erro de chave mudada

  @CLNTI-B02 @unit-level
  Scenario: Uma shell de 30 por 100 muda para 40 por 120 e termina com o código 3
    Given uma ligação a um nó
    When o desktop abre uma shell de 30 por 100, escreve "stty size", muda para 40 por 120, escreve "stty size; exit 3"
    Then a saída contém "30 100" e "40 120" e a shell termina com o código 3

  @CLNTI-B03 @unit-level
  Scenario: Dez chamadas ao mesmo tempo recebem cada uma a sua resposta
    Given uma ligação a um nó com dez ficheiros "f0" a "f9", cada um com o seu número
    When o desktop lê os dez ao mesmo tempo
    Then cada leitura devolve o número do seu ficheiro

  @CLNTI-B04 @unit-level
  Scenario: Um método desconhecido chega como erro com o código -32601
    Given uma ligação a um nó
    When o desktop chama o método "fs.apagatudo"
    Then a chamada falha com o código -32601

  @CLNTI-B05 @unit-level
  Scenario: Os atalhos escrevem, leem, listam e correm em bytes
    Given uma ligação a um nó e uma pasta vazia nele
    When o desktop escreve "conteúdo" em "a.txt", lê-o, lista a pasta e corre "sh -c 'cat a.txt; exit 2'" nela
    Then a leitura dá "conteúdo", a lista tem "a.txt" com 9 bytes, e o processo dá o código 2 e a saída "conteúdo"

  @CLNTI-B06 @unit-level
  Scenario: Fechar a ligação acaba a chamada que esperava
    Given uma ligação a um nó e uma chamada que corre "sleep 5"
    When o desktop fecha a ligação
    Then a chamada termina com erro antes do fim do sleep, e uma chamada nova também falha

  @CLNTI-E01 @unit-level
  Scenario: Um endereço sem nó dá erro com o endereço
    Given o endereço "127.0.0.1:1"
    When o desktop se liga
    Then a ligação falha com um erro que contém "127.0.0.1:1"

  @CLNTI-E02 @unit-level
  Scenario: Uma identidade que o nó não autoriza é recusada
    Given um nó que não autoriza a identidade do desktop
    When o desktop se liga
    Then a ligação falha com erro

  @CLNTI-E03 @unit-level
  Scenario: O nó que cai durante a chamada termina-a com erro
    Given uma ligação a um nó e uma chamada que corre "sleep 5"
    When o nó deixa de servir e a ligação cai
    Then a chamada termina com erro antes do fim do sleep, e o Done fecha-se

  @CLNTI-B07 @unit-level
  Scenario: As sessões do nó
    Given um cliente ligado como dono a um nó com sessões
    When cria uma sessão, a lista, trava-a e termina-a
    Then a lista tem a sessão onde pode escrever, depois travada, e no fim nenhuma

  @CLNTI-B08 @unit-level
  Scenario: Ligar-se a uma sessão
    Given uma sessão de uma shell num nó, com "marca-1" escrito por um primeiro terminal
    When um segundo terminal se liga a ela e escreve "echo $((40+2))"
    Then o segundo lê "marca-1" e depois "42", e fechá-lo deixa a sessão na lista

  @CLNTI-E04 @unit-level
  Scenario: Ligar-se a uma sessão que não existe
    Given um cliente ligado a um nó com sessões
    When se liga à sessão "nenhuma"
    Then recebe ErrNoSession

  @CLNTI-B01#02 @unit-level
  Scenario: Com a chave do nó já sabida
    Given um nó, e a sua chave dada ao cliente sem lista de nós conhecidos
    When o cliente se liga, e depois liga-se com a chave de outro nó
    Then a primeira ligação entra e a segunda é recusada
