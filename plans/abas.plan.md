<!-- @anchors
  code: CEACN
  updated_at: 2026-10-17
-->
# Configuração e tipos de aba

> **Code**: `CEACN`
>
> Pedido pelo dono a 2026-10-17. As fases são do agente e estão por rever pelo dono.

## Objective

A janela do Polvo tem um botão de configuração que abre a configuração, com os atalhos de teclado como
primeira secção. Um atalho (e o botão "+") abre uma aba nova, e o utilizador escolhe o tipo dela: um
terminal, a árvore de ficheiros, a informação de git, um editor vazio, ou um ficheiro do computador.

## Rationale

O dono quer a janela a crescer para um ambiente de trabalho: abas de vários tipos na doca, e a
configuração à mão, a começar pelos atalhos. Os tipos de aba começam pela máquina local; o plano do
espelho (ESRMS-W05) estende a árvore e o editor aos ficheiros das outras máquinas.

## Phases

### CEACN-W01 — A configuração e os atalhos

- `desktop/src/settings.spec.md` — nasce: o ficheiro da configuração do utilizador (na pasta de
  configuração do sistema), os valores por omissão, ler e gravar
- `desktop/src/keymap.spec.md` — nasce: as ações da janela e o atalho de cada uma (os de omissão, mudar
  um atalho, um atalho que já é de outra ação)
- `desktop/src/settings_panel.spec.md` — nasce: a aba da configuração, com as secções, e a dos atalhos
  (a lista das ações, carregar nas teclas novas para mudar um)
- `desktop/src/window.spec.md` — muda: o botão de configuração na barra, e as ações da janela pelos
  atalhos

### CEACN-W02 — O escolhedor do tipo de aba (depends on CEACN-W01)

- `desktop/src/new_tab.spec.md` — nasce: o escolhedor que o atalho de aba nova e o botão "+" abrem, com
  os tipos de aba (só o teclado chega para escolher)
- `desktop/src/window.spec.md` — muda: o "+" e o atalho abrem o escolhedor

### CEACN-W03 — Os ficheiros (depends on CEACN-W02)

- `desktop/src/files.spec.md` — nasce: a árvore de ficheiros de uma pasta, que abre um ficheiro no editor
- `desktop/src/editor.spec.md` — nasce: o editor de código do gpui-component (o realce da linguagem),
  vazio ou com um ficheiro, gravar, e a marca de alterado
- `desktop/src/new_tab.spec.md` — muda: abrir um ficheiro do computador pelo diálogo do sistema

### CEACN-W04 — O git (depends on CEACN-W02)

- `desktop/src/git.spec.md` — nasce: a informação de git de uma pasta: o ramo, o estado dos ficheiros e
  os últimos commits

## Out of Scope

- Os ficheiros e o git das outras máquinas: o plano do espelho (ESRMS-W05) estende estas abas.
- Os temas próprios do utilizador.

## Definition of Done

- O botão de configuração abre a configuração; mudar o atalho de aba nova muda-o, e fica gravado.
- O atalho de aba nova abre o escolhedor, e cada tipo abre a aba dele na doca.
- `make test` passa com os testes da janela; `anchors check` passa os gates bloqueantes.
