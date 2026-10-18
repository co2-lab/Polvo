<!-- @anchors
  code: MQNAS
  updated_at: 2026-10-17
-->
# Nodes — a aba das máquinas e das sessões delas

> **Code**: `MQNAS`

## Overview
A aba das máquinas (`NodesPanel`) mostra, pela instalação (`Station`), esta máquina e as que o
utilizador juntou, cada uma com as suas sessões de terminal. Dela abrem-se as sessões em abas do
terminal, criam-se sessões novas, travam-se as desta máquina, e juntam-se e esquecem-se máquinas pelo
endereço. Abrir uma sessão diz à janela qual (`NodesEvent`), com o `SessionTab` do painel do terminal. O
que a aba pede à instalação corre fora da linha da janela, porque uma máquina pode demorar a responder.

## Rules

### MQNAS-B01 — As máquinas e as sessões delas
A aba mostra esta máquina primeiro, com o endereço onde o nó dela escuta (ou porque não escuta) e a chave
da instalação, que um botão copia para a área de transferência; depois as máquinas juntadas, pela ordem em
que se juntaram. Cada máquina mostra as suas sessões: o id, o programa, o tamanho, os espectadores, e se
está travada ou só se vê.

### MQNAS-B02 — Atualizar
O botão de atualizar lê tudo outra vez; cada ação da aba (criar, travar, juntar, esquecer) também.

### MQNAS-B03 — Abrir uma sessão
Carregar numa sessão abre-a numa aba do terminal ligada a ela (sem ser a dona), que escreve nela se a
janela pode escrever.

### MQNAS-B04 — Uma sessão nova
O botão de sessão nova de uma máquina cria lá uma sessão da shell e abre-a numa aba que é a dona dela.

### MQNAS-B05 — Travar as sessões desta máquina
Cada sessão desta máquina tem o botão de travar (só ver para quem se liga de fora) e de destravar.

### MQNAS-B06 — Juntar e esquecer uma máquina
Escrever o endereço de uma máquina e carregar em juntar junta-a; cada máquina juntada tem o botão de a
esquecer.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `Station` | a instalação aberta | — | o `main`: sem instalação a janela não mostra esta aba |
| `addr` | máquina e porta de um nó | um que não responde ou recusa a chave | este módulo: a mensagem (`MQNAS-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `MQNAS-E01` | uma máquina juntada não responde | a máquina mostra porquê no lugar das sessões, e as outras continuam |
| `MQNAS-E02` | juntar falha (não responde, recusa a chave) | a mensagem com o motivo, e as máquinas ficam como estavam |

## Rule uses

| Rule | Uses |
| --- | --- |
| `MQNAS-B01` | `Station` |
| `MQNAS-B02` | `Station` |
| `MQNAS-B03` | `Station` |
| `MQNAS-B04` | `Station` |
| `MQNAS-B05` | `Station` |
| `MQNAS-B06` | `Station`, `addr` |
| `MQNAS-E01` | `Station` |
| `MQNAS-E02` | `addr` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
