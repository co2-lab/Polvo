// @anchors
//   code: MCTST
//   ref: MCPTL

package mcp

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/progress"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sessions"
	"github.com/co2-lab/polvo/internal/sshx"
	"github.com/co2-lab/polvo/internal/transport"
	"golang.org/x/crypto/ssh"
)

func newSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// nodeAndPool serves a node and returns its address and a pool allowed into it.
func nodeAndPool(t *testing.T) (string, *Pool) {
	t.Helper()
	id := newSigner(t)
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go node.Serve(l, node.Config{HostKey: newSigner(t), Authorized: sshx.NewAuthorizedKeys(ak), Control: node.ServeControl})
	p := NewPool(hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
	t.Cleanup(p.Close)
	return l.Addr().String(), p
}

func callTool(t *testing.T, p *Pool, name string, args map[string]any) Result {
	t.Helper()
	i := slices.IndexFunc(p.Tools(), func(x Tool) bool { return x.Name == name })
	if i < 0 {
		t.Fatalf("no tool %s", name)
	}
	b, _ := json.Marshal(args)
	r, err := p.Tools()[i].Call(b)
	if err != nil {
		return Result{Text: err.Error(), IsError: true}
	}
	return r
}

func TestTools_run(t *testing.T) {
	t.Run("MCPTL-B01: polvo_run devolve o código 3, a saída e o erro", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		r := callTool(t, p, "polvo_run", map[string]any{"node": addr, "command": "echo saida; echo erro >&2; exit 3"})
		if r.IsError || !strings.Contains(r.Text, "exit code: 3") || !strings.Contains(r.Text, "--- stdout ---\nsaida") || !strings.Contains(r.Text, "--- stderr ---\nerro") {
			t.Fatalf("want code 3, stdout and stderr, got %+v", r)
		}
	})
	t.Run("MCPTL-B01: polvo_run com pasta e entrada", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		r := callTool(t, p, "polvo_run", map[string]any{"node": addr, "command": "pwd; cat", "dir": "/tmp", "stdin": "entrada"})
		if r.IsError || !strings.Contains(r.Text, "/tmp\nentrada") {
			t.Fatalf("want /tmp and entrada, got %+v", r)
		}
	})
	t.Run("MCPTL-E01#01: Sem o comando, polvo_run é uma falha", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		if r := callTool(t, p, "polvo_run", map[string]any{"node": addr}); !r.IsError || !strings.Contains(r.Text, "command") {
			t.Fatalf("want a failure naming command, got %+v", r)
		}
	})
	t.Run("MCPTL-E02: Um nó que não responde é uma falha com o endereço", func(t *testing.T) {
		_, p := nodeAndPool(t)
		if r := callTool(t, p, "polvo_run", map[string]any{"node": "127.0.0.1:1", "command": "true"}); !r.IsError || !strings.Contains(r.Text, "127.0.0.1:1") {
			t.Fatalf("want a failure with the address, got %+v", r)
		}
	})
}

func TestTools_files(t *testing.T) {
	t.Run("MCPTL-B02: polvo_read_file devolve o texto, e um ficheiro binário é uma falha", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "a.txt"), []byte("ola"), 0o644)
		os.WriteFile(filepath.Join(d, "b.bin"), []byte{0xff, 0xfe, 0x00, 0x01}, 0o644)
		if r := callTool(t, p, "polvo_read_file", map[string]any{"node": addr, "path": filepath.Join(d, "a.txt")}); r.IsError || r.Text != "ola" {
			t.Fatalf("want ola, got %+v", r)
		}
		if r := callTool(t, p, "polvo_read_file", map[string]any{"node": addr, "path": filepath.Join(d, "b.bin")}); !r.IsError || !strings.Contains(r.Text, "4 bytes") {
			t.Fatalf("want a failure with the size, got %+v", r)
		}
	})
	t.Run("MCPTL-B03: polvo_write_file escreve e diz quanto escreveu", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		f := filepath.Join(t.TempDir(), "c.txt")
		r := callTool(t, p, "polvo_write_file", map[string]any{"node": addr, "path": f, "content": "conteúdo"})
		b, _ := os.ReadFile(f)
		if r.IsError || !strings.Contains(r.Text, "9 bytes") || !strings.Contains(r.Text, f) || string(b) != "conteúdo" {
			t.Fatalf("want 9 bytes written to %s, got %+v and %q", f, r, b)
		}
	})
	t.Run("MCPTL-B04: polvo_list mostra as pastas com barra e os ficheiros com o tamanho", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		d := t.TempDir()
		os.Mkdir(filepath.Join(d, "d"), 0o755)
		os.WriteFile(filepath.Join(d, "f.txt"), []byte("abc"), 0o644)
		if r := callTool(t, p, "polvo_list", map[string]any{"node": addr, "path": d}); r.IsError || r.Text != "d/\nf.txt (3 bytes)" {
			t.Fatalf("want d/ and f.txt (3 bytes), got %+v", r)
		}
	})
	t.Run("MCPTL-E03: Ler um ficheiro que não existe é uma falha com a mensagem do nó", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		if r := callTool(t, p, "polvo_read_file", map[string]any{"node": addr, "path": "/nao/existe"}); !r.IsError || !strings.Contains(r.Text, "/nao/existe") {
			t.Fatalf("want a failure with the path, got %+v", r)
		}
	})
}

func TestTools_pool(t *testing.T) {
	t.Run("MCPTL-B05: Duas chamadas ao mesmo nó usam a mesma ligação", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		callTool(t, p, "polvo_run", map[string]any{"node": addr, "command": "true"})
		first := p.nodes[addr]
		callTool(t, p, "polvo_run", map[string]any{"node": addr, "command": "true"})
		if len(p.nodes) != 1 || p.nodes[addr] != first {
			t.Fatalf("want one connection reused, got %d", len(p.nodes))
		}
	})
	t.Run("MCPTL-B06: A ligação caída é refeita na chamada seguinte", func(t *testing.T) {
		addr, p := nodeAndPool(t)
		callTool(t, p, "polvo_run", map[string]any{"node": addr, "command": "true"})
		first := p.nodes[addr]
		first.Close()
		if r := callTool(t, p, "polvo_run", map[string]any{"node": addr, "command": "echo outra"}); r.IsError || !strings.Contains(r.Text, "outra") {
			t.Fatalf("want the command run on a new connection, got %+v", r)
		}
		if p.nodes[addr] == first {
			t.Fatal("want a new connection")
		}
	})
}

// progressNode serves a node with a progress board, and gives its address, the board and a pool that
// knows where it runs: in the session "abc" of that node.
func progressNode(t *testing.T) (string, *progress.Board, *Pool) {
	t.Helper()
	id := newSigner(t)
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	board, err := progress.NewBoard(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	go node.Serve(l, node.Config{HostKey: newSigner(t), Authorized: sshx.NewAuthorizedKeys(ak), Control: node.Controller{Progress: board}.Serve})
	p := NewPool(hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
	t.Cleanup(p.Close)
	p.SetHere(Here{Node: l.Addr().String(), Session: "abc"})
	return l.Addr().String(), board, p
}

func job(t *testing.T, b *progress.Board, agent string) progress.Progress {
	t.Helper()
	got, err := b.Get(agent)
	if err != nil {
		t.Fatalf("no job of %s: %v", agent, err)
	}
	return got
}

func TestTools_progress(t *testing.T) {
	start := map[string]any{"goal": "migrar", "steps": []string{"ler", "escrever"}}
	t.Run("MCPTL-B07: polvo_progress_start começa um trabalho", func(t *testing.T) {
		_, b, p := progressNode(t)
		r := callTool(t, p, "polvo_progress_start", start)
		got := job(t, b, "session-abc")
		if r.IsError || !strings.Contains(r.Text, "1. ler") || !strings.Contains(r.Text, "2. escrever") || got.Goal != "migrar" {
			t.Fatalf("got %+v and %+v", r, got)
		}
	})
	t.Run("MCPTL-B08: polvo_progress_step muda o estado de um passo", func(t *testing.T) {
		_, b, p := progressNode(t)
		callTool(t, p, "polvo_progress_start", start)
		r := callTool(t, p, "polvo_progress_step", map[string]any{"step": "ler", "state": "done"})
		if r.IsError || job(t, b, "session-abc").Steps[0].State != progress.Done {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("MCPTL-B09: polvo_progress_note deixa uma nota", func(t *testing.T) {
		_, b, p := progressNode(t)
		callTool(t, p, "polvo_progress_start", start)
		r := callTool(t, p, "polvo_progress_note", map[string]any{"text": "a meio"})
		log := job(t, b, "session-abc").Log
		if r.IsError || log[len(log)-1].Text != "a meio" {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("MCPTL-B10: polvo_progress_finish termina o trabalho", func(t *testing.T) {
		_, b, p := progressNode(t)
		callTool(t, p, "polvo_progress_start", start)
		r := callTool(t, p, "polvo_progress_finish", map[string]any{"summary": "feito"})
		if r.IsError || !job(t, b, "session-abc").Finished {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("MCPTL-B11: O agente e o nó dados", func(t *testing.T) {
		addr, b, p := progressNode(t)
		p.SetHere(Here{})
		r := callTool(t, p, "polvo_progress_start", map[string]any{"goal": "g", "agent": "outro", "node": addr})
		if r.IsError || job(t, b, "outro").Goal != "g" {
			t.Fatalf("got %+v", r)
		}
	})
	t.Run("MCPTL-E04: Sem agente fora de uma sessão", func(t *testing.T) {
		_, _, p := progressNode(t)
		p.SetHere(Here{})
		r := callTool(t, p, "polvo_progress_start", start)
		if !r.IsError || !strings.Contains(r.Text, "agent") {
			t.Fatalf("want the tool to ask for the agent, got %+v", r)
		}
	})
}

func TestTools_questions(t *testing.T) {
	t.Run("MCPTL-B12: polvo_progress_answer responde a uma pergunta do Polvo", func(t *testing.T) {
		_, b, p := progressNode(t)
		callTool(t, p, "polvo_progress_start", map[string]any{"goal": "g"})
		q, err := b.Ask("session-abc", "onde vais?")
		if err != nil {
			t.Fatal(err)
		}
		r := callTool(t, p, "polvo_progress_answer", map[string]any{"question": q.ID, "text": "a meio"})
		got := job(t, b, "session-abc").Questions[0]
		if r.IsError || got.State != progress.Answered || got.Answer != "a meio" {
			t.Fatalf("got %+v and %+v", r, got)
		}
	})
	t.Run("MCPTL-B13: A conversa do Claude Code vai com o trabalho", func(t *testing.T) {
		addr, b, p := progressNode(t)
		home := t.TempDir()
		folder := filepath.Join(home, ".claude", "projects", "-p")
		os.MkdirAll(folder, 0o700)
		os.WriteFile(filepath.Join(folder, "a.jsonl"), []byte(`{"type":"assistant","message":{"model":"m"}}`+"\n"), 0o600)
		p.SetHere(Here{Node: addr, Session: "abc", Claude: &Claude{Home: home, Dir: "/p", ID: "a"}})
		callTool(t, p, "polvo_progress_start", map[string]any{"goal": "g"})
		conv := job(t, b, "session-abc").Conversation
		if conv == nil || conv.ID != "a" || conv.Dir != "/p" || conv.Kind != "claude" || conv.Model != "m" {
			t.Fatalf("got %+v", conv)
		}
	})
}

func sessionNode(t *testing.T) (string, *sessions.Manager, *Pool) {
	t.Helper()
	id := newSigner(t)
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	m := sessions.NewManager()
	go node.Serve(l, node.Config{HostKey: newSigner(t), Authorized: sshx.NewAuthorizedKeys(ak), Control: node.Controller{Sessions: m}.Serve})
	p := NewPool(hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
	t.Cleanup(p.Close)
	return l.Addr().String(), m, p
}

func newShell(t *testing.T, m *sessions.Manager, argv ...string) string {
	t.Helper()
	if len(argv) == 0 {
		argv = []string{"/bin/sh"}
	}
	info, err := m.Create(argv, "", pty.Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close(info.ID) })
	return info.ID
}

func TestTools_sessions(t *testing.T) {
	t.Run("MCPTL-B14: polvo_sessions lista as sessões de um nó", func(t *testing.T) {
		addr, m, p := sessionNode(t)
		id := newShell(t, m, "/bin/sh", "-c", "sleep 30")
		r := callTool(t, p, "polvo_sessions", map[string]any{"node": addr})
		if r.IsError || r.Text != id+"  /bin/sh -c sleep 30  80x24  you may type" {
			t.Fatalf("want the session's line, got %+v", r)
		}
	})
	t.Run("MCPTL-B15: O agente escreve numa sessão e espera pelo resultado", func(t *testing.T) {
		addr, m, p := sessionNode(t)
		id := newShell(t, m)
		typed := callTool(t, p, "polvo_type", map[string]any{"node": addr, "session": id, "text": "echo marca-$((40+2))", "keys": []string{"enter"}})
		waited := callTool(t, p, "polvo_wait", map[string]any{"node": addr, "session": id, "text": "marca-42", "timeout": 10})
		shown := callTool(t, p, "polvo_screen", map[string]any{"node": addr, "session": id})
		if typed.IsError || waited.IsError || !strings.Contains(waited.Text, "marca-42") || !strings.Contains(shown.Text, "marca-42") {
			t.Fatalf("want marca-42 waited for and shown, got %+v, %+v, %+v", typed, waited, shown)
		}
	})
	t.Run("MCPTL-E05: A espera acaba sem o texto", func(t *testing.T) {
		addr, m, p := sessionNode(t)
		id := newShell(t, m)
		r := callTool(t, p, "polvo_wait", map[string]any{"node": addr, "session": id, "text": "nunca-aparece", "timeout": 1})
		if !r.IsError || !strings.Contains(r.Text, "nunca-aparece") || !strings.Contains(r.Text, "it shows") {
			t.Fatalf("want the failure with the text and the screen, got %+v", r)
		}
	})
	t.Run("MCPTL-E01#02: Uma tecla que não existe", func(t *testing.T) {
		addr, m, p := sessionNode(t)
		id := newShell(t, m)
		r := callTool(t, p, "polvo_type", map[string]any{"node": addr, "session": id, "keys": []string{"hiper"}})
		if !r.IsError || !strings.Contains(r.Text, "hiper") {
			t.Fatalf("want the failure with the key, got %+v", r)
		}
	})
}

func TestTools_p2p(t *testing.T) {
	t.Run("MCPTL-B18: Os nós de outras redes", func(t *testing.T) {
		_, hk, _ := ed25519.GenerateKey(rand.Reader)
		far, err := transport.Open(hk, "", 0)
		if errors.Is(err, transport.ErrNotBuilt) {
			t.Skip("built without the transport (make test-p2p runs these)")
		}
		if err != nil {
			t.Fatal(err)
		}
		defer far.Close()
		host, _ := ssh.NewSignerFromKey(hk)
		id := newSigner(t)
		ak := filepath.Join(t.TempDir(), "authorized_keys")
		os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
		go node.Serve(far.Listener(), node.Config{HostKey: host, Authorized: sshx.NewAuthorizedKeys(ak), Control: node.ServeControl})
		_, ck, _ := ed25519.GenerateKey(rand.Reader)
		me, err := transport.Open(ck, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		defer me.Close()
		p := NewPool(hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
		defer p.Close()
		p.SetPeer(me)
		r := callTool(t, p, "polvo_run", map[string]any{"node": far.Addr(), "command": "echo do-agente"})
		if r.IsError || !strings.Contains(r.Text, "do-agente") {
			t.Fatalf("want do-agente, got %+v", r)
		}
	})
}
