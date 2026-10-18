<!-- @anchors
  code: BRWSR
  updated_at: 2026-10-18
-->
# Browser

> **Code**: `BRWSR`
>
> Pedido pelo dono a 2026-10-18, para referência futura: um browser dentro do Polvo, para o agente abrir e
> controlar uma página sem dependências externas. Ainda sem fases: o desenho começa quando o dono o pedir.

## Objective

Uma aba do Polvo com uma página web, que o utilizador vê e usa, e que o agente (pelo `polvo mcp`) abre,
lê e controla: navegar, ler o conteúdo, carregar, escrever, e tirar uma imagem.

## Rationale

Hoje um agente que precisa de uma página depende de um browser instalado e de uma ferramenta à parte
(Playwright, Chrome com o DevTools Protocol). Dentro do Polvo, a página fica na mesma janela das sessões,
o utilizador vê o que o agente faz, e a mesma permissão das máquinas vale para ela.

## Design

Por desenhar. As perguntas que o desenho tem de responder, com pesquisa:
- o motor: a WebView do sistema (WebView2, WKWebView, WebKitGTK, pelo `wry`), um Chromium embutido
  (CEF), ou um motor em Rust (Servo); e como o gpui a mostra numa aba;
- o controlo: as ferramentas MCP (`polvo_browser_*`) e o protocolo por baixo (o do motor, ou o WebDriver
  BiDi);
- uma página de outra máquina: o browser corre na máquina da sessão, ou nesta, pelos sockets encaminhados.

### O controlo como o do Playwright (pedido do dono a 2026-10-18)

O agente controla a página do Polvo como o Playwright controla um browser: navegar, esperar por um
elemento, carregar, escrever, ler o texto e o DOM, tirar uma imagem, ver os pedidos de rede. O Playwright
fala com cada motor pelo protocolo de automação dele (o CDP do Chromium; o Firefox e o WebKit com versões
modificadas), e isso decide o motor:
- **um Chromium embutido (CEF)** dá o CDP igual nos três sistemas: as ferramentas do Polvo falam CDP, e
  até o Playwright de fora se pode ligar (`connectOverCDP`); custa uns 100 MB e as atualizações do
  Chromium;
- **a WebView do sistema** é leve, mas o controlo não é igual: o WebView2 (Windows) tem o CDP, o
  WebKitGTK (Linux) tem o WebDriver, e o WKWebView (macOS) não tem automação; o que sobra é injetar
  JavaScript, que não dá eventos de teclado e rato verdadeiros, nem iframes de outras origens, nem a rede.

Decidido pelo dono a 2026-10-18, sobre a proposta do agente: o CEF, porque o controlo é o objetivo e só
ele o dá igual em todo o lado; as ferramentas são nossas (`polvo_browser_*`, com a forma da API do
Playwright) sobre o CDP.

**No mobile** o CEF não corre (é só Windows, macOS e Linux), e o browser fica fora do mobile (abaixo). Se
um dia entrar: no Android a WebView do sistema é um Chromium e dá o CDP (com o
`setWebContentsDebuggingEnabled`), por isso as mesmas ferramentas servem; no iOS todo o browser tem de usar
o WebKit (a regra da App Store, menos para os browsers na UE), e o WKWebView não tem um protocolo de
automação oficial: sobra o JavaScript injetado, com os limites de cima.

## Phases

Por desenhar.

## Out of Scope

- O mobile: o browser é só do desktop (o dono, a 2026-10-18).

## Revisions

> **BRWSR-R0001:** o dono decidiu a 2026-10-18 o CEF e as ferramentas ao estilo do Playwright sobre o CDP,
> e perguntou pelo mobile: o CEF não corre lá; o Android daria o CDP pela WebView, o iOS não.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

## Definition of Done

- Por definir com as fases.
