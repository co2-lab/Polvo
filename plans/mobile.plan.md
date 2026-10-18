<!-- @anchors
  code: MOBIL
  updated_at: 2026-10-18
-->
# Mobile

> **Code**: `MOBIL`
>
> Pedido pelo dono a 2026-10-18: uma versão mobile do Polvo para controlar as sessões das outras máquinas.
> A stack (Flutter) foi decidida pelo dono a 2026-10-18. As fases estão por desenhar, depois do P2P
> (`ESRMS-W04`) e da identidade (`IDNTD`).

## Objective

No telemóvel (iOS e Android), ver as máquinas da pessoa e as que lhe partilharam, ligar-se às sessões
delas de qualquer rede, ver o ecrã e escrever nelas, e aceitar convites.

## Rationale

O telemóvel é só cliente: não tem sessões nem nó. O centro da app é um terminal num ecrã pequeno, com o
teclado do ecrã e as teclas especiais.

## Design

- **A UI em Flutter** (uma app para iOS e Android).
- **O núcleo partilhado com o desktop**, pela API C: o P2P (`p2p/`, em Rust, pelo `flutter_rust_bridge`),
  o cliente das sessões (a `libpolvo`, em Go), e a emulação do terminal (a `libghostty-vt`, a mesma do
  desktop), que a app desenha num canvas do Flutter.
- **O que é de cada plataforma**, por plugins: o keychain, o login pelo browser do sistema
  (`ASWebAuthenticationSession`, Custom Tabs), a câmara para o QR code, e voltar a ligar quando a app
  regressa do segundo plano.

## Phases

Por desenhar.

## Out of Scope

- Sessões no telemóvel (o telemóvel não é um nó).

## Revisions

> **MOBIL-R0001:** o dono escolheu o Flutter a 2026-10-18, sobre a comparação do agente com o React Native
> (o terminal numa WebView é onde é mais fraco) e o nativo em Swift e Kotlin (a melhor experiência, mas
> duas apps para manter).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

## Definition of Done

- Por definir com as fases.
