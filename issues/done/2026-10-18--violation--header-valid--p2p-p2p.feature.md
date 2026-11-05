# VIOLATION: p2p/p2p.feature

- **kind:** violation
- **target (ruled):** p2p/p2p.feature
- **gate:** header-valid
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

the code `PTFTR` is also the own code of internal/pty/pty.feature — a code is the address of ONE file. Give this file another (`anchors code generate <name>`), or remove the `code:` line and let `anchors check --fix` generate one.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
