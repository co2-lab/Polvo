<!-- @anchors
  code: PRFSE
  updated_at: 2026-10-17
-->
# Prefs — as preferências do ambiente de trabalho que a janela segue

> **Code**: `PRFSE`

## Overview
A janela do Polvo segue o ambiente de trabalho em vez de impor o seu aspeto: a ordem e o lado dos
botões da barra de título, o tema claro ou escuro e a fonte monoespaçada do terminal. No GNOME, estes
valores vêm das chaves do `gsettings` (`button-layout`, `color-scheme` e `monospace-font-name`).

As preferências são o `Prefs`: os botões da esquerda e os da direita (`Button`), se o tema é escuro, o
nome da fonte e o tamanho dela em píxeis. O `read` lê-as do sistema; o `read_with` lê-as por uma função
que dá o valor de cada chave, e é o que os testes usam.

## Rules

### PRFSE-B01 — Os botões seguem a disposição do sistema
A disposição é o texto "esquerda:direita", com os nomes separados por vírgulas. `close`, `minimize` e
`maximize` são os botões, pela ordem em que aparecem; os outros nomes (`appmenu`, `icon`, `spacer`)
ignoram-se. Uma disposição sem ":" põe todos os botões à direita.

### PRFSE-B02 — O tema segue o esquema de cores
O esquema `prefer-dark` dá o tema escuro; `prefer-light` e `default` dão o tema claro.

### PRFSE-B03 — A fonte segue a fonte monoespaçada do sistema
A fonte monoespaçada é o nome seguido do tamanho em pontos ("Ubuntu Sans Mono 11"). O tamanho passa a
píxeis a 96 pontos por polegada, arredondado (11 pontos dão 15 píxeis). A variável `POLVO_FONT`, quando
existe, substitui o nome.

### PRFSE-B04 — Sem a preferência, o valor por omissão
Uma chave que o sistema não dá (outro ambiente de trabalho, ou sem o `gsettings`) usa o valor por
omissão desse item: os botões de minimizar, maximizar e fechar à direita, o tema escuro, e a fonte
"DejaVu Sans Mono" de 11 pontos.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `button-layout` | "esquerda:direita" com nomes separados por vírgulas | ausente | este módulo: o valor por omissão (`PRFSE-B04`) |
| `color-scheme` | `prefer-dark`, `prefer-light`, `default` | ausente ou outro | este módulo: ausente é o escuro (`PRFSE-B04`); outro é o claro |
| `monospace-font-name` | "Nome Tamanho" | ausente, sem tamanho, ou tamanho que não é número | este módulo: a fonte por omissão (`PRFSE-E01`) |
| `POLVO_FONT` | o nome de uma fonte | vazio | este módulo: vazio não substitui |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `PRFSE-E01` | a fonte monoespaçada não tem tamanho, ou o tamanho não é um número positivo | a fonte por omissão inteira (o nome e o tamanho), como se a chave faltasse, e o aviso com a fonte recusada na saída de erro |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULT_LAYOUT` | ":minimize,maximize,close" | a disposição mais comum (Windows, KDE, GNOME de origem com os três botões) |
| `DEFAULT_FONT` | "DejaVu Sans Mono" | vem com quase todas as distribuições Linux |
| `DEFAULT_POINTS` | 11 | o tamanho por omissão do GNOME |

## Rule uses

| Rule | Uses |
| --- | --- |
| `PRFSE-B01` | `button-layout` |
| `PRFSE-B02` | `color-scheme` |
| `PRFSE-B03` | `monospace-font-name`, `POLVO_FONT` |
| `PRFSE-B04` | `button-layout`, `color-scheme`, `monospace-font-name`, `DEFAULT_LAYOUT`, `DEFAULT_FONT`, `DEFAULT_POINTS` |
| `PRFSE-E01` | `monospace-font-name`, `DEFAULT_FONT`, `DEFAULT_POINTS` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
