# VIOLATION: internal/proto/codec.spec.md

- **kind:** violation
- **target (ruled):** internal/proto/codec.spec.md
- **gate:** rule-uses-implemented
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

18 field(s) a rule says it uses appear in none of the code the spec governs: CDCOE-B01 → `a linha lida`; CDCOE-B02 → `a linha lida`; CDCOE-B02 → `a mensagem a escrever`; CDCOE-B03 → `a linha lida`; CDCOE-B03 → `a mensagem a escrever`; CDCOE-B04 → `a mensagem a escrever`; CDCOE-B05 → `a linha lida`; CDCOE-B05 → `pedido inválido`; CDCOE-E01 → `a linha lida`; CDCOE-E01 → `erro de parse`; CDCOE-E02 → `a linha lida`; CDCOE-E02 → `pedido inválido`; CDCOE-E03 → `a linha lida`; CDCOE-E03 → `tamanho máximo de uma linha`; CDCOE-E04 → `a linha lida`; CDCOE-E05 → `a mensagem a escrever`; CDCOE-I01 → `a linha lida`; CDCOE-I01 → `a mensagem a escrever`. The rule is orphan of code — it checks nothing the implementation touches. Use the field in the code, or fix what the rule says it uses

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
