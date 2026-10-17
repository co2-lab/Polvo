# language: en
# @anchors
#   code: STFTS
#   ref: STTNG
#   updated_at: 2026-10-17
#   layer: feature

@STTNG
Feature: Settings

  @STTNG-B01 @unit-level
  Scenario: O ficheiro fica na pasta de configuração do sistema
    Given a pasta de configuração do utilizador
    When o Polvo pede o ficheiro da configuração
    Then é o "settings.json" na pasta "polvo" dela

  @STTNG-B02 @unit-level
  Scenario: Ler devolve o que foi gravado
    Given a configuração com a aba nova em "ctrl-alt-n" gravada, e uma pasta sem nenhuma
    When o Polvo as lê
    Then a primeira tem a aba nova em "ctrl-alt-n", e a segunda não muda nada

  @STTNG-B03 @unit-level
  Scenario: Gravar cria a pasta e não deixa ficheiros a meio
    Given uma pasta que ainda não existe
    When o Polvo grava uma configuração nela
    Then a pasta passa a existir só com o "settings.json"

  @STTNG-E01 @unit-level
  Scenario: Um ficheiro estragado não muda nada
    Given um "settings.json" com "isto não é json", e outro com "keys" numa lista
    When o Polvo os lê
    Then nenhum muda nada

  @STTNG-E02 @unit-level
  Scenario: Uma pasta que não se escreve dá o erro
    Given o ficheiro dentro de um caminho que é um ficheiro, e não uma pasta
    When o Polvo grava uma configuração
    Then a gravação dá o erro
