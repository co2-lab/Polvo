<!-- @anchors
  code: PTRNS
  updated_at: 2026-10-18
-->
# P2P — o transporte entre redes: uma instalação liga-se a outra pela chave dela

> **Code**: `PTRNS`

## Overview
Cada instalação tem um endpoint do iroh (`Peer`, aberto pelo `bind`) cujo endereço é a chave ed25519 dela:
a mesma que o SSH já verifica. O endpoint fura o NAT quando dá, e passa por um relay (`Relay`) quando não
dá. Onde uma instalação se alcança agora (`addr`) é a chave, os endereços diretos e o relay. Uma ligação
(`Stream`, pelo `connect` de um lado e pelo `accept` do outro) é um stream QUIC nos dois sentidos, com a
chave do outro lado (`remote`), por cima do qual corre o SSH do Polvo, como corre sobre TCP. A API C
(`ffi`) dá o mesmo ao nó em Go: endpoints e streams são números.

## Rules

### PTRNS-B01 — O endpoint de uma instalação
O `bind` abre o endpoint da chave cujo segredo são os 32 bytes dados, com o relay pedido ("default": os
públicos do iroh; nenhum; ou um URL), e aceita só as ligações do Polvo (`ALPN`). O `id` é a chave pública;
o `addr` diz, em JSON, a chave, os endereços diretos e o relay.

### PTRNS-B02 — Ligar e aceitar
O `connect` liga-se à instalação do `addr` que ela deu, e o `accept` dela recebe a ligação; o que um lado
escreve o outro lê, nos dois sentidos, e o fim de um lado é o fim do que o outro lê.

### PTRNS-B03 — A chave do outro lado
Cada lado sabe a chave do outro (`remote`), que o handshake provou: ligar-se a um `addr` com uma chave que
não é a de quem responde falha.

### PTRNS-B04 — A API C
As mesmas operações em C, para o nó em Go: abrir o endpoint (`polvo_p2p_open`), o endereço e a chave
(`polvo_p2p_addr`, `polvo_p2p_id`), aceitar e ligar (`polvo_p2p_accept`, `polvo_p2p_connect`), ler e
escrever (`polvo_p2p_read`, `polvo_p2p_write`), dizer que não se escreve mais (`polvo_p2p_close_write`),
fechar o stream (`polvo_p2p_close`, que espera até `CLOSE_WAIT` que o outro lado tenha o que se escreveu)
e o endpoint (`polvo_p2p_shutdown`, depois do qual o `accept` à espera dá 0), e libertar o texto que deram
(`polvo_p2p_free`).

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `seed` | os 32 bytes do segredo de uma chave ed25519 | outro tamanho | quem chama |
| `relay` | "default", vazio, ou o URL de um relay | um texto que não é URL | este módulo: o erro (`PTRNS-E02`) |
| `addr` | o JSON que o `addr` de outra instalação deu | outro texto | este módulo: o erro (`PTRNS-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `PTRNS-E01` | o `addr` não é um endereço, ou a instalação não se alcança por nenhum caminho | o `connect` devolve o erro com o motivo |
| `PTRNS-E02` | o relay não é um URL, ou o endpoint não abre | o `bind` devolve o erro com o motivo |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `ALPN` | `polvo/ssh/1` | o protocolo de uma ligação do Polvo, com a versão: o endpoint recusa os outros |
| `CLOSE_WAIT` | 5 s | o que fechar espera pelo outro lado: uma rede lenta tem tempo de entregar o fim, e um lado que sumiu não prende nada |

## Rule uses

| Rule | Uses |
| --- | --- |
| `PTRNS-B01` | `seed`, `relay`, `ALPN` |
| `PTRNS-B02` | `addr` |
| `PTRNS-B03` | `addr` |
| `PTRNS-B04` | `seed`, `relay`, `addr`, `CLOSE_WAIT` |
| `PTRNS-E01` | `addr` |
| `PTRNS-E02` | `relay` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
