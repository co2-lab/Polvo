// @anchors
//   code: CLIMP
//   ref: CLNTI

// Package hub is the desktop side: the connections to the nodes, their terminals and the control
// calls.
package hub

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/co2-lab/polvo/internal/proto"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sshx"
	"golang.org/x/crypto/ssh"
)

// Config is how the desktop connects.
type Config struct {
	// Identity is the installation's key pair.
	Identity ssh.Signer
	// KnownHosts is the list of the nodes met before.
	KnownHosts *sshx.KnownHosts
	// User is the SSH user name; empty is "polvo".
	User string
}

// ErrClosed ends the calls of a closed connection (CLNTI-B06).
var ErrClosed = errors.New("hub: the connection to the node is closed")

// Node is a connection to a node.
type Node struct {
	client *ssh.Client

	mu     sync.Mutex
	ctrl   *control
	closed bool
}

// Dial connects to the node at addr (CLNTI-B01).
func Dial(addr string, cfg Config) (*Node, error) {
	if err := cfg.check(); err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		// CLNTI-E01: the error names the address.
		return nil, fmt.Errorf("hub: connecting to %s: %w", addr, err)
	}
	return DialConn(conn, addr, cfg)
}

// DialConn runs the SSH handshake over a connection already open to addr.
func DialConn(conn net.Conn, addr string, cfg Config) (*Node, error) {
	if err := cfg.check(); err != nil {
		conn.Close()
		return nil, err
	}
	user := cfg.User
	if user == "" {
		user = "polvo"
	}
	// CLNTI-B01: the installation's identity, and the node's key checked against the known hosts.
	cc, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(cfg.Identity)},
		HostKeyCallback: cfg.KnownHosts.Check,
	})
	if err != nil {
		conn.Close()
		// CLNTI-E02: a refused identity, and the changed key of CLNTI-B01, come back here.
		return nil, fmt.Errorf("hub: connecting to %s: %w", addr, err)
	}
	return &Node{client: ssh.NewClient(cc, chans, reqs)}, nil
}

// check is CLNTI-E02: the identity and the known hosts are required.
func (c Config) check() error {
	if c.Identity == nil {
		return errors.New("hub: no identity")
	}
	if c.KnownHosts == nil {
		return errors.New("hub: no known hosts")
	}
	return nil
}

// Close ends the connection; the calls still waiting fail (CLNTI-B06).
func (n *Node) Close() error {
	n.mu.Lock()
	n.closed = true
	c := n.ctrl
	n.mu.Unlock()
	err := n.client.Close()
	if c != nil {
		c.fail(ErrClosed)
	}
	return err
}

// Terminal is an interactive shell on the node (CLNTI-B02).
type Terminal struct {
	s   *ssh.Session
	in  io.WriteCloser
	out io.Reader
}

// Shell opens the node's shell in a terminal of the given size (CLNTI-B02).
func (n *Node) Shell(size pty.Size) (*Terminal, error) {
	s, err := n.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("hub: opening a session: %w", err)
	}
	if size.Rows == 0 || size.Cols == 0 {
		size = pty.Size{Rows: 24, Cols: 80}
	}
	in, err := s.StdinPipe()
	if err != nil {
		s.Close()
		return nil, err
	}
	out, err := s.StdoutPipe()
	if err != nil {
		s.Close()
		return nil, err
	}
	if err := s.RequestPty("xterm-256color", int(size.Rows), int(size.Cols), ssh.TerminalModes{}); err != nil {
		s.Close()
		return nil, fmt.Errorf("hub: asking for a terminal: %w", err)
	}
	if err := s.Shell(); err != nil {
		s.Close()
		return nil, fmt.Errorf("hub: starting the shell: %w", err)
	}
	return &Terminal{s: s, in: in, out: out}, nil
}

// Read returns what the shell wrote.
func (t *Terminal) Read(p []byte) (int, error) { return t.out.Read(p) }

// Write sends p to the shell as if typed.
func (t *Terminal) Write(p []byte) (int, error) { return t.in.Write(p) }

// Resize changes the size the shell sees.
func (t *Terminal) Resize(size pty.Size) error {
	return t.s.WindowChange(int(size.Rows), int(size.Cols))
}

// Wait waits for the shell to end and returns how it ended.
func (t *Terminal) Wait() (pty.ExitStatus, error) {
	err := t.s.Wait()
	var ee *ssh.ExitError
	switch {
	case err == nil:
		return pty.ExitStatus{}, nil
	case errors.As(err, &ee) && ee.Signal() != "":
		return pty.ExitStatus{Code: -1, Signal: ee.Signal()}, nil
	case errors.As(err, &ee):
		return pty.ExitStatus{Code: ee.ExitStatus()}, nil
	}
	return pty.ExitStatus{}, err
}

// Close ends the terminal session.
func (t *Terminal) Close() error { return t.s.Close() }

// control is the session of the "polvo" subsystem, shared by every call.
type control struct {
	enc *proto.Encoder

	mu      sync.Mutex
	next    int64
	pending map[int64]chan *proto.Message
	err     error
}

func (n *Node) control() (*control, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil, ErrClosed
	}
	if n.ctrl != nil {
		n.ctrl.mu.Lock()
		err := n.ctrl.err
		n.ctrl.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return n.ctrl, nil
	}
	s, err := n.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("hub: opening the control: %w", err)
	}
	in, err := s.StdinPipe()
	if err != nil {
		s.Close()
		return nil, err
	}
	out, err := s.StdoutPipe()
	if err != nil {
		s.Close()
		return nil, err
	}
	if err := s.RequestSubsystem("polvo"); err != nil {
		s.Close()
		return nil, fmt.Errorf("hub: opening the control: %w", err)
	}
	c := &control{enc: proto.NewEncoder(in), pending: map[int64]chan *proto.Message{}}
	go c.read(proto.NewDecoder(out))
	n.ctrl = c
	return c, nil
}

// read hands each response to the call with its id (CLNTI-B03); when the stream ends, every call
// still waiting fails (CLNTI-E03).
func (c *control) read(d *proto.Decoder) {
	for {
		m, err := d.Read()
		if err != nil {
			var re *proto.ReadError
			if errors.As(err, &re) {
				continue
			}
			c.fail(fmt.Errorf("hub: the control stream ended: %w", err))
			return
		}
		id, ok := m.ID.Int()
		if !ok || m.Kind() != proto.Response {
			continue
		}
		c.mu.Lock()
		ch := c.pending[id]
		delete(c.pending, id)
		c.mu.Unlock()
		if ch != nil {
			ch <- m
		}
	}
}

func (c *control) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.err = err
	}
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
}

// Call sends a control request and decodes its result into result, when not nil (CLNTI-B03). An
// error answer comes back as a *proto.Error with the node's code (CLNTI-B04).
func (n *Node) Call(method string, params, result any) error {
	c, err := n.control()
	if err != nil {
		return err
	}
	b, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("hub: encoding the params of %s: %w", method, err)
	}
	ch := make(chan *proto.Message, 1)
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return c.err
	}
	c.next++
	id := c.next
	c.pending[id] = ch
	c.mu.Unlock()
	if err := c.enc.Write(&proto.Message{ID: proto.IntID(id), Method: method, Params: b}); err != nil {
		c.fail(fmt.Errorf("hub: sending %s: %w", method, err))
		return err
	}
	m, ok := <-ch
	if !ok {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.err
	}
	if m.Error != nil {
		return m.Error
	}
	if result != nil {
		if err := json.Unmarshal(m.Result, result); err != nil {
			return fmt.Errorf("hub: decoding the result of %s: %w", method, err)
		}
	}
	return nil
}

// ReadFile returns the bytes of a file on the node (CLNTI-B05).
func (n *Node) ReadFile(path string) ([]byte, error) {
	var r struct {
		Data string `json:"data"`
	}
	if err := n.Call("fs.read", map[string]string{"path": path}, &r); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(r.Data)
}

// WriteFile writes data to a file on the node (CLNTI-B05).
func (n *Node) WriteFile(path string, data []byte) error {
	return n.Call("fs.write", map[string]string{"path": path, "data": base64.StdEncoding.EncodeToString(data)}, nil)
}

// Entry is one entry of a folder on the node.
type Entry struct {
	Name string `json:"name"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
}

// List returns the entries of a folder on the node, by name (CLNTI-B05).
func (n *Node) List(path string) ([]Entry, error) {
	var r struct {
		Entries []Entry `json:"entries"`
	}
	if err := n.Call("fs.list", map[string]string{"path": path}, &r); err != nil {
		return nil, err
	}
	return r.Entries, nil
}

// RunResult is how a process on the node ended and what it wrote.
type RunResult struct {
	Code   int
	Signal string
	Stdout []byte
	Stderr []byte
}

// Run runs argv on the node until it ends (CLNTI-B05).
func (n *Node) Run(argv []string, dir string, stdin []byte) (RunResult, error) {
	var r struct {
		Code   int    `json:"code"`
		Signal string `json:"signal"`
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	err := n.Call("proc.run", map[string]any{"argv": argv, "dir": dir, "stdin": base64.StdEncoding.EncodeToString(stdin)}, &r)
	if err != nil {
		return RunResult{}, err
	}
	out, err := base64.StdEncoding.DecodeString(r.Stdout)
	if err != nil {
		return RunResult{}, err
	}
	errb, err := base64.StdEncoding.DecodeString(r.Stderr)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{Code: r.Code, Signal: r.Signal, Stdout: out, Stderr: errb}, nil
}
