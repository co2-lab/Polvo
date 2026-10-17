// @anchors
//   code: CLTST
//   ref: CLNTI

package hub

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/proto"
	"github.com/co2-lab/polvo/internal/pty"
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
	addr string
	cfg  Config
	l    net.Listener
}

// serve starts a node with hostKey that authorizes allowed, and a desktop config with identity.
func serve(t *testing.T, hostKey, allowed, identity ssh.Signer, known string) world {
	t.Helper()
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(allowed.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go node.Serve(l, node.Config{HostKey: hostKey, Authorized: sshx.NewAuthorizedKeys(ak), Control: node.ServeControl})
	return world{addr: l.Addr().String(), l: l, cfg: Config{Identity: identity, KnownHosts: sshx.NewKnownHosts(known)}}
}

func connected(t *testing.T) (*Node, world) {
	t.Helper()
	id := signer(t)
	w := serve(t, signer(t), id, id, filepath.Join(t.TempDir(), "known_hosts"))
	n, err := Dial(w.addr, w.cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Close() })
	return n, w
}

func TestClient_dial(t *testing.T) {
	t.Run("CLNTI-B01: O primeiro contacto regista o nó, a segunda ligação entra, e um nó com outra chave é recusado", func(t *testing.T) {
		id, host := signer(t), signer(t)
		known := filepath.Join(t.TempDir(), "known_hosts")
		w := serve(t, host, id, id, known)
		for i := 0; i < 2; i++ {
			n, err := Dial(w.addr, w.cfg)
			if err != nil {
				t.Fatalf("connection %d: %v", i+1, err)
			}
			n.Close()
		}
		w.l.Close()
		l, err := net.Listen("tcp", w.addr)
		if err != nil {
			t.Skip("the port was taken again: " + err.Error())
		}
		ak := filepath.Join(t.TempDir(), "authorized_keys")
		os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
		go node.Serve(l, node.Config{HostKey: signer(t), Authorized: sshx.NewAuthorizedKeys(ak)})
		defer l.Close()
		_, err = Dial(w.addr, w.cfg)
		var hc *sshx.HostKeyChangedError
		if !errors.As(err, &hc) {
			t.Fatalf("want HostKeyChangedError, got %v", err)
		}
	})
	t.Run("CLNTI-E01: Um endereço sem nó dá erro com o endereço", func(t *testing.T) {
		_, err := Dial("127.0.0.1:1", Config{Identity: signer(t), KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "k"))})
		if err == nil || !strings.Contains(err.Error(), "127.0.0.1:1") {
			t.Fatalf("want an error naming the address, got %v", err)
		}
	})
	t.Run("CLNTI-E02: Uma identidade que o nó não autoriza é recusada", func(t *testing.T) {
		w := serve(t, signer(t), signer(t), signer(t), filepath.Join(t.TempDir(), "known_hosts"))
		if _, err := Dial(w.addr, w.cfg); err == nil {
			t.Fatal("want the identity refused")
		}
	})
}

// readUntil reads the terminal until want shows up.
func readUntil(t *testing.T, term *Terminal, out *bytes.Buffer, want string) {
	t.Helper()
	b := make([]byte, 256)
	for !strings.Contains(out.String(), want) {
		n, err := term.Read(b)
		out.Write(b[:n])
		if err != nil {
			t.Fatalf("want %q, got %q and %v", want, out.String(), err)
		}
	}
}

func TestClient_terminal(t *testing.T) {
	t.Run("CLNTI-B02: Uma shell de 30 por 100 muda para 40 por 120 e termina com o código 3", func(t *testing.T) {
		n, _ := connected(t)
		term, err := n.Shell(pty.Size{Rows: 30, Cols: 100})
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		term.Write([]byte("stty size\n"))
		readUntil(t, term, &out, "30 100")
		if err := term.Resize(pty.Size{Rows: 40, Cols: 120}); err != nil {
			t.Fatal(err)
		}
		term.Write([]byte("stty size; exit 3\n"))
		readUntil(t, term, &out, "40 120")
		st, err := term.Wait()
		if err != nil || st.Code != 3 {
			t.Fatalf("want code 3, got %+v %v", st, err)
		}
	})
}

func TestClient_control(t *testing.T) {
	t.Run("CLNTI-B03: Dez chamadas ao mesmo tempo recebem cada uma a sua resposta", func(t *testing.T) {
		n, _ := connected(t)
		d := t.TempDir()
		for i := 0; i < 10; i++ {
			os.WriteFile(filepath.Join(d, fmt.Sprintf("f%d", i)), []byte(fmt.Sprint(i)), 0o644)
		}
		var wg sync.WaitGroup
		errs := make(chan error, 10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				b, err := n.ReadFile(filepath.Join(d, fmt.Sprintf("f%d", i)))
				if err != nil || string(b) != fmt.Sprint(i) {
					errs <- fmt.Errorf("f%d: got %q %v", i, b, err)
				}
			}(i)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Error(err)
		}
	})
	t.Run("CLNTI-B04: Um método desconhecido chega como erro com o código -32601", func(t *testing.T) {
		n, _ := connected(t)
		err := n.Call("fs.apagatudo", map[string]string{}, nil)
		var pe *proto.Error
		if !errors.As(err, &pe) || pe.Code != proto.CodeMethodNotFound {
			t.Fatalf("want -32601, got %v", err)
		}
	})
	t.Run("CLNTI-B05: Os atalhos escrevem, leem, listam e correm em bytes", func(t *testing.T) {
		n, _ := connected(t)
		d := t.TempDir()
		p := filepath.Join(d, "a.txt")
		if err := n.WriteFile(p, []byte("conteúdo")); err != nil {
			t.Fatal(err)
		}
		b, err := n.ReadFile(p)
		if err != nil || string(b) != "conteúdo" {
			t.Fatalf("want conteúdo, got %q %v", b, err)
		}
		es, err := n.List(d)
		if err != nil || len(es) != 1 || es[0].Name != "a.txt" || es[0].Size != 9 {
			t.Fatalf("want a.txt of 9 bytes, got %+v %v", es, err)
		}
		r, err := n.Run([]string{"sh", "-c", "cat a.txt; exit 2"}, d, nil)
		if err != nil || r.Code != 2 || string(r.Stdout) != "conteúdo" {
			t.Fatalf("want code 2 and conteúdo, got %+v %v", r, err)
		}
	})
	t.Run("CLNTI-B06: Fechar a ligação acaba a chamada que esperava", func(t *testing.T) {
		n, _ := connected(t)
		done := make(chan error, 1)
		go func() {
			_, err := n.Run([]string{"sleep", "5"}, "", nil)
			done <- err
		}()
		time.Sleep(200 * time.Millisecond)
		n.Close()
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("want the waiting call to fail")
			}
		case <-time.After(4 * time.Second):
			t.Fatal("the waiting call did not end")
		}
		if _, err := n.ReadFile("/etc/hostname"); err == nil {
			t.Fatal("want a new call to fail")
		}
	})
	t.Run("CLNTI-E03: O nó que cai durante a chamada termina-a com erro", func(t *testing.T) {
		id := signer(t)
		w := serve(t, signer(t), id, id, filepath.Join(t.TempDir(), "known_hosts"))
		raw, err := net.Dial("tcp", w.addr)
		if err != nil {
			t.Fatal(err)
		}
		n, err := DialConn(raw, w.addr, w.cfg)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, err := n.Run([]string{"sleep", "5"}, "", nil)
			done <- err
		}()
		time.Sleep(200 * time.Millisecond)
		raw.Close()
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("want the call to fail")
			}
		case <-time.After(4 * time.Second):
			t.Fatal("the call did not end")
		}
	})
}
