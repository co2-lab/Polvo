// @anchors
//   code: BRTST
//   ref: BRDGE

package bridge

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func open(t *testing.T) Handle {
	t.Helper()
	h, err := Open(t.TempDir(), "", "/bin/sh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Close(h) })
	return h
}

func call(t *testing.T, h Handle, req string, out any) {
	t.Helper()
	b, err := Call(h, []byte(req))
	if err != nil {
		t.Fatalf("%s: %v", req, err)
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("%s: %s: %v", req, b, err)
		}
	}
}

type list struct {
	Sessions []struct {
		ID     string `json:"id"`
		Locked bool   `json:"locked"`
	} `json:"sessions"`
}

func newSession(t *testing.T, h Handle) string {
	t.Helper()
	var made struct {
		ID string `json:"id"`
	}
	call(t, h, `{"op":"new","machine":"local","rows":24,"cols":80}`, &made)
	return made.ID
}

func TestBridge_station(t *testing.T) {
	t.Run("BRDGE-B01: Abrir e fechar uma estação", func(t *testing.T) {
		a, err := Open(t.TempDir(), "", "/bin/sh")
		if err != nil {
			t.Fatal(err)
		}
		b := open(t)
		Close(a)
		if _, err := Call(a, []byte(`{"op":"machines"}`)); a == b || err == nil {
			t.Fatalf("want two numbers, and the closed one refused; got %d %d %v", a, b, err)
		}
	})
	t.Run("BRDGE-B02: Os pedidos sobre uma estação", func(t *testing.T) {
		h := open(t)
		var status struct {
			Listen string `json:"listen"`
			Key    string `json:"key"`
		}
		call(t, h, `{"op":"status"}`, &status)
		var machines struct {
			Machines []struct {
				Name  string `json:"name"`
				Local bool   `json:"local"`
			} `json:"machines"`
		}
		call(t, h, `{"op":"machines"}`, &machines)
		id := newSession(t, h)
		call(t, h, `{"op":"lock","machine":"local","id":"`+id+`","locked":true}`, nil)
		var l list
		call(t, h, `{"op":"sessions","machine":"local"}`, &l)
		call(t, h, `{"op":"close","machine":"local","id":"`+id+`"}`, nil)
		if status.Listen != "" || !strings.HasPrefix(status.Key, "ssh-ed25519 ") || len(machines.Machines) != 1 || !machines.Machines[0].Local || len(l.Sessions) != 1 || !l.Sessions[0].Locked {
			t.Fatalf("got %+v, %+v, %+v", status, machines, l)
		}
	})
	t.Run("BRDGE-B03: Um terminal ligado a uma sessão", func(t *testing.T) {
		h := open(t)
		id := newSession(t, h)
		term, err := Attach(h, "local", id)
		if err != nil {
			t.Fatal(err)
		}
		// a mark only the result has: the echo of the line and the prompt do not make it
		if _, err := Write(term, []byte("echo marca-$((40+2))\n")); err != nil {
			t.Fatal(err)
		}
		var got strings.Builder
		end := time.Now().Add(10 * time.Second)
		buf := make([]byte, 4096)
		for !strings.Contains(got.String(), "marca-42") && time.Now().Before(end) {
			n, err := Read(term, buf)
			got.Write(buf[:n])
			if err != nil || n == 0 {
				break
			}
		}
		if err := Resize(term, 30, 100); err != nil {
			t.Fatal(err)
		}
		Detach(term)
		var l list
		call(t, h, `{"op":"sessions","machine":"local"}`, &l)
		if !strings.Contains(got.String(), "marca-42") || len(l.Sessions) != 1 {
			t.Fatalf("want 42 and the session alive, got %q and %+v", got.String(), l)
		}
	})
	t.Run("BRDGE-E01: Um número que não vale", func(t *testing.T) {
		h := open(t)
		_, readErr := Read(h, make([]byte, 8))
		_, callErr := Call(0, []byte(`{"op":"machines"}`))
		if readErr == nil || callErr == nil || !strings.Contains(callErr.Error(), "handle") {
			t.Fatalf("want both refused, got %v and %v", readErr, callErr)
		}
	})
	t.Run("BRDGE-E02: Um pedido que não se percebe", func(t *testing.T) {
		h := open(t)
		_, a := Call(h, []byte("nao json"))
		_, b := Call(h, []byte(`{"op":"voar"}`))
		if a == nil || b == nil || !strings.Contains(b.Error(), "voar") {
			t.Fatalf("want both refused, got %v and %v", a, b)
		}
	})
	t.Run("BRDGE-E03: O erro da máquina chega a quem chamou", func(t *testing.T) {
		h := open(t)
		_, err := Call(h, []byte(`{"op":"sessions","machine":"nenhuma"}`))
		if err == nil || !strings.Contains(err.Error(), "nenhuma") {
			t.Fatalf("want the machine's error, got %v", err)
		}
	})
}

type allProgress struct {
	Machines []struct {
		Name    string `json:"name"`
		Version uint64 `json:"version"`
		Error   string `json:"error"`
		Agents  []struct {
			Agent string `json:"agent"`
		} `json:"agents"`
	} `json:"machines"`
}

// report has an agent of the station's machine start a job, as the agent's tool does.
func report(t *testing.T, h Handle, agent string) {
	t.Helper()
	s, _ := stationOf(h)
	n, err := s.Node("local")
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Call("progress.report", map[string]any{"agent": agent, "op": "start", "goal": "g"}, &struct{}{}); err != nil {
		t.Fatal(err)
	}
}

func TestBridge_progress(t *testing.T) {
	t.Run("BRDGE-B04#01: O progresso de uma máquina, e perguntar a um agente", func(t *testing.T) {
		h := open(t)
		report(t, h, "a")
		var l struct {
			Version uint64 `json:"version"`
			Agents  []struct {
				Agent string `json:"agent"`
			} `json:"agents"`
		}
		call(t, h, `{"op":"progress","machine":"local"}`, &l)
		var q struct {
			ID string `json:"id"`
		}
		call(t, h, `{"op":"ask","machine":"local","agent":"a","text":"onde vais?"}`, &q)
		if len(l.Agents) != 1 || l.Agents[0].Agent != "a" || q.ID == "" {
			t.Fatalf("got %+v and %+v", l, q)
		}
	})
	t.Run("BRDGE-B04#02: Esperar pela mudança de qualquer máquina", func(t *testing.T) {
		h := open(t)
		var first allProgress
		call(t, h, `{"op":"progress_all","timeout_ms":1000}`, &first)
		go func() {
			time.Sleep(100 * time.Millisecond)
			report(t, h, "a")
		}()
		var got allProgress
		req := fmt.Sprintf(`{"op":"progress_all","since":{"local":%d},"timeout_ms":5000}`, first.Machines[0].Version)
		call(t, h, req, &got)
		if len(got.Machines) != 1 || len(got.Machines[0].Agents) != 1 {
			t.Fatalf("want the agent, got %+v", got)
		}
	})
	t.Run("BRDGE-B04#03: Uma máquina que a janela não conhece volta logo", func(t *testing.T) {
		h := open(t)
		began := time.Now()
		var got allProgress
		// at once, and not after the 20 s of the wait: half of it tells them apart in both engines
		call(t, h, `{"op":"progress_all","timeout_ms":20000}`, &got)
		if time.Since(began) > 10*time.Second || len(got.Machines) != 1 || got.Machines[0].Name != "local" {
			t.Fatalf("want local at once, got %+v in %v", got, time.Since(began))
		}
	})
}
