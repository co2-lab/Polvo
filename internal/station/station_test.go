// @anchors
//   code: STTST
//   ref: STATN

package station

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/discovery"
	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sshx"
	"github.com/co2-lab/polvo/internal/transport"
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
	t.Run("STATN-B12: Aceitar as outras máquinas, e deixar de aceitar", func(t *testing.T) {
		first, firstDir, second, _ := pair(t)
		addr := first.ListenAddr()
		if err := second.Add(addr); err != nil {
			t.Fatal(err)
		}
		if _, err := nodeOf(t, second, addr).Sessions(); err != nil {
			t.Fatal(err)
		}
		if err := first.Listen(""); err != nil {
			t.Fatal(err)
		}
		stopped := first.ListenAddr()
		n, _ := second.Node(addr)
		var cut error
		end := time.Now().Add(5 * time.Second)
		for cut == nil && time.Now().Before(end) {
			if n == nil {
				break
			}
			_, cut = n.Sessions()
			time.Sleep(20 * time.Millisecond)
		}
		if err := first.Listen(addr); err != nil {
			t.Fatal(err)
		}
		if _, err := second.Node(addr); stopped != "" || n != nil && cut == nil || err != nil || first.ListenAddr() != addr {
			t.Fatalf("want the other cut while stopped and back after, got %q, %v, %v, %q", stopped, cut, err, first.ListenAddr())
		}
		first.Close()
		again := open(t, firstDir, Options{Shell: sh.Shell})
		if again.ListenAddr() != addr {
			t.Fatalf("want the reopened one listening where it was left, %s, got %q", addr, again.ListenAddr())
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

func TestStation_env(t *testing.T) {
	t.Run("STATN-B07: As sessões sabem onde está o nó", func(t *testing.T) {
		dir := t.TempDir()
		s := open(t, dir, sh)
		n := nodeOf(t, s, LOCAL)
		id, err := n.NewSession([]string{"/bin/sh", "-c", "echo no=$POLVO_NODE.; sleep 30"}, "", size)
		if err != nil {
			t.Fatal(err)
		}
		term, err := n.Attach(id)
		if err != nil {
			t.Fatal(err)
		}
		defer term.Close()
		var got bytes.Buffer
		buf := make([]byte, 4096)
		end := time.Now().Add(10 * time.Second)
		for !strings.Contains(got.String(), ".") || !strings.Contains(got.String(), "no=") {
			if time.Now().After(end) {
				t.Fatalf("no address, got %q", got.String())
			}
			k, err := term.Read(buf)
			got.Write(buf[:k])
			if err != nil {
				break
			}
		}
		addr := strings.TrimSuffix(strings.SplitN(strings.SplitN(got.String(), "no=", 2)[1], ".\r", 2)[0], ".")
		identity, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
		if err != nil {
			t.Fatal(err)
		}
		agent, err := hub.Dial(addr, hub.Config{Identity: identity, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))})
		if err != nil {
			t.Fatalf("connecting to %q: %v", addr, err)
		}
		defer agent.Close()
		if list, err := agent.Sessions(); err != nil || len(list) != 1 || list[0].ID != id {
			t.Fatalf("want the session, got %+v %v", list, err)
		}
	})
}

// testGroup is the mDNS group on the loopback at a free port, apart from the machine's mDNS.
func testGroup(t *testing.T) *discovery.Options {
	t.Helper()
	lo, err := net.InterfaceByName("lo")
	if err != nil {
		t.Skip("no loopback interface")
	}
	free, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	port := free.LocalAddr().(*net.UDPAddr).Port
	free.Close()
	return &discovery.Options{Group: &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: port}, Interface: lo}
}

// codeShownBy waits for the code a Station shows.
func codeShownBy(s *Station) func() (string, error) {
	return func() (string, error) {
		end := time.Now().Add(5 * time.Second)
		for time.Now().Before(end) {
			if ps := s.Pairings(); len(ps) > 0 {
				return ps[0].Code, nil
			}
			time.Sleep(10 * time.Millisecond)
		}
		return "", errors.New("no code shown")
	}
}

func TestStation_pairing(t *testing.T) {
	t.Run("STATN-B08: O nó desta máquina emparelha", func(t *testing.T) {
		first := open(t, t.TempDir(), Options{Listen: "127.0.0.1:0", Shell: sh.Shell})
		other := open(t, t.TempDir(), sh)
		var shown []string
		err := other.Pair(first.ListenAddr(), func() (string, error) {
			code, err := codeShownBy(first)()
			for _, p := range first.Pairings() {
				shown = append(shown, p.Name, p.Fingerprint)
			}
			return code, err
		})
		if err != nil || len(shown) != 2 || shown[1] != ssh.FingerprintSHA256(other.PublicKey()) || len(first.Pairings()) != 0 {
			t.Fatalf("want the request shown with the other's key, and gone after; got %v %v %v", err, shown, first.Pairings())
		}
	})
	t.Run("STATN-B11: Emparelhar com uma máquina", func(t *testing.T) {
		first := open(t, t.TempDir(), Options{Listen: "127.0.0.1:0", Shell: sh.Shell})
		other := open(t, t.TempDir(), sh)
		if err := other.Pair(first.ListenAddr(), codeShownBy(first)); err != nil {
			t.Fatal(err)
		}
		names := names(other.Machines())
		if _, err := nodeOf(t, other, first.ListenAddr()).Sessions(); err != nil || len(names) != 2 || names[1] != first.ListenAddr() {
			t.Fatalf("want the first among the machines, its sessions listed; got %v %v", names, err)
		}
	})
}

func TestStation_discovery(t *testing.T) {
	t.Run("STATN-B09#01: Anunciar-se na rede", func(t *testing.T) {
		g := testGroup(t)
		s := open(t, t.TempDir(), Options{Listen: ":0", Shell: sh.Shell, Name: "oficina", Discovery: g})
		found, err := discovery.Browse(context.Background(), 500*time.Millisecond, *g)
		_, port, _ := net.SplitHostPort(s.ListenAddr())
		if err != nil || len(found) != 1 || found[0].Name != "oficina" || found[0].Fingerprint != s.HostFingerprint() || !strings.HasSuffix(found[0].Addr, ":"+port) {
			t.Fatalf("got %+v %v", found, err)
		}
	})
	t.Run("STATN-B09#02: No loopback não se anuncia", func(t *testing.T) {
		g := testGroup(t)
		open(t, t.TempDir(), Options{Listen: "127.0.0.1:0", Shell: sh.Shell, Name: "so-aqui", Discovery: g})
		found, err := discovery.Browse(context.Background(), 300*time.Millisecond, *g)
		if err != nil || len(found) != 0 {
			t.Fatalf("want nothing announced, got %+v %v", found, err)
		}
	})
	t.Run("STATN-B10: Procurar as máquinas da rede", func(t *testing.T) {
		g := testGroup(t)
		firstDir := t.TempDir()
		first := open(t, firstDir, Options{Listen: ":0", Shell: sh.Shell, Name: "primeira", Discovery: g})
		second := open(t, t.TempDir(), Options{Listen: ":0", Shell: sh.Shell, Name: "segunda", Discovery: g})
		authorize(t, firstDir, second)
		_, port, _ := net.SplitHostPort(first.ListenAddr())
		if err := second.Add("127.0.0.1:" + port); err != nil {
			t.Fatal(err)
		}
		found, err := second.Discover(500 * time.Millisecond)
		if err != nil || len(found) != 1 || found[0].Name != "primeira" || !found[0].Known {
			t.Fatalf("want only the first, known; got %+v %v", found, err)
		}
	})
}

// p2pPair is pair, with both reaching each other between networks; skipped without the transport.
func p2pPair(t *testing.T) (first *Station, firstDir string, second *Station, secondDir string) {
	t.Helper()
	first, firstDir, second, secondDir = pair(t)
	if first.P2PAddr() == "" {
		t.Skip("built without the transport (make test-p2p runs these)")
	}
	return
}

func TestStation_p2p(t *testing.T) {
	t.Run("STATN-B13: As máquinas de outras redes, pela chave", func(t *testing.T) {
		first, _, second, _ := p2pPair(t)
		id := first.mustID(t)
		name, err := second.AddP2P(first.P2PAddr())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := nodeOf(t, second, name).Sessions(); err != nil {
			t.Fatal(err)
		}
		first.Listen("")
		cut := false
		for end := time.Now().Add(10 * time.Second); !cut && time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
			if n, err := second.Node(name); err != nil {
				cut = true
			} else if _, err := n.Sessions(); err != nil {
				cut = true
			}
		}
		if name != P2P_PREFIX+id || !cut {
			t.Fatalf("want %s and the connection refused while not accepting, got %s and %v", P2P_PREFIX+id, name, cut)
		}
	})
	t.Run("STATN-B15: O histórico dos endereços de cada máquina", func(t *testing.T) {
		first, _, second, secondDir := p2pPair(t)
		name, err := second.AddP2P(first.P2PAddr())
		if err != nil {
			t.Fatal(err)
		}
		second.Close()
		again := open(t, secondDir, sh)
		history := again.addrOf(strings.TrimPrefix(name, P2P_PREFIX))
		if _, err := nodeOf(t, again, name).Sessions(); err != nil || !strings.Contains(history, "Ip") {
			t.Fatalf("want the sessions by the history %s, got %v", history, err)
		}
	})
}

// mustID is the key in this installation's p2p address.
func (s *Station) mustID(t *testing.T) string {
	t.Helper()
	var e endpoint
	if err := json.Unmarshal([]byte(s.P2PAddr()), &e); err != nil || e.ID == "" {
		t.Fatalf("no p2p address: %q", s.P2PAddr())
	}
	return e.ID
}

func TestStation_noP2P(t *testing.T) {
	t.Run("STATN-E03: Uma máquina pela chave, sem o transporte", func(t *testing.T) {
		s := open(t, t.TempDir(), sh)
		if s.P2PAddr() != "" {
			t.Skip("built with the transport")
		}
		addr := `{"id":"` + strings.Repeat("ab", 32) + `","addrs":[{"Ip":"127.0.0.1:1"}]}`
		_, err := s.AddP2P(addr)
		if err == nil || !strings.Contains(err.Error(), "no transport") || len(s.Machines()) != 1 {
			t.Fatalf("want the error and no machine joined, got %v and %v", err, s.Machines())
		}
	})
}

func TestStation_newAddress(t *testing.T) {
	t.Run("STATN-B16: O endereço novo chega a quem conhece esta máquina", func(t *testing.T) {
		first, _, second, secondDir := p2pPair(t)
		name, err := second.AddP2P(first.P2PAddr())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := nodeOf(t, second, name).Sessions(); err != nil {
			t.Fatal(err)
		}
		id := strings.TrimPrefix(name, P2P_PREFIX)
		_, knows := first.readHistory()[second.mustID(t)]
		// the second forgets where the first is
		os.WriteFile(filepath.Join(secondDir, "addresses.json"), []byte(`{"`+id+`":{"addrs":[]}}`), 0o600)
		first.tellAddr()
		back := false
		for end := time.Now().Add(10 * time.Second); !back && time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
			back = len(second.readHistory()[id].Addrs) > 0
		}
		if !knows || !back {
			t.Fatalf("want the first knowing the second, and the second with the first's address again, got %v and %v", knows, back)
		}
	})
	t.Run("STATN-E04: Um endereço de quem esta instalação não conhece", func(t *testing.T) {
		first, _, second, _ := p2pPair(t)
		c, err := second.p2p.DialFor(first.P2PAddr(), transport.KIND_ADDR)
		if err != nil {
			t.Fatal(err)
		}
		c.Write([]byte(second.P2PAddr()))
		c.CloseWrite()
		io.Copy(io.Discard, c)
		c.Close()
		time.Sleep(300 * time.Millisecond)
		if _, kept := first.readHistory()[second.mustID(t)]; kept {
			t.Fatal("want the address of an unknown installation ignored")
		}
	})
}
