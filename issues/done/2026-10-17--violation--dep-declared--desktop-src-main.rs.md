# VIOLATION: desktop/src/main.rs

- **kind:** violation
- **target (ruled):** desktop/src/main.rs
- **gate:** dep-declared
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

2 import(s) of a governed file with no `@dep:` flag: 16 `keys` → KYIMK; 17 `prefs` → PRIMP. Every import is a dependency, and the map knows only the flagged ones: write `@dep: <code>` on the import line (`anchors check --fix` writes it), or `@no-dep: <reason>`. — the flags and their form: `anchors guide header`

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
