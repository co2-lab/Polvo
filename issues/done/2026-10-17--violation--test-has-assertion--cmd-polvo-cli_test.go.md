# VIOLATION: cmd/polvo/cli_test.go

- **kind:** violation
- **target (ruled):** cmd/polvo/cli_test.go
- **gate:** test-has-assertion
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

1 test(s) with no assertion in the body — 141: CLIXX-B05: a janela local de 33 por 99 passa a 44 por 111 e a shell vê-o. A test that checks nothing passes whatever the code does: assert the outcome the scenario describes

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
