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
> fases são do agente e estão por aprovar pelo dono (as `Open Decisions`).

## Objective

Uma pessoa entra no Polvo com a conta que já tem (GitHub, Google, Apple, Microsoft, ou o fornecedor OIDC da
empresa) e vê, em qualquer dispositivo, as máquinas dela e as que lhe partilharam, com as sessões de cada
uma. Partilha uma máquina com outra pessoa pela conta dela (ver, controlar, encaminhar sockets) em vez de
trocar chaves. Sem conta, o Polvo continua a funcionar como hoje (emparelhar por código).

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

- `conta/` (Go) — nasce: o servidor; o login pelos fornecedores, as contas, os dispositivos, a aprovação de
  um dispositivo novo, e os certificados SSH
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

- `conta/shares.spec.md` — nasce: partilhar uma máquina com uma pessoa (ver, controlar, encaminhar), e
  deixar de partilhar
- `desktop/src/nodes.spec.md` — muda: partilhar a partir da aba das máquinas, e os dispositivos com acesso
  (revogar)

## Out of Scope

- Contas com palavra-passe próprias do Polvo.
- Partilhar uma sessão sozinha (a partilha é da máquina, e as sessões travadas continuam só para ver).
- A app mobile: tem um plano próprio, que usa esta identidade.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |
| `IDNTD-D01` | O serviço de contas: um servidor nosso, pequeno, em Go, que fala OIDC/OAuth diretamente com cada fornecedor (proposta do agente: o que precisamos é saber quem é a pessoa, e as partilhas são o nosso modelo; o MFA é o do fornecedor), ou um fornecedor de identidade completo e alojável (Zitadel em Go, com os logins sociais e as organizações prontos; Keycloak, maduro mas pesado, 2 GB de memória)? @no-code: decisão por tomar pelo dono | dono | a fase IDNTD-W01 |
| `IDNTD-D02` | A co2lab aloja o serviço (e o relay do iroh) por omissão, e quem quiser aloja o seu? Proposta do agente: sim. @no-code: decisão por tomar pelo dono | dono | a fase IDNTD-W01 |
| `IDNTD-D03` | A conta é opcional (sem ela, o emparelhamento por código continua)? Proposta do agente: sim. @no-code: decisão por tomar pelo dono | dono | as fases IDNTD-W02 e IDNTD-W03 |
| `IDNTD-D04` | Os dispositivos da mesma conta são o dono das máquinas uns dos outros, depois de aprovados por um que já lá está? Proposta do agente: sim. @no-code: decisão por tomar pelo dono | dono | a fase IDNTD-W02 |
| `IDNTD-D05` | Os fornecedores da primeira versão: GitHub, Google, Apple, Microsoft e o OIDC genérico (o Apple é obrigatório na App Store quando há login social). Mais algum (GitLab, Discord)? @no-code: decisão por tomar pelo dono | dono | a fase IDNTD-W01 |

## Definition of Done

- Uma pessoa entra com a conta do GitHub no desktop e no telemóvel, aprova o telemóvel no desktop, e vê no
  telemóvel as máquinas dela e as sessões de cada uma, de outra rede.
- Partilha uma máquina com outra pessoa pela conta dela, com a permissão de só ver; essa pessoa vê a máquina
  na lista dela e não consegue escrever nas sessões; deixar de partilhar corta-a.
- Sem conta, tudo o que existe hoje continua a funcionar.
