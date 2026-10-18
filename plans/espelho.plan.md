<!-- @anchors
  code: ESRMS
  updated_at: 2026-10-18
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

- `p2p/` (Rust) — nasce: o transporte entre redes sobre o iroh, com a chave ed25519 da instalação como
  endereço, e uma API C mínima (escutar, ligar-se, aceitar) que o nó em Go liga por cgo e o desktop usa
  em Rust
- `internal/transport/p2p.spec.md` — nasce: o lado Go do transporte, que dá ao nó e ao cliente uma
  `net.Conn` por cima da qual o SSH corre como corre sobre TCP
- `internal/pairing/pairing.spec.md` — muda: emparelhar pelo P2P, e o pedido num QR code (a chave do nó e
  o endereço P2P), para um telemóvel que não está na mesma rede
- `internal/transport/addresses.spec.md` — nasce: quando os IPs mudam, as duas máquinas não se perdem.
  Cada uma guarda o histórico dos endereços de cada máquina com que falou (não só o último: quando e quantas
  vezes respondeu em cada um, até um limite), e dá-lhe os seus sempre que mudam.
  Por ordem:
  1. com a ligação viva, o iroh passa-a para o endereço novo sem a cortar;
  2. a máquina cujo IP mudou ainda tem o endereço da outra: liga-se a ela e manda-lhe os seus novos,
     assinados pela chave dela (o automático, pedido pelo dono; funciona sem servidor nenhum);
  3. ao perder a ligação, tenta também os endereços antigos do histórico, os mais recentes e os mais
     usados primeiro, alguns de cada vez: uma máquina que alterna entre duas redes (casa e escritório,
     Wi-Fi e dados) volta a ser achada num endereço onde já esteve (pedido pelo dono). Só vale a máquina
     que provar a chave no handshake: um IP antigo que agora é de outra máquina só falha;
  4. com um relay ou o DNS do iroh, cada uma acha a outra pela chave, mesmo que as duas tenham mudado;
  5. quando nada disto chega (as duas mudaram, e não há servidor), o manual: a janela avisa que perdeu
     a ligação com a máquina e mostra, nesse aviso, os endereços desta máquina agora (de cada rede) e o
     último que conhecia da outra, com onde escrever o novo; como a chave da outra já é conhecida, não é
     preciso outro convite
- `desktop/src/nodes.spec.md` — muda: o aviso da ligação perdida, os endereços desta máquina, e mudar o
  endereço de uma máquina à mão

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

none

## Revisions

> **ESRMS-R0001:** as permissões entram no âmbito, pedidas pelo dono a 2026-10-17 depois de ter dito que
> todos escrevem: cada máquina decide, com uma configuração por omissão, se quem se liga de fora só vê ou
> controla, e pode travar uma sessão para só ver. Entra também a W06: o `claude` local nas sessões das
> outras máquinas pelo `polvo mcp`, e os sockets encaminhados entre máquinas.

> **ESRMS-R0002:** a árvore de ficheiros e o editor nascem no plano `abas.plan.md` (CEACN-W03), para a
> máquina local, quando o dono pediu os tipos de aba; a W05 passa a estendê-los às outras máquinas.

> **ESRMS-R0003:** a W02 fez-se assim, por razões que só a implementação mostrou: a instalação vista pela
> janela é uma unidade sua (`internal/station`), que corre o nó desta máquina no processo e chega a todas as
> máquinas pelo mesmo cliente; a API C é a ponte em Go puro (`internal/bridge`, testada no `go test` e no
> Galaxy), e a `cmd/libpolvo` só a exporta, porque o cgo não corre nos testes de Go nem no Galaxy: a forma
> de atravessar para C ficou na spec do lado que a testa, o `desktop/src/remote.spec.md`. Os terminais
> desta máquina passam pelo nó dela no Linux e no macOS; no Windows, até o pty do Go servir o ConPTY (o
> `creack/pty` só serve o Unix; o candidato é o `aymanbagabas/go-pty`) e o cgo ter o compilador de C do
> Windows, a janela corre os terminais nos ptys dela, sem espelho. Isso fica por fazer, numa fase do
> Windows.

> **ESRMS-R0004:** a W05 e a W06 fizeram-se assim. Na W05, a árvore e o editor ganharam a origem dos
> ficheiros (`FileSource`: esta máquina ou o nó de outra), o `~` passou a ser a pasta pessoal no controlo
> do nó, e a aba das máquinas abre a árvore de cada uma. Na W06, as ferramentas do agente pediram ao nó o
> ecrã de uma sessão em texto e escrever nela sem se ligar (`SSSNS`, `CNTRL`, `CLNTI`); o encaminhamento
> fez-se no servidor do nó (`SRVRE-B13`), no cliente (`CLNTI-B11`) e no `internal/forward`, e ganhou o
> `polvo forward` na CLI para se poder usar. Por omissão só o dono encaminha (`NDCNN-D01`, por rever
> pelo dono). Na janela, o encaminhamento ainda não tem aba: fica para quando o dono o pedir.

> **ESRMS-R0005:** a `ESRMS-D01` resolveu-se pelo iroh (Rust), aprovado pelo dono a 2026-10-18 depois de
> o agente comparar as bibliotecas de todas as línguas: o iroh 1.0 (estável desde junho de 2026) endereça
> pela chave ed25519 (a mesma que o SSH já verifica), fura o NAT perto de 9 em 10 vezes, e o relay dele é
> sem estado e alojável por nós; tem bindings oficiais para Swift e Kotlin, para a versão mobile que o dono
> quer fazer (controlar as sessões das máquinas pelo telemóvel). O pion/ice pedia uma sinalização e um
> TURN nossos; o go-libp2p fura menos (70% medido, condicional); o tsnet pede um servidor de coordenação.
> O nó em Go liga-o por cgo, por uma API C nossa (as bindings Go da comunidade só servem o Linux); o
> Galaxy ganha o cgo numa caixa do Bavel.

> **ESRMS-R0007:** a W04 fez-se assim: o transporte é o `p2p/` (o iroh) com a API C, o lado Go é o
> `internal/transport` (a etiqueta `polvo_p2p`), e a estação liga-se às máquinas pela chave; o endereço que
> muda (`ESRMS-R0006`) ficou na estação (`STATN-B15` a `STATN-B17`), e não num `addresses.spec.md`
> próprio. Emparelhar pelo P2P é o mesmo emparelhamento por outra ligação (`STATN-B18`); o QR code sai
> desta fase: não há ainda quem o leia (é para o telemóvel), e as formas de passar um convite, com o QR,
> são do plano da identidade (`IDNTD`, os convites).

> **ESRMS-R0006:** o dono pediu a 2026-10-18 que as máquinas não se percam quando os IPs mudam: a que
> mudou e ainda conhece a outra manda-lhe os endereços novos sozinha; e, quando isso não chega, os
> utilizadores mudam-nos à mão, com a janela a mostrar os endereços novos, sobretudo no aviso de que a
> ligação se perdeu. Entra na W04 (`internal/transport/addresses.spec.md`). Pediu também que se guarde o
> histórico dos endereços da outra máquina e que, ao perder a ligação, se tentem os antigos com um
> handshake: uma máquina que alterna entre duas redes volta a ser achada.

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
