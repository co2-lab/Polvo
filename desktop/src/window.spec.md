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

O `PolvoWindow::new` recebe as preferências, a forma de fazer os terminais (`NewTerminal`) e o ficheiro
da organização (`LayoutFile`), quando a há; o `open` abre a janela com a shell do utilizador.

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

### WNDWI-B10 — O botão de terminal novo
O botão "+" da barra abre um terminal novo numa aba da doca, com o foco.

### WNDWI-B11 — O fim de uma shell tira a aba dela
Quando a shell de um terminal termina, a aba dele sai da doca. Quando sai a última, a janela fecha.

### WNDWI-B13 — Fechar a aba termina a shell
Fechar a aba de um terminal na doca termina a shell dele. Fechar a última aba fecha a janela.

### WNDWI-B12 — A organização volta
Ao abrir, quando o ficheiro da organização tem uma, a doca fica igual a como ficou (as abas, os
grupos, as divisões e os tamanhos), e cada terminal arranca uma shell nova. Sem organização, a doca
abre com um terminal. Cada mudança da doca grava a organização, `SAVE_DELAY` depois da última mudança.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Prefs` | as preferências do `prefs::read` | — | o `Prefs`: nunca falta nenhum valor (os de omissão) |
| `NewTerminal` | a forma de fazer um terminal novo | um terminal que não arranca | esta unidade: o aviso (`WNDWI-E02`, `WNDWI-E04`) |
| `LayoutFile` | o ficheiro da organização, ou nenhum | um ficheiro que não se grava | esta unidade: o aviso (`WNDWI-E03`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `WNDWI-E01` | o sistema não abre a janela (sem ecrã, sem GPU) @no-scenario: a plataforma de teste abre sempre a janela | o aviso com o motivo na saída de erro, e a aplicação termina |
| `WNDWI-E02` | o terminal novo do botão "+" não arranca | o aviso com o motivo na saída de erro, e a doca fica como estava |
| `WNDWI-E03` | a organização não se grava | o aviso com o motivo na saída de erro, e a janela continua |
| `WNDWI-E04` | ao abrir, nenhum terminal arranca | o aviso com o motivo na saída de erro, e a janela fecha; um terminal restaurado que não arranca fica na aba com o motivo escrito nele |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `TITLE_H` | 38 px | a altura da barra do GNOME (libadwaita) |
| `SAVE_DELAY` | 500 ms | um arrasto muda a doca muitas vezes seguidas; grava-se uma vez, no fim |

## Rule uses

| Rule | Uses |
| --- | --- |
| `WNDWI-B01` | `Prefs`, `TITLE_H` |
| `WNDWI-B02` | `Prefs` |
| `WNDWI-B07` | `Prefs` |
| `WNDWI-B09` | `Prefs` |
| `WNDWI-B10` | `NewTerminal` |
| `WNDWI-B11` | `NewTerminal` |
| `WNDWI-E01` | `Prefs` @no-scenario: a plataforma de teste abre sempre a janela |
| `WNDWI-B12` | `LayoutFile`, `NewTerminal`, `SAVE_DELAY` |
| `WNDWI-B13` | `NewTerminal` |
| `WNDWI-E02` | `NewTerminal` |
| `WNDWI-E03` | `LayoutFile` |
| `WNDWI-E04` | `NewTerminal` |

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

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
