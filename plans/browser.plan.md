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

## Phases

Por desenhar.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

## Definition of Done

- Por definir com as fases.
