<!-- @anchors
  code: MQNAS
  updated_at: 2026-10-18
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

### MQNAS-B07 — As máquinas da rede
O botão de procurar procura as máquinas que se anunciam na rede (`found`), e a aba mostra-as: uma que esta
instalação não conhece tem o botão de emparelhar; uma conhecida que não está nas máquinas, o de juntar; as
outras não têm botão.

### MQNAS-B08 — Emparelhar
Emparelhar com uma máquina começa o emparelhamento e pede o código que aparece nela; confirmar dá-lho, e
quando acaba bem a máquina passa a estar nas máquinas; cancelar desiste.

### MQNAS-B09 — Os pedidos de emparelhamento desta máquina
A aba mostra os pedidos de emparelhamento que esta máquina recebe, com o nome de quem pede, a impressão
digital e o código, e lê-os outra vez a cada `PAIRINGS_EVERY` enquanto vive.

### MQNAS-B10 — Os ficheiros de uma máquina
O botão "Ficheiros" de uma máquina pede à janela a árvore dos ficheiros dela, a começar na pasta pessoal.

### MQNAS-B11 — Deixar outras máquinas ligarem-se
Esta máquina só aceita ligações de outras quando o utilizador o liga: o botão "Deixar outras máquinas
ligarem-se" põe o nó dela a escutar em `ACCEPT_LISTEN` (e a anunciar-se na rede), e o mesmo botão,
"Deixar de aceitar outras máquinas", desliga-o e corta as ligações delas. A escolha fica para a próxima
vez que a janela abre.

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
| `MQNAS-E03` | emparelhar falha (o código errado, a máquina) | a mensagem com o motivo, e as máquinas ficam como estavam |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `PAIRINGS_EVERY` | 1 s | um pedido aparece logo; ler a lista desta máquina não custa nada |
| `ACCEPT_LISTEN` | `:2222` | o endereço do `polvo node`: quem já liga a máquina pelo `polvo node` encontra-a no mesmo sítio |

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
| `MQNAS-B07` | `Station`, `addr` |
| `MQNAS-B08` | `Station`, `addr` |
| `MQNAS-B09` | `Station`, `PAIRINGS_EVERY` |
| `MQNAS-B10` | `Station` |
| `MQNAS-B11` | `Station`, `ACCEPT_LISTEN` |
| `MQNAS-E03` | `addr` |

## Revisions

> **MQNAS-R0001:** entram as máquinas da rede e o emparelhamento (`MQNAS-B07` a `MQNAS-B09`, `MQNAS-E03`),
> do plano ESRMS, W03.

> **MQNAS-R0003:** entra o `MQNAS-B11` (deixar outras máquinas ligarem-se): o dono decidiu a 2026-10-18
> que a janela não escuta por omissão.

> **MQNAS-R0002:** entra o `MQNAS-B10` (os ficheiros de uma máquina), do plano ESRMS, W05.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
