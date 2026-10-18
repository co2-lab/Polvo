// @anchors
//   code: BRIMP
//   ref: BRDGE

// Package bridge is the API the C library gives the window: stations and terminals by number.
package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sshx"
	"github.com/co2-lab/polvo/internal/station"
)

// Handle is the number of an open station or terminal.
type Handle int64

type terminal struct {
	t     *hub.Terminal
	owner Handle
}

var (
	mu        sync.Mutex
	last      Handle
	stations  = map[Handle]*station.Station{}
	terminals = map[Handle]*terminal{}
)

func next() Handle {
	last++
	return last
}

// Open opens the station of dir, listening on listen (empty: not) with shell for new sessions, and
// gives its number (BRDGE-B01).
func Open(dir, listen, shell string) (Handle, error) {
	opt := station.Options{Listen: listen}
	if shell != "" {
		opt.Shell = []string{shell}
	}
	s, err := station.Open(dir, opt)
	if err != nil {
		return 0, err
	}
	mu.Lock()
	defer mu.Unlock()
	h := next()
	stations[h] = s
	return h, nil
}

func errHandle(h Handle) error { return fmt.Errorf("bridge: handle %d is not open", h) }

func stationOf(h Handle) (*station.Station, error) {
	mu.Lock()
	defer mu.Unlock()
	s := stations[h]
	if s == nil {
		// BRDGE-E01
		return nil, errHandle(h)
	}
	return s, nil
}

func terminalOf(h Handle) (*hub.Terminal, error) {
	mu.Lock()
	defer mu.Unlock()
	t := terminals[h]
	if t == nil {
		// BRDGE-E01
		return nil, errHandle(h)
	}
	return t.t, nil
}

// Close closes the station and its terminals (BRDGE-B01).
func Close(h Handle) {
	mu.Lock()
	s := stations[h]
	delete(stations, h)
	var ts []*hub.Terminal
	for k, t := range terminals {
		if t.owner == h {
			ts = append(ts, t.t)
			delete(terminals, k)
		}
	}
	mu.Unlock()
	for _, t := range ts {
		t.Close()
	}
	if s != nil {
		s.Close()
	}
}

type request struct {
	Op      string   `json:"op"`
	Addr    string   `json:"addr"`
	Name    string   `json:"name"`
	Machine string   `json:"machine"`
	ID      string   `json:"id"`
	Argv    []string `json:"argv"`
	Dir     string   `json:"dir"`
	Rows    uint16   `json:"rows"`
	Cols    uint16   `json:"cols"`
	Locked  bool     `json:"locked"`
}

// Call answers a request in JSON about the station (BRDGE-B02).
func Call(h Handle, req []byte) ([]byte, error) {
	s, err := stationOf(h)
	if err != nil {
		return nil, err
	}
	var r request
	if err := json.Unmarshal(req, &r); err != nil {
		// BRDGE-E02
		return nil, fmt.Errorf("bridge: the request is not JSON: %w", err)
	}
	res, err := answer(s, r)
	if err != nil {
		// BRDGE-E03 and BRDGE-E02
		return nil, err
	}
	return json.Marshal(res)
}

func answer(s *station.Station, r request) (any, error) {
	switch r.Op {
	case "status":
		out := map[string]string{"listen": s.ListenAddr(), "key": sshx.AuthorizedLine(s.PublicKey(), "polvo")}
		if err := s.ListenError(); err != nil {
			out["listen_error"] = err.Error()
		}
		return out, nil
	case "machines":
		type m struct {
			Name  string `json:"name"`
			Local bool   `json:"local"`
		}
		out := []m{}
		for _, x := range s.Machines() {
			out = append(out, m{x.Name, x.Local})
		}
		return map[string]any{"machines": out}, nil
	case "add":
		return struct{}{}, s.Add(r.Addr)
	case "remove":
		return struct{}{}, s.Remove(r.Name)
	}
	switch r.Op {
	case "sessions", "new", "close", "lock":
	default:
		// BRDGE-E02
		return nil, fmt.Errorf("bridge: unknown op %q", r.Op)
	}
	n, err := s.Node(r.Machine)
	if err != nil {
		return nil, err
	}
	switch r.Op {
	case "sessions":
		list, err := n.Sessions()
		if err != nil {
			return nil, err
		}
		if list == nil {
			list = []hub.Session{}
		}
		return map[string]any{"sessions": list}, nil
	case "new":
		id, err := n.NewSession(r.Argv, r.Dir, pty.Size{Rows: r.Rows, Cols: r.Cols})
		return map[string]string{"id": id}, err
	case "close":
		return struct{}{}, n.CloseSession(r.ID)
	default:
		return struct{}{}, n.LockSession(r.ID, r.Locked)
	}
}

// Attach connects a terminal to a session of a machine of the station and gives its number (BRDGE-B03).
func Attach(h Handle, machine, id string) (Handle, error) {
	s, err := stationOf(h)
	if err != nil {
		return 0, err
	}
	n, err := s.Node(machine)
	if err != nil {
		return 0, err
	}
	t, err := n.Attach(id)
	if err != nil {
		return 0, err
	}
	mu.Lock()
	defer mu.Unlock()
	th := next()
	terminals[th] = &terminal{t: t, owner: h}
	return th, nil
}

// Read gives what the session shows, as it comes; zero bytes and no error at the end (BRDGE-B03).
func Read(h Handle, p []byte) (int, error) {
	t, err := terminalOf(h)
	if err != nil {
		return 0, err
	}
	n, err := t.Read(p)
	if n > 0 {
		return n, nil
	}
	if errors.Is(err, io.EOF) || err != nil && strings.Contains(err.Error(), "closed") {
		return 0, nil
	}
	return n, err
}

// Write types p in the session (BRDGE-B03).
func Write(h Handle, p []byte) (int, error) {
	t, err := terminalOf(h)
	if err != nil {
		return 0, err
	}
	return t.Write(p)
}

// Resize changes the session's size (BRDGE-B03).
func Resize(h Handle, rows, cols uint16) error {
	t, err := terminalOf(h)
	if err != nil {
		return err
	}
	return t.Resize(pty.Size{Rows: rows, Cols: cols})
}

// Detach lets go of the terminal; the session goes on (BRDGE-B03).
func Detach(h Handle) {
	mu.Lock()
	t := terminals[h]
	delete(terminals, h)
	mu.Unlock()
	if t != nil {
		t.t.Close()
	}
}
