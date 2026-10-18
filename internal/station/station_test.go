// @anchors
//   code: STTST
//   ref: STATN

package station

import (
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sshx"
)

var size = pty.Size{Rows: 24, Cols: 80}

var sh = Options{Shell: []string{"/bin/sh"}}

func open(t *testing.T, dir string, opt Options) *Station {
	t.Helper()
	s, err := Open(dir, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func nodeOf(t *testing.T, s *Station, name string) *hub.Node {
	t.Helper()
	n, err := s.Node(name)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// readFor reads r until it has want, or fails after a while.
func readFor(t *testing.T, r io.Reader, want string) {
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
	}
	if !strings.Contains(got.String(), want) {
		t.Fatalf("want %q, got %q", want, got.String())
	}
}

// authorize lets other's installation into s's node.
func authorize(t *testing.T, dir string, other *Station) {
	t.Helper()
	line := sshx.AuthorizedLine(other.PublicKey(), "teste")
	if err := os.WriteFile(filepath.Join(dir, "authorized_keys"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
}

// pair opens a first Station that listens and a second one it authorizes.
func pair(t *testing.T) (first *Station, firstDir string, second *Station, secondDir string) {
	firstDir, secondDir = t.TempDir(), t.TempDir()
	first = open(t, firstDir, Options{Listen: "127.0.0.1:0", Shell: sh.Shell})
	second = open(t, secondDir, sh)
	authorize(t, firstDir, second)
	return
}

func names(ms []Machine) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

func TestStation_local(t *testing.T) {
	t.Run("STATN-B01: A máquina desta janela", func(t *testing.T) {
		s, err := Open(t.TempDir(), sh)
		if err != nil {
			t.Fatal(err)
		}
		n := nodeOf(t, s, LOCAL)
		id, err := n.NewSession(nil, "", size)
		if err != nil {
			t.Fatal(err)
		}
		term, err := n.Attach(id)
		if err != nil {
			t.Fatal(err)
		}
		term.Write([]byte("echo $((40+2))\n"))
		readFor(t, term, "42")
		if err := n.LockSession(id, true); err != nil {
			t.Fatal(err)
		}
		list, _ := n.Sessions()
		if len(list) != 1 || !list[0].Locked || !list[0].Writable {
			t.Fatalf("want the session locked and still the owner's, got %+v", list)
		}
		mgr := s.sessions
		s.Close()
		end := time.Now().Add(5 * time.Second)
		for len(mgr.List()) != 0 && time.Now().Before(end) {
			time.Sleep(20 * time.Millisecond)
		}
		if len(mgr.List()) != 0 {
			t.Fatalf("want no session after the Close, got %v", mgr.List())
		}
	})
}

func TestStation_machines(t *testing.T) {
	t.Run("STATN-B02: As outras máquinas espelham esta", func(t *testing.T) {
		first, _, second, _ := pair(t)
		id, err := nodeOf(t, first, LOCAL).NewSession([]string{"/bin/sh", "-c", "sleep 30"}, "", size)
		if err != nil {
			t.Fatal(err)
		}
		if err := second.Add(first.ListenAddr()); err != nil {
			t.Fatal(err)
		}
		list, err := nodeOf(t, second, first.ListenAddr()).Sessions()
		if err != nil || len(list) != 1 || list[0].ID != id {
			t.Fatalf("want the first's session, got %+v %v", list, err)
		}
	})
	t.Run("STATN-B03: Juntar uma máquina", func(t *testing.T) {
		first, _, second, secondDir := pair(t)
		if err := second.Add(first.ListenAddr()); err != nil {
			t.Fatal(err)
		}
		second.Close()
		again := open(t, secondDir, sh)
		if got := names(again.Machines()); strings.Join(got, ",") != LOCAL+","+first.ListenAddr() {
			t.Fatalf("want local and the first, got %v", got)
		}
	})
	t.Run("STATN-B04: Esquecer uma máquina", func(t *testing.T) {
		first, _, second, secondDir := pair(t)
		if err := second.Add(first.ListenAddr()); err != nil {
			t.Fatal(err)
		}
		if err := second.Remove(first.ListenAddr()); err != nil {
			t.Fatal(err)
		}
		second.Close()
		again := open(t, secondDir, sh)
		if got := names(again.Machines()); strings.Join(got, ",") != LOCAL {
			t.Fatalf("want only local, got %v", got)
		}
	})
	t.Run("STATN-B05: Ligar outra vez quando caiu", func(t *testing.T) {
		first, firstDir, second, _ := pair(t)
		addr := first.ListenAddr()
		if err := second.Add(addr); err != nil {
			t.Fatal(err)
		}
		if _, err := nodeOf(t, second, addr).Sessions(); err != nil {
			t.Fatal(err)
		}
		first.Close()
		reopened := open(t, firstDir, Options{Listen: addr, Shell: sh.Shell})
		if err := reopened.ListenError(); err != nil {
			t.Fatal(err)
		}
		id, err := nodeOf(t, reopened, LOCAL).NewSession([]string{"/bin/sh", "-c", "sleep 30"}, "", size)
		if err != nil {
			t.Fatal(err)
		}
		var list []hub.Session
		end := time.Now().Add(5 * time.Second)
		for time.Now().Before(end) {
			if n, err := second.Node(addr); err == nil {
				if list, err = n.Sessions(); err == nil {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		if len(list) != 1 || list[0].ID != id {
			t.Fatalf("want the reopened first's session, got %+v", list)
		}
	})
	t.Run("STATN-E01: Um endereço ocupado", func(t *testing.T) {
		busy, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer busy.Close()
		s := open(t, t.TempDir(), Options{Listen: busy.Addr().String(), Shell: sh.Shell})
		if s.ListenError() == nil || s.ListenAddr() != "" {
			t.Fatalf("want the listen error, got %v %q", s.ListenError(), s.ListenAddr())
		}
		if _, err := nodeOf(t, s, LOCAL).Sessions(); err != nil {
			t.Fatalf("want the local node working, got %v", err)
		}
	})
	t.Run("STATN-E02: Uma máquina que não responde", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := l.Addr().String()
		l.Close()
		s := open(t, t.TempDir(), sh)
		err = s.Add(addr)
		if err == nil || !strings.Contains(err.Error(), addr) || strings.Join(names(s.Machines()), ",") != LOCAL {
			t.Fatalf("want the error with the address and only local, got %v %v", err, s.Machines())
		}
		if _, err := s.Node(addr); err == nil || errors.Is(err, nil) {
			t.Fatalf("want the unknown machine refused, got %v", err)
		}
	})
}

func TestStation_progress(t *testing.T) {
	t.Run("STATN-B06: O progresso dos agentes desta máquina", func(t *testing.T) {
		dir := t.TempDir()
		s := open(t, dir, sh)
		report := map[string]any{"agent": "claude-1", "op": "start", "goal": "migrar", "steps": []string{"ler"}}
		if err := nodeOf(t, s, LOCAL).Call("progress.report", report, &struct{}{}); err != nil {
			t.Fatal(err)
		}
		s.Close()
		again := open(t, dir, sh)
		var l struct {
			Agents []struct {
				Agent string `json:"agent"`
			} `json:"agents"`
		}
		if err := nodeOf(t, again, LOCAL).Call("progress.list", struct{}{}, &l); err != nil {
			t.Fatal(err)
		}
		if len(l.Agents) != 1 || l.Agents[0].Agent != "claude-1" {
			t.Fatalf("want the agent's job, got %+v", l)
		}
	})
}
