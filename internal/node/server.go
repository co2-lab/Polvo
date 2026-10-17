// @anchors
//   code: SRIMP
//   ref: SRVRE

// Package node is `polvo node`: the SSH server on a remote machine, with its terminal sessions,
// its commands and the Polvo control subsystem.
package node

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"sync"
	"syscall"

	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sshx"
	"golang.org/x/crypto/ssh"
)

// Config is what a node serves with.
type Config struct {
	// HostKey is the node's identity.
	HostKey ssh.Signer
	// Authorized is the list of keys that may enter (SRVRE-B01).
	Authorized *sshx.AuthorizedKeys
	// Shell is the program and arguments of the node's shell; empty is /bin/sh.
	Shell []string
	// Control receives the sessions of the "polvo" subsystem (SRVRE-B07); nil refuses them.
	Control func(io.ReadWriteCloser)
}

func (c Config) shell() []string {
	if len(c.Shell) == 0 {
		return []string{"/bin/sh"}
	}
	return c.Shell
}

func (c Config) server() (*ssh.ServerConfig, error) {
	// SRVRE-E01: without the identity or the list nobody can be served.
	if c.HostKey == nil {
		return nil, errors.New("node: no host key")
	}
	if c.Authorized == nil {
		return nil, errors.New("node: no authorized keys")
	}
	sc := &ssh.ServerConfig{
		// SRVRE-B01: only a public key in the list enters; no password nor keyboard-interactive
		// callback is set, so those methods are refused.
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			ok, err := c.Authorized.Allows(key)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, errors.New("node: key not authorized")
			}
			return nil, nil
		},
	}
	sc.AddHostKey(c.HostKey)
	return sc, nil
}

// Serve accepts connections on l until it is closed.
func Serve(l net.Listener, cfg Config) error {
	sc, err := cfg.server()
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go serveConn(conn, cfg, sc)
	}
}

// ServeConn serves one connection already open.
func ServeConn(conn net.Conn, cfg Config) error {
	sc, err := cfg.server()
	if err != nil {
		conn.Close()
		return err
	}
	return serveConn(conn, cfg, sc)
}

func serveConn(conn net.Conn, cfg Config, sc *ssh.ServerConfig) error {
	sconn, chans, reqs, err := ssh.NewServerConn(conn, sc)
	if err != nil {
		return err
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		// SRVRE-B08: only sessions are served; port forwarding is refused.
		if nc.ChannelType() != "session" {
			nc.Reject(ssh.UnknownChannelType, "node: unknown channel type")
			continue
		}
		ch, creqs, err := nc.Accept()
		if err != nil {
			continue
		}
		s := &session{cfg: cfg, ch: ch}
		go s.serve(creqs)
	}
	return nil
}

type session struct {
	cfg Config
	ch  ssh.Channel

	mu      sync.Mutex
	started bool
	ptyReq  *pty.Size
	term    *pty.Session
}

type ptyRequest struct {
	Term          string
	Cols, Rows    uint32
	Width, Height uint32
	Modes         string
}

type windowChange struct {
	Cols, Rows    uint32
	Width, Height uint32
}

type commandRequest struct{ Command string }

func (s *session) serve(reqs <-chan *ssh.Request) {
	for r := range reqs {
		ok := false
		switch r.Type {
		case "pty-req":
			var p ptyRequest
			if ssh.Unmarshal(r.Payload, &p) == nil {
				s.mu.Lock()
				s.ptyReq = &pty.Size{Rows: uint16(p.Rows), Cols: uint16(p.Cols)}
				s.mu.Unlock()
				ok = true
			}
		case "window-change":
			// SRVRE-B03: the new size reaches the running terminal.
			var w windowChange
			if ssh.Unmarshal(r.Payload, &w) == nil {
				size := pty.Size{Rows: uint16(w.Rows), Cols: uint16(w.Cols)}
				s.mu.Lock()
				if s.term != nil {
					ok = s.term.Resize(size) == nil
				} else if s.ptyReq != nil {
					*s.ptyReq, ok = size, true
				}
				s.mu.Unlock()
			}
		case "shell":
			ok = s.start(s.cfg.shell())
		case "exec":
			var c commandRequest
			if ssh.Unmarshal(r.Payload, &c) == nil {
				ok = s.start(append(append([]string{}, s.cfg.shell()...), "-c", c.Command))
			}
		case "subsystem":
			// SRVRE-B07: the "polvo" subsystem is handed to the control; any other is refused.
			var c commandRequest
			if ssh.Unmarshal(r.Payload, &c) == nil && c.Command == "polvo" && s.cfg.Control != nil && s.claim() {
				ok = true
				go s.cfg.Control(s.ch)
			}
		}
		// SRVRE-B08: an unknown request is refused and the session goes on.
		if r.WantReply {
			r.Reply(ok, nil)
		}
	}
}

// claim marks the session's one program as started.
func (s *session) claim() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return false
	}
	s.started = true
	return true
}

func (s *session) start(argv []string) bool {
	if !s.claim() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptyReq == nil {
		go s.runCommand(argv)
		return true
	}
	// the terminal is born before the reply, so a window change that follows finds it
	t, err := pty.Start(argv, *s.ptyReq)
	if err != nil {
		go s.fail(err)
		return true
	}
	s.term = t
	go s.runTerminal(t)
	return true
}

// runTerminal serves a program running in a terminal (SRVRE-B02, SRVRE-B05).
func (s *session) runTerminal(t *pty.Session) {
	go io.Copy(t, s.ch)
	io.Copy(s.ch, t)
	st, err := t.Wait()
	t.Close()
	if err != nil {
		s.fail(err)
		return
	}
	s.exit(st.Code, st.Signal)
}

// runCommand runs argv with no terminal, output and error apart (SRVRE-B04).
func (s *session) runCommand(argv []string) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = s.ch, s.ch.Stderr()
	in, err := cmd.StdinPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		s.fail(err)
		return
	}
	go func() {
		io.Copy(in, s.ch)
		in.Close()
	}()
	err = cmd.Wait()
	var ee *exec.ExitError
	switch {
	case err == nil:
		s.exit(0, "")
	case errors.As(err, &ee):
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			s.exit(-1, ws.Signal().String())
		} else {
			s.exit(ee.ExitCode(), "")
		}
	default:
		s.fail(err)
	}
}

// fail ends a session whose program did not start: 127 and the message on stderr (SRVRE-E02).
func (s *session) fail(err error) {
	fmt.Fprintf(s.ch.Stderr(), "polvo node: %v\n", err)
	s.exit(127, "")
}

type exitStatus struct{ Status uint32 }

type exitSignal struct {
	Signal     string
	CoreDumped bool
	Message    string
	Lang       string
}

// exit sends how the program ended and closes the session (SRVRE-B06). signal is the name Go
// gives the signal ("killed"); SSH carries the short name ("KILL").
func (s *session) exit(code int, signal string) {
	if signal != "" {
		s.ch.SendRequest("exit-signal", false, ssh.Marshal(exitSignal{Signal: signalName(signal)}))
	} else {
		s.ch.SendRequest("exit-status", false, ssh.Marshal(exitStatus{Status: uint32(code)}))
	}
	s.ch.Close()
}

// the SSH names of the signals (RFC 4254 6.10), by the text Go gives each one
var signalNames = func() map[string]string {
	m := map[string]string{}
	for sig, name := range map[syscall.Signal]string{
		syscall.SIGABRT: "ABRT", syscall.SIGALRM: "ALRM", syscall.SIGFPE: "FPE", syscall.SIGHUP: "HUP",
		syscall.SIGILL: "ILL", syscall.SIGINT: "INT", syscall.SIGKILL: "KILL", syscall.SIGPIPE: "PIPE",
		syscall.SIGQUIT: "QUIT", syscall.SIGSEGV: "SEGV", syscall.SIGTERM: "TERM",
	} {
		m[sig.String()] = name
	}
	return m
}()

func signalName(goName string) string {
	if n, ok := signalNames[goName]; ok {
		return n
	}
	return goName
}
