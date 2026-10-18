# charmbracelet/x/vt v0.1.0, with one fix

A copy of `github.com/charmbracelet/x/vt` v0.1.0 (MIT, see LICENSE), used by the Polvo node through a
`replace` in `go.mod`, with one change:

- `Emulator.closed` is an `atomic.Bool`. In v0.1.0 it is a plain `bool` that `Close` writes while
  `Read` (draining the emulator's answers on another goroutine) reads it: a data race the Go race
  detector reports (`internal/sessions`, SSSNS). The fix belongs upstream; until it is there, this copy.
  Upstream already has it as an open pull request, charmbracelet/x#881 (2026-06-01, the same change);
  when it is merged and tagged, the `replace` goes away. The `SafeEmulator` of charmbracelet/x#850 does
  not cover it: its `Read` takes no lock.

The tests of the package were left out of the copy.
