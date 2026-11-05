# VIOLATION: desktop/src/files.rs

- **kind:** violation
- **target (ruled):** desktop/src/files.rs
- **gate:** dep-honored
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

line 11 flags `@dep: RMTEC`, and `remote` resolves to RMIMP — the import changed and the flag stayed (`anchors check --fix` corrects it) — the flags and their form: `anchors guide header`

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
