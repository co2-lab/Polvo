// @anchors
//   code: MCTSS
//   ref: MCPSR

package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/co2-lab/polvo/internal/proto"
)

var schema = json.RawMessage(`{"type":"object","properties":{"texto":{"type":"string"}}}`)

func tools() []Tool {
	return []Tool{
		{Name: "eco", Description: "devolve o texto", InputSchema: schema, Call: func(args json.RawMessage) (Result, error) {
			var a struct {
				Texto string `json:"texto"`
			}
			json.Unmarshal(args, &a)
			return Result{Text: a.Texto}, nil
		}},
		{Name: "falha", Description: "falha sempre", InputSchema: schema, Call: func(json.RawMessage) (Result, error) {
			return Result{}, errors.New("sem rede")
		}},
		{Name: "lenta", Description: "demora", InputSchema: schema, Call: func(json.RawMessage) (Result, error) {
			time.Sleep(500 * time.Millisecond)
			return Result{Text: "fim"}, nil
		}},
	}
}

type client struct {
	enc  *proto.Encoder
	dec  *proto.Decoder
	in   io.WriteCloser
	done chan error
}

func start(t *testing.T) client {
	t.Helper()
	ir, iw := io.Pipe()
	or, ow := io.Pipe()
	s := &Server{Name: "polvo", Version: "0.1.0", Tools: tools()}
	done := make(chan error, 1)
	go func() { done <- s.Serve(ir, ow); ow.Close() }()
	t.Cleanup(func() { iw.Close() })
	return client{enc: proto.NewEncoder(iw), dec: proto.NewDecoder(or), in: iw, done: done}
}

func (c client) send(t *testing.T, id int64, method string, params any) {
	t.Helper()
	b, _ := json.Marshal(params)
	m := &proto.Message{Method: method, Params: b}
	if id != 0 {
		m.ID = proto.IntID(id)
	}
	go c.enc.Write(m)
}

func (c client) next(t *testing.T) *proto.Message {
	t.Helper()
	m, err := c.dec.Read()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func (c client) call(t *testing.T, id int64, method string, params any, v any) *proto.Message {
	t.Helper()
	c.send(t, id, method, params)
	m := c.next(t)
	if v != nil && m.Error == nil {
		if err := json.Unmarshal(m.Result, v); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

type initResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct {
		Tools *struct{} `json:"tools"`
	} `json:"capabilities"`
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type callResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func TestServer_protocol(t *testing.T) {
	t.Run("MCPSR-B01: A inicialização devolve a versão pedida, as ferramentas como capacidade e o servidor", func(t *testing.T) {
		c := start(t)
		for i, v := range []struct{ ask, want string }{{"2025-03-26", "2025-03-26"}, {"1999-01-01", "2025-06-18"}} {
			var r initResult
			c.call(t, int64(i+1), "initialize", map[string]any{"protocolVersion": v.ask, "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "t", "version": "1"}}, &r)
			if r.ProtocolVersion != v.want || r.Capabilities.Tools == nil || r.ServerInfo.Name != "polvo" || r.ServerInfo.Version != "0.1.0" {
				t.Fatalf("want %s with tools and polvo 0.1.0, got %+v", v.want, r)
			}
		}
	})
	t.Run("MCPSR-B02: A lista tem as duas ferramentas pela ordem do servidor", func(t *testing.T) {
		c := start(t)
		var r struct {
			Tools []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				InputSchema json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		}
		c.call(t, 1, "tools/list", map[string]any{}, &r)
		if len(r.Tools) != 3 || r.Tools[0].Name != "eco" || r.Tools[1].Name != "falha" || r.Tools[0].Description == "" || len(r.Tools[0].InputSchema) == 0 {
			t.Fatalf("want eco, falha, lenta with description and schema, got %+v", r)
		}
	})
	t.Run("MCPSR-B03: A ferramenta eco devolve o texto, e a ferramenta que falha devolve um resultado de erro", func(t *testing.T) {
		c := start(t)
		var a, b callResult
		c.call(t, 1, "tools/call", map[string]any{"name": "eco", "arguments": map[string]string{"texto": "ola"}}, &a)
		c.call(t, 2, "tools/call", map[string]any{"name": "falha", "arguments": map[string]string{}}, &b)
		if len(a.Content) != 1 || a.Content[0].Type != "text" || a.Content[0].Text != "ola" || a.IsError {
			t.Fatalf("want the text ola, got %+v", a)
		}
		if len(b.Content) != 1 || b.Content[0].Text != "sem rede" || !b.IsError {
			t.Fatalf("want an error result sem rede, got %+v", b)
		}
	})
	t.Run("MCPSR-B04: O ping tem resposta vazia", func(t *testing.T) {
		m := start(t).call(t, 1, "ping", map[string]any{}, nil)
		if m.Error != nil || strings.TrimSpace(string(m.Result)) != "{}" {
			t.Fatalf("want {}, got %+v %s", m.Error, m.Result)
		}
	})
	t.Run("MCPSR-B05: A notificação de inicializado não tem resposta", func(t *testing.T) {
		c := start(t)
		c.send(t, 0, "notifications/initialized", map[string]any{})
		time.Sleep(50 * time.Millisecond)
		if n, _ := c.call(t, 9, "ping", map[string]any{}, nil).ID.Int(); n != 9 {
			t.Fatalf("want only the answer to 9, got %d", n)
		}
	})
	t.Run("MCPSR-B06: Uma ferramenta lenta não atrasa o ping que chega depois", func(t *testing.T) {
		c := start(t)
		c.send(t, 1, "tools/call", map[string]any{"name": "lenta", "arguments": map[string]any{}})
		time.Sleep(50 * time.Millisecond)
		c.send(t, 2, "ping", map[string]any{})
		if n, _ := c.next(t).ID.Int(); n != 2 {
			t.Fatalf("want the ping first, got %d", n)
		}
		if n, _ := c.next(t).ID.Int(); n != 1 {
			t.Fatalf("want the slow call next, got %d", n)
		}
	})
	t.Run("MCPSR-B07: O fim da entrada acaba o servidor depois da resposta em curso", func(t *testing.T) {
		c := start(t)
		c.send(t, 1, "tools/call", map[string]any{"name": "lenta", "arguments": map[string]any{}})
		time.Sleep(50 * time.Millisecond)
		c.in.Close()
		var r callResult
		m := c.next(t)
		json.Unmarshal(m.Result, &r)
		if len(r.Content) != 1 || r.Content[0].Text != "fim" {
			t.Fatalf("want the slow answer, got %+v", m)
		}
		if err := <-c.done; err != nil {
			t.Fatalf("want the server to end with no error, got %v", err)
		}
	})
}

func TestServer_errors(t *testing.T) {
	t.Run("MCPSR-E01: Um método desconhecido dá -32601", func(t *testing.T) {
		m := start(t).call(t, 1, "resources/list", map[string]any{}, nil)
		if m.Error == nil || m.Error.Code != proto.CodeMethodNotFound {
			t.Fatalf("want -32601, got %+v", m.Error)
		}
	})
	t.Run("MCPSR-E02: Uma ferramenta que não existe dá -32602", func(t *testing.T) {
		m := start(t).call(t, 1, "tools/call", map[string]any{"name": "nao-existe", "arguments": map[string]any{}}, nil)
		if m.Error == nil || m.Error.Code != proto.CodeInvalidParams {
			t.Fatalf("want -32602, got %+v", m.Error)
		}
	})
	t.Run("MCPSR-E03: Uma linha que não é JSON tem resposta de erro e o servidor continua", func(t *testing.T) {
		c := start(t)
		go io.WriteString(c.in, "isto nao e json\n")
		m := c.next(t)
		if m.Error == nil || m.Error.Code != proto.CodeParseError || !m.ID.IsNull() {
			t.Fatalf("want -32700 with the null id, got %+v", m)
		}
		if n, _ := c.call(t, 3, "ping", map[string]any{}, nil).ID.Int(); n != 3 {
			t.Fatalf("want the ping 3, got %d", n)
		}
	})
}
