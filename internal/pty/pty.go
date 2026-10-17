// @anchors
//   code: PTIMP
//   ref: PTYXX

// Package pty runs a process inside a virtual terminal: the node's remote shell.
package pty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	cpty "github.com/creack/pty"
)

// Size is a terminal size in character cells.
type Size struct {
	Rows, Cols uint16
}

func (s Size) orDefault() Size {
	if s.Rows == 0 {
		s.Rows = 24
	}
	if s.Cols == 0 {
		s.Cols = 80
	}
	return s
}

// ExitStatus is how the process ended (PTYXX-B04): the exit code, or -1 and the signal's name.
type ExitStatus struct {
	Code   int
	Signal string
}

// Session is a process and the outside of its terminal.
type Session struct {
	cmd    *exec.Cmd
	master *os.File

	// mu orders Resize and Close: both reach the master's descriptor (PTYXX-B06)
	mu     sync.Mutex
	closed bool

	waitOnce sync.Once
	status   ExitStatus
	waitErr  error
}

// Start runs argv inside a new terminal of the given size (PTYXX-B01), in the node's directory and
// environment.
func Start(argv []string, size Size) (*Session, error) {
	return StartIn(argv, "", nil, size)
}

// StartIn runs argv inside a new terminal of the given size, in dir (the node's when empty) and with
// env added to the node's environment, an added name replacing the node's (PTYXX-B07).
func StartIn(argv []string, dir string, env []string, size Size) (*Session, error) {
	// PTYXX-E02: without a program there is nothing to start.
	if len(argv) == 0 {
		return nil, errors.New("pty: no program to start")
	}
	size = size.orDefault()
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	if len(env) > 0 {
		// a later NAME=value wins over an earlier one
		cmd.Env = append(os.Environ(), env...)
	}
	// PTYXX-B01: the process gets the terminal as stdin, stdout, stderr and controlling terminal.
	master, err := cpty.StartWithSize(cmd, &cpty.Winsize{Rows: size.Rows, Cols: size.Cols})
	if err != nil {
		// PTYXX-E01: the error names the program, and nothing is left running.
		return nil, fmt.Errorf("pty: starting %s: %w", argv[0], err)
	}
	return &Session{cmd: cmd, master: master}, nil
}

// Read returns what the process wrote to its terminal (PTYXX-B02).
func (s *Session) Read(p []byte) (int, error) {
	n, err := s.master.Read(p)
	// PTYXX-B05: once the process closed the terminal, Linux answers EIO on the outside; that is
	// the normal end of the stream.
	if err != nil && errors.Is(err, syscall.EIO) {
		return n, io.EOF
	}
	return n, err
}

// Write sends p to the process as if typed (PTYXX-B02).
func (s *Session) Write(p []byte) (int, error) { return s.master.Write(p) }

// Resize changes the size the process sees (PTYXX-B03, PTYXX-I01).
func (s *Session) Resize(size Size) error {
	size = size.orDefault()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return os.ErrClosed
	}
	return cpty.Setsize(s.master, &cpty.Winsize{Rows: size.Rows, Cols: size.Cols})
}

// Signal sends sig to the process.
func (s *Session) Signal(sig os.Signal) error { return s.cmd.Process.Signal(sig) }

// Wait waits for the process to end and returns how it ended (PTYXX-B04).
func (s *Session) Wait() (ExitStatus, error) {
	s.waitOnce.Do(func() {
		err := s.cmd.Wait()
		var ee *exec.ExitError
		switch {
		case err == nil:
			s.status = ExitStatus{Code: 0}
		case errors.As(err, &ee):
			st, _ := ee.Sys().(syscall.WaitStatus)
			if st.Signaled() {
				s.status = ExitStatus{Code: -1, Signal: st.Signal().String()}
			} else {
				s.status = ExitStatus{Code: ee.ExitCode()}
			}
		default:
			s.waitErr = err
		}
	})
	return s.status, s.waitErr
}

// Close hangs up the terminal and releases it (PTYXX-B06). A process that already ended is not
// affected.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.cmd.ProcessState == nil {
		_ = s.cmd.Process.Signal(syscall.SIGHUP)
	}
	return s.master.Close()
}
