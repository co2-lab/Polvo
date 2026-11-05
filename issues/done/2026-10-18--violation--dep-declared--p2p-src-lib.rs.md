# VIOLATION: p2p/src/lib.rs

- **kind:** violation
- **target (ruled):** p2p/src/lib.rs
- **gate:** dep-declared
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

1 import(s) of a governed file with no `@dep:` flag: 12 `ffi` → PTFFI. Every import is a dependency, and the map knows only the flagged ones: write `@dep: <code>` on the import line (`anchors check --fix` writes it), or `@no-dep: <reason>`. — the flags and their form: `anchors guide header`

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
