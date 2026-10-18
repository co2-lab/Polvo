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
> fases são do agente; o desenho distribuído e os convites são do dono (`IDNTD-R0003`).

## Objective

Uma pessoa entra no Polvo com a conta que já tem (GitHub, Google, Apple, Microsoft, ou o fornecedor OIDC da
empresa) e vê, em qualquer dispositivo, as máquinas dela e as que lhe partilharam, com as sessões de cada
uma. Dar acesso a outra pessoa é convidá-la: o convite leva o nome dela e uma forma de o passar que quem
convida escolhe (um IP de uma das redes e um código curto, um link, um QR code), e a pessoa confirma-o.
Nada disto depende de um servidor central: sem servidores, tudo continua a funcionar.

## Rationale

Hoje a identidade é a chave de cada instalação: partilhar é copiar chaves ou emparelhar com um código, as
permissões do `node.json` nomeiam impressões digitais, e um telemóvel novo tem de ser emparelhado com cada
máquina. Uma conta junta as chaves de uma pessoa: as permissões passam a nomear pessoas, e encontrar as
máquinas é ler a lista da conta. O dono pediu que fosse distribuído: um servidor central seria um ponto
único de falha e de confiança, e nomes únicos globais exigem um árbitro (o triângulo de Zooko).

## Design

- **A conta é uma chave.** O primeiro dispositivo de uma pessoa cria a chave da conta (ed25519). A conta
  assina as chaves dos dispositivos dela; um dispositivo novo só entra aprovado por outro que já lá está
  (como ligar um dispositivo no Signal), com um convite como os de baixo. Os dispositivos da mesma conta são
  donos das máquinas uns dos outros. A lista dos dispositivos da conta, assinada por ela, é publicada pelo
  P2P (o DNS do iroh ou a DHT), e qualquer dispositivo da conta a lê: assim aparecem as máquinas dela.
- **A conta do fornecedor prova quem é a pessoa**, sem servidor nosso: o login (OIDC, com PKCE, no
  browser do sistema; RFC 8252) põe a chave da conta Polvo no pedido, e o fornecedor devolve um token
  assinado por ele que a nomeia (a técnica do OpenPubkey, que o `opkssh` usa para o SSH); qualquer máquina
  confere a assinatura com as chaves públicas do fornecedor. O GitHub não dá tokens que outros possam
  conferir: a prova é um gist público assinado pela conta (como o Keybase), que o Polvo publica sozinho. A
  Apple entra na lista porque a App Store o exige com login social (a regra 4.8). A autenticação forte
  (MFA) é a do fornecedor; o Polvo não guarda palavras-passe.
- **O nome** de cada pessoa é escolhido por ela (sugerido do nome de utilizador do fornecedor, ou do nome e
  do apelido) e não é único: serve para se reconhecer, e quem o lê guarda-o como um contacto. Quem é
  quem decide-se pela chave da conta, nunca pelo nome.
- **Convidar** (dar acesso a uma máquina, ou aprovar um dispositivo da própria conta):
  1. Quem convida escolhe a máquina, a permissão (ver, controlar, encaminhar) e o nome da pessoa.
  2. Escolhe como passar o convite. O Polvo mostra todas as formas, com o que sabe de cada uma: cada IP
     desta máquina com a rede dele (a rede local, a VPN, o IP público) e se é alcançável de fora (o
     diagnóstico do NAT do iroh, e a porta que o router abriu por UPnP/NAT-PMP/PCP), mais um código curto
     (`7-tigre-azul`); o link ou o QR code com o endereço P2P da máquina (a chave e o relay) e um segredo
     longo; ou o código curto sozinho, por um ponto de encontro, quando há um. O utilizador escolhe.
  3. A outra pessoa usa o que recebeu: com o código curto, os dois provam que o conhecem por um PAKE (um
     erro cancela o convite); com o link, pelo segredo. Vê quem convida, a máquina e a permissão, e aceita.
  4. Quem convidou vê quem aceitou (o nome e a impressão digital da conta) e confirma; só então o acesso
     vale. Pode retirá-lo a qualquer momento.
  O convite vale uma hora e uma vez; fica à espera na máquina partilhada.
- **Depois do convite** cada lado guarda a chave do outro, e liga-se pela chave, de qualquer rede: o iroh
  acha onde ela está (o endereço que cada dispositivo publica, assinado) e fura o NAT, ou passa pelo relay.
- **O nó** guarda as permissões por conta (a chave da conta, com o nome como etiqueta) no `node.json`, e
  aceita os dispositivos que a conta assinou.
- **Servidores, todos opcionais e de qualquer um**: os relays do iroh (os públicos, os nossos, os de
  quem quiser), e o ponto de encontro do código curto sozinho. Nenhum vê o conteúdo: o tráfego é P2P e
  cifrado pelo SSH, e o ponto de encontro só liga dois dispositivos durante a hora do convite. Sem
  nenhum, os convites por IP e por link ou QR continuam a funcionar (na mesma rede, ou com uma das
  máquinas alcançável).

## Phases

### IDNTD-W01 — A conta e os convites (depends on ESRMS-W04)

- `internal/account/account.spec.md` — nasce: a chave da conta, os dispositivos assinados por ela, e a
  lista publicada pelo P2P
- `internal/invite/invite.spec.md` — nasce: o convite (a permissão, o nome, a validade, o uso único), as
  formas de o passar (o IP e o código curto com o PAKE, o link e o QR code, o ponto de encontro) e as duas
  confirmações
- `internal/nodeconf/nodeconf.spec.md` — muda: as permissões por conta
- `internal/node/server.spec.md` — muda: os dispositivos de uma conta entram com a permissão dela

### IDNTD-W02 — A prova pelo fornecedor (depends on IDNTD-W01)

- `internal/account/providers.spec.md` — nasce: o login com o Google, a Microsoft, a Apple e o OIDC
  genérico (o token que nomeia a chave da conta), e a prova do GitHub por um gist

### IDNTD-W03 — Na janela (depends on IDNTD-W01)

- `desktop/src/account.spec.md` — nasce: entrar, sair, a prova do fornecedor, e aprovar um dispositivo novo
- `desktop/src/invite.spec.md` — nasce: convidar (escolher a forma: o IP de cada rede com o código, o link,
  o QR code), aceitar um convite, e confirmar quem aceitou
- `desktop/src/nodes.spec.md` — muda: as máquinas da conta e as partilhadas, e as pessoas com acesso a
  cada uma (retirar)

## Out of Scope

- Contas com palavra-passe próprias do Polvo.
- Nomes únicos globais.
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

> **IDNTD-R0003:** o dono pediu a 2026-10-18 que o serviço fosse distribuído, e desenhou os convites: o
> nome não precisa de ser único, porque quem é convidado confirma com o código que quem convidou recebeu;
> e é o utilizador quem escolhe como passar o convite (o IP de que rede, o link, o QR code, ou outra
> forma). Sai o servidor de contas da `IDNTD-R0002`: a conta é uma chave, a prova do fornecedor confere-se
> sem nós, e os servidores que ficam (os relays, o ponto de encontro) são opcionais e de qualquer um. O
> código Polvo único (`IDNTD-D06`) dá lugar ao nome livre.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none

## Definition of Done

- Uma pessoa entra com a conta do GitHub no desktop e no telemóvel, aprova o telemóvel no desktop, e vê no
  telemóvel as máquinas dela e as sessões de cada uma, de outra rede.
- Convida outra pessoa para uma máquina, com a permissão de só ver, por um IP e um código curto, e outra
  por um QR code; depois de elas aceitarem e de quem convidou confirmar, veem a máquina na lista delas e
  não conseguem escrever nas sessões; retirar o acesso corta-as.
- Com todos os servidores desligados, um convite pelo IP na mesma rede funciona do princípio ao fim.
- Sem conta, tudo o que existe hoje continua a funcionar.
