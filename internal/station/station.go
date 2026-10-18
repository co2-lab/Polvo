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

	"crypto/ed25519"
	"encoding/hex"
	"github.com/co2-lab/polvo/internal/discovery"
	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/nodeconf"
	"github.com/co2-lab/polvo/internal/pairing"
	"github.com/co2-lab/polvo/internal/progress"
	"github.com/co2-lab/polvo/internal/sessions"
	"github.com/co2-lab/polvo/internal/sshx"
	"github.com/co2-lab/polvo/internal/transport"
	"sort"
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
	// Relay is the relay of the transport between networks: transport.DEFAULT_RELAY, empty for none, or
	// a relay's URL (STATN-B14).
	Relay string
	// P2PPort is the UDP port of the transport; 0 is any (STATN-B14).
	P2PPort int
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

	// listenMu keeps the listening apart: Listen changes it while the window runs (STATN-B12)
	listenMu  sync.Mutex
	listener  net.Listener
	listenErr error
	cfg       node.Config
	// accepting says whether the others are let in, by the network and between networks (STATN-B12)
	accepting bool

	// p2p is the transport between networks; nil without it (STATN-B14)
	p2p       transport.Peer
	p2pErr    error
	histMu    sync.Mutex
	announcer *discovery.Announcer
	discovery *discovery.Options
	name      string

	pairMu   sync.Mutex
	requests []pairing.Request
	// own is where this window reaches its own node: a loopback address only it uses.
	own net.Listener

	// connMu keeps conns apart from mu: the Node holds mu while its own node accepts
	connMu sync.Mutex
	// conns are the connections the node took, true for those of the other machines
	conns    map[net.Conn]bool
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
		nodes: map[string]*hub.Node{}, hostKey: host.PublicKey(), conns: map[net.Conn]bool{}, discovery: opt.Discovery, name: name}
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
	cfg.Control = node.Controller{Sessions: m, Shell: opt.Shell, Progress: board, P2P: s.P2PAddr}.Serve
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
	s.cfg = cfg
	if err := s.openP2P(filepath.Join(dir, "host_ed25519"), opt, cfg); err != nil {
		return nil, err
	}
	listen := opt.Listen
	if listen == "" {
		// STATN-B12: where the user left it
		if b, err := os.ReadFile(s.acceptFile()); err == nil {
			var a accepting
			if json.Unmarshal(b, &a) == nil {
				listen = a.Listen
			}
		}
	}
	// STATN-E01: a failure leaves this machine working, without the others
	s.listen(listen)
	return s, nil
}

// accepting is where the user left this machine listening for the others (STATN-B12).
type accepting struct {
	Listen string `json:"listen"`
}

func (s *Station) acceptFile() string { return filepath.Join(s.dir, "accept.json") }

// Listen makes this machine's node listen for the others at addr, or stop with an empty addr, while the
// window runs, and the installation remembers it for the next Open (STATN-B12): stopping also ends the
// connections the others had made. A failure is STATN-E01, and is not remembered.
func (s *Station) Listen(addr string) error {
	if err := s.listen(addr); err != nil {
		return err
	}
	b, _ := json.Marshal(accepting{Listen: addr})
	return os.WriteFile(s.acceptFile(), b, 0o600)
}

func (s *Station) listen(addr string) error {
	s.listenMu.Lock()
	defer s.listenMu.Unlock()
	s.accepting = addr != ""
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
	if s.announcer != nil {
		s.announcer.Close()
		s.announcer = nil
	}
	s.listenErr = nil
	s.connMu.Lock()
	var outside []net.Conn
	for c, out := range s.conns {
		if out {
			outside = append(outside, c)
			delete(s.conns, c)
		}
	}
	s.connMu.Unlock()
	for _, c := range outside {
		c.Close()
	}
	if addr == "" {
		return nil
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		// STATN-E01
		s.listenErr = fmt.Errorf("station: listening on %s: %w", addr, err)
		return s.listenErr
	}
	s.listener = s.trackOutside(l)
	go node.Serve(s.listener, s.cfg)
	s.announce(l.Addr())
	return nil
}

// track keeps the connections l accepts, for the Close to end them.
func (s *Station) track(l net.Listener) net.Listener { return &tracking{Listener: l, s: s} }

// trackOutside is track for the listener of the other machines, whose connections Listen ends.
func (s *Station) trackOutside(l net.Listener) net.Listener {
	return &tracking{Listener: l, s: s, outside: true}
}

type tracking struct {
	net.Listener
	s       *Station
	outside bool
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
	t.s.conns[c] = t.outside
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
	s.listenMu.Lock()
	defer s.listenMu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// ListenError says why the node does not listen (STATN-E01).
func (s *Station) ListenError() error {
	s.listenMu.Lock()
	defer s.listenMu.Unlock()
	return s.listenErr
}

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
	if isP2P(name) {
		return s.dialP2P(name)
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
	s.listen("")
	if s.p2p != nil {
		s.p2p.Close()
	}
	s.own.Close()
	// the connections the node took: closing the listener leaves them open
	s.connMu.Lock()
	s.shut = true
	conns := s.conns
	s.conns = map[net.Conn]bool{}
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
	// @no-guard: hub.Pair fails on an address that is not a node's (STATN-E02), and Add guards after it
	cfg := hub.Config{Identity: s.id, KnownHosts: s.known}
	if err := hub.Pair(addr, cfg, s.name, typed); err != nil {
		return err
	}
	return s.Add(addr)
}

// P2P_PREFIX names a machine reached between networks by its key: "p2p:" and the key (STATN-B13).
const P2P_PREFIX = "p2p:"

// MAX_HISTORY is how many addresses of a machine the history keeps, the most recent ones (STATN-B15).
const MAX_HISTORY = 16

// seen is an address where a machine answered, when it last did, and how many times.
type seen struct {
	Addr  string    `json:"addr"`
	Last  time.Time `json:"last"`
	Count int       `json:"count"`
}

// known is what the history keeps of a machine: its relay, and the addresses it answered on.
type known struct {
	Relay string `json:"relay,omitempty"`
	Addrs []seen `json:"addrs"`
}

// endpoint is the address JSON of the transport: the key, and the paths to it.
type endpoint struct {
	ID    string            `json:"id"`
	Addrs []json.RawMessage `json:"addrs"`
}

type path struct {
	IP    string `json:"Ip,omitempty"`
	Relay string `json:"Relay,omitempty"`
}

func (s *Station) historyFile() string { return filepath.Join(s.dir, "addresses.json") }

// openP2P opens this installation's endpoint with the node's key, and serves the node over it while the
// others are accepted (STATN-B14); without the transport, the station works without it.
func (s *Station) openP2P(hostPath string, opt Options, cfg node.Config) error {
	b, err := os.ReadFile(hostPath)
	if err != nil {
		return err
	}
	raw, err := ssh.ParseRawPrivateKey(b)
	if err != nil {
		return err
	}
	k, ok := raw.(*ed25519.PrivateKey)
	if !ok {
		return errors.New("station: the node's key is not ed25519")
	}
	p, err := transport.Open(*k, opt.Relay, opt.P2PPort)
	if errors.Is(err, transport.ErrNotBuilt) {
		return nil
	}
	if err != nil {
		// STATN-E03: this machine works, without the other networks
		s.p2pErr = err
		return nil
	}
	s.p2p = p
	go s.acceptP2P(cfg)
	return nil
}

// acceptP2P serves the node to the connections of the other installations, while this machine accepts
// them (STATN-B14); the others are closed.
func (s *Station) acceptP2P(cfg node.Config) {
	l := s.p2p.Listener()
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		s.listenMu.Lock()
		on := s.accepting
		s.listenMu.Unlock()
		if !on {
			c.Close()
			continue
		}
		s.connMu.Lock()
		if s.shut {
			s.connMu.Unlock()
			c.Close()
			return
		}
		s.conns[c] = true
		s.connMu.Unlock()
		go node.ServeConn(&tracked{Conn: c, s: s}, cfg)
	}
}

// P2PAddr is where this installation is reached between networks now, as the transport gives it; empty
// without the transport (STATN-B14).
func (s *Station) P2PAddr() string {
	if s.p2p == nil {
		return ""
	}
	return s.p2p.Addr()
}

// P2PError says why the transport did not open (STATN-E03).
func (s *Station) P2PError() error { return s.p2pErr }

// keyOf is the SSH key a p2p name says, for the node to be checked against it (STATN-B13).
func keyOf(name string) (ssh.PublicKey, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(name, P2P_PREFIX))
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("station: %s is not a machine's key", name)
	}
	return ssh.NewPublicKey(ed25519.PublicKey(b))
}

// readHistory is the history of every machine reached between networks.
func (s *Station) readHistory() map[string]known {
	h := map[string]known{}
	if b, err := os.ReadFile(s.historyFile()); err == nil {
		json.Unmarshal(b, &h)
	}
	return h
}

// remember adds where a machine is now to its history (STATN-B15).
func (s *Station) remember(addr string) (string, error) {
	var e endpoint
	if err := json.Unmarshal([]byte(addr), &e); err != nil || e.ID == "" {
		return "", fmt.Errorf("station: %q is not a machine's address", addr)
	}
	s.histMu.Lock()
	defer s.histMu.Unlock()
	h := s.readHistory()
	k := h[e.ID]
	now := time.Now()
	for _, raw := range e.Addrs {
		var p path
		if json.Unmarshal(raw, &p) != nil {
			continue
		}
		if p.Relay != "" {
			k.Relay = p.Relay
			continue
		}
		if p.IP == "" {
			continue
		}
		found := false
		for i := range k.Addrs {
			if k.Addrs[i].Addr == p.IP {
				k.Addrs[i].Last, k.Addrs[i].Count, found = now, k.Addrs[i].Count+1, true
			}
		}
		if !found {
			k.Addrs = append(k.Addrs, seen{Addr: p.IP, Last: now, Count: 1})
		}
	}
	// the most recent first, then the most used; the oldest beyond the limit are forgotten
	sort.SliceStable(k.Addrs, func(i, j int) bool {
		if !k.Addrs[i].Last.Equal(k.Addrs[j].Last) {
			return k.Addrs[i].Last.After(k.Addrs[j].Last)
		}
		return k.Addrs[i].Count > k.Addrs[j].Count
	})
	if len(k.Addrs) > MAX_HISTORY {
		k.Addrs = k.Addrs[:MAX_HISTORY]
	}
	h[e.ID] = k
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := s.historyFile() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return "", err
	}
	return P2P_PREFIX + e.ID, os.Rename(tmp, s.historyFile())
}

// addrOf is the address of a machine with every path its history has: the transport tries them all at
// once, the old ones too (STATN-B15).
func (s *Station) addrOf(id string) string {
	s.histMu.Lock()
	k := s.readHistory()[id]
	s.histMu.Unlock()
	e := endpoint{ID: id}
	for _, a := range k.Addrs {
		b, _ := json.Marshal(path{IP: a.Addr})
		e.Addrs = append(e.Addrs, b)
	}
	if k.Relay != "" {
		b, _ := json.Marshal(path{Relay: k.Relay})
		e.Addrs = append(e.Addrs, b)
	}
	b, _ := json.Marshal(e)
	return string(b)
}

// dialP2P connects to a machine by its key, over the transport, with its node's key checked against it
// (STATN-B13), and keeps where it is now in the history (STATN-B15).
func (s *Station) dialP2P(name string) (*hub.Node, error) {
	if s.p2p == nil {
		// STATN-E03
		return nil, fmt.Errorf("station: %s is reached between networks, and this installation has no transport", name)
	}
	key, err := keyOf(name)
	if err != nil {
		return nil, err
	}
	conn, err := s.p2p.Dial(s.addrOf(strings.TrimPrefix(name, P2P_PREFIX)))
	if err != nil {
		// STATN-E02
		return nil, fmt.Errorf("station: %s: %w", name, err)
	}
	n, err := hub.DialConn(conn, name, hub.Config{Identity: s.id, HostKey: key})
	if err != nil {
		return nil, err
	}
	var now struct {
		Addr string `json:"addr"`
	}
	if n.Call("p2p.addr", struct{}{}, &now) == nil {
		s.remember(now.Addr)
	}
	return n, nil
}

// AddP2P joins the machine at addr (the address its installation gave), reached between networks by its
// key (STATN-B13); its name is "p2p:" and the key.
func (s *Station) AddP2P(addr string) (string, error) {
	// @no-guard: remember refuses an address that is not a machine's (STATN-E02), and Add guards the name
	name, err := s.remember(addr)
	if err != nil {
		return "", err
	}
	return name, s.Add(name)
}

// isP2P says whether a machine's name is its key.
func isP2P(name string) bool { return strings.HasPrefix(name, P2P_PREFIX) }
