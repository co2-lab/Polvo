<!-- @anchors
  code: IDNTD
  updated_at: 2026-10-18
-->
# Identidade

> **Code**: `IDNTD`
>
> Pedido pelo dono a 2026-10-18: um login social/federado (GitHub, Google, Apple, Microsoft, etc.) para
> encontrar as sessões das outras máquinas, partilhar e dar permissões; uma identidade reforça a distinção
> e o controlo. Com a versão mobile, para controlar as sessões das máquinas pelo telemóvel. O desenho e as
> fases são do agente; as decisões foram aprovadas pelo dono a 2026-10-18 (`IDNTD-R0002`).

## Objective

Uma pessoa entra no Polvo com a conta que já tem (GitHub, Google, Apple, Microsoft, ou o fornecedor OIDC da
empresa) e vê, em qualquer dispositivo, as máquinas dela e as que lhe partilharam, com as sessões de cada
uma. Cada pessoa tem um código Polvo (o nome dela no Polvo, como `adriel.silva`), e partilhar uma
máquina é convidar outra pessoa pelo código dela (ver, controlar, encaminhar sockets), em vez de trocar
chaves. Sem conta, o Polvo continua a funcionar como hoje (emparelhar por código).

## Rationale

Hoje a identidade é a chave de cada instalação: partilhar é copiar chaves ou emparelhar com um código, as
permissões do `node.json` nomeiam impressões digitais, e um telemóvel novo tem de ser emparelhado com cada
máquina. Uma conta junta as chaves de uma pessoa: as permissões passam a nomear pessoas, e encontrar as
máquinas é pedir a lista à conta.

## Design

- **O serviço de contas** (`polvo-conta`): um servidor nosso, open source e alojável por quem quiser (o da
  co2lab por omissão), que:
  - autentica pelos fornecedores (OIDC/OAuth 2.0 diretamente com cada um, e o OIDC genérico para o
    fornecedor de uma empresa); não guarda palavras-passe: a autenticação forte (o MFA) é a do fornecedor;
  - guarda as contas, os dispositivos de cada conta (a chave ed25519 de cada instalação, que é também o
    endereço P2P do iroh) e as partilhas;
  - emite certificados SSH de curta duração (`ssh.Certificate`, um dia) que ligam a chave de um
    dispositivo à conta: o nó confia na autoridade do serviço e lê a conta no certificado, como o
    Teleport e o smallstep fazem.
- **O código Polvo**: cada conta ganha, ao entrar a primeira vez, um código único que a identifica para os
  outros: o nome de utilizador do fornecedor (`adrielcodeco`) ou o nome e o apelido (`adriel.silva`), com
  um número no fim quando já está tomado; a pessoa pode mudá-lo depois. Por dentro, as partilhas guardam o
  id da conta, por isso mudar o código não as estraga. O email e a conta do fornecedor não se mostram a
  ninguém.
- **Convidar**: partilhar é convidar pelo código, com a permissão (ver, controlar, encaminhar). A outra
  pessoa vê o convite e aceita-o (ou recusa); só depois de aceitar a máquina aparece na lista dela e o nó a
  deixa entrar. Quem convidou pode retirar o acesso a qualquer momento.
- **O nó** aceita, além das chaves autorizadas, os certificados do serviço; o `node.json` ganha permissões
  por pessoa (`"people": {"github:joao": "control"}`), e as partilhas feitas na conta chegam ao nó
  assinadas pelo serviço. Os dispositivos da mesma conta são o dono.
- **Um dispositivo novo numa conta** só passa a valer depois de aprovado por outro que já lá está (como
  ligar um dispositivo no Signal): uma conta social roubada não dá logo as máquinas.
- **Nunca passa pelo serviço** o conteúdo das sessões nem os ficheiros: o tráfego é P2P (iroh) e cifrado
  de ponta a ponta pelo SSH. O serviço só sabe as contas, os dispositivos e as partilhas.
- **O login** no desktop é o fluxo do OAuth para apps nativas (RFC 8252: o browser do sistema, PKCE e um
  redirecionamento para o loopback); no mobile, a sessão de autenticação do sistema
  (`ASWebAuthenticationSession`, Custom Tabs). Na App Store, oferecer login social obriga a oferecer
  também um como o Sign in with Apple (a regra 4.8): a Apple entra na lista.

## Phases

### IDNTD-W01 — O serviço de contas

- `conta/` (Go) — nasce: o servidor; o login pelos fornecedores, as contas e o código Polvo de cada uma,
  os dispositivos, a aprovação de um dispositivo novo, e os certificados SSH
- `conta/providers.spec.md` — nasce: GitHub, Google, Apple, Microsoft e o OIDC genérico

### IDNTD-W02 — O nó aceita pessoas (depends on IDNTD-W01)

- `internal/nodeconf/nodeconf.spec.md` — muda: as permissões por pessoa, e as partilhas assinadas
- `internal/node/server.spec.md` — muda: os certificados do serviço, com a conta como identidade
- `internal/hub/client.spec.md` — muda: ligar-se com o certificado da conta

### IDNTD-W03 — Entrar na janela, e as máquinas da conta (depends on IDNTD-W02, ESRMS-W04)

- `internal/station/station.spec.md` — muda: a conta da instalação; as máquinas da conta e as
  partilhadas, pelo P2P
- `desktop/src/account.spec.md` — nasce: entrar, sair, e aprovar um dispositivo novo
- `desktop/src/nodes.spec.md` — muda: as máquinas da conta e as partilhadas com cada um

### IDNTD-W04 — Partilhar (depends on IDNTD-W03)

- `conta/shares.spec.md` — nasce: convidar uma pessoa pelo código Polvo para uma máquina (ver,
  controlar, encaminhar), aceitar ou recusar o convite, e retirar o acesso
- `desktop/src/nodes.spec.md` — muda: convidar a partir da aba das máquinas, os convites recebidos, e os
  dispositivos e as pessoas com acesso (retirar)

## Out of Scope

- Contas com palavra-passe próprias do Polvo.
- Partilhar uma sessão sozinha (a partilha é da máquina, e as sessões travadas continuam só para ver).
- A app mobile: tem um plano próprio, que usa esta identidade.

## Revisions

> **IDNTD-R0001:** o dono pediu a 2026-10-18 que cada pessoa tenha um código (nome e apelido, nome de
> utilizador, etc.) e que dar acesso a outra seja convidá-la por esse código.

> **IDNTD-R0002:** o dono aprovou a 2026-10-18 as propostas do agente: o serviço de contas é nosso,
> pequeno, em Go, e fala OIDC/OAuth diretamente com cada fornecedor (era a `IDNTD-D01`); a co2lab aloja-o
> (e o relay do iroh), e a app vem configurada para ele, mas quem quiser aloja o seu e aponta a app para
> lá (`IDNTD-D02`); a conta é opcional (`IDNTD-D03`); os dispositivos da mesma conta são donos uns dos
> outros depois de aprovados (`IDNTD-D04`); os fornecedores da primeira versão são GitHub, Google, Apple,
> Microsoft e o OIDC genérico (`IDNTD-D05`); o código Polvo segue a proposta (`IDNTD-D06`).

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

## Definition of Done

- Uma pessoa entra com a conta do GitHub no desktop e no telemóvel, aprova o telemóvel no desktop, e vê no
  telemóvel as máquinas dela e as sessões de cada uma, de outra rede.
- Convida outra pessoa pelo código Polvo dela para uma máquina, com a permissão de só ver; depois de ela
  aceitar, vê a máquina na lista dela e não consegue escrever nas sessões; retirar o acesso corta-a.
- Sem conta, tudo o que existe hoje continua a funcionar.
