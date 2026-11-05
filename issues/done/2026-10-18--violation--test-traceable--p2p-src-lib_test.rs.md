# VIOLATION: p2p/src/lib_test.rs

- **kind:** violation
- **target (ruled):** p2p/src/lib_test.rs
- **gate:** test-traceable
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

test does not cite any code from `p2p/src/lib.feature` (PTRNS-B01, PTRNS-B02, PTRNS-B03…).

Without code, test is INVISIBLE to relational gates: `feature-test-match` reports scenarios as not implemented even if this file proves them, and whoever fixes it writes a second test for same behavior. Cite code in test case title or in an adjacent comment (`PTRNS-B01`)

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
