# charmbracelet/x/vt v0.1.0, with one fix

A copy of `github.com/charmbracelet/x/vt` v0.1.0 (MIT, see LICENSE), used by the Polvo node through a
`replace` in `go.mod`, with one change:

- `Emulator.closed` is an `atomic.Bool`. In v0.1.0 it is a plain `bool` that `Close` writes while
  `Read` (draining the emulator's answers on another goroutine) reads it: a data race the Go race
  detector reports (`internal/sessions`, SSSNS). The fix belongs upstream; until it is there, this copy.

The tests of the package were left out of the copy.
