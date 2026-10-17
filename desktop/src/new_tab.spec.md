<!-- @anchors
  code: NWTBS
  updated_at: 2026-10-17
-->
# New tab — o escolhedor do tipo da aba nova

> **Code**: `NWTBS`

## Overview
O escolhedor (`NewTabChooser`) aparece por cima da doca quando o utilizador pede uma aba nova (o botão
"+" ou o atalho). Mostra os tipos de aba que a janela sabe abrir (`TabKind`), um por linha, com o nome
e a tecla do número, e o primeiro escolhido. O que o utilizador fez vai para a janela num `ChooserEvent`:
escolher um tipo diz-lhe qual (`Chosen`); fechar sem escolher di-lo também (`Dismissed`). Os tipos que a janela sabe abrir crescem com o plano CEACN: o
terminal primeiro, e os ficheiros, o editor e o git nas fases seguintes.

## Rules

### NWTBS-B01 — Os tipos, com o primeiro escolhido
O escolhedor mostra os tipos de aba que recebeu, pela ordem em que os recebeu, cada um com o nome e o
número (de 1 em diante); o primeiro está escolhido, e o escolhedor tem o foco.

### NWTBS-B02 — Escolher só com o teclado
As setas para cima e para baixo mudam o escolhido (sem passar do primeiro nem do último), e o Enter
escolhe-o. A tecla do número de um tipo escolhe-o logo.

### NWTBS-B03 — Escolher com o rato
Carregar num tipo escolhe-o.

### NWTBS-B04 — Fechar sem escolher
O Esc fecha o escolhedor sem escolher nada.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `TabKind` | os tipos que a janela sabe abrir, pelo menos um | nenhum | a janela: o terminal está sempre |
| `keystroke` | as teclas premidas com o escolhedor aberto | um número sem tipo | este módulo: não faz nada |

## Rule uses

| Rule | Uses |
| --- | --- |
| `NWTBS-B01` | `TabKind` |
| `NWTBS-B02` | `TabKind`, `keystroke` |
| `NWTBS-B03` | `TabKind` |
| `NWTBS-B04` | `keystroke` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
