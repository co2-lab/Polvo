# language: en
# @anchors
#   code: RMFST
#   ref: RMTEC
#   updated_at: 2026-10-18
#   layer: feature

@RMTEC
Feature: Remote

  @RMTEC-B01 @unit-level
  Scenario: Abrir a instalação
    Given uma pasta vazia
    When a janela abre a instalação sem escutar
    Then as máquinas são só a local, e o estado não escuta

  @RMTEC-B02 @unit-level
  Scenario: As sessões de uma máquina
    Given a instalação aberta
    When cria uma sessão local, trava-a, lista-as e termina-a
    Then a lista tem a sessão travada onde a janela escreve, e no fim nenhuma

  @RMTEC-B03 @unit-level
  Scenario: Um terminal de uma sessão
    Given uma sessão local da shell
    When um terminal se liga, escreve "echo marca-$((40+2))", muda o tamanho e é largado
    Then lê "marca-42", e a sessão continua na lista

  @RMTEC-E01 @unit-level
  Scenario: O erro da biblioteca
    Given a instalação aberta
    When pede as sessões da máquina "nenhuma"
    Then o erro diz "nenhuma"

  @RMTEC-B04 @unit-level
  Scenario: Juntar e esquecer uma máquina
    Given duas instalações, a segunda a escutar e com a chave da primeira autorizada
    When a primeira junta a segunda pelo endereço, pede as sessões dela, e depois esquece-a
    Then a segunda aparece nas máquinas com as sessões dela, e no fim só fica a local

  @RMTEC-B05 @unit-level
  Scenario: O progresso dos agentes
    Given uma instalação onde o agente "a" tem o trabalho "migrar" com o passo "ler" em curso
    When a janela pede o progresso sem versões, e pergunta ao agente "onde vais?"
    Then esta máquina tem o agente com o trabalho e o passo, e a pergunta tem um id

  @RMTEC-B06 @unit-level
  Scenario: A rede e o emparelhamento
    Given duas instalações, a primeira a escutar, que não se conhecem
    When a segunda procura na rede, começa a emparelhar com a primeira, e dá o código que a primeira mostra
    Then a procura responde, a primeira mostra o pedido, e a segunda passa a ter a primeira nas máquinas

  @RMTEC-B07 @unit-level
  Scenario: Os ficheiros de uma máquina
    Given uma instalação e uma pasta vazia
    When grava "notas.txt" com "ola", lista a pasta e lê o ficheiro
    Then a lista tem "notas.txt", e a leitura dá "ola"
