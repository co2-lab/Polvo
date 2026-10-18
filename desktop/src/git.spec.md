<!-- @anchors
  code: GITRS
  updated_at: 2026-10-18
-->
# Git — a informação de git de uma pasta, numa aba da doca

> **Code**: `GITRS`

## Overview
A aba do git (`GitPanel`) mostra o estado do repositório de uma pasta: o ramo, os ficheiros com
alterações e os últimos commits. Lê-o do próprio `git` do utilizador (`GIT`), com o `git status
--porcelain=v2` e o `git log`, para dar o que o git dá na linha de comandos (a configuração dele, os
ficheiros ignorados, os submódulos). A leitura (`read_repo`) dá o estado do repositório (`Repo`): onde está a HEAD (`Head`), os ficheiros
alterados (`Change`, cada um no seu grupo, `Group`) e os commits (`Commit`); ou porque não o leu
(`GitError`). Corre fora da linha da janela. Carregar num ficheiro
alterado diz à janela qual (`GitEvent`), e a janela abre-o no editor. A organização da doca guarda a
pasta (`GitPanel::root`), para a aba voltar com ela.

## Rules

### GITRS-B01 — O ramo
A aba mostra o ramo atual e, quando tem um ramo a montante, quantos commits está à frente e atrás
dele. Sem ramo (a HEAD destacada), mostra o commit onde está.

### GITRS-B02 — Os ficheiros com alterações
A aba mostra os ficheiros com alterações em grupos, cada um pelo caminho: em conflito, preparados
para o commit, alterados e não seguidos. Um ficheiro preparado e depois alterado está nos dois grupos.
Sem alterações, diz que não há.

### GITRS-B03 — Os últimos commits
A aba mostra os últimos `LOG_LIMIT` commits do ramo, do mais novo para o mais velho, cada um com o
hash curto, o assunto, o autor e a data.

### GITRS-B04 — Atualizar
O botão de atualizar lê o repositório outra vez.

### GITRS-B05 — Abrir um ficheiro alterado
Carregar num ficheiro alterado que existe abre-o (`Open`, com o caminho).

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |

none
| `root` | uma pasta dentro de um repositório git | uma pasta fora de um repositório | este módulo: a mensagem (`GITRS-E01`) |
| `GIT` | o programa `git` no `PATH` | sem `git`, ou um que falha | este módulo: a mensagem (`GITRS-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `GITRS-E01` | a pasta não está num repositório git | a aba diz que a pasta não é de um repositório, e o botão de atualizar continua |
| `GITRS-E02` | o `git` não existe ou falha | a aba mostra o motivo (o erro do `git`), e o botão de atualizar continua |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `GIT` | `git` | o git do utilizador, o mesmo que ele usa no terminal |
| `LOG_LIMIT` | 30 | o que cabe numa aba sem rolar muito; a história toda é do `git log` no terminal |

## Rule uses

| Rule | Uses |
| --- | --- |
| `GITRS-B01` | `root`, `GIT` |
| `GITRS-B02` | `root`, `GIT` |
| `GITRS-B03` | `root`, `GIT`, `LOG_LIMIT` |
| `GITRS-B04` | `root` |
| `GITRS-B05` | `root` |
| `GITRS-E01` | `root` |
| `GITRS-E02` | `GIT` |

## Revisions

> **GITRS-R0001:** o dono aprovou a 2026-10-18 que a aba leia do `git` em linha de comandos (era a
> `GITRS-D01`): a semântica exata do git do utilizador (a configuração, os hooks, as credenciais).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
