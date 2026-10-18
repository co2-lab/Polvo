<!-- @anchors
  code: PRGRS
  updated_at: 2026-10-18
-->
# Progress — o progresso dos agentes de uma máquina

> **Code**: `PRGRS`

## Overview
O quadro (`Board`) guarda o progresso de cada agente que trabalha na máquina (um `claude` num terminal,
por exemplo): o trabalho dele (`Progress`), com o objetivo, os passos (`Step`, cada um num estado,
`State`) e o registo do que aconteceu (`Entry`), por ordem de chegada. O agente reporta pelo `Start`,
`Step`, `Note` e `Finish`; quem acompanha lê pelo `List` e pelo `Get`, e espera por mudanças pelo `Wait`.
O trabalho guarda também a conversa do agente (`Conversation`), quando
ele a deu ao começar, e as perguntas que o Polvo lhe fez (`Question`, cada uma por responder, respondida ou falhada: o
`QuestionState`), com as respostas. Cada mudança sobe
a versão do quadro (`Version`) e grava o trabalho do agente num ficheiro da pasta do
quadro (`NewBoard`), que o quadro volta a ler quando a máquina arranca outra vez.

## Rules

### PRGRS-B01 — Começar um trabalho
O `Start` dá ao agente um trabalho novo, com o objetivo, os passos (todos por fazer) e a conversa dele,
quando a tem, e regista o começo; o trabalho anterior do agente, se o havia, dá-lhe lugar.

### PRGRS-B02 — O estado de um passo
O `Step` muda o estado de um passo (por fazer, em curso, feito, bloqueado), com uma nota opcional, e
regista a mudança.

### PRGRS-B03 — Uma nota
O `Note` regista uma nota do agente, sem mudar os passos.

### PRGRS-B04 — Terminar
O `Finish` marca o trabalho como terminado, com o resumo, e regista o fim.

### PRGRS-B05 — Ler o progresso
O `List` dá os trabalhos de todos os agentes, o mudado mais recentemente primeiro, com a versão do
quadro; o `Get` dá o de um agente.

### PRGRS-B06 — Esperar por uma mudança
O `Wait` com a versão que quem acompanha já tem volta logo se o quadro já mudou desde ela, e senão
espera pela próxima mudança, ou até o prazo acabar, e dá a lista e a versão.

### PRGRS-B07 — O progresso fica gravado
Cada mudança grava o trabalho do agente; um quadro novo na mesma pasta tem os mesmos trabalhos.

### PRGRS-B08 — O registo tem um limite
O registo de um trabalho guarda as últimas `LOG_LIMIT` entradas; as mais velhas saem.

### PRGRS-B09 — Perguntar ao agente
O `Ask` regista uma pergunta ao agente, com um id que nenhuma outra pergunta do trabalho tem, por
responder.

### PRGRS-B10 — A resposta
O `Answer` dá a resposta a uma pergunta por responder, que passa a respondida; o `Fail` marca-a como
falhada, com o motivo (quando não houve como perguntar).

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `agent` | um nome de até `AGENT_MAX` caracteres, sem `/`, `\` nem o zero | vazio, maior, ou com esses caracteres | este módulo: o erro (`PRGRS-E01`) |
| `step` | o id de um passo do trabalho | outro | este módulo: `ErrNoStep` (`PRGRS-E03`) |
| `State` | `todo`, `doing`, `done`, `blocked` | outro | este módulo: o erro (`PRGRS-E04`) |
| `question` | o id de uma pergunta por responder do trabalho | outro, ou uma já respondida ou falhada | este módulo: `ErrNoQuestion` (`PRGRS-E07`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `PRGRS-E01` | o nome do agente não é aceite | o erro diz porquê, e nada muda |
| `PRGRS-E02` | o agente reporta sem ter começado um trabalho | `ErrNoJob`, e nada muda |
| `PRGRS-E03` | o passo não é do trabalho | `ErrNoStep`, e nada muda |
| `PRGRS-E04` | o estado não é um dos quatro | o erro diz qual, e nada muda |
| `PRGRS-E05` | o trabalho não se grava | o erro volta a quem reportou, e o quadro em memória fica com a mudança |
| `PRGRS-E06` | um ficheiro da pasta não é um trabalho | o quadro arranca sem ele, com o aviso na saída de erro |
| `PRGRS-E07` | a pergunta não é do trabalho, ou já não está por responder | `ErrNoQuestion`, e nada muda |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `AGENT_MAX` | 128 | um nome de agente é um nome, e vira o nome de um ficheiro |
| `LOG_LIMIT` | 500 | o painel mostra os últimos estados; um registo sem fim cresce o ficheiro a cada nota |

## Rule uses

| Rule | Uses |
| --- | --- |
| `PRGRS-B01` | `agent` |
| `PRGRS-B02` | `agent`, `step`, `State` |
| `PRGRS-B03` | `agent` |
| `PRGRS-B04` | `agent` |
| `PRGRS-B05` | `agent` |
| `PRGRS-B06` | `agent` |
| `PRGRS-B07` | `agent` |
| `PRGRS-B08` | `LOG_LIMIT` |
| `PRGRS-E01` | `agent`, `AGENT_MAX` |
| `PRGRS-E02` | `agent` |
| `PRGRS-E03` | `step` |
| `PRGRS-E04` | `State` |
| `PRGRS-E05` | `agent` |
| `PRGRS-E06` | `agent` |
| `PRGRS-B09` | `agent` |
| `PRGRS-B10` | `agent`, `question` |
| `PRGRS-E07` | `question` |

## Revisions

> **PRGRS-R0001:** entram a conversa do agente no `Start` e as perguntas (`PRGRS-B09`, `PRGRS-B10`,
> `PRGRS-E07`), do plano AGPRG, W03.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
