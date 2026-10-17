// @anchors
//   code: WBTST
//   ref: WEBSR

package web

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/sshx"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
)

func newSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type world struct {
	srv  *httptest.Server
	node string
}

// setup serves a node and the UI server allowed into it.
func setup(t *testing.T) world {
	t.Helper()
	id := newSigner(t)
	ak := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(ak, []byte(sshx.AuthorizedLine(id.PublicKey(), "")), 0o600)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go node.Serve(l, node.Config{HostKey: newSigner(t), Authorized: sshx.NewAuthorizedKeys(ak), Control: node.ServeControl})
	s := New(Config{
		Hub:   hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))},
		Files: fstest.MapFS{"index.html": {Data: []byte("<h1>polvo</h1>")}},
		Token: "segredo",
	})
	t.Cleanup(s.Close)
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return world{srv: srv, node: l.Addr().String()}
}

type answer struct {
	Status int
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (w world) call(t *testing.T, token, node, method string, params any) answer {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"node": node, "method": method, "params": params})
	req, _ := http.NewRequest(http.MethodPost, w.srv.URL+"/api/call", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("X-Polvo-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	a := answer{Status: resp.StatusCode}
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func TestServer_files(t *testing.T) {
	t.Run("WEBSR-B01: A raiz dá o index.html sem segredo", func(t *testing.T) {
		w := setup(t)
		resp, err := http.Get(w.srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || string(b) != "<h1>polvo</h1>" {
			t.Fatalf("want 200 and the index, got %d %q", resp.StatusCode, b)
		}
	})
	t.Run("WEBSR-B01: Os ficheiros embutidos da UI servem-se", func(t *testing.T) {
		s := New(Config{Files: UI(), Token: "x"})
		defer s.Close()
		srv := httptest.NewServer(s.Handler())
		defer srv.Close()
		for _, p := range []string{"/", "/app.js", "/style.css", "/xterm.css"} {
			resp, err := http.Get(srv.URL + p)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("%s: want 200, got %d", p, resp.StatusCode)
			}
		}
	})
	t.Run("WEBSR-B06: Um pedido para um nome que não é local é recusado", func(t *testing.T) {
		w := setup(t)
		req, _ := http.NewRequest(http.MethodGet, w.srv.URL+"/", nil)
		req.Host = "polvo.mal.exemplo"
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("want 403, got %d", resp.StatusCode)
		}
	})
}

func TestServer_calls(t *testing.T) {
	t.Run("WEBSR-B02: A chamada fs.list com o segredo devolve as entradas do nó", func(t *testing.T) {
		w := setup(t)
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "a.txt"), []byte("a"), 0o644)
		a := w.call(t, "segredo", w.node, "fs.list", map[string]string{"path": d})
		if a.Status != 200 || a.Error != nil || !strings.Contains(string(a.Result), `"a.txt"`) {
			t.Fatalf("want the entry a.txt, got %+v %s", a, a.Result)
		}
	})
	t.Run("WEBSR-B02: Uma recusa do nó volta como objeto de erro", func(t *testing.T) {
		w := setup(t)
		a := w.call(t, "segredo", w.node, "fs.read", map[string]string{"path": "/nao/existe"})
		if a.Error == nil || a.Error.Code != -32000 || !strings.Contains(a.Error.Message, "/nao/existe") {
			t.Fatalf("want the error -32000, got %+v", a)
		}
	})
	t.Run("WEBSR-B03: Sem o segredo, ou com outro, a chamada é recusada", func(t *testing.T) {
		w := setup(t)
		for _, tok := range []string{"", "errado"} {
			if a := w.call(t, tok, w.node, "fs.list", map[string]string{"path": "/"}); a.Status != http.StatusUnauthorized {
				t.Fatalf("token %q: want 401, got %d", tok, a.Status)
			}
		}
	})
	t.Run("WEBSR-E01: Um nó que não responde volta como objeto de erro", func(t *testing.T) {
		w := setup(t)
		a := w.call(t, "segredo", "127.0.0.1:1", "fs.list", map[string]string{"path": "/"})
		if a.Error == nil || !strings.Contains(a.Error.Message, "127.0.0.1:1") {
			t.Fatalf("want an error with the address, got %+v", a)
		}
	})
	t.Run("WEBSR-E02: Um GET à chamada e um corpo que não é JSON são recusados", func(t *testing.T) {
		w := setup(t)
		req, _ := http.NewRequest(http.MethodGet, w.srv.URL+"/api/call", nil)
		req.Header.Set("X-Polvo-Token", "segredo")
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()
		req2, _ := http.NewRequest(http.MethodPost, w.srv.URL+"/api/call", strings.NewReader("isto nao e json"))
		req2.Header.Set("X-Polvo-Token", "segredo")
		resp2, _ := http.DefaultClient.Do(req2)
		resp2.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed || resp2.StatusCode != http.StatusBadRequest {
			t.Fatalf("want 405 and 400, got %d and %d", resp.StatusCode, resp2.StatusCode)
		}
	})
}

func (w world) terminal(t *testing.T, origin string) (*websocket.Conn, error) {
	t.Helper()
	u := strings.Replace(w.srv.URL, "http://", "ws://", 1) + "/ws/term?t=segredo&node=" + w.node + "&rows=30&cols=100"
	return websocket.Dial(u, "", origin)
}

func TestServer_terminal(t *testing.T) {
	t.Run("WEBSR-B04: O terminal por WebSocket corre um comando e termina com o código 4", func(t *testing.T) {
		w := setup(t)
		c, err := w.terminal(t, w.srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		websocket.JSON.Send(c, map[string]any{"resize": []int{40, 120}})
		websocket.JSON.Send(c, map[string]any{"input": "stty size; exit 4\n"})
		var out bytes.Buffer
		var last string
		for {
			var frame []byte
			if err := websocket.Message.Receive(c, &frame); err != nil {
				break
			}
			if bytes.HasPrefix(frame, []byte(`{"exit"`)) {
				last = string(frame)
				continue
			}
			out.Write(frame)
		}
		if !strings.Contains(out.String(), "40 120") || !strings.Contains(last, `"exit":4`) {
			t.Fatalf("want 40 120 and exit 4, got %q and %q", out.String(), last)
		}
	})
	t.Run("WEBSR-B05: O WebSocket de outra origem é recusado", func(t *testing.T) {
		w := setup(t)
		u := w.srv.URL + "/ws/term?t=segredo&node=" + w.node
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==") // gitleaks:allow (the sample nonce of RFC 6455)
		req.Header.Set("Origin", "http://mal.exemplo")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("want 403, got %d", resp.StatusCode)
		}
	})
}
