# Project: Polvo

Polvo é uma aplicação desktop multiplataforma (Linux, macOS, Windows): um editor simples e um terminal
que trabalham tanto na máquina local como em máquinas remotas. Um agente na máquina local usa o Polvo
para comunicar com os agentes e os processos das outras máquinas e para os controlar.

## Stack
| item | choice |
| --- | --- |
| language (core) | Go 1.26 |
| language (UI) | TypeScript 5, sem framework |
| manager | go mod (core); npm só para as dependências da UI |
| runtime | Galaxy (`GALAXY_ENGINE=go galaxy`) como alvo; o toolchain `go` como oráculo |
| transporte remoto | SSH (`golang.org/x/crypto/ssh`) |
| protocolo de controlo | JSON-RPC 2.0 no subsistema SSH `polvo` |
| terminal | PTY no nó (Unix: `openpty`; Windows: ConPTY) |
| UI (alvo) | janela nativa desenhada na GPU: GPUI (Rust) para a janela, libghostty-vt (Zig) para o núcleo do terminal; o editor num componente GPUI (por escolher) |
| UI (atual, até a nativa existir) | HTML/CSS/TS servida pelo próprio binário, xterm.js (terminal) e CodeMirror 6 (editor), aberta no browser |
| bundler da UI atual | esbuild |
| integração com o agente local | servidor MCP por stdio (`polvo mcp`) |

## Artefactos
Um único executável, `polvo`, com estes subcomandos:
- `polvo`: a aplicação desktop. Liga-se aos nós e serve a UI.
- `polvo node`: o daemon sem UI que corre em cada máquina remota (um servidor SSH).
- `polvo mcp`: a ponte MCP por stdio que o agente local usa para controlar os nós.
- `polvo keygen`: gera o par de chaves ed25519 de uma instalação.

## Architecture
- **Pattern:** modular por domínio, com portas nas fronteiras de I/O.
- **Paradigm:** procedural com tipos. Interfaces Go só nas fronteiras (transporte, PTY, sistema de
  ficheiros). Sem estado global mutável.
- **Organization:** um pacote por domínio em `internal/`. `cmd/polvo` só faz o parse dos argumentos e
  a ligação dos pacotes.
- **Boundaries:**
  - `internal/proto` define as mensagens e não importa nenhum outro pacote do Polvo.
  - `internal/node` e `internal/hub` falam só por `internal/proto` sobre um `io.ReadWriteCloser`.
  - O SSH vive só em `internal/sshx`.
  - Os testes ligam o hub e o nó por `net.Pipe`, sem rede.
- **Várias línguas num executável:** cada parte usa o melhor componente do mercado na língua dele (o
  servidor e o SSH em Go, a janela em Rust, o núcleo do terminal em Zig); o Galaxy junta-as num só
  executável. Cada componente corre como o oficial, sem alterações.
- **Dois motores:** todo o código Go e todos os testes têm de dar o mesmo resultado em `go` e em
  `galaxy`. Uma divergência é um defeito do Galaxy: corrige-se como caixa no Bavel, nunca com um
  contorno no Polvo.

## Macro structure
```
cmd/polvo/         main e subcomandos
internal/proto/    mensagens JSON-RPC (pedidos, respostas, eventos) e o codec
internal/sshx/     servidor e cliente SSH, chaves, authorized_keys, known_hosts
internal/node/     o daemon: sessões de terminal, processos, ficheiros
internal/pty/      a abstração de PTY por sistema
internal/hub/      o lado desktop: as ligações aos nós e o estado das sessões
internal/web/      o servidor HTTP/WebSocket da UI e os ficheiros embutidos
internal/mcp/      o servidor MCP por stdio para o agente local
ui/                o código TypeScript da UI (o bundle vai para internal/web/dist/)
```

## File conventions
| item | choice |
| --- | --- |
| extensions | `.go`, `.ts`, `.css`, `.html` |
| test (Go) | `*_test.go`, ao lado do código |
| test (UI) | `*.test.ts`, ao lado do código |
| co-location | yes |
| spec / feature | `<nome>.spec.md` e `<nome>.feature`, ao lado do código da unidade |

## Formatting
| item | choice |
| --- | --- |
| indentation | Go: tabs; TS/CSS/HTML: 2 espaços |
| line width | 100 |
| formatter | gofmt (Go), prettier (TS/CSS/HTML) |
| linter | go vet (Go), tsc `--strict` (TS) |
| names | Go: PascalCase exportado, camelCase interno; TS: camelCase, PascalCase para tipos |
| language of the code | inglês (identificadores, comentários, mensagens de erro) |
| language of the docs | português (PROJECT, INSIGHTS, specs) |

## Tooling
- editors: VS Code
- extensions: gopls, prettier
- versioned: `.editorconfig`, `Makefile`
- `make test` corre os testes Go com o `go test` e com o `galaxy test`, e falha se algum dos dois falhar.
