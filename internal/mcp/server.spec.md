<!-- @anchors
  code: MCPSR
  updated_at: 2026-10-17
-->
# Server — o protocolo MCP do lado do servidor, por um fluxo de mensagens

> **Code**: `MCPSR`

## Overview
O agente local fala com o Polvo pelo Model Context Protocol (MCP): o agente é o cliente, o `polvo mcp`
é o servidor, e as mensagens são as do JSON-RPC 2.0, uma por linha, pelo stdin e pelo stdout. O
servidor (`Server`, com o `Name`, a `Version` e as ferramentas, `Tool`) responde à inicialização, à
lista das ferramentas e à chamada de uma ferramenta, até o fluxo acabar (`Serve`). Esta unidade não
sabe o que as ferramentas fazem: recebe-as já feitas.

## Rules

### MCPSR-B01 — A inicialização diz a versão, as capacidades e o servidor
O pedido `initialize` recebe como resposta a versão do protocolo, as capacidades (só as ferramentas) e
o nome e a versão do servidor. A versão é a que o cliente pediu, se o servidor a conhece; senão é a
mais recente que conhece.

### MCPSR-B02 — A lista das ferramentas
O pedido `tools/list` recebe todas as ferramentas, cada uma com o nome, a descrição e o esquema dos
argumentos, pela ordem em que o servidor as tem.

### MCPSR-B03 — A chamada de uma ferramenta devolve o resultado em texto
O pedido `tools/call` com o nome de uma ferramenta e os argumentos corre a ferramenta e recebe o
resultado como conteúdo de texto (`Result`). Quando a ferramenta falha, a falha também é um resultado,
marcado como erro, para o agente a ler e decidir. Não é um erro do protocolo.

### MCPSR-B04 — O ping
O pedido `ping` recebe uma resposta vazia.

### MCPSR-B05 — As notificações não têm resposta
Uma notificação (a de inicializado, a de cancelamento, ou outra) não recebe resposta.

### MCPSR-B06 — Os pedidos respondem-se de forma independente
Uma ferramenta lenta não atrasa a resposta aos pedidos que chegam depois. Cada resposta leva o
identificador do seu pedido.

### MCPSR-B07 — O fim do fluxo acaba o servidor
Quando o fluxo de entrada acaba, o servidor espera pelas respostas em curso e termina sem erro.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `method` | `initialize`, `tools/list`, `tools/call`, `ping` e as notificações | outro método | este módulo: o erro de método não encontrado (`MCPSR-E01`) |
| `name` | o nome de uma ferramenta do servidor | outro nome | este módulo: o erro de parâmetros inválidos (`MCPSR-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `MCPSR-E01` | o pedido tem um método que o servidor não conhece | o erro de método não encontrado (-32601) |
| `MCPSR-E02` | o `tools/call` nomeia uma ferramenta que não existe, ou os parâmetros não têm a forma | o erro de parâmetros inválidos (-32602) |
| `MCPSR-E03` | uma linha não é uma mensagem | o erro do codec respondido, com o identificador nulo quando não se leu, e o servidor continua |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `ProtocolVersions` | "2025-06-18", "2025-03-26", "2024-11-05" | as versões do MCP que este servidor fala, da mais recente para a mais antiga |

## Rule uses

| Rule | Uses |
| --- | --- |
| `MCPSR-B01` | `method`, `ProtocolVersions` |
| `MCPSR-B02` | `method` |
| `MCPSR-B03` | `method`, `name` |
| `MCPSR-B04` | `method` |
| `MCPSR-B05` | `method` |
| `MCPSR-B06` | `method` |
| `MCPSR-B07` | `method` |
| `MCPSR-E01` | `method` |
| `MCPSR-E02` | `name` |
| `MCPSR-E03` | `method` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
