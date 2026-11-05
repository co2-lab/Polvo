# VIOLATION: cmd/libpolvo/libpolvo.spec.md

- **kind:** violation
- **target (ruled):** cmd/libpolvo/libpolvo.spec.md
- **gate:** failure-declared
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

the code has handling paths that no declared failure (`-E`) accounts for: 6 found, 0 declared.
Somebody wrote a defence and never declared what it prevents. The defence may be right — what is missing is the spec saying which failure it answers, so whoever reads it knows whether it still applies.
Three ways out: declare the failure as a rule (`CODE-E01`); if a rule of another letter already states it, alias it (`| CODE-E01 | REF[CODE-B05]: why |`); if the matches are normal flow (a lazy map init, a pattern that did not match), close the section with `## Errors` + `none — <why>`.

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
