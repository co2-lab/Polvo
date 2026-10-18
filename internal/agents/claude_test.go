// @anchors
//   code: CLTSQ
//   ref: CLDAG

package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/progress"
)

// project makes the folder of the conversations of dir, under home, with these conversations, each with
// the model of its assistant; the last is the most recent.
func project(t *testing.T, home, dir string, convs ...[2]string) {
	t.Helper()
	p := filepath.Join(home, ".claude", "projects", slug(dir))
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-time.Hour)
	for _, c := range convs {
		line := `{"type":"assistant","message":{"model":"` + c[1] + `","content":[]}}` + "\n"
		f := filepath.Join(p, c[0]+".jsonl")
		if err := os.WriteFile(f, []byte(`{"type":"user"}`+"\n"+line), 0o600); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(f, when, when)
		when = when.Add(time.Minute)
	}
}

// fakeClaude is a claude that prints out and keeps its arguments, one per line, in args.
func fakeClaude(t *testing.T, out string) (bin, args string) {
	t.Helper()
	d := t.TempDir()
	args = filepath.Join(d, "args")
	bin = filepath.Join(d, "claude")
	script := "#!/bin/sh\nfor a in \"$@\"; do echo \"$a\" >> " + args + "; done\npwd >> " + args + "\ncat <<'FIM'\n" + out + "\nFIM\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, args
}

func TestClaude_find(t *testing.T) {
	t.Run("CLDAG-B01#01: A conversa do id que o claude dá", func(t *testing.T) {
		home := t.TempDir()
		project(t, home, "/p", [2]string{"a", "m-a"}, [2]string{"b", "m-b"})
		c, err := Find(home, "/p", "a")
		if err != nil || c != (progress.Conversation{Kind: "claude", ID: "a", Dir: "/p", Model: "m-a"}) {
			t.Fatalf("got %+v %v", c, err)
		}
	})
	t.Run("CLDAG-B01#02: Sem id, a mais recente", func(t *testing.T) {
		home := t.TempDir()
		project(t, home, "/p", [2]string{"a", "m-a"}, [2]string{"b", "m-b"})
		c, err := Find(home, "/p", "")
		if err != nil || c.ID != "b" || c.Model != "m-b" {
			t.Fatalf("got %+v %v", c, err)
		}
	})
	t.Run("CLDAG-E01: Uma pasta sem conversas", func(t *testing.T) {
		_, err := Find(t.TempDir(), "/sem/nada", "")
		if err == nil || !strings.Contains(err.Error(), "/sem/nada") {
			t.Fatalf("want the error with the folder, got %v", err)
		}
	})
}

func TestClaude_ask(t *testing.T) {
	t.Run("CLDAG-B02: Perguntar", func(t *testing.T) {
		dir := t.TempDir()
		bin, args := fakeClaude(t, `{"type":"result","is_error":false,"result":"a meio"}`)
		got, err := Ask(bin, progress.Conversation{Kind: "claude", ID: "a", Dir: dir, Model: "m-a"}, "onde vais?")
		raw, _ := os.ReadFile(args)
		lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
		joined := strings.Join(lines, " ")
		if err != nil || got != "a meio" {
			t.Fatalf("got %q %v", got, err)
		}
		for _, want := range []string{"-p", "--resume a", "--fork-session", "--output-format json", "--permission-mode plan", "--model m-a"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("want %q in %q", want, joined)
			}
		}
		if lines[len(lines)-1] != dir || !strings.Contains(joined, "onde vais?") {
			t.Fatalf("want it run in %s with the question, got %q", dir, joined)
		}
	})
	t.Run("CLDAG-E02: Um claude que falha", func(t *testing.T) {
		bin, _ := fakeClaude(t, `{"type":"result","is_error":true,"result":"sem saldo"}`)
		_, err := Ask(bin, progress.Conversation{Kind: "claude", ID: "a", Dir: t.TempDir()}, "?")
		if err == nil || !strings.Contains(err.Error(), "sem saldo") {
			t.Fatalf("want the error with sem saldo, got %v", err)
		}
	})
}
