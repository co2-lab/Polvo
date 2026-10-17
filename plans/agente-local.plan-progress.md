<!-- anchors:progress — the STATE of plan `agente-local.plan.md`.

This file stays OUTSIDE THE MAP on purpose: it exists to CHANGE, and a
file that changes by nature cannot be confronted by the gates that
charge a justification for change. The plan next to it is the DECISION;
altering it has to mean the decision changed.

Mark `[x]` here, never in the plan.
-->

# Progress — AGLCG

## AGLCG-W01 — A ponte MCP

- [x] `internal/mcp/server.spec.md`
- [x] `internal/mcp/tools.spec.md`

## AGLCG-W02 — O subcomando

- [x] `cmd/polvo/cli.spec.md` (o `polvo mcp`)

## Definition of Done (2026-10-17)

- [x] `make test` verde no `go test` e no `galaxy test`.
- [x] Um cliente MCP real: o `claude -p` com `--mcp-config` a lançar a ponte NO GALAXY
  (`galaxy run ./cmd/polvo mcp`) correu o `polvo_run` num `polvo node` e devolveu
  "ola-do-polvo / Linux". Pediu a caixa 1400 do Bavel (o stdin em fluxo).
- [x] `anchors check` sem gates bloqueantes a falhar.
