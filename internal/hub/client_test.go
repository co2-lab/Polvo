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
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/pairing"
	"github.com/co2-lab/polvo/internal/proto"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sessions"
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

// withSessions connects as the owner to a node that keeps sessions.
func withSessions(t *testing.T) *Node {
	t.Helper()
	id, host := signer(t), signer(t)
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	m := sessions.NewManager()
	t.Cleanup(func() {
		for _, s := range m.List() {
			m.Close(s.ID)
		}
	})
	cfg := node.Config{HostKey: host, Authorized: sshx.NewAuthorizedKeys(ak), Sessions: m, Shell: []string{"/bin/sh"}}
	cfg.Control = node.Controller{Sessions: m, Shell: cfg.Shell}.Serve
	go node.Serve(l, cfg)
	n, err := Dial(l.Addr().String(), Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Close() })
	return n
}

// readFor reads r until it has want, or fails after a while.
func readFor(t *testing.T, r io.Reader, want string) string {
	t.Helper()
	var got bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		b := make([]byte, 4096)
		for !strings.Contains(got.String(), want) {
			k, err := r.Read(b)
			got.Write(b[:k])
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("want %q", want)
	}
	if !strings.Contains(got.String(), want) {
		t.Fatalf("want %q, got %q", want, got.String())
	}
	return got.String()
}

func TestClient_sessions(t *testing.T) {
	size := pty.Size{Rows: 24, Cols: 80}
	t.Run("CLNTI-B07: As sessões do nó", func(t *testing.T) {
		n := withSessions(t)
		id, err := n.NewSession([]string{"/bin/sh", "-c", "sleep 30"}, "", size)
		if err != nil {
			t.Fatal(err)
		}
		list, err := n.Sessions()
		if err != nil || len(list) != 1 || list[0].ID != id || !list[0].Writable || list[0].Locked || list[0].Cols != 80 {
			t.Fatalf("want the session, writable, got %+v %v", list, err)
		}
		if err := n.LockSession(id, true); err != nil {
			t.Fatal(err)
		}
		list, _ = n.Sessions()
		if !list[0].Locked {
			t.Fatalf("want it locked, got %+v", list)
		}
		if err := n.CloseSession(id); err != nil {
			t.Fatal(err)
		}
		end := time.Now().Add(5 * time.Second)
		for len(list) != 0 && time.Now().Before(end) {
			time.Sleep(20 * time.Millisecond)
			list, _ = n.Sessions()
		}
		if len(list) != 0 {
			t.Fatalf("want no session, got %+v", list)
		}
	})
	t.Run("CLNTI-B10: O ecrã de uma sessão e escrever nela", func(t *testing.T) {
		n := withSessions(t)
		id, err := n.NewSession([]string{"/bin/sh"}, "", size)
		if err != nil {
			t.Fatal(err)
		}
		defer n.CloseSession(id)
		if err := n.Type(id, []byte("echo marca-$((40+2))\n")); err != nil {
			t.Fatal(err)
		}
		var text string
		for end := time.Now().Add(10 * time.Second); !strings.Contains(text, "marca-42") && time.Now().Before(end); {
			if text, err = n.Screen(id); err != nil {
				t.Fatal(err)
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !strings.Contains(text, "marca-42") {
			t.Fatalf("want marca-42 on the screen, got %q", text)
		}
	})
	t.Run("CLNTI-B11: Os sockets da máquina do nó", func(t *testing.T) {
		n := withSessions(t)
		target, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer target.Close()
		go func() {
			if c, err := target.Accept(); err == nil {
				io.Copy(c, c)
				c.Close()
			}
		}()
		c, err := n.Connect("tcp", target.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		c.Write([]byte("ola"))
		back := make([]byte, 3)
		io.ReadFull(c, back)
		c.Close()
		l, err := n.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		go func() {
			if c, err := net.Dial("tcp", l.Addr().String()); err == nil {
				c.Write([]byte("veio"))
				c.Close()
			}
		}()
		in, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		came, _ := io.ReadAll(in)
		if string(back) != "ola" || string(came) != "veio" {
			t.Fatalf("want ola back and veio in, got %q and %q", back, came)
		}
	})
	t.Run("CLNTI-B08: Ligar-se a uma sessão", func(t *testing.T) {
		n := withSessions(t)
		id, err := n.NewSession(nil, "", size)
		if err != nil {
			t.Fatal(err)
		}
		first, err := n.Attach(id)
		if err != nil {
			t.Fatal(err)
		}
		first.Write([]byte("echo marca-$((0+1))\n"))
		readFor(t, first, "marca-1")
		second, err := n.Attach(id)
		if err != nil {
			t.Fatal(err)
		}
		readFor(t, second, "marca-1")
		second.Write([]byte("echo $((40+2))\n"))
		readFor(t, second, "42")
		second.Close()
		first.Close()
		time.Sleep(100 * time.Millisecond)
		if list, _ := n.Sessions(); len(list) != 1 {
			t.Fatalf("want the session alive, got %+v", list)
		}
	})
	t.Run("CLNTI-E04: Ligar-se a uma sessão que não existe", func(t *testing.T) {
		n := withSessions(t)
		if _, err := n.Attach("nenhuma"); !errors.Is(err, ErrNoSession) {
			t.Fatalf("want ErrNoSession, got %v", err)
		}
	})
}

func TestClient_dial(t *testing.T) {
	t.Run("CLNTI-B01#02: Com a chave do nó já sabida", func(t *testing.T) {
		id, host := signer(t), signer(t)
		w := serve(t, host, id, id, filepath.Join(t.TempDir(), "known_hosts"))
		n, err := Dial(w.addr, Config{Identity: id, HostKey: host.PublicKey()})
		if err != nil {
			t.Fatalf("want in with the node's key, got %v", err)
		}
		n.Close()
		if _, err := Dial(w.addr, Config{Identity: id, HostKey: signer(t).PublicKey()}); err == nil {
			t.Fatal("want another node's key refused")
		}
	})
	t.Run("CLNTI-B01#01: O primeiro contacto regista o nó, a segunda ligação entra, e um nó com outra chave é recusado", func(t *testing.T) {
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
		select {
		case <-n.Done():
		case <-time.After(4 * time.Second):
			t.Fatal("want Done closed when the connection fell")
		}
	})
}

func TestClient_pairing(t *testing.T) {
	t.Run("CLNTI-B09: Emparelhar com um nó", func(t *testing.T) {
		host, id := signer(t), signer(t)
		keys := filepath.Join(t.TempDir(), "authorized_keys")
		shown := make(chan pairing.Request, 1)
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { l.Close() })
		go node.Serve(l, node.Config{HostKey: host, Authorized: sshx.NewAuthorizedKeys(keys), Control: node.ServeControl,
			Pairer: pairing.NewPairer(keys, func(r pairing.Request) { shown <- r })})
		known := filepath.Join(t.TempDir(), "known_hosts")
		cfg := Config{Identity: id, KnownHosts: sshx.NewKnownHosts(known)}
		code := func() (string, error) {
			// the node's key is not known before the code proves it
			if b, _ := os.ReadFile(known); len(b) > 0 {
				return "", fmt.Errorf("the node's key was recorded before the code: %q", b)
			}
			return (<-shown).Code, nil
		}
		if err := Pair(l.Addr().String(), cfg, "portatil", code); err != nil {
			t.Fatal(err)
		}
		n, err := Dial(l.Addr().String(), cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()
		if _, err := n.List(t.TempDir()); err != nil {
			t.Fatalf("want the node's control after pairing, got %v", err)
		}
		if b, _ := os.ReadFile(known); !strings.Contains(string(b), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(host.PublicKey())))) {
			t.Fatalf("want the node's key known, got %q", b)
		}
	})
}
