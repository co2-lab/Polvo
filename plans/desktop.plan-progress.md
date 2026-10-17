<!-- anchors:progress — the STATE of plan `desktop.plan.md`.

This file stays OUTSIDE THE MAP on purpose: it exists to CHANGE, and a
file that changes by nature cannot be confronted by the gates that
charge a justification for change. The plan next to it is the DECISION;
altering it has to mean the decision changed.

Mark `[x]` here, never in the plan.
-->

# Progress — DSKTP

## DSKTP-W01 — O servidor da UI

- [x] `internal/web/server.spec.md`

## DSKTP-W02 — A UI

- [x] `ui/app.spec.md`

## DSKTP-W03 — O subcomando

- [x] `cmd/polvo/cli.spec.md` (o `polvo` sem subcomando e o `polvo ui`)

## Definition of Done (2026-10-17)

- [x] `make test` verde: `go test`, `galaxy test` (os 8 pacotes) e a UI (vitest 15/15, `tsc`).
- [x] Num Chrome real (headless, puppeteer-core) contra `galaxy run ./cmd/polvo ui --no-browser` e um
  `polvo node`: a UI abriu e o segredo saiu do endereço, o nó entrou, a pasta listou, o ficheiro abriu,
  a edição marcou ●, o Ctrl+S gravou no disco do nó, e o terminal correu
  `echo ola-do-terminal-$((40+2))` → `ola-do-terminal-42`. Pediu as caixas 1401–1404, 1406 e 1407 do
  Bavel (a 1407: o FileServer no Galaxy servia só 512 bytes de cada ficheiro).
- [x] `anchors check` sem gates bloqueantes a falhar (o pre-commit).
