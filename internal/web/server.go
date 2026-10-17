// @anchors
//   code: WBIMS
//   ref: WEBSR

// Package web is the local HTTP server of the Polvo UI: the UI files, the control calls to the
// nodes and their terminals over WebSocket.
package web

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/mcp"
	"github.com/co2-lab/polvo/internal/proto"
	"github.com/co2-lab/polvo/internal/pty"
	"golang.org/x/net/websocket"
)

//go:embed dist
var dist embed.FS

// UI is the UI bundle embedded in the executable (ui/, built by `npm run build`) (WEBSR-B01).
func UI() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}

// Config is what the UI server runs with.
type Config struct {
	// Hub is how the server connects to the nodes.
	Hub hub.Config
	// Files are the UI files.
	Files fs.FS
	// Token is the session secret; empty makes a random one.
	Token string
}

// Server is the UI server of one session.
type Server struct {
	token string
	files fs.FS
	pool  *mcp.Pool
}

// New makes the server of a session.
func New(cfg Config) *Server {
	tok := cfg.Token
	if tok == "" {
		b := make([]byte, 24)
		rand.Read(b)
		tok = hex.EncodeToString(b)
	}
	return &Server{token: tok, files: cfg.Files, pool: mcp.NewPool(cfg.Hub)}
}

// Token is the session secret the browser has to show.
func (s *Server) Token() string { return s.token }

// Close ends the connections to the nodes.
func (s *Server) Close() { s.pool.Close() }

// Handler answers the UI's requests.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// WEBSR-B01: the UI files need no secret: they hold nothing usable without it.
	mux.Handle("/", http.FileServerFS(s.files))
	mux.HandleFunc("/api/call", s.call)
	mux.Handle("/ws/term", s.terminal())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WEBSR-B06: only requests to a local name (the defence against DNS rebinding).
		if !localHost(r.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func localHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// tokenOK is WEBSR-B03, in constant time.
func (s *Server) tokenOK(tok string) bool {
	return tok != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(s.token)) == 1
}

type callError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// call is WEBSR-B02: a control call to a node; a refusal comes back as an error object.
func (s *Server) call(w http.ResponseWriter, r *http.Request) {
	if !s.tokenOK(r.Header.Get("X-Polvo-Token")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// WEBSR-E02
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Node   string          `json:"node"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, proto.MaxLine)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var result json.RawMessage
	err := errors.New("no node")
	if req.Node != "" {
		err = s.pool.With(req.Node, func(n *hub.Node) error { return n.Call(req.Method, req.Params, &result) })
	}
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		// WEBSR-E01
		ce := callError{Code: proto.CodeInternalError, Message: err.Error()}
		var pe *proto.Error
		if errors.As(err, &pe) {
			ce = callError{Code: pe.Code, Message: pe.Message}
		}
		json.NewEncoder(w).Encode(map[string]any{"error": ce})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"result": result})
}

// terminal is WEBSR-B04 and WEBSR-B05.
func (s *Server) terminal() http.Handler {
	ws := websocket.Server{
		// WEBSR-B05: only the server's own origin.
		Handshake: func(_ *websocket.Config, r *http.Request) error {
			o, err := url.Parse(r.Header.Get("Origin"))
			if err != nil || o.Host != r.Host {
				return errors.New("cross-origin")
			}
			return nil
		},
		Handler: s.serveTerminal,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.tokenOK(r.URL.Query().Get("t")) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ws.ServeHTTP(w, r)
	})
}

type termMessage struct {
	Input  string `json:"input"`
	Resize []int  `json:"resize"`
}

func (s *Server) serveTerminal(c *websocket.Conn) {
	defer c.Close()
	q := c.Request().URL.Query()
	rows, _ := strconv.Atoi(q.Get("rows"))
	cols, _ := strconv.Atoi(q.Get("cols"))
	var t *hub.Terminal
	err := s.pool.With(q.Get("node"), func(n *hub.Node) (err error) {
		t, err = n.Shell(pty.Size{Rows: uint16(rows), Cols: uint16(cols)})
		return err
	})
	if err != nil {
		// WEBSR-E01
		websocket.JSON.Send(c, map[string]string{"error": err.Error()})
		return
	}
	defer t.Close()
	go func() {
		for {
			var m termMessage
			if err := websocket.JSON.Receive(c, &m); err != nil {
				t.Close()
				return
			}
			if m.Input != "" {
				t.Write([]byte(m.Input))
			}
			if len(m.Resize) == 2 {
				t.Resize(pty.Size{Rows: uint16(m.Resize[0]), Cols: uint16(m.Resize[1])})
			}
		}
	}()
	buf := make([]byte, 32<<10)
	for {
		n, err := t.Read(buf)
		if n > 0 {
			if websocket.Message.Send(c, buf[:n]) != nil {
				return
			}
		}
		if err != nil {
			break
		}
	}
	st, err := t.Wait()
	end := map[string]any{"exit": st.Code, "signal": st.Signal}
	if err != nil {
		end["error"] = err.Error()
	}
	websocket.JSON.Send(c, end)
}
