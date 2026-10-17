# VIOLATION: ui/app.test.ts

- **kind:** violation
- **target (ruled):** ui/app.test.ts
- **gate:** dep-declared
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 import(s) of a governed file with no `@dep:` flag: 6 `./app` → APIMP. Every import is a dependency, and the map knows only the flagged ones: write `@dep: <code>` on the import line (`anchors check --fix` writes it), or `@no-dep: <reason>`. — the flags and their form: `anchors guide header`

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
