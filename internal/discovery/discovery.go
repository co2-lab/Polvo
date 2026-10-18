// @anchors
//   code: DSIMP
//   ref: DSCVR

// Package discovery finds the nodes of the local network by DNS-SD over mDNS (RFC 6762, 6763): the
// node answers the questions for its service, and the window asks.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/ipv4"
)

// SERVICE is the DNS-SD service type of Polvo's node.
const SERVICE = "_polvo._tcp.local."

// GROUP is the group and the port of mDNS over IPv4.
const GROUP = "224.0.0.251:5353"

// Service is what a node announces (DSCVR-B01).
type Service struct {
	// Instance is the name of this node among the others: the machine's, by default.
	Instance string
	// Host is the machine's name, without the .local.
	Host string
	Port int
	IPs  []net.IP
	// Fingerprint is the SHA256 fingerprint of the node's key.
	Fingerprint string
}

// Options are where discovery happens: the group (GROUP when nil) and the interface (all when nil).
type Options struct {
	Group     *net.UDPAddr
	Interface *net.Interface
}

func (o Options) group() *net.UDPAddr {
	if o.Group != nil {
		return o.Group
	}
	g, _ := net.ResolveUDPAddr("udp4", GROUP)
	return g
}

// Found is a node that answered (DSCVR-B02).
type Found struct {
	Instance string
	// Name is the machine's name the node gave.
	Name string
	// Addr is where the node listens, host:port (DSCVR-B04).
	Addr        string
	Fingerprint string
}

// Announcer answers the questions for the service while it lives.
type Announcer struct {
	conn  *net.UDPConn
	pc    *ipv4.PacketConn
	group *net.UDPAddr
	reply []byte
	done  sync.WaitGroup
}

func instanceName(instance string) string { return instance + "." + SERVICE }

// answer is the packet that answers a question for the service (DSCVR-B01).
func answer(s Service) (packet []byte, err error) {
	inst := dnsmessage.MustNewName(instanceName(s.Instance))
	host, err := dnsmessage.NewName(s.Host + ".local.")
	if err != nil {
		return nil, err
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, Authoritative: true})
	b.EnableCompression()
	b.StartAnswers()
	hdr := func(n dnsmessage.Name) dnsmessage.ResourceHeader {
		return dnsmessage.ResourceHeader{Name: n, Class: dnsmessage.ClassINET, TTL: 120}
	}
	if err := b.PTRResource(hdr(dnsmessage.MustNewName(SERVICE)), dnsmessage.PTRResource{PTR: inst}); err != nil {
		return nil, err
	}
	b.StartAdditionals()
	if err := b.SRVResource(hdr(inst), dnsmessage.SRVResource{Target: host, Port: uint16(s.Port)}); err != nil {
		return nil, err
	}
	txt := []string{"fp=" + s.Fingerprint, "name=" + s.Host}
	if err := b.TXTResource(hdr(inst), dnsmessage.TXTResource{TXT: txt}); err != nil {
		return nil, err
	}
	for _, ip := range s.IPs {
		if v4 := ip.To4(); v4 != nil {
			if err := b.AResource(hdr(host), dnsmessage.AResource{A: [4]byte(v4)}); err != nil {
				return nil, err
			}
		}
	}
	return b.Finish()
}

// Announce answers, until Close, the questions for the service in the group (DSCVR-B01); the error when
// the group or the interface do not open (DSCVR-E01).
func Announce(s Service, o Options) (*Announcer, error) {
	reply, err := answer(s)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	g := o.group()
	if !g.IP.IsMulticast() {
		// DSCVR-E01
		return nil, fmt.Errorf("discovery: %s is not a multicast group", g)
	}
	conn, err := net.ListenMulticastUDP("udp4", o.Interface, g)
	if err != nil {
		// DSCVR-E01
		return nil, fmt.Errorf("discovery: listening on %s: %w", g, err)
	}
	pc := ipv4.NewPacketConn(conn)
	if o.Interface != nil {
		pc.SetMulticastInterface(o.Interface)
	}
	pc.SetMulticastLoopback(true)
	a := &Announcer{conn: conn, pc: pc, group: g, reply: reply}
	a.done.Add(1)
	go a.serve()
	return a, nil
}

// asked says whether the packet is a question for the service; a packet that is not DNS is not
// (DSCVR-E02).
func asked(packet []byte) bool {
	var parser dnsmessage.Parser
	h, err := parser.Start(packet)
	if err != nil || h.Response {
		return false
	}
	qs, err := parser.AllQuestions()
	if err != nil {
		return false
	}
	for _, q := range qs {
		if strings.EqualFold(q.Name.String(), SERVICE) && (q.Type == dnsmessage.TypePTR || q.Type == dnsmessage.TypeALL) {
			return true
		}
	}
	return false
}

func (a *Announcer) serve() {
	defer a.done.Done()
	buf := make([]byte, 9000)
	for {
		n, from, err := a.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if !asked(buf[:n]) {
			continue
		}
		// DSCVR-B03: from mDNS's port the answer goes to the group; from another, back to it
		to := from
		if from.Port == a.group.Port {
			to = a.group
		}
		a.conn.WriteToUDP(a.reply, to)
	}
}

// Close stops answering.
func (a *Announcer) Close() error {
	err := a.conn.Close()
	a.done.Wait()
	return err
}

func question() []byte {
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	b.StartQuestions()
	b.Question(dnsmessage.Question{Name: dnsmessage.MustNewName(SERVICE), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET})
	m, _ := b.Finish()
	return m
}

// heard gathers what the answers said, by instance.
type heard struct {
	instances map[string]bool
	srv       map[string]dnsmessage.SRVResource
	txt       map[string][]string
	a         map[string]net.IP
	from      map[string]net.IP
}

func (h *heard) take(packet []byte, from net.IP) {
	var m dnsmessage.Message
	if m.Unpack(packet) != nil || !m.Header.Response {
		// DSCVR-E02
		return
	}
	var insts []string
	for _, r := range append(m.Answers, m.Additionals...) {
		name := strings.ToLower(r.Header.Name.String())
		switch b := r.Body.(type) {
		case *dnsmessage.PTRResource:
			if name == SERVICE {
				insts = append(insts, strings.ToLower(b.PTR.String()))
			}
		case *dnsmessage.SRVResource:
			h.srv[name] = *b
		case *dnsmessage.TXTResource:
			h.txt[name] = b.TXT
		case *dnsmessage.AResource:
			if _, ok := h.a[name]; !ok {
				h.a[name] = net.IP(b.A[:])
			}
		}
	}
	for _, i := range insts {
		h.instances[i] = true
		h.from[i] = from
	}
}

// Browse asks for the service and gives, until d passes, the nodes that answer (DSCVR-B02).
func Browse(ctx context.Context, d time.Duration, o Options) ([]Found, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	defer conn.Close()
	pc := ipv4.NewPacketConn(conn)
	if o.Interface != nil {
		pc.SetMulticastInterface(o.Interface)
	}
	pc.SetMulticastLoopback(true)
	g := o.group()
	if _, err := conn.WriteToUDP(question(), g); err != nil {
		return nil, fmt.Errorf("discovery: asking %s: %w", g, err)
	}
	h := &heard{instances: map[string]bool{}, srv: map[string]dnsmessage.SRVResource{}, txt: map[string][]string{}, a: map[string]net.IP{}, from: map[string]net.IP{}}
	end := time.Now().Add(d)
	again := time.Now().Add(d / 3)
	buf := make([]byte, 9000)
	for time.Now().Before(end) && ctx.Err() == nil {
		if time.Now().After(again) {
			// a second question, for a node that missed the first
			conn.WriteToUDP(question(), g)
			again = end
		}
		next := end
		if again.Before(next) {
			next = again
		}
		conn.SetReadDeadline(next)
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return nil, fmt.Errorf("discovery: %w", err)
		}
		h.take(buf[:n], from.IP)
	}
	var out []Found
	for inst := range h.instances {
		srv, ok := h.srv[inst]
		if !ok {
			continue
		}
		f := Found{Instance: strings.TrimSuffix(inst, "."+SERVICE)}
		for _, t := range h.txt[inst] {
			if v, ok := strings.CutPrefix(t, "fp="); ok {
				f.Fingerprint = v
			}
			if v, ok := strings.CutPrefix(t, "name="); ok {
				f.Name = v
			}
		}
		if f.Name == "" {
			f.Name = f.Instance
		}
		// DSCVR-B04
		ip := h.a[strings.ToLower(srv.Target.String())]
		if ip == nil {
			ip = h.from[inst]
		}
		f.Addr = net.JoinHostPort(ip.String(), strconv.Itoa(int(srv.Port)))
		out = append(out, f)
	}
	return out, nil
}
