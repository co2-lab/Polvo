// @anchors
//   code: STIMP
//   ref: STATN

// Package station is the installation of Polvo as the window sees it: this machine's node, run in the
// process, and the other machines, all reached by the same client.
package station

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/discovery"
	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/nodeconf"
	"github.com/co2-lab/polvo/internal/pairing"
	"github.com/co2-lab/polvo/internal/progress"
	"github.com/co2-lab/polvo/internal/sessions"
	"github.com/co2-lab/polvo/internal/sshx"
)

// LOCAL is the name of this window's machine, which has no address.
const LOCAL = "local"

// Options are how the Station opens.
type Options struct {
	// Listen is where this machine's node listens for the others (STATN-B02); empty does not listen.
	Listen string
	// Shell is the program of a new session that names none; empty is /bin/sh.
	Shell []string
	// Name is this machine's name in the network; empty is the host's name.
	Name string
	// Discovery is where this node announces itself while it listens, and where Discover asks; nil
	// neither announces nor asks (STATN-B09, STATN-B10).
	Discovery *discovery.Options
}

// Machine is a machine the window reaches (STATN-B03).
type Machine struct {
	// Name is LOCAL, or the address of another machine.
	Name  string
	Local bool
}

// Station is the installation as the window sees it.
type Station struct {
	dir      string
	id       ssh.Signer
	hostKey  ssh.PublicKey
	known    *sshx.KnownHosts
	sessions *sessions.Manager

	listener  net.Listener
	listenErr error
	announcer *discovery.Announcer
	discovery *discovery.Options
	name      string

	pairMu   sync.Mutex
	requests []pairing.Request
	// own is where this window reaches its own node: a loopback address only it uses.
	own net.Listener

	// connMu keeps conns apart from mu: the Node holds mu while its own node accepts
	connMu   sync.Mutex
	conns    map[net.Conn]struct{}
	shut     bool
	mu       sync.Mutex
	machines []string
	nodes    map[string]*hub.Node
	closed   bool
}

type saved struct {
	Machines []string `json:"machines"`
}

// Open opens the installation of dir and runs this machine's node in the process (STATN-B01), listening
// where opt says (STATN-B02).
func Open(dir string, opt Options) (*Station, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	id, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return nil, err
	}
	host, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "host_ed25519"))
	if err != nil {
		return nil, err
	}
	perms, err := nodeconf.Load(filepath.Join(dir, "node.json"), id.PublicKey())
	if err != nil {
		return nil, err
	}
	// STATN-B06
	board, err := progress.NewBoard(filepath.Join(dir, "progress"))
	if err != nil {
		return nil, err
	}
	m := sessions.NewManager()
	name := opt.Name
	if name == "" {
		name, _ = os.Hostname()
	}
	s := &Station{dir: dir, id: id, known: sshx.NewKnownHosts(filepath.Join(dir, "known_hosts")), sessions: m,
		nodes: map[string]*hub.Node{}, hostKey: host.PublicKey(), conns: map[net.Conn]struct{}{}, discovery: opt.Discovery, name: name}
	// STATN-B08: the requests shown stay until they end
	pairer := pairing.NewPairer(filepath.Join(dir, "authorized_keys"), func(r pairing.Request) {
		s.pairMu.Lock()
		s.requests = append(s.requests, r)
		s.pairMu.Unlock()
	})
	pairer.OnDone = func(r pairing.Request, _ error) {
		s.pairMu.Lock()
		defer s.pairMu.Unlock()
		for i, x := range s.requests {
			if x == r {
				s.requests = append(s.requests[:i], s.requests[i+1:]...)
				break
			}
		}
	}
	cfg := node.Config{
		HostKey:    host,
		Authorized: sshx.NewAuthorizedKeys(filepath.Join(dir, "authorized_keys")),
		Shell:      opt.Shell,
		Sessions:   m,
		Perms:      perms.PermFor,
		Forward:    perms.MayForward,
		Owner:      id.PublicKey(),
		Pairer:     pairer,
	}
	cfg.Control = node.Controller{Sessions: m, Shell: opt.Shell, Progress: board}.Serve
	if b, err := os.ReadFile(s.file()); err == nil {
		var sv saved
		if json.Unmarshal(b, &sv) == nil {
			s.machines = sv.Machines
		}
	}
	// a loopback listener, not a net.Pipe: both ends of SSH write their version first, and an
	// unbuffered pipe blocks them both
	own, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s.own = s.track(own)
	go node.Serve(s.own, cfg)
	// STATN-B07: an agent in a session reports to this node
	m.SetEnv([]string{"POLVO_NODE=" + own.Addr().String()})
	if opt.Listen != "" {
		l, err := net.Listen("tcp", opt.Listen)
		if err != nil {
			// STATN-E01: this machine still works, without the others
			s.listenErr = fmt.Errorf("station: listening on %s: %w", opt.Listen, err)
		} else {
			s.listener = s.track(l)
			go node.Serve(s.listener, cfg)
			s.announce(l.Addr())
		}
	}
	return s, nil
}

// track keeps the connections l accepts, for the Close to end them.
func (s *Station) track(l net.Listener) net.Listener { return &tracking{Listener: l, s: s} }

type tracking struct {
	net.Listener
	s *Station
}

func (t *tracking) Accept() (net.Conn, error) {
	c, err := t.Listener.Accept()
	if err != nil {
		return nil, err
	}
	t.s.connMu.Lock()
	defer t.s.connMu.Unlock()
	if t.s.shut {
		c.Close()
		return nil, errors.New("station: closed")
	}
	t.s.conns[c] = struct{}{}
	return &tracked{Conn: c, s: t.s}, nil
}

type tracked struct {
	net.Conn
	s *Station
}

func (c *tracked) Close() error {
	c.s.connMu.Lock()
	delete(c.s.conns, c.Conn)
	c.s.connMu.Unlock()
	return c.Conn.Close()
}

func (s *Station) file() string { return filepath.Join(s.dir, "machines.json") }

// PublicKey is this installation's key, the one the other machines authorize.
func (s *Station) PublicKey() ssh.PublicKey { return s.id.PublicKey() }

// ListenAddr is where this machine's node listens for the others; empty when it does not (STATN-B02).
func (s *Station) ListenAddr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// ListenError says why the node does not listen (STATN-E01).
func (s *Station) ListenError() error { return s.listenErr }

// Machines gives this window's machine and then the known ones, in the order they were added (STATN-B03).
func (s *Station) Machines() []Machine {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Machine{{Name: LOCAL, Local: true}}
	for _, m := range s.machines {
		out = append(out, Machine{Name: m})
	}
	return out
}

func (s *Station) remembers(name string) bool {
	for _, m := range s.machines {
		if m == name {
			return true
		}
	}
	return false
}

// save writes the known machines at once.
func (s *Station) save() error {
	b, err := json.MarshalIndent(saved{Machines: s.machines}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.file() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.file())
}

// dial connects to a machine: this one inside the process, as the owner; another by its address.
func (s *Station) dial(name string) (*hub.Node, error) {
	cfg := hub.Config{Identity: s.id, KnownHosts: s.known}
	if name == LOCAL {
		conn, err := net.Dial("tcp", s.own.Addr().String())
		if err != nil {
			return nil, err
		}
		// this machine's node has no address for the known hosts: its key is known
		return hub.DialConn(conn, LOCAL, hub.Config{Identity: s.id, HostKey: s.hostKey})
	}
	n, err := hub.Dial(name, cfg)
	if err != nil {
		// STATN-E02: the client's error names the address
		return nil, err
	}
	return n, nil
}

// Add connects to the machine at addr and, when it does, keeps it (STATN-B03); the error when it
// does not answer or refuses this installation (STATN-E02).
func (s *Station) Add(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if addr == LOCAL || s.remembers(addr) {
		return nil
	}
	n, err := s.dial(addr)
	if err != nil {
		return err
	}
	s.nodes[addr] = n
	s.machines = append(s.machines, addr)
	return s.save()
}

// Remove closes the connection to a known machine and forgets it (STATN-B04).
func (s *Station) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := s.nodes[name]; n != nil {
		n.Close()
		delete(s.nodes, name)
	}
	kept := s.machines[:0]
	for _, m := range s.machines {
		if m != name {
			kept = append(kept, m)
		}
	}
	s.machines = kept
	return s.save()
}

var errUnknown = errors.New("station: unknown machine")

// Node gives the connection to a machine, connecting the first time and again after it fell
// (STATN-B05).
func (s *Station) Node(name string) (*hub.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New("station: closed")
	}
	if name != LOCAL && !s.remembers(name) {
		return nil, fmt.Errorf("%w: %s", errUnknown, name)
	}
	if n := s.nodes[name]; n != nil {
		select {
		case <-n.Done():
		default:
			return n, nil
		}
	}
	n, err := s.dial(name)
	if err != nil {
		return nil, err
	}
	s.nodes[name] = n
	return n, nil
}

// Close stops listening, ends the connections of the other machines and its own, and ends this
// machine's sessions (STATN-B01).
func (s *Station) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	nodes := s.nodes
	s.nodes = map[string]*hub.Node{}
	s.mu.Unlock()
	if s.listener != nil {
		s.listener.Close()
	}
	if s.announcer != nil {
		s.announcer.Close()
	}
	s.own.Close()
	// the connections the node took: closing the listener leaves them open
	s.connMu.Lock()
	s.shut = true
	conns := s.conns
	s.conns = map[net.Conn]struct{}{}
	s.connMu.Unlock()
	for c := range conns {
		c.Close()
	}
	for _, n := range nodes {
		n.Close()
	}
	for _, i := range s.sessions.List() {
		s.sessions.Close(i.ID)
	}
	return nil
}

// HostFingerprint is the SHA256 fingerprint of this machine's node's key, the one the others see.
func (s *Station) HostFingerprint() string { return ssh.FingerprintSHA256(s.hostKey) }

// announce announces this node in the network while it listens (STATN-B09); a failure only means the
// others will not find it by themselves.
func (s *Station) announce(addr net.Addr) {
	tcp, ok := addr.(*net.TCPAddr)
	// on the loopback, no other machine reaches this node
	if s.discovery == nil || !ok || tcp.IP.IsLoopback() {
		return
	}
	port := tcp.Port
	var ips []net.IP
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() && n.IP.To4() != nil {
				ips = append(ips, n.IP)
			}
		}
	}
	fp := s.HostFingerprint()
	short := strings.TrimPrefix(fp, "SHA256:")
	if len(short) > 6 {
		short = short[:6]
	}
	svc := discovery.Service{Instance: s.name + "-" + short, Host: s.name, Port: port, IPs: ips, Fingerprint: fp}
	a, err := discovery.Announce(svc, *s.discovery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "polvo: this machine is not announced in the network: %v\n", err)
		return
	}
	s.announcer = a
}

// Pairings are the pairing requests this node shows, until they end (STATN-B08).
func (s *Station) Pairings() []pairing.Request {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	return append([]pairing.Request(nil), s.requests...)
}

// Discovered is a machine of the network, and whether this installation knows it (STATN-B10).
type Discovered struct {
	discovery.Found
	Known bool
}

// knownFingerprints are the fingerprints of the nodes in the known hosts.
func (s *Station) knownFingerprints() map[string]bool {
	out := map[string]bool{}
	b, err := os.ReadFile(filepath.Join(s.dir, "known_hosts"))
	if err != nil {
		return out
	}
	for len(b) > 0 {
		_, _, key, _, rest, err := ssh.ParseKnownHosts(b)
		if err != nil {
			break
		}
		out[ssh.FingerprintSHA256(key)] = true
		b = rest
	}
	return out
}

// Discover gives the machines that announce themselves in the network, without this one, each with
// whether it is known (STATN-B10).
func (s *Station) Discover(d time.Duration) ([]Discovered, error) {
	if s.discovery == nil {
		return nil, nil
	}
	found, err := discovery.Browse(context.Background(), d, *s.discovery)
	if err != nil {
		return nil, err
	}
	known := s.knownFingerprints()
	mine := s.HostFingerprint()
	var out []Discovered
	for _, f := range found {
		if f.Fingerprint == mine {
			continue
		}
		out = append(out, Discovered{Found: f, Known: known[f.Fingerprint]})
	}
	return out, nil
}

// Pair pairs this installation with the machine at addr, by the code the user reads on it, and adds it
// (STATN-B11).
func (s *Station) Pair(addr string, typed func() (string, error)) error {
	cfg := hub.Config{Identity: s.id, KnownHosts: s.known}
	if err := hub.Pair(addr, cfg, s.name, typed); err != nil {
		return err
	}
	return s.Add(addr)
}
