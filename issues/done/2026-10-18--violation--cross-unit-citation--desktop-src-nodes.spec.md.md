# VIOLATION: desktop/src/nodes.spec.md

- **kind:** violation
- **target (ruled):** desktop/src/nodes.spec.md
- **gate:** cross-unit-citation
- **owner:** agente
- **detected on:** 2026-10-18

## Violated invariant

the spec cites 1 rule(s) of other units that live in no product doctrine: `STATN-D01` (108). Citing another unit's rule by its code is right; what two units share lives in the product, which comes first: realize a doctrine rule on the cited rule's line (`@realizes`), so neither spec watches the other

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
