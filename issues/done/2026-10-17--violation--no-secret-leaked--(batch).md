# VIOLATION: (batch)

- **kind:** violation
- **target (ruled):** (batch)
- **gate:** no-secret-leaked
- **owner:** agente
- **detected on:** 2026-10-17

## Violated invariant

Finding:     input := "[1;3;mREDACTED[0m"
Secret:      [1;3;mREDACTED[0m
RuleID:      jwt
Entropy:     5.249295
File:        app/internal/secrets/secrets_test.go
Line:        264
Commit:      8d75c8b29f00d17160f606b357888668ac88a7e9
Author:      Codeco
Email:       adrielcodeco@hotmail.com
Date:        2026-04-14T02:22:54Z
Fingerprint: 8d75c8b29f00d17160f606b357888668ac88a7e9:app/internal/secrets/secrets_test.go:jwt:264
Link:        https://github.com/co2-lab/Polvo/blob/8d75c8b29f00d17160f60
… (saída truncada)

---
_Recorded by Anchors. Resolve it by moving to `doing/` and, once handled, to `done/`._
