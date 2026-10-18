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
	"time"
	"unicode/utf8"

	"github.com/co2-lab/polvo/internal/discovery"
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
	return openWith(dir, listen, shell, &discovery.Options{})
}

// openWith opens the station announcing itself, and asking, where disc says.
func openWith(dir, listen, shell string, disc *discovery.Options) (Handle, error) {
	opt := station.Options{Listen: listen, Discovery: disc}
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
	Agent   string   `json:"agent"`
	Code    string   `json:"code"`
	Path    string   `json:"path"`
	Text    string   `json:"text"`
	// Since is the version the window has of each machine's progress (BRDGE-B04).
	Since     map[string]uint64 `json:"since"`
	TimeoutMS int64             `json:"timeout_ms"`
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
	case "progress_all":
		return progressAll(s, r.Since, time.Duration(r.TimeoutMS)*time.Millisecond), nil
	case "listen":
		// BRDGE-B02: accept the other machines at addr, or stop with it empty
		return struct{}{}, s.Listen(r.Addr)
	case "discover":
		return discover(s, time.Duration(r.TimeoutMS)*time.Millisecond)
	case "pairings":
		type req struct {
			Name        string `json:"name"`
			Fingerprint string `json:"fingerprint"`
			Code        string `json:"code"`
		}
		out := []req{}
		for _, p := range s.Pairings() {
			out = append(out, req{p.Name, p.Fingerprint, p.Code})
		}
		return map[string]any{"requests": out}, nil
	case "pair_start":
		return map[string]string{"id": pairStart(s, r.Addr)}, nil
	case "pair_code":
		return struct{}{}, pairCode(r.ID, r.Code)
	case "pair_cancel":
		pairCancel(r.ID)
		return struct{}{}, nil
	}
	switch r.Op {
	case "sessions", "new", "close", "lock", "progress", "ask", "fs_list", "fs_read", "fs_write":
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
	// BRDGE-B06: the files of the machine, by its node
	case "fs_list":
		entries, err := n.List(r.Path)
		if entries == nil {
			entries = []hub.Entry{}
		}
		return map[string]any{"entries": entries}, err
	case "fs_read":
		b, err := n.ReadFile(r.Path)
		if err != nil {
			return nil, err
		}
		if !utf8.Valid(b) {
			return nil, fmt.Errorf("bridge: %s is not UTF-8 text", r.Path)
		}
		return map[string]string{"text": string(b)}, nil
	case "fs_write":
		return struct{}{}, n.WriteFile(r.Path, []byte(r.Text))
	case "progress":
		var out json.RawMessage
		err := n.Call("progress.list", struct{}{}, &out)
		return out, err
	case "ask":
		var out json.RawMessage
		err := n.Call("progress.ask", map[string]string{"agent": r.Agent, "text": r.Text}, &out)
		return out, err
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

// machineProgress is a machine's progress, or why it was not read.
type machineProgress struct {
	Name    string          `json:"name"`
	Version uint64          `json:"version"`
	Agents  json.RawMessage `json:"agents,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// readProgress reads a machine's progress: at once, or waiting for a change since `since` for at most d.
func readProgress(s *station.Station, name string, since *uint64, d time.Duration) machineProgress {
	out := machineProgress{Name: name}
	n, err := s.Node(name)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	var r struct {
		Version uint64          `json:"version"`
		Agents  json.RawMessage `json:"agents"`
	}
	if since == nil {
		err = n.Call("progress.list", struct{}{}, &r)
	} else {
		err = n.Call("progress.wait", map[string]any{"since": *since, "timeout_ms": d.Milliseconds()}, &r)
	}
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Version, out.Agents = r.Version, r.Agents
	return out
}

// readAll reads every machine's progress at once.
func readAll(s *station.Station) []machineProgress {
	ms := s.Machines()
	out := make([]machineProgress, len(ms))
	var wg sync.WaitGroup
	for i, m := range ms {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = readProgress(s, m.Name, nil, 0)
		}()
	}
	wg.Wait()
	return out
}

func changed(all []machineProgress, since map[string]uint64) bool {
	for _, m := range all {
		v, known := since[m.Name]
		if !known || m.Error == "" && m.Version != v {
			return true
		}
	}
	return false
}

// progressAll is BRDGE-B04: every machine's progress, as soon as one changed since the versions the
// window has, or when d passes.
func progressAll(s *station.Station, since map[string]uint64, d time.Duration) map[string]any {
	all := readAll(s)
	if changed(all, since) || d <= 0 {
		return map[string]any{"machines": all}
	}
	first := make(chan struct{}, len(all))
	for _, m := range all {
		if m.Error != "" {
			continue
		}
		v := since[m.Name]
		go func() {
			if got := readProgress(s, m.Name, &v, d); got.Error == "" && got.Version != v {
				first <- struct{}{}
			}
		}()
	}
	select {
	case <-first:
	case <-time.After(d):
	}
	return map[string]any{"machines": readAll(s)}
}

func discover(s *station.Station, d time.Duration) (any, error) {
	found, err := s.Discover(d)
	if err != nil {
		return nil, err
	}
	type m struct {
		Instance    string `json:"instance"`
		Name        string `json:"name"`
		Addr        string `json:"addr"`
		Fingerprint string `json:"fingerprint"`
		Known       bool   `json:"known"`
	}
	out := []m{}
	for _, f := range found {
		out = append(out, m{f.Instance, f.Name, f.Addr, f.Fingerprint, f.Known})
	}
	return map[string]any{"machines": out}, nil
}

// pairingRun is a pairing waiting for the code the user reads on the other machine (BRDGE-B05).
type pairingRun struct {
	code   chan string
	cancel chan struct{}
	done   chan error
}

var (
	pairMu   sync.Mutex
	pairLast int
	pairs    = map[string]*pairingRun{}
)

// pairStart begins pairing with the machine at addr, and gives the pairing's id (BRDGE-B05).
func pairStart(s *station.Station, addr string) string {
	p := &pairingRun{code: make(chan string, 1), cancel: make(chan struct{}), done: make(chan error, 1)}
	pairMu.Lock()
	pairLast++
	id := fmt.Sprintf("pair-%d", pairLast)
	pairs[id] = p
	pairMu.Unlock()
	go func() {
		p.done <- s.Pair(addr, func() (string, error) {
			select {
			case c := <-p.code:
				return c, nil
			case <-p.cancel:
				return "", errors.New("bridge: the pairing was cancelled")
			}
		})
	}()
	return id
}

func pairingOf(id string) (*pairingRun, error) {
	pairMu.Lock()
	defer pairMu.Unlock()
	p := pairs[id]
	if p == nil {
		return nil, fmt.Errorf("bridge: no pairing %s", id)
	}
	return p, nil
}

// pairCode gives the pairing the code and waits for its end (BRDGE-B05).
func pairCode(id, code string) error {
	p, err := pairingOf(id)
	if err != nil {
		return err
	}
	select {
	case p.code <- code:
	default:
	}
	err = <-p.done
	pairMu.Lock()
	delete(pairs, id)
	pairMu.Unlock()
	return err
}

// pairCancel gives the pairing up (BRDGE-B05).
func pairCancel(id string) {
	pairMu.Lock()
	p := pairs[id]
	delete(pairs, id)
	pairMu.Unlock()
	if p != nil {
		close(p.cancel)
	}
}
