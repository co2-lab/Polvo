// @anchors
//   code: FRTST
//   ref: FRWRD

package forward

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/sessions"
	"github.com/co2-lab/polvo/internal/sshx"
)

func signer(t *testing.T) ssh.Signer {
	t.Helper()
	_, k, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// nodeVia serves a node on this machine and connects to it; change sets its configuration.
func nodeVia(t *testing.T, change func(*node.Config)) *hub.Node {
	t.Helper()
	id := signer(t)
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	cfg := node.Config{HostKey: signer(t), Authorized: sshx.NewAuthorizedKeys(ak)}
	if change != nil {
		change(&cfg)
	}
	go node.Serve(l, cfg)
	n, err := hub.Dial(l.Addr().String(), hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Close() })
	return n
}

// echo serves on l a socket that writes back what it reads.
func echo(t *testing.T, l net.Listener) {
	t.Helper()
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()
}

// roundTrip connects to addr, writes "ola" and reads what comes back.
func roundTrip(t *testing.T, network, addr string) string {
	t.Helper()
	c, err := net.Dial(network, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	c.Write([]byte("ola"))
	b := make([]byte, 3)
	n, _ := io.ReadFull(c, b)
	return string(b[:n])
}

func sock(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "fw")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return filepath.Join(d, "s")
}

func TestForward_parse(t *testing.T) {
	t.Run("FRWRD-B01: O texto de um socket", func(t *testing.T) {
		var got []string
		for _, s := range []string{"unix:/tmp/s", "maquina:80", ":8080", "8080"} {
			a, err := Parse(s)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, a.Network+" "+a.Address)
		}
		if fmt.Sprint(got) != "[unix /tmp/s tcp maquina:80 tcp 127.0.0.1:8080 tcp 127.0.0.1:8080]" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("FRWRD-E01: Um texto que não é de um socket", func(t *testing.T) {
		_, e1 := Parse("maquina:porta")
		_, e2 := Parse("70000")
		if e1 == nil || e2 == nil {
			t.Fatalf("want two errors, got %v and %v", e1, e2)
		}
	})
}

func TestForward_carry(t *testing.T) {
	t.Run("FRWRD-B02: Daqui para lá", func(t *testing.T) {
		target, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		echo(t, target)
		f, err := Local(nodeVia(t, nil), Addr{"tcp", "127.0.0.1:0"}, Addr{"tcp", target.Addr().String()})
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if got := roundTrip(t, "tcp", f.Addr().String()); got != "ola" {
			t.Fatalf("want ola, got %q", got)
		}
	})
	t.Run("FRWRD-B03: De lá para cá", func(t *testing.T) {
		path := sock(t)
		target, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		echo(t, target)
		f, err := Remote(nodeVia(t, nil), Addr{"tcp", "127.0.0.1:0"}, Addr{"unix", path})
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if got := roundTrip(t, "tcp", f.Addr().String()); got != "ola" {
			t.Fatalf("want ola, got %q", got)
		}
	})
	t.Run("FRWRD-B04: Acabar", func(t *testing.T) {
		f, err := Local(nodeVia(t, nil), Addr{"tcp", "127.0.0.1:0"}, Addr{"tcp", "127.0.0.1:1"})
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		if c, err := net.Dial("tcp", f.Addr().String()); err == nil {
			c.Close()
			t.Fatal("want the port closed")
		}
	})
	t.Run("FRWRD-E02: O nó recusa encaminhar", func(t *testing.T) {
		via := nodeVia(t, func(c *node.Config) { c.Perms = func(ssh.PublicKey) sessions.Perm { return sessions.Control } })
		f, err := Remote(via, Addr{"tcp", "127.0.0.1:0"}, Addr{"tcp", "127.0.0.1:1"})
		if err == nil {
			f.Close()
			t.Fatal("want the node to refuse")
		}
	})
	t.Run("FRWRD-E03: Um destino que não responde", func(t *testing.T) {
		f, err := Local(nodeVia(t, nil), Addr{"tcp", "127.0.0.1:0"}, Addr{"tcp", "127.0.0.1:1"})
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		a, b := roundTrip(t, "tcp", f.Addr().String()), roundTrip(t, "tcp", f.Addr().String())
		if a != "" || b != "" {
			t.Fatalf("want each connection closed with nothing, got %q and %q", a, b)
		}
	})
}
