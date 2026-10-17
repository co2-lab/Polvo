<!-- @anchors
  code: KYMPE
  updated_at: 2026-10-17
-->
# Keymap — as ações da janela e o atalho de teclado de cada uma

> **Code**: `KYMPE`

## Overview
A janela tem ações que se fazem pelo teclado (`Action`): abrir uma aba nova, fechar a aba, copiar,
colar, abrir a configuração. O `Keymap` diz o atalho de cada uma e que ação um atalho faz. O utilizador
muda os atalhos na configuração; o que difere dos de omissão grava-se, e volta a aplicar-se ao abrir.
Um atalho recusado vem com o motivo (`KeymapError`).

Um atalho escreve-se como o GPUI o escreve: os modificadores e a tecla separados por hífens
(`ctrl-shift-t`). O `normalize` dá a forma única de um atalho.

## Rules

### KYMPE-B01 — Os atalhos por omissão
Sem nada mudado, os atalhos são os de `DEFAULTS`.

### KYMPE-B02 — A forma única de um atalho
A ordem dos modificadores e as maiúsculas não contam: `shift-ctrl-T` e `ctrl-shift-t` são o mesmo
atalho. Os modificadores são `ctrl`, `alt`, `shift` e `cmd` (a tecla do sistema; `super` e `win` são o
mesmo).

### KYMPE-B03 — A ação de um atalho
O `action_for` dá a ação que um atalho faz, ou nenhuma.

### KYMPE-B04 — Mudar um atalho
O `set` dá a uma ação um atalho novo. O atalho que já é de outra ação é recusado (`KYMPE-E01`), e o
mapa fica como estava.

### KYMPE-B05 — Gravar e aplicar o que mudou
O `overrides` dá os atalhos que diferem dos de omissão, pelo id de cada ação; o `with_overrides` aplica-os
sobre os de omissão.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `keystroke` | modificadores e uma tecla, separados por hífens | vazio, só modificadores, um modificador desconhecido | este módulo: o erro (`KYMPE-E02`) |
| `overrides` | o id de uma ação e um atalho | um id desconhecido ou um atalho inválido | este módulo: ignora-se com o aviso (`KYMPE-E03`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `KYMPE-E01` | o atalho novo já é de outra ação | o `set` recusa-o com a ação que o tem, e o mapa fica como estava |
| `KYMPE-E02` | o atalho não é um atalho | o `normalize` e o `set` recusam-no com o motivo |
| `KYMPE-E03` | um atalho gravado é de uma ação que não existe, inválido, ou de outra ação | ignora-se, com o aviso na saída de erro, e fica o de omissão |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULTS` | aba nova `ctrl-shift-t`, fechar aba `ctrl-shift-w`, copiar `ctrl-shift-c`, colar `ctrl-shift-v`, configuração `ctrl-,` | os dos terminais do Linux (GNOME Terminal, Ghostty): o Ctrl sem Shift é da shell |

## Rule uses

| Rule | Uses |
| --- | --- |
| `KYMPE-B01` | `DEFAULTS` |
| `KYMPE-B02` | `keystroke` |
| `KYMPE-B03` | `keystroke` |
| `KYMPE-B04` | `keystroke` |
| `KYMPE-B05` | `overrides`, `DEFAULTS` |
| `KYMPE-E01` | `keystroke` |
| `KYMPE-E02` | `keystroke` |
| `KYMPE-E03` | `overrides` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
