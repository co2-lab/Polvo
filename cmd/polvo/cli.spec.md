<!-- @anchors
  code: CLIXX
  updated_at: 2026-10-18
-->
# Cli — os subcomandos do `polvo`: a identidade, o nó e a ligação

> **Code**: `CLIXX`

## Overview
O executável `polvo` é um só, com subcomandos. O `keygen` cria a identidade da instalação, o `node`
põe a máquina a servir como nó, o `connect` abre um terminal num nó a partir do terminal local, o
`mcp` é a ponte do agente local, e o `polvo` sem subcomando (ou o `ui`) abre a aplicação desktop. A
pasta de configuração (a identidade, a chave do nó, as chaves autorizadas, os nós conhecidos) é a
`polvo` dentro da pasta de configuração do utilizador, ou a que a opção `--dir` diz. O ponto de entrada
testável é o `Run`, com a entrada, a saída e o erro (`Stdio`).

## Rules

### CLIXX-B01 — `keygen` cria a identidade e mostra a chave pública
O `polvo keygen` cria a identidade na pasta de configuração, se ainda não existe, e escreve na saída a
linha de chave autorizada da chave pública, para ser acrescentada à lista de um nó. Uma segunda
execução mostra a mesma linha.

### CLIXX-B02 — `node` serve no endereço pedido com a pasta de configuração
O `polvo node` serve como nó no endereço da opção `--listen` (por omissão ":2222"), com a chave do nó e
a lista de chaves autorizadas da pasta de configuração; a chave do nó nasce na primeira vez. A shell das sessões é a da variável `SHELL`. Escreve na
saída uma linha com o endereço em que ficou a escutar e a impressão digital da chave do nó.

As shells do nó são sessões que vivem sem quem as vê, e o que cada chave pode fazer nelas vem do
`node.json` da pasta de configuração; a chave da própria instalação (a identidade `id_ed25519`) é a
dona. Um `node.json` estragado não deixa o nó arrancar (`CLIXX-E02`). Cada sessão recebe no ambiente o
endereço do nó por dentro da máquina (`POLVO_NODE`: o de `--listen`, com `127.0.0.1` quando escuta em
todas as interfaces), para um agente que corre nela lhe reportar o progresso; o nó guarda o progresso na
pasta `progress` da pasta de configuração.

### CLIXX-B03 — `connect` liga o terminal local a uma shell do nó
O `polvo connect ENDEREÇO` abre uma shell no nó com o tamanho do terminal local (24 por 80 se a
entrada não é um terminal). A entrada vai para a shell, a saída da shell vai para a saída, e o código de
saída do `polvo` é o da shell (255 se a shell morreu por um sinal, como o `ssh`).

### CLIXX-B04 — O terminal local fica em modo cru durante a ligação
Quando a entrada é um terminal, o `connect` põe-no em modo cru durante a ligação (as teclas vão direitas
para a shell remota, sem eco local) e repõe o modo de antes quando a ligação acaba.

### CLIXX-B05 — A mudança de tamanho da janela local chega ao nó
Quando a entrada é um terminal, cada mudança de tamanho da janela local é enviada à shell remota.

### CLIXX-B06 — `mcp` é a ponte do agente local pelo stdin e pelo stdout
O `polvo mcp` serve o protocolo MCP pelo stdin e pelo stdout, com as ferramentas do Polvo, a identidade
e os nós conhecidos da pasta de configuração. O nome do servidor é "polvo", com a versão do executável.
Acaba sem erro quando o stdin acaba.

### CLIXX-B07 — `polvo` sem subcomando (e `polvo ui`) abre a aplicação desktop
O `polvo` sem subcomando, ou o `polvo ui`, arranca o servidor local da UI no endereço da opção
`--listen` (por omissão "127.0.0.1:0", uma porta livre), com a identidade e os nós conhecidos da pasta
de configuração. Escreve na saída o endereço da UI com o segredo da sessão e abre-o no browser do
sistema; com `--no-browser`, só o escreve. O terminal da própria máquina (o nó `local`) corre a shell
do `SHELL`. Serve até o processo acabar. Um `--listen` que não é um endereço local é
recusado (o erro e o código 1): a UI dá shells nos nós, e só a própria máquina a pode abrir.

## Environment Variables
| Variable | Type | Required | Default | Values | Deprecated | Description |
| --- | --- | --- | --- | --- | --- | --- |
| SHELL | string | no | /bin/sh | o caminho de uma shell | no | a shell que o `polvo node` dá às sessões, e a do terminal local da UI |
| POLVO_NODE | string | no | 127.0.0.1:2222 | máquina:porta | no | onde o `polvo mcp` reporta o progresso do agente: o nó da sessão onde ele corre, que a sessão lhe dá |
| POLVO_SESSION | string | no | — | o id de uma sessão do nó | no | a sessão onde o agente corre: o `polvo mcp` reporta com o nome dela quando o agente não dá outro |

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `args` | um subcomando conhecido, as opções dele e os argumentos | um subcomando desconhecido, opções inválidas, um argumento em falta | este módulo: a ajuda no erro e o código 2 (`CLIXX-E01`) |
| `--dir` | uma pasta onde o utilizador pode escrever | uma pasta sem permissão | este módulo: o erro e o código 1 (`CLIXX-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `CLIXX-E01` | o subcomando não existe, uma opção é inválida ou falta um argumento (o `polvo` sem nada é a UI, `CLIXX-B07`) | a forma de uso na saída de erro e o código de saída 2 |
| `CLIXX-E02` | a operação falha (o nó não responde, a chave do nó mudou, a pasta não se escreve) | "polvo: " e o erro na saída de erro, e o código de saída 1 |

## Rule uses

| Rule | Uses |
| --- | --- |
| `CLIXX-B01` | `args`, `--dir` |
| `CLIXX-B02` | `args`, `--dir`, `SHELL` |
| `CLIXX-B03` | `args`, `--dir` |
| `CLIXX-B04` | `args` |
| `CLIXX-B05` | `args` |
| `CLIXX-B06` | `args`, `--dir` |
| `CLIXX-B07` | `args`, `--dir`, `SHELL` |
| `CLIXX-E01` | `args` |
| `CLIXX-E02` | `args`, `--dir` |

## Implementation Notes

- **@TBD: Windows** para o `CLIXX-B05`: o Windows não tem o sinal de mudança de janela; a consola
  avisa-a por eventos, e isso entra com a caixa 1391 do Bavel (a consola e o ConPTY).

## Revisions

> **CLIXX-R0001:** nasceu o `CLIXX-B06` (o `polvo mcp`), pela fase W02 do plano AGLCG: a ponte MCP da W01
> dele precisa de um subcomando que a ligue à identidade e aos nós conhecidos da instalação. As
> regras anteriores não mudam.

> **CLIXX-R0002:** nasceu o `CLIXX-B07` (a UI), pela fase W03 do plano DSKTP: o `polvo` sem subcomando
> deixa de ser um erro de uso e passa a abrir a aplicação desktop, que é o produto. O `CLIXX-E01`
> continua a valer para um subcomando desconhecido, uma opção inválida ou um argumento em falta.

> **CLIXX-R0003:** o `CLIXX-B07` dá à UI a shell do `SHELL` para o terminal da própria máquina (o nó
> `local` do servidor da UI). As regras anteriores não mudam.

> **CLIXX-R0004:** o `CLIXX-B02` liga o nó às sessões e à configuração das permissões (plano ESRMS,
> W01). As regras anteriores não mudam.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
