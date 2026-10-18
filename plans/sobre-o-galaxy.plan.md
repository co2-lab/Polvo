<!-- @anchors
  code: GLXPV
  updated_at: 2026-10-18
-->
# Polvo sobre o Galaxy

> **Code**: `GLXPV`
>
> Pedido pelo dono a 2026-10-18 (a opção A): o Polvo novo existe para provar que o Galaxy funciona e tem
> propósito. Corre num só processo do Galaxy, com cada camada na língua que o dono escolheu por mérito
> (a janela em Rust com o gpui, o núcleo do terminal em Zig com a `libghostty-vt`, o servidor em Go), as
> camadas a falar pela fronteira do Galaxy, e o Galaxy a dar só as primitivas das línguas para o nativo.
> As fases do produto que faltam (`IDNTD`, `MOBIL`, `BRWSR`) esperam por este plano. O desenho e as
> fases são do agente, aprovados pelo dono a 2026-10-18 (`GLXPV-R0001`).

## Objective

O executável do Polvo é o Galaxy a correr o código do Polvo nas três línguas, juntas num processo: o Go do
nó e da estação, o Rust da janela e do transporte entre redes, e o Zig do núcleo do terminal. Uma camada
chama a outra pela fronteira do Galaxy, sem ABI C entre código nosso. O que só existe nativo (o sistema
operativo, a GPU, as bibliotecas do sistema) entra pela primitiva de cada língua (o `extern "C"` do Rust, o
cgo do Go, a interoperabilidade C do Zig), que o Galaxy implementa.

## Rationale

Hoje o Polvo é feito de executáveis por língua colados pela ABI C: a janela é um binário Rust do `cargo`
que leva uma biblioteca Go do `go build` (a `libpolvo`, c-archive), e o transporte entre redes é uma
biblioteca Rust com uma API C que o Go chama por cgo. O Galaxy só entra como segundo motor nos testes do
código Go. Isso não prova nada sobre o Galaxy, e força decisões de arquitetura e de língua para caber na
cola, que um Polvo sobre o Galaxy deitaria fora (o dono, a 2026-10-18).

## O que o Polvo usa, medido (2026-10-18)

- **Go** (o nó, as sessões, o SSH, o MCP, a estação, a ponte): o `go.mod` tem 5 dependências diretas (o
  `x/crypto`, o `x/net`, o `x/term`, o `creack/pty`, o `charmbracelet/x/vt`) e 18 indiretas. Já corre no
  Galaxy (`make test-galaxy`); as lacunas que apareceram viraram caixas do Bavel (1386 a 1415).
- **Rust** (a janela `desktop/` e o transporte `p2p/`): 668 crates no grafo do Linux, cerca de 5,6
  milhões de linhas. As famílias maiores: o gpui (347 mil), o x11 e o ash (318 mil, o X e o Vulkan), o
  wgpu e o naga (260 mil, a GPU), o regex (135 mil), o tokio (133 mil), o serde (102 mil), o iroh e o
  noq (117 mil), o rustls e o ring (92 mil). No Galaxy, o Rust lê e expande a gramática inteira (a P0 do
  `docs/PLANO-RUST.md` do Bavel, a caixa 1082); tipos, traits, genéricos, `async` e `unsafe` ainda não
  correm.
- **Zig**: a `libghostty-vt` (o núcleo do terminal do Ghostty). No Galaxy, o Zig é um embrião.
- **C e assembly que vêm nas crates**: as 30 gramáticas do tree-sitter (o realce do editor), o `ring`
  (a criptografia do rustls e do iroh, em C e assembly), o `blake3`.
- **Bibliotecas do sistema**: o freetype e o fontconfig (as fontes), o wayland e o x11 (as janelas), o
  carregador do Vulkan, e as chamadas do sistema (o pty, os sockets).

## Design

- **Um processo, o Galaxy.** Em desenvolvimento, `galaxy run` com o código do Polvo nas três línguas; o
  produto é um executável do `galaxy build` (o `docs/PLANO-COMPILACAO.md` do Bavel).
- **A fronteira entre as línguas.** A janela (Rust) chama a estação (Go) e o transporte (Rust) chama o nó
  (Go) como chamadas no mesmo processo, pelo orquestrador de fronteira do Galaxy (`docs/18`, o
  `guides/fronteira.md` do Bavel), com os valores a cruzar crus. A ABI C entre código nosso sai: a
  `libpolvo` como c-archive, a API C do `p2p/` (`ffi.rs`) e o cgo do `internal/transport`.
- **O nativo pela primitiva da língua.** O Galaxy não tem API própria: o que é do sistema entra como a
  língua o faria — o `extern "C"`/`libc` do Rust, o cgo do Go, a interoperabilidade C do Zig — e o
  Galaxy implementa essas primitivas (carregar a biblioteca do sistema e chamar a função), validadas
  contra o compilador oficial de cada língua.
- **Os dois motores, também no Rust e no Zig.** Cada parte do Polvo tem de dar o mesmo resultado nos
  testes com o compilador oficial (`go test`, `cargo test`, `zig test`) e no Galaxy. Uma divergência é
  uma caixa no Bavel, nunca um contorno no Polvo.
- **O desempenho conta desde o princípio.** Uma janela a 60 imagens por segundo e um handshake de QUIC
  interpretados podem não chegar: cada fase mede o tempo no Galaxy (o interpretador e, quando existir, o
  JIT do plano de compilação), e um caso lento é uma caixa de desempenho no Bavel.

## Phases

### GLXPV-W01 — A medição de cada língua contra o Polvo

O que falta no Galaxy para cada crate e pacote do Polvo, medido como a suíte oficial do Go foi medida: as
crates do grafo, por ordem de dependência (das folhas para o gpui), cada uma pelo Galaxy, com o motivo de
cada falha agrupado por causa; o mesmo para a `libghostty-vt`; a lista das primitivas nativas que o grafo
chama (as funções do `libc`, das bibliotecas do sistema, e o C e o assembly das crates); e o tempo dos
caminhos quentes. Sai daqui a lista de caixas do Bavel e a ordem delas.

- `bavel/docs/PLANO-POLVO.md` — nasce: a medição, as famílias de causas, e as caixas por ordem
- `plans/sobre-o-galaxy.plan.md` — muda: as fases seguintes com os números da medição

### GLXPV-W02 — A fronteira Go↔Rust, sem ABI C (depends on GLXPV-W01)

A ponte da janela (`desktop/src/remote.rs` e o `internal/bridge`) passa a ser uma chamada entre línguas
no mesmo processo do Galaxy: o Rust da ponte e o `serde_json` a correr no Galaxy (as fases do Rust que o
pedem), e o mecanismo de chamar uma função de outra língua com os valores crus.

### GLXPV-W03 — As primitivas nativas das línguas (depends on GLXPV-W01)

O `extern "C"` do Rust, o cgo do Go e a interoperabilidade C do Zig no Galaxy, para as bibliotecas do
sistema e as chamadas do sistema que o Polvo usa (o pty, os sockets UDP e TCP, o freetype, o fontconfig, o
wayland, o x11, o Vulkan).

### GLXPV-W04 — O transporte entre redes no Galaxy (depends on GLXPV-W02, GLXPV-W03)

O `p2p/` sobre o iroh, com o tokio, o rustls e a criptografia, a correr no Galaxy, e o nó em Go a usá-lo
pela fronteira (sai o `ffi.rs` e o cgo do `internal/transport`).

### GLXPV-W05 — O núcleo do terminal em Zig (depends on GLXPV-W03)

A `libghostty-vt` a correr no Galaxy, chamada pela janela pela fronteira.

### GLXPV-W06 — A janela (depends on GLXPV-W02, GLXPV-W03, GLXPV-W05)

O gpui, o gpui-component, o wgpu e o naga, as fontes e o realce (o tree-sitter) a correr no Galaxy: a
janela do Polvo inteira.

### GLXPV-W07 — O executável do Polvo (depends on GLXPV-W04, GLXPV-W06)

O `galaxy build` faz o executável do Polvo com as três línguas e as primitivas nativas, em Linux, macOS e
Windows; os testes de todo o Polvo passam no Galaxy como passam nos compiladores oficiais.

## Out of Scope

- Funcionalidades novas do Polvo (as fases que faltam de `IDNTD`, `MOBIL` e `BRWSR`): esperam por este
  plano.
- Trocar a língua de uma camada para caber no que o Galaxy já corre (a opção B, que o dono não escolheu).

## Revisions

> **GLXPV-R0001:** o dono aprovou o plano a 2026-10-18, com as duas propostas do agente: o C das crates
> corre pelo frontend de C do Galaxy e o assembly dá lugar à variante em Rust ou C de cada crate, e as
> bibliotecas do sistema entram pela primitiva da língua (era a `GLXPV-D01`); a cola em C de hoje continua
> a fazer o produto funcionar até a fase que a substitui passar (era a `GLXPV-D02`).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

## Definition of Done

- O Polvo corre num só processo do Galaxy (`galaxy run` em desenvolvimento, e o executável do `galaxy
  build`), com o Go, o Rust e o Zig juntos, e sem ABI C entre código nosso.
- Os testes do Polvo passam no Galaxy e nos compiladores oficiais (`go test`, `cargo test`, `zig test`).
- A janela abre, espelha uma sessão de outra máquina por outra rede, e o terminal desenha a 60 imagens por
  segundo no Galaxy.
