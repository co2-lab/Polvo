<!-- @anchors
  code: TRPNT
  updated_at: 2026-10-18
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
Os atalhos do mapa de atalhos (o `Keymap` da aplicação) não vão: os de copiar e colar são do painel
(`TRPNT-B02`), e os das ações da janela (aba nova, fechar a aba, configuração) seguem para a janela.
O de gravar é do editor: no terminal vai para a shell como as outras teclas.

### TRPNT-B02 — Copiar e colar
- O atalho de copiar (por omissão Ctrl+Shift+C) copia a seleção para a área de transferência.
- O atalho de colar (por omissão Ctrl+Shift+V) cola a área de transferência na shell.
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

### TRPNT-B10 — O terminal de uma sessão
O `session` faz o painel de uma sessão de uma máquina (`SessionTab`): o ecrã é o dela, e o que se escreve
vai para ela. Quando a janela fez a sessão para esta aba (a dona), fechar a aba termina-a, como a shell
de um terminal local. Quando a aba se ligou a uma sessão que já existia, fechar a aba só a larga, e a
organização da doca guarda a máquina e a sessão, para a aba voltar ligada a ela.

### TRPNT-B11 — Só ver
O painel de uma sessão onde a janela não pode escrever mostra "Só ver", e as teclas e o colar não vão
para a sessão; copiar e os atalhos da janela continuam.

### TRPNT-B12 — A marca do agente na aba
O título da aba de uma sessão onde corre um agente começa pela marca dele (`set_mark`): a trabalhar, à
espera de uma resposta, bloqueado, ou terminado.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Prefs` | as preferências do `prefs::read` | — | o `Prefs`: nunca falta nenhum valor (os de omissão) |
| `key` | qualquer tecla que o painel recebe | — | o `Keys`: uma tecla desconhecida passa o carácter |
| `ClipboardItem` | o texto da área de transferência ou da seleção primária | vazia, ou sem texto | esta unidade: colar não manda nada |
| `SessionTab` | uma sessão viva de uma máquina juntada, e se a janela é dona dela e pode escrever | uma que acabou ou não existe | o `Station`: o erro dele volta a quem pediu o painel |

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
| `TRPNT-B10` | `SessionTab` |
| `TRPNT-B11` | `SessionTab`, `key` |
| `TRPNT-B12` | `SessionTab` |

## Revisions

> **TRPNT-R0001:** entra o `TRPNT-B09` (o painel de um terminal que não arrancou), para a organização
> restaurada da doca: a doca reconstrói cada aba por um construtor que tem de dar sempre um painel, e um
> terminal cuja shell já não arranca fica com o motivo à vista em vez de desaparecer em silêncio.

> **TRPNT-R0002:** copiar e colar passam a ser os atalhos do mapa de atalhos (plano CEACN, W01), que o
> utilizador muda; os atalhos das ações da janela deixam de ir para a shell.

> **TRPNT-R0003:** o atalho de gravar do mapa (o Ctrl+S, do editor) vai para a shell (plano CEACN, W03).

> **TRPNT-R0005:** entra o `TRPNT-B12` (a marca do agente na aba), do plano AGPRG, W04.

> **TRPNT-R0004:** entram o `TRPNT-B10` (o terminal de uma sessão, dona ou não) e o `TRPNT-B11` (só ver),
> para as abas das sessões desta máquina e das outras (plano ESRMS, W02).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
