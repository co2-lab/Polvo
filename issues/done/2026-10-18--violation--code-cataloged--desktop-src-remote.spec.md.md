# VIOLATION: desktop/src/remote.spec.md

- **kind:** violation
- **target (ruled):** desktop/src/remote.spec.md
- **gate:** code-cataloged
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

3 symbol(s) exported by `desktop/src/remote.rs` that the spec does not catalog: AgentStep (line 93), AgentEntry (line 102), AgentQuestion (line 113).

A symbol outside the catalog crosses the pipeline without a rule to confront — no one knows what it should do.

Two solutions: catalog it in the spec (the NAME just needs to appear in the text of a rule), or declare that it carries no rule — on the symbol's line, or in the comment directly above it:
    // @no-rule: <why this symbol has no rule>
    export function algo() { … }

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
