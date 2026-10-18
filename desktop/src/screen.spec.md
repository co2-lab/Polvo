<!-- @anchors
  code: SCRNE
  updated_at: 2026-10-17
-->
# Screen — o terminal: a shell num pty e o ecrã na libghostty-vt

> **Code**: `SCRNE`

## Overview
O `Screen` é um terminal sem janela. Corre a shell num pty (o `portable-pty`, que é ConPTY no Windows e
o pty do Unix nos outros) e guarda o estado do ecrã na libghostty-vt, o núcleo do Ghostty: o texto, as
cores, o cursor, o histórico e a seleção. A janela só desenha o que o `Screen` lhe dá e passa-lhe o
teclado e o rato.

- O `spawn` arranca a shell num pty com o tamanho (`Size`) e devolve o `Screen` e o canal (`Events`)
  dos acontecimentos (`Event`): a saída da shell e o fim dela.
- O `new` faz um `Screen` sem pty, que escreve num `Write` qualquer; é o que os testes usam para ver o
  que o terminal manda à shell.
- O `feed` passa a saída da shell à libghostty-vt.
- O `key` e o `paste` mandam uma tecla (`Input`) ou um texto colado à shell.
- O `resize`, o `scroll` e os gestos (`press`, `drag`, `release`) mudam o tamanho, o histórico à vista
  e a seleção.
- O `frame` dá o desenho do ecrã (`Frame`): as linhas em troços (`Run`) com o texto e as cores
  (`Rgb`), as cores por omissão e a paleta. O `text`
  dá só o texto, e o `selection_text` o texto selecionado.

## Rules

### SCRNE-B01 — A shell corre num pty do tamanho do terminal
O `spawn` corre o programa dado num pty com as linhas e as colunas do `Size`, com `TERM=xterm-256color`
e `COLORTERM=truecolor`, na pasta dada.

### SCRNE-B02 — A saída da shell é o ecrã
O que a shell escreve, passado ao `feed`, é o ecrã: o texto, as mudanças de linha, as cores e o movimento
do cursor, como o Ghostty os interpreta.

### SCRNE-B03 — As respostas do terminal voltam à shell
Quando a shell pergunta ao terminal (a posição do cursor, os atributos do terminal), a resposta da
libghostty-vt é escrita para a shell.

### SCRNE-B04 — O tamanho muda o terminal e o pty
O `resize` muda as linhas e as colunas da libghostty-vt (o texto volta a quebrar nas novas colunas) e as
do pty, e a shell vê o tamanho novo.

### SCRNE-B05 — O scroll mostra o histórico, e escrever volta ao fim
O `scroll` move a vista pelo histórico (para cima com um número negativo). Uma tecla ou um texto colado
voltam a vista ao fim, onde a shell escreve.

### SCRNE-B06 — A seleção segue os gestos do rato
Os gestos em píxeis, contados a partir do canto da grelha, fazem a seleção como no Ghostty:
- carregar e arrastar seleciona de uma célula à outra, e uma célula entra quando o rato passa do meio
  dela;
- dois cliques seguidos (a menos de `REPEAT_CLICK` e de uma célula um do outro) selecionam a palavra;
- três cliques seguidos selecionam a linha.

Um clique sem arrastar não seleciona nada.

O `selection_text` dá o texto selecionado, sem os espaços do fim das linhas.

### SCRNE-B07 — Colar respeita o modo do terminal
Com o modo de colar entre parênteses ligado pela shell, o texto colado vai entre `ESC [ 200 ~` e
`ESC [ 201 ~`. Sem ele, as mudanças de linha vão como `\r`. Em ambos, os caracteres de controlo
perigosos passam a espaços.

### SCRNE-B08 — O desenho mostra o cursor e a seleção
No `frame`, a célula do cursor (quando ele está visível e na vista) e as células selecionadas têm a
frente e o fundo trocados. Os troços juntam as células seguidas com as mesmas cores e o mesmo estilo.

### SCRNE-B09 — O fim da shell chega pelo canal
Quando a shell termina, o canal dá o `Event::Exit` depois da última saída.

### SCRNE-B10 — O título do terminal é o que a shell lhe dá
O `title` é o título que a shell pôs no terminal (a sequência `OSC 0` ou `OSC 2`, a que o `bash` e o
`zsh` usam para mostrar a pasta ou o comando); vazio enquanto a shell não puser nenhum.

### SCRNE-B11 — Largar o terminal termina a shell
Quando o `Screen` deixa de existir (a aba do terminal fechou), a shell dele é terminada, e o canal dá
o fim. Nenhuma shell fica a correr sem terminal.

### SCRNE-B12 — O terminal de uma sessão
O `attach` faz o terminal de uma sessão de uma máquina (`RemoteTerm`), em vez de um pty da janela: a
saída da sessão é o ecrã (primeiro o ecrã atual dela), o que o terminal envia (as teclas, o que se cola,
as respostas) vai para a sessão, o tamanho muda o da sessão, e o fim dela chega pelo canal. Largar o
terminal larga a sessão, que continua (quem a termina é quem a fez), e o canal dá o fim.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `argv` | o programa e os argumentos | vazio, ou um programa que não arranca | este módulo: o erro do `spawn` (`SCRNE-E01`) |
| `Size` | linhas e colunas maiores que zero | zero | este módulo: o mínimo é uma linha e uma coluna |
| `data` | qualquer sequência de bytes da shell | — | a libghostty-vt: o que não entende ignora, sem estragar o estado |
| `RemoteTerm` | o terminal de uma sessão viva | uma sessão que acabou | este módulo: o canal dá o fim |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `SCRNE-E01` | o programa da shell não arranca (não existe, sem permissão), ou o pty não abre | o `spawn` devolve o erro com o programa e o motivo, e não há `Screen` |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `SCROLLBACK` | 10000 | as linhas de histórico, o mesmo do Ghostty por omissão |
| `REPEAT_CLICK` | 500 ms | a pausa máxima entre os cliques de um duplo clique, a do GNOME por omissão |

## Rule uses

| Rule | Uses |
| --- | --- |
| `SCRNE-B01` | `argv`, `Size` |
| `SCRNE-B02` | `data` |
| `SCRNE-B03` | `data` |
| `SCRNE-B04` | `Size` |
| `SCRNE-B05` | `data`, `SCROLLBACK` |
| `SCRNE-B06` | `data`, `Size`, `REPEAT_CLICK` |
| `SCRNE-B07` | `data` |
| `SCRNE-B08` | `data` |
| `SCRNE-B09` | `argv` |
| `SCRNE-B10` | `data` |
| `SCRNE-B11` | `argv` |
| `SCRNE-B12` | `RemoteTerm`, `Size`, `data` |
| `SCRNE-E01` | `argv` |

## Revisions

> **SCRNE-R0001:** entram o `SCRNE-B10` (o título) e o `SCRNE-B11` (largar o terminal termina a shell),
> para o painel do terminal na doca (plano JNNTJ, fase W02): a aba mostra o título, e fechá-la não
> deixa a shell órfã. As outras regras não mudam.

> **SCRNE-R0002:** entra o `SCRNE-B12` (o terminal de uma sessão), para a janela mostrar as sessões desta
> máquina e das outras pelo nó (plano ESRMS, W02).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
