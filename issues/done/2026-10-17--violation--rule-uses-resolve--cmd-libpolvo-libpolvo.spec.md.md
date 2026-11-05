# VIOLATION: cmd/libpolvo/libpolvo.spec.md

- **kind:** violation
- **target (ruled):** cmd/libpolvo/libpolvo.spec.md
- **gate:** rule-uses-resolve
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 use(s) point at nothing the spec declares: LBPLV-B01 → `char *`. A field must be in one of the spec's tables (data contract, domain, props, signature, state shape); a `DEPn` names a dependency, which the spec no longer declares — cite the field the rule reads. A rule that reads what does not exist validates nothing

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
