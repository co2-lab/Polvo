<!-- @anchors
  code: AGNTS
  updated_at: 2026-10-18
-->
# Agents — o progresso dos agentes de todas as máquinas, e as perguntas a eles

> **Code**: `AGNTS`

## Overview
O `AgentsFeed` mantém, pela instalação (`Station`), o progresso dos agentes de todas as máquinas: espera
pela próxima mudança de qualquer uma (a espera longa do nó) e avisa quem o ouve (`FeedEvent`). A janela
tem um sempre que a instalação abriu: dele vem a aba dos agentes e o indicador na aba da sessão onde cada
agente corre (`AgentMark`).

A aba dos agentes (`AgentsPanel`) mostra o que os agentes já reportaram, sem gastar nada: o objetivo de
cada um, os passos com o estado, os últimos estados do registo (`LAST_ENTRIES`), e as perguntas feitas, com
as respostas. Só pergunta quando o utilizador carrega em perguntar: uma das perguntas comuns
(`COMMON_QUESTIONS`) ou uma escrita por ele.

## Rules

### AGNTS-B01 — Os agentes de todas as máquinas
A aba mostra os agentes de cada máquina, cada um com o objetivo, se terminou, quantos passos estão feitos,
e os passos com o estado.

### AGNTS-B02 — Os últimos estados
Cada agente mostra os últimos `LAST_ENTRIES` estados do registo, o mais novo primeiro: o começo, a
mudança de um passo (com a nota), as notas, e o fim.

### AGNTS-B03 — Perguntar
Cada agente tem as perguntas comuns e uma caixa para escrever outra; carregar numa delas, ou em perguntar
com o texto escrito, pergunta ao agente pela máquina dele.

### AGNTS-B04 — As perguntas e as respostas
Cada agente mostra as perguntas feitas, com a resposta, "à espera", ou o motivo de ter falhado.

### AGNTS-B05 — O feed acompanha as mudanças
O feed lê o progresso de todas as máquinas e, enquanto vive, espera pela próxima mudança de qualquer uma
(até `WAIT_MS`) e lê outra vez; cada leitura avisa quem o ouve.

### AGNTS-B06 — A marca de um agente numa sessão
Do feed sai a marca de cada agente que corre numa sessão (`session-` e o id): a trabalhar, à espera de uma
resposta a uma pergunta, bloqueado num passo, ou terminado.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Station` | a instalação aberta | — | a janela: sem ela não há feed |
| `question` | o texto da pergunta | vazio | este módulo: perguntar não faz nada |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `AGNTS-E01` | uma máquina não responde | a máquina mostra porquê, e as outras continuam |
| `AGNTS-E02` | perguntar falha (a máquina, o agente) | a mensagem com o motivo, junto do agente |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `LAST_ENTRIES` | 8 | os últimos estados cabem sem rolar; o registo inteiro fica no nó |
| `WAIT_MS` | 20000 | a espera longa volta antes dos 30 s do nó, e o feed pede outra |
| `COMMON_QUESTIONS` | "Em que ponto estás?", "O que falta fazer?", "Estás bloqueado em alguma coisa?" | as perguntas que se fazem a quem está a trabalhar |

## Rule uses

| Rule | Uses |
| --- | --- |
| `AGNTS-B01` | `Station` |
| `AGNTS-B02` | `LAST_ENTRIES` |
| `AGNTS-B03` | `question`, `COMMON_QUESTIONS` |
| `AGNTS-B04` | `Station` |
| `AGNTS-B05` | `Station`, `WAIT_MS` |
| `AGNTS-B06` | `Station` |
| `AGNTS-E01` | `Station` |
| `AGNTS-E02` | `question` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
