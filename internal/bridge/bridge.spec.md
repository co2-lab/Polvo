<!-- @anchors
  code: BRDGE
  updated_at: 2026-10-17
-->
# Bridge — as estações e os terminais por números, para a janela os usar através de C

> **Code**: `BRDGE`

## Overview
A ponte é a API que a biblioteca C (`cmd/libpolvo`) exporta para a janela, em Go puro, para correr igual
no `go test` e no `galaxy test`. Uma estação (`station.Station`) e um terminal ligado a uma sessão
(`hub.Terminal`) são números (`Handle`), porque um ponteiro de Go não atravessa para C. O `Open` abre uma
estação e dá o número dela; o `Call` atende um pedido em JSON sobre ela; o `Attach` liga um terminal a
uma sessão e dá o número dele, que o `Read`, o `Write`, o `Resize` e o `Detach` usam; o `Close` fecha a
estação e os terminais dela.

O `Call` recebe `{"op": …}` e devolve o resultado em JSON:
- `status`: onde o nó desta máquina escuta (`listen`) e, se não escuta, porquê (`listen_error`), e a
  chave desta instalação como a linha que as outras máquinas juntam ao `authorized_keys` (`key`);
- `machines`: as máquinas (`name`, `local`);
- `add` (`addr`) e `remove` (`name`): juntar e esquecer uma máquina;
- `sessions` (`machine`): as sessões dela, como o `hub.Session`;
- `new` (`machine`, `argv`, `dir`, `rows`, `cols`): cria uma sessão e dá o `id`;
- `close` e `lock` (`machine`, `id`, `locked`): termina, trava ou destrava uma sessão.

## Rules

### BRDGE-B01 — Abrir e fechar uma estação
O `Open` abre a estação de uma pasta (com o endereço para escutar e a shell) e dá um número que nenhuma
outra estação ou terminal aberto tem. O `Close` fecha-a com os terminais dela, e o número deixa de valer.

### BRDGE-B02 — Os pedidos sobre uma estação
O `Call` atende os pedidos da lista acima e devolve o resultado em JSON.

### BRDGE-B03 — Um terminal ligado a uma sessão
O `Attach` liga um terminal a uma sessão de uma máquina e dá o número dele. O `Read` dá o que a sessão
mostra (primeiro o ecrã atual), à medida que chega, e o fim (zero bytes) quando o terminal acaba; o
`Write` escreve na sessão; o `Resize` muda-lhe o tamanho; o `Detach` larga o terminal, e a sessão
continua.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Handle` | o número de uma estação ou de um terminal abertos | outro | este módulo: o erro (`BRDGE-E01`) |
| `request` | um pedido em JSON com um `op` da lista | um JSON inválido, ou outro `op` | este módulo: o erro (`BRDGE-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `BRDGE-E01` | o número não é de nada aberto, ou é do tipo errado | o erro diz que o número não vale |
| `BRDGE-E02` | o pedido não é JSON, ou o `op` não existe | o erro diz porquê, e nada muda |
| `BRDGE-E03` | a estação, a máquina ou o nó falham o pedido | o erro deles chega a quem chamou |

## Rule uses

| Rule | Uses |
| --- | --- |
| `BRDGE-B01` | `Handle` |
| `BRDGE-B02` | `Handle`, `request` |
| `BRDGE-B03` | `Handle` |
| `BRDGE-E01` | `Handle` |
| `BRDGE-E02` | `request` |
| `BRDGE-E03` | `request` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
