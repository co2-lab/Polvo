<!-- @anchors
  code: NDCNN
  updated_at: 2026-10-17
-->
# Nodeconf — a configuração do Polvo de uma máquina: quem pode o quê nas sessões dela

> **Code**: `NDCNN`

## Overview
Cada máquina decide o que pode fazer quem se liga a ela. A configuração do nó é o ficheiro
`node.json` na pasta de configuração do Polvo dessa máquina, que o `Load` lê para um `Config`: a permissão por omissão
de quem se liga de fora (`guest`) e a de cada chave autorizada (`keys`, pela impressão digital da chave).
O `PermFor` dá a permissão (`sessions.Perm`) de uma chave. A chave da própria instalação é a do dono.

```json
{ "guest": "view", "keys": { "SHA256:abc…": "control" } }
```

## Rules

### NDCNN-B01 — Sem ficheiro, uma chave autorizada controla
Sem o `node.json`, a configuração é a de omissão: quem se liga com uma chave autorizada controla
(`DEFAULT_GUEST`). Autorizar uma chave já é confiar nela; só ver é o que a configuração restringe.

### NDCNN-B02 — A permissão de uma chave
O `PermFor` dá a permissão que o ficheiro dá à impressão digital da chave (`view` ou `control`), e a
`guest` às chaves que o ficheiro não nomeia.

### NDCNN-B03 — A chave da instalação é o dono
A chave da própria instalação (a que o `Load` recebe) tem a permissão de dono, seja qual for o ficheiro:
a janela da própria máquina controla sempre as sessões dela.

## Domain

| Input | Accepts | Outside the domain | Who guarantees |
| --- | --- | --- | --- |
| `node.json` | o objeto com `guest` e `keys`, ou nenhum ficheiro | JSON estragado | este módulo: o erro (`NDCNN-E01`) |
| `guest` | `view` ou `control` | outro valor | este módulo: o erro (`NDCNN-E02`) |
| `keys` | impressões digitais `SHA256:…` com `view` ou `control` | outro valor | este módulo: o erro (`NDCNN-E02`) |

## Errors / Failures
| Rule | Condition | Failure |
| --- | --- | --- |
| `NDCNN-E01` | o `node.json` existe mas não é o objeto da configuração | o `Load` devolve o erro com o ficheiro, e o nó não arranca: arrancar a adivinhar as permissões seria pior |
| `NDCNN-E02` | uma permissão que não é `view` nem `control` | o `Load` devolve o erro com o valor, e o nó não arranca |

## Business Constants
| Constant | Value | Why |
| --- | --- | --- |
| `DEFAULT_GUEST` | `control` | a chave autorizada já é de confiança (como o `polvo connect` e o agente precisam); só ver é escolha de quem configura |

## Rule uses

| Rule | Uses |
| --- | --- |
| `NDCNN-B01` | `node.json`, `DEFAULT_GUEST` |
| `NDCNN-B02` | `guest`, `keys` |
| `NDCNN-B03` | `keys` |
| `NDCNN-E01` | `node.json` |
| `NDCNN-E02` | `guest`, `keys` |

## Revisions

> **NDCNN-R0001:** o `DEFAULT_GUEST` passa de `view` a `control`. Com `view`, a chave do desktop
> autorizada num nó só via, e o `polvo connect`, os terminais remotos e o `polvo mcp` deixavam de abrir
> shells; autorizar uma chave já é confiar nela. Decisão do agente, por rever pelo dono.

## Open Decisions

| Code | Question | Who decides | Becomes |
| --- | --- | --- | --- |

none
