<!-- @anchors
  code: WNDWI
  updated_at: 2026-10-17
-->
# Window — a janela GPUI do terminal

> **Code**: `WNDWI`

## Overview
A janela do Polvo é uma vista GPUI (`TerminalView`) desenhada na GPU: a barra de título por cima e o
terminal (`Screen`) por baixo. A janela não interpreta o terminal: desenha o que o `Screen` lhe dá e
passa-lhe o teclado, o rato, o que se cola e o tamanho. Segue as preferências do ambiente de trabalho
(`Prefs`): a ordem dos botões, o tema e a fonte.

No GNOME com Wayland o sistema não desenha a moldura das janelas: a janela desenha a sua, sem sombra
nem margem à volta (o dono aprovou-a assim a 2026-10-17).

O `TerminalView::new` recebe o `Screen`, o canal dos acontecimentos da shell (quando há shell) e as
preferências; o `open` abre a janela com ela.

## Rules

### WNDWI-B01 — A barra segue o sistema
A barra de título tem os botões de fechar, minimizar e maximizar nos lados e pela ordem das
preferências, só os que o sistema da janela suporta (fechar está sempre), e o título "Polvo" ao centro.

### WNDWI-B02 — Os botões fazem o que dizem
Fechar fecha a janela, minimizar minimiza-a e maximizar alterna entre maximizada e o tamanho anterior.

### WNDWI-B03 — O teclado vai para a shell
Uma tecla na janela vai para a shell como o `Keys` a traduz e a libghostty-vt a codifica. Os atalhos de
copiar e colar (`WNDWI-B04`) não vão.

### WNDWI-B04 — Copiar e colar
- Ctrl+Shift+C copia a seleção para a área de transferência.
- Ctrl+Shift+V cola a área de transferência na shell.
- Selecionar com o rato copia também para a seleção primária do sistema, e o botão do meio cola-a.

### WNDWI-B05 — O rato seleciona e a roda faz scroll
Carregar, arrastar e soltar o botão esquerdo sobre o terminal são os gestos de seleção do `Screen`,
em píxeis a partir do canto do terminal. A roda move a vista pelo histórico, três linhas por passo.

### WNDWI-B06 — O terminal enche a janela
O terminal ocupa a janela por baixo da barra, menos a margem `PADDING` à volta. As linhas e as colunas
são o que cabe dessa área em células da fonte, e mudam quando a janela muda de tamanho.

### WNDWI-B07 — O tema e a fonte seguem as preferências
As cores da janela (`Colors`) são as do tema: a barra é escura no tema escuro e clara no claro. O
terminal usa a fonte e o tamanho das preferências.

### WNDWI-B08 — O fim da shell fecha a janela
Quando a shell termina, a janela fecha.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Prefs` | as preferências do `prefs::read` | — | o `Prefs`: nunca falta nenhum valor (os de omissão) |
| `key` | qualquer tecla que a janela recebe | — | o `Keys`: uma tecla desconhecida passa o carácter |
| `ClipboardItem` | o texto da área de transferência ou da seleção primária | vazia, ou sem texto | esta unidade: colar não manda nada |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `WNDWI-E01` | o sistema não abre a janela (sem ecrã, sem GPU) @no-scenario: a plataforma de teste abre sempre a janela | o aviso com o motivo na saída de erro, e a aplicação termina |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `TITLE_H` | 38 px | a altura da barra do GNOME (libadwaita) |
| `PADDING` | 8 px | a margem do terminal, a do Ghostty por omissão é parecida |
| `SCROLL_LINES` | 3 | as linhas por passo da roda, como na maioria dos terminais |

## Rule uses

| Rule | Uses |
| --- | --- |
| `WNDWI-B01` | `Prefs`, `TITLE_H` |
| `WNDWI-B02` | `Prefs` |
| `WNDWI-B03` | `key` |
| `WNDWI-B04` | `key`, `ClipboardItem` |
| `WNDWI-B05` | `SCROLL_LINES` |
| `WNDWI-B06` | `TITLE_H`, `PADDING`, `Prefs` |
| `WNDWI-B07` | `Prefs` |
| `WNDWI-B08` | `Prefs` |
| `WNDWI-E01` | `Prefs` @no-scenario: a plataforma de teste abre sempre a janela |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
