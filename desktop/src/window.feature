# language: en
# @anchors
#   code: WNFTW
#   ref: WNDWI
#   updated_at: 2026-10-17
#   layer: feature

@WNDWI
Feature: Window

  @WNDWI-B01#01 @unit-level
  Scenario: Os botões à esquerda pela ordem do sistema
    Given as preferências com os botões fechar, minimizar e maximizar à esquerda
    When a janela abre
    Then fechar, minimizar e maximizar estão por esta ordem à esquerda do título "Polvo"

  @WNDWI-B01#02 @unit-level
  Scenario: Os botões à direita pela ordem do sistema
    Given as preferências com os botões minimizar, maximizar e fechar à direita
    When a janela abre
    Then minimizar, maximizar e fechar estão por esta ordem à direita do título, que fica ao centro

  @WNDWI-B02 @unit-level
  Scenario: Fechar fecha a janela
    Given uma janela aberta
    When o utilizador carrega em fechar
    Then a janela deixa de existir

  @WNDWI-B07 @unit-level
  Scenario: O tema da barra segue as preferências
    Given as preferências do tema claro
    When a janela abre
    Then a barra é clara

  @WNDWI-B09 @unit-level
  Scenario: Arrastar a aba para a borda divide a doca
    Given uma janela com dois terminais em abas no mesmo grupo
    When o utilizador arrasta a aba do segundo para a borda direita do terminal à vista
    Then os dois terminais ficam à vista, lado a lado, o segundo à direita

  @WNDWI-B14 @unit-level
  Scenario: As abas ficam em baixo
    Given uma janela com dois terminais em abas no mesmo grupo
    When a janela é desenhada
    Then as abas ficam por baixo do terminal à vista

  @WNDWI-B10 @unit-level
  Scenario: O botão de terminal novo
    Given uma janela com um terminal
    When o utilizador carrega em "+" e escreve "x"
    Then a doca tem dois terminais, e só a shell do novo recebe "x"

  @WNDWI-B11 @unit-level
  Scenario: O fim de uma shell tira a aba, e a última fecha a janela
    Given uma janela com dois terminais
    When a shell do primeiro termina, e depois a do segundo
    Then fica um terminal na doca, e depois a janela deixa de existir

  @WNDWI-B13 @unit-level
  Scenario: Fechar a aba termina a shell
    Given uma janela com dois terminais
    When o utilizador fecha a aba do primeiro, e depois a do segundo
    Then a shell do primeiro deixa de ter terminal, e depois a janela deixa de existir

  @WNDWI-B12#01 @unit-level
  Scenario: Cada mudança da doca grava a organização
    Given uma janela com o ficheiro da organização numa pasta vazia
    When o utilizador abre um segundo terminal e passa meio segundo
    Then o ficheiro tem a organização com os dois terminais

  @WNDWI-B12#02 @unit-level
  Scenario: Ao abrir, a organização volta
    Given o ficheiro com uma organização de dois terminais lado a lado
    When a janela abre
    Then a doca tem os dois terminais lado a lado, cada um com uma shell nova

  @WNDWI-E02 @unit-level
  Scenario: Um terminal novo que não arranca deixa a doca como estava
    Given uma janela com um terminal, cujos terminais novos não arrancam
    When o utilizador carrega em "+"
    Then a doca continua com um terminal

  @WNDWI-E03 @unit-level
  Scenario: Uma organização que não se grava deixa a janela a funcionar
    Given uma janela cujo ficheiro da organização fica dentro de um ficheiro, e não de uma pasta
    When o utilizador abre um segundo terminal e passa meio segundo
    Then a janela continua aberta com os dois terminais

  @WNDWI-E04 @unit-level
  Scenario: Sem nenhum terminal que arranque, a janela fecha
    Given uma janela cujos terminais não arrancam
    When a janela abre
    Then a janela deixa de existir
