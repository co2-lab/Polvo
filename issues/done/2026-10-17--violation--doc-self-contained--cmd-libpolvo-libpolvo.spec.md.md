# VIOLATION: cmd/libpolvo/libpolvo.spec.md

- **kind:** violation
- **target (ruled):** cmd/libpolvo/libpolvo.spec.md
- **gate:** doc-self-contained
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 reference(s) that only POINT, without bringing text:
    line 25 — points to `remote_test.rs`: do terminal, e menos um com o erro. @no-scenario: o cgo não corre nos…
  The body of this spec becomes documentation: whoever reads it in `docs/` does not have the repository open, and navigating to pointed file is what documentation exists to eliminate.
  BRING THE TEXT — reference with quoted snippet together is NOT flagged.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
