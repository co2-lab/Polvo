<!-- @anchors
  code: FLTRE
  updated_at: 2026-10-18
-->
# Files — a árvore de ficheiros de uma pasta, numa aba da doca

> **Code**: `FLTRE`

## Overview
A árvore (`FileTree`) é uma aba da doca com o conteúdo de uma pasta: as pastas, que se expandem e
recolhem, e os ficheiros, que se abrem. A árvore lê cada pasta só quando ela se expande, e de cada vez
que se expande, para mostrar o que lá está agora. Abrir um ficheiro diz à janela qual (`FileTreeEvent`),
e a janela abre-o no editor. O título da aba é o nome da pasta, e a organização da doca guarda a pasta
(`FileTree::root`), para a árvore voltar com ela.

## Rules

### FLTRE-B01 — O conteúdo da pasta
A árvore mostra o que está na pasta: primeiro as pastas e depois os ficheiros, cada grupo pelo nome
(sem distinguir maiúsculas), com as pastas recolhidas. O primeiro está escolhido.

### FLTRE-B02 — Expandir e recolher uma pasta
Carregar numa pasta recolhida, ou premir a seta para a direita ou o Enter nela, expande-a: o conteúdo
dela aparece por baixo, um nível para dentro, pela ordem do `FLTRE-B01`. Carregar numa pasta expandida,
ou premir a seta para a esquerda nela, recolhe-a.

### FLTRE-B03 — Abrir um ficheiro
Carregar num ficheiro, ou premir o Enter nele, abre-o (`Open`, com o caminho).

### FLTRE-B04 — Andar com o teclado
As setas para cima e para baixo mudam o escolhido, sem passar do primeiro nem do último. A seta para a
esquerda num ficheiro ou numa pasta recolhida escolhe a pasta onde está.

### FLTRE-B05 — A árvore de outra máquina
A árvore pode ser de uma pasta de qualquer máquina (`FileSource`: esta, ou o nó de uma máquina, pelo
`FileTree::on`). A de outra máquina lê cada pasta pelo nó dela, fora da linha da janela, com uma linha "a
ler…" no lugar do conteúdo enquanto espera; o `~` é a pasta pessoal dela. O título diz a máquina, e a
organização da doca guarda-a com a pasta.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `root` | uma pasta | uma pasta que não se lê | este módulo: o motivo no lugar do conteúdo (`FLTRE-E01`) |
| `keystroke` | as teclas premidas com a árvore com o foco | — | este módulo: as outras não fazem nada |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `FLTRE-E01` | uma pasta não se lê (sem permissão, apagada entretanto) | no lugar do conteúdo dela, uma linha com o motivo; o resto da árvore continua |

## Rule uses

| Rule | Uses |
| --- | --- |
| `FLTRE-B01` | `root` |
| `FLTRE-B02` | `root`, `keystroke` |
| `FLTRE-B03` | `keystroke` |
| `FLTRE-B04` | `keystroke` |
| `FLTRE-E01` | `root` |
| `FLTRE-B05` | `root` |

## Revisions

> **FLTRE-R0001:** entra o `FLTRE-B05` (a árvore de outra máquina), do plano ESRMS, W05.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
