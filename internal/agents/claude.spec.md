<!-- @anchors
  code: CLDAG
  updated_at: 2026-10-18
-->
# Claude — perguntar ao `claude` de uma conversa, pelo CLI dele

> **Code**: `CLDAG`

## Overview
Quando o agente de uma sessão é o `claude` (o Claude Code), o Polvo pergunta-lhe pelo CLI dele: o `Ask`
corre `claude -p --resume <conversa> --fork-session` na pasta da conversa, com a pergunta. A resposta vem
com o contexto todo da conversa, numa cópia, sem interromper o agente nem mexer na conversa dele. A cópia
usa o modelo da conversa e as mesmas ferramentas, para ler a conversa da cache de prompts e não a preço
cheio, e o modo de planeamento (`plan`), em que nada se escreve.

O `Find` encontra a conversa de um `claude` (`progress.Conversation`) a partir da pasta onde ele corre:
as conversas do Claude Code ficam em `~/.claude/projects/<pasta>/<id>.jsonl`, e o modelo de cada uma está
nas mensagens do assistente.

## Rules

### CLDAG-B01 — Encontrar a conversa
O `Find` dá a conversa com o id que o `claude` dá aos programas que corre (o `CLAUDE_CODE_SESSION_ID`),
quando ela está na pasta do projeto; sem esse id, ou sem essa conversa, dá a mudada mais recentemente da
pasta do projeto. A conversa vem com o modelo da última mensagem do assistente, lido do fim do ficheiro
(no máximo `TAIL_MAX`).

### CLDAG-B02 — Perguntar
O `Ask` corre o `claude` (`CLAUDE_BIN`) na pasta da conversa com `-p`, `--resume` e o id, `--fork-session`,
`--output-format json`, `--permission-mode plan`, o `--model` da conversa (quando o sabe) e a pergunta, e
dá o texto da resposta.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `dir` | a pasta onde o `claude` corre | uma pasta sem conversas do `claude` | este módulo: o erro (`CLDAG-E01`) |
| `Conversation` | a conversa de um `claude` | outra | quem pergunta |
| `question` | o texto da pergunta | — | quem pergunta |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `CLDAG-E01` | a pasta do projeto não tem conversas | o erro com a pasta |
| `CLDAG-E02` | o `claude` não corre, falha, passa de `ASK_TIMEOUT`, ou responde com erro | o erro com o motivo (o do `claude`, quando o dá) |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `CLAUDE_BIN` | `claude` | o CLI do Claude Code no `PATH` |
| `ASK_TIMEOUT` | 5 min | uma pergunta lê a conversa toda; mais do que isto é um `claude` parado |
| `TAIL_MAX` | 1 MiB | uma conversa pode ter gigabytes; o modelo está nas últimas mensagens |

## Rule uses

| Rule | Uses |
| --- | --- |
| `CLDAG-B01` | `dir`, `TAIL_MAX` |
| `CLDAG-B02` | `Conversation`, `question`, `CLAUDE_BIN` |
| `CLDAG-E01` | `dir` |
| `CLDAG-E02` | `CLAUDE_BIN`, `ASK_TIMEOUT` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
