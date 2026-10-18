// @anchors
//   code: SSTSS
//   ref: SSSNS

package sessions

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/pty"
)

var size = pty.Size{Rows: 24, Cols: 80}

func sh(script string) []string { return []string{"/bin/sh", "-c", script} }

func create(t *testing.T, m *Manager, argv []string) Info {
	t.Helper()
	info, err := m.Create(argv, t.TempDir(), size)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close(info.ID) })
	return info
}

func attach(t *testing.T, m *Manager, id string, perm Perm) (*Viewer, *watch) {
	t.Helper()
	v, err := m.Attach(id, perm)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })
	return v, watchViewer(v)
}

// watch reads a viewer to its end in the background, for the test to wait on what arrives.
type watch struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	err  error
	done chan struct{}
}

func watchViewer(v *Viewer) *watch {
	w := &watch{done: make(chan struct{})}
	go func() {
		defer close(w.done)
		b := make([]byte, 32<<10)
		for {
			n, err := v.Read(b)
			w.mu.Lock()
			w.buf.Write(b[:n])
			if err != nil {
				w.err = err
			}
			w.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return w
}

func (w *watch) text() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *watch) waitFor(t *testing.T, want string) string {
	t.Helper()
	return w.waitForIn(t, want, 10*time.Second)
}

func (w *watch) waitForIn(t *testing.T, want string, d time.Duration) string {
	t.Helper()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if s := w.text(); strings.Contains(s, want) {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want %q, got %q", want, w.text())
	return ""
}

func (w *watch) waitEnd(t *testing.T) (string, error) {
	t.Helper()
	select {
	case <-w.done:
	case <-time.After(10 * time.Second):
		t.Fatalf("the viewer did not end; got %q", w.text())
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String(), w.err
}

func ids(list []Info) []string {
	var out []string
	for _, i := range list {
		out = append(out, i.ID)
	}
	return out
}

func TestSessions_create(t *testing.T) {
	t.Run("SSSNS-B01: Criar uma sessão", func(t *testing.T) {
		m := NewManager()
		dir := t.TempDir()
		a, err := m.Create(sh("echo $TERM; pwd; sleep 5"), dir, size)
		if err != nil {
			t.Fatal(err)
		}
		defer m.Close(a.ID)
		b := create(t, m, sh("sleep 5"))
		_, w := attach(t, m, a.ID, View)
		out := w.waitFor(t, dir)
		if a.ID == b.ID || !strings.Contains(out, "xterm-256color") {
			t.Fatalf("want two ids and xterm-256color, got %q %q and %q", a.ID, b.ID, out)
		}
	})
	t.Run("SSSNS-B02: Listar as sessões", func(t *testing.T) {
		m := NewManager()
		a := create(t, m, sh("sleep 5"))
		b := create(t, m, sh("sleep 5"))
		attach(t, m, b.ID, View)
		list := m.List()
		if strings.Join(ids(list), ",") != a.ID+","+b.ID {
			t.Fatalf("want %s,%s, got %v", a.ID, b.ID, ids(list))
		}
		second := list[1]
		if second.Argv[0] != "/bin/sh" || second.Size != size || second.Viewers != 1 || second.Locked {
			t.Fatalf("want /bin/sh, %v, 1 viewer, unlocked; got %+v", size, second)
		}
	})
	t.Run("SSSNS-B11: O ambiente de uma sessão", func(t *testing.T) {
		m := NewManager()
		m.SetEnv([]string{"POLVO_NODE=127.0.0.1:2222"})
		s := create(t, m, sh("echo sessao=$POLVO_SESSION no=$POLVO_NODE; sleep 5"))
		_, w := attach(t, m, s.ID, View)
		w.waitFor(t, "sessao="+s.ID+" no=127.0.0.1:2222")
	})
	t.Run("SSSNS-E01: Um programa que não arranca não faz sessão", func(t *testing.T) {
		m := NewManager()
		_, err := m.Create([]string{"/nao/existe/shell"}, t.TempDir(), size)
		if err == nil || !strings.Contains(err.Error(), "/nao/existe/shell") || len(m.List()) != 0 {
			t.Fatalf("want the error with the program and no session, got %v and %v", err, m.List())
		}
	})
	t.Run("SSSNS-E02: Um id que não é de nenhuma sessão", func(t *testing.T) {
		m := NewManager()
		_, a := m.Attach("nenhuma", View)
		l, c := m.Lock("nenhuma", true), m.Close("nenhuma")
		if !errors.Is(a, ErrNoSession) || !errors.Is(l, ErrNoSession) || !errors.Is(c, ErrNoSession) {
			t.Fatalf("want ErrNoSession three times, got %v, %v, %v", a, l, c)
		}
	})
}

func TestSessions_viewers(t *testing.T) {
	t.Run("SSSNS-B03: Quem se liga recebe o ecrã atual e depois a saída ao vivo", func(t *testing.T) {
		m := NewManager()
		s := create(t, m, []string{"/bin/sh"})
		first, w1 := attach(t, m, s.ID, Control)
		// a mark only the result has: the echo of the line and the prompt do not make it
		first.Write([]byte("echo marca-$((0+1))\n"))
		w1.waitFor(t, "marca-1")
		_, w2 := attach(t, m, s.ID, View)
		w2.waitFor(t, "marca-1")
		first.Write([]byte("echo $((40+2))\n"))
		out := w2.waitFor(t, "\n42")
		if strings.Count(out, "\n42") != 1 {
			t.Fatalf("want 42 once, got %q", out)
		}
	})
	t.Run("SSSNS-B04: A saída chega a todos, e o ecrã acompanha sem espectadores", func(t *testing.T) {
		m := NewManager()
		s := create(t, m, sh("sleep 0.3; echo tarde; sleep 5"))
		time.Sleep(time.Second)
		_, w1 := attach(t, m, s.ID, View)
		_, w2 := attach(t, m, s.ID, View)
		w1.waitFor(t, "tarde")
		w2.waitFor(t, "tarde")
	})
	t.Run("SSSNS-B05: Controlar é escrever e mudar o tamanho, e vale o último tamanho", func(t *testing.T) {
		m := NewManager()
		s := create(t, m, []string{"/bin/sh"})
		a, w := attach(t, m, s.ID, Control)
		b, _ := attach(t, m, s.ID, Control)
		if err := a.Resize(pty.Size{Rows: 30, Cols: 100}); err != nil {
			t.Fatal(err)
		}
		if err := b.Resize(pty.Size{Rows: 20, Cols: 60}); err != nil {
			t.Fatal(err)
		}
		a.Write([]byte("stty size\n"))
		w.waitFor(t, "20 60")
	})
	t.Run("SSSNS-B06: Só ver não escreve", func(t *testing.T) {
		m := NewManager()
		s := create(t, m, []string{"/bin/sh"})
		v, _ := attach(t, m, s.ID, View)
		_, werr := v.Write([]byte("echo nao\n"))
		rerr := v.Resize(pty.Size{Rows: 10, Cols: 10})
		if !errors.Is(werr, ErrViewOnly) || !errors.Is(rerr, ErrViewOnly) {
			t.Fatalf("want ErrViewOnly twice, got %v and %v", werr, rerr)
		}
	})
	t.Run("SSSNS-B07: A sessão travada só deixa ver, menos ao dono", func(t *testing.T) {
		m := NewManager()
		s := create(t, m, []string{"/bin/sh"})
		if err := m.Lock(s.ID, true); err != nil {
			t.Fatal(err)
		}
		guest, w := attach(t, m, s.ID, Control)
		owner, _ := attach(t, m, s.ID, Owner)
		_, gerr := guest.Write([]byte("echo convidado\n"))
		if _, err := owner.Write([]byte("echo dono-$((1+1))\n")); err != nil {
			t.Fatal(err)
		}
		w.waitFor(t, "dono-2")
		locked := m.List()[0].Locked
		m.Lock(s.ID, false)
		if _, err := guest.Write([]byte("echo livre-$((2+2))\n")); err != nil {
			t.Fatal(err)
		}
		w.waitFor(t, "livre-4")
		if !errors.Is(gerr, ErrViewOnly) || !locked || m.List()[0].Locked {
			t.Fatalf("want the guest refused while locked and the lock listed, got %v, %v, %v", gerr, locked, m.List()[0].Locked)
		}
	})
	t.Run("SSSNS-B10: As respostas do ecrã do nó não chegam ao programa", func(t *testing.T) {
		m := NewManager()
		script := `printf '\033[c'; sleep 1; stty -echo -icanon min 0 time 0; x=$(dd bs=64 count=1 2>/dev/null); echo "got:[$x]"; sleep 5`
		s := create(t, m, sh(script))
		_, w := attach(t, m, s.ID, View)
		out := w.waitFor(t, "got:[")
		if !strings.Contains(out, "got:[]") {
			t.Fatalf("want nothing back to the program, got %q", out)
		}
	})
	t.Run("SSSNS-E04: Um espectador que fica para trás é desligado", func(t *testing.T) {
		m := NewManager()
		m.SetBacklog(4 << 10)
		// bursts that each fit in the backlog of a viewer that reads, and together pass that of one that
		// does not: a single burst larger than the backlog could leave behind even a viewer that reads
		s := create(t, m, sh("sleep 0.5; for i in 1 2 3; do head -c 3000 /dev/zero | tr '\\0' x; sleep 0.3; done; echo; echo acabou; sleep 5"))
		slow, err := m.Attach(s.ID, View)
		if err != nil {
			t.Fatal(err)
		}
		defer slow.Close()
		_, fast := attach(t, m, s.ID, View)
		fast.waitForIn(t, "acabou", time.Minute)
		_, rerr := io.ReadAll(slow)
		if !errors.Is(rerr, ErrTooSlow) {
			t.Fatalf("want ErrTooSlow for the viewer that did not read, got %v", rerr)
		}
	})
}

func TestSessions_end(t *testing.T) {
	t.Run("SSSNS-B08: O fim do programa acaba a sessão", func(t *testing.T) {
		m := NewManager()
		s, err := m.Create(sh("sleep 0.3; echo fim; exit 3"), t.TempDir(), size)
		if err != nil {
			t.Fatal(err)
		}
		v, w := attach(t, m, s.ID, View)
		out, rerr := w.waitEnd(t)
		st, ended := v.Exit()
		_, again := m.Attach(s.ID, View)
		if !strings.Contains(out, "fim") || rerr != io.EOF || !ended || st.Code != 3 || len(m.List()) != 0 || !errors.Is(again, ErrNoSession) {
			t.Fatalf("want fim, EOF, exit 3, no session; got %q, %v, %v %+v, %v, %v", out, rerr, ended, st, m.List(), again)
		}
	})
	t.Run("SSSNS-B09: Terminar uma sessão", func(t *testing.T) {
		m := NewManager()
		s, err := m.Create(sh("sleep 30"), t.TempDir(), size)
		if err != nil {
			t.Fatal(err)
		}
		_, w := attach(t, m, s.ID, View)
		if err := m.Close(s.ID); err != nil {
			t.Fatal(err)
		}
		w.waitEnd(t)
		if len(m.List()) != 0 {
			t.Fatalf("want no session, got %v", m.List())
		}
	})
}
