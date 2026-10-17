<!-- @anchors
  code: AGLCG
  updated_at: 2026-10-17
-->
# AgenteLocal

> **Code**: `AGLCG`
>
> ⚠️ O âmbito deste plano foi aprovado pelo agente e **ainda não foi revisto pelo dono**. A meta dada
> pelo dono é "criar a nova versão do Polvo usando o Galaxy/Bavel como núcleo", e o propósito é o dele:
> "vou usar um agente na minha máquina no meu Polvo para comunicar com os agentes de outras máquinas e
> controlá-los". O registo está no INSIGHTS.md.

## Objective

Um agente na máquina local (o Claude Code ou outro cliente MCP) controla as máquinas onde corre o
`polvo node`: corre comandos nelas, lê e escreve os ficheiros delas e lista as pastas. Usa para isso o
`polvo mcp`, uma ponte MCP por stdio. O executável é o mesmo, e o comportamento é o mesmo no `go` e no
Galaxy.

## Rationale

O plano TRRMT deu o canal: o nó, o controlo e o cliente. Sem a ponte, só um humano o usa, pelo `polvo
connect`. O agente local é a razão de ser do novo Polvo. O MCP é o que os agentes já falam, e por stdio
não abre nenhuma porta. As mensagens do MCP são as do codec da W01 do TRRMT (JSON-RPC 2.0, uma por
linha).

## Phases

### AGLCG-W01 — A ponte MCP

Entrega o protocolo e as ferramentas sobre o cliente do TRRMT.

- `internal/mcp/server.spec.md` — o protocolo MCP do lado do servidor: a inicialização, a lista das
  ferramentas, a chamada de uma ferramenta, o ping, e o que acontece a um pedido que não conhece
- `internal/mcp/tools.spec.md` — as ferramentas do Polvo: correr um comando num nó, ler e escrever um
  ficheiro de um nó, listar uma pasta de um nó; as ligações aos nós reutilizadas entre chamadas

### AGLCG-W02 — O subcomando (depends on AGLCG-W01)

Só começa depois da W01, porque liga a ponte à identidade e aos nós conhecidos da instalação.

- `cmd/polvo/cli.spec.md` — muda: nasce o subcomando `polvo mcp` (a ponte pelo stdin e stdout, com a
  pasta de configuração)

## Out of Scope

- O terminal interativo pelo MCP. Uma ferramenta devolve um resultado; uma sessão interativa é da UI.
- Os nomes dos nós (um ficheiro de nós com nomes e endereços). As ferramentas recebem o endereço; os
  nomes ficam para o plano da UI, que também os mostra.
- Os recursos e os prompts do MCP: só as ferramentas.
- O transporte HTTP do MCP: só o stdio.

## Definition of Done

- `make test` passa com o `go test` e com o `galaxy test`.
- Um cliente MCP real (o `claude mcp add polvo -- polvo mcp`) lista as ferramentas e corre um comando
  num `polvo node`.
- `anchors check` passa os gates bloqueantes.
