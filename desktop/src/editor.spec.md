<!-- @anchors
  code: EDTRS
  updated_at: 2026-10-18
-->
# Editor — o editor de código numa aba da doca

> **Code**: `EDTRS`

## Overview
O editor (`EditorPanel`) é uma aba da doca com o editor de código do gpui-component: os números das
linhas, o realce da linguagem (pelo tree-sitter) e a procura. Abre vazio (`EditorPanel::empty`) ou com
um ficheiro de texto (`EditorPanel::open`). O título da aba é o nome do ficheiro, com uma marca enquanto
há alterações por gravar. A organização da doca guarda o caminho do ficheiro de cada editor
(`EditorPanel::path`), para o editor voltar com ele.

## Rules

### EDTRS-B01 — O editor vazio
O editor vazio não tem texto nem ficheiro, e o título é "Sem título".

### EDTRS-B02 — O editor com um ficheiro
O editor de um ficheiro mostra o texto dele, realçado na linguagem que a extensão diz (`language_for`;
texto simples quando a extensão não é de nenhuma), e o título é o nome do ficheiro.

### EDTRS-B03 — A marca de alterado
Escrever no editor põe a marca "●" antes do título; gravar tira-a.

### EDTRS-B04 — Gravar
O atalho de gravar do mapa (`Keymap`, por omissão Ctrl+S) grava o texto no ficheiro do editor. O editor
vazio pergunta primeiro onde gravar, pelo diálogo do sistema; o ficheiro escolhido passa a ser o dele
(o título e a linguagem seguem-no). Cancelar o diálogo não grava nada.

### EDTRS-B05 — O ficheiro de outra máquina
O editor pode abrir um ficheiro de outra máquina (`EditorPanel::open_on` com o `FileSource` do nó dela):
diz "a abrir…" até o nó dar o texto, e grava pelo nó, sem parar a janela; o que se escreve enquanto o
nó grava continua com a marca. O título diz a máquina, e a organização da doca guarda-a com o ficheiro.
O limite `MAX_OPEN` vale também aqui, contado depois de o nó o ler.

### EDTRS-B06 — Fechar com alterações por gravar
Fechar um editor com alterações por gravar (o botão da aba, a ação de fechar, ou o atalho da janela) não o
fecha logo: pergunta pelo diálogo do sistema se as grava. "Gravar" grava (como no `EDTRS-B04`) e fecha
depois de gravar; se gravar falha (`EDTRS-E02`), fica aberto. "Descartar" fecha sem gravar. "Cancelar"
deixa-o como estava. Sem alterações, fecha logo. O editor diz à janela que fechou pelo `EditorEvent`.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `path` | um ficheiro de texto em UTF-8 até `MAX_OPEN` | um que não se lê, que não é UTF-8, ou maior | este módulo: a mensagem (`EDTRS-E01`) |
| `keystroke` | as teclas premidas no editor | — | o editor do gpui-component |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `EDTRS-E01` | o ficheiro não se lê, não é texto UTF-8, ou passa de `MAX_OPEN` | a aba mostra o motivo em vez do editor, e não grava nada |
| `EDTRS-E02` | o ficheiro não se grava | a mensagem com o motivo por baixo do editor, e a marca de alterado fica |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `MAX_OPEN` | 16 MiB | o editor guarda o texto todo em memória; um ficheiro maior é um registo ou um binário, não código |

## Rule uses

| Rule | Uses |
| --- | --- |
| `EDTRS-B01` | `keystroke` |
| `EDTRS-B02` | `path` |
| `EDTRS-B03` | `keystroke` |
| `EDTRS-B04` | `path`, `keystroke` |
| `EDTRS-E01` | `path`, `MAX_OPEN` |
| `EDTRS-E02` | `path` |
| `EDTRS-B05` | `path`, `keystroke`, `MAX_OPEN` |
| `EDTRS-B06` | `path`, `keystroke` |

## Revisions

> **EDTRS-R0002:** entra o `EDTRS-B06` (fechar com alterações por gravar): o dono decidiu a 2026-10-18
> que se pergunta (era a `EDTRS-D01`). A doca do gpui-component não perguntava ao painel antes de o fechar:
> a cópia em `third_party/` ganha o `request_close`, até o upstream o ter.

> **EDTRS-R0001:** entra o `EDTRS-B05` (o ficheiro de outra máquina), do plano ESRMS, W05.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
