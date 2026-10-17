<!-- @anchors
  code: CLNTI
  updated_at: 2026-10-17
-->
# Client — a ligação do desktop a um nó: o terminal e as chamadas de controlo

> **Code**: `CLNTI`

## Overview
O desktop (a CLI agora, a UI e a ponte do agente depois) liga-se a um nó pelo `Dial` e recebe um
`Node`. Por ele abre terminais interativos (`Shell`, que dá um `Terminal`) e faz chamadas de controlo
(`Call` e os atalhos `ReadFile`, `WriteFile`, `List` e `Run`), e trabalha com as sessões de terminal do
nó (`Sessions`, que dá cada `Session`; `NewSession`, `CloseSession`, `LockSession`; e `Attach`, que liga
um `Terminal` a uma sessão que já existe). A ligação autentica-se com a identidade
da instalação e confere a chave do nó na lista de nós conhecidos. A configuração é o `Config`.

## Rules

### CLNTI-B01 — A ligação usa a identidade e confere a chave do nó
A ligação apresenta-se com a identidade da instalação. A chave do nó é conferida na lista de nós
conhecidos: o primeiro contacto regista-a, a mesma chave entra, e uma chave mudada recusa a ligação com
o erro de chave mudada. O `DialConn` faz a mesma ligação sobre uma ligação de rede já aberta.

### CLNTI-B02 — O terminal interativo tem tamanho, entrada, saída e fim
O `Shell` abre uma shell num terminal do nó com o tamanho pedido. O que se escreve no `Terminal` chega à
shell, o que a shell escreve lê-se do `Terminal`, o `Resize` muda o tamanho que a shell vê, e o `Wait`
devolve o código de saída (ou o nome do sinal) quando a shell termina.

### CLNTI-B03 — Cada chamada recebe a sua resposta
O `Call` envia um pedido de controlo e devolve o resultado da resposta com o mesmo identificador. Várias
chamadas ao mesmo tempo recebem cada uma a sua, por qualquer ordem que o nó responda.

### CLNTI-B04 — Uma resposta de erro é um erro com o código
Uma resposta de erro do nó chega a quem chamou como um erro que traz o código e a mensagem do nó
(`proto.Error`).

### CLNTI-B05 — Os atalhos de controlo falam em bytes
O `ReadFile` devolve os bytes de um ficheiro do nó, o `WriteFile` escreve-os, o `List` devolve as
entradas de uma pasta (`Entry`), e o `Run` devolve o código, a saída e o erro de um processo
(`RunResult`). A conversão de e para base64 é do atalho, não de quem o chama.

### CLNTI-B06 — Fechar a ligação acaba as chamadas pendentes
O `Close` fecha a ligação ao nó. Uma chamada que ainda esperava resposta termina com erro, e uma chamada
nova também.

### CLNTI-B07 — As sessões do nó
O `Sessions` dá as sessões vivas do nó, cada uma com o id, o programa, o tamanho, os espectadores, se
está travada e se esta ligação pode escrever nela. O `NewSession` cria uma sessão (o programa pedido, ou
a shell do nó) e devolve o id; o `CloseSession` termina-a; o `LockSession` trava-a ou destrava-a.

### CLNTI-B08 — Ligar-se a uma sessão
O `Attach` dá um `Terminal` ligado a uma sessão que já existe: lê primeiro o ecrã atual dela e depois a
saída, e o que se escreve chega à sessão quando a ligação a pode controlar. Fechar o `Terminal` deixa a
sessão viva.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `addr` | máquina e porta do nó | um endereço sem porta | o `Dial`: devolve o erro de endereço (`CLNTI-E01`) |
| `Identity` | a identidade da instalação | nenhuma | quem liga: o `Dial` recusa sem ela (`CLNTI-E02`) |
| `KnownHosts` | a lista de nós conhecidos | nenhuma | quem liga: o `Dial` recusa sem ela (`CLNTI-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `CLNTI-E01` | o nó não responde no endereço | a ligação falha com um erro que nomeia o endereço |
| `CLNTI-E02` | falta a identidade ou a lista de nós conhecidos, ou o nó recusa a identidade | a ligação falha com erro |
| `CLNTI-E03` | a ligação cai enquanto uma chamada espera | a chamada termina com erro, sem ficar parada |
| `CLNTI-E04` | o `Attach` pede uma sessão que não existe | `ErrNoSession`, sem terminal |

## Rule uses

| Rule | Uses |
| --- | --- |
| `CLNTI-B01` | `Identity`, `KnownHosts` |
| `CLNTI-B02` | `addr` |
| `CLNTI-B03` | `addr` |
| `CLNTI-B04` | `addr` |
| `CLNTI-B05` | `addr` |
| `CLNTI-B06` | `addr` |
| `CLNTI-E01` | `addr` |
| `CLNTI-E02` | `Identity`, `KnownHosts` |
| `CLNTI-E03` | `addr` |
| `CLNTI-B07` | `addr` |
| `CLNTI-B08` | `addr` |
| `CLNTI-E04` | `addr` |

## Revisions

> **CLNTI-R0001:** entram as sessões do nó (`CLNTI-B07`, `CLNTI-B08`, `CLNTI-E04`), para a janela as
> mostrar e espelhar (plano ESRMS, W02).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
