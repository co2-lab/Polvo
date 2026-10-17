# VIOLATION: ui/app.test.ts

- **kind:** violation
- **target (ruled):** ui/app.test.ts
- **gate:** test-has-assertion
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

17 test(s) with no assertion in the body — 84: APPUI-B01: O segredo lido do fragmento sai da barra de endereço; 93: APPUI-B02: O nó acrescentado fica lembrado e escolhê-lo lista a raiz; 103: APPUI-B03: A árvore mostra a pasta de cima, as pastas e os ficheiros por ordem; 113: APPUI-B03: Clicar numa pasta abre-a; 123: APPUI-B04: Abrir um ficheiro põe o texto no editor; 131: APPUI-B05: Alterar põe a marca e gravar escreve, tira a marca e diz gravado; 144: APPUI-B06: O terminal envia a entrada e o tamanho, mostra a saída e o fim; 161: APPUI-B07: A própria máquina vem primeiro e já escolhida, sem ficar guardada; 174: APPUI-M05: O nome da própria máquina; 178: APPUI-E01: Uma chamada que falha mostra o erro na barra; 186: APPUI-E02: Sem o segredo só aparece a mensagem; 193: APPUI-S01: O estado sem segredo esconde a janela; 198: APPUI-S02: O estado pronto mostra a lista dos nós e a barra de estado; 204: APPUI-M01: A mensagem sem segredo; 208: APPUI-M02: A mensagem de erro; 212: APPUI-M03: A mensagem de gravado; 216: APPUI-M04: A mensagem de fim do terminal. A test that checks nothing passes whatever the code does: assert the outcome the scenario describes

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
