<!-- anchors:progress v1 — written by `anchors progress`; never edit by hand -->
# Progress — 100% (64/64 points)

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

## D03 — Janela nativa (plan: plans/janela.plan.md) — 100% (7/7)
### D03.S1 — JNNTJ-W01 — O terminal nativo — 100% (4/4)
- [x] D03.T1 — `desktop/src/prefs.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D03.T2 — `desktop/src/keys.spec.md` · 1 (estimate) · done 2026-10-17 by migrated
- [x] D03.T3 — `desktop/src/screen.spec.md` · 1 (estimate) · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D03.T4 — `desktop/src/window.spec.md` · 1 (estimate) · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D03.S2 — JNNTJ-W02 — A doca — 100% (3/3)
- [x] D03.T5 — `desktop/src/panel.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D03.T6 — `desktop/src/window.spec.md` (muda) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D03.T7 — `desktop/src/layout.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main

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

## D05 — Espelho remoto (plan: plans/espelho.plan.md) — 100% (12/12)
### D05.S1 — ESRMS-W01 — As sessões no nó — 100% (4/4)
- [x] D05.T1 — `internal/sessions/sessions.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T2 — `internal/nodeconf/nodeconf.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T3 — `internal/node/server.spec.md` (muda) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T4 — `internal/node/control.spec.md` (muda) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D05.S2 — ESRMS-W02 — A janela como cliente dos nós (depends on ESRMS-W01) — 100% (8/8)
- [x] D05.T5 — `internal/hub/client.spec.md` (muda: as sessões, o Done e a HostKey) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T6 — `internal/node/control.spec.md` e `server.spec.md` (mudam: quem pode escrever, e a chave da instalação como dona) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T7 — `internal/station/station.spec.md` (nasce: o nó desta máquina na janela e as outras) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T8 — `internal/bridge/bridge.spec.md` (nasce: a API da cmd/libpolvo, em Go puro) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T9 — `desktop/src/remote.spec.md` (nasce: a libpolvo na janela, com a forma de atravessar para C, em vez do cmd/libpolvo/libpolvo.spec.md) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T10 — `desktop/src/screen.spec.md` e `panel.spec.md` (mudam: o terminal de uma sessão, dona ou alheia, e o só ver) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T11 — `desktop/src/nodes.spec.md` (nasce: as máquinas e as sessões) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D05.T12 — `desktop/src/window.spec.md` (muda: a aba das máquinas, e os terminais como sessões desta máquina) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D05.S3 — ESRMS-W03 — Descoberta e emparelhamento (depends on ESRMS-W02) — — (0/0)
### D05.S4 — ESRMS-W04 — P2P através de NAT (depends on ESRMS-W03) — — (0/0)
### D05.S5 — ESRMS-W05 — Os ficheiros de qualquer máquina (depends on ESRMS-W02) — — (0/0)

## D06 — Agentes (plan: plans/agentes.plan.md) — 100% (12/12)
### D06.S1 — AGPRG-W01 — O progresso no nó — 100% (3/3)
- [x] D06.T1 — `internal/progress/progress.spec.md` · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T2 — `internal/node/control.spec.md` (muda: os pedidos do progresso, com a espera longa em vez das notificações) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T3 — `internal/station/station.spec.md` (muda: o quadro no nó da janela) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
### D06.S2 — AGPRG-W02 — As ferramentas do agente (depends on AGPRG-W01) — 100% (2/2)
- [x] D06.T4 — `internal/mcp/tools.spec.md` (muda: as ferramentas do progresso; a de responder passa à W03) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T5 — `internal/sessions/sessions.spec.md` e `station.spec.md` (mudam: o ambiente das sessões) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
### D06.S3 — AGPRG-W03 — Perguntar ao agente (depends on AGPRG-W02) — 100% (4/4)
- [x] D06.T6 — `internal/progress/progress.spec.md` (muda: as perguntas e a conversa do agente) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T7 — `internal/node/control.spec.md` (muda: o progress.ask, pelo claude ou pela sessão) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T8 — `internal/agents/claude.spec.md` · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T9 — `internal/mcp/tools.spec.md` (muda: responder, e a conversa do Claude Code) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
### D06.S4 — AGPRG-W04 — O painel dos agentes (depends on AGPRG-W03) — 100% (3/3)
- [x] D06.T10 — `desktop/src/agents.spec.md` · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T11 — `internal/bridge/bridge.spec.md` e `desktop/src/remote.spec.md` (mudam: o progresso de todas as máquinas) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main
- [x] D06.T12 — `desktop/src/panel.spec.md` e `window.spec.md` (mudam: a aba dos agentes e a marca na aba da sessão) · 1 · done 2026-10-18 by OK-BR-PE0EF71F/polvo-main

## D07 — Configuracao e tipos de aba (plan: plans/abas.plan.md) — 100% (11/11)
### D07.S1 — CEACN-W01 — A configuração e os atalhos — 100% (4/4)
- [x] D07.T1 — `desktop/src/settings.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T2 — `desktop/src/keymap.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T3 — `desktop/src/settings_panel.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T4 — `desktop/src/window.spec.md` (muda) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D07.S2 — CEACN-W02 — O escolhedor do tipo de aba (depends on CEACN-W01) — 100% (2/2)
- [x] D07.T5 — `desktop/src/new_tab.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T6 — `desktop/src/window.spec.md` (muda) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D07.S3 — CEACN-W03 — Os ficheiros (depends on CEACN-W02) — 100% (3/3)
- [x] D07.T7 — `desktop/src/files.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T8 — `desktop/src/editor.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T9 — `desktop/src/window.spec.md` (muda: abrir um ficheiro do computador, em vez do new_tab.spec.md) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
### D07.S4 — CEACN-W04 — O git (depends on CEACN-W02) — 100% (2/2)
- [x] D07.T10 — `desktop/src/git.spec.md` · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main
- [x] D07.T11 — `desktop/src/window.spec.md` (muda: o tipo de aba do git) · 1 · done 2026-10-17 by OK-BR-PE0EF71F/polvo-main

<!-- anchors:progress sum=78b13b9e4fc0 -->
