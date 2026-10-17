# Project insights

> Este é o registo da fase DESCOBRIR. As DECISÕES estão no PROJECT.md; aqui ficam as perguntas, as
> respostas e o que foi descartado.
>
> ⚠️ **A entrevista não foi feita etapa a etapa.** O dono definiu a meta ("criar a nova versão do
> Polvo usando o Galaxy/Bavel como núcleo") e pediu para não parar a perguntar. Por isso:
> - **R (dono):** são as palavras do dono, tiradas da conversa de 2026-10-17.
> - **Decidido pelo agente:** é uma decisão que o agente tomou sozinho e que fica **para o dono
>   rever**.
>
> Quando o dono mudar uma decisão, altera-se o PROJECT.md e regista-se a mudança aqui.

## Etapa 1 — Propósito e forma

**R (dono):** "a minha versão do polvo será um projeto desktop cross platform que funciona como um
editor simples e um terminal mas com a capacidade de se comunicar com processos em outras máquinas,
como se um terminal remoto, vou usar um agente na minha máquina no meu polvo pra se
comunicar/controlar os outros agentes em outras máquinas"; "eu vou fazer um polvo do zero, não vou
reaproveitar o código do polvo"; "a ideia é fazer tudo em um executável só em vez de ter um monte de
processo trabalhando junto em paralelo".

**Decidido:**
- Um só executável, com os subcomandos `polvo`, `polvo node`, `polvo mcp` e `polvo keygen`.
- O agente local entra pelo MCP por stdio (decidido pelo agente).

**Descartado:**
- **Dois binários (um desktop e um daemon).** Contraria o "um executável só". O mesmo binário em
  dois modos também dá uma única versão para atualizar.
- **O agente local a falar com o nó por uma API HTTP própria.** O MCP é o que os agentes (Claude Code
  e outros) já falam, e por stdio não abre nenhuma porta.
- **Reaproveitar o Polvo antigo** (Go + React + Tauri + sidecar por HTTP/WS). O dono recomeça do zero
  com outro propósito.

## Etapa 2 — Linguagem e runtime

**R (dono):** "começar a construir usando o galaxy como núcleo"; "não fique preso no que o galaxy tem
hoje e sim no que precisamos que ele tenha".

**Decidido:**
- O núcleo em Go 1.26, a correr no Galaxy, com o `go` como oráculo.
- A UI em TypeScript sem framework (decidido pelo agente).

**Porquê o Go:**
- O frontend Go é o mais avançado do Galaxy: a suíte oficial dá 118/141, e boa parte dos pacotes do GO Feature Flag passam nos testes.
- O `x/crypto/ssh` já corre de ponta a ponta no Galaxy: servidor e cliente no mesmo processo,
  handshake `mlkem768x25519`, chave ed25519, canal e `exec` iguais ao `go run` (Bavel, caixa 1386,
  commit `e2baa92d`).

**Descartado:**
- **Rust.** O Galaxy não tem frontend Rust ao nível do Go.
- **TypeScript/Node no núcleo.** O motor node do Galaxy ainda não tem o `http` completo (a caixa 1384
  está estacionada), e o núcleo precisa de PTY e de SSH.
- **React na UI.** A UI tem dois componentes pesados, o xterm.js e o CodeMirror, e pouco estado à
  volta deles. O React só acrescentava o build e o runtime.
- **Tauri / Electron como casca.** São mais processos e mais runtimes, contra o "executável só".
  - A casca nativa vem do Galaxy: um binding de webview, que é uma lacuna do Galaxy e entra como
    caixa no Bavel.
  - Até lá, `polvo` serve a UI em `127.0.0.1` e abre o browser.

## Etapa 3 — Arquitetura e paradigma

**Decidido pelo agente:**
- Modular por domínio em `internal/`, procedural com tipos, interfaces só nas fronteiras de I/O.
- SSH como transporte.
- JSON-RPC 2.0 num subsistema SSH `polvo` para o controlo.
- O terminal remoto pelo canal `session` normal do SSH (`pty-req` + `shell`). Assim um `ssh` comum
  também entra num nó.

**Descartado:**
- **Um protocolo próprio sobre TCP/TLS.** O SSH já dá autenticação por chave, multiplexação de canais,
  `known_hosts` e interoperabilidade com as ferramentas de sempre.
- **gRPC.** Exige geração de código e HTTP/2, e não acrescenta nada ao SSH para um controlo ponto a
  ponto.
- **Camadas horizontais** (handlers/services/repositories). O Polvo tem domínios com fronteiras
  claras (nó, hub, transporte, UI, MCP), não um CRUD.
- **O nó a ligar-se ao desktop (conexão reversa, para atravessar NAT).** Fica fora até haver um caso
  concreto. Não está no PROJECT.md porque não foi decidido.

## Etapa 4 — Estrutura macro e convenções de ficheiros

**Decidido pelo agente:**
- A árvore do PROJECT.md.
- Os testes ao lado do código (`*_test.go`, `*.test.ts`).
- As specs (`<nome>.spec.md`) e as features (`<nome>.feature`) ao lado do código, como o `anchors init --colocation` as procura.
- O `make test` corre o `go test` e o `galaxy test`.

**Descartado:**
- **Uma árvore `tests/` separada.** Contraria o idioma do Go, e o `galaxy test` mede o pacote como o
  `go test`.
- **`pkg/` para o código público.** O Polvo é uma aplicação, não uma biblioteca: tudo é `internal/`.

## Etapa 5 — Ferramentas e formatação

**Decidido pelo agente:**
- gofmt e go vet; prettier e `tsc --strict`; largura 100.
- O código em inglês (como no Polvo antigo) e os documentos em português (como o dono escreve).

**Descartado:**
- **biome em vez de prettier.** Seria mais uma ferramenta nova. O prettier é o que o ecossistema do
  xterm.js e do CodeMirror usa.
- **Identificadores em português (como no Bavel).** O Polvo é open source, para a comunidade Go/TS.

## Revisão de inconsistências

- **O paradigma procedural e a UI.** A UI em TypeScript sem framework também é procedural (funções e
  módulos ES); não colide.
- **A co-locação "yes" e as `specs/` no topo.** COLIDIA: o `anchors init --colocation` procura spec,
  feature e teste ao lado do código. Resolvido a favor da co-locação; a pasta `specs/` saiu.
- **O padrão de teste do anchors (`*.test.*`) e o do Go (`*_test.go`).** COLIDIA: o `go test` só vê
  `*_test.go`. O `anchors.yaml` passou a `**/*_test.go` e `{{name}}_test.{{ext}}`.
- **"Tudo num executável" e o npm.** O npm só entra no build da UI; o artefacto continua a ser um
  binário com a UI embutida (`//go:embed`); não colide.
- **"O Galaxy como alvo" e o `golang.org/x/crypto/ssh`.** Já corre (caixa 1386).
  - Ainda por provar no Galaxy: o PTY, o `net/http` com WebSocket, e o MCP por stdio.
  - Cada lacuna que aparecer vira uma caixa no Bavel (regra "Dois motores" do PROJECT.md).
