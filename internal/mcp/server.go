// @anchors
//   code: MCIMS
//   ref: MCPSR

// Package mcp is `polvo mcp`: the Model Context Protocol server the local agent talks to, over
// stdin and stdout, with the Polvo tools on top of the hub.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/co2-lab/polvo/internal/proto"
)

// ProtocolVersions are the MCP versions this server speaks, newest first (MCPSR-B01).
var ProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// Result is what a tool call returns: text, marked as an error when the tool failed (MCPSR-B03).
type Result struct {
	Text    string
	IsError bool
}

// Tool is one tool the server offers.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	Call        func(args json.RawMessage) (Result, error)
}

// Server answers the MCP requests of one client.
type Server struct {
	Name    string
	Version string
	Tools   []Tool
}

// Serve reads requests from r and writes the answers to w until r ends (MCPSR-B07).
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	enc, dec := proto.NewEncoder(w), proto.NewDecoder(r)
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		m, err := dec.Read()
		var re *proto.ReadError
		switch {
		case errors.As(err, &re):
			// MCPSR-E03: a line that is not a message is answered, and the server goes on.
			id := re.ID
			if id.IsZero() {
				id = proto.NullID()
			}
			enc.Write(&proto.Message{ID: id, Error: &proto.Error{Code: re.Code, Message: re.Message}})
			continue
		case errors.Is(err, proto.ErrTooLarge):
			enc.Write(&proto.Message{ID: proto.NullID(), Error: &proto.Error{Code: proto.CodeInvalidRequest, Message: err.Error()}})
			continue
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return err
		}
		// MCPSR-B05: a notification has no answer.
		if m.Kind() != proto.Request {
			continue
		}
		// MCPSR-B06: each request is answered on its own.
		wg.Add(1)
		go func(m *proto.Message) {
			defer wg.Done()
			enc.Write(s.answer(m))
		}(m)
	}
}

func (s *Server) answer(m *proto.Message) *proto.Message {
	res, perr := s.dispatch(m)
	if perr != nil {
		return &proto.Message{ID: m.ID, Error: perr}
	}
	b, err := json.Marshal(res)
	if err != nil {
		return &proto.Message{ID: m.ID, Error: &proto.Error{Code: proto.CodeInternalError, Message: err.Error()}}
	}
	return &proto.Message{ID: m.ID, Result: b}
}

func invalidParams(format string, a ...any) *proto.Error {
	return &proto.Error{Code: proto.CodeInvalidParams, Message: fmt.Sprintf(format, a...)}
}

func (s *Server) dispatch(m *proto.Message) (any, *proto.Error) {
	switch m.Method {
	case "initialize":
		// MCPSR-B01: the version the client asked for when this server knows it, else the newest.
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return nil, invalidParams("invalid initialize params")
		}
		version := ProtocolVersions[0]
		if slices.Contains(ProtocolVersions, p.ProtocolVersion) {
			version = p.ProtocolVersion
		}
		return map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": s.Name, "version": s.Version},
		}, nil
	case "tools/list":
		// MCPSR-B02
		list := make([]map[string]any, 0, len(s.Tools))
		for _, t := range s.Tools {
			list = append(list, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema})
		}
		return map[string]any{"tools": list}, nil
	case "tools/call":
		return s.call(m.Params)
	case "ping":
		// MCPSR-B04
		return map[string]any{}, nil
	}
	// MCPSR-E01
	return nil, &proto.Error{Code: proto.CodeMethodNotFound, Message: "method not found: " + m.Method}
}

// call is MCPSR-B03: the tool's failure is a result marked as an error, not a protocol error.
func (s *Server) call(params json.RawMessage) (any, *proto.Error) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if json.Unmarshal(params, &p) != nil {
		return nil, invalidParams("invalid tools/call params")
	}
	i := slices.IndexFunc(s.Tools, func(t Tool) bool { return t.Name == p.Name })
	if i < 0 {
		// MCPSR-E02
		return nil, invalidParams("unknown tool: %s", p.Name)
	}
	if len(p.Arguments) == 0 {
		p.Arguments = json.RawMessage("{}")
	}
	r, err := s.Tools[i].Call(p.Arguments)
	if err != nil {
		r = Result{Text: err.Error(), IsError: true}
	}
	return map[string]any{
		"content": []map[string]string{{"type": "text", "text": r.Text}},
		"isError": r.IsError,
	}, nil
}
