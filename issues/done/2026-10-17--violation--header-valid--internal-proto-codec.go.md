# VIOLATION: internal/proto/codec.go

- **kind:** violation
- **target (ruled):** internal/proto/codec.go
- **gate:** header-valid
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

the `@anchors` header has no `code:` of its own — since format 7 every governed file carries its own code (beside the `ref:` of its unit, when it has one), so the dependency chain can name it. `anchors check --fix` generates one from the file's name and type; `anchors code generate <name>` gives one by hand.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
