# VIOLATION: desktop/src/nodes.spec.md

- **kind:** violation
- **target (ruled):** desktop/src/nodes.spec.md
- **gate:** failure-logged
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

2 handled failure(s) that RECORD nothing: MQNAS-E01, MQNAS-E02.
A handling path that swallows the failure without logging is the perfect silence: it happens, nothing knows, and no tool downstream has anything to read. It is the piece that sustains everything else.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
