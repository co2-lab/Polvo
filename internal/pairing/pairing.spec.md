<!-- @anchors
  code: PRNGS
  updated_at: 2026-10-18
-->
# Pairing — a primeira ligação entre duas instalações, autorizada por um código curto

> **Code**: `PRNGS`

## Overview
Uma instalação (o cliente) liga-se pela primeira vez a um nó que ainda não a conhece. O nó deixa-a entrar
só para emparelhar, e os dois correm o protocolo deste módulo por esse canal. O nó (o `Pairer`, que o `NewPairer` faz sobre o ficheiro das chaves autorizadas, pelo
`Serve`) mostra o pedido (`Request`: o nome da instalação, a impressão digital da chave dela e o código); o
utilizador lê o código no ecrã do nó e escreve-o no cliente (`Pair`, que pede o código pelo `Typed`). O
`OnDone` do nó sabe quando cada pedido acabou, bem ou mal.
Quando o código bate, o nó junta a chave do cliente às chaves autorizadas, e o cliente fica a conhecer a
chave do nó.

O código (`Code`) tem `CODE_DIGITS` dígitos e é calculado nos dois lados a partir das duas chaves e de dois
números aleatórios, o do cliente trocado com compromisso: um intermediário entre os dois daria códigos
diferentes, e o cliente, que confere o código escrito com o que calculou, não o envia.

## Rules

### PRNGS-B01 — O pedido
O cliente apresenta a chave e o nome; o nó mostra o pedido com o nome, a impressão digital e o código.

### PRNGS-B02 — O código
O código é o mesmo nos dois lados quando as chaves de um são as que o outro vê: os primeiros bytes do
SHA-256 das duas chaves e dos dois números, em `CODE_DIGITS` dígitos. O cliente compromete-se com o seu
número (o hash dele) antes de ver o do nó, e só o revela depois.

### PRNGS-B03 — Emparelhar
Com o código certo escrito no cliente, o nó junta a chave do cliente às chaves autorizadas, com o nome
dele, e os dois acabam com sucesso; daí em diante o cliente entra no nó como qualquer chave autorizada.

### PRNGS-B04 — O cliente confere o código
Um código escrito que não é o que o cliente calculou não chega ao nó: o cliente desiste com o erro de
código errado, e o nó não junta nada.

### PRNGS-B05 — O fim de um pedido
Quando um pedido mostrado acaba, o nó avisa o `OnDone` com o pedido e o erro (nenhum, quando emparelhou).

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `code` | os `CODE_DIGITS` dígitos que o nó mostrou | outros | este módulo: o erro (`PRNGS-E01`) |
| `message` | as mensagens do protocolo, uma por linha, em JSON | outra coisa, ou o número que não bate com o compromisso | este módulo: o erro (`PRNGS-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `PRNGS-E01` | o código escrito não é o do nó | `ErrWrongCode` nos dois lados, e nada se junta |
| `PRNGS-E02` | uma mensagem fora do protocolo, ou um número que não bate com o compromisso | o nó desiste com o erro, e nada se junta |
| `PRNGS-E03` | mais de `MAX_FAILURES` falhas em `FAILURE_WINDOW` | o nó recusa os pedidos seguintes até passar a janela (`ErrTooMany`) |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `CODE_DIGITS` | 6 | o que se lê e escreve sem enganos; um intermediário acerta 1 vez num milhão |
| `MAX_FAILURES` | 5 | quem tenta adivinhar gasta as tentativas depressa |
| `FAILURE_WINDOW` | 1 min | as tentativas voltam ao fim de um minuto, para quem se enganou |

## Rule uses

| Rule | Uses |
| --- | --- |
| `PRNGS-B01` | `message` |
| `PRNGS-B02` | `code`, `CODE_DIGITS` |
| `PRNGS-B03` | `code` |
| `PRNGS-B04` | `code` |
| `PRNGS-B05` | `message` |
| `PRNGS-E01` | `code` |
| `PRNGS-E02` | `message` |
| `PRNGS-E03` | `MAX_FAILURES`, `FAILURE_WINDOW` |

## Revisions

> **PRNGS-R0001:** entra o `PRNGS-B05` (o fim de um pedido), para a janela tirar o pedido do ecrã.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
