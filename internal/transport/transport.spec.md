<!-- @anchors
  code: TRNSP
  updated_at: 2026-10-18
-->
# Transport — a ligação entre redes para o nó e para o cliente

> **Code**: `TRNSP`

## Overview
O nó e o cliente ligam-se entre redes pelo endpoint da instalação (`Peer`, aberto pelo `Open` com a chave
dela), que é o do `p2p/` (o iroh, em Rust) pela API C dele. O `Listener` dá as ligações das outras
instalações como um `net.Listener`, e o `Dial` liga-se a outra pelo endereço que ela deu (`Addr`); cada
ligação (`Conn`) é uma `net.Conn` com a chave do outro lado (`Remote`), e o endereço de uma ponta (`Addr`,
como `net.Addr`: o `Network` é "p2p" e o `String` é a chave). A `Conn` faz o que uma `net.Conn` faz
(`Read`, `Write`, `Close`, `LocalAddr`, `RemoteAddr`), menos os prazos (`SetDeadline`, `SetReadDeadline`,
`SetWriteDeadline`, `TRNSP-E03`). O SSH do Polvo corre por cima,
como corre sobre TCP. O transporte entra no executável com a etiqueta de compilação `polvo_p2p` (e o
cgo); sem ela, o `Open` diz que não está (`ErrNotBuilt`).

## Rules

### TRNSP-B01 — O endpoint da instalação
O `Open` abre o endpoint da chave ed25519 da instalação, com o relay pedido (`DEFAULT_RELAY`, nenhum, ou o
URL de um); o `ID` é a chave pública, e o `Addr` diz onde a instalação se alcança agora.

### TRNSP-B02 — Ligar e aceitar
O `Dial` liga-se à instalação do endereço dado, e o `Accept` do `Listener` dela recebe a ligação; os bytes
passam nos dois sentidos, e o `CloseWrite` de um lado é o fim do que o outro lê. Fechar o `Peer` fecha o
`Listener`, e o `Accept` à espera dá `net.ErrClosed`.

### TRNSP-B03 — A chave do outro lado
Cada ligação sabe a chave do outro lado (`Remote`, e o `RemoteAddr`), que o handshake provou.

### TRNSP-B04 — O SSH por cima
O nó serve uma ligação aceite como serve uma de TCP, e o cliente liga-se pela que o `Dial` deu: um comando
corre no nó do outro lado.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `key` | a chave ed25519 da instalação | nenhuma | quem chama |
| `relay` | `DEFAULT_RELAY`, vazio, ou o URL de um relay | outro texto | o `p2p/`: o erro (`TRNSP-E01`) |
| `addr` | o endereço que o `Addr` de outra instalação deu | outro texto, ou uma instalação que não se alcança | o `p2p/`: o erro (`TRNSP-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `TRNSP-E01` | o endpoint não abre, ou a outra instalação não se alcança | o `Open` ou o `Dial` devolvem o erro do `p2p/` |
| `TRNSP-E02` | o executável foi compilado sem o transporte | o `Open` devolve o `ErrNotBuilt` |
| `TRNSP-E03` | quem usa a ligação pede um prazo | o `SetDeadline` e os irmãos devolvem o erro: o stream não tem prazos, e o SSH não os usa |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULT_RELAY` | `default` | os relays públicos do iroh, para funcionar entre redes sem configurar nada |

## Rule uses

| Rule | Uses |
| --- | --- |
| `TRNSP-B01` | `key`, `relay`, `DEFAULT_RELAY` |
| `TRNSP-B02` | `addr` |
| `TRNSP-B03` | `addr` |
| `TRNSP-B04` | `addr` |
| `TRNSP-E01` | `relay`, `addr` |
| `TRNSP-E02` | `key` |
| `TRNSP-E03` | `addr` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
