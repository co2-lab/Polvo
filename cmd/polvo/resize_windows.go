//go:build windows

// @anchors
//   code: CLRSW
//   ref: CLIXX

package main

import (
	"os"

	"github.com/co2-lab/polvo/internal/hub"
)

// watchResize: the Windows console reports window changes by events, not a signal (@TBD: Windows,
// CLIXX-B05).
func watchResize(_ *os.File, _ *hub.Terminal) (stop func()) { return func() {} }
