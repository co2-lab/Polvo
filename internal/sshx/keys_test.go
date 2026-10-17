// @anchors
//   code: KYTST
//   ref: KYSEX

package sshx

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func newKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s.PublicKey()
}

func same(a, b ssh.PublicKey) bool { return string(a.Marshal()) == string(b.Marshal()) }

func perm(t *testing.T, p string) os.FileMode {
	t.Helper()
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return st.Mode().Perm()
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var node = &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 2222}

func TestKeys_identity(t *testing.T) {
	t.Run("KYSEX-B01: A identidade nasce no primeiro pedido e é a mesma no segundo", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "id", "polvo_ed25519")
		s1, created1, err := LoadOrCreateIdentity(p)
		if err != nil {
			t.Fatal(err)
		}
		s2, created2, err := LoadOrCreateIdentity(p)
		if err != nil {
			t.Fatal(err)
		}
		if !created1 || created2 {
			t.Fatalf("want created on the first call only, got %v %v", created1, created2)
		}
		if !same(s1.PublicKey(), s2.PublicKey()) || s1.PublicKey().Type() != "ssh-ed25519" {
			t.Fatalf("want the same ssh-ed25519 key, got %s and %s", s1.PublicKey().Type(), s2.PublicKey().Type())
		}
	})
	t.Run("KYSEX-B02: A chave privada nasce só para o dono", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the mode bits are Unix")
		}
		dir := filepath.Join(t.TempDir(), "id")
		p := filepath.Join(dir, "polvo_ed25519")
		if _, _, err := LoadOrCreateIdentity(p); err != nil {
			t.Fatal(err)
		}
		if m := perm(t, p); m != 0o600 {
			t.Fatalf("want the key 0600, got %o", m)
		}
		if m := perm(t, dir); m != 0o700 {
			t.Fatalf("want the folder 0700, got %o", m)
		}
	})
}

func TestKeys_authorized(t *testing.T) {
	t.Run("KYSEX-B03: A chave da lista entra, a outra não, e a lista vale na verificação seguinte", func(t *testing.T) {
		a, b := newKey(t), newKey(t)
		p := filepath.Join(t.TempDir(), "authorized_keys")
		if err := os.WriteFile(p, []byte("# os meus portateis\n\n"+AuthorizedLine(a, "a")), 0o600); err != nil {
			t.Fatal(err)
		}
		ak := NewAuthorizedKeys(p)
		if ok, err := ak.Allows(a); !ok || err != nil {
			t.Fatalf("want A allowed, got %v %v", ok, err)
		}
		if ok, err := ak.Allows(b); ok || err != nil {
			t.Fatalf("want B refused, got %v %v", ok, err)
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(AuthorizedLine(b, "b"))
		f.Close()
		if ok, err := ak.Allows(b); !ok || err != nil {
			t.Fatalf("want B allowed after the edit, got %v %v", ok, err)
		}
	})
	t.Run("KYSEX-B04: Uma linha que não é chave é saltada e comunicada", func(t *testing.T) {
		a := newKey(t)
		p := filepath.Join(t.TempDir(), "authorized_keys")
		if err := os.WriteFile(p, []byte("ssh-ed25519 lixo\n"+AuthorizedLine(a, "a")), 0o600); err != nil {
			t.Fatal(err)
		}
		ak := NewAuthorizedKeys(p)
		var warned []int
		ak.Warn = func(line int, err error) { warned = append(warned, line) }
		if ok, err := ak.Allows(a); !ok || err != nil {
			t.Fatalf("want A allowed, got %v %v", ok, err)
		}
		if len(warned) != 1 || warned[0] != 1 {
			t.Fatalf("want one warning for line 1, got %v", warned)
		}
	})
	t.Run("KYSEX-E03: Sem a lista de chaves autorizadas ninguém entra", func(t *testing.T) {
		ak := NewAuthorizedKeys(filepath.Join(t.TempDir(), "nao-existe"))
		if ok, err := ak.Allows(newKey(t)); ok || err != nil {
			t.Fatalf("want refused without an error, got %v %v", ok, err)
		}
	})
	t.Run("KYSEX-I01: A linha escrita para uma chave é aceite para essa chave", func(t *testing.T) {
		a := newKey(t)
		p := filepath.Join(t.TempDir(), "authorized_keys")
		if err := os.WriteFile(p, []byte(AuthorizedLine(a, "")), 0o600); err != nil {
			t.Fatal(err)
		}
		if ok, err := NewAuthorizedKeys(p).Allows(a); !ok || err != nil {
			t.Fatalf("want A allowed, got %v %v", ok, err)
		}
	})
}

func TestKeys_knownHosts(t *testing.T) {
	t.Run("KYSEX-B05: O primeiro contacto com o nó \"10.0.0.1:2222\" regista a chave dele", func(t *testing.T) {
		n := newKey(t)
		p := filepath.Join(t.TempDir(), "known_hosts")
		if err := NewKnownHosts(p).Check("10.0.0.1:2222", node, n); err != nil {
			t.Fatalf("want the first contact accepted, got %v", err)
		}
		lines := strings.Split(strings.TrimSpace(read(t, p)), "\n")
		if len(lines) != 1 || !strings.Contains(lines[0], "10.0.0.1") ||
			!strings.Contains(lines[0], strings.TrimSpace(string(ssh.MarshalAuthorizedKey(n)))) {
			t.Fatalf("want one line with N for the node, got %q", lines)
		}
	})
	t.Run("KYSEX-B06: A segunda ligação com a mesma chave é aceite sem mudar a lista", func(t *testing.T) {
		n := newKey(t)
		p := filepath.Join(t.TempDir(), "known_hosts")
		kh := NewKnownHosts(p)
		if err := kh.Check("10.0.0.1:2222", node, n); err != nil {
			t.Fatal(err)
		}
		before := read(t, p)
		if err := kh.Check("10.0.0.1:2222", node, n); err != nil {
			t.Fatalf("want the same key accepted, got %v", err)
		}
		if after := read(t, p); after != before {
			t.Fatalf("the list changed: %q -> %q", before, after)
		}
	})
	t.Run("KYSEX-B07: Uma chave diferente para um nó conhecido é recusada", func(t *testing.T) {
		n, m := newKey(t), newKey(t)
		p := filepath.Join(t.TempDir(), "known_hosts")
		kh := NewKnownHosts(p)
		if err := kh.Check("10.0.0.1:2222", node, n); err != nil {
			t.Fatal(err)
		}
		before := read(t, p)
		err := kh.Check("10.0.0.1:2222", node, m)
		var hc *HostKeyChangedError
		if !errors.As(err, &hc) || hc.Host != "10.0.0.1:2222" {
			t.Fatalf("want HostKeyChangedError for 10.0.0.1:2222, got %v", err)
		}
		if after := read(t, p); after != before {
			t.Fatalf("the list changed: %q -> %q", before, after)
		}
	})
	t.Run("KYSEX-I02: A chave registada é aceite por uma lista nova lida do mesmo ficheiro", func(t *testing.T) {
		n := newKey(t)
		p := filepath.Join(t.TempDir(), "known_hosts")
		if err := NewKnownHosts(p).Check("10.0.0.1:2222", node, n); err != nil {
			t.Fatal(err)
		}
		if err := NewKnownHosts(p).Check("10.0.0.1:2222", node, n); err != nil {
			t.Fatalf("want accepted by a new list, got %v", err)
		}
	})
}

func TestKeys_identityErrors(t *testing.T) {
	t.Run("KYSEX-E01: Uma chave privada legível por outros é recusada", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the mode bits are Unix")
		}
		p := filepath.Join(t.TempDir(), "id", "polvo_ed25519")
		if _, _, err := LoadOrCreateIdentity(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, err := LoadOrCreateIdentity(p)
		var pe *PermissionError
		if !errors.As(err, &pe) || pe.Path != p || pe.Mode != 0o644 {
			t.Fatalf("want PermissionError with %s and 0644, got %v", p, err)
		}
	})
	t.Run("KYSEX-E02: Um ficheiro que não é chave dá erro com o caminho", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "polvo_ed25519")
		if err := os.WriteFile(p, []byte("isto nao e uma chave"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, _, err := LoadOrCreateIdentity(p)
		if err == nil || !strings.Contains(err.Error(), p) {
			t.Fatalf("want an error naming %s, got %v", p, err)
		}
	})
}
