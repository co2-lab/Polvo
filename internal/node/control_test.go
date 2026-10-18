// @anchors
//   code: CNTST
//   ref: CNTRL

package node

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/progress"
	"github.com/co2-lab/polvo/internal/proto"
	"github.com/co2-lab/polvo/internal/pty"
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
	return controlWith(t, ServeControl, sessions.Owner)
}

// controlWith serves serve on one end of a pipe with perm, and returns the other.
func controlWith(t *testing.T, serve func(io.ReadWriteCloser, sessions.Perm), perm sessions.Perm) peer {
	t.Helper()
	a, b := net.Pipe()
	go serve(a, perm)
	t.Cleanup(func() { b.Close() })
	return peer{enc: proto.NewEncoder(b), dec: proto.NewDecoder(b), raw: b}
}

type sessionList struct {
	Sessions []struct {
		ID       string   `json:"id"`
		Argv     []string `json:"argv"`
		Rows     int      `json:"rows"`
		Locked   bool     `json:"locked"`
		Writable bool     `json:"writable"`
	} `json:"sessions"`
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

func TestControl_sessions(t *testing.T) {
	newSession := map[string]any{"argv": []string{"/bin/sh", "-c", "sleep 5"}, "rows": 24, "cols": 80}
	t.Run("CNTRL-B07#01: Criar, listar e terminar uma sessão pelo controlo", func(t *testing.T) {
		c := Controller{Sessions: sessions.NewManager()}
		p := controlWith(t, c.Serve, sessions.Owner)
		var made struct{ ID string }
		result(t, p.call(t, 1, "session.new", newSession), &made)
		var l sessionList
		result(t, p.call(t, 2, "session.list", struct{}{}), &l)
		if len(l.Sessions) != 1 || l.Sessions[0].ID != made.ID || l.Sessions[0].Argv[2] != "sleep 5" || l.Sessions[0].Rows != 24 {
			t.Fatalf("want the session %s, got %+v", made.ID, l)
		}
		result(t, p.call(t, 3, "session.close", map[string]string{"id": made.ID}), &struct{}{})
		end := time.Now().Add(5 * time.Second)
		for len(c.Sessions.List()) != 0 && time.Now().Before(end) {
			time.Sleep(20 * time.Millisecond)
		}
		if len(c.Sessions.List()) != 0 {
			t.Fatalf("want no session, got %v", c.Sessions.List())
		}
	})
	t.Run("CNTRL-B07#02: O dono trava uma sessão", func(t *testing.T) {
		c := Controller{Sessions: sessions.NewManager()}
		p := controlWith(t, c.Serve, sessions.Owner)
		var made struct{ ID string }
		result(t, p.call(t, 1, "session.new", newSession), &made)
		defer c.Sessions.Close(made.ID)
		result(t, p.call(t, 2, "session.lock", map[string]any{"id": made.ID, "locked": true}), &struct{}{})
		var l sessionList
		result(t, p.call(t, 3, "session.list", struct{}{}), &l)
		if !l.Sessions[0].Locked {
			t.Fatalf("want it locked, got %+v", l)
		}
	})
	t.Run("CNTRL-B07#04: A lista diz a cada ligação se pode escrever", func(t *testing.T) {
		c := Controller{Sessions: sessions.NewManager()}
		free, err := c.Sessions.Create([]string{"/bin/sh", "-c", "sleep 5"}, "", pty.Size{Rows: 24, Cols: 80})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Sessions.Close(free.ID)
		locked, err := c.Sessions.Create([]string{"/bin/sh", "-c", "sleep 5"}, "", pty.Size{Rows: 24, Cols: 80})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Sessions.Close(locked.ID)
		c.Sessions.Lock(locked.ID, true)
		var got []bool
		for i, perm := range []sessions.Perm{sessions.View, sessions.Control, sessions.Owner} {
			var l sessionList
			result(t, controlWith(t, c.Serve, perm).call(t, int64(i+1), "session.list", struct{}{}), &l)
			for _, s := range l.Sessions {
				got = append(got, s.Writable)
			}
		}
		if fmt.Sprint(got) != "[false false true false true true]" {
			t.Fatalf("want view nothing, control the free one, owner both; got %v", got)
		}
	})
	t.Run("CNTRL-B07#03: Sem sessões, os pedidos de sessão dão erro", func(t *testing.T) {
		wantError(t, control(t).call(t, 1, "session.list", struct{}{}), CodeOperation, "sessions")
	})
	t.Run("CNTRL-B08: O que cada permissão pode fazer", func(t *testing.T) {
		c := Controller{Sessions: sessions.NewManager()}
		dir := t.TempDir()
		f := filepath.Join(dir, "a.txt")
		os.WriteFile(f, []byte("ola"), 0o644)
		info, err := c.Sessions.Create([]string{"/bin/sh", "-c", "sleep 5"}, dir, pty.Size{Rows: 24, Cols: 80})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Sessions.Close(info.ID)
		v := controlWith(t, c.Serve, sessions.View)
		var read struct{ Data string }
		result(t, v.call(t, 1, "fs.read", map[string]string{"path": f}), &read)
		var l sessionList
		result(t, v.call(t, 2, "session.list", struct{}{}), &l)
		wantError(t, v.call(t, 3, "fs.write", map[string]string{"path": f, "data": b64("x")}), CodeForbidden, "permission")
		wantError(t, v.call(t, 4, "proc.run", map[string]any{"argv": []string{"true"}}), CodeForbidden, "permission")
		wantError(t, v.call(t, 5, "session.new", newSession), CodeForbidden, "permission")
		ctl := controlWith(t, c.Serve, sessions.Control)
		wantError(t, ctl.call(t, 6, "session.lock", map[string]any{"id": info.ID, "locked": true}), CodeForbidden, "permission")
		if unb64(t, read.Data) != "ola" || len(l.Sessions) != 1 {
			t.Fatalf("want the read and the list, got %q and %+v", read.Data, l)
		}
		if b, _ := os.ReadFile(f); string(b) != "ola" {
			t.Fatalf("want the file untouched, got %q", b)
		}
	})
}

func TestControl_progress(t *testing.T) {
	newBoard := func(t *testing.T) *progress.Board {
		b, err := progress.NewBoard(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	type job struct {
		Agent    string `json:"agent"`
		Finished bool   `json:"finished"`
		Steps    []struct {
			State string `json:"state"`
		} `json:"steps"`
		Log []struct {
			Text string `json:"text"`
		} `json:"log"`
	}
	type list struct {
		Version uint64 `json:"version"`
		Agents  []job  `json:"agents"`
	}
	t.Run("CNTRL-B09#01: O agente reporta e quem acompanha lê", func(t *testing.T) {
		p := controlWith(t, Controller{Progress: newBoard(t)}.Serve, sessions.Control)
		result(t, p.call(t, 1, "progress.report", map[string]any{"agent": "a", "op": "start", "goal": "migrar", "steps": []string{"ler"}}), &struct{}{})
		result(t, p.call(t, 2, "progress.report", map[string]any{"agent": "a", "op": "step", "step": "ler", "state": "done"}), &struct{}{})
		result(t, p.call(t, 3, "progress.report", map[string]any{"agent": "a", "op": "note", "text": "uma nota"}), &struct{}{})
		result(t, p.call(t, 4, "progress.report", map[string]any{"agent": "a", "op": "finish", "text": "feito"}), &struct{}{})
		var got job
		result(t, p.call(t, 5, "progress.get", map[string]string{"agent": "a"}), &got)
		var l list
		result(t, p.call(t, 6, "progress.list", struct{}{}), &l)
		if !got.Finished || got.Steps[0].State != "done" || len(got.Log) != 4 || len(l.Agents) != 1 || l.Version != 4 {
			t.Fatalf("got %+v and %+v", got, l)
		}
	})
	t.Run("CNTRL-B09#02: Esperar pelo progresso", func(t *testing.T) {
		b := newBoard(t)
		b.Start("a", "g", nil)
		p := controlWith(t, Controller{Progress: b}.Serve, sessions.View)
		go func() {
			time.Sleep(100 * time.Millisecond)
			b.Note("a", "chegou")
		}()
		var l list
		result(t, p.call(t, 1, "progress.wait", map[string]any{"since": b.Version(), "timeout_ms": 5000}), &l)
		if l.Version != 2 || l.Agents[0].Log[1].Text != "chegou" {
			t.Fatalf("want the note, got %+v", l)
		}
	})
	t.Run("CNTRL-B09#03: Quem só vê não reporta, e sem quadro há erro", func(t *testing.T) {
		v := controlWith(t, Controller{Progress: newBoard(t)}.Serve, sessions.View)
		wantError(t, v.call(t, 1, "progress.report", map[string]any{"agent": "a", "op": "start"}), CodeForbidden, "permission")
		wantError(t, control(t).call(t, 2, "progress.list", struct{}{}), CodeOperation, "progress")
	})
}
