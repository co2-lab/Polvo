# VIOLATION: internal/proto/codec.spec.md

- **kind:** violation
- **target (ruled):** internal/proto/codec.spec.md
- **gate:** rule-uses-declared
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

3 rule(s) do not say what they use: CDCOE-B01, CDCOE-E01, CDCOE-I01. Give each a row in `Validations`, `Presentation validations` or `Rule uses` — the fields and the spec's codes it reads —, or `@no-uses: <why>` on the rule's line. Without it, a field changed in the contract points at no rule

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
