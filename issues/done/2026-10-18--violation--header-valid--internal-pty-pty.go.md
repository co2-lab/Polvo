# VIOLATION: internal/pty/pty.go

- **kind:** violation
- **target (ruled):** internal/pty/pty.go
- **gate:** header-valid
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

the code `PTIMP` is also the own code of p2p/src/lib.rs — a code is the address of ONE file. Give this file another (`anchors code generate <name>`), or remove the `code:` line and let `anchors check --fix` generate one.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
