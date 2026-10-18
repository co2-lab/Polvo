<!-- @anchors
  code: KYSEX
  updated_at: 2026-10-18
-->
# Keys — a identidade de uma instalação, quem entra num nó e que nós o desktop conhece

> **Code**: `KYSEX`

## Overview
Cada instalação do Polvo tem um par de chaves que a identifica: o desktop prova com ele quem é a um nó,
e o nó prova com o seu quem é ao desktop. O nó decide quem entra por uma lista de chaves autorizadas. O
desktop lembra-se da chave de cada nó a que já se ligou e recusa um nó cuja chave mudou. Os ficheiros
seguem os formatos do OpenSSH, para que as chaves e as listas sirvam também às ferramentas de sempre.

## Rules

### KYSEX-B01 — A identidade nasce uma vez e é reutilizada
A primeira vez que uma instalação pede a sua identidade, nasce um par de chaves ed25519 novo e fica
guardado no caminho pedido. As vezes seguintes devolvem o mesmo par. Quem pede fica a saber se a
identidade acabou de nascer. É o `LoadOrCreateIdentity`.

### KYSEX-B02 — A chave privada só é legível pelo dono
O ficheiro da chave privada nasce legível e gravável só pelo dono, e a pasta que o guarda, quando é
criada, só é acessível pelo dono.

### KYSEX-B08 — A chave em bruto
O `RawKey` dá a chave privada ed25519 da identidade (criando-a como o `LoadOrCreateIdentity`, se não
existe), para o transporte entre redes, que usa a chave e não um assinante do SSH. Uma identidade que não é
ed25519 é um erro (`KYSEX-E03`).

### KYSEX-B03 — Entra quem está na lista de chaves autorizadas
Um nó aceita uma chave que está na lista de chaves autorizadas e recusa uma que não está. As linhas
vazias e os comentários da lista são ignorados. A lista é lida de novo em cada verificação, por isso
uma chave acrescentada ou retirada vale para a ligação seguinte, sem reiniciar o nó. A lista é o
`AuthorizedKeys` (`NewAuthorizedKeys`, `Allows`), e uma linha para ela é o `AuthorizedLine`.

### KYSEX-B04 — Uma linha malformada não tranca as outras
Uma linha da lista que não é uma chave é saltada e comunicada a quem a usa (o número da linha e o
motivo, pelo `Warn`). As outras linhas continuam a valer.

### KYSEX-B05 — O primeiro contacto com um nó regista a chave dele
Quando o desktop se liga a um nó que nunca viu, aceita a ligação e regista a chave do nó na lista de
nós conhecidos. A lista é o `KnownHosts` (`NewKnownHosts`, `Check`).

### KYSEX-B06 — A mesma chave volta a ser aceite
Quando o desktop se liga outra vez a um nó conhecido e a chave é a registada, a ligação é aceite e a
lista não muda.

### KYSEX-B07 — A chave mudada é recusada
Quando o desktop se liga a um nó conhecido e a chave não é a registada, a ligação é recusada com o erro
de chave mudada (`HostKeyChangedError`, com o nó), e a lista não muda: só o dono decide trocar a chave
de um nó, editando a lista.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `path` | um caminho de ficheiro onde o utilizador pode escrever | um caminho numa pasta sem permissão | este módulo: devolve o erro do sistema com o caminho (`KYSEX-E02`) |
| `key` | uma chave pública SSH já validada pelo protocolo | uma chave nula | o transporte SSH, que só chama a verificação com a chave que o par apresentou |
| `hostport` | o endereço com que o desktop chamou o nó (máquina e porta) | um endereço vazio | quem liga: o endereço é o que foi pedido para a ligação |

## Invariants

| Rule | Always holds | How it is proven |
| --- | --- | --- |
| `KYSEX-I01` | a linha de chave autorizada que este módulo escreve para uma chave pública é aceite pela lista para essa mesma chave | closed cycle: escrever a linha da chave na lista e verificar a chave |
| `KYSEX-I02` | a chave registada no primeiro contacto é a aceite numa lista de nós conhecidos lida de novo do mesmo ficheiro | closed cycle: registar pelo primeiro contacto e verificar com uma lista nova sobre o mesmo ficheiro |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `KYSEX-E01` | num sistema Unix, o ficheiro da chave privada é legível ou gravável por outros além do dono | a identidade é recusada com o erro de permissões (`PermissionError`, com o caminho e as permissões encontradas) |
| `KYSEX-E02` | o ficheiro da chave privada existe mas não é uma chave, ou não se consegue ler nem criar | erro com o caminho do ficheiro |
| `KYSEX-E03` | a lista de chaves autorizadas não existe | ninguém entra, e não é um erro: um nó sem lista está fechado |
| `KYSEX-E03` | a identidade não é uma chave ed25519 | o `RawKey` devolve o erro com o ficheiro |

## Rule uses

| Rule | Uses |
| --- | --- |
| `KYSEX-B01` | `path` |
| `KYSEX-B02` | `path` |
| `KYSEX-B03` | `path`, `key` |
| `KYSEX-B04` | `path` |
| `KYSEX-B05` | `path`, `hostport`, `key` |
| `KYSEX-B06` | `path`, `hostport`, `key` |
| `KYSEX-B07` | `path`, `hostport`, `key` |
| `KYSEX-E01` | `path` |
| `KYSEX-E02` | `path` |
| `KYSEX-E03` | `path`, `key` |
| `KYSEX-I01` | `path`, `key` |
| `KYSEX-I02` | `path`, `hostport`, `key` |
| `KYSEX-B08` | `path` |
| `KYSEX-E03` | `path` |

## Revisions

> **KYSEX-R0001:** entra o `KYSEX-B08` (a chave em bruto), para o transporte entre redes (plano ESRMS, W04).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
