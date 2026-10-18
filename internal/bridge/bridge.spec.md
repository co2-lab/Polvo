<!-- @anchors
  code: BRDGE
  updated_at: 2026-10-18
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
- `status`: onde o nó desta máquina escuta (`listen`) e, se não escuta, porquê (`listen_error`), a
  chave desta instalação como a linha que as outras máquinas juntam ao `authorized_keys` (`key`), e onde
  ela se alcança entre redes (`p2p`; e `p2p_error`, quando o transporte não abriu);
- `listen` (`addr`): o nó desta máquina passa a escutar as outras nesse endereço, ou deixa de escutar com
  ele vazio, e a instalação lembra-o;
- `machines`: as máquinas (`name`, `local`);
- `add` (`addr`) e `remove` (`name`): juntar e esquecer uma máquina;
- `add_p2p` (`addr`): juntar uma máquina de outra rede pelo endereço que a instalação dela deu, e dá o
  nome dela (`name`);
- `sessions` (`machine`): as sessões dela, como o `hub.Session`;
- `new` (`machine`, `argv`, `dir`, `rows`, `cols`): cria uma sessão e dá o `id`;
- `close` e `lock` (`machine`, `id`, `locked`): termina, trava ou destrava uma sessão;
- `progress` (`machine`): o progresso dos agentes dela, com a versão;
- `ask` (`machine`, `agent`, `text`): pergunta a um agente dela, e dá o id da pergunta;
- `progress_all` (`since`, a versão que a janela tem de cada máquina, e `timeout_ms`): o progresso de
  todas as máquinas, quando alguma mudou desde a versão dada, ou quando passa o prazo;
- `discover` (`timeout_ms`): as máquinas que se anunciam na rede, cada uma com se já é conhecida;
- `pairings`: os pedidos de emparelhamento que esta máquina mostra, com o código;
- `pair_start` (`addr`), `pair_code` (`id`, `code`) e `pair_cancel` (`id`): emparelhar com uma máquina;
- `fs_list` (`machine`, `path`): as entradas de uma pasta dela (`name`, `dir`, `size`);
- `fs_read` (`machine`, `path`): o texto de um ficheiro dela;
- `fs_write` (`machine`, `path`, `text`): grava o texto num ficheiro dela.

A estação abre-se a anunciar-se e a procurar no mDNS da rede (o `Open`).

## Rules

### BRDGE-B01 — Abrir e fechar uma estação
O `Open` abre a estação de uma pasta (com o endereço para escutar, a shell, e o relay e a porta do
transporte entre redes) e dá um número que nenhuma
outra estação ou terminal aberto tem. O `Close` fecha-a com os terminais dela, e o número deixa de valer.

### BRDGE-B02 — Os pedidos sobre uma estação
O `Call` atende os pedidos da lista acima e devolve o resultado em JSON.

### BRDGE-B03 — Um terminal ligado a uma sessão
O `Attach` liga um terminal a uma sessão de uma máquina e dá o número dele. O `Read` dá o que a sessão
mostra (primeiro o ecrã atual), à medida que chega, e o fim (zero bytes) quando o terminal acaba; o
`Write` escreve na sessão; o `Resize` muda-lhe o tamanho; o `Detach` larga o terminal, e a sessão
continua.

### BRDGE-B04 — O progresso dos agentes de todas as máquinas
O `progress_all` volta logo quando alguma máquina já mudou desde a versão dada (uma que a janela ainda não
conhece mudou sempre), e senão espera pela primeira mudança de qualquer uma, ou pelo prazo; dá todas as
máquinas, cada uma com a versão e os agentes, ou o erro dela.

### BRDGE-B05 — Emparelhar em dois passos
O `pair_start` começa a emparelhar com a máquina do endereço e dá o id do emparelhamento logo; o
`pair_code` dá-lhe o código que o utilizador leu na outra máquina e espera pelo fim: nada, quando emparelhou
e a máquina se juntou, ou o erro. O `pair_cancel` desiste.

### BRDGE-B06 — Os ficheiros de uma máquina
O `fs_list`, o `fs_read` e o `fs_write` são os pedidos de ficheiros do nó da máquina; o `fs_read` de um
ficheiro que não é texto em UTF-8 dá o erro.

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
| `BRDGE-B04` | `Handle`, `request` |
| `BRDGE-B05` | `Handle`, `request` |
| `BRDGE-B06` | `Handle`, `request` |

## Revisions

> **BRDGE-R0005:** o `Open` recebe o relay e a porta do transporte, o `status` diz o endereço entre redes, e
> entra o `add_p2p`, do plano ESRMS, W04.

> **BRDGE-R0004:** entra o pedido `listen`, para a janela aceitar as outras máquinas só quando o utilizador
> o liga (o dono decidiu-o a 2026-10-18).

> **BRDGE-R0003:** entram os ficheiros de uma máquina (`fs_list`, `fs_read`, `fs_write`, e o `BRDGE-B06`),
> do plano ESRMS, W05.

> **BRDGE-R0002:** entram a descoberta e o emparelhamento (`discover`, `pairings`, `pair_*`, e o
> `BRDGE-B05`), do plano ESRMS, W03.

> **BRDGE-R0001:** entram os pedidos do progresso dos agentes (`progress`, `ask`, `progress_all`, e o
> `BRDGE-B04`), do plano AGPRG, W04.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
