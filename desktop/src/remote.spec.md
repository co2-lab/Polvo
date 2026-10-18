<!-- @anchors
  code: RMTEC
  updated_at: 2026-10-17
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

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `machine` | `LOCAL`, ou uma máquina juntada | outra | a biblioteca: o erro dela (`RMTEC-E01`) |
| `id` | o id de uma sessão viva | outro | a biblioteca: o erro dela (`RMTEC-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `RMTEC-E01` | a biblioteca recusa o pedido (a máquina, a sessão, a ligação) | o erro com o texto dela |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `LOCAL` | `local` | o nome da máquina desta janela na biblioteca |

## Rule uses

| Rule | Uses |
| --- | --- |
| `RMTEC-B01` | `LOCAL` |
| `RMTEC-B02` | `machine`, `id` |
| `RMTEC-B03` | `machine`, `id` |
| `RMTEC-E01` | `machine`, `id` |
| `RMTEC-B04` | `machine` |

## Revisions

> **RMTEC-R0001:** o `RMTEC-B03` passa a ter o `detach`, para o ecrã de uma sessão a largar enquanto a
> leitura dela ainda está à espera (o `Screen::attach`, plano ESRMS, W02).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
