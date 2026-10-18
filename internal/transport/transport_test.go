// @anchors
//   code: TRTST
//   ref: TRNSP

package transport_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/sshx"
	"github.com/co2-lab/polvo/internal/transport"
)

func key(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func peer(t *testing.T) (transport.Peer, ed25519.PrivateKey) {
	t.Helper()
	k := key(t)
	p, err := transport.Open(k, "")
	if errors.Is(err, transport.ErrNotBuilt) {
		t.Skip("built without the transport (make test-p2p runs these)")
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p, k
}

func TestTransport_peer(t *testing.T) {
	t.Run("TRNSP-B01: O endpoint da instalação", func(t *testing.T) {
		p, k := peer(t)
		var addr struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(p.Addr()), &addr)
		pub, _ := ssh.NewPublicKey(k.Public())
		if p.ID() == "" || addr.ID != p.ID() || pub == nil {
			t.Fatalf("want the id named by the address, got %q and %q", p.ID(), p.Addr())
		}
	})
	t.Run("TRNSP-B02: Ligar, aceitar, e a chave do outro lado", func(t *testing.T) {
		a, _ := peer(t)
		b, _ := peer(t)
		remote := make(chan string, 1)
		go func() {
			c, err := b.Listener().Accept()
			if err != nil {
				remote <- err.Error()
				return
			}
			remote <- c.(*transport.Conn).Remote()
			got, _ := io.ReadAll(c)
			c.Write(got)
			c.Close()
		}()
		c, err := a.Dial(b.Addr())
		if err != nil {
			t.Fatal(err)
		}
		c.Write([]byte("ola"))
		c.CloseWrite()
		back, _ := io.ReadAll(c)
		theirs := <-remote
		waiting := make(chan error, 1)
		go func() {
			_, err := b.Listener().Accept()
			waiting <- err
		}()
		time.Sleep(200 * time.Millisecond)
		b.Close()
		var closed error
		select {
		case closed = <-waiting:
		case <-time.After(10 * time.Second):
		}
		if string(back) != "ola" || theirs != a.ID() || c.Remote() != b.ID() || !errors.Is(closed, net.ErrClosed) {
			t.Fatalf("got %q, %q (want %q), %q (want %q), %v", back, theirs, a.ID(), c.Remote(), b.ID(), closed)
		}
	})
	t.Run("TRNSP-E01: Um endereço que não é", func(t *testing.T) {
		a, _ := peer(t)
		if _, err := a.Dial("isto não é um endereço"); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("TRNSP-E03: A ligação não tem prazos", func(t *testing.T) {
		a, _ := peer(t)
		b, _ := peer(t)
		go func() {
			if c, err := b.Listener().Accept(); err == nil {
				io.Copy(io.Discard, c)
			}
		}()
		c, err := a.Dial(b.Addr())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if c.SetDeadline(time.Now()) == nil {
			t.Fatal("want an error")
		}
	})
}

func signer(t *testing.T) ssh.Signer {
	t.Helper()
	s, err := ssh.NewSignerFromKey(key(t))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTransport_none(t *testing.T) {
	t.Run("TRNSP-E02: O executável sem o transporte", func(t *testing.T) {
		p, err := transport.Open(key(t), "")
		if err == nil {
			p.Close()
			t.Skip("built with the transport")
		}
		if !errors.Is(err, transport.ErrNotBuilt) {
			t.Fatalf("want ErrNotBuilt, got %v", err)
		}
	})
}

func TestTransport_ssh(t *testing.T) {
	t.Run("TRNSP-B04: O SSH por cima", func(t *testing.T) {
		host, client := signer(t), signer(t)
		ak := filepath.Join(t.TempDir(), "authorized_keys")
		os.WriteFile(ak, []byte(sshx.AuthorizedLine(client.PublicKey(), "")), 0o600)
		b, _ := peer(t)
		go node.Serve(b.Listener(), node.Config{HostKey: host, Authorized: sshx.NewAuthorizedKeys(ak), Control: node.ServeControl})
		a, _ := peer(t)
		c, err := a.Dial(b.Addr())
		if err != nil {
			t.Fatal(err)
		}
		n, err := hub.DialConn(c, b.ID(), hub.Config{Identity: client, HostKey: host.PublicKey()})
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()
		r, err := n.Run([]string{"/bin/sh", "-c", "echo marca-$((40+2))"}, "", nil)
		if err != nil || !strings.Contains(string(r.Stdout), "marca-42") {
			t.Fatalf("want marca-42, got %+v %v", r, err)
		}
	})
}
