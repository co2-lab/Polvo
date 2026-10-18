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
URL de um), na porta UDP pedida (`DEFAULT_PORT` por omissão; outra quando está tomada); o `ID` é a chave pública, e o `Addr` diz onde a instalação se alcança agora.

### TRNSP-B02 — Ligar e aceitar
O `Dial` liga-se à instalação do endereço dado, e o `Accept` do `Listener` dela recebe a ligação; os bytes
passam nos dois sentidos, e o `CloseWrite` de um lado é o fim do que o outro lê. Fechar o `Peer` fecha o
`Listener`, e o `Accept` à espera dá `net.ErrClosed`.

### TRNSP-B05 — Para que é uma ligação
O `DialFor` abre uma ligação para o SSH (`KIND_SSH`, a do `Dial`) ou para levar o endereço novo de uma
instalação (`KIND_ADDR`), e a ligação aceite di-lo (`Kind`).

### TRNSP-B06 — A chave que um endereço nomeia
O `IsAddr` diz se um texto é um endereço do transporte (o JSON que uma instalação dá, ou `p2p:` e a
chave), e o `KeyOf` dá a chave que ele nomeia, para quem se liga exigir essa chave ao nó do outro lado.

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
| `port` | uma porta UDP, ou 0 | — | o `p2p/`: a tomada dá outra |
| `addr` | o endereço que o `Addr` de outra instalação deu | outro texto, ou uma instalação que não se alcança | o `p2p/`: o erro (`TRNSP-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `TRNSP-E01` | o endpoint não abre, ou a outra instalação não se alcança | o `Open` ou o `Dial` devolvem o erro do `p2p/` |
| `TRNSP-E02` | o executável foi compilado sem o transporte | o `Open` devolve o `ErrNotBuilt` |
| `TRNSP-E04` | o texto não nomeia a chave de uma instalação | o `KeyOf` devolve o erro com o texto |
| `TRNSP-E03` | quem usa a ligação pede um prazo | o `SetDeadline` e os irmãos devolvem o erro: o stream não tem prazos, e o SSH não os usa |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULT_RELAY` | `default` | os relays públicos do iroh, para funcionar entre redes sem configurar nada |
| `DEFAULT_PORT` | 2222 | o número da porta TCP do nó: quem passa um endereço à mão só precisa do IP |
| `KIND_SSH` | `P` | o primeiro byte de uma ligação para o SSH, o mesmo do `p2p/` |
| `KIND_ADDR` | `A` | o de uma ligação que leva um endereço novo |

## Rule uses

| Rule | Uses |
| --- | --- |
| `TRNSP-B01` | `key`, `relay`, `port`, `DEFAULT_RELAY`, `DEFAULT_PORT` |
| `TRNSP-B02` | `addr` |
| `TRNSP-B03` | `addr` |
| `TRNSP-B04` | `addr` |
| `TRNSP-B05` | `addr`, `KIND_SSH`, `KIND_ADDR` |
| `TRNSP-B06` | `addr` |
| `TRNSP-E04` | `addr` |
| `TRNSP-E01` | `relay`, `addr` |
| `TRNSP-E02` | `key` |
| `TRNSP-E03` | `addr` |

## Revisions

> **TRNSP-R0003:** entram o `TRNSP-B06` e o `TRNSP-E04` (a chave que um endereço nomeia), para a CLI e o
> agente se ligarem a um nó por um endereço do transporte.

> **TRNSP-R0002:** entra o `TRNSP-B05` (para que é uma ligação), como o `PTRNS-R0002` do `p2p/`.

> **TRNSP-R0001:** o `Open` recebe a porta (`DEFAULT_PORT`), pela mesma razão do `PTRNS-R0001` do `p2p/`.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
