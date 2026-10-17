<!-- @anchors
  code: PTYXX
  updated_at: 2026-10-17
-->
# Pty — um processo dentro de um terminal virtual

> **Code**: `PTYXX`

## Overview
O nó corre a shell (ou outro programa interativo) de quem se liga dentro de um terminal virtual, para
o programa se comportar como num terminal a sério: as cores, o editor de linha, o tamanho da janela, o
Ctrl-C. A sessão de terminal é o par: o processo, e o lado de fora do terminal, por onde se lê o que o
processo escreve e se escreve o que ele lê. É o `Session`, nascida pelo `Start`.

## Rules

### PTYXX-B01 — O processo nasce dentro de um terminal com o tamanho pedido
O processo vê-se ligado a um terminal (a entrada, a saída e o erro são esse terminal, e é o terminal
de controlo dele), com as linhas e as colunas pedidas no `Start` (o `Size`).

### PTYXX-B02 — O que o processo escreve sai na sessão, e o que se escreve na sessão chega ao processo
A leitura da sessão (`Read`) devolve o que o processo escreveu no terminal. A escrita na sessão
(`Write`) chega ao processo como se fosse teclada.

### PTYXX-B03 — O redimensionamento chega ao processo
Mudar o tamanho da sessão (`Resize`) muda as linhas e as colunas que o processo vê no terminal.

### PTYXX-B04 — O fim do processo devolve o código de saída
Esperar pelo processo (`Wait`) devolve o código de saída dele quando ele termina. Um processo morto por
um sinal devolve o código -1 e o nome do sinal (`ExitStatus`). Um sinal enviado à sessão (`Signal`)
chega ao processo.

### PTYXX-B05 — Depois do fim do processo, a leitura acaba sem erro
Quando o processo termina e já se leu tudo o que ele escreveu, a leitura da sessão dá o fim normal do
fluxo, e não um erro.

### PTYXX-B06 — Fechar a sessão termina o processo
Fechar a sessão (`Close`) envia ao processo o sinal de terminal desligado e liberta o terminal. Um
processo que já tinha terminado não é afetado.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `argv` | o programa (um nome no PATH ou um caminho) e os argumentos | uma lista vazia | este módulo: recusa-a com erro (`PTYXX-E02`) |
| `Size` | linhas e colunas entre 1 e 65535 | zero ou negativo | este módulo: um zero passa a 24 linhas por 80 colunas |

## Invariants

| Rule | Always holds | How it is proven |
| --- | --- | --- |
| `PTYXX-I01` | o tamanho que o processo vê é o último que a sessão recebeu, seja o do `Start` seja o do último `Resize` | closed cycle: arrancar com um tamanho, redimensionar, e perguntar ao processo o tamanho |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `PTYXX-E01` | o programa não existe ou não se pode executar | o arranque falha com um erro que nomeia o programa, e nenhum processo fica a correr |
| `PTYXX-E02` | a lista do programa e argumentos é vazia | o arranque falha com erro, sem processo |

## Rule uses

| Rule | Uses |
| --- | --- |
| `PTYXX-B01` | `argv`, `Size` |
| `PTYXX-B02` | `argv` |
| `PTYXX-B03` | `Size` |
| `PTYXX-B04` | `argv` |
| `PTYXX-B05` | `argv` |
| `PTYXX-B06` | `argv` |
| `PTYXX-E01` | `argv` |
| `PTYXX-E02` | `argv` |
| `PTYXX-I01` | `Size` |

## Implementation Notes

- **@TBD: Windows.** O pseudo-console do Windows (ConPTY) fica para quando o Galaxy o tiver (caixa
  1391 do Bavel). Em Windows, o `Start` falha com um erro que o diz.

## Revisions

> **PTYXX-R0001:** a nota do Windows passou a nomear a caixa 1391 do Bavel (era 1389). A numeração das
> caixas seguintes mudou quando a 1388 e a 1389 nasceram antes; a decisão (`@TBD: Windows`) é a mesma.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
