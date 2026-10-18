<!-- @anchors
  code: RMTEC
  updated_at: 2026-10-18
-->
# Remote — as máquinas e as sessões de terminal, pela biblioteca do Polvo

> **Code**: `RMTEC`

## Overview
A janela chega às sessões de terminal desta máquina e das outras pela biblioteca do Polvo (`libpolvo`,
o Go ligado ao executável). O `Station::open` abre a instalação de uma pasta, com o nó desta máquina; o
`Station` dá as máquinas (`Machine`), junta-as e esquece-as, e dá as sessões de cada uma (`SessionInfo`),
cria-as, trava-as e termina-as; o estado do nó (`Status`) diz onde escuta. O `attach` liga um terminal
(`RemoteTerm`) a uma sessão: lê-se, escreve-se e muda de tamanho como um pty, e largá-lo deixa a sessão
viva. Um erro da biblioteca chega como o texto dela (`RemoteError`).

A biblioteca é a ponte de Go (`BRDGE`) compilada como biblioteca C estática (`go build
-buildmode=c-archive`, pelo `build.rs`) e ligada ao executável: um programa só, com as duas línguas. As
funções C (`polvo_open`, `polvo_close`, `polvo_call`, `polvo_attach`, `polvo_read`, `polvo_write`,
`polvo_resize`, `polvo_detach` e `polvo_free`) são as da ponte com o mesmo nome; atravessam assim:
- a instalação e os terminais são números (`int64_t`), e zero é falha;
- os textos entram e saem em UTF-8 terminado em zero, e o que a biblioteca dá larga-se com o `polvo_free`;
- um erro sai pelo último argumento (`char **err`), também a largar com o `polvo_free`;
- o `polvo_read` dá os bytes lidos, zero no fim do terminal, e menos um com o erro.

## Rules

### RMTEC-B01 — Abrir a instalação
O `Station::open` abre a instalação da pasta pedida; as máquinas começam pela desta janela (`LOCAL`), e o
estado diz onde o nó escuta.

### RMTEC-B02 — As sessões de uma máquina
O `new_session` cria uma sessão (a shell da instalação) com o tamanho pedido e dá o id; o `sessions` dá as
sessões, cada uma com se está travada e se esta janela pode escrever nela; o `lock` trava-a ou destrava-a;
o `close_session` termina-a.

### RMTEC-B03 — Um terminal de uma sessão
O `attach` dá um terminal da sessão: o `read` dá o que a sessão mostra, primeiro o ecrã atual; o `write`
escreve nela; o `resize` muda-lhe o tamanho. Largar o terminal (o `detach`, ou deixá-lo cair) deixa a
sessão viva, e um `read` à espera dá o fim.

### RMTEC-B04 — Juntar e esquecer uma máquina
O estado dá a chave desta instalação, como a linha que a outra máquina junta ao `authorized_keys`. Com
ela autorizada lá, o `add` junta a outra máquina pelo endereço, e as sessões dela passam a dar-se pelo
nome (o endereço); o `remove` esquece-a.

### RMTEC-B05 — O progresso dos agentes
O `progress_all` dá o progresso dos agentes de todas as máquinas (`MachineProgress`, com os trabalhos,
`AgentJob`, e os passos, `AgentStep`, o registo, `AgentEntry`, e as perguntas, `AgentQuestion`, de cada
um), assim que alguma mudou desde as versões que
a janela tem, ou quando passa o prazo; o `ask` pergunta a um agente de uma máquina e dá o id da pergunta.

### RMTEC-B06 — A rede e o emparelhamento
O `discover` dá as máquinas que se anunciam na rede (`NetMachine`, com se já são conhecidas); o `pairings`
dá os pedidos de emparelhamento que esta máquina mostra (`PairingRequest`, com o código); o `pair_start`
começa a emparelhar com a máquina de um endereço, o `pair_code` dá-lhe o código lido nela e espera pelo
fim, e o `pair_cancel` desiste.

### RMTEC-B07 — Os ficheiros de uma máquina
O `list_dir` dá as entradas de uma pasta de uma máquina (`RemoteEntry`), o `read_text` o texto de um
ficheiro dela, e o `write_text` grava-o.

### RMTEC-B08 — Aceitar as outras máquinas
O `set_listen` põe o nó desta máquina a escutar as outras no endereço dado, ou deixa de escutar com ele
vazio; a instalação lembra-o para a próxima vez que abre.

### RMTEC-B09 — As máquinas de outras redes
O `open_with` abre a instalação com o relay e a porta UDP (`P2P_PORT`) do transporte entre redes; o
`status` diz onde ela se alcança entre redes (`p2p`, e `p2p_error` quando o transporte não abriu); e o
`add_p2p` junta uma máquina pelo endereço que a instalação dela deu, e dá o nome dela. O executável leva a
API C do transporte (`p2p/`), que a biblioteca chama.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `machine` | `LOCAL`, ou uma máquina juntada | outra | a biblioteca: o erro dela (`RMTEC-E01`) |
| `id` | o id de uma sessão viva | outro | a biblioteca: o erro dela (`RMTEC-E01`) |
| `listen` | um endereço onde escutar (máquina:porta, ou `:porta`), ou vazio | um ocupado ou sem permissão | a biblioteca: o erro dela (`RMTEC-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `RMTEC-E01` | a biblioteca recusa o pedido (a máquina, a sessão, a ligação) | o erro com o texto dela |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `LOCAL` | `local` | o nome da máquina desta janela na biblioteca |
| `P2P_PORT` | 2222 | o número da porta do nó, em UDP: quem passa um endereço à mão só precisa do IP |

## Rule uses

| Rule | Uses |
| --- | --- |
| `RMTEC-B01` | `LOCAL` |
| `RMTEC-B02` | `machine`, `id` |
| `RMTEC-B03` | `machine`, `id` |
| `RMTEC-E01` | `machine`, `id` |
| `RMTEC-B04` | `machine` |
| `RMTEC-B05` | `machine` |
| `RMTEC-B06` | `machine` |
| `RMTEC-B07` | `machine` |
| `RMTEC-B08` | `listen` |
| `RMTEC-B09` | `listen`, `P2P_PORT` |

## Revisions

> **RMTEC-R0006:** entra o `RMTEC-B09` (as máquinas de outras redes), do plano ESRMS, W04.

> **RMTEC-R0005:** entra o `RMTEC-B08` (aceitar as outras máquinas), por a janela já não escutar por
> omissão, como o dono decidiu a 2026-10-18.

> **RMTEC-R0004:** entra o `RMTEC-B07` (os ficheiros de uma máquina), do plano ESRMS, W05.

> **RMTEC-R0003:** entra o `RMTEC-B06` (a rede e o emparelhamento), do plano ESRMS, W03.

> **RMTEC-R0002:** entra o `RMTEC-B05` (o progresso dos agentes), do plano AGPRG, W04.

> **RMTEC-R0001:** o `RMTEC-B03` passa a ter o `detach`, para o ecrã de uma sessão a largar enquanto a
> leitura dela ainda está à espera (o `Screen::attach`, plano ESRMS, W02).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
