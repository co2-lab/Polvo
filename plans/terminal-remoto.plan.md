<!-- @anchors
  code: TRRMT
  updated_at: 2026-10-17
-->
# TerminalRemoto

> **Code**: `TRRMT`
>
> ⚠️ O âmbito deste plano foi aprovado pelo agente e **ainda não foi revisto pelo dono**. A meta dada
> pelo dono foi "criar a nova versão do Polvo usando o Galaxy/Bavel como núcleo", com o pedido de não
> parar para perguntar. O registo está no INSIGHTS.md.

## Objective

Um `polvo` numa máquina abre um terminal interativo noutra máquina onde corre o `polvo node`, e chama
operações de controlo sobre ela (ler e escrever ficheiros, correr um processo e ler a saída). O mesmo
binário serve os dois lados e dá o mesmo resultado compilado pelo `go` e corrido pelo Galaxy.

## Rationale

O terminal remoto é o centro do produto: o editor, a UI e o agente local (o MCP) são clientes do mesmo
canal. Sem ele não há nada para a UI mostrar nem para o agente controlar. É também o primeiro programa
real que põe o Galaxy à prova como núcleo: SSH, PTY, processos e ficheiros.

## Phases

### TRRMT-W01 — O protocolo e as identidades

Entrega a linguagem comum entre o desktop e o nó, e as chaves que os identificam.

- `internal/proto/codec.spec.md` — as mensagens de controlo (pedido, resposta, notificação, erro), o
  enquadramento num fluxo de bytes e o que acontece a uma mensagem malformada
- `internal/sshx/keys.spec.md` — o par de chaves de uma instalação, as chaves autorizadas num nó e as
  chaves de nó conhecidas num desktop (a primeira ligação e a chave que mudou)

### TRRMT-W02 — O nó (depends on TRRMT-W01)

Só começa depois da W01, porque o nó fala o protocolo e autentica com as chaves dela.

- `internal/pty/pty.spec.md` — um terminal virtual com um processo dentro, o tamanho da janela e o fim
  do processo
- `internal/node/server.spec.md` — o nó: quem entra, a sessão de terminal, o comando avulso e as
  operações de controlo (ficheiros e processos)

### TRRMT-W03 — O desktop na linha de comandos (depends on TRRMT-W02)

Só começa depois da W02, porque precisa de um nó para se ligar.

- `internal/hub/client.spec.md` — a ligação a um nó, o terminal interativo e as chamadas de controlo
- `cmd/polvo/cli.spec.md` — os subcomandos `node`, `keygen` e `connect`, os argumentos e os códigos de
  saída

## Out of Scope

- A UI (o editor e o terminal no ecrã): plano seguinte, sobre o `internal/hub` desta.
- O `polvo mcp` para o agente local: plano seguinte, sobre as operações de controlo desta.
- A casca nativa (a webview do Galaxy): é uma caixa no Bavel antes de ser um plano aqui.
- A ligação reversa (o nó a ligar-se ao desktop por trás de NAT): sem caso concreto ainda.
- O Windows no nó (ConPTY): a W02 especifica o comportamento para todos os sistemas, mas só o
  implementa em Linux e macOS. O Windows fica `@TBD` na spec do PTY.

## Definition of Done

- `make test` passa com o `go test` e com o `galaxy test`, nas duas execuções.
- Com o `polvo node` numa máquina, o `polvo connect` noutra dá uma shell interativa, e o
  redimensionamento da janela chega ao nó.
- `anchors check` passa os gates bloqueantes.
