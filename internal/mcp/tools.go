// @anchors
//   code: MCIMT
//   ref: MCPTL

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/co2-lab/polvo/internal/hub"
)

// Pool keeps one connection per node, reused between calls (MCPTL-B05).
type Pool struct {
	cfg hub.Config

	mu    sync.Mutex
	nodes map[string]*hub.Node
}

// NewPool connects with the installation's identity and known hosts.
func NewPool(cfg hub.Config) *Pool { return &Pool{cfg: cfg, nodes: map[string]*hub.Node{}} }

// Close ends every connection.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, n := range p.nodes {
		n.Close()
		delete(p.nodes, addr)
	}
}

func (p *Pool) node(addr string) (*hub.Node, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n, ok := p.nodes[addr]; ok {
		return n, nil
	}
	n, err := hub.Dial(addr, p.cfg)
	if err != nil {
		// MCPTL-E02: the error of hub.Dial names the address.
		return nil, err
	}
	p.nodes[addr] = n
	return n, nil
}

func (p *Pool) drop(addr string, n *hub.Node) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.nodes[addr] == n {
		delete(p.nodes, addr)
	}
	n.Close()
}

// With runs f on the node at addr; a dropped connection is made again once (MCPTL-B06).
func (p *Pool) With(addr string, f func(*hub.Node) error) error {
	for try := 0; ; try++ {
		n, err := p.node(addr)
		if err != nil {
			return err
		}
		err = f(n)
		if try == 0 && (errors.Is(err, hub.ErrClosed) || errors.Is(err, io.EOF)) {
			p.drop(addr, n)
			continue
		}
		return err
	}
}

// args decodes the arguments of a tool and checks the required ones (MCPTL-E01).
func args(raw json.RawMessage, v any, required map[string]*string) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	for name, val := range required {
		if *val == "" {
			return fmt.Errorf("missing argument: %s", name)
		}
	}
	return nil
}

func objectSchema(props string, required ...string) json.RawMessage {
	req, _ := json.Marshal(required)
	return json.RawMessage(`{"type":"object","properties":{` + props + `},"required":` + string(req) + `}`)
}

const nodeProp = `"node":{"type":"string","description":"the node's address, host:port"}`

// Tools are the Polvo tools over this pool.
func (p *Pool) Tools() []Tool {
	return []Tool{
		{
			Name:        "polvo_run",
			Description: "Run a shell command on a Polvo node until it ends; returns the exit code, stdout and stderr.",
			InputSchema: objectSchema(nodeProp+`,"command":{"type":"string"},"dir":{"type":"string"},"stdin":{"type":"string"}`, "node", "command"),
			Call:        p.run,
		},
		{
			Name:        "polvo_read_file",
			Description: "Read a text file on a Polvo node.",
			InputSchema: objectSchema(nodeProp+`,"path":{"type":"string"}`, "node", "path"),
			Call:        p.readFile,
		},
		{
			Name:        "polvo_write_file",
			Description: "Write a text file on a Polvo node, creating or replacing it.",
			InputSchema: objectSchema(nodeProp+`,"path":{"type":"string"},"content":{"type":"string"}`, "node", "path"),
			Call:        p.writeFile,
		},
		{
			Name:        "polvo_list",
			Description: "List a folder on a Polvo node: folders end with a slash, files show their size.",
			InputSchema: objectSchema(nodeProp+`,"path":{"type":"string"}`, "node", "path"),
			Call:        p.list,
		},
	}
}

// run is MCPTL-B01: a non-zero exit code is part of the result, not a failure of the tool.
func (p *Pool) run(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Command, Dir, Stdin string }
	if err := args(raw, &a, map[string]*string{"node": &a.Node, "command": &a.Command}); err != nil {
		return Result{}, err
	}
	var r hub.RunResult
	err := p.With(a.Node, func(n *hub.Node) (err error) {
		r, err = n.Run([]string{"sh", "-c", a.Command}, a.Dir, []byte(a.Stdin))
		return err
	})
	if err != nil {
		return Result{}, err
	}
	var b strings.Builder
	if r.Signal != "" {
		fmt.Fprintf(&b, "signal: %s\n", r.Signal)
	} else {
		fmt.Fprintf(&b, "exit code: %d\n", r.Code)
	}
	fmt.Fprintf(&b, "--- stdout ---\n%s", r.Stdout)
	fmt.Fprintf(&b, "\n--- stderr ---\n%s", r.Stderr)
	return Result{Text: b.String()}, nil
}

// readFile is MCPTL-B02: a file that is not UTF-8 text is a failure, with its size.
func (p *Pool) readFile(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Path string }
	if err := args(raw, &a, map[string]*string{"node": &a.Node, "path": &a.Path}); err != nil {
		return Result{}, err
	}
	var data []byte
	err := p.With(a.Node, func(n *hub.Node) (err error) {
		data, err = n.ReadFile(a.Path)
		return err
	})
	if err != nil {
		// MCPTL-E03: the node's message names the path.
		return Result{}, err
	}
	if !utf8.Valid(data) {
		return Result{}, fmt.Errorf("%s is a binary file (%d bytes), not text", a.Path, len(data))
	}
	return Result{Text: string(data)}, nil
}

// writeFile is MCPTL-B03.
func (p *Pool) writeFile(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Path, Content string }
	if err := args(raw, &a, map[string]*string{"node": &a.Node, "path": &a.Path}); err != nil {
		return Result{}, err
	}
	err := p.With(a.Node, func(n *hub.Node) error { return n.WriteFile(a.Path, []byte(a.Content)) })
	if err != nil {
		return Result{}, err
	}
	return Result{Text: fmt.Sprintf("wrote %d bytes to %s", len(a.Content), a.Path)}, nil
}

// list is MCPTL-B04.
func (p *Pool) list(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Path string }
	if err := args(raw, &a, map[string]*string{"node": &a.Node, "path": &a.Path}); err != nil {
		return Result{}, err
	}
	var es []hub.Entry
	err := p.With(a.Node, func(n *hub.Node) (err error) {
		es, err = n.List(a.Path)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	lines := make([]string, 0, len(es))
	for _, e := range es {
		if e.Dir {
			lines = append(lines, e.Name+"/")
		} else {
			lines = append(lines, fmt.Sprintf("%s (%d bytes)", e.Name, e.Size))
		}
	}
	return Result{Text: strings.Join(lines, "\n")}, nil
}
