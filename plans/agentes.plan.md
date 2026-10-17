<!-- @anchors
  code: AGPRG
  updated_at: 2026-10-17
-->
# Agentes

> **Code**: `AGPRG`
>
> Pedido pelo dono a 2026-10-17. As fases são do agente e estão por rever pelo dono.

## Objective

Um agente que trabalha num terminal do Polvo (um `claude`, por exemplo) reporta o progresso ao Polvo, e o
Polvo mostra-o na janela: o objetivo, os passos e os últimos estados, de cada agente de cada máquina
ligada. O utilizador responde ao agente pela janela, e o agente lê as mensagens.

## Rationale

O dono quer acompanhar na janela o que os agentes estão a fazer, como o progresso que o anchors grava
num ficheiro, e falar com eles sem sair do Polvo. O canal já existe: o agente usa o `polvo mcp`, e os nós
ligam as máquinas.

## Phases

### AGPRG-W01 — O progresso no nó

O nó da máquina do agente guarda o progresso de cada agente num ficheiro: o objetivo, os passos (por
fazer, em curso, feitos, bloqueados) e o registo dos estados, por ordem de chegada. O controlo do nó dá-o
a quem o pede e avisa quem está à escuta de cada mudança.

- `internal/progress/progress.spec.md` — nasce: o progresso de um agente, as mudanças e o ficheiro
- `internal/node/control.spec.md` — muda: os pedidos do progresso (reportar, ler, escutar)

### AGPRG-W02 — As ferramentas do agente (depends on AGPRG-W01)

- `internal/mcp/tools.spec.md` — muda: começar um trabalho, mudar o estado de um passo, deixar uma nota,
  terminar, e ler as mensagens do utilizador

### AGPRG-W03 — O painel dos agentes (depends on AGPRG-W02)

Depende da janela como cliente dos nós (ESRMS-W02).

- `desktop/src/agents.spec.md` — nasce: o painel da doca com os agentes de todas as máquinas ligadas, o
  progresso de cada um, a lista dos últimos estados, e a caixa para lhe escrever; o indicador na aba da
  sessão onde o agente corre

## Out of Scope

- Pôr agentes a trabalhar a partir da janela (o agente corre-se no terminal, como hoje).
- O histórico de trabalhos já terminados para além do ficheiro de cada agente.

## Definition of Done

- Um `claude` num terminal, com o `polvo mcp`, reporta um trabalho com passos, e a janela mostra o
  progresso e os últimos estados enquanto ele trabalha; o mesmo para um agente noutra máquina.
- Uma mensagem escrita na janela chega ao agente pela ferramenta dele.
- `make test` passa com o `go test`, o `galaxy test` e os testes da janela; `anchors check` passa os
  gates bloqueantes.
