# language: en
# @anchors
#   code: TRFTR
#   ref: TRNSP
#   updated_at: 2026-10-18
Feature: A ligação entre redes para o nó e para o cliente

  @TRNSP-B01 @unit-level
  Scenario: O endpoint da instalação
    Given a chave de uma instalação, sem relay
    When se abre o endpoint
    Then o ID é a chave pública dela, e o endereço nomeia-a

  @TRNSP-B02 @TRNSP-B03 @unit-level
  Scenario: Ligar, aceitar, e a chave do outro lado
    Given duas instalações sem relay
    When uma se liga à outra, escreve "ola" e diz que acabou, e a outra devolve o que leu; e depois fecha-se a que aceita
    Then a primeira lê "ola" e o fim, cada lado sabe a chave do outro, e o Accept à espera dá que fechou

  @TRNSP-B04 @unit-level
  Scenario: O SSH por cima
    Given um nó que serve as ligações que a sua instalação aceita
    When outra instalação se liga a ele pelo transporte e corre "echo marca-$((40+2))"
    Then a saída é "marca-42"

  @TRNSP-E01 @unit-level
  Scenario: Um endereço que não é
    Given uma instalação
    When se liga a "isto não é um endereço"
    Then é um erro

  @TRNSP-E03 @unit-level
  Scenario: A ligação não tem prazos
    Given uma ligação entre duas instalações
    When se lhe pede um prazo
    Then é um erro

  @TRNSP-E02 @unit-level
  Scenario: O executável sem o transporte
    Given um executável compilado sem a etiqueta polvo_p2p
    When se abre o endpoint
    Then é o ErrNotBuilt

  @TRNSP-B05 @unit-level
  Scenario: Para que é uma ligação
    Given duas instalações
    When uma abre uma ligação para o SSH e outra para um endereço
    Then a outra aceita-as com KIND_SSH e KIND_ADDR

  @TRNSP-B06 @TRNSP-E04 @unit-level
  Scenario: A chave que um endereço nomeia
    Given o JSON de um endereço com o id K, o texto "p2p:K", e o texto "maquina:2222"
    When se pergunta se são endereços e que chave nomeiam
    Then os dois primeiros são, e nomeiam K; o terceiro não é, e o KeyOf dá um erro
