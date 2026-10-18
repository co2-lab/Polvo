// @anchors
//   code: BRTST
//   ref: BRDGE

package bridge

import (
	"github.com/co2-lab/polvo/internal/discovery"
	"os"
	"path/filepath"

	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func open(t *testing.T) Handle {
	t.Helper()
	h, err := Open(t.TempDir(), "", "/bin/sh", "", 0)
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
		a, err := Open(t.TempDir(), "", "/bin/sh", "", 0)
		if err != nil {
			t.Fatal(err)
		}
		b := open(t)
		Close(a)
		if _, err := Call(a, []byte(`{"op":"machines"}`)); a == b || err == nil {
			t.Fatalf("want two numbers, and the closed one refused; got %d %d %v", a, b, err)
		}
	})
	t.Run("BRDGE-B02#01: Os pedidos sobre uma estação", func(t *testing.T) {
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
		var listening struct {
			Listen string `json:"listen"`
		}
		call(t, h, `{"op":"listen","addr":"127.0.0.1:0"}`, nil)
		call(t, h, `{"op":"status"}`, &listening)
		call(t, h, `{"op":"listen","addr":""}`, nil)
		if listening.Listen == "" {
			t.Fatalf("want it listening after the listen, got %+v", listening)
		}
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

// openIn opens a station that listens, announced in a test group of the loopback.
func openIn(t *testing.T, g *discovery.Options) Handle {
	t.Helper()
	h, err := openWith(t.TempDir(), ":0", "/bin/sh", "", 0, g)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Close(h) })
	return h
}

func testGroup(t *testing.T) *discovery.Options {
	t.Helper()
	lo, err := net.InterfaceByName("lo")
	if err != nil {
		t.Skip("no loopback interface")
	}
	free, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	port := free.LocalAddr().(*net.UDPAddr).Port
	free.Close()
	return &discovery.Options{Group: &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: port}, Interface: lo}
}

type shownCode struct {
	Requests []struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"requests"`
}

// codeOf waits for the code the station shows.
func codeOf(t *testing.T, h Handle) string {
	t.Helper()
	end := time.Now().Add(5 * time.Second)
	for time.Now().Before(end) {
		var s shownCode
		call(t, h, `{"op":"pairings"}`, &s)
		if len(s.Requests) > 0 {
			return s.Requests[0].Code
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no code shown")
	return ""
}

func TestBridge_pairing(t *testing.T) {
	t.Run("BRDGE-B05#01: Emparelhar em dois passos", func(t *testing.T) {
		g := testGroup(t)
		first, second := openIn(t, g), openIn(t, g)
		var found struct {
			Machines []struct {
				Addr  string `json:"addr"`
				Known bool   `json:"known"`
			} `json:"machines"`
		}
		call(t, second, `{"op":"discover","timeout_ms":500}`, &found)
		if len(found.Machines) != 1 || found.Machines[0].Known {
			t.Fatalf("want the first, unknown; got %+v", found)
		}
		addr := found.Machines[0].Addr
		var started struct {
			ID string `json:"id"`
		}
		call(t, second, `{"op":"pair_start","addr":"`+addr+`"}`, &started)
		code := codeOf(t, first)
		call(t, second, fmt.Sprintf(`{"op":"pair_code","id":"%s","code":"%s"}`, started.ID, code), nil)
		var ms struct {
			Machines []struct {
				Name string `json:"name"`
			} `json:"machines"`
		}
		call(t, second, `{"op":"machines"}`, &ms)
		if len(ms.Machines) != 2 || ms.Machines[1].Name != addr {
			t.Fatalf("want the first among the machines, got %+v", ms)
		}
	})
	t.Run("BRDGE-B05#02: Um código errado, e desistir", func(t *testing.T) {
		g := testGroup(t)
		first, second := openIn(t, g), openIn(t, g)
		s, _ := stationOf(first)
		var started struct {
			ID string `json:"id"`
		}
		call(t, second, `{"op":"pair_start","addr":"`+s.ListenAddr()+`"}`, &started)
		codeOf(t, first)
		_, wrong := Call(second, []byte(fmt.Sprintf(`{"op":"pair_code","id":"%s","code":"000000"}`, started.ID)))
		var again struct {
			ID string `json:"id"`
		}
		call(t, second, `{"op":"pair_start","addr":"`+s.ListenAddr()+`"}`, &again)
		call(t, second, fmt.Sprintf(`{"op":"pair_cancel","id":"%s"}`, again.ID), nil)
		_, gone := Call(second, []byte(fmt.Sprintf(`{"op":"pair_code","id":"%s","code":"1"}`, again.ID)))
		if wrong == nil || gone == nil {
			t.Fatalf("want the wrong code refused and the cancelled pairing gone, got %v and %v", wrong, gone)
		}
	})
}

func TestBridge_files(t *testing.T) {
	t.Run("BRDGE-B06: Os ficheiros de uma máquina", func(t *testing.T) {
		h := open(t)
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "a.txt"), []byte("ola"), 0o644)
		os.WriteFile(filepath.Join(dir, "bin"), []byte{0xff, 0xfe}, 0o644)
		js := func(s string) string { b, _ := json.Marshal(s); return string(b) }
		call(t, h, `{"op":"fs_write","machine":"local","path":`+js(filepath.Join(dir, "b.txt"))+`,"text":"escrito"}`, nil)
		var l struct {
			Entries []struct {
				Name string `json:"name"`
				Dir  bool   `json:"dir"`
			} `json:"entries"`
		}
		call(t, h, `{"op":"fs_list","machine":"local","path":`+js(dir)+`}`, &l)
		var r struct {
			Text string `json:"text"`
		}
		call(t, h, `{"op":"fs_read","machine":"local","path":`+js(filepath.Join(dir, "a.txt"))+`}`, &r)
		_, binErr := Call(h, []byte(`{"op":"fs_read","machine":"local","path":`+js(filepath.Join(dir, "bin"))+`}`))
		written, _ := os.ReadFile(filepath.Join(dir, "b.txt"))
		if len(l.Entries) != 3 || r.Text != "ola" || binErr == nil || string(written) != "escrito" {
			t.Fatalf("got %+v, %q, %v, %q", l, r.Text, binErr, written)
		}
	})
}

func TestBridge_p2p(t *testing.T) {
	t.Run("BRDGE-B02#02: Juntar uma máquina de outra rede", func(t *testing.T) {
		d1, d2 := t.TempDir(), t.TempDir()
		h1, err := Open(d1, "127.0.0.1:0", "/bin/sh", "", 0)
		if err != nil {
			t.Fatal(err)
		}
		defer Close(h1)
		h2, err := Open(d2, "", "/bin/sh", "", 0)
		if err != nil {
			t.Fatal(err)
		}
		defer Close(h2)
		var first, second struct {
			Key string `json:"key"`
			P2P string `json:"p2p"`
		}
		call(t, h1, `{"op":"status"}`, &first)
		call(t, h2, `{"op":"status"}`, &second)
		if first.P2P == "" {
			t.Skip("built without the transport (make test-p2p runs these)")
		}
		os.WriteFile(filepath.Join(d1, "authorized_keys"), []byte(second.Key+"\n"), 0o600)
		req, _ := json.Marshal(map[string]string{"op": "add_p2p", "addr": first.P2P})
		var added struct {
			Name string `json:"name"`
		}
		call(t, h2, string(req), &added)
		var l list
		call(t, h2, `{"op":"sessions","machine":"`+added.Name+`"}`, &l)
		if !strings.HasPrefix(added.Name, "p2p:") {
			t.Fatalf("want a p2p name, got %q", added.Name)
		}
	})
}
