<!-- @anchors
  code: STPNS
  updated_at: 2026-10-17
-->
# Settings panel — a aba da configuração

> **Code**: `STPNS`

## Overview
A configuração da janela é uma aba da doca (`SettingsPanel`): as secções à esquerda e a secção aberta à
direita. A primeira secção são os atalhos de teclado, onde o utilizador muda o atalho de cada ação do
mapa de atalhos (o `Keymap` da aplicação). O que muda grava-se no ficheiro da configuração e vale logo.

## Rules

### STPNS-B01 — As secções, com os atalhos abertos
A aba mostra as secções à esquerda; a dos atalhos de teclado está aberta.

### STPNS-B02 — Cada ação com o atalho dela
A secção dos atalhos mostra cada ação do mapa com o nome e o atalho atual.

### STPNS-B03 — Mudar um atalho
Carregar no atalho de uma ação põe-no à escuta; as teclas premidas a seguir passam a ser o atalho
dela. O mapa da aplicação muda logo, e o que difere dos de omissão grava-se no ficheiro.

### STPNS-B04 — Cancelar, e os modificadores sozinhos
À escuta, o Esc cancela e o atalho fica como estava; premir só modificadores (Ctrl, Alt, Shift) continua
à escuta, à espera da tecla.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `keystroke` | as teclas premidas à escuta | o atalho de outra ação | este módulo: a mensagem (`STPNS-E01`) |
| `SettingsFile` | o ficheiro da configuração, ou nenhum | um que não se grava | este módulo: a mensagem (`STPNS-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `STPNS-E01` | as teclas são o atalho de outra ação | a mensagem diz de que ação são, e nada muda |
| `STPNS-E02` | a configuração não se grava | a mensagem diz porquê; o atalho novo vale até a janela fechar |

## Rule uses

| Rule | Uses |
| --- | --- |
| `STPNS-B01` | `keystroke` |
| `STPNS-B02` | `keystroke` |
| `STPNS-B03` | `keystroke`, `SettingsFile` |
| `STPNS-B04` | `keystroke` |
| `STPNS-E01` | `keystroke` |
| `STPNS-E02` | `SettingsFile` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
