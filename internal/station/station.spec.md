<!-- @anchors
  code: STATN
  updated_at: 2026-10-18
-->
# Station — a instalação do Polvo vista pela janela: o nó desta máquina e as outras máquinas

> **Code**: `STATN`

## Overview
A `Station` é o que a janela usa para chegar às sessões de terminal de qualquer máquina. O `Open` abre a
instalação de uma pasta (as chaves, a lista de chaves autorizadas, a configuração do nó, as máquinas
conhecidas) e corre o nó desta máquina dentro do processo, com as sessões dela. A janela chega a todas as
máquinas pelo mesmo caminho, o cliente (`hub.Node`): à sua (`LOCAL`) por uma ligação dentro do processo,
como dona, e às outras pela rede. O nó desta máquina escuta também no endereço pedido (`Options`), para
as outras máquinas espelharem as sessões dela com as permissões do `node.json`; o `PublicKey` é a chave
desta instalação, a que as outras máquinas autorizam.

As máquinas que o utilizador junta (`Machine`) ficam gravadas em `machines.json` e voltam no `Open`
seguinte; a ligação a cada uma faz-se quando é usada.

## Rules

### STATN-B01 — A máquina desta janela
O `Node(LOCAL)` dá o nó desta máquina, como dono: cria, lista, trava e termina sessões, e liga-se a elas.
As sessões vivem enquanto a `Station` está aberta; o `Close` termina-as, deixa de escutar e fecha as
ligações das outras máquinas.

### STATN-B02 — As outras máquinas espelham esta
Com um endereço para escutar, o nó desta máquina aceita ligações de outras instalações pela rede
(`ListenAddr`), com as chaves autorizadas e as permissões do `node.json` (também a de encaminhar
sockets), como o `polvo node`.

### STATN-B03 — Juntar uma máquina
O `Add` liga-se a uma máquina pelo endereço e, quando liga, junta-a às conhecidas e grava-a. O `Machines`
dá a máquina desta janela primeiro e depois as conhecidas, pela ordem em que se juntaram; um `Open`
seguinte dá as mesmas.

### STATN-B04 — Esquecer uma máquina
O `Remove` fecha a ligação a uma máquina conhecida e esquece-a, também no ficheiro.

### STATN-B05 — Ligar quando se usa, e outra vez quando caiu
O `Node` de uma máquina conhecida liga-se a ela da primeira vez que é pedido, e dá a mesma ligação
enquanto está viva; depois de a ligação cair, o pedido seguinte liga outra vez.

### STATN-B06 — O progresso dos agentes desta máquina
O nó desta máquina guarda o progresso dos agentes dela (`progress`) na pasta `progress` da instalação: o
que um agente reporta pelo controlo fica lá, e volta no `Open` seguinte.

### STATN-B07 — As sessões sabem onde está o nó
Cada sessão desta máquina recebe no ambiente o endereço onde o nó dela a serve de dentro da máquina
(`POLVO_NODE`), para um agente que corre nela reportar o progresso a este nó com a chave da instalação.

### STATN-B08 — O nó desta máquina emparelha
O nó desta máquina deixa uma instalação que não conhece emparelhar (`pairing`), e os pedidos que mostra
ficam no `Pairings` (o nome, a impressão digital e o código) até acabarem, bem ou mal.

### STATN-B09 — Anunciar-se na rede
A escutar num endereço que as outras máquinas alcançam (não o loopback), o nó desta máquina anuncia-se na
rede local (`discovery`), com o nome da máquina, a porta onde
escuta e a impressão digital da chave dele (`HostFingerprint`); o `Close` acaba o anúncio.

### STATN-B10 — Procurar as máquinas da rede
O `Discover` dá as máquinas que se anunciam na rede, sem esta instalação, cada uma com se já é conhecida
(`Discovered`).

### STATN-B11 — Emparelhar com uma máquina
O `Pair` emparelha esta instalação com a máquina de um endereço, pelo código que o utilizador lê nela, e
junta-a às máquinas conhecidas.

### STATN-B12 — Aceitar as outras máquinas, e deixar de aceitar
Com a janela aberta, o `Listen` põe o nó desta máquina a escutar as outras num endereço (e a anunciar-se,
como no `STATN-B09`), ou deixa de escutar com o endereço vazio; deixar de escutar fecha também as ligações
que as outras máquinas tinham feito, e as sessões continuam. A instalação lembra-o: o `Open` sem endereço
escuta onde o utilizador deixou (por omissão, em lado nenhum).

### STATN-B13 — As máquinas de outras redes, pela chave
O `AddP2P` junta a máquina do endereço que a instalação dela deu (o do transport), e chama-a pela chave:
`P2P_PREFIX` e a chave. Ligar-se a ela é pelo transporte entre redes, e o nó dela só é aceite se a chave do
SSH for essa mesma, que o transporte já provou.

### STATN-B14 — Esta máquina entre redes
A instalação abre o endpoint do transporte com a chave do nó desta máquina (o relay e a porta das
`Options`), e o `P2PAddr` diz onde se alcança. As outras instalações entram por ele só enquanto esta
máquina as aceita (`STATN-B12`): o mesmo interruptor serve a rede e as outras redes. Sem o transporte no
executável, tudo o resto funciona.

### STATN-B15 — O histórico dos endereços de cada máquina
Cada vez que se liga a uma máquina pela chave, a instalação guarda onde ela está agora (o `p2p.addr` do
nó dela) no histórico: os endereços, quando e quantas vezes cada um respondeu, até `MAX_HISTORY`, os mais
recentes primeiro. Ligar-se a ela dá ao transporte todos os do histórico de uma vez, e ele tenta-os todos:
uma máquina que alterna entre duas redes volta a ser achada num endereço onde já esteve.

### STATN-B16 — O endereço novo chega a quem conhece esta máquina
Quem se liga a uma máquina pela chave diz-lhe onde se alcança (o `p2p.seen` do nó dela), e ela guarda-o no
histórico. A cada `ADDR_EVERY`, cada instalação vê os seus endereços diretos; quando mudam, manda o
endereço novo a todas as do histórico (uma ligação do transporte que só o leva), porque a que mudou ainda
sabe o endereço das outras. Quem o recebe guarda-o no histórico, aceite ou não as outras máquinas.

### STATN-B17 — O endereço novo de uma máquina, à mão
Quando nada do resto chega (as duas máquinas mudaram, e não há relay), o utilizador passa os endereços à
mão. O `Addrs` dá onde esta instalação se alcança agora, um endereço por rede (`Reach`), com o nome da rede (para os
dar à outra máquina); o `Hint` recebe o endereço novo de uma máquina pela chave (um IP, com a porta ou com
a porta do transporte, ou o endereço que a janela dela copiou), põe-no no histórico, e a ligação seguinte
tenta-o.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `dir` | a pasta da instalação (criada se não existe) | uma que não se pode criar | este módulo: o `Open` devolve o erro |
| `Options` | o endereço onde escutar (vazio: não escuta), a shell, o nome da máquina e onde se anuncia (o `discovery.Options`) | um endereço ocupado | este módulo: a `Station` abre sem escutar (`STATN-E01`) |
| `addr` | máquina e porta de um nó, ou o endereço do transport de uma instalação | um nó que não responde ou recusa a chave | este módulo: o erro (`STATN-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `STATN-E01` | o endereço para escutar não se abre (ocupado, sem permissão) | a `Station` abre sem escutar, a máquina desta janela funciona, e o `ListenError` diz porquê |
| `STATN-E02` | a máquina não responde, ou recusa a chave desta instalação | o `Add` e o `Node` devolvem o erro com o endereço; o `Add` não a junta |
| `STATN-E04` | o endereço que chega não é de quem o mandou, ou vem de uma instalação que esta não conhece, ou passa de `MAX_ADDR` | é ignorado, e o histórico fica como estava |
| `STATN-E05` | o endereço do `Hint` não é um IP nem um endereço dessa máquina, ou a máquina não é pela chave | o `Hint` devolve o erro, e o histórico fica como estava |
| `STATN-E03` | o transporte não abre, ou o executável não o tem, e uma máquina é pela chave | o `P2PError` diz porquê; ligar-se a essa máquina dá o erro, e o resto funciona |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `LOCAL` | `local` | o nome da máquina desta janela, que não tem endereço |
| `P2P_PREFIX` | `p2p:` | o nome de uma máquina pela chave: não se confunde com um endereço máquina:porta |
| `ADDR_EVERY` | 10 s | uma mudança de rede chega às outras em segundos, e ver os endereços desta máquina não custa nada |
| `MAX_ADDR` | 64 KiB | um endereço tem uma chave e alguns caminhos; mais do que isto não é um endereço |
| `MAX_HISTORY` | 16 | as redes por onde uma máquina costuma andar cabem com folga, e tentar todas ao mesmo tempo não pesa |

## Rule uses

| Rule | Uses |
| --- | --- |
| `STATN-B01` | `dir`, `LOCAL` |
| `STATN-B02` | `Options` |
| `STATN-B03` | `addr`, `dir` |
| `STATN-B04` | `addr`, `dir` |
| `STATN-B05` | `addr` |
| `STATN-E01` | `Options` |
| `STATN-E02` | `addr` |
| `STATN-B06` | `dir` |
| `STATN-B07` | `dir` |
| `STATN-B08` | `dir` |
| `STATN-B09` | `Options` |
| `STATN-B10` | `Options` |
| `STATN-B11` | `addr` |
| `STATN-B12` | `Options`, `addr` |
| `STATN-B13` | `addr`, `P2P_PREFIX` |
| `STATN-B14` | `Options` |
| `STATN-B15` | `addr`, `MAX_HISTORY` |
| `STATN-E03` | `Options`, `addr` |
| `STATN-B16` | `addr`, `ADDR_EVERY` |
| `STATN-E04` | `addr`, `MAX_ADDR` |
| `STATN-B17` | `addr`, `P2P_PREFIX` |
| `STATN-E05` | `addr` |

## Revisions

> **STATN-R0006:** entram o `STATN-B17` e o `STATN-E05` (o endereço novo de uma máquina, à mão), a parte
> manual do `ESRMS-R0006`.

> **STATN-R0005:** entram o `STATN-B16` e o `STATN-E04` (o endereço novo chega a quem conhece esta máquina),
> a parte automática do `ESRMS-R0006`, pedida pelo dono. O plano dizia que nasceria um
> `internal/transport/addresses.spec.md`; fica aqui, porque é a estação que sabe as máquinas e o histórico.

> **STATN-R0004:** entram as máquinas de outras redes pela chave (`STATN-B13` a `STATN-B15`, `STATN-E03`),
> do plano ESRMS, W04.

> **STATN-R0003:** entra o `STATN-B12` (aceitar e deixar de aceitar as outras máquinas com a janela aberta),
> porque o dono decidiu a 2026-10-18 que a janela não escuta por omissão (era a `STATN-D01`): o utilizador
> liga-o na aba das máquinas.

> **STATN-R0002:** entram o emparelhamento, o anúncio e a procura na rede (`STATN-B08` a `STATN-B11`), do
> plano ESRMS, W03.

> **STATN-R0001:** entra o `STATN-B06` (o progresso dos agentes desta máquina), do plano AGPRG, W01, e o
> `STATN-B07` (as sessões sabem onde está o nó), da W02.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
