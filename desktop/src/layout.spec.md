<!-- @anchors
  code: DCLYD
  updated_at: 2026-10-17
-->
# Layout — a organização da doca, guardada e restaurada

> **Code**: `DCLYD`

## Overview
A organização da doca (as abas, os grupos, as divisões e os tamanhos, o `DockAreaState` do
gpui-component) guarda-se num ficheiro e volta na próxima vez que o Polvo abre. O `LayoutFile` é esse
ficheiro: o `default` dá o da instalação, o `read` lê a organização guardada e o `write` grava-a.

## Rules

### DCLYD-B01 — O ficheiro fica na pasta de configuração do sistema
O ficheiro por omissão é o `layout.json` na pasta `polvo` da pasta de configuração do utilizador, a de
cada sistema: no Linux o `XDG_CONFIG_HOME` (ou `~/.config`), no macOS o `~/Library/Application
Support`, no Windows o `%APPDATA%`.

### DCLYD-B02 — Gravar nunca deixa o ficheiro a meio
O `write` cria a pasta quando falta, escreve a organização num ficheiro ao lado e só depois lhe dá o
nome do ficheiro, de uma vez: quem lê vê a organização antiga ou a nova, nunca metade.

### DCLYD-B03 — Ler devolve a organização guardada
O `read` devolve a organização que o `write` gravou, igual. Sem ficheiro, não há organização, e não é
um erro: é a primeira vez.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `layout.json` | uma organização gravada pelo `write`, da versão `LAYOUT_VERSION` | JSON estragado, ou de outra versão | este módulo: nenhuma organização (`DCLYD-E01`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `DCLYD-E01` | o ficheiro não é uma organização (estragado) ou é de outra versão | nenhuma organização, e o aviso com o ficheiro e o motivo na saída de erro |
| `DCLYD-E02` | a pasta ou o ficheiro não se escrevem (sem permissão, disco cheio) | o erro volta a quem gravou, e o ficheiro anterior fica como estava |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `LAYOUT_VERSION` | 1 | a versão da organização; muda quando os painéis mudam de forma, e a antiga deixa de servir |

## Rule uses

| Rule | Uses |
| --- | --- |
| `DCLYD-B01` | `layout.json` |
| `DCLYD-B02` | `layout.json` |
| `DCLYD-B03` | `layout.json`, `LAYOUT_VERSION` |
| `DCLYD-E01` | `layout.json`, `LAYOUT_VERSION` |
| `DCLYD-E02` | `layout.json` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
