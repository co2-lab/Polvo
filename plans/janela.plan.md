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

### JNNTJ-W02 — A doca (depends on JNNTJ-W01)

A janela passa a ser uma doca: os terminais são painéis que o utilizador organiza arrastando e
largando, em abas ou divididos lado a lado, e a organização volta igual na próxima vez. Só dentro da
janela; tirar um painel para uma janela à parte fica fora desta fase.

- `desktop/src/panel.spec.md` — nasce: um terminal como painel da doca (o desenho, o teclado, o rato,
  copiar e colar e o tamanho, que saem da janela)
- `desktop/src/window.spec.md` — muda: o corpo da janela é a doca; o botão de abrir um terminal; a
  janela fecha quando sai o último terminal
- `desktop/src/layout.spec.md` — nasce: a organização da doca guardada e restaurada

### JNNTJ-W03 — Um executável (depends on JNNTJ-W02)

O Go e o Rust num só executável `polvo`, e o `polvo` sem subcomando abre a janela nativa.

## Out of Scope

- O macOS e o Windows: o GPUI, a libghostty-vt e o portable-pty correm nos três, mas aqui só há Linux
  para provar. A prova nos outros dois fica para quando houver a máquina (o dono testa em Win e Mac).
- Correr a janela no Galaxy: é o plano do Bavel "Rust e Zig no Galaxy", medido contra esta crate.
- Tirar um painel para uma janela do sistema à parte (vários monitores): o dono decidiu a 2026-10-17
  que, por agora, a organização é dentro da janela.
- A configuração do utilizador (atalhos, temas próprios).

## Revisions

> **JNNTJ-R0001:** entra a fase W02, a doca, pedida pelo dono a 2026-10-17 ("janelas organizadas pelo
> utilizador, arrastando e soltando"), depois de ver o terminal nativo. Os nós, o editor e o executável
> passam a W03, W04 e W05. Os separadores e a divisão do ecrã, que estavam fora do âmbito, são agora a
> W02.

> **JNNTJ-R0002:** as fases dos nós e do editor passam para o plano `espelho.plan.md` (ESRMS), quando o
> dono pediu a 2026-10-17 que a janela espelhe os terminais abertos noutras máquinas, em intranets e em
> P2P: as sessões passam a viver no `polvo node`, o que muda o nó, o cliente e a janela. O executável
> único passa a W03.

## Definition of Done

- `make test` passa com os testes da janela (`cargo test`) além dos de hoje.
- A janela abre no GNOME com a barra ao estilo do sistema, corre a shell do utilizador, e o scroll, a
  seleção e copiar e colar funcionam.
- `anchors check` passa os gates bloqueantes.
