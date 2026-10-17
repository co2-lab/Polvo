<!-- @anchors
  code: CDCOE
  updated_at: 2026-10-17
-->
# Codec — as mensagens de controlo entre o desktop e o nó, num fluxo de bytes

> **Code**: `CDCOE`

## Overview
O desktop, o nó e a ponte do agente local trocam mensagens de controlo por um fluxo de bytes ordenado e
fiável (um canal SSH, um pipe, um socket). O codec escreve e lê essas mensagens. É ele que decide o que
é uma mensagem, onde acaba uma e começa a seguinte, e o que acontece a bytes que não formam uma mensagem.

As mensagens seguem o JSON-RPC 2.0: o pedido (espera resposta), a notificação (não espera) e a resposta
(com resultado ou com erro). O codec não interpreta os métodos nem os parâmetros: isso é de quem o usa.

## Rules

### CDCOE-B01 — Uma mensagem por linha
Cada mensagem é um objeto JSON numa só linha, terminada por um fim de linha. O leitor entrega as
mensagens pela ordem em que chegaram, uma por linha. Uma linha vazia não é mensagem e é ignorada.
O leitor é o `Decoder` (`NewDecoder`, `NewDecoderSize`, `Read`); o escritor é o `Encoder`
(`NewEncoder`, `Write`).

### CDCOE-B02 — O tipo da mensagem vem dos campos que tem
Uma mensagem com método e com identificador é um pedido. Com método e sem identificador é uma
notificação. Sem método, com identificador e com exatamente um de resultado ou erro, é uma resposta.
Toda a mensagem declara a versão "2.0" do protocolo. A mensagem é a `Message`; o tipo dela é o
`Kind` (`Request`, `Notification`, `Response`, ou `Invalid` para o que não é nenhum dos três), e o
erro de uma resposta é o `Error`, com o código e o texto.

### CDCOE-B03 — O identificador é texto ou inteiro, e volta igual
O identificador de um pedido é um texto ou um número inteiro. A resposta leva o mesmo identificador,
com o mesmo tipo: o pedido com o identificador 7 não é respondido pelo identificador "7". O
identificador é o `ID`: criado por `IntID` ou `StringID`, lido por `Int` ou `String`, comparado por
`Equal` (valor e tipo), ausente quando `IsZero`, e impresso por `Format` tal como viaja.

### CDCOE-B04 — Escritas concorrentes não se misturam
Várias tarefas podem escrever no mesmo fluxo ao mesmo tempo. Cada mensagem chega inteira e numa
linha só: nunca há bytes de uma mensagem no meio de outra.

### CDCOE-B06 — A resposta de erro sem identificador leva o identificador nulo
Quando o erro é de uma mensagem cujo identificador não se conseguiu ler (um erro de parse), a resposta
de erro leva o identificador nulo (`NullID`, `IsNull`). Só uma resposta de erro o pode levar: um pedido
ou uma resposta com resultado com o identificador nulo não é uma mensagem.

### CDCOE-B05 — O lote não é aceite
Um lote (vários pedidos numa lista) é recusado como pedido inválido. O Polvo envia uma mensagem de cada
vez.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `line` | qualquer sequência de bytes até ao fim de linha | uma linha maior do que o limite | este codec: recusa-a (`CDCOE-E03`) e segue para a linha seguinte |
| `Message` | um pedido, uma notificação ou uma resposta bem formados | uma mensagem sem método e sem identificador, ou uma resposta com resultado e erro ao mesmo tempo | este codec: recusa escrevê-la e nada sai no fluxo (`CDCOE-E05`) |

## Invariants

| Rule | Always holds | How it is proven |
| --- | --- | --- |
| `CDCOE-I01` | escrever uma mensagem e lê-la do outro lado dá a mesma mensagem: o tipo, o identificador (com o tipo dele), o método, os parâmetros, o resultado e o erro | closed cycle: escrever pedidos, notificações e respostas e lê-los pelo leitor |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `CDCOE-E01` | a linha não é JSON válido | erro de parse (`CodeParseError`) para essa linha, como um `ReadError`; a leitura continua na linha seguinte |
| `CDCOE-E02` | a linha é JSON válido mas não é uma mensagem (B02, B05) | pedido inválido (`CodeInvalidRequest`), como um `ReadError` com o identificador quando se consegue ler; a leitura continua na linha seguinte |
| `CDCOE-E03` | a linha passa o limite de tamanho | mensagem grande demais (`ErrTooLarge`); o resto da linha é descartado e a leitura continua na linha seguinte |
| `CDCOE-E04` | o fluxo acaba a meio de uma linha | fim inesperado do fluxo; um fim entre linhas é o fim normal |
| `CDCOE-E05` | pede-se para escrever uma mensagem fora do domínio | a escrita é recusada e nada sai no fluxo |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `MaxLine` | 16 MiB | um ficheiro lido ou escrito pelo controlo cabe numa mensagem; o limite impede que um par malicioso esgote a memória do outro lado |
| `CodeParseError` | -32700 | o código do JSON-RPC 2.0 |
| `CodeInvalidRequest` | -32600 | o código do JSON-RPC 2.0 |
| `CodeMethodNotFound`, `CodeInvalidParams`, `CodeInternalError` | -32601, -32602, -32603 | os códigos do JSON-RPC 2.0 que quem usa o codec devolve nas respostas de erro |

## Rule uses

| Rule | Uses |
| --- | --- |
| `CDCOE-B01` | `line` |
| `CDCOE-B02` | `line`, `Message` |
| `CDCOE-B03` | `line`, `Message` |
| `CDCOE-B04` | `Message` |
| `CDCOE-B05` | `line`, `CodeInvalidRequest` |
| `CDCOE-B06` | `line`, `Message` |
| `CDCOE-E01` | `line`, `CodeParseError` |
| `CDCOE-E02` | `line`, `CodeInvalidRequest` |
| `CDCOE-E03` | `line`, `MaxLine` |
| `CDCOE-E04` | `line` |
| `CDCOE-E05` | `Message` |
| `CDCOE-I01` | `line`, `Message` |

## Revisions

> **CDCOE-R0001:** nasceu a `CDCOE-B06` (o identificador nulo). O controlo do nó (`CNTRL-E05`) tem de
> responder a uma linha que não é JSON, e o JSON-RPC 2.0 manda responder-lhe com `"id": null`; o codec
> não o representava. Nada do resto mudou.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
