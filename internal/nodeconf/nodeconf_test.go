// @anchors
//   code: NCTND
//   ref: NDCNN

package nodeconf

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/sessions"
)

func key(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func write(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "node.json")
	os.WriteFile(p, []byte(text), 0o600)
	return p
}

func TestNodeconf(t *testing.T) {
	t.Run("NDCNN-B01: Sem ficheiro, uma chave autorizada controla", func(t *testing.T) {
		c, err := Load(filepath.Join(t.TempDir(), "node.json"), key(t))
		if err != nil || c.PermFor(key(t)) != sessions.Control {
			t.Fatalf("want control, got %v and %v", c, err)
		}
	})
	t.Run("NDCNN-B02: A permissão de uma chave", func(t *testing.T) {
		named := key(t)
		c, err := Load(write(t, `{"guest":"view","keys":{"`+ssh.FingerprintSHA256(named)+`":"control"}}`), key(t))
		if err != nil || c.PermFor(named) != sessions.Control || c.PermFor(key(t)) != sessions.View {
			t.Fatalf("want control and view, got %v", err)
		}
	})
	t.Run("NDCNN-B03: A chave da instalação é o dono", func(t *testing.T) {
		own := key(t)
		c, err := Load(write(t, `{"guest":"view","keys":{"`+ssh.FingerprintSHA256(own)+`":"view"}}`), own)
		if err != nil || c.PermFor(own) != sessions.Owner {
			t.Fatalf("want owner, got %v", err)
		}
	})
	t.Run("NDCNN-B04: Quem pode encaminhar sockets", func(t *testing.T) {
		own, a, b, c := key(t), key(t), key(t), key(t)
		fp := ssh.FingerprintSHA256
		viewer := `"keys":{"` + fp(c) + `":"view"}`
		named, err := Load(write(t, `{"forward":["`+fp(a)+`","`+fp(c)+`"],`+viewer+`}`), own)
		if err != nil {
			t.Fatal(err)
		}
		none, _ := Load(filepath.Join(t.TempDir(), "node.json"), own)
		all, _ := Load(write(t, `{"forward":["*"],`+viewer+`}`), own)
		got := []bool{named.MayForward(own), named.MayForward(a), named.MayForward(b), named.MayForward(c),
			none.MayForward(own), none.MayForward(a), all.MayForward(b), all.MayForward(c)}
		want := []bool{true, true, false, false, true, false, true, false}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})
	t.Run("NDCNN-E01: Um ficheiro estragado não deixa o nó arrancar", func(t *testing.T) {
		p := write(t, "isto não é json")
		if _, err := Load(p, key(t)); err == nil || !strings.Contains(err.Error(), p) {
			t.Fatalf("want the error with %s, got %v", p, err)
		}
	})
	t.Run("NDCNN-E02: Uma permissão desconhecida não deixa o nó arrancar", func(t *testing.T) {
		if _, err := Load(write(t, `{"guest":"admin"}`), key(t)); err == nil || !strings.Contains(err.Error(), "admin") {
			t.Fatalf("want the error with admin, got %v", err)
		}
	})
}
