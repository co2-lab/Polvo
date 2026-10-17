<!-- @anchors
  code: SRVRE
  updated_at: 2026-10-17
-->
# Server — o nó: quem entra, a sessão de terminal e o comando avulso

> **Code**: `SRVRE`

## Overview
O nó é o `polvo node` numa máquina remota: um servidor SSH. Deixa entrar só as chaves autorizadas, e
cada ligação abre sessões. Numa sessão, quem liga pede uma shell interativa num terminal, ou um
comando avulso, ou o subsistema de controlo do Polvo. Os pedidos de sessão são os do SSH, e por isso
um cliente `ssh` comum também entra num nó. A configuração é o `Config`; o nó serve pelo `Serve`
(um ponto de escuta) ou pelo `ServeConn` (uma ligação já aberta).

## Rules

### SRVRE-B01 — Só entra uma chave autorizada
Uma ligação cuja chave pública está na lista de chaves autorizadas do nó entra. Uma ligação com outra
chave, ou sem chave (palavra-passe, teclado interativo), é recusada antes de abrir qualquer sessão.

### SRVRE-B02 — A shell interativa é uma sessão do nó
Uma sessão que pede um terminal e depois uma shell cria uma sessão nova nas sessões do nó (`Sessions`),
com a shell do nó (`Shell`) num terminal com o tamanho pedido, e liga-se a ela. O que a shell escreve
chega a quem ligou, e o que quem ligou escreve chega à shell. Fechar a ligação deixa a sessão a correr;
o fim da shell acaba a sessão e a ligação (`SRVRE-B06`).

### SRVRE-B03 — O redimensionamento da janela chega à shell
Um pedido de mudança de janela numa sessão com terminal muda o tamanho que o processo vê.

### SRVRE-B04 — O comando avulso corre sem terminal, com a saída e o erro separados
Uma sessão que pede um comando, sem ter pedido um terminal, corre-o pela shell do nó. A saída do
comando chega pelo canal de dados e o erro pelo canal de erro, separados, e a entrada da sessão chega
ao comando.

### SRVRE-B05 — O comando avulso com terminal corre no terminal
Uma sessão que pediu um terminal e depois um comando corre o comando nesse terminal, como a shell.

### SRVRE-B06 — O fim do processo devolve o estado e fecha a sessão
Quando o processo da sessão termina, o nó envia o código de saída (ou o sinal que o matou) e fecha a
sessão.

### SRVRE-B07 — O subsistema polvo entrega a sessão ao controlo
Uma sessão que pede o subsistema "polvo" é entregue ao controlo do nó (`Control`), que fala o protocolo
de mensagens sobre ela. Outro subsistema é recusado.

### SRVRE-B08 — O que o nó não serve é recusado
Um canal que não é de sessão (o reencaminhamento de portas) é recusado com o motivo de tipo
desconhecido. Um pedido de sessão que o nó não conhece é recusado, e a sessão continua.

### SRVRE-B09 — Ligar-se a uma sessão que já existe
O pedido `attach@polvo`, com o id de uma sessão do nó, liga a ligação a essa sessão como espectador:
recebe o ecrã atual e depois a saída, e as mudanças de janela mudam o tamanho da sessão. Fechar a
ligação deixa a sessão a correr.

### SRVRE-B10 — Cada ligação tem a permissão da chave dela
A permissão de uma ligação é a que o nó dá à chave dela (`Perms`, sem ela a de dono). Uma ligação que
só vê não cria sessões nem corre comandos (a shell e o comando avulso são recusados), e numa sessão a
que se ligou o que escreve não chega à shell.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `HostKey` | a identidade do nó | nenhuma | quem arranca o nó: o servidor recusa arrancar sem ela (`SRVRE-E01`) |
| `Authorized` | a lista de chaves autorizadas | nenhuma | quem arranca o nó: sem lista ninguém entra (`SRVRE-E01`) |
| `Shell` | o programa e os argumentos da shell | uma lista vazia | este módulo: a lista vazia passa a "/bin/sh" |
| `Sessions` | as sessões do nó | nenhumas | este módulo: o nó faz as suas |
| `Perms` | a permissão de cada chave | nenhuma | este módulo: sem ela, todas são donas |
| `id` | o id de uma sessão viva do nó | outro | este módulo: o pedido é recusado (`SRVRE-E03`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `SRVRE-E01` | a configuração não tem a identidade do nó ou a lista de chaves autorizadas | o nó recusa servir, com erro |
| `SRVRE-E02` | a shell ou o comando não arranca | a sessão recebe o código de saída 127 e a mensagem no canal de erro, e fecha |
| `SRVRE-E03` | o `attach@polvo` com um id que não é de nenhuma sessão | o pedido é recusado, e a ligação continua |

## Rule uses

| Rule | Uses |
| --- | --- |
| `SRVRE-B01` | `Authorized` |
| `SRVRE-B02` | `Shell` |
| `SRVRE-B03` | `Shell` |
| `SRVRE-B04` | `Shell` |
| `SRVRE-B05` | `Shell` |
| `SRVRE-B06` | `Shell` |
| `SRVRE-B07` | `HostKey` |
| `SRVRE-B08` | `HostKey` |
| `SRVRE-E01` | `HostKey`, `Authorized` |
| `SRVRE-B09` | `Sessions`, `id` |
| `SRVRE-B10` | `Perms` |
| `SRVRE-E02` | `Shell` |
| `SRVRE-E03` | `id` |

## Revisions

> **SRVRE-R0001:** as shells do nó passam a ser sessões que vivem sem quem as vê, com vários
> espectadores e a permissão de cada chave (plano ESRMS, W01): o `SRVRE-B02` muda, e entram o
> `SRVRE-B09` (ligar-se a uma sessão), o `SRVRE-B10` (a permissão) e o `SRVRE-E03`.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
