<!-- anchors:progress — the STATE of plan `terminal-remoto.plan.md`.

This file stays OUTSIDE THE MAP on purpose: it exists to CHANGE, and a
file that changes by nature cannot be confronted by the gates that
charge a justification for change. The plan next to it is the DECISION;
altering it has to mean the decision changed.

Mark `[x]` here, never in the plan.
-->

# Progress — TRRMT

## TRRMT-W01 — O protocolo e as identidades

- [x] `internal/proto/codec.spec.md`
- [x] `internal/sshx/keys.spec.md`

## TRRMT-W02 — O nó

- [x] `internal/pty/pty.spec.md`
- [x] `internal/node/server.spec.md`
- [x] `internal/node/control.spec.md`

## TRRMT-W03 — O desktop na linha de comandos

- [x] `internal/hub/client.spec.md`
- [x] `cmd/polvo/cli.spec.md`

## Definition of Done (2026-10-17)

- [x] `make test` verde no `go test` e no `galaxy test`, nos seis pacotes.
- [x] O `polvo node` a correr no Galaxy (`GALAXY_ENGINE=go galaxy run ./cmd/polvo node`) aceita o
  `polvo connect` compilado pelo `go`: uma bash real num pty, `stty size` e o código de saída 7.
  O redimensionamento da janela é provado pelo `CLIXX-B05` (um terminal real e o `SIGWINCH`).
- [x] `anchors check` sem gates bloqueantes a falhar.
