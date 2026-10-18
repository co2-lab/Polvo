<!-- @anchors
  code: MCPTL
  updated_at: 2026-10-18
-->
# Tools — as ferramentas do Polvo para o agente local: comandos, ficheiros e pastas dos nós

> **Code**: `MCPTL`

## Overview
As ferramentas que o `polvo mcp` oferece ao agente local, sobre o cliente dos nós. Cada ferramenta
recebe o endereço do nó e os argumentos dela, e devolve texto para o agente ler. As ligações aos nós
ficam num conjunto (`Pool`, criado pelo `NewPool` com a configuração do cliente), reutilizadas entre
chamadas e fechadas no fim (`Close`). O conjunto dá as ferramentas pelo `Tools`.

As ferramentas do progresso (`polvo_progress_*`) reportam o trabalho do agente ao nó da máquina onde ele
corre. O conjunto sabe onde isso é (`SetHere`, com o `Here`): o nó e a sessão onde o agente corre, que o
`polvo mcp` lê do ambiente (`POLVO_NODE` e `POLVO_SESSION`, que a sessão do Polvo lhe dá), e se o agente
é o Claude Code, com a conversa dele (o `CLAUDE_CODE_SESSION_ID`, a pasta e a pasta pessoal), para o Polvo
lhe poder perguntar.

As ferramentas das sessões (`polvo_sessions`, `polvo_screen`, `polvo_type` e `polvo_wait`) deixam o agente
trabalhar nas sessões de terminal de qualquer nó, com a permissão que esse nó dá à instalação.

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

### MCPTL-B07 — `polvo_progress_start` começa um trabalho
A ferramenta `polvo_progress_start` começa o trabalho do agente no nó, com o objetivo e os passos, e
devolve os passos com o id de cada um, para o agente os nomear depois.

### MCPTL-B08 — `polvo_progress_step` muda o estado de um passo
A ferramenta `polvo_progress_step` muda o estado de um passo (pelo id ou pelo título: `todo`, `doing`,
`done` ou `blocked`), com uma nota opcional.

### MCPTL-B09 — `polvo_progress_note` deixa uma nota
A ferramenta `polvo_progress_note` deixa uma nota no registo do trabalho.

### MCPTL-B10 — `polvo_progress_finish` termina o trabalho
A ferramenta `polvo_progress_finish` termina o trabalho, com o resumo.

### MCPTL-B11 — O agente e o nó por omissão
Sem o argumento `agent`, o agente é o da sessão onde corre (`session-` e o id da sessão); sem o `node`, o
nó é o da máquina onde corre (o do `Here`, ou `DEFAULT_NODE`).

### MCPTL-B12 — `polvo_progress_answer` responde a uma pergunta do Polvo
A ferramenta `polvo_progress_answer` dá a resposta do agente à pergunta do Polvo com o id dado.

### MCPTL-B13 — A conversa do Claude Code vai com o trabalho
Quando o agente é o Claude Code, o `polvo_progress_start` dá ao nó a conversa dele (`agents.Find`), para o
Polvo lhe perguntar pelo CLI; sem conversa encontrada, o trabalho começa sem ela.

### MCPTL-B14 — `polvo_sessions` lista as sessões de um nó
A ferramenta `polvo_sessions` devolve as sessões vivas do nó, uma por linha: o id, o programa, o tamanho
(colunas por linhas), e se o agente pode escrever nela ou só ver.

### MCPTL-B15 — `polvo_screen` lê o ecrã de uma sessão
A ferramenta `polvo_screen` devolve o ecrã atual da sessão, em texto simples.

### MCPTL-B16 — `polvo_type` escreve numa sessão
A ferramenta `polvo_type` escreve na sessão o texto tal como vem e depois as teclas pelo nome (`enter`,
`tab`, `escape`, `backspace`, `delete`, as setas, `home`, `end`, `pageup`, `pagedown`, e `ctrl-a` a
`ctrl-z`), e diz quantos bytes escreveu. O nó recusa-o a quem só vê, e numa sessão travada a quem não é
o dono (`MCPTL-E03`).

### MCPTL-B17 — `polvo_wait` espera por um texto no ecrã
A ferramenta `polvo_wait` lê o ecrã da sessão a cada `WAIT_POLL` até ele mostrar o texto, e devolve-o;
espera até ao `timeout` pedido (em segundos), `WAIT_DEFAULT` sem ele, e nunca mais que `WAIT_MAX`.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `node` | o endereço do nó (máquina e porta) | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `command` | um comando para a shell do nó | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `path` | um caminho no nó | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `content` | o texto a escrever | nenhum (o vazio cria um ficheiro vazio) | quem chama |
| `session` | o id de uma sessão do nó | vazio | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `keys` | os nomes das teclas do `MCPTL-B16` | outro nome | esta unidade: a falha da ferramenta (`MCPTL-E01`) |
| `timeout` | segundos, maior que zero | zero ou negativo (vale o `WAIT_DEFAULT`) | esta unidade |
| `agent` | o nome do agente, ou nenhum dentro de uma sessão do Polvo | nenhum fora de uma sessão | esta unidade: a falha da ferramenta (`MCPTL-E04`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `MCPTL-E01` | falta um argumento obrigatório, ou os argumentos não têm a forma | a falha da ferramenta, com o argumento em falta |
| `MCPTL-E02` | o nó não responde ou recusa a identidade | a falha da ferramenta, com o endereço |
| `MCPTL-E03` | o nó recusa a operação (o ficheiro não existe, sem permissão, o programa não arranca) | a falha da ferramenta, com a mensagem do nó |
| `MCPTL-E04` | uma ferramenta do progresso sem `agent`, fora de uma sessão do Polvo | a falha da ferramenta, que pede o nome do agente |
| `MCPTL-E05` | o ecrã não mostra o texto do `polvo_wait` até ao fim da espera | a falha da ferramenta, com o texto e o ecrã que se vê |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULT_NODE` | `127.0.0.1:2222` | o nó desta máquina, onde o `polvo node` e a janela escutam por omissão |
| `WAIT_DEFAULT` | 30 s | o que um comando costuma levar a responder, sem prender o agente muito tempo |
| `WAIT_MAX` | 5 min | uma compilação ou uns testes longos, sem deixar o agente preso para sempre |
| `WAIT_POLL` | 250 ms | o ecrã muda à velocidade de quem lê; mais depressa só carrega o nó |

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
| `MCPTL-B07` | `node`, `agent` |
| `MCPTL-B08` | `node`, `agent` |
| `MCPTL-B09` | `node`, `agent` |
| `MCPTL-B10` | `node`, `agent` |
| `MCPTL-B11` | `node`, `agent`, `DEFAULT_NODE` |
| `MCPTL-E04` | `agent` |
| `MCPTL-B12` | `node`, `agent` |
| `MCPTL-B13` | `node`, `agent` |
| `MCPTL-B14` | `node` |
| `MCPTL-B15` | `node`, `session` |
| `MCPTL-B16` | `node`, `session`, `keys` |
| `MCPTL-B17` | `node`, `session`, `timeout`, `WAIT_DEFAULT`, `WAIT_MAX`, `WAIT_POLL` |
| `MCPTL-E05` | `session`, `timeout` |

## Implementation Notes

- **@TBD: Windows** para a shell do `polvo_run`: num nó Windows a shell é outra (o `cmd`/PowerShell), e
  isso entra com a caixa 1391 do Bavel. Num nó Unix é o `sh -c`.

## Revisions

> **MCPTL-R0001:** o `With` passa a ser público: o servidor da UI (`WEBSR`, plano DSKTP) reutiliza o
> conjunto das ligações em vez de o repetir. As regras não mudam.

> **MCPTL-R0002:** entram as ferramentas do progresso (`MCPTL-B07` a `MCPTL-B11`, `MCPTL-E04`), do plano
> AGPRG, W02, e as da W03: responder a uma pergunta (`MCPTL-B12`) e a conversa do Claude Code
> (`MCPTL-B13`).

> **MCPTL-R0003:** entram as ferramentas das sessões (`MCPTL-B14` a `MCPTL-B17`, `MCPTL-E05`), do plano
> ESRMS, W06; o nó por omissão do `MCPTL-B11` vale também para elas.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
