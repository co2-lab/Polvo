# VIOLATION: desktop/src/git.spec.md

- **kind:** violation
- **target (ruled):** desktop/src/git.spec.md
- **gate:** code-cataloged
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

6 symbol(s) exported by `desktop/src/git.rs` that the spec does not catalog: Head (line 27), Group (line 36), Change (line 67), Commit (line 74), Repo (line 83).

A symbol outside the catalog crosses the pipeline without a rule to confront — no one knows what it should do.

Two solutions: catalog it in the spec (the NAME just needs to appear in the text of a rule), or declare that it carries no rule — on the symbol's line, or in the comment directly above it:
    // @no-rule: <why this symbol has no rule>
    export function algo() { … }

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
