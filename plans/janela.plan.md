<!-- @anchors
  code: JNNTJ
  updated_at: 2026-10-17
-->
# Janela nativa

> **Code**: `JNNTJ`
>
> A arquitetura foi decidida pelo dono em 2026-10-17 (INSIGHTS.md, "a janela nativa"). As fases são do
> agente e estão por rever pelo dono.

## Objective

O Polvo abre numa janela nativa, desenhada na GPU, com um terminal ao nível dos melhores do mercado.
A janela é Rust (GPUI, a UI do Zed), o núcleo do terminal é a libghostty-vt (Zig, o núcleo do Ghostty)
e o servidor dos nós continua em Go. A janela segue o ambiente de trabalho: no GNOME, a barra de título
com os botões na ordem do sistema, o tema e a fonte monoespaçada dele.

## Rationale

O dono quer "um terminal num desktop cross platform", e o melhor do mercado em cada parte. Os melhores
terminais (Ghostty, WezTerm, Alacritty, Warp) são nativos e desenhados na GPU; a UI no browser do plano
DSKTP fica até esta a substituir. O protótipo de referência (o terminal a correr a shell numa janela
GPUI, com a barra ao estilo do GNOME) foi aprovado pelo dono a 2026-10-17.

## Phases

### JNNTJ-W01 — O terminal nativo

A crate `desktop/` (Rust) com a janela e o terminal da máquina local. É a base das fases seguintes.

- `desktop/src/prefs.spec.md` — as preferências do ambiente de trabalho que a janela segue: a ordem
  dos botões da barra, o tema claro ou escuro e a fonte monoespaçada
- `desktop/src/keys.spec.md` — a tecla que a janela recebe, traduzida para a tecla da libghostty-vt
- `desktop/src/screen.spec.md` — o terminal: a shell num pty, o estado do ecrã na libghostty-vt, o
  tamanho, o scroll, a seleção e o texto a copiar
- `desktop/src/window.spec.md` — a janela GPUI: a barra de título ao estilo do sistema, o desenho do
  ecrã do terminal, o teclado, o rato, copiar e colar

### JNNTJ-W02 — Os nós (depends on JNNTJ-W01)

Os terminais nos nós remotos e no nó local, pelo servidor Go que já existe (`internal/web`, `internal/hub`).
A forma da ligação entre a janela e o Go decide-se nesta fase.

### JNNTJ-W03 — O editor (depends on JNNTJ-W02)

O editor do gpui-component, com o realce de sintaxe, e a árvore de pastas de um nó.

### JNNTJ-W04 — Um executável (depends on JNNTJ-W03)

O Go e o Rust num só executável `polvo`, e o `polvo` sem subcomando abre a janela nativa.

## Out of Scope

- O macOS e o Windows: o GPUI, a libghostty-vt e o portable-pty correm nos três, mas aqui só há Linux
  para provar. A prova nos outros dois fica para quando houver a máquina (o dono testa em Win e Mac).
- Correr a janela no Galaxy: é o plano do Bavel "Rust e Zig no Galaxy", medido contra esta crate.
- Separadores, divisão do ecrã e a configuração do utilizador.

## Definition of Done

- `make test` passa com os testes da janela (`cargo test`) além dos de hoje.
- A janela abre no GNOME com a barra ao estilo do sistema, corre a shell do utilizador, e o scroll, a
  seleção e copiar e colar funcionam.
- `anchors check` passa os gates bloqueantes.
