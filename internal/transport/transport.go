// @anchors
//   code: TRIMP
//   ref: TRNSP

// Package transport gives the node and the client a connection between networks (TRNSP): the endpoint of
// the installation (p2p/, Rust, over iroh) through its C API, as a net.Listener and a net.Conn, for the
// SSH of Polvo to run on top as it runs on TCP. Without the build tag polvo_p2p (and cgo), Open says so.
package transport

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

// ErrNotBuilt is TRNSP-E02: this executable was built without the transport.
var ErrNotBuilt = errors.New("transport: this executable was built without the p2p transport (the polvo_p2p tag and cgo)")

// DEFAULT_RELAY is the relay an endpoint uses when none is given: the public ones of iroh (TRNSP-B01).
const DEFAULT_RELAY = "default"

// Peer is the endpoint of this installation.
type Peer interface {
	// ID is this installation's key, which is its address.
	ID() string
	// Addr is where this installation is reached now: the key, the direct addresses and the relay, as JSON.
	Addr() string
	// Listener gives the connections of the other installations (TRNSP-B02).
	Listener() net.Listener
	// Dial connects to the installation at addr, as its Addr gave it (TRNSP-B02, TRNSP-E01).
	Dial(addr string) (*Conn, error)
	// Close ends the endpoint and its connections.
	Close() error
}

// Open opens the endpoint of the installation whose key is key, with the relay ("default", "" for none,
// or a relay's URL) (TRNSP-B01).
func Open(key ed25519.PrivateKey, relay string) (Peer, error) {
	p, err := open(key.Seed(), relay)
	if err != nil {
		// TRNSP-E01, TRNSP-E02
		return nil, fmt.Errorf("transport: %w", err)
	}
	return p, nil
}

// Addr is a peer's address in the net.Addr form: its key.
type Addr string

func (a Addr) Network() string { return "p2p" }
func (a Addr) String() string  { return string(a) }

// Conn is a connection to or from another installation: a stream both ways, with its key (TRNSP-B03). It
// is a net.Conn, for the node to serve it and the client to dial over it, as over TCP (TRNSP-B04).
type Conn struct {
	stream stream
	local  Addr
	remote Addr
}

// stream is what the C API gives a connection.
type stream interface {
	read(p []byte) (int, error)
	write(p []byte) (int, error)
	closeWrite()
	close()
}

// Remote is the key of the other installation, which the handshake proved (TRNSP-B03).
func (c *Conn) Remote() string { return string(c.remote) }

func (c *Conn) Read(p []byte) (int, error)  { return c.stream.read(p) }
func (c *Conn) Write(p []byte) (int, error) { return c.stream.write(p) }

// CloseWrite says nothing more is written: the other side reads the end (TRNSP-B02).
func (c *Conn) CloseWrite() error {
	c.stream.closeWrite()
	return nil
}

func (c *Conn) Close() error {
	c.stream.close()
	return nil
}

func (c *Conn) LocalAddr() net.Addr  { return c.local }
func (c *Conn) RemoteAddr() net.Addr { return c.remote }

// noDeadline is TRNSP-E03: the stream has no deadlines; SSH does not use them.
func noDeadline() error {
	return errors.New("transport: the p2p stream has no deadlines")
}

func (c *Conn) SetDeadline(time.Time) error      { return noDeadline() }
func (c *Conn) SetReadDeadline(time.Time) error  { return noDeadline() }
func (c *Conn) SetWriteDeadline(time.Time) error { return noDeadline() }

// remoteOf is the key an address names; the address itself when it names none.
func remoteOf(addr string) Addr {
	var a struct {
		ID string `json:"id"`
	}
	if json.Unmarshal([]byte(addr), &a) == nil && a.ID != "" {
		return Addr(a.ID)
	}
	return Addr(addr)
}
