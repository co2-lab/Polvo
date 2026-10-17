# language: en
# @anchors
#   code: FLFST
#   ref: FLTRE
#   updated_at: 2026-10-17
#   layer: feature

@FLTRE
Feature: Files

  @FLTRE-B01 @unit-level
  Scenario: O conteúdo da pasta
    Given uma pasta com "b.txt", "A.md", a pasta "zeta" e a pasta "Alfa"
    When a árvore a mostra
    Then a ordem é "Alfa", "zeta", "A.md", "b.txt", e "Alfa" está escolhida

  @FLTRE-B02#01 @unit-level
  Scenario: Carregar numa pasta expande-a e recolhe-a
    Given a árvore de uma pasta com a pasta "src", que tem "main.rs"
    When o utilizador carrega em "src", e depois outra vez
    Then "main.rs" aparece por baixo de "src", um nível para dentro, e depois desaparece

  @FLTRE-B02#02 @unit-level
  Scenario: As setas expandem e recolhem
    Given a árvore de uma pasta com a pasta "src", que tem "main.rs"
    When o utilizador prime a seta para a direita, e depois a seta para a esquerda
    Then "main.rs" aparece, e depois desaparece

  @FLTRE-B02#03 @unit-level
  Scenario: Expandir lê a pasta outra vez
    Given a árvore com a pasta "src" expandida e recolhida
    When aparece "novo.rs" em "src" e o utilizador a expande outra vez
    Then "novo.rs" aparece

  @FLTRE-B03#01 @unit-level
  Scenario: Carregar num ficheiro abre-o
    Given a árvore de uma pasta com "notas.txt"
    When o utilizador carrega em "notas.txt"
    Then a árvore abre o caminho de "notas.txt"

  @FLTRE-B03#02 @unit-level
  Scenario: O Enter abre o ficheiro escolhido
    Given a árvore de uma pasta com a pasta "src" e "notas.txt"
    When o utilizador prime a seta para baixo e o Enter
    Then a árvore abre o caminho de "notas.txt"

  @FLTRE-B04 @unit-level
  Scenario: Andar com o teclado
    Given a árvore com a pasta "src" expandida, que tem "main.rs"
    When o utilizador prime a seta para baixo três vezes, e depois a seta para a esquerda
    Then o escolhido é "main.rs" depois das setas para baixo, e "src" depois da seta para a esquerda

  @FLTRE-E01 @unit-level
  Scenario: Uma pasta que não se lê
    Given a árvore de uma pasta com a pasta "fechada", que foi apagada depois de a árvore abrir
    When o utilizador a expande
    Then por baixo dela aparece o motivo, e o resto da árvore continua
