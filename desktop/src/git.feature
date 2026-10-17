# language: en
# @anchors
#   code: GTFST
#   ref: GITRS
#   updated_at: 2026-10-17
#   layer: feature

@GITRS
Feature: Git

  @GITRS-B01#01 @unit-level
  Scenario: O ramo
    Given um repositório no ramo "trabalho" com um commit
    When a aba o mostra
    Then o ramo é "trabalho"

  @GITRS-B01#02 @unit-level
  Scenario: À frente e atrás do ramo a montante
    Given um clone com um commit local a mais e um commit da origem a menos
    When a aba o mostra
    Then diz um à frente e um atrás

  @GITRS-B01#03 @unit-level
  Scenario: A HEAD destacada
    Given um repositório com a HEAD destacada no primeiro commit
    When a aba o mostra
    Then diz o hash curto desse commit

  @GITRS-B02#01 @unit-level
  Scenario: Os ficheiros com alterações
    Given um repositório com "a.txt" preparado e depois alterado, "b.txt" alterado e "c.txt" novo
    When a aba o mostra
    Then os preparados têm "a.txt", os alterados "a.txt" e "b.txt", e os não seguidos "c.txt"

  @GITRS-B02#02 @unit-level
  Scenario: Sem alterações
    Given um repositório sem alterações
    When a aba o mostra
    Then diz que não há alterações

  @GITRS-B02#03 @unit-level
  Scenario: Um conflito
    Given um repositório com um merge em conflito em "a.txt"
    When a aba o mostra
    Then os em conflito têm "a.txt"

  @GITRS-B03 @unit-level
  Scenario: Os últimos commits
    Given um repositório com os commits "um" e "dois"
    When a aba o mostra
    Then os commits são "dois" e "um", cada um com o hash curto e o autor

  @GITRS-B04 @unit-level
  Scenario: Atualizar
    Given a aba de um repositório sem alterações
    When aparece "novo.txt" e o utilizador carrega em atualizar
    Then os não seguidos têm "novo.txt"

  @GITRS-B05 @unit-level
  Scenario: Abrir um ficheiro alterado
    Given a aba de um repositório com "b.txt" alterado
    When o utilizador carrega em "b.txt"
    Then a aba abre o caminho de "b.txt"

  @GITRS-E01 @unit-level
  Scenario: Uma pasta fora de um repositório
    Given uma pasta que não é de nenhum repositório
    When a aba a mostra
    Then diz que a pasta não é de um repositório git

  @GITRS-E02 @unit-level
  Scenario: Um git que falha
    Given a aba com um programa git que não existe
    When a aba lê o repositório
    Then mostra o motivo
