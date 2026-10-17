// @anchors
//   code: CNTST
//   ref: CNTRL

package node

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/proto"
	"github.com/co2-lab/polvo/internal/sessions"
)

type peer struct {
	enc *proto.Encoder
	dec *proto.Decoder
	raw net.Conn
}

// control serves the control on one end of a pipe and returns the other.
func control(t *testing.T) peer {
	t.Helper()
	a, b := net.Pipe()
	go ServeControl(a, sessions.Owner)
	t.Cleanup(func() { b.Close() })
	return peer{enc: proto.NewEncoder(b), dec: proto.NewDecoder(b), raw: b}
}

func (p peer) call(t *testing.T, id int64, method string, params any) *proto.Message {
	t.Helper()
	p.send(t, id, method, params)
	return p.next(t)
}

func (p peer) send(t *testing.T, id int64, method string, params any) {
	t.Helper()
	b, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	m := &proto.Message{Method: method, Params: b}
	if id != 0 {
		m.ID = proto.IntID(id)
	}
	go p.enc.Write(m)
}

func (p peer) next(t *testing.T) *proto.Message {
	t.Helper()
	m, err := p.dec.Read()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func result(t *testing.T, m *proto.Message, v any) {
	t.Helper()
	if m.Error != nil {
		t.Fatalf("want a result, got error %d %s", m.Error.Code, m.Error.Message)
	}
	if err := json.Unmarshal(m.Result, v); err != nil {
		t.Fatal(err)
	}
}

func wantError(t *testing.T, m *proto.Message, code int, contains string) {
	t.Helper()
	if m.Error == nil || m.Error.Code != code || !strings.Contains(m.Error.Message, contains) {
		t.Fatalf("want error %d with %q, got %+v", code, contains, m.Error)
	}
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func unb64(t *testing.T, s string) string {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestControl_files(t *testing.T) {
	t.Run("CNTRL-B01: fs.read devolve o conteúdo e o tamanho", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "a.txt")
		os.WriteFile(p, []byte("ola polvo"), 0o644)
		var r struct {
			Data string `json:"data"`
			Size int64  `json:"size"`
		}
		result(t, control(t).call(t, 1, "fs.read", map[string]string{"path": p}), &r)
		if unb64(t, r.Data) != "ola polvo" || r.Size != 9 {
			t.Fatalf("want ola polvo and 9, got %+v", r)
		}
	})
	t.Run("CNTRL-B02: fs.write cria o ficheiro com 0644 e depois substitui-o", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "b.txt")
		c := control(t)
		for i, s := range []string{"um", "dois"} {
			var r struct{}
			result(t, c.call(t, int64(i+1), "fs.write", map[string]string{"path": p, "data": b64(s)}), &r)
		}
		b, _ := os.ReadFile(p)
		st, _ := os.Stat(p)
		if string(b) != "dois" || st.Mode().Perm() != 0o644 {
			t.Fatalf("want dois and 0644, got %q %o", b, st.Mode().Perm())
		}
	})
	t.Run("CNTRL-B03: fs.list devolve as entradas por ordem de nome", func(t *testing.T) {
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "z.txt"), []byte("zzz"), 0o644)
		os.Mkdir(filepath.Join(d, "m"), 0o755)
		os.WriteFile(filepath.Join(d, "a.txt"), []byte("a"), 0o644)
		var r struct {
			Entries []struct {
				Name string `json:"name"`
				Dir  bool   `json:"dir"`
				Size int64  `json:"size"`
			} `json:"entries"`
		}
		result(t, control(t).call(t, 1, "fs.list", map[string]string{"path": d}), &r)
		e := r.Entries
		if len(e) != 3 || e[0].Name != "a.txt" || e[0].Size != 1 || e[1].Name != "m" || !e[1].Dir || e[2].Name != "z.txt" || e[2].Size != 3 {
			t.Fatalf("want a.txt, m/, z.txt; got %+v", e)
		}
	})
	t.Run("CNTRL-E02: Um fs.write com base64 inválido dá parâmetros inválidos", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "c.txt")
		wantError(t, control(t).call(t, 1, "fs.write", map[string]string{"path": p, "data": "!!!"}), proto.CodeInvalidParams, "")
		if _, err := os.Stat(p); err == nil {
			t.Fatal("want no file")
		}
	})
	t.Run("CNTRL-E03: Ler um ficheiro que não existe dá erro de operação", func(t *testing.T) {
		wantError(t, control(t).call(t, 1, "fs.read", map[string]string{"path": "/nao/existe"}), CodeOperation, "/nao/existe")
	})
	t.Run("CNTRL-E04: Um ficheiro maior do que 8 MiB não é lido", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "grande")
		f, _ := os.Create(p)
		f.Truncate(MaxFile + 1)
		f.Close()
		wantError(t, control(t).call(t, 1, "fs.read", map[string]string{"path": p}), CodeOperation, "too large")
	})
}

type runResult struct {
	Code   int    `json:"code"`
	Signal string `json:"signal"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

func TestControl_processes(t *testing.T) {
	t.Run("CNTRL-B04: proc.run devolve o código, a saída e o erro separados", func(t *testing.T) {
		var r runResult
		result(t, control(t).call(t, 1, "proc.run", map[string]any{
			"argv": []string{"sh", "-c", "cat; echo erro >&2; exit 5"}, "stdin": b64("entrada"),
		}), &r)
		if r.Code != 5 || unb64(t, r.Stdout) != "entrada" || unb64(t, r.Stderr) != "erro\n" {
			t.Fatalf("want 5, entrada, erro; got %+v", r)
		}
	})
	t.Run("CNTRL-B04: proc.run corre na pasta pedida", func(t *testing.T) {
		var r runResult
		result(t, control(t).call(t, 1, "proc.run", map[string]any{"argv": []string{"pwd"}, "dir": "/tmp"}), &r)
		if unb64(t, r.Stdout) != "/tmp\n" {
			t.Fatalf("want /tmp, got %q", unb64(t, r.Stdout))
		}
	})
	t.Run("CNTRL-B05: Um sleep lento não atrasa um fs.list que chega depois", func(t *testing.T) {
		c := control(t)
		c.send(t, 1, "proc.run", map[string]any{"argv": []string{"sleep", "2"}})
		time.Sleep(100 * time.Millisecond)
		c.send(t, 2, "fs.list", map[string]string{"path": t.TempDir()})
		if n, _ := c.next(t).ID.Int(); n != 2 {
			t.Fatalf("want the answer to 2 first, got %d", n)
		}
		if n, _ := c.next(t).ID.Int(); n != 1 {
			t.Fatalf("want the answer to 1 next, got %d", n)
		}
	})
	t.Run("CNTRL-B06: Uma notificação fs.read não tem resposta", func(t *testing.T) {
		c := control(t)
		c.send(t, 0, "fs.read", map[string]string{"path": "/nao/existe"})
		time.Sleep(50 * time.Millisecond)
		c.send(t, 7, "fs.list", map[string]string{"path": t.TempDir()})
		if n, _ := c.next(t).ID.Int(); n != 7 {
			t.Fatalf("want only the answer to 7, got %d", n)
		}
	})
	t.Run("CNTRL-E01: Um método desconhecido dá método não encontrado", func(t *testing.T) {
		wantError(t, control(t).call(t, 1, "fs.apagatudo", map[string]string{}), proto.CodeMethodNotFound, "")
	})
	t.Run("CNTRL-E05: Uma linha que não é mensagem tem resposta de erro e a sessão continua", func(t *testing.T) {
		c := control(t)
		go io.WriteString(c.raw, "isto nao e json\n")
		if m := c.next(t); m.Error == nil || m.Error.Code != proto.CodeParseError {
			t.Fatalf("want the parse error, got %+v", m)
		}
		var r struct{}
		result(t, c.call(t, 2, "fs.list", map[string]string{"path": t.TempDir()}), &r)
	})
}
