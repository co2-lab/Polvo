<!-- @anchors
  code: WNDWI
  updated_at: 2026-10-17
-->
# Window — a janela do Polvo: a barra do sistema e a doca dos terminais

> **Code**: `WNDWI`

## Overview
A janela do Polvo (`PolvoWindow`) é uma vista GPUI desenhada na GPU: a barra de título por cima e, por
baixo, a doca (a `DockArea` do gpui-component), onde os terminais (`TerminalPanel`) são abas que o
utilizador organiza arrastando e largando. A janela segue as preferências do ambiente de trabalho
(`Prefs`): a ordem dos botões e o tema.

No GNOME com Wayland o sistema não desenha a moldura das janelas: a janela desenha a sua, sem sombra
nem margem à volta (o dono aprovou-a assim a 2026-10-17).

O `PolvoWindow::new` recebe as preferências, a forma de fazer os terminais (`NewTerminal`), o ficheiro
da organização (`LayoutFile`), quando a há, e a pasta da janela (`root`, a pasta onde o Polvo abriu); o `open` abre a janela com a shell do utilizador.

## Rules

### WNDWI-B01 — A barra segue o sistema
A barra de título tem os botões de fechar, minimizar e maximizar nos lados e pela ordem das
preferências, só os que o sistema da janela suporta (fechar está sempre), e o título "Polvo" ao centro.

### WNDWI-B02 — Os botões fazem o que dizem
Fechar fecha a janela, minimizar minimiza-a e maximizar alterna entre maximizada e o tamanho anterior.

### WNDWI-B07 — O tema da barra segue as preferências
As cores da janela (`Colors`) são as do tema: a barra é escura no tema escuro e clara no claro.

### WNDWI-B09 — O corpo é a doca
Os terminais são abas na doca por baixo da barra. A aba de um terminal arrasta-se e larga-se:
- no meio de um grupo, junta-se às abas dele;
- numa borda (esquerda, direita, cima ou baixo), parte esse grupo em dois e fica desse lado.

Os divisores entre os grupos arrastam-se para mudar o tamanho de cada um.

### WNDWI-B14 — As abas ficam em baixo
Em cada grupo da doca, a barra das abas fica por baixo dos terminais (o dono pediu-a assim a
2026-10-17). Arrastar e largar as abas funciona como em cima.

### WNDWI-B10 — O botão de aba nova
O botão "+" da barra abre o escolhedor do tipo da aba nova (`NWTBS`), com os tipos que a janela sabe
abrir (`KINDS`): o terminal, a árvore de ficheiros (`WNDWI-B18`), o editor vazio e um ficheiro do
computador (`WNDWI-B19`). Escolher o terminal abre um terminal novo numa aba da doca, com o foco.

### WNDWI-B11 — O fim de uma shell tira a aba dela
Quando a shell de um terminal termina, a aba dele sai da doca. Quando sai a última, a janela fecha.

### WNDWI-B13 — Fechar a aba termina a shell
Fechar a aba de um terminal na doca termina a shell dele. Fechar a última aba fecha a janela.

### WNDWI-B12 — A organização volta
Ao abrir, quando o ficheiro da organização tem uma, a doca fica igual a como ficou (as abas, os
grupos, as divisões e os tamanhos), cada terminal arranca uma shell nova, cada árvore volta com a pasta
dela e cada editor com o ficheiro dele (um editor sem ficheiro volta vazio). Sem organização, a doca
abre com um terminal. Cada mudança da doca grava a organização, `SAVE_DELAY` depois da última mudança.

### WNDWI-B15 — O botão de configuração
O botão de configuração da barra (a roda dentada, ao lado do "+") abre a aba da configuração na doca,
com o foco. A janela só tem uma: se já está aberta, o botão mostra-a. A aba da configuração também
volta com a organização (`WNDWI-B12`).

### WNDWI-B16 — Os atalhos da janela
As ações da janela fazem-se pelos atalhos do mapa da aplicação (`Keymap`), em qualquer aba: a aba
nova abre o escolhedor (como o "+"), fechar a aba fecha a aba que tem o foco (o terminal termina
a shell, como no `WNDWI-B13`), e a configuração abre-a (como o botão). Fechar a última aba fecha a
janela.

### WNDWI-B17 — Fechar o escolhedor sem escolher
Fechar o escolhedor sem escolher (o Esc, ou carregar fora dele) deixa a doca como estava e devolve o
foco à aba que o tinha. Pedir outra aba nova com o escolhedor aberto não abre um segundo.

### WNDWI-B18 — A árvore de ficheiros
O tipo "Árvore de ficheiros" abre a árvore (`FLTRE`) da pasta da janela numa aba, com o foco. Um
ficheiro aberto na árvore abre-se num editor (`EDTRS`) numa aba nova; se já está aberto num editor, a
doca mostra esse.

### WNDWI-B19 — O editor
O tipo "Editor vazio" abre um editor sem ficheiro numa aba, com o foco. O tipo "Abrir um ficheiro…"
pergunta qual pelo diálogo do sistema e abre-o num editor (como no `WNDWI-B18`); cancelar o diálogo
deixa a doca como estava.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Prefs` | as preferências do `prefs::read` | — | o `Prefs`: nunca falta nenhum valor (os de omissão) |
| `NewTerminal` | a forma de fazer um terminal novo | um terminal que não arranca | esta unidade: o aviso (`WNDWI-E02`, `WNDWI-E04`) |
| `root` | a pasta onde o Polvo abriu | uma pasta que não se lê | a árvore (`FLTRE`): o motivo no lugar do conteúdo |
| `LayoutFile` | o ficheiro da organização, ou nenhum | um ficheiro que não se grava | esta unidade: o aviso (`WNDWI-E03`) |
| `Keymap` | o mapa de atalhos da aplicação, ou nenhum (os de omissão) | — | o `Keymap`: cada ação tem um atalho |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `WNDWI-E01` | o sistema não abre a janela (sem ecrã, sem GPU) @no-scenario: a plataforma de teste abre sempre a janela | o aviso com o motivo na saída de erro, e a aplicação termina |
| `WNDWI-E02` | o terminal novo escolhido não arranca | o aviso com o motivo na saída de erro, e a doca fica como estava |
| `WNDWI-E03` | a organização não se grava | o aviso com o motivo na saída de erro, e a janela continua |
| `WNDWI-E04` | ao abrir, nenhum terminal arranca | o aviso com o motivo na saída de erro, e a janela fecha; um terminal restaurado que não arranca fica na aba com o motivo escrito nele |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `TITLE_H` | 38 px | a altura da barra do GNOME (libadwaita) |
| `KINDS` | `[TERMINAL, FILES, EDITOR, OPEN_FILE]` | os tipos de aba que o escolhedor mostra (`WNDWI-B10`), pela ordem do mais usado |
| `SAVE_DELAY` | 500 ms | um arrasto muda a doca muitas vezes seguidas; grava-se uma vez, no fim |

## Rule uses

| Rule | Uses |
| --- | --- |
| `WNDWI-B01` | `Prefs`, `TITLE_H` |
| `WNDWI-B02` | `Prefs` |
| `WNDWI-B07` | `Prefs` |
| `WNDWI-B09` | `Prefs` |
| `WNDWI-B10` | `NewTerminal`, `KINDS` |
| `WNDWI-B14` | `Prefs` |
| `WNDWI-B11` | `NewTerminal` |
| `WNDWI-E01` | `Prefs` @no-scenario: a plataforma de teste abre sempre a janela |
| `WNDWI-B12` | `LayoutFile`, `NewTerminal`, `SAVE_DELAY` |
| `WNDWI-B13` | `NewTerminal` |
| `WNDWI-E02` | `NewTerminal` |
| `WNDWI-E03` | `LayoutFile` |
| `WNDWI-E04` | `NewTerminal` |
| `WNDWI-B15` | `Prefs`, `LayoutFile` |
| `WNDWI-B16` | `Keymap`, `NewTerminal` |
| `WNDWI-B17` | `Keymap` |
| `WNDWI-B18` | `root` |
| `WNDWI-B19` | `KINDS` |

## Revisions

> **WNDWI-R0001:** a janela passa a ter a doca (plano JNNTJ, fase W02). As regras do terminal saem para o
> painel do terminal (a spec `panel.spec.md`): o `WNDWI-B03` (o teclado), o `WNDWI-B04` (copiar e
> colar), o `WNDWI-B05` (o rato e a roda) e o `WNDWI-B06` (o terminal enche a janela, agora o painel). O
> `WNDWI-B07` fica só com as cores da barra (a fonte também passou para o painel). O `WNDWI-B08` (o fim
> da shell fecha a janela) passa a `WNDWI-B11`: sai a aba, e a janela fecha com a última. Entram o
> `WNDWI-B09` (a doca) e o `WNDWI-B10` (o terminal novo).

> **WNDWI-R0002:** entra o `WNDWI-B12` (a organização guardada e restaurada) com o `WNDWI-E03` e o
> `WNDWI-E04`, e o `new` passa a receber o ficheiro da organização em vez do primeiro terminal. Entra o
> `WNDWI-B13`: a aba fechada pelo utilizador deixava a shell a correr, porque a janela guardava o
> painel.

> **WNDWI-R0003:** entra o `WNDWI-B14` (as abas em baixo), pedido pelo dono depois de experimentar a doca.

> **WNDWI-R0004:** entram o `WNDWI-B15` (o botão de configuração) e o `WNDWI-B16` (os atalhos da janela),
> do plano CEACN, fase W01. A janela fecha com a última aba, seja de terminal ou da configuração.

> **WNDWI-R0005:** o `WNDWI-B10` passa a abrir o escolhedor do tipo da aba nova em vez de um terminal, e o
> atalho de aba nova do `WNDWI-B16` também (plano CEACN, fase W02). Entra o `WNDWI-B17`: fechar o
> escolhedor sem escolher.

> **WNDWI-R0006:** o escolhedor ganha a árvore de ficheiros, o editor vazio e abrir um ficheiro do
> computador (plano CEACN, fase W03): entram o `WNDWI-B18` e o `WNDWI-B19`, o `WNDWI-B12` passa a guardar
> a pasta das árvores e o ficheiro dos editores, e o `new` passa a receber a pasta da janela.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
