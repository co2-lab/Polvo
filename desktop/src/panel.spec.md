<!-- @anchors
  code: TRPNT
  updated_at: 2026-10-17
-->
# Panel — um terminal como painel da doca

> **Code**: `TRPNT`

## Overview
O `TerminalPanel` é um terminal (`Screen`) dentro da doca da janela: uma aba que o utilizador arrasta
para outro grupo ou para uma borda, ao lado de outros terminais. O painel desenha o ecrã, passa-lhe o
teclado, o rato, o que se cola e o tamanho, e avisa (`TerminalEvent`) quando a shell termina, para a
janela o tirar da doca.

O `TerminalPanel::new` recebe o `Screen`, o canal dos acontecimentos da shell (quando há shell), o nome
da shell e as preferências (`Prefs`); o `spawn` arranca a shell do utilizador e faz o painel dela.

## Rules

### TRPNT-B01 — O teclado vai para a shell
Uma tecla no painel com o foco vai para a shell como o `Keys` a traduz e a libghostty-vt a codifica.
Os atalhos de copiar e colar (`TRPNT-B02`) não vão.

### TRPNT-B02 — Copiar e colar
- Ctrl+Shift+C copia a seleção para a área de transferência.
- Ctrl+Shift+V cola a área de transferência na shell.
- Selecionar com o rato copia também para a seleção primária do sistema, e o botão do meio cola-a.

### TRPNT-B03 — O rato seleciona e a roda faz scroll
Carregar, arrastar e soltar o botão esquerdo sobre o terminal são os gestos de seleção do `Screen`,
em píxeis a partir do canto do terminal. A roda move a vista pelo histórico, `SCROLL_LINES` linhas por
passo.

### TRPNT-B04 — O terminal enche o painel
O terminal ocupa o painel menos a margem `PADDING` à volta. As linhas e as colunas são o que cabe dessa
área em células da fonte, e mudam quando o painel muda de tamanho (a janela, ou o divisor da doca).

### TRPNT-B05 — A fonte segue as preferências
O terminal usa a fonte e o tamanho das preferências; as cores do ecrã são as do terminal.

### TRPNT-B06 — A aba diz o título da shell
O título da aba é o título que a shell deu ao terminal (o `title` do `Screen`), ou o nome da shell
enquanto ela não deu nenhum.

### TRPNT-B07 — O fim da shell avisa
Quando a shell termina, o painel emite o `TerminalEvent::Exited`.

### TRPNT-B08 — Clicar dá o foco
Carregar no terminal dá-lhe o foco: as teclas seguintes vão para a shell dele, e não para a de outro
painel.

### TRPNT-B09 — Um terminal que não arrancou diz porquê
O `failed` faz um painel sem shell que mostra no ecrã o motivo de a shell não ter arrancado: a janela
usa-o quando um terminal restaurado não arranca, e a aba fica com o motivo à vista até o utilizador a
fechar.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Prefs` | as preferências do `prefs::read` | — | o `Prefs`: nunca falta nenhum valor (os de omissão) |
| `key` | qualquer tecla que o painel recebe | — | o `Keys`: uma tecla desconhecida passa o carácter |
| `ClipboardItem` | o texto da área de transferência ou da seleção primária | vazia, ou sem texto | esta unidade: colar não manda nada |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `TRPNT-E01` | a shell do utilizador não arranca no `spawn` | o erro do `Screen`, com o programa e o motivo, volta a quem pediu o terminal, e não nasce painel |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `PADDING` | 8 px | a margem do terminal, a do Ghostty por omissão é parecida |
| `SCROLL_LINES` | 3 | as linhas por passo da roda, como na maioria dos terminais |
| `SHELL_FALLBACK` | /bin/sh | a shell quando o `SHELL` falta |

## Rule uses

| Rule | Uses |
| --- | --- |
| `TRPNT-B01` | `key` |
| `TRPNT-B02` | `key`, `ClipboardItem` |
| `TRPNT-B03` | `SCROLL_LINES` |
| `TRPNT-B04` | `PADDING`, `Prefs` |
| `TRPNT-B05` | `Prefs` |
| `TRPNT-B06` | `Prefs` |
| `TRPNT-B07` | `Prefs` |
| `TRPNT-B08` | `key` |
| `TRPNT-B09` | `Prefs` |
| `TRPNT-E01` | `SHELL_FALLBACK` |

## Revisions

> **TRPNT-R0001:** entra o `TRPNT-B09` (o painel de um terminal que não arrancou), para a organização
> restaurada da doca: a doca reconstrói cada aba por um construtor que tem de dar sempre um painel, e um
> terminal cuja shell já não arranca fica com o motivo à vista em vez de desaparecer em silêncio.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
