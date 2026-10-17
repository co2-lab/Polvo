# VIOLATION: ui/app.spec.md

- **kind:** violation
- **target (ruled):** ui/app.spec.md
- **gate:** rule-uses-declared
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

4 rule(s) do not say what they use: APPUI-M01, APPUI-M02, APPUI-M03, APPUI-M04. Give each a row in `Validations`, `Presentation validations` or `Rule uses` — the fields and the spec's codes it reads —, or `@no-uses: <why>` on the rule's line. Without it, a field changed in the contract points at no rule

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
