<!-- @anchors
  code: ESRMS
  updated_at: 2026-10-17
-->
# Espelho remoto

> **Code**: `ESRMS`
>
> O desenho foi aprovado pelo dono a 2026-10-17: as sessões vivem no `polvo node` de cada máquina, com
> vários espectadores, e a janela é cliente dos nós; tem de funcionar em intranets e em P2P; e cada
> máquina controla as permissões das suas sessões (ver ou controlar), com uma configuração por omissão.
> As fases são do agente e estão por rever pelo dono.

## Objective

O Polvo de uma máquina espelha os terminais de outra: vê o que aparece num terminal aberto lá, escreve
nele como se lá estivesse, abre terminais novos nessa máquina, e vê e edita os ficheiros dela. As
máquinas encontram-se na rede local, ligam-se por endereço numa intranet, e ligam-se entre redes
diferentes (através de NAT) pela chave de cada uma.

## Rationale

O dono quer que um agente (um `claude` num terminal) inicie o Polvo noutra máquina e que o Polvo da
máquina dele espelhe os processos dessa, como um terminal aberto lá. Hoje cada ligação ao `polvo node`
cria uma shell só sua, que morre com a ligação: não há nada para espelhar. As sessões têm de viver no
nó, independentes de quem as vê.

## Phases

### ESRMS-W01 — As sessões no nó

O `polvo node` guarda as sessões de terminal da máquina: cada uma é uma shell num pty, que continua sem
espectadores e aceita vários ao mesmo tempo. Quem chega a meio recebe o ecrã atual: o nó guarda o
estado de cada ecrã na libghostty-vt, que o escreve de volta em sequências VT.

- `internal/sessions/sessions.spec.md` — nasce: criar, listar e terminar sessões; ligar-se como
  espectador; o ecrã atual para quem chega; o tamanho; a permissão de cada espectador (ver, ou
  controlar: escrever e mudar o tamanho), e a sessão travada para só ver
- `internal/nodeconf/nodeconf.spec.md` — nasce: a configuração do Polvo da máquina, com a permissão por
  omissão de quem se liga de fora (ver ou controlar) e a de cada chave autorizada
- `internal/node/server.spec.md` — muda: o terminal de uma ligação SSH é uma sessão nova ou existente,
  e não uma shell só dela
- `internal/node/control.spec.md` — muda: os pedidos `session.list` e `session.new`

### ESRMS-W02 — A janela como cliente dos nós (depends on ESRMS-W01)

O cliente Go (`internal/hub`) entra no mesmo executável que a janela, como biblioteca C (cgo nas
ferramentas reais, a mistura de línguas no Galaxy). A janela mostra as máquinas e as sessões delas, abre
uma sessão como aba da doca, e os terminais locais passam também pelo nó local, para poderem ser
espelhados. Ligar a uma máquina pelo endereço serve as intranets.

- `internal/hub/client.spec.md` — muda: listar e ligar-se a sessões
- `cmd/libpolvo/libpolvo.spec.md` — nasce: a API C do cliente para a janela (ligar, sessões, ler,
  escrever, redimensionar)
- `desktop/src/nodes.spec.md` — nasce: as máquinas e as sessões na janela; travar e destravar as
  sessões da própria máquina; a sessão de outra só para ver mostra-o e não aceita teclas
- `desktop/src/screen.spec.md` — muda: o terminal pode ser alimentado por uma sessão de um nó, e não só
  por um pty local

### ESRMS-W03 — Descoberta e emparelhamento (depends on ESRMS-W02)

- `internal/discovery/discovery.spec.md` — nasce: o nó anuncia-se e a janela encontra os nós da rede
  local por mDNS (`_polvo._tcp`)
- `internal/pairing/pairing.spec.md` — nasce: a primeira ligação entre duas instalações autoriza a chave
  com um código curto que a máquina remota mostra

### ESRMS-W04 — P2P através de NAT (depends on ESRMS-W03)

Ligar a um nó noutra rede pela chave pública dele: furar o NAT e, quando não dá, passar por um relay. O
SSH do Polvo corre por cima, como corre sobre TCP.

- `internal/transport/p2p.spec.md` — nasce: o transporte entre redes; a biblioteca (iroh, libp2p ou
  tsnet) decide-se nesta fase, com pesquisa

### ESRMS-W05 — Os ficheiros de qualquer máquina (depends on ESRMS-W02)

A árvore de ficheiros e o editor (que nascem no plano `abas.plan.md`, CEACN-W03, para a máquina local)
passam a servir os ficheiros de qualquer nó, pelo controlo que o nó já tem (`fs.list`, `fs.read`,
`fs.write`).

- `desktop/src/files.spec.md` — muda: a árvore de ficheiros de um nó
- `desktop/src/editor.spec.md` — muda: abrir, editar e gravar um ficheiro de um nó

### ESRMS-W06 — O agente e os sockets entre máquinas (depends on ESRMS-W02)

O `claude` da máquina local trabalha nas sessões das outras pelo `polvo mcp`, e os programas de uma
máquina falam com os serviços de outra por sockets encaminhados pelo Polvo. As permissões da máquina dona
valem para os dois.

- `internal/mcp/tools.spec.md` — muda: as ferramentas das sessões de qualquer nó (listar, ler o ecrã
  atual, escrever texto e teclas, esperar por uma saída)
- `internal/forward/forward.spec.md` — nasce: encaminhar um socket TCP ou Unix entre duas máquinas pelo
  nó, nos dois sentidos (como o `ssh -L` e o `ssh -R`)
- `internal/nodeconf/nodeconf.spec.md` — muda: a permissão de encaminhar sockets

## Out of Scope

- Gravar e reproduzir sessões.
- Copiar ficheiros grandes entre máquinas.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |
| `ESRMS-D01` | A biblioteca do transporte P2P: iroh (Rust, QUIC, relays), libp2p (Go/Rust) ou tsnet (Tailscale em Go) | o agente, com pesquisa, e o dono aprova | a fase ESRMS-W04 |

## Revisions

> **ESRMS-R0001:** as permissões entram no âmbito, pedidas pelo dono a 2026-10-17 depois de ter dito que
> todos escrevem: cada máquina decide, com uma configuração por omissão, se quem se liga de fora só vê ou
> controla, e pode travar uma sessão para só ver. Entra também a W06: o `claude` local nas sessões das
> outras máquinas pelo `polvo mcp`, e os sockets encaminhados entre máquinas.

> **ESRMS-R0002:** a árvore de ficheiros e o editor nascem no plano `abas.plan.md` (CEACN-W03), para a
> máquina local, quando o dono pediu os tipos de aba; a W05 passa a estendê-los às outras máquinas.

## Definition of Done

- Duas instâncias do Polvo (duas máquinas, ou duas instalações na mesma máquina) espelham a mesma
  sessão: o que uma escreve aparece nas duas, e quem chega a meio vê o ecrã atual; com a permissão de
  só ver, as teclas de quem se liga de fora não chegam à sessão.
- A janela encontra a outra máquina na rede local, liga-se a ela pelo endereço, e liga-se entre redes
  diferentes através do P2P.
- Um ficheiro da outra máquina abre, edita-se e grava-se na janela.
- O `claude` local, pelo `polvo mcp`, lê o ecrã de uma sessão da outra máquina e escreve nela; um socket
  da outra máquina responde por um encaminhamento local.
- `make test` passa com o `go test`, o `galaxy test` e os testes da janela; `anchors check` passa os
  gates bloqueantes.
