//go:build !windows

// @anchors
//   code: CLRSU
//   ref: CLIXX

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/pty"
	"golang.org/x/term"
)

// watchResize sends each change of the local window to the remote shell (CLIXX-B05).
func watchResize(local *os.File, t *hub.Terminal) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				if cols, rows, err := term.GetSize(int(local.Fd())); err == nil {
					t.Resize(pty.Size{Rows: uint16(rows), Cols: uint16(cols)})
				}
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
