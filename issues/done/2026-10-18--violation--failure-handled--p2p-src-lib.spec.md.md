# VIOLATION: p2p/src/lib.spec.md

- **kind:** violation
- **target (ruled):** p2p/src/lib.spec.md
- **gate:** failure-handled
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

2 declared failure(s) with NO handling path in the code: PTRNS-E01, PTRNS-E02.
A failure catalogued and not handled is a promise the code does not keep: the spec says how the unit fails, and nothing in it deals with that. Treating is not "having a catch" — any path that turns the failure into part of the flow counts, and the project declares the shapes in `handle_patterns`.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
