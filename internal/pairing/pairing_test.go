// @anchors
//   code: PNTST
//   ref: PRNGS

package pairing

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/sshx"
	"golang.org/x/crypto/ssh"
)

func signer(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type world struct {
	pairer  *Pairer
	keys    string
	client  ssh.Signer
	host    ssh.Signer
	shown   chan Request
	nodeErr chan error
}

// start serves a pairing over a pipe and gives the client's end.
func start(t *testing.T, w *world) net.Conn {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { a.Close(); b.Close() })
	go func() { w.nodeErr <- w.pairer.Serve(b, w.client.PublicKey(), w.host.PublicKey()) }()
	return a
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{keys: filepath.Join(t.TempDir(), "authorized_keys"), client: signer(t), host: signer(t), shown: make(chan Request, 4), nodeErr: make(chan error, 4)}
	w.pairer = NewPairer(w.keys, func(r Request) { w.shown <- r })
	return w
}

// typedFromNode is a user who types the code the node shows.
func (w *world) typedFromNode() func() (string, error) {
	return func() (string, error) {
		select {
		case r := <-w.shown:
			return r.Code, nil
		case <-time.After(5 * time.Second):
			return "", errors.New("the node showed no code")
		}
	}
}

func (w *world) authorized(t *testing.T) bool {
	t.Helper()
	ok, _ := sshx.NewAuthorizedKeys(w.keys).Allows(w.client.PublicKey())
	return ok
}

func TestPairing(t *testing.T) {
	t.Run("PRNGS-B01: O pedido", func(t *testing.T) {
		w := newWorld(t)
		c := start(t, w)
		var seen Request
		err := Pair(c, "portatil", w.host.PublicKey(), w.client.PublicKey(), func() (string, error) {
			select {
			case seen = <-w.shown:
				return seen.Code, nil
			case <-time.After(5 * time.Second):
				return "", errors.New("no code")
			}
		})
		if err != nil || seen.Name != "portatil" || seen.Fingerprint != ssh.FingerprintSHA256(w.client.PublicKey()) || !regexp.MustCompile(`^\d{6}$`).MatchString(seen.Code) {
			t.Fatalf("got %+v %v", seen, err)
		}
	})
	t.Run("PRNGS-B02: O código é o mesmo nos dois lados", func(t *testing.T) {
		c, h, other := signer(t).PublicKey(), signer(t).PublicKey(), signer(t).PublicKey()
		n1, n2 := []byte("um"), []byte("dois")
		if Code(c, h, n1, n2) != Code(c, h, n1, n2) || Code(c, h, n1, n2) == Code(c, other, n1, n2) || len(Code(c, h, n1, n2)) != CODE_DIGITS {
			t.Fatal("want the same code for the same keys, and another for another node's key")
		}
	})
	t.Run("PRNGS-B03: Emparelhar", func(t *testing.T) {
		w := newWorld(t)
		c := start(t, w)
		err := Pair(c, "portatil", w.host.PublicKey(), w.client.PublicKey(), w.typedFromNode())
		if err != nil || <-w.nodeErr != nil || !w.authorized(t) {
			t.Fatalf("want both ends done and the key authorized, got %v", err)
		}
	})
	t.Run("PRNGS-B04: Um intermediário", func(t *testing.T) {
		w := newWorld(t)
		c := start(t, w)
		// the client sees another key for the node: the codes differ
		err := Pair(c, "portatil", signer(t).PublicKey(), w.client.PublicKey(), w.typedFromNode())
		// the code never reached the node: it saw the client give up, not a wrong code
		nodeErr := <-w.nodeErr
		if !errors.Is(err, ErrWrongCode) || nodeErr == nil || errors.Is(nodeErr, ErrWrongCode) || w.authorized(t) {
			t.Fatalf("want ErrWrongCode in the client, the client giving up in the node, nothing authorized; got %v and %v", err, nodeErr)
		}
	})
	t.Run("PRNGS-E01: Um código errado", func(t *testing.T) {
		w := newWorld(t)
		c := start(t, w)
		// a client that skips its check, as an attacker would
		r := bufio.NewReader(c)
		enc := json.NewEncoder(c)
		nonce := []byte("0123456789abcdef")
		enc.Encode(map[string]string{"name": "x", "commit": hexHash(nonce)})
		r.ReadBytes('\n')
		enc.Encode(map[string]string{"nonce": hex.EncodeToString(nonce)})
		<-w.shown
		enc.Encode(map[string]string{"code": "000000x"})
		r.ReadBytes('\n')
		if err := <-w.nodeErr; !errors.Is(err, ErrWrongCode) || w.authorized(t) {
			t.Fatalf("want ErrWrongCode and nothing authorized, got %v", err)
		}
	})
	t.Run("PRNGS-E02: Um número que não bate com o compromisso", func(t *testing.T) {
		w := newWorld(t)
		c := start(t, w)
		r := bufio.NewReader(c)
		enc := json.NewEncoder(c)
		enc.Encode(map[string]string{"name": "x", "commit": hexHash([]byte("prometido"))})
		r.ReadBytes('\n')
		enc.Encode(map[string]string{"nonce": hex.EncodeToString([]byte("outro"))})
		if err := <-w.nodeErr; err == nil || errors.Is(err, ErrWrongCode) || w.authorized(t) {
			t.Fatalf("want the commitment error, got %v", err)
		}
	})
	t.Run("PRNGS-E03: Demasiadas falhas", func(t *testing.T) {
		w := newWorld(t)
		for i := 0; i < MAX_FAILURES; i++ {
			w.pairer.failed()
		}
		c := start(t, w)
		err := Pair(c, "x", w.host.PublicKey(), w.client.PublicKey(), w.typedFromNode())
		if nodeErr := <-w.nodeErr; !errors.Is(nodeErr, ErrTooMany) || err == nil {
			t.Fatalf("want ErrTooMany, got %v and %v", nodeErr, err)
		}
	})
}

func TestPairing_done(t *testing.T) {
	t.Run("PRNGS-B05: O fim de um pedido", func(t *testing.T) {
		w := newWorld(t)
		done := make(chan error, 2)
		w.pairer.OnDone = func(r Request, err error) { done <- err }
		Pair(start(t, w), "a", w.host.PublicKey(), w.client.PublicKey(), w.typedFromNode())
		<-w.nodeErr
		first := <-done
		c := start(t, w)
		r := bufio.NewReader(c)
		enc := json.NewEncoder(c)
		n := []byte("0123456789abcdef")
		enc.Encode(map[string]string{"name": "b", "commit": hexHash(n)})
		r.ReadBytes('\n')
		enc.Encode(map[string]string{"nonce": hex.EncodeToString(n)})
		<-w.shown
		enc.Encode(map[string]string{"code": "errado"})
		r.ReadBytes('\n')
		<-w.nodeErr
		if second := <-done; first != nil || !errors.Is(second, ErrWrongCode) {
			t.Fatalf("want nil then ErrWrongCode, got %v and %v", first, second)
		}
	})
}
