<!-- @anchors
  code: CNTRL
  updated_at: 2026-10-17
-->
# Control — as operações de controlo de um nó: ficheiros e processos

> **Code**: `CNTRL`

## Overview
O controlo é o que o desktop e o agente local pedem a um nó sem passar por uma shell: ler e escrever
um ficheiro, listar uma pasta e correr um processo até ao fim com a saída dele. Fala o protocolo de
mensagens (o codec da W01) sobre a sessão do subsistema "polvo" (`ServeControl`). Cada pedido tem
resposta; uma notificação não tem. Os dados binários (o conteúdo de um ficheiro, a entrada e a saída
de um processo) viajam em base64.

## Rules

### CNTRL-B01 — `fs.read` devolve o conteúdo de um ficheiro
O pedido `fs.read` com o caminho de um ficheiro devolve o conteúdo dele e o tamanho.

### CNTRL-B02 — `fs.write` cria ou substitui um ficheiro
O pedido `fs.write` com um caminho e um conteúdo escreve o ficheiro, criando-o com as permissões 0644
ou substituindo o que lá estava. A pasta tem de existir. A resposta é vazia.

### CNTRL-B03 — `fs.list` devolve as entradas de uma pasta por ordem de nome
O pedido `fs.list` com o caminho de uma pasta devolve as entradas dela por ordem de nome, cada uma com
o nome, se é pasta e o tamanho.

### CNTRL-B04 — `proc.run` corre um processo até ao fim e devolve o que ele produziu
O pedido `proc.run` com o programa e os argumentos, e opcionalmente a pasta e a entrada, corre o
processo sem terminal até ao fim e devolve o código de saída (ou -1 e o nome do sinal), a saída e o
erro, separados.

### CNTRL-B05 — Os pedidos respondem-se de forma independente
Um pedido lento (um `proc.run` demorado) não atrasa a resposta aos que chegam depois dele. Cada
resposta leva o identificador do seu pedido.

### CNTRL-B06 — Uma notificação não tem resposta
Uma mensagem sem identificador não recebe resposta, mesmo de erro.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `path` | um caminho do sistema de ficheiros do nó | um caminho vazio | este módulo: responde com parâmetros inválidos (`CNTRL-E02`) |
| `argv` | o programa e os argumentos | uma lista vazia | este módulo: responde com parâmetros inválidos (`CNTRL-E02`) |
| `data` | texto em base64 | texto que não é base64 | este módulo: responde com parâmetros inválidos (`CNTRL-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `CNTRL-E01` | o método não existe | erro de método não encontrado (`CodeMethodNotFound`) |
| `CNTRL-E02` | os parâmetros não têm a forma do método | erro de parâmetros inválidos (`CodeInvalidParams`) |
| `CNTRL-E03` | o sistema recusa a operação (o ficheiro não existe, sem permissão, o programa não arranca) | erro de operação (`CodeOperation`), com a mensagem do sistema |
| `CNTRL-E04` | o ficheiro a ler é maior do que o limite | erro de operação com o motivo do tamanho, sem ler o ficheiro |
| `CNTRL-E05` | a linha recebida não é uma mensagem | erro do codec respondido com o identificador quando o há, e a sessão continua |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `MaxFile` | 8 MiB | em base64 cabe numa linha do codec (16 MiB), com folga |
| `CodeOperation` | -32000 | o primeiro código do intervalo que o JSON-RPC 2.0 deixa às aplicações |

## Rule uses

| Rule | Uses |
| --- | --- |
| `CNTRL-B01` | `path` |
| `CNTRL-B02` | `path`, `data` |
| `CNTRL-B03` | `path` |
| `CNTRL-B04` | `argv`, `data` |
| `CNTRL-B05` | `argv` |
| `CNTRL-B06` | `path` |
| `CNTRL-E01` | `path` |
| `CNTRL-E02` | `path`, `argv`, `data` |
| `CNTRL-E03` | `path`, `argv`, `CodeOperation` |
| `CNTRL-E04` | `path`, `MaxFile` |
| `CNTRL-E05` | `path` |

## Revisions

> **CNTRL-R0001:** o `Answer` responde a um pedido sem sessão, com as mesmas operações e os mesmos
> erros: é como o servidor da UI atende o nó `local`, a própria máquina. As regras não mudam.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
