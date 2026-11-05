# VIOLATION: internal/station/station.spec.md

- **kind:** violation
- **target (ruled):** internal/station/station.spec.md
- **gate:** rule-uses-implemented
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

2 field(s) a rule says it uses appear in none of the code the spec governs: STATN-B13 → `P2P_PREFIX`; STATN-B15 → `MAX_HISTORY`. The rule is orphan of code — it checks nothing the implementation touches. Use the field in the code, or fix what the rule says it uses

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
