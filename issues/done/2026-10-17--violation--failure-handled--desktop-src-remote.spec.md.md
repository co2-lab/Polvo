# VIOLATION: desktop/src/remote.spec.md

- **kind:** violation
- **target (ruled):** desktop/src/remote.spec.md
- **gate:** failure-handled
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 declared failure(s) with NO handling path in the code: RMTEC-E01.
A failure catalogued and not handled is a promise the code does not keep: the spec says how the unit fails, and nothing in it deals with that. Treating is not "having a catch" — any path that turns the failure into part of the flow counts, and the project declares the shapes in `handle_patterns`.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
