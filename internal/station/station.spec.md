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

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `dir` | a pasta da instalação (criada se não existe) | uma que não se pode criar | este módulo: o `Open` devolve o erro |
| `Options` | o endereço onde escutar (vazio: não escuta), a shell, o nome da máquina e onde se anuncia (o `discovery.Options`) | um endereço ocupado | este módulo: a `Station` abre sem escutar (`STATN-E01`) |
| `addr` | máquina e porta de um nó | um nó que não responde ou recusa a chave | este módulo: o erro (`STATN-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `STATN-E01` | o endereço para escutar não se abre (ocupado, sem permissão) | a `Station` abre sem escutar, a máquina desta janela funciona, e o `ListenError` diz porquê |
| `STATN-E02` | a máquina não responde, ou recusa a chave desta instalação | o `Add` e o `Node` devolvem o erro com o endereço; o `Add` não a junta |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `LOCAL` | `local` | o nome da máquina desta janela, que não tem endereço |

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

## Revisions

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
