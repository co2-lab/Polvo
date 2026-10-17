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

  @WNDWI-E02 @unit-level
  Scenario: Um terminal novo que não arranca deixa a doca como estava
    Given uma janela com um terminal, cujos terminais novos não arrancam
    When o utilizador carrega em "+"
    Then a doca continua com um terminal
