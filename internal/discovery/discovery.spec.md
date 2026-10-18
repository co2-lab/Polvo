<!-- @anchors
  code: DSCVR
  updated_at: 2026-10-18
-->
# Discovery — os nós da rede local, por mDNS

> **Code**: `DSCVR`

## Overview
O nó anuncia-se na rede local pelo DNS-SD sobre mDNS (RFC 6762 e 6763), com o tipo de serviço
`_polvo._tcp` (`SERVICE`), e a janela encontra os nós da rede pelo mesmo protocolo, o que o avahi e o
Bonjour também falam. O `Announce` responde, até ao `Close` do `Announcer`, às perguntas pelo serviço, com o
serviço (`Service`): o nome da instância, a máquina, a porta, os endereços e a impressão digital da chave
do nó. O `Browse` pergunta e dá os nós que respondem até ao prazo (`Found`). O grupo de multicast e a
interface são da configuração (`Options`), com o grupo do mDNS por omissão (`GROUP`).

## Rules

### DSCVR-B01 — Anunciar
O anúncio responde a uma pergunta pelo `_polvo._tcp.local` com o registo PTR da instância, o SRV (a
máquina e a porta), o TXT (a impressão digital, `fp=`, e o nome, `name=`) e os A dos endereços.

### DSCVR-B02 — Procurar
O `Browse` pergunta pelo serviço no grupo e dá, até ao prazo, os nós que respondem, um por instância,
cada um com o nome, o endereço (máquina e porta) e a impressão digital.

### DSCVR-B03 — A quem se responde
Uma pergunta que vem da porta do mDNS (como as do avahi) tem a resposta no grupo; uma de outra porta tem-na
no endereço de onde veio (a resposta unicast do RFC 6762, §6.7).

### DSCVR-B04 — O endereço de um nó
O endereço de um nó encontrado é o primeiro A da resposta com a porta do SRV; sem A, é o de onde veio a
resposta.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Options` | um grupo de multicast e uma interface (ou todas) | um grupo onde não se pode escutar | este módulo: o erro (`DSCVR-E01`) |
| `packet` | uma mensagem DNS | outra coisa | este módulo: ignora-a (`DSCVR-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `DSCVR-E01` | o grupo ou a interface não se abrem | o `Announce` dá o erro, e nada fica a escutar |
| `DSCVR-E02` | chega um pacote que não é uma mensagem DNS, ou uma resposta de outro serviço | ignora-se, e o resto continua |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `SERVICE` | `_polvo._tcp.local.` | o tipo de serviço DNS-SD do nó do Polvo |
| `GROUP` | `224.0.0.251:5353` | o grupo e a porta do mDNS em IPv4 |

## Rule uses

| Rule | Uses |
| --- | --- |
| `DSCVR-B01` | `SERVICE`, `packet` |
| `DSCVR-B02` | `Options`, `SERVICE` |
| `DSCVR-B03` | `Options`, `GROUP` |
| `DSCVR-B04` | `packet` |
| `DSCVR-E01` | `Options` |
| `DSCVR-E02` | `packet` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
