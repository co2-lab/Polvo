# VIOLATION: p2p/src/lib.spec.md

- **kind:** violation
- **target (ruled):** p2p/src/lib.spec.md
- **gate:** rule-uses-implemented
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

1 field(s) a rule says it uses appear in none of the code the spec governs: PTRNS-B04 → `CLOSE_WAIT`. The rule is orphan of code — it checks nothing the implementation touches. Use the field in the code, or fix what the rule says it uses

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
