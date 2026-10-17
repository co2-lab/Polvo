# VIOLATION: ui/app.ts

- **kind:** violation
- **target (ruled):** ui/app.ts
- **gate:** used-by-declared
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

`App` is imported by APMAN, APTST and carries no `@used-by:` (`anchors check --fix` writes it); `Deps` is imported by APTST and carries no `@used-by:` (`anchors check --fix` writes it); `Editor` is imported by APMAN, APTST and carries no `@used-by:` (`anchors check --fix` writes it); `SocketLike` is imported by APMAN, APTST and carries no `@used-by:` (`anchors check --fix` writes it); `Terminal` is imported by APMAN, APTST and carries no `@used-by:` (`anchors check --fix` writes it); `messages` is imported by APTST and carries no `@used-by:` (`anchors check --fix` writes it); `tokenFromHash` is imported by APTST and carries no `@used-by:` (`anchors check --fix` writes it) — the flags and their form: `anchors guide header`

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
