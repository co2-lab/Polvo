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
	"time"
	"unicode/utf8"

	"github.com/co2-lab/polvo/internal/agents"
	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/transport"
)

// Pool keeps one connection per node, reused between calls (MCPTL-B05).
type Pool struct {
	cfg hub.Config

	mu    sync.Mutex
	nodes map[string]*hub.Node
	here  Here
	// peer reaches the nodes of other networks (MCPTL-B18); nil reaches only host:port
	peer transport.Peer
}

// SetPeer gives the pool this installation's endpoint, for the nodes of other networks (MCPTL-B18).
func (p *Pool) SetPeer(peer transport.Peer) {
	p.mu.Lock()
	p.peer = peer
	p.mu.Unlock()
}

// NewPool connects with the installation's identity and known hosts.
func NewPool(cfg hub.Config) *Pool { return &Pool{cfg: cfg, nodes: map[string]*hub.Node{}} }

// DEFAULT_NODE is this machine's node, where the polvo node and the window listen by default
// (MCPTL-B11).
const DEFAULT_NODE = "127.0.0.1:2222"

// Here is where the agent runs: the node of its machine and its session, empty when it runs outside
// Polvo (MCPTL-B11).
type Here struct {
	Node    string
	Session string
	// Claude is set when the agent is Claude Code (MCPTL-B13).
	Claude *Claude
}

// Claude is where to find the conversation of a Claude Code agent: its home, the folder it runs in, and
// the conversation's id it gave (empty when it gave none).
type Claude struct {
	Home string
	Dir  string
	ID   string
}

// SetHere tells the pool where the agent runs (MCPTL-B11).
func (p *Pool) SetHere(h Here) {
	p.mu.Lock()
	p.here = h
	p.mu.Unlock()
}

// whereAndWho is MCPTL-B11: the node and the agent a progress tool reports to; the failure of
// MCPTL-E04 when there is no agent.
func (p *Pool) whereAndWho(node, agent string) (string, string, error) {
	p.mu.Lock()
	h := p.here
	p.mu.Unlock()
	if node == "" {
		node = h.Node
	}
	if node == "" {
		node = DEFAULT_NODE
	}
	if agent == "" && h.Session != "" {
		agent = "session-" + h.Session
	}
	if agent == "" {
		// MCPTL-E04
		return "", "", fmt.Errorf("missing argument: agent (this agent runs outside a Polvo session; give it a name)")
	}
	return node, agent, nil
}

// report sends a progress report of the agent to its node.
func (p *Pool) report(node, agent string, params map[string]any) error {
	node, agent, err := p.whereAndWho(node, agent)
	if err != nil {
		return err
	}
	params["agent"] = agent
	return p.With(node, func(n *hub.Node) error { return n.Call("progress.report", params, &struct{}{}) })
}

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
	var n *hub.Node
	var err error
	if transport.IsAddr(addr) && p.peer != nil {
		// MCPTL-B18
		n, err = hub.DialVia(p.peer, addr, p.cfg)
	} else {
		n, err = hub.Dial(addr, p.cfg)
	}
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
		{
			Name:        "polvo_sessions",
			Description: "List the terminal sessions of a Polvo node: id, program, size, and whether you may type in it.",
			InputSchema: objectSchema(`"node":{"type":"string","description":"the node's address, host:port; by default this machine's"}`),
			Call:        p.sessions,
		},
		{
			Name:        "polvo_screen",
			Description: "Read the current screen of a terminal session, as plain text.",
			InputSchema: objectSchema(sessionProps, "session"),
			Call:        p.screen,
		},
		{
			Name:        "polvo_type",
			Description: "Type in a terminal session: the text as it is, then the keys by name (enter, tab, escape, backspace, delete, up, down, left, right, home, end, pageup, pagedown, ctrl-a to ctrl-z).",
			InputSchema: objectSchema(sessionProps+`,"text":{"type":"string"},"keys":{"type":"array","items":{"type":"string"}}`, "session"),
			Call:        p.typeIn,
		},
		{
			Name:        "polvo_wait",
			Description: "Wait until the screen of a terminal session shows the text, for up to timeout seconds (30 by default, 300 at most); returns the screen.",
			InputSchema: objectSchema(sessionProps+`,"text":{"type":"string"},"timeout":{"type":"number"}`, "session", "text"),
			Call:        p.wait,
		},
		{
			Name:        "polvo_progress_start",
			Description: "Start reporting a job to Polvo: its goal and its steps. Polvo shows your progress to the user. Returns the steps with their ids.",
			InputSchema: objectSchema(progressProps+`,"goal":{"type":"string"},"steps":{"type":"array","items":{"type":"string"}}`, "goal"),
			Call:        p.progressStart,
		},
		{
			Name:        "polvo_progress_step",
			Description: "Move a step of your job (by its id or title) to todo, doing, done or blocked, with an optional note.",
			InputSchema: objectSchema(progressProps+`,"step":{"type":"string"},"state":{"type":"string","enum":["todo","doing","done","blocked"]},"note":{"type":"string"}`, "step", "state"),
			Call:        p.progressStep,
		},
		{
			Name:        "polvo_progress_note",
			Description: "Leave a note in the log of your job.",
			InputSchema: objectSchema(progressProps+`,"text":{"type":"string"}`, "text"),
			Call:        p.progressNote,
		},
		{
			Name:        "polvo_progress_answer",
			Description: "Answer a question Polvo asked you (it comes with its id, like q-1).",
			InputSchema: objectSchema(progressProps+`,"question":{"type":"string"},"text":{"type":"string"}`, "question", "text"),
			Call:        p.progressAnswer,
		},
		{
			Name:        "polvo_progress_finish",
			Description: "Finish your job, with a summary.",
			InputSchema: objectSchema(progressProps+`,"summary":{"type":"string"}`, "summary"),
			Call:        p.progressFinish,
		},
	}
}

const sessionProps = `"node":{"type":"string","description":"the node's address, host:port; by default this machine's"},` +
	`"session":{"type":"string","description":"the session's id, from polvo_sessions"}`

const progressProps = `"agent":{"type":"string","description":"your name; inside a Polvo session it is the session's"},` +
	`"node":{"type":"string","description":"the node's address, host:port; by default this machine's"}`

// progressStart is MCPTL-B07.
func (p *Pool) progressStart(raw json.RawMessage) (Result, error) {
	var a struct {
		Node, Agent, Goal string
		Steps             []string
	}
	if err := args(raw, &a, map[string]*string{"goal": &a.Goal}); err != nil {
		return Result{}, err
	}
	params := map[string]any{"op": "start", "goal": a.Goal, "steps": a.Steps}
	p.mu.Lock()
	claude := p.here.Claude
	p.mu.Unlock()
	if claude != nil {
		// MCPTL-B13: without a conversation found, the job starts without it
		if conv, err := agents.Find(claude.Home, claude.Dir, claude.ID); err == nil {
			params["conversation"] = conv
		}
	}
	if err := p.report(a.Node, a.Agent, params); err != nil {
		return Result{}, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "started: %s\n", a.Goal)
	for i, s := range a.Steps {
		fmt.Fprintf(&b, "%d. %s\n", i+1, s)
	}
	return Result{Text: b.String()}, nil
}

// progressStep is MCPTL-B08.
func (p *Pool) progressStep(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Agent, Step, State, Note string }
	if err := args(raw, &a, map[string]*string{"step": &a.Step, "state": &a.State}); err != nil {
		return Result{}, err
	}
	if err := p.report(a.Node, a.Agent, map[string]any{"op": "step", "step": a.Step, "state": a.State, "text": a.Note}); err != nil {
		return Result{}, err
	}
	return Result{Text: fmt.Sprintf("step %s: %s", a.Step, a.State)}, nil
}

// progressNote is MCPTL-B09.
func (p *Pool) progressNote(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Agent, Text string }
	if err := args(raw, &a, map[string]*string{"text": &a.Text}); err != nil {
		return Result{}, err
	}
	if err := p.report(a.Node, a.Agent, map[string]any{"op": "note", "text": a.Text}); err != nil {
		return Result{}, err
	}
	return Result{Text: "noted"}, nil
}

// progressFinish is MCPTL-B10.
func (p *Pool) progressFinish(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Agent, Summary string }
	if err := args(raw, &a, map[string]*string{"summary": &a.Summary}); err != nil {
		return Result{}, err
	}
	if err := p.report(a.Node, a.Agent, map[string]any{"op": "finish", "text": a.Summary}); err != nil {
		return Result{}, err
	}
	return Result{Text: "finished: " + a.Summary}, nil
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

// WAIT_DEFAULT, WAIT_MAX and WAIT_POLL are how long polvo_wait waits when not told, the longest it waits,
// and how often it reads the screen (MCPTL-B17).
const (
	WAIT_DEFAULT = 30 * time.Second
	WAIT_MAX     = 5 * time.Minute
	WAIT_POLL    = 250 * time.Millisecond
)

// nodeOr is the node a session tool works on: the one given, or the agent's machine's (MCPTL-B11).
func (p *Pool) nodeOr(node string) string {
	if node != "" {
		return node
	}
	p.mu.Lock()
	node = p.here.Node
	p.mu.Unlock()
	if node == "" {
		node = DEFAULT_NODE
	}
	return node
}

// sessions is MCPTL-B14.
func (p *Pool) sessions(raw json.RawMessage) (Result, error) {
	var a struct{ Node string }
	if err := args(raw, &a, nil); err != nil {
		return Result{}, err
	}
	var list []hub.Session
	err := p.With(p.nodeOr(a.Node), func(n *hub.Node) (err error) {
		list, err = n.Sessions()
		return err
	})
	if err != nil {
		return Result{}, err
	}
	if len(list) == 0 {
		return Result{Text: "no sessions"}, nil
	}
	lines := make([]string, 0, len(list))
	for _, s := range list {
		may := "you may type"
		if !s.Writable {
			may = "view only"
		}
		lines = append(lines, fmt.Sprintf("%s  %s  %dx%d  %s", s.ID, strings.Join(s.Argv, " "), s.Cols, s.Rows, may))
	}
	return Result{Text: strings.Join(lines, "\n")}, nil
}

// screen is MCPTL-B15.
func (p *Pool) screen(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Session string }
	if err := args(raw, &a, map[string]*string{"session": &a.Session}); err != nil {
		return Result{}, err
	}
	var text string
	err := p.With(p.nodeOr(a.Node), func(n *hub.Node) (err error) {
		text, err = n.Screen(a.Session)
		return err
	})
	return Result{Text: text}, err
}

// keyBytes are what the keys of polvo_type send (MCPTL-B16).
var keyBytes = map[string]string{
	"enter": "\r", "tab": "\t", "escape": "\x1b", "backspace": "\x7f", "delete": "\x1b[3~",
	"up": "\x1b[A", "down": "\x1b[B", "right": "\x1b[C", "left": "\x1b[D",
	"home": "\x1b[H", "end": "\x1b[F", "pageup": "\x1b[5~", "pagedown": "\x1b[6~",
}

// keyOf is the bytes of a key by its name; false for a name it does not know (MCPTL-E01).
func keyOf(name string) (string, bool) {
	name = strings.ToLower(name)
	if b, ok := keyBytes[name]; ok {
		return b, true
	}
	if l, ok := strings.CutPrefix(name, "ctrl-"); ok && len(l) == 1 && l[0] >= 'a' && l[0] <= 'z' {
		return string(rune(l[0] & 0x1f)), true
	}
	return "", false
}

// typeIn is MCPTL-B16.
func (p *Pool) typeIn(raw json.RawMessage) (Result, error) {
	var a struct {
		Node, Session, Text string
		Keys                []string
	}
	if err := args(raw, &a, map[string]*string{"session": &a.Session}); err != nil {
		return Result{}, err
	}
	data := a.Text
	for _, k := range a.Keys {
		b, ok := keyOf(k)
		if !ok {
			// MCPTL-E01
			return Result{}, fmt.Errorf("invalid arguments: unknown key %q", k)
		}
		data += b
	}
	if data == "" {
		// MCPTL-E01
		return Result{}, fmt.Errorf("missing argument: text or keys")
	}
	err := p.With(p.nodeOr(a.Node), func(n *hub.Node) error { return n.Type(a.Session, []byte(data)) })
	if err != nil {
		return Result{}, err
	}
	return Result{Text: fmt.Sprintf("typed %d bytes in %s", len(data), a.Session)}, nil
}

// wait is MCPTL-B17.
func (p *Pool) wait(raw json.RawMessage) (Result, error) {
	var a struct {
		Node, Session, Text string
		Timeout             float64
	}
	if err := args(raw, &a, map[string]*string{"session": &a.Session, "text": &a.Text}); err != nil {
		return Result{}, err
	}
	limit := WAIT_DEFAULT
	if a.Timeout > 0 {
		limit = min(time.Duration(a.Timeout*float64(time.Second)), WAIT_MAX)
	}
	node := p.nodeOr(a.Node)
	end := time.Now().Add(limit)
	for {
		var text string
		err := p.With(node, func(n *hub.Node) (err error) {
			text, err = n.Screen(a.Session)
			return err
		})
		if err != nil {
			return Result{}, err
		}
		if strings.Contains(text, a.Text) {
			return Result{Text: text}, nil
		}
		if time.Now().After(end) {
			// MCPTL-E05
			return Result{}, fmt.Errorf("the screen did not show %q in %s; it shows:\n%s", a.Text, limit, text)
		}
		time.Sleep(WAIT_POLL)
	}
}

// progressAnswer is MCPTL-B12.
func (p *Pool) progressAnswer(raw json.RawMessage) (Result, error) {
	var a struct{ Node, Agent, Question, Text string }
	if err := args(raw, &a, map[string]*string{"question": &a.Question, "text": &a.Text}); err != nil {
		return Result{}, err
	}
	if err := p.report(a.Node, a.Agent, map[string]any{"op": "answer", "question": a.Question, "text": a.Text}); err != nil {
		return Result{}, err
	}
	return Result{Text: "answered " + a.Question}, nil
}
