<!-- @anchors
  code: APPUI
  updated_at: 2026-10-17
-->
# App — a janela do Polvo: os nós, a árvore de pastas, o editor e os terminais

> **Code**: `APPUI`

## Overview
A UI do Polvo é uma página que o `polvo` abre no browser. À esquerda estão os nós (os endereços que o
utilizador acrescentou) e a árvore de pastas do nó escolhido. Ao centro fica o editor do ficheiro
aberto, com o botão de gravar. Em baixo ficam os terminais abertos nesse nó, e a barra de estado diz o
que aconteceu.

A janela é a `App`, construída sobre uma raiz da página e as dependências (`Deps`): a rede, o
armazenamento do browser, a fábrica do editor (`Editor`) e a fábrica do terminal (`Terminal`). As
chamadas de controlo passam pela `Api`, com o segredo que o `tokenFromHash` lê do endereço.

## Rules

### APPUI-B01 — O segredo vem do endereço e sai da barra de endereço
O segredo da sessão chega no fragmento do endereço (`#t=…`), lido pelo `tokenFromHash`. A janela
guarda-o e tira-o da barra de endereço (sem recarregar a página), para não ficar no histórico.

### APPUI-B02 — Os nós ficam lembrados e escolher um mostra a pasta dele
O utilizador acrescenta um nó pelo endereço (`addNode`). A lista fica guardada no browser e volta na
próxima abertura. Escolher um nó lista a pasta do campo do caminho (a raiz por omissão).

### APPUI-B03 — A árvore mostra as pastas e os ficheiros, e navega
A pasta aberta (`openFolder`) mostra primeiro a entrada para a pasta de cima, depois as pastas (com
uma barra no fim) e depois os ficheiros, cada grupo por ordem de nome. Clicar numa pasta abre-a, e
clicar num ficheiro abre-o no editor.

### APPUI-B04 — Abrir um ficheiro põe o texto no editor
Abrir um ficheiro (`openFile`) lê-o do nó, mostra o texto no editor e mostra o caminho como título.

### APPUI-B05 — O ficheiro alterado tem a marca, e gravar tira-a
Uma alteração no editor põe a marca de alterado ao lado do título. Gravar (o botão, ou Ctrl+S; o
`save`) escreve o texto do editor no nó, tira a marca e mostra a mensagem de gravado.

### APPUI-B06 — O terminal liga-se ao nó por WebSocket
Um terminal novo (`newTerminal`) abre o WebSocket do terminal com o segredo, o nó e o tamanho do
terminal.
- O que se escreve vai como entrada, e a mudança de tamanho vai como novo tamanho.
- A saída da shell aparece no terminal.
- Quando a shell termina, aparece a mensagem de fim com o código.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `hash` | o fragmento do endereço com `t=` | sem `t=` | esta unidade: a mensagem de falta de segredo (`APPUI-E02`) |
| `node` | um endereço de nó (máquina:porta) | vazio | esta unidade: o campo vazio não acrescenta nada |
| `path` | um caminho no nó | vazio | esta unidade: o vazio é a raiz |

## States

### APPUI-S01 — Sem segredo
**When**: o endereço não traz o segredo.

**Elements** — `[V]` visible · `[D]` dynamic · `[X]` disabled · `[H]` hidden · `[O]` optional:

- `[V]` a mensagem de falta de segredo (`APPUI-M01`)
- `[H]` a lista dos nós, a árvore, o editor e os terminais

### APPUI-S02 — Pronta
**When**: o segredo foi lido.

**Elements** — `[V]` visible · `[D]` dynamic · `[X]` disabled · `[H]` hidden · `[O]` optional:

- `[V]` a lista dos nós e o campo para acrescentar
- `[D]` a árvore da pasta do nó escolhido
- `[D]` o editor com o título do ficheiro e a marca de alterado
- `[D]` os terminais
- `[V]` a barra de estado

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `APPUI-E01` | uma chamada ao nó falha (o nó não responde, o ficheiro não existe) | a barra de estado mostra o erro (`APPUI-M02`), e o resto da janela fica como estava |
| `APPUI-E02` | o endereço não traz o segredo | a janela mostra só a mensagem `APPUI-M01` |

## User Messages
| Rule | Condition | Message |
| --- | --- | --- |
| `APPUI-M01` | sem o segredo da sessão | Abra o Polvo pelo endereço que o comando polvo mostrou (falta o segredo da sessão). |
| `APPUI-M02` | uma chamada falhou | Erro: {mensagem} |
| `APPUI-M03` | o ficheiro foi gravado | Gravado: {caminho} |
| `APPUI-M04` | a shell do terminal terminou | [terminou: código {código}] |

## Rule uses

| Rule | Uses |
| --- | --- |
| `APPUI-B01` | `hash` |
| `APPUI-B02` | `node`, `path` |
| `APPUI-B03` | `node`, `path` |
| `APPUI-B04` | `node`, `path` |
| `APPUI-B05` | `node`, `path`, `APPUI-M03` |
| `APPUI-B06` | `node`, `APPUI-M04` |
| `APPUI-E01` | `node`, `APPUI-M02` |
| `APPUI-E02` | `hash`, `APPUI-M01` |
| `APPUI-S01` | `hash`, `APPUI-M01` |
| `APPUI-S02` | `hash` |
| `APPUI-M01` | `hash` |
| `APPUI-M02` | `node` |
| `APPUI-M03` | `path` |
| `APPUI-M04` | `node` |

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
