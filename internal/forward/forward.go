// @anchors
//   code: FRIMP
//   ref: FRWRD

// Package forward carries sockets between two machines through the node, both ways, like ssh -L and
// ssh -R (FRWRD).
package forward

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

// DEFAULT_HOST is where a socket given only by its port listens (FRWRD-B01).
const DEFAULT_HOST = "127.0.0.1"

// Addr is a socket: "tcp" with host:port, or "unix" with a path.
type Addr struct {
	Network string
	Address string
}

func (a Addr) String() string {
	if a.Network == "unix" {
		return "unix:" + a.Address
	}
	return a.Address
}

// Parse reads a socket from text (FRWRD-B01): "unix:/path", "host:port", ":port" or "port"; anything else
// is FRWRD-E01.
func Parse(text string) (Addr, error) {
	if path, ok := strings.CutPrefix(text, "unix:"); ok && path != "" {
		return Addr{Network: "unix", Address: path}, nil
	}
	host, port := DEFAULT_HOST, strings.TrimPrefix(text, ":")
	if i := strings.LastIndex(text, ":"); i > 0 {
		host, port = strings.Trim(text[:i], "[]"), text[i+1:]
	}
	if n, err := strconv.Atoi(port); err != nil || n < 0 || n > 65535 {
		// FRWRD-E01
		return Addr{}, fmt.Errorf("forward: %q is not a socket (unix:/path, host:port or port)", text)
	}
	return Addr{Network: "tcp", Address: net.JoinHostPort(host, port)}, nil
}

// Via is what reaches the node's machine: hub.Node.
type Via interface {
	Connect(network, addr string) (net.Conn, error)
	Listen(network, addr string) (net.Listener, error)
}

// Forward is a forwarding that runs until Close.
type Forward struct {
	l    net.Listener
	dial func() (net.Conn, error)

	mu    sync.Mutex
	conns map[net.Conn]struct{}
	done  bool
}

// Local listens on listen on this machine and takes each connection there, through the node, to target on
// its machine (FRWRD-B02); FRWRD-E02 when it cannot listen.
func Local(via Via, listen, target Addr) (*Forward, error) {
	l, err := net.Listen(listen.Network, listen.Address)
	if err != nil {
		// FRWRD-E02
		return nil, fmt.Errorf("forward: listen on %s: %w", listen, err)
	}
	return start(l, func() (net.Conn, error) { return via.Connect(target.Network, target.Address) }), nil
}

// Remote makes the node listen on listen on its machine and takes each connection there to target on this
// machine (FRWRD-B03); FRWRD-E02 when the node refuses.
func Remote(via Via, listen, target Addr) (*Forward, error) {
	l, err := via.Listen(listen.Network, listen.Address)
	if err != nil {
		// FRWRD-E02
		return nil, fmt.Errorf("forward: the node did not listen on %s: %w", listen, err)
	}
	return start(l, func() (net.Conn, error) { return net.Dial(target.Network, target.Address) }), nil
}

func start(l net.Listener, dial func() (net.Conn, error)) *Forward {
	f := &Forward{l: l, dial: dial, conns: map[net.Conn]struct{}{}}
	go f.accept()
	return f
}

// Addr is where the forwarding listens: on this machine for Local, on the node's for Remote.
func (f *Forward) Addr() net.Addr { return f.l.Addr() }

// Close stops listening and ends the connections that were passing (FRWRD-B04).
func (f *Forward) Close() error {
	f.mu.Lock()
	f.done = true
	for c := range f.conns {
		c.Close()
	}
	f.mu.Unlock()
	return f.l.Close()
}

func (f *Forward) accept() {
	for {
		in, err := f.l.Accept()
		if err != nil {
			return
		}
		go f.carry(in)
	}
}

// track keeps c to end it with the forwarding; false when the forwarding already ended.
func (f *Forward) track(c net.Conn) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.done {
		return false
	}
	f.conns[c] = struct{}{}
	return true
}

func (f *Forward) untrack(cs ...net.Conn) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range cs {
		delete(f.conns, c)
	}
}

// carry takes in to the target until one side ends.
func (f *Forward) carry(in net.Conn) {
	defer in.Close()
	out, err := f.dial()
	if err != nil {
		// FRWRD-E03: this connection ends; the next ones are tried again
		return
	}
	defer out.Close()
	if !f.track(in) || !f.track(out) {
		return
	}
	defer f.untrack(in, out)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(out, in); closeWrite(out) }()
	go func() { defer wg.Done(); io.Copy(in, out); closeWrite(in) }()
	wg.Wait()
}

// closeWrite tells the other side that nothing more comes, or ends the connection when it cannot.
func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
		return
	}
	c.Close()
}
