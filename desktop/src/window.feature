# language: en
# @anchors
#   code: WNFTW
#   ref: WNDWI
#   updated_at: 2026-10-18
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
  Scenario: O botão de aba nova
    Given uma janela com um terminal
    When o utilizador carrega em "+", escolhe o terminal e escreve "x"
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
    When o utilizador carrega em "+" e escolhe o terminal
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

  @WNDWI-B15#01 @unit-level
  Scenario: O botão de configuração abre a configuração
    Given uma janela com um terminal
    When o utilizador carrega no botão de configuração
    Then a doca tem a aba da configuração, com o foco

  @WNDWI-B15#02 @unit-level
  Scenario: A configuração só abre uma vez
    Given uma janela com a configuração aberta e um terminal novo à frente
    When o utilizador carrega outra vez no botão de configuração
    Then a doca continua com uma aba da configuração, e é ela que tem o foco

  @WNDWI-B16#01 @unit-level
  Scenario: O atalho de aba nova abre o escolhedor
    Given uma janela com um terminal com o foco
    When o utilizador prime Ctrl+Shift+T e o Enter
    Then a doca tem dois terminais, e o terminal não recebeu as teclas

  @WNDWI-B16#02 @unit-level
  Scenario: O atalho de fechar fecha a aba com o foco
    Given uma janela com dois terminais, o segundo com o foco
    When o utilizador prime Ctrl+Shift+W
    Then fica o primeiro terminal

  @WNDWI-B16#03 @unit-level
  Scenario: Os atalhos seguem o mapa da aplicação
    Given o mapa com a configuração em Ctrl+Alt+S
    When o utilizador prime Ctrl+Alt+S num terminal
    Then a doca tem a aba da configuração

  @WNDWI-B17#01 @unit-level
  Scenario: O Esc fecha o escolhedor e devolve o foco
    Given uma janela com um terminal com o foco
    When o utilizador carrega em "+" e prime o Esc
    Then a doca continua com um terminal, o escolhedor fechou, e o terminal tem o foco

  @WNDWI-B17#02 @unit-level
  Scenario: Só um escolhedor de cada vez
    Given uma janela com o escolhedor aberto
    When o utilizador prime o atalho de aba nova
    Then o escolhedor continua o mesmo

  @WNDWI-B17#03 @unit-level
  Scenario: Carregar fora do escolhedor fecha-o
    Given uma janela com o escolhedor aberto
    When o utilizador carrega fora dele
    Then o escolhedor fecha e a doca continua com um terminal

  @WNDWI-B18#01 @unit-level
  Scenario: A árvore de ficheiros abre a pasta da janela
    Given uma janela cuja pasta tem "notas.txt"
    When o utilizador escolhe a árvore de ficheiros no escolhedor
    Then a doca tem a árvore da pasta, com "notas.txt", e com o foco

  @WNDWI-B18#02 @unit-level
  Scenario: Um ficheiro aberto na árvore abre num editor, uma vez
    Given uma janela com a árvore da pasta, que tem "notas.txt"
    When o utilizador abre "notas.txt" na árvore duas vezes
    Then a doca tem um editor com "notas.txt"

  @WNDWI-B19#01 @unit-level
  Scenario: O editor vazio
    Given uma janela com um terminal
    When o utilizador escolhe o editor vazio no escolhedor
    Then a doca tem um editor sem ficheiro, com o foco

  @WNDWI-B19#02 @unit-level
  Scenario: Abrir um ficheiro do computador
    Given uma janela com um terminal
    When o utilizador escolhe abrir um ficheiro e escolhe "notas.txt" no diálogo
    Then a doca tem um editor com "notas.txt"

  @WNDWI-B19#03 @unit-level
  Scenario: Cancelar o diálogo deixa a doca como estava
    Given uma janela com um terminal
    When o utilizador escolhe abrir um ficheiro e cancela o diálogo
    Then a doca continua só com o terminal

  @WNDWI-B12#03 @unit-level
  Scenario: A árvore, o git e o editor voltam com a pasta e o ficheiro
    Given uma janela com a árvore da pasta, a aba do git dela e um editor de "notas.txt", com a organização gravada
    When a janela abre outra vez com o mesmo ficheiro da organização
    Then a doca tem a árvore e a aba do git da mesma pasta, e o editor de "notas.txt"

  @WNDWI-B20#01 @unit-level
  Scenario: A informação de git abre a pasta da janela
    Given uma janela cuja pasta é um repositório git
    When o utilizador escolhe a informação de git no escolhedor
    Then a doca tem a aba do git da pasta, com o foco

  @WNDWI-B20#02 @unit-level
  Scenario: Um ficheiro alterado abre num editor
    Given uma janela com a aba do git de um repositório com "notas.txt" alterado
    When o utilizador carrega em "notas.txt"
    Then a doca tem um editor com "notas.txt"

  @WNDWI-B21#01 @unit-level
  Scenario: As máquinas
    Given uma janela com a instalação aberta
    When o utilizador escolhe as máquinas no escolhedor, e depois outra vez
    Then a doca tem uma aba das máquinas, com o foco

  @WNDWI-B21#02 @unit-level
  Scenario: Sem a instalação, não há máquinas
    Given uma janela sem a instalação
    When o utilizador abre o escolhedor
    Then o escolhedor não tem as máquinas

  @WNDWI-B22#01 @unit-level
  Scenario: Uma sessão abre numa aba
    Given uma janela com a aba das máquinas, e uma sessão local
    When o utilizador carrega na sessão
    Then a doca tem uma aba do terminal ligada a ela

  @WNDWI-B22#02 @unit-level
  Scenario: A aba de uma sessão volta, ou diz porquê
    Given a organização gravada com a aba de uma sessão alheia, e a sessão terminada depois
    When a janela abre outra vez com o mesmo ficheiro da organização
    Then a aba diz que a sessão não se ligou

  @WNDWI-B23 @unit-level
  Scenario: Os agentes
    Given uma janela com a instalação e o progresso dos agentes
    When o utilizador escolhe os agentes no escolhedor, e depois outra vez
    Then a doca tem uma aba dos agentes, com o foco

  @WNDWI-B24 @unit-level
  Scenario: A marca do agente na aba da sessão dele
    Given uma janela com a aba de uma sessão onde corre um agente à espera de uma resposta, e a de outra sem agente
    When o progresso dos agentes muda
    Then a aba da primeira começa por "?", e a da outra não tem marca
