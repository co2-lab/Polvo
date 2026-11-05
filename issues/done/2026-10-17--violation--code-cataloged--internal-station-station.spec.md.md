# VIOLATION: internal/station/station.spec.md

- **kind:** violation
- **target (ruled):** internal/station/station.spec.md
- **gate:** code-cataloged
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 symbol(s) exported by `internal/station/station.go` that the spec does not catalog: PublicKey (line 166).

A symbol outside the catalog crosses the pipeline without a rule to confront — no one knows what it should do.

Two solutions: catalog it in the spec (the NAME just needs to appear in the text of a rule), or declare that it carries no rule — on the symbol's line, or in the comment directly above it:
    // @no-rule: <why this symbol has no rule>
    export function algo() { … }

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
