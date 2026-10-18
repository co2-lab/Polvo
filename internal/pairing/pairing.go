// @anchors
//   code: PNIMP
//   ref: PRNGS

// Package pairing authorizes the first connection between two installations with a short code that
// both compute from their keys and two random numbers, the client's exchanged with a commitment: the
// node shows it, the user types it in the client, and a man in the middle would see two different codes.
package pairing

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/sshx"
)

// CODE_DIGITS is how long the code is (PRNGS-B02).
const CODE_DIGITS = 6

// MAX_FAILURES is how many failures a node takes in FAILURE_WINDOW before it refuses (PRNGS-E03).
const MAX_FAILURES = 5

// FAILURE_WINDOW is how long a failure counts (PRNGS-E03).
const FAILURE_WINDOW = time.Minute

// ErrWrongCode is a code that is not the node's (PRNGS-E01, PRNGS-B04).
var ErrWrongCode = errors.New("pairing: the code is not the one the node shows")

// ErrTooMany refuses a request after too many failures (PRNGS-E03).
var ErrTooMany = errors.New("pairing: too many failed pairings; try again in a minute")

// Request is what the node shows the user (PRNGS-B01).
type Request struct {
	// Name is the name the client gave.
	Name string
	// Fingerprint is the SHA256 fingerprint of the client's key.
	Fingerprint string
	// Code is the code the user types in the client.
	Code string
}

// Code is the code of a pairing (PRNGS-B02): the first bytes of the SHA-256 of both keys and both
// numbers, in CODE_DIGITS digits.
func Code(client, host ssh.PublicKey, nClient, nNode []byte) string {
	h := sha256.New()
	h.Write([]byte("polvo-pairing-1"))
	for _, part := range [][]byte{client.Marshal(), host.Marshal(), nClient, nNode} {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(part)))
		h.Write(n[:])
		h.Write(part)
	}
	sum := h.Sum(nil)
	mod := uint64(1)
	for i := 0; i < CODE_DIGITS; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", CODE_DIGITS, binary.BigEndian.Uint64(sum[:8])%mod)
}

func hexHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// message is any message of the protocol, one per line (PRNGS-E02 for the others).
type message struct {
	Name   string `json:"name,omitempty"`
	Commit string `json:"commit,omitempty"`
	Nonce  string `json:"nonce,omitempty"`
	Code   string `json:"code,omitempty"`
	Abort  string `json:"abort,omitempty"`
	OK     *bool  `json:"ok,omitempty"`
	Error  string `json:"error,omitempty"`
}

type peer struct {
	r   *bufio.Reader
	enc *json.Encoder
}

func newPeer(rw io.ReadWriter) peer { return peer{r: bufio.NewReader(rw), enc: json.NewEncoder(rw)} }

func (p peer) send(m message) error { return p.enc.Encode(m) }

func (p peer) read() (message, error) {
	line, err := p.r.ReadBytes('\n')
	if err != nil {
		return message{}, fmt.Errorf("pairing: %w", err)
	}
	var m message
	if err := json.Unmarshal(line, &m); err != nil {
		// PRNGS-E02
		return message{}, fmt.Errorf("pairing: a message out of the protocol: %w", err)
	}
	return m, nil
}

func nonce() []byte {
	b := make([]byte, 16)
	rand.Read(b)
	return b
}

// Pairer is the node's side: it shows each request and authorizes the key whose code is right.
type Pairer struct {
	authorized string
	show       func(Request)
	// OnDone hears the end of each request shown, with its error, none when it paired (PRNGS-B05).
	OnDone func(Request, error)

	mu       sync.Mutex
	failures []time.Time
}

// NewPairer pairs into the authorized keys at path, showing each request with show.
func NewPairer(path string, show func(Request)) *Pairer {
	return &Pairer{authorized: path, show: show}
}

// failed counts a failure (PRNGS-E03).
func (p *Pairer) failed() {
	p.mu.Lock()
	p.failures = append(p.failures, time.Now())
	p.mu.Unlock()
}

// tooMany says whether the failures of the window reached the limit (PRNGS-E03).
func (p *Pairer) tooMany() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	kept := p.failures[:0]
	for _, at := range p.failures {
		if time.Since(at) < FAILURE_WINDOW {
			kept = append(kept, at)
		}
	}
	p.failures = kept
	return len(kept) >= MAX_FAILURES
}

// authorize adds the client's key to the authorized keys, with its name (PRNGS-B03).
func (p *Pairer) authorize(key ssh.PublicKey, name string) error {
	f, err := os.OpenFile(p.authorized, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("pairing: %w", err)
	}
	defer f.Close()
	comment := strings.Map(func(r rune) rune {
		if r < ' ' || r == '\x7f' {
			return '_'
		}
		return r
	}, name)
	_, err = fmt.Fprintln(f, sshx.AuthorizedLine(key, comment))
	return err
}

// Serve is the node's side of a pairing with the client whose key is client, over rw; host is the
// node's key.
func (p *Pairer) Serve(rw io.ReadWriter, client, host ssh.PublicKey) error {
	c := newPeer(rw)
	// PRNGS-B02: the client commits to its number before it sees the node's
	hello, err := c.read()
	if err != nil {
		return err
	}
	// the request is read before the refusal: the two ends never write at once
	if p.tooMany() {
		// PRNGS-E03
		c.send(message{Error: ErrTooMany.Error()})
		return ErrTooMany
	}
	mine := nonce()
	if err := c.send(message{Nonce: hex.EncodeToString(mine)}); err != nil {
		return err
	}
	reveal, err := c.read()
	if err != nil {
		return err
	}
	theirs, err := hex.DecodeString(reveal.Nonce)
	if err != nil || subtle.ConstantTimeCompare([]byte(hexHash(theirs)), []byte(hello.Commit)) != 1 {
		// PRNGS-E02
		p.failed()
		return errors.New("pairing: the client's number is not the one it committed to")
	}
	code := Code(client, host, theirs, mine)
	// PRNGS-B01
	req := Request{Name: hello.Name, Fingerprint: ssh.FingerprintSHA256(client), Code: code}
	p.show(req)
	err = p.finish(c, client, hello.Name, code)
	if p.OnDone != nil {
		// PRNGS-B05
		p.OnDone(req, err)
	}
	return err
}

// finish reads the code the client typed and pairs it when it is right (PRNGS-B03).
func (p *Pairer) finish(c peer, client ssh.PublicKey, name, code string) error {
	typed, err := c.read()
	if err != nil {
		return err
	}
	if typed.Abort != "" {
		p.failed()
		return fmt.Errorf("pairing: the client gave up: %s", typed.Abort)
	}
	if subtle.ConstantTimeCompare([]byte(typed.Code), []byte(code)) != 1 {
		// PRNGS-E01
		p.failed()
		no := false
		c.send(message{OK: &no})
		return ErrWrongCode
	}
	if err := p.authorize(client, name); err != nil {
		return err
	}
	yes := true
	return c.send(message{OK: &yes})
}

// Pair is the client's side, over rw, as name, with the node whose key it sees is host: typed gives the
// code the user read on the node; one that is not the code the client computed is not sent (PRNGS-B04).
func Pair(rw io.ReadWriter, name string, host ssh.PublicKey, client ssh.PublicKey, typed func() (string, error)) error {
	n := newPeer(rw)
	mine := nonce()
	if err := n.send(message{Name: name, Commit: hexHash(mine)}); err != nil {
		return err
	}
	theirs, err := n.read()
	if err != nil {
		return err
	}
	if theirs.Error != "" {
		if theirs.Error == ErrTooMany.Error() {
			return ErrTooMany
		}
		return fmt.Errorf("pairing: the node refused: %s", theirs.Error)
	}
	nodeNonce, err := hex.DecodeString(theirs.Nonce)
	if err != nil {
		return fmt.Errorf("pairing: a message out of the protocol: %w", err)
	}
	if err := n.send(message{Nonce: hex.EncodeToString(mine)}); err != nil {
		return err
	}
	code := Code(client, host, mine, nodeNonce)
	got, err := typed()
	if err != nil {
		n.send(message{Abort: err.Error()})
		return err
	}
	got = strings.ReplaceAll(strings.TrimSpace(got), " ", "")
	if subtle.ConstantTimeCompare([]byte(got), []byte(code)) != 1 {
		// PRNGS-B04: a man in the middle, or a typo
		n.send(message{Abort: "wrong code"})
		return ErrWrongCode
	}
	if err := n.send(message{Code: got}); err != nil {
		return err
	}
	done, err := n.read()
	if err != nil {
		return err
	}
	if done.OK == nil || !*done.OK {
		return ErrWrongCode
	}
	return nil
}
