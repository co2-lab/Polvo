<!-- anchors:progress v1 — written by `anchors progress`; never edit by hand -->
# Progress — 89% (26/29 points)

## D01 — AgenteLocal (plan: plans/agente-local.plan.md) — 100% (6/6)
### D01.S1 — AGLCG-W01 — A ponte MCP — 100% (2/2)
- [x] D01.T1 — `internal/mcp/server.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D01.T2 — `internal/mcp/tools.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
### D01.S2 — AGLCG-W02 — O subcomando — 100% (1/1)
- [x] D01.T3 — `cmd/polvo/cli.spec.md` (o `polvo mcp`) · 1 (estimate) · done 2026-10-17 by migrated
### D01.S3 — Definition of Done (2026-10-17) — 100% (3/3)
- [x] D01.T4 — `make test` verde no `go test` e no `galaxy test`. · 1 (estimate) · done 2026-10-17 by migrated
- [x] D01.T5 — Um cliente MCP real: o `claude -p` com `--mcp-config` a lançar a ponte NO GALAXY · 1 (estimate) · done 2026-10-17 by migrated
- [x] D01.T6 — `anchors check` sem gates bloqueantes a falhar. · 1 (estimate) · done 2026-10-17 by migrated

## D02 — Desktop (plan: plans/desktop.plan.md) — 100% (6/6)
### D02.S1 — DSKTP-W01 — O servidor da UI — 100% (1/1)
- [x] D02.T1 — `internal/web/server.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
### D02.S2 — DSKTP-W02 — A UI — 100% (1/1)
- [x] D02.T2 — `ui/app.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
### D02.S3 — DSKTP-W03 — O subcomando — 100% (1/1)
- [x] D02.T3 — `cmd/polvo/cli.spec.md` (o `polvo` sem subcomando e o `polvo ui`) · 1 (estimate) · done 2026-10-17 by migrated
### D02.S4 — Definition of Done (2026-10-17) — 100% (3/3)
- [x] D02.T4 — `make test` verde: `go test`, `galaxy test` (os 8 pacotes) e a UI (vitest 15/15, `tsc`). · 1 (estimate) · done 2026-10-17 by migrated
- [x] D02.T5 — Num Chrome real (headless, puppeteer-core) contra `galaxy run ./cmd/polvo ui --no-browser` e um · 1 (estimate) · done 2026-10-17 by migrated
- [x] D02.T6 — `anchors check` sem gates bloqueantes a falhar (o pre-commit). · 1 (estimate) · done 2026-10-17 by migrated

## D03 — Janela nativa (plan: plans/janela.plan.md) — 57% (4/7)
### D03.S1 — JNNTJ-W01 — O terminal nativo — 100% (4/4)
- [x] D03.T1 — `desktop/src/prefs.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D03.T2 — `desktop/src/keys.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D03.T3 — `desktop/src/screen.spec.md` · 1 (estimate) · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D03.T4 — `desktop/src/window.spec.md` · 1 (estimate) · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D03.S2 — JNNTJ-W02 — A doca — 0% (0/3)
- [ ] D03.T5 — `desktop/src/panel.spec.md` · 1
- [ ] D03.T6 — `desktop/src/window.spec.md` (muda) · 1
- [ ] D03.T7 — `desktop/src/layout.spec.md` · 1

## D04 — TerminalRemoto (plan: plans/terminal-remoto.plan.md) — 100% (10/10)
### D04.S1 — TRRMT-W01 — O protocolo e as identidades — 100% (2/2)
- [x] D04.T1 — `internal/proto/codec.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D04.T2 — `internal/sshx/keys.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
### D04.S2 — TRRMT-W02 — O nó — 100% (3/3)
- [x] D04.T3 — `internal/pty/pty.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D04.T4 — `internal/node/server.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D04.T5 — `internal/node/control.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
### D04.S3 — TRRMT-W03 — O desktop na linha de comandos — 100% (2/2)
- [x] D04.T6 — `internal/hub/client.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D04.T7 — `cmd/polvo/cli.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
### D04.S4 — Definition of Done (2026-10-17) — 100% (3/3)
- [x] D04.T8 — `make test` verde no `go test` e no `galaxy test`, nos seis pacotes. · 1 (estimate) · done 2026-10-17 by migrated
- [x] D04.T9 — O `polvo node` a correr no Galaxy (`GALAXY_ENGINE=go galaxy run ./cmd/polvo node`) aceita o · 1 (estimate) · done 2026-10-17 by migrated
- [x] D04.T10 — `anchors check` sem gates bloqueantes a falhar. · 1 (estimate) · done 2026-10-17 by migrated

<!-- anchors:progress sum=422697f228c0 -->
