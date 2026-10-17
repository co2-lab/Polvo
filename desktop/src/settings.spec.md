<!-- @anchors
  code: STTNG
  updated_at: 2026-10-17
-->
# Settings — a configuração do utilizador, guardada num ficheiro

> **Code**: `STTNG`

## Overview
O que o utilizador muda na configuração da janela guarda-se num ficheiro e volta na próxima vez. Hoje
são os atalhos de teclado (o que difere dos de omissão, pelo id de cada ação). O `SettingsFile` é o
ficheiro: o `default` dá o da instalação, o `read` lê a configuração (`Settings`) e o `write` grava-a.

```json
{ "keys": { "new-tab": "ctrl-alt-n" } }
```

## Rules

### STTNG-B01 — O ficheiro fica na pasta de configuração do sistema
O ficheiro por omissão é o `settings.json` na pasta `polvo` da pasta de configuração do utilizador, a de
cada sistema.

### STTNG-B02 — Ler devolve o que foi gravado
O `read` devolve a configuração que o `write` gravou, igual. Sem ficheiro, a configuração não muda nada.

### STTNG-B03 — Gravar nunca deixa o ficheiro a meio
O `write` cria a pasta quando falta, escreve a configuração num ficheiro ao lado e só depois lhe dá o
nome, de uma vez.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `settings.json` | o objeto com `keys` (texto por texto), ou nenhum ficheiro | JSON estragado, ou `keys` de outra forma | este módulo: a configuração que não muda nada (`STTNG-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `STTNG-E01` | o ficheiro não é uma configuração | a configuração que não muda nada, e o aviso com o ficheiro na saída de erro: a janela abre com os valores de omissão |
| `STTNG-E02` | a pasta ou o ficheiro não se escrevem | o erro volta a quem gravou, e o ficheiro anterior fica como estava |

## Rule uses

| Rule | Uses |
| --- | --- |
| `STTNG-B01` | `settings.json` |
| `STTNG-B02` | `settings.json` |
| `STTNG-B03` | `settings.json` |
| `STTNG-E01` | `settings.json` |
| `STTNG-E02` | `settings.json` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
