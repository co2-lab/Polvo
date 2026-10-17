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
ligada. E o Polvo pergunta ao agente quando quer (o estado, o progresso, outras particularidades da
sessão), e o agente responde-lhe.

## Rationale

O dono quer acompanhar na janela o que os agentes estão a fazer, como o progresso que o anchors grava
num ficheiro, e perguntar-lhes sem sair do Polvo. O canal já existe: o agente usa o `polvo mcp`, e os nós
ligam as máquinas.

O MCP é sobretudo o agente a chamar o Polvo; o pedido no outro sentido (o "sampling") pede uma resposta
ao modelo do cliente fora da conversa do agente, que não sabe do trabalho. Mas o Polvo é dono do terminal
onde o agente corre: escreve a pergunta lá, como mensagem do utilizador com um identificador, e o agente
responde pela ferramenta, ligada ao identificador.

## Phases

### AGPRG-W01 — O progresso no nó

O nó da máquina do agente guarda o progresso de cada agente num ficheiro: o objetivo, os passos (por
fazer, em curso, feitos, bloqueados) e o registo dos estados, por ordem de chegada. O controlo do nó dá-o
a quem o pede e avisa quem está à escuta de cada mudança.

- `internal/progress/progress.spec.md` — nasce: o progresso de um agente, as mudanças e o ficheiro
- `internal/node/control.spec.md` — muda: os pedidos do progresso (reportar, ler, escutar)

### AGPRG-W02 — As ferramentas do agente (depends on AGPRG-W01)

- `internal/mcp/tools.spec.md` — muda: começar um trabalho, mudar o estado de um passo, deixar uma nota,
  terminar, e responder a uma pergunta do Polvo pelo identificador dela

### AGPRG-W03 — Perguntar ao agente (depends on AGPRG-W02)

Depende das sessões no nó (ESRMS-W01): a pergunta é escrita na sessão do agente.

- `internal/progress/progress.spec.md` — muda: as perguntas do Polvo a um agente, com o identificador, e
  as respostas; uma pergunta sem resposta fica pendente
- `internal/node/control.spec.md` — muda: perguntar a um agente (escreve a pergunta na sessão dele)

### AGPRG-W04 — O painel dos agentes (depends on AGPRG-W03)

Depende da janela como cliente dos nós (ESRMS-W02).

- `desktop/src/agents.spec.md` — nasce: o painel da doca com os agentes de todas as máquinas ligadas, o
  progresso de cada um, a lista dos últimos estados, as perguntas feitas e as respostas, e o botão de
  perguntar (as perguntas comuns já feitas, e uma livre); o indicador na aba da sessão onde o agente corre

## Out of Scope

- Pôr agentes a trabalhar a partir da janela (o agente corre-se no terminal, como hoje).
- O histórico de trabalhos já terminados para além do ficheiro de cada agente.

## Revisions

> **AGPRG-R0001:** a caixa de mensagens dá lugar às perguntas do Polvo ao agente, como o dono a explicou
> a 2026-10-17: o Polvo pergunta (o estado, o progresso, as particularidades da sessão) e o agente
> responde. Entra a fase W03, e o painel passa a W04.

## Definition of Done

- Um `claude` num terminal, com o `polvo mcp`, reporta um trabalho com passos, e a janela mostra o
  progresso e os últimos estados enquanto ele trabalha; o mesmo para um agente noutra máquina.
- Uma pergunta feita na janela chega ao agente na sessão dele, e a resposta, pela ferramenta, aparece no
  painel junto da pergunta.
- `make test` passa com o `go test`, o `galaxy test` e os testes da janela; `anchors check` passa os
  gates bloqueantes.
