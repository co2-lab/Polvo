# VIOLATION: cmd/libpolvo/libpolvo.spec.md

- **kind:** violation
- **target (ruled):** cmd/libpolvo/libpolvo.spec.md
- **gate:** rule-uses-implemented
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 field(s) a rule says it uses appear in none of the code the spec governs: LBPLV-B01 → `char *`. The rule is orphan of code — it checks nothing the implementation touches. Use the field in the code, or fix what the rule says it uses

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
