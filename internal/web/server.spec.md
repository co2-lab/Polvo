<!-- @anchors
  code: WEBSR
  updated_at: 2026-10-17
-->
# Server — o servidor HTTP local da UI: os ficheiros, as chamadas aos nós e os terminais

> **Code**: `WEBSR`

## Overview
A UI do Polvo corre no browser (até o Galaxy ter a janela nativa) e fala com este servidor, que corre
na máquina local. O servidor dá os ficheiros da UI, as chamadas de controlo aos nós e os terminais por
WebSocket. Quem fala com ele chega aos nós, que são shells noutras máquinas. Por isso só aceita pedidos
locais, e as chamadas exigem o segredo da sessão (`Token`), que o `polvo` dá ao browser que abre.

A configuração é o `Config`: a configuração do cliente dos nós, os ficheiros da UI e o segredo (vazio
para nascer um aleatório). O servidor nasce pelo `New`, responde pelo `Handler` e fecha as ligações aos
nós pelo `Close`.

## Rules

### WEBSR-B01 — Os ficheiros da UI servem-se sem segredo
Um `GET` a um caminho da UI (a raiz dá o `index.html`) devolve o ficheiro, sem pedir o segredo: os
ficheiros não guardam nada que se possa usar sem ele. Os ficheiros vêm embutidos no executável (`UI`).

### WEBSR-B02 — A chamada de controlo leva o segredo e devolve o resultado
Um `POST` a `/api/call` com o segredo no cabeçalho `X-Polvo-Token` e o corpo com o nó, o método e os
parâmetros faz a chamada de controlo ao nó e devolve o resultado. Uma recusa do nó volta como objeto
de erro com o código e a mensagem.

### WEBSR-B03 — Sem o segredo certo, nada se faz
Uma chamada a `/api/call` ou a `/ws/term` sem o segredo, ou com outro, é recusada (401) antes de tocar
em qualquer nó.

### WEBSR-B04 — O terminal por WebSocket
O WebSocket `/ws/term`, com o segredo e o nó na consulta e o tamanho (linhas e colunas), abre uma shell
nesse nó.
- A saída da shell chega em mensagens binárias.
- O browser envia mensagens de texto em JSON: a entrada (`input`) ou o novo tamanho (`resize`).
- Quando a shell termina, chega uma mensagem de texto com o código de saída (ou o sinal) e a ligação
  fecha.

### WEBSR-B05 — O WebSocket de outra origem é recusado
Um WebSocket cuja origem não é a do próprio servidor é recusado (403), mesmo com o segredo.

### WEBSR-B06 — Só pedidos locais
Um pedido cujo destino (o `Host`) não é um endereço local (`127.0.0.1`, `::1`, `localhost`) é recusado
(403). É a defesa contra o DNS rebinding: uma página de outro sítio que aponte um nome seu para o
endereço local.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Host` | um endereço local com a porta | qualquer outro | este módulo: 403 (`WEBSR-B06`) |
| `X-Polvo-Token` | o segredo da sessão | vazio ou outro | este módulo: 401 (`WEBSR-B03`) |
| `node` | o endereço de um nó | vazio | este módulo: o objeto de erro (`WEBSR-E01`) |
| `method` | um método de controlo do nó | outro | o nó: o erro de método não encontrado, devolvido como objeto de erro |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `WEBSR-E01` | o nó não responde, falta o nó, ou o nó recusa | o objeto de erro com a mensagem (na chamada), ou a mensagem de texto de erro e o fecho (no terminal) |
| `WEBSR-E02` | o corpo da chamada não é JSON ou não é um `POST` | 400 ou 405, sem tocar em nenhum nó |

## Rule uses

| Rule | Uses |
| --- | --- |
| `WEBSR-B01` | `Host` |
| `WEBSR-B02` | `X-Polvo-Token`, `node`, `method` |
| `WEBSR-B03` | `X-Polvo-Token` |
| `WEBSR-B04` | `X-Polvo-Token`, `node` |
| `WEBSR-B05` | `Host` |
| `WEBSR-B06` | `Host` |
| `WEBSR-E01` | `node` |
| `WEBSR-E02` | `method` |

## Revisions

> **WEBSR-R0001:** o `UI` dá os ficheiros embutidos no executável (o bundle da UI do plano DSKTP-W02), que
> o `polvo` passa ao `Config.Files`. As regras não mudam.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
