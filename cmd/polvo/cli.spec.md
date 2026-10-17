<!-- @anchors
  code: CLIXX
  updated_at: 2026-10-17
-->
# Cli — os subcomandos do `polvo`: a identidade, o nó e a ligação

> **Code**: `CLIXX`

## Overview
O executável `polvo` é um só, com subcomandos. O `keygen` cria a identidade da instalação, o `node`
põe a máquina a servir como nó, e o `connect` abre um terminal num nó a partir do terminal local. A
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

### CLIXX-B03 — `connect` liga o terminal local a uma shell do nó
O `polvo connect ENDEREÇO` abre uma shell no nó com o tamanho do terminal local (24 por 80 se a
entrada não é um terminal). A entrada vai para a shell, a saída da shell vai para a saída, e o código de
saída do `polvo` é o da shell (255 se a shell morreu por um sinal, como o `ssh`).

### CLIXX-B04 — O terminal local fica em modo cru durante a ligação
Quando a entrada é um terminal, o `connect` põe-no em modo cru durante a ligação (as teclas vão direitas
para a shell remota, sem eco local) e repõe o modo de antes quando a ligação acaba.

### CLIXX-B05 — A mudança de tamanho da janela local chega ao nó
Quando a entrada é um terminal, cada mudança de tamanho da janela local é enviada à shell remota.

## Environment Variables
| Variable | Type | Required | Default | Values | Deprecated | Description |
| --- | --- | --- | --- | --- | --- | --- |
| SHELL | string | no | /bin/sh | o caminho de uma shell | no | a shell que o `polvo node` dá às sessões |

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `args` | um subcomando conhecido, as opções dele e os argumentos | um subcomando desconhecido, opções inválidas, um argumento em falta | este módulo: a ajuda no erro e o código 2 (`CLIXX-E01`) |
| `--dir` | uma pasta onde o utilizador pode escrever | uma pasta sem permissão | este módulo: o erro e o código 1 (`CLIXX-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `CLIXX-E01` | o subcomando não existe, uma opção é inválida ou falta um argumento | a forma de uso na saída de erro e o código de saída 2 |
| `CLIXX-E02` | a operação falha (o nó não responde, a chave do nó mudou, a pasta não se escreve) | "polvo: " e o erro na saída de erro, e o código de saída 1 |

## Rule uses

| Rule | Uses |
| --- | --- |
| `CLIXX-B01` | `args`, `--dir` |
| `CLIXX-B02` | `args`, `--dir`, `SHELL` |
| `CLIXX-B03` | `args`, `--dir` |
| `CLIXX-B04` | `args` |
| `CLIXX-B05` | `args` |
| `CLIXX-E01` | `args` |
| `CLIXX-E02` | `args`, `--dir` |

## Implementation Notes

- **@TBD: Windows** para o `CLIXX-B05`: o Windows não tem o sinal de mudança de janela; a consola
  avisa-a por eventos, e isso entra com a caixa 1391 do Bavel (a consola e o ConPTY).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
