// @anchors
//   code: MCTST
//   ref: MCPTL

package mcp

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/sshx"
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
	t.Run("MCPTL-E01: Sem o comando, polvo_run é uma falha", func(t *testing.T) {
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
