<!-- @anchors
  code: SSSNS
  updated_at: 2026-10-18
-->
# Sessions — as sessões de terminal de um nó, com os seus espectadores

> **Code**: `SSSNS`

## Overview
Uma sessão é um programa (a shell, normalmente) num pty do nó, que vive independente de quem a vê. O
`Manager` (o `NewManager` faz um vazio) guarda as sessões do nó: cria-as, lista-as (`Info`), liga-lhes
espectadores (`Viewer`), trava-as e termina-as. Um espectador lê (`Read`) o que a sessão mostra e, com a
permissão (`Perm`), escreve nela (`Write`) e muda-lhe o tamanho (`Resize`); no fim, o `Exit` diz como o
programa acabou. É assim que o Polvo de uma máquina espelha os terminais de outra.

Quem se liga a meio recebe primeiro o ecrã atual. Para isso, cada sessão acompanha a saída do programa
num emulador de terminal do lado do nó (`Screen`, hoje o `charmbracelet/x/vt`, em Go puro, para o nó
correr igual no `go` e no Galaxy), que reescreve o ecrã em sequências VT.

## Rules

### SSSNS-B01 — Criar uma sessão
O `Create` corre o programa num pty do tamanho pedido, na pasta pedida, com `TERM=xterm-256color`, e
devolve a sessão com um id que nenhuma outra sessão do nó tem. A sessão entra na lista.

### SSSNS-B02 — Listar as sessões
O `List` dá as sessões vivas pela ordem em que nasceram, cada uma com o id, o programa, o tamanho, o
número de espectadores, se está travada e quando nasceu.

### SSSNS-B03 — Quem se liga recebe o ecrã atual e depois a saída ao vivo
O `Attach` dá um espectador que lê primeiro o ecrã atual da sessão (o texto, as cores e o cursor) e a
seguir a saída do programa à medida que chega, sem perder nem repetir nada entre os dois.

### SSSNS-B04 — A saída chega a todos os espectadores
Com vários espectadores, todos recebem a mesma saída. Sem nenhum, a sessão continua e o ecrã dela
continua a acompanhar o programa.

### SSSNS-B05 — Controlar é escrever e mudar o tamanho
Um espectador que controla escreve na sessão como se teclasse e muda-lhe o tamanho. O tamanho da sessão
é o do último espectador que o mudou.

### SSSNS-B06 — Só ver não escreve
Um espectador com a permissão de ver recebe tudo, mas escrever e mudar o tamanho são recusados
(`SSSNS-E03`).

### SSSNS-B07 — A sessão travada só deixa ver
Uma sessão travada (`Lock`) trata os espectadores que controlam como se só vissem. O dono da máquina
(a permissão `Owner`, a janela local) controla sempre. Destravar devolve o controlo.

### SSSNS-B08 — O fim do programa acaba a sessão
Quando o programa termina, cada espectador lê o resto da saída e depois o fim, com o código de saída
(ou o sinal), e a sessão sai da lista.

### SSSNS-B09 — Terminar uma sessão
O `Close` termina o programa da sessão, e a sessão acaba como no `SSSNS-B08`.

### SSSNS-B11 — O ambiente de uma sessão
O programa de uma sessão recebe no ambiente o id dela (`POLVO_SESSION`) e o ambiente comum do `Manager`
(o `SetEnv`: por exemplo, onde está o nó, `POLVO_NODE`), para um agente que corre nela saber onde está.

### SSSNS-B10 — As respostas do ecrã do nó não chegam ao programa
O emulador do nó responde às perguntas que o programa faz ao terminal; essas respostas descartam-se,
porque quem responde ao programa é o terminal de cada espectador.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `argv` | o programa e os argumentos | vazio, ou um programa que não arranca | este módulo: o erro (`SSSNS-E01`) |
| `id` | o id de uma sessão viva | um id que nenhuma sessão viva tem | este módulo: `ErrNoSession` (`SSSNS-E02`) |
| `Perm` | `View`, `Control` ou `Owner`; o `Pairing`, abaixo do `View`, é de quem só pode emparelhar | — | o tipo |
| `Size` | linhas e colunas maiores que zero | zero | o `pty`: o tamanho por omissão |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `SSSNS-E01` | o programa não arranca (não existe, sem permissão) | o `Create` devolve o erro com o programa, e nenhuma sessão nasce |
| `SSSNS-E02` | o id não é de nenhuma sessão viva | `ErrNoSession` |
| `SSSNS-E03` | um espectador que só vê escreve ou muda o tamanho | `ErrViewOnly`, e a sessão fica como estava |
| `SSSNS-E04` | um espectador fica para trás mais do que `VIEWER_BACKLOG` | o espectador é desligado com `ErrTooSlow`, e os outros e o programa continuam |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `VIEWER_BACKLOG` | 4 MiB | o que um espectador pode ter por ler antes de ser desligado, por omissão (o `SetBacklog` muda-o): uma rede lenta não pode travar a sessão nem os outros |

## Rule uses

| Rule | Uses |
| --- | --- |
| `SSSNS-B01` | `argv`, `Size` |
| `SSSNS-B02` | `id` |
| `SSSNS-B03` | `id`, `Perm` |
| `SSSNS-B04` | `id` |
| `SSSNS-B05` | `Perm`, `Size` |
| `SSSNS-B06` | `Perm` |
| `SSSNS-B07` | `id`, `Perm` |
| `SSSNS-B08` | `id` |
| `SSSNS-B09` | `id` |
| `SSSNS-B10` | `argv` |
| `SSSNS-B11` | `argv`, `id` |
| `SSSNS-E01` | `argv` |
| `SSSNS-E02` | `id` |
| `SSSNS-E03` | `Perm` |
| `SSSNS-E04` | `VIEWER_BACKLOG` |

## Revisions

> **SSSNS-R0001:** entra o `SSSNS-B11` (o ambiente de uma sessão), para o agente que corre numa sessão
> reportar o progresso dele com o nome da sessão (plano AGPRG, W02).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
