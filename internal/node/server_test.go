// @anchors
//   code: SRTST
//   ref: SRVRE

package node

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/sshx"
	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/pairing"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sessions"
)

func signer(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type fixture struct {
	addr   string
	client ssh.Signer
	host   ssh.Signer
}

// startNode serves a node on 127.0.0.1 that authorizes one fresh client key.
func startNode(t *testing.T, change func(*Config)) fixture {
	t.Helper()
	host, client := signer(t), signer(t)
	p := filepath.Join(t.TempDir(), "authorized_keys")
	if err := os.WriteFile(p, []byte(sshx.AuthorizedLine(client.PublicKey(), "teste")), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{HostKey: host, Authorized: sshx.NewAuthorizedKeys(p)}
	if change != nil {
		change(&cfg)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go Serve(l, cfg)
	return fixture{addr: l.Addr().String(), client: client, host: host}
}

func (f fixture) dial(t *testing.T, auth ssh.AuthMethod) (*ssh.Client, error) {
	t.Helper()
	return ssh.Dial("tcp", f.addr, &ssh.ClientConfig{
		User: "polvo", Auth: []ssh.AuthMethod{auth}, HostKeyCallback: ssh.FixedHostKey(f.host.PublicKey()),
	})
}

func (f fixture) session(t *testing.T) *ssh.Session {
	t.Helper()
	c, err := f.dial(t, ssh.PublicKeys(f.client))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	s, err := c.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// shell runs an interactive shell in a terminal, types input and returns everything it printed.
func shell(t *testing.T, s *ssh.Session, rows, cols int, before func(), input string) string {
	t.Helper()
	if err := s.RequestPty("xterm", rows, cols, ssh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	in, _ := s.StdinPipe()
	var out bytes.Buffer
	s.Stdout = &out
	if err := s.Shell(); err != nil {
		t.Fatal(err)
	}
	if before != nil {
		before()
	}
	io.WriteString(in, input)
	s.Wait()
	return out.String()
}

func TestServer_auth(t *testing.T) {
	t.Run("SRVRE-B01: A chave autorizada entra, a outra chave e a palavra-passe não", func(t *testing.T) {
		f := startNode(t, nil)
		c, err := f.dial(t, ssh.PublicKeys(f.client))
		if err != nil {
			t.Fatalf("want the authorized key in, got %v", err)
		}
		if _, err := c.NewSession(); err != nil {
			t.Fatalf("want a session, got %v", err)
		}
		c.Close()
		if _, err := f.dial(t, ssh.PublicKeys(signer(t))); err == nil {
			t.Fatal("want another key refused")
		}
		if _, err := f.dial(t, ssh.Password("segredo")); err == nil {
			t.Fatal("want the password refused")
		}
	})
}

func TestServer_owner(t *testing.T) {
	t.Run("SRVRE-B11: A chave da instalação entra sempre, como dona", func(t *testing.T) {
		own := signer(t)
		f := startNode(t, func(c *Config) {
			c.Owner = own.PublicKey()
			c.Perms = func(ssh.PublicKey) sessions.Perm { return sessions.View }
		})
		c, err := f.dial(t, ssh.PublicKeys(own))
		if err != nil {
			t.Fatalf("want the installation's key in, got %v", err)
		}
		defer c.Close()
		s, err := c.NewSession()
		if err != nil {
			t.Fatal(err)
		}
		out := shell(t, s, 24, 80, nil, "echo dona-$((2+2)); exit\n")
		if !strings.Contains(out, "dona-4") {
			t.Fatalf("want the owner's shell, got %q", out)
		}
	})
}

func TestServer_terminal(t *testing.T) {
	t.Run("SRVRE-B02: A shell interativa corre num terminal de 30 por 100", func(t *testing.T) {
		f := startNode(t, nil)
		out := shell(t, f.session(t), 30, 100, nil, "stty size; exit\n")
		if !strings.Contains(out, "30 100") {
			t.Fatalf("want 30 100, got %q", out)
		}
	})
	t.Run("SRVRE-B03: A mudança de janela para 40 por 120 chega à shell", func(t *testing.T) {
		f := startNode(t, nil)
		s := f.session(t)
		out := shell(t, s, 24, 80, func() {
			if err := s.WindowChange(40, 120); err != nil {
				t.Fatal(err)
			}
		}, "stty size; exit\n")
		if !strings.Contains(out, "40 120") {
			t.Fatalf("want 40 120, got %q", out)
		}
	})
	t.Run("SRVRE-B05: O comando avulso com terminal vê o terminal", func(t *testing.T) {
		f := startNode(t, nil)
		s := f.session(t)
		if err := s.RequestPty("xterm", 24, 80, ssh.TerminalModes{}); err != nil {
			t.Fatal(err)
		}
		out, _ := s.Output("tty >/dev/null && echo tem-tty")
		if !strings.Contains(string(out), "tem-tty") {
			t.Fatalf("want tem-tty, got %q", out)
		}
	})
}

func TestServer_exec(t *testing.T) {
	t.Run("SRVRE-B04: O comando avulso separa a saída do erro e recebe a entrada", func(t *testing.T) {
		f := startNode(t, nil)
		s := f.session(t)
		var out, errb bytes.Buffer
		s.Stdin, s.Stdout, s.Stderr = strings.NewReader("entrada"), &out, &errb
		if err := s.Run("echo saida; echo erro >&2; cat"); err != nil {
			t.Fatal(err)
		}
		if out.String() != "saida\nentrada" || errb.String() != "erro\n" {
			t.Fatalf("want stdout and stderr apart, got %q and %q", out.String(), errb.String())
		}
	})
	t.Run("SRVRE-B06: O código de saída 7 chega a quem ligou", func(t *testing.T) {
		f := startNode(t, nil)
		var ee *ssh.ExitError
		if err := f.session(t).Run("exit 7"); !errors.As(err, &ee) || ee.ExitStatus() != 7 {
			t.Fatalf("want exit status 7, got %v", err)
		}
	})
	t.Run("SRVRE-B06: O sinal que matou o comando chega a quem ligou", func(t *testing.T) {
		f := startNode(t, nil)
		var ee *ssh.ExitError
		if err := f.session(t).Run("kill -9 $$"); !errors.As(err, &ee) || ee.Signal() != "KILL" {
			t.Fatalf("want the KILL signal, got %v", err)
		}
	})
	t.Run("SRVRE-E02: Uma shell que não existe dá 127 e a mensagem no erro", func(t *testing.T) {
		f := startNode(t, func(c *Config) { c.Shell = []string{"nao-existe-polvo"} })
		s := f.session(t)
		var errb bytes.Buffer
		s.Stderr = &errb
		var ee *ssh.ExitError
		if err := s.Run("echo x"); !errors.As(err, &ee) || ee.ExitStatus() != 127 || !strings.Contains(errb.String(), "nao-existe-polvo") {
			t.Fatalf("want 127 and the message, got %v and %q", err, errb.String())
		}
	})
}

func TestServer_subsystemsAndRefusals(t *testing.T) {
	t.Run("SRVRE-B07: O subsistema polvo chega ao controlo e o sftp é recusado", func(t *testing.T) {
		f := startNode(t, func(c *Config) {
			c.Control = func(rw io.ReadWriteCloser, _ sessions.Perm) {
				b := make([]byte, 2)
				io.ReadFull(rw, b)
				io.WriteString(rw, "controlo:"+string(b))
				rw.Close()
			}
		})
		s := f.session(t)
		in, _ := s.StdinPipe()
		out, _ := s.StdoutPipe()
		if err := s.RequestSubsystem("polvo"); err != nil {
			t.Fatal(err)
		}
		io.WriteString(in, "oi")
		got, _ := io.ReadAll(out)
		if string(got) != "controlo:oi" {
			t.Fatalf("want controlo:oi, got %q", got)
		}
		if err := f.session(t).RequestSubsystem("sftp"); err == nil {
			t.Fatal("want sftp refused")
		}
	})
	t.Run("SRVRE-B08: Um canal desconhecido, o encaminhamento sem permissão e um pedido desconhecido são recusados", func(t *testing.T) {
		f := startNode(t, func(c *Config) { c.Perms = func(ssh.PublicKey) sessions.Perm { return sessions.Control } })
		c, err := f.dial(t, ssh.PublicKeys(f.client))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_, _, err = c.OpenChannel("x11", nil)
		var oe *ssh.OpenChannelError
		if !errors.As(err, &oe) || oe.Reason != ssh.UnknownChannelType {
			t.Fatalf("want unknown channel type, got %v", err)
		}
		_, err = c.Dial("tcp", "127.0.0.1:1")
		if !errors.As(err, &oe) || oe.Reason != ssh.Prohibited {
			t.Fatalf("want the forward prohibited, got %v", err)
		}
		if l, err := c.Listen("tcp", "127.0.0.1:0"); err == nil {
			l.Close()
			t.Fatal("want the listen refused")
		}
		s, err := c.NewSession()
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := s.SendRequest("xpto", true, nil); ok || err != nil {
			t.Fatalf("want xpto refused, got %v %v", ok, err)
		}
		if out, err := s.Output("echo ainda"); err != nil || string(out) != "ainda\n" {
			t.Fatalf("want the session still running, got %q %v", out, err)
		}
	})
	t.Run("SRVRE-E01: Sem a identidade do nó o nó não serve", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		if err := Serve(l, Config{}); err == nil {
			t.Fatal("want an error")
		}
	})
}

type idRequest struct{ ID string }

// collect reads r in the background, for the test to wait on what arrives.
func collect(r io.Reader) func() string {
	var mu sync.Mutex
	var buf bytes.Buffer
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := r.Read(b)
			mu.Lock()
			buf.Write(b[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}
}

func waitFor(t *testing.T, got func() string, want string) {
	t.Helper()
	end := time.Now().Add(10 * time.Second)
	for time.Now().Before(end) {
		if strings.Contains(got(), want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want %q, got %q", want, got())
}

// openShell opens an interactive shell on a connection of its own, and returns its input and output.
func openShell(t *testing.T, f fixture) (*ssh.Client, io.Writer, func() string) {
	t.Helper()
	c, err := f.dial(t, ssh.PublicKeys(f.client))
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPty("xterm", 24, 80, ssh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	in, _ := s.StdinPipe()
	out, _ := s.StdoutPipe()
	if err := s.Shell(); err != nil {
		t.Fatal(err)
	}
	return c, in, collect(out)
}

func TestServer_sessions(t *testing.T) {
	t.Run("SRVRE-B02: Fechar a ligação deixa a sessão a correr", func(t *testing.T) {
		m := sessions.NewManager()
		f := startNode(t, func(c *Config) { c.Sessions = m })
		c, in, out := openShell(t, f)
		io.WriteString(in, "echo marca-$((1+1))\n")
		waitFor(t, out, "marca-2")
		c.Close()
		time.Sleep(200 * time.Millisecond)
		if len(m.List()) != 1 {
			t.Fatalf("want the session still running, got %v", m.List())
		}
		m.Close(m.List()[0].ID)
	})
	t.Run("SRVRE-B09: Ligar-se a uma sessão que já existe", func(t *testing.T) {
		m := sessions.NewManager()
		f := startNode(t, func(c *Config) { c.Sessions = m })
		c, in, out := openShell(t, f)
		io.WriteString(in, "echo marca-$((0+1))\n")
		waitFor(t, out, "marca-1")
		c.Close()
		id := m.List()[0].ID
		defer m.Close(id)
		s := f.session(t)
		in2, _ := s.StdinPipe()
		out2, _ := s.StdoutPipe()
		if ok, err := s.SendRequest("attach@polvo", true, ssh.Marshal(idRequest{id})); !ok || err != nil {
			t.Fatalf("want the attach accepted, got %v %v", ok, err)
		}
		got := collect(out2)
		waitFor(t, got, "marca-1")
		io.WriteString(in2, "echo soma-$((2+3))\n")
		waitFor(t, got, "soma-5")
	})
	t.Run("SRVRE-B10: Uma ligação que só vê não cria sessões nem escreve", func(t *testing.T) {
		m := sessions.NewManager()
		f := startNode(t, func(c *Config) {
			c.Sessions = m
			c.Perms = func(ssh.PublicKey) sessions.Perm { return sessions.View }
		})
		info, err := m.Create([]string{"/bin/sh"}, t.TempDir(), pty.Size{Rows: 24, Cols: 80})
		if err != nil {
			t.Fatal(err)
		}
		defer m.Close(info.ID)
		owner, _ := m.Attach(info.ID, sessions.Owner)
		defer owner.Close()
		seen := collect(owner)
		s := f.session(t)
		s.RequestPty("xterm", 24, 80, ssh.TerminalModes{})
		shellErr := s.Shell()
		_, execErr := f.session(t).Output("echo nao")
		v := f.session(t)
		in, _ := v.StdinPipe()
		if ok, err := v.SendRequest("attach@polvo", true, ssh.Marshal(idRequest{info.ID})); !ok || err != nil {
			t.Fatalf("want the view-only attach accepted, got %v %v", ok, err)
		}
		io.WriteString(in, "echo $((7*6))\n")
		owner.Write([]byte("echo dono-$((1+2))\n"))
		waitFor(t, seen, "dono-3")
		if shellErr == nil || execErr == nil || strings.Contains(seen(), "42") {
			t.Fatalf("want shell and exec refused and no 42, got %v, %v, %q", shellErr, execErr, seen())
		}
	})
	t.Run("SRVRE-E03: Um id que não é de nenhuma sessão é recusado", func(t *testing.T) {
		f := startNode(t, nil)
		s := f.session(t)
		if ok, err := s.SendRequest("attach@polvo", true, ssh.Marshal(idRequest{"nenhuma"})); ok || err != nil {
			t.Fatalf("want the attach refused, got %v %v", ok, err)
		}
		if out, err := s.Output("echo ainda"); err != nil || string(out) != "ainda\n" {
			t.Fatalf("want the session still running, got %q %v", out, err)
		}
	})
}

func TestServer_pairing(t *testing.T) {
	t.Run("SRVRE-B12: Uma chave desconhecida só emparelha", func(t *testing.T) {
		shown := make(chan pairing.Request, 1)
		var keys string
		stranger := signer(t)
		f := startNode(t, func(c *Config) {
			keys = filepath.Join(t.TempDir(), "authorized_keys")
			c.Authorized = sshx.NewAuthorizedKeys(keys)
			c.Pairer = pairing.NewPairer(keys, func(r pairing.Request) { shown <- r })
		})
		c, err := f.dial(t, ssh.PublicKeys(stranger))
		if err != nil {
			t.Fatalf("want the unknown key in, to pair; got %v", err)
		}
		defer c.Close()
		s1, _ := c.NewSession()
		shellErr := s1.Shell()
		s2, _ := c.NewSession()
		in, _ := s2.StdinPipe()
		out, _ := s2.StdoutPipe()
		if err := s2.RequestSubsystem("pairing@polvo"); err != nil {
			t.Fatal(err)
		}
		rw := struct {
			io.Reader
			io.Writer
		}{out, in}
		err = pairing.Pair(rw, "estranho", f.host.PublicKey(), stranger.PublicKey(), func() (string, error) { return (<-shown).Code, nil })
		if shellErr == nil || err != nil {
			t.Fatalf("want the shell refused and the pairing done, got %v and %v", shellErr, err)
		}
		again, err := f.dial(t, ssh.PublicKeys(stranger))
		if err != nil {
			t.Fatal(err)
		}
		defer again.Close()
		s3, _ := again.NewSession()
		if out := shell(t, s3, 24, 80, nil, "echo emparelhado-$((1+1)); exit\n"); !strings.Contains(out, "emparelhado-2") {
			t.Fatalf("want a shell now, got %q", out)
		}
	})
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

// roundTrip writes "ola" in c and reads it back.
func roundTrip(t *testing.T, c net.Conn) string {
	t.Helper()
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := c.Write([]byte("ola")); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 3)
	if _, err := io.ReadFull(c, b); err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// sock is a path for a Unix socket, short enough for the system's limit.
func sock(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "pf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return filepath.Join(d, "s")
}

func TestServer_forward(t *testing.T) {
	t.Run("SRVRE-B13#01: Encaminhar da ligação para um socket do nó", func(t *testing.T) {
		tl, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		echo(t, tl)
		path := sock(t)
		ul, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		echo(t, ul)
		f := startNode(t, nil)
		c, err := f.dial(t, ssh.PublicKeys(f.client))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		var got []string
		for _, target := range [][2]string{{"tcp", tl.Addr().String()}, {"unix", path}} {
			conn, err := c.Dial(target[0], target[1])
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, roundTrip(t, conn))
		}
		if strings.Join(got, ",") != "ola,ola" {
			t.Fatalf("want ola twice, got %v", got)
		}
	})
	t.Run("SRVRE-B13#02: Encaminhar de um socket do nó para a ligação", func(t *testing.T) {
		f := startNode(t, nil)
		c, err := f.dial(t, ssh.PublicKeys(f.client))
		if err != nil {
			t.Fatal(err)
		}
		tl, err := c.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		path := sock(t)
		ul, err := c.ListenUnix(path)
		if err != nil {
			t.Fatal(err)
		}
		echo(t, tl)
		echo(t, ul)
		var got []string
		for _, target := range [][2]string{{"tcp", tl.Addr().String()}, {"unix", path}} {
			conn, err := net.Dial(target[0], target[1])
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, roundTrip(t, conn))
		}
		c.Close()
		closed := false
		for end := time.Now().Add(5 * time.Second); !closed && time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
			conn, err := net.Dial("tcp", tl.Addr().String())
			closed = err != nil
			if conn != nil {
				conn.Close()
			}
		}
		if strings.Join(got, ",") != "ola,ola" || !closed {
			t.Fatalf("want ola twice and the listener closed with the connection, got %v and %v", got, closed)
		}
	})
	t.Run("SRVRE-E04: Um socket que não responde", func(t *testing.T) {
		f := startNode(t, nil)
		c, err := f.dial(t, ssh.PublicKeys(f.client))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_, err = c.Dial("tcp", "127.0.0.1:1")
		var oe *ssh.OpenChannelError
		if !errors.As(err, &oe) || oe.Reason != ssh.ConnectionFailed {
			t.Fatalf("want the connection failed, got %v", err)
		}
	})
}
