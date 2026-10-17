<!-- @anchors
  code: DSKTP
  updated_at: 2026-10-17
-->
# Desktop

> **Code**: `DSKTP`
>
> ⚠️ O âmbito deste plano foi aprovado pelo agente e **ainda não foi revisto pelo dono**. O registo está
> no INSIGHTS.md.

## Objective

O `polvo` sem subcomando é a aplicação desktop: um editor simples e um terminal sobre os nós. O
utilizador escolhe um nó, navega nas pastas dele, abre um ficheiro no editor, grava-o, e abre terminais
no nó, tudo numa janela. O executável é o mesmo, e o comportamento é o mesmo no `go` e no Galaxy.

## Rationale

Os planos TRRMT e AGLCG deram o canal e a ponte do agente; falta o que o dono descreveu primeiro: "um
editor simples e um terminal". Sem a UI, o Polvo é uma ferramenta de linha de comandos. A UI fala com
o `internal/hub`, que já existe e já é provado nos dois motores.

## Phases

### DSKTP-W01 — O servidor da UI

Entrega o lado Go da UI.
- O servidor só aceita ligações locais e só o browser que recebeu o segredo da sessão.
- Expõe as chamadas de controlo e os terminais dos nós.

- `internal/web/server.spec.md` — o servidor HTTP local: o segredo da sessão, os ficheiros da UI
  embutidos, as chamadas de controlo aos nós e os terminais por WebSocket

### DSKTP-W02 — A UI (depends on DSKTP-W01)

Só começa depois da W01, porque fala com as rotas dela.

- `ui/app.spec.md` — a janela: a lista dos nós, a árvore de pastas de um nó, o editor com abrir e
  gravar, e os terminais por baixo

### DSKTP-W03 — O subcomando (depends on DSKTP-W02)

Só começa depois da W02, porque embute o bundle dela.

- `cmd/polvo/cli.spec.md` — muda: o `polvo` sem subcomando (e o `polvo ui`) arranca o servidor, mostra o
  endereço com o segredo e abre o browser

## Out of Scope

- A janela nativa (a webview do Galaxy): é uma caixa no Bavel. Até ela existir, o browser do sistema.
- Os nomes dos nós guardados num ficheiro da instalação: a UI lembra os endereços no browser.
- O realce de sintaxe por linguagem e a pesquisa no projeto: o editor desta fase é texto simples, com
  números de linha.
- O agente dentro da UI: o agente local usa o `polvo mcp` (AGLCG).

## Definition of Done

- `make test` passa com o `go test` e com o `galaxy test`; os testes da UI passam no `npm test`.
- Com o `polvo` a correr no Galaxy, o browser abre a UI; num nó real, abre-se uma pasta, edita-se e
  grava-se um ficheiro, e corre-se um comando num terminal.
- `anchors check` passa os gates bloqueantes.
