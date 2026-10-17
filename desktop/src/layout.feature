# language: en
# @anchors
#   code: DLFDC
#   ref: DCLYD
#   updated_at: 2026-10-17
#   layer: feature

@DCLYD
Feature: Layout

  @DCLYD-B01 @unit-level
  Scenario: O ficheiro fica na pasta de configuração do sistema
    Given a pasta de configuração do utilizador
    When o Polvo pede o ficheiro da organização
    Then é o "layout.json" na pasta "polvo" dela

  @DCLYD-B02 @unit-level
  Scenario: Gravar cria a pasta e não deixa ficheiros a meio
    Given uma pasta que ainda não existe
    When o Polvo grava uma organização nela
    Then a pasta passa a existir só com o "layout.json", sem o ficheiro do lado

  @DCLYD-B03 @unit-level
  Scenario: Ler devolve a organização guardada
    Given uma organização gravada, e uma pasta sem nenhuma
    When o Polvo as lê
    Then a primeira é igual à gravada, e a segunda não tem organização

  @DCLYD-E01 @unit-level
  Scenario: Um ficheiro estragado ou de outra versão não dá organização
    Given um "layout.json" com "isto não é json", e outro com a versão 99
    When o Polvo os lê
    Then nenhum dá organização

  @DCLYD-E02 @unit-level
  Scenario: Uma pasta que não se escreve dá o erro
    Given o ficheiro dentro de um caminho que é um ficheiro, e não uma pasta
    When o Polvo grava uma organização
    Then a gravação dá o erro
