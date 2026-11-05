# VIOLATION: internal/station/station.go

- **kind:** violation
- **target (ruled):** internal/station/station.go
- **gate:** sibling-guard
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

asymmetry between sibling functions: `Pair` receives `addr` without any guard, while the siblings guard (Add, Listen). Either guard is missing in one that lacks it, or it is unnecessary in ones that have it — both readings cannot be right at same time. If function does not need guard (because it DELEGATES to a sibling that already guards, for example), mark `// @no-guard: <reason>` in its body — waiving is a recorded decision; duplicating guard just to silence gate hides reason.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
