// @anchors
//   code: SSIMS
//   ref: SSSNS

// Package sessions keeps the terminal sessions of a node: programs in a pty that live apart from
// whoever watches them, with any number of viewers.
package sessions

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"

	"github.com/co2-lab/polvo/internal/pty"
)

// ErrNoSession is the error of an id no live session has (SSSNS-E02).
var ErrNoSession = errors.New("sessions: no such session")

// ErrViewOnly refuses a viewer that may only watch (SSSNS-E03).
var ErrViewOnly = errors.New("sessions: this viewer may only watch")

// ErrTooSlow ends a viewer that fell behind (SSSNS-E04).
var ErrTooSlow = errors.New("sessions: the viewer fell too far behind")

// VIEWER_BACKLOG is what a viewer may have unread before it is let go (SSSNS-E04).
const VIEWER_BACKLOG = 4 << 20

// Perm is what a viewer may do.
type Perm int

// Pairing may only pair: the key the node does not know yet (SSSNS-B06, below View).
const Pairing Perm = -1

const (
	// View watches.
	View Perm = iota
	// Control watches, types and resizes, unless the session is locked.
	Control
	// Owner controls even a locked session: the machine's own window.
	Owner
)

// Info describes a live session.
type Info struct {
	ID      string
	Argv    []string
	Size    pty.Size
	Viewers int
	Locked  bool
	Created time.Time
}

// Manager keeps the sessions of a node.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	order    []string
	backlog  int
	env      []string
}

// NewManager makes a manager with no sessions, whose viewers may fall VIEWER_BACKLOG behind.
func NewManager() *Manager { return &Manager{sessions: map[string]*session{}, backlog: VIEWER_BACKLOG} }

// SetBacklog sets how far behind a viewer may fall before it is let go (SSSNS-E04).
func (m *Manager) SetBacklog(n int) {
	m.mu.Lock()
	m.backlog = n
	m.mu.Unlock()
}

// SetEnv sets what every new session gets in its environment, as NAME=value (SSSNS-B11).
func (m *Manager) SetEnv(env []string) {
	m.mu.Lock()
	m.env = append([]string(nil), env...)
	m.mu.Unlock()
}

// Create runs argv in a pty of size, in dir, and keeps it as a session (SSSNS-B01), with its id and
// the manager's environment (SSSNS-B11).
func (m *Manager) Create(argv []string, dir string, size pty.Size) (Info, error) {
	m.mu.Lock()
	var id string
	for {
		id = newID()
		if _, taken := m.sessions[id]; !taken {
			break
		}
	}
	// the id is kept while the program starts, for no other session to take it
	m.sessions[id] = nil
	env := append([]string{"TERM=xterm-256color", "POLVO_SESSION=" + id}, m.env...)
	m.mu.Unlock()
	t, err := pty.StartIn(argv, dir, env, size)
	if err != nil {
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		// SSSNS-E01: the pty's error names the program.
		return Info{}, fmt.Errorf("sessions: %w", err)
	}
	s := &session{
		id:      id,
		argv:    append([]string(nil), argv...),
		created: time.Now(),
		t:       t,
		scr:     newScreen(size),
		size:    size,
		viewers: map[*Viewer]struct{}{},
		m:       m,
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.order = append(m.order, id)
	m.mu.Unlock()
	go s.pump()
	return s.info(), nil
}

// List gives the live sessions in the order they were born (SSSNS-B02).
func (m *Manager) List() []Info {
	m.mu.Lock()
	ss := make([]*session, 0, len(m.order))
	for _, id := range m.order {
		ss = append(ss, m.sessions[id])
	}
	m.mu.Unlock()
	out := make([]Info, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.info())
	}
	return out
}

func (m *Manager) get(id string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[id]
	if s == nil {
		// SSSNS-E02: no session, or one whose program is still starting
		return nil, ErrNoSession
	}
	return s, nil
}

func (m *Manager) forget(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	for i, o := range m.order {
		if o == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
}

// Attach makes a viewer of the session that reads its current screen and then its output (SSSNS-B03).
func (m *Manager) Attach(id string, perm Perm) (*Viewer, error) {
	s, err := m.get(id)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	backlog := m.backlog
	m.mu.Unlock()
	v := &Viewer{s: s, perm: perm, backlog: backlog}
	v.cond = sync.NewCond(&v.mu)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return nil, ErrNoSession
	}
	// the screen and the subscription in one step: nothing lost or repeated between them
	v.queue = s.scr.snapshot()
	s.viewers[v] = struct{}{}
	return v, nil
}

// Lock makes the session view-only for all but its owner, or lifts that (SSSNS-B07).
func (m *Manager) Lock(id string, locked bool) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.locked = locked
	s.mu.Unlock()
	return nil
}

// Close ends the session's program (SSSNS-B09).
func (m *Manager) Close(id string) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	return s.t.Close()
}

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type session struct {
	id      string
	argv    []string
	created time.Time
	t       *pty.Session
	m       *Manager

	mu      sync.Mutex
	scr     *screen
	size    pty.Size
	viewers map[*Viewer]struct{}
	locked  bool
	done    bool
}

func (s *session) info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Info{ID: s.id, Argv: s.argv, Size: s.size, Viewers: len(s.viewers), Locked: s.locked, Created: s.created}
}

// pump takes the program's output to the screen and to every viewer (SSSNS-B04), and ends the
// session with the program (SSSNS-B08).
func (s *session) pump() {
	buf := make([]byte, 32<<10)
	for {
		n, err := s.t.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.scr.write(chunk)
			for v := range s.viewers {
				if !v.push(chunk) {
					// SSSNS-E04
					delete(s.viewers, v)
				}
			}
			s.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
	st, _ := s.t.Wait()
	s.t.Close()
	s.mu.Lock()
	s.done = true
	for v := range s.viewers {
		v.finish(st)
	}
	s.viewers = map[*Viewer]struct{}{}
	s.scr.close()
	s.mu.Unlock()
	s.m.forget(s.id)
}

// may says whether perm may type and resize now (SSSNS-B05, SSSNS-B06, SSSNS-B07).
func (s *session) may(perm Perm) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return perm == Owner || perm == Control && !s.locked
}

// Viewer watches a session, and with the permission types in it and resizes it.
type Viewer struct {
	s       *session
	perm    Perm
	backlog int

	mu    sync.Mutex
	cond  *sync.Cond
	queue []byte
	err   error
	exit  pty.ExitStatus
	ended bool
}

// push queues output for the viewer; false when it fell too far behind and is let go (SSSNS-E04).
func (v *Viewer) push(p []byte) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return false
	}
	if len(v.queue)+len(p) > v.backlog {
		v.queue, v.err = nil, ErrTooSlow
		v.cond.Broadcast()
		return false
	}
	v.queue = append(v.queue, p...)
	v.cond.Broadcast()
	return true
}

// finish tells the viewer the program ended, after the output it still has to read (SSSNS-B08).
func (v *Viewer) finish(st pty.ExitStatus) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.exit, v.ended = st, true
	if v.err == nil {
		v.err = io.EOF
	}
	v.cond.Broadcast()
}

// Read reads the screen the session had when the viewer came, then its output; io.EOF after the
// program ended, ErrTooSlow when the viewer fell behind.
func (v *Viewer) Read(p []byte) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for len(v.queue) == 0 && v.err == nil {
		v.cond.Wait()
	}
	if len(v.queue) > 0 {
		n := copy(p, v.queue)
		v.queue = v.queue[n:]
		return n, nil
	}
	return 0, v.err
}

// Write types p in the session (SSSNS-B05); ErrViewOnly when the viewer may only watch (SSSNS-E03).
func (v *Viewer) Write(p []byte) (int, error) {
	if !v.s.may(v.perm) {
		return 0, ErrViewOnly
	}
	return v.s.t.Write(p)
}

// Resize makes size the session's size (SSSNS-B05); ErrViewOnly when the viewer may only watch.
func (v *Viewer) Resize(size pty.Size) error {
	if !v.s.may(v.perm) {
		return ErrViewOnly
	}
	s := v.s
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.t.Resize(size); err != nil {
		return err
	}
	s.size = size
	s.scr.resize(size)
	return nil
}

// Exit is how the program ended, once it did.
func (v *Viewer) Exit() (pty.ExitStatus, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.exit, v.ended
}

// Close stops watching; the session goes on.
func (v *Viewer) Close() error {
	s := v.s
	s.mu.Lock()
	delete(s.viewers, v)
	s.mu.Unlock()
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err == nil {
		v.err = io.ErrClosedPipe
	}
	v.cond.Broadcast()
	return nil
}

// screen follows the program's output in a terminal emulator on the node, to rewrite the current
// screen for a viewer that comes later (SSSNS-B03).
type screen struct {
	emu *vt.Emulator
	alt bool
}

func newScreen(size pty.Size) *screen {
	rows, cols := int(size.Rows), int(size.Cols)
	if rows == 0 || cols == 0 {
		rows, cols = 24, 80
	}
	s := &screen{emu: vt.NewEmulator(cols, rows)}
	s.emu.SetCallbacks(vt.Callbacks{AltScreen: func(on bool) { s.alt = on }})
	// SSSNS-B10: the emulator's answers to the program go nowhere: each viewer's terminal answers.
	go io.Copy(io.Discard, s.emu)
	return s
}

func (s *screen) write(p []byte) { s.emu.Write(p) }

func (s *screen) resize(size pty.Size) { s.emu.Resize(int(size.Cols), int(size.Rows)) }

func (s *screen) close() { s.emu.Close() }

// snapshot is the current screen as VT: the screen it is on, the text with its styles, and the cursor.
func (s *screen) snapshot() []byte {
	var b bytes.Buffer
	if s.alt {
		b.WriteString("\x1b[?1049h")
	}
	b.WriteString("\x1b[0m\x1b[H\x1b[2J")
	b.WriteString(strings.ReplaceAll(s.emu.Render(), "\n", "\r\n"))
	b.WriteString("\x1b[0m")
	pos := s.emu.CursorPosition()
	fmt.Fprintf(&b, "\x1b[%d;%dH", pos.Y+1, pos.X+1)
	if s.emu.CursorHidden() {
		b.WriteString("\x1b[?25l")
	} else {
		b.WriteString("\x1b[?25h")
	}
	return b.Bytes()
}
