// @anchors
//   code: PRTST
//   ref: PRGRS

package progress

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func board(t *testing.T) (*Board, string) {
	t.Helper()
	dir := t.TempDir()
	b, err := NewBoard(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b, dir
}

func start(t *testing.T, b *Board, agent string, steps ...string) {
	t.Helper()
	if _, err := b.Start(agent, "migrar", steps); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, b *Board, agent string) Progress {
	t.Helper()
	p, err := b.Get(agent)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func last(p Progress) Entry { return p.Log[len(p.Log)-1] }

func TestBoard_report(t *testing.T) {
	t.Run("PRGRS-B01: Começar um trabalho", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "claude-1", "antigo")
		b.Note("claude-1", "velha")
		start(t, b, "claude-1", "ler", "escrever")
		p := get(t, b, "claude-1")
		if p.Goal != "migrar" || len(p.Steps) != 2 || p.Steps[0].Title != "ler" || p.Steps[1].State != Todo || len(p.Log) != 1 || p.Log[0].Kind != "start" {
			t.Fatalf("want the new job, got %+v", p)
		}
	})
	t.Run("PRGRS-B02: O estado de um passo", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "ler")
		if err := b.Step("a", "ler", Doing, "a ler o código"); err != nil {
			t.Fatal(err)
		}
		p := get(t, b, "a")
		if p.Steps[0].State != Doing || last(p).Step != "1" || last(p).State != Doing || last(p).Text != "a ler o código" {
			t.Fatalf("want the step doing, logged; got %+v", p)
		}
	})
	t.Run("PRGRS-B03: Uma nota", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "ler")
		if err := b.Note("a", "encontrei um teste partido"); err != nil {
			t.Fatal(err)
		}
		p := get(t, b, "a")
		if last(p).Kind != "note" || last(p).Text != "encontrei um teste partido" || p.Steps[0].State != Todo {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("PRGRS-B04: Terminar", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "ler")
		if err := b.Finish("a", "feito"); err != nil {
			t.Fatal(err)
		}
		p := get(t, b, "a")
		if !p.Finished || last(p).Kind != "finish" || last(p).Text != "feito" {
			t.Fatalf("got %+v", p)
		}
	})
}

func TestBoard_read(t *testing.T) {
	t.Run("PRGRS-B05: Ler o progresso", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "x")
		start(t, b, "b", "y")
		time.Sleep(2 * time.Millisecond)
		b.Note("a", "agora")
		version, list := b.List()
		if version != b.Version() || len(list) != 2 || list[0].Agent != "a" || list[1].Agent != "b" || get(t, b, "b").Agent != "b" {
			t.Fatalf("got %d %+v", version, list)
		}
	})
	t.Run("PRGRS-B06#01: Esperar quando o quadro já mudou", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "x")
		b.Note("a", "1")
		b.Note("a", "2")
		v, _ := b.Wait(context.Background(), b.Version()-1, time.Minute)
		if v != 3 {
			t.Fatalf("want version 3 at once, got %d", v)
		}
	})
	t.Run("PRGRS-B06#02: Esperar pela próxima mudança", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "x")
		since := b.Version()
		go func() {
			time.Sleep(50 * time.Millisecond)
			b.Note("a", "chegou")
		}()
		v, list := b.Wait(context.Background(), since, 5*time.Second)
		if v != since+1 || last(list[0]).Text != "chegou" {
			t.Fatalf("want the note, got %d %+v", v, list)
		}
	})
	t.Run("PRGRS-B06#03: O prazo da espera", func(t *testing.T) {
		b, _ := board(t)
		since := b.Version()
		began := time.Now()
		v, _ := b.Wait(context.Background(), since, 100*time.Millisecond)
		if v != since || time.Since(began) < 100*time.Millisecond {
			t.Fatalf("want the same version after the wait, got %d in %v", v, time.Since(began))
		}
	})
	t.Run("PRGRS-B07: O progresso fica gravado", func(t *testing.T) {
		b, dir := board(t)
		start(t, b, "a", "ler")
		b.Step("a", "1", Done, "")
		again, err := NewBoard(dir)
		if err != nil {
			t.Fatal(err)
		}
		p := get(t, again, "a")
		if p.Goal != "migrar" || p.Steps[0].State != Done || len(p.Log) != 2 {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("PRGRS-B08: O registo tem um limite", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "x")
		for i := 0; i < LOG_LIMIT+10; i++ {
			b.Note("a", "nota")
		}
		b.Note("a", "a mais nova")
		p := get(t, b, "a")
		if len(p.Log) != LOG_LIMIT || last(p).Text != "a mais nova" {
			t.Fatalf("want %d entries ending in the newest, got %d", LOG_LIMIT, len(p.Log))
		}
	})
}

func TestBoard_errors(t *testing.T) {
	t.Run("PRGRS-E01: Um nome de agente que não é aceite", func(t *testing.T) {
		b, _ := board(t)
		for _, agent := range []string{"a/b", "", strings.Repeat("x", AGENT_MAX+1)} {
			if _, err := b.Start(agent, "g", nil); err == nil {
				t.Fatalf("want %q refused", agent)
			}
		}
		if _, list := b.List(); len(list) != 0 {
			t.Fatalf("want an empty board, got %+v", list)
		}
	})
	t.Run("PRGRS-E02: Reportar sem trabalho", func(t *testing.T) {
		b, _ := board(t)
		if err := b.Note("x", "n"); !errors.Is(err, ErrNoJob) {
			t.Fatalf("want ErrNoJob, got %v", err)
		}
	})
	t.Run("PRGRS-E03: Um passo que não é do trabalho", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "ler")
		if err := b.Step("a", "voar", Done, ""); !errors.Is(err, ErrNoStep) {
			t.Fatalf("want ErrNoStep, got %v", err)
		}
	})
	t.Run("PRGRS-E04: Um estado que não existe", func(t *testing.T) {
		b, _ := board(t)
		start(t, b, "a", "ler")
		err := b.Step("a", "ler", State("talvez"), "")
		if err == nil || !strings.Contains(err.Error(), "talvez") || get(t, b, "a").Steps[0].State != Todo {
			t.Fatalf("want the state refused, got %v", err)
		}
	})
	t.Run("PRGRS-E05: Um trabalho que não se grava", func(t *testing.T) {
		b, dir := board(t)
		os.RemoveAll(dir)
		os.WriteFile(dir, []byte("not a folder"), 0o600)
		_, err := b.Start("a", "g", []string{"x"})
		if err == nil {
			t.Fatal("want the write error")
		}
		if _, err := b.Get("a"); err != nil {
			t.Fatalf("want the job in memory, got %v", err)
		}
	})
	t.Run("PRGRS-E06: Um ficheiro que não é um trabalho", func(t *testing.T) {
		b, dir := board(t)
		start(t, b, "a", "x")
		os.WriteFile(filepath.Join(dir, "estragado.json"), []byte("{nao"), 0o600)
		again, err := NewBoard(dir)
		if err != nil {
			t.Fatal(err)
		}
		if _, list := again.List(); len(list) != 1 || list[0].Agent != "a" {
			t.Fatalf("want only a, got %+v", list)
		}
	})
}
