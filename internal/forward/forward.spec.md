<!-- @anchors
  code: FRWRD
  updated_at: 2026-10-18
-->
# Forward — os sockets encaminhados entre duas máquinas, pelo nó

> **Code**: `FRWRD`

## Overview
Um programa de uma máquina fala com um serviço de outra por um socket encaminhado pelo Polvo, nos dois
sentidos, como o `ssh -L` e o `ssh -R`. O `Local` escuta nesta máquina e liga cada ligação a um socket da
máquina do nó; o `Remote` põe o nó a escutar na máquina dele e liga cada ligação a um socket desta. Os
sockets (`Addr`) são TCP ou Unix, e o `Parse` lê-os do texto (o `String` dá-o de volta). O nó é qualquer coisa que se ligue e escute
pela máquina dele (`Via`, que o `hub.Node` é). Um encaminhamento (`Forward`) diz onde escuta (`Addr`) e
acaba com o `Close`.

## Rules

### FRWRD-B01 — O texto de um socket
O `Parse` lê `unix:` seguido do caminho como um socket Unix, e `máquina:porta` como TCP; só a porta (com
ou sem os dois pontos) é a porta em `DEFAULT_HOST`.

### FRWRD-B02 — Daqui para lá
O `Local` escuta no socket desta máquina e liga cada ligação que lá chega, pelo nó, ao socket da máquina
dele; os bytes passam nos dois sentidos até um dos lados acabar.

### FRWRD-B03 — De lá para cá
O `Remote` põe o nó a escutar no socket da máquina dele e liga cada ligação que lá chega ao socket desta
máquina, da mesma forma.

### FRWRD-B04 — Acabar
O `Close` deixa de escutar (o nó também, no `Remote`); as ligações que já passavam acabam com isso.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `text` | `unix:/caminho`, `máquina:porta`, `:porta` ou `porta` | outro texto, ou uma porta que não é um número de 0 a 65535 | este módulo: o erro (`FRWRD-E01`) |
| `listen` | um socket onde se pode escutar (TCP com a porta 0 escolhe uma) | um ocupado, ou sem permissão | este módulo: o erro (`FRWRD-E02`) |
| `target` | um socket que aceita ligações | um que não responde | este módulo: essa ligação fecha (`FRWRD-E03`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `FRWRD-E01` | o texto não é de um socket | o `Parse` devolve o erro com o texto |
| `FRWRD-E02` | não se pode escutar no socket, ou o nó recusa (sem a permissão de encaminhar) | o `Local` ou o `Remote` devolve o erro, e nada fica a escutar |
| `FRWRD-E03` | o socket de destino não responde | a ligação que chegou fecha-se, e o encaminhamento continua para as seguintes |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULT_HOST` | `127.0.0.1` | só a porta escuta só para esta máquina: abrir um encaminhamento à rede é escolha explícita |

## Rule uses

| Rule | Uses |
| --- | --- |
| `FRWRD-B01` | `text`, `DEFAULT_HOST` |
| `FRWRD-B02` | `listen`, `target` |
| `FRWRD-B03` | `listen`, `target` |
| `FRWRD-B04` | `listen` |
| `FRWRD-E01` | `text` |
| `FRWRD-E02` | `listen` |
| `FRWRD-E03` | `target` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
