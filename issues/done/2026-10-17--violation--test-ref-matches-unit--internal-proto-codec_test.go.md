# VIOLATION: internal/proto/codec_test.go

- **kind:** violation
- **target (ruled):** internal/proto/codec_test.go
- **gate:** test-ref-matches-unit
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

the `ref: CDCOE` names internal/proto/codec.go, and the test reaches none of it — no import, no name it defines, no declared invocation. Point the `ref:` at the unit the test exercises, or declare how it calls it (`invocations` on the gate)

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
