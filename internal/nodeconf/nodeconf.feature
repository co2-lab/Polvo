# language: en
# @anchors
#   code: NCFND
#   ref: NDCNN
#   updated_at: 2026-10-18
#   layer: feature

@NDCNN
Feature: Nodeconf

  @NDCNN-B01 @unit-level
  Scenario: Sem ficheiro, uma chave autorizada controla
    Given uma pasta sem o node.json
    When o nó lê a configuração
    Then uma chave de fora controla

  @NDCNN-B02 @unit-level
  Scenario: A permissão de uma chave
    Given um node.json com o guest "view" e uma chave com "control"
    When o nó pergunta a permissão dessa chave e a de outra
    Then a nomeada controla e a outra só vê

  @NDCNN-B03 @unit-level
  Scenario: A chave da instalação é o dono
    Given um node.json que dá "view" à chave da própria instalação
    When o nó pergunta a permissão dela
    Then é o dono

  @NDCNN-E01 @unit-level
  Scenario: Um ficheiro estragado não deixa o nó arrancar
    Given um node.json com "isto não é json"
    When o nó lê a configuração
    Then dá o erro com o ficheiro

  @NDCNN-E02 @unit-level
  Scenario: Uma permissão desconhecida não deixa o nó arrancar
    Given um node.json com o guest "admin"
    When o nó lê a configuração
    Then dá o erro com "admin"

  @NDCNN-B04 @unit-level
  Scenario: Quem pode encaminhar sockets
    Given um node.json com a chave "A" na lista forward, a chave "B" que controla fora dela, e a "C" só a ver
    When se pergunta quem pode encaminhar, também sem a lista e com "*"
    Then o dono e a "A" podem, a "B" e a "C" não; sem a lista só o dono; com "*" a "B" também, e a "C" não
