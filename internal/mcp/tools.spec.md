<!-- @anchors
  code: MCPTL
  updated_at: 2026-10-17
-->
# Tools — as ferramentas do Polvo para o agente local: comandos, ficheiros e pastas dos nós

> **Code**: `MCPTL`

## Overview
As ferramentas que o `polvo mcp` oferece ao agente local, sobre o cliente dos nós. Cada ferramenta
recebe o endereço do nó e os argumentos dela, e devolve texto para o agente ler. As ligações aos nós
ficam num conjunto (`Pool`, criado pelo `NewPool` com a configuração do cliente), reutilizadas entre
chamadas e fechadas no fim (`Close`). O conjunto dá as ferramentas pelo `Tools`.

## Rules

### MCPTL-B01 — `polvo_run` corre um comando e devolve o código, a saída e o erro
A ferramenta `polvo_run` corre o comando dado pela shell do nó (opcionalmente numa pasta e com uma
entrada) até ao fim, e devolve em texto o código de saída, a saída e o erro, cada um com o seu
cabeçalho. Um código diferente de 0 faz parte do resultado: não é uma falha da ferramenta.

### MCPTL-B02 — `polvo_read_file` devolve o conteúdo de um ficheiro de texto
A ferramenta `polvo_read_file` devolve o conteúdo de um ficheiro do nó. Um ficheiro que não é texto
(não é UTF-8 válido) é uma falha da ferramenta, com o tamanho, porque o agente não o pode ler como texto.

### MCPTL-B03 — `polvo_write_file` escreve um ficheiro
A ferramenta `polvo_write_file` escreve o conteúdo dado num ficheiro do nó, criando-o ou substituindo-o,
e devolve quantos bytes escreveu e onde.

### MCPTL-B04 — `polvo_list` devolve as entradas de uma pasta
A ferramenta `polvo_list` devolve as entradas de uma pasta do nó por ordem de nome, uma por linha: as
pastas com uma barra no fim, e os ficheiros com o tamanho.

### MCPTL-B05 — A ligação a um nó é reutilizada
Duas chamadas ao mesmo nó usam a mesma ligação. O conjunto dá a ligação a quem a pede para uma operação (`With`); a UI usa-o assim.

### MCPTL-B06 — Uma ligação caída é refeita
Quando a ligação a um nó cai, a chamada seguinte a esse nó liga-se outra vez.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `node` | o endereço do nó (máquina e porta) | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `command` | um comando para a shell do nó | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `path` | um caminho no nó | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `content` | o texto a escrever | nenhum (o vazio cria um ficheiro vazio) | quem chama |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `MCPTL-E01` | falta um argumento obrigatório, ou os argumentos não têm a forma | a falha da ferramenta, com o argumento em falta |
| `MCPTL-E02` | o nó não responde ou recusa a identidade | a falha da ferramenta, com o endereço |
| `MCPTL-E03` | o nó recusa a operação (o ficheiro não existe, sem permissão, o programa não arranca) | a falha da ferramenta, com a mensagem do nó |

## Rule uses

| Rule | Uses |
| --- | --- |
| `MCPTL-B01` | `node`, `command` |
| `MCPTL-B02` | `node`, `path` |
| `MCPTL-B03` | `node`, `path`, `content` |
| `MCPTL-B04` | `node`, `path` |
| `MCPTL-B05` | `node` |
| `MCPTL-B06` | `node` |
| `MCPTL-E01` | `node`, `command`, `path` |
| `MCPTL-E02` | `node` |
| `MCPTL-E03` | `node`, `path` |

## Implementation Notes

- **@TBD: Windows** para a shell do `polvo_run`: num nó Windows a shell é outra (o `cmd`/PowerShell), e
  isso entra com a caixa 1391 do Bavel. Num nó Unix é o `sh -c`.

## Revisions

> **MCPTL-R0001:** o `With` passa a ser público: o servidor da UI (`WEBSR`, plano DSKTP) reutiliza o
> conjunto das ligações em vez de o repetir. As regras não mudam.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
