// @anchors
//   code: PRIMQ
//   ref: PRGRS

// Package progress keeps the progress of the agents of a machine: each agent's job, with its goal,
// its steps and the log of what happened, in a file per agent.
package progress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AGENT_MAX is the longest agent name (PRGRS-E01).
const AGENT_MAX = 128

// LOG_LIMIT is how many entries a job's log keeps (PRGRS-B08).
const LOG_LIMIT = 500

// State is where a step is.
type State string

const (
	Todo    State = "todo"
	Doing   State = "doing"
	Done    State = "done"
	Blocked State = "blocked"
)

func (s State) valid() bool { return s == Todo || s == Doing || s == Done || s == Blocked }

// ErrNoJob is the report of an agent with no job (PRGRS-E02).
var ErrNoJob = errors.New("progress: the agent has no job; start one first")

// ErrNoQuestion names a question the job does not have waiting for its answer (PRGRS-E07).
var ErrNoQuestion = errors.New("progress: no such question waiting for an answer")

// QuestionState is where a question is.
type QuestionState string

const (
	Pending  QuestionState = "pending"
	Answered QuestionState = "answered"
	Failed   QuestionState = "failed"
)

// Question is a question Polvo asked the agent, and its answer (PRGRS-B09, PRGRS-B10).
type Question struct {
	ID       string        `json:"id"`
	Text     string        `json:"text"`
	Asked    time.Time     `json:"asked"`
	State    QuestionState `json:"state"`
	Answer   string        `json:"answer,omitempty"`
	Answered time.Time     `json:"answered,omitempty"`
}

// Conversation is the agent's conversation, for Polvo to ask it things: the agent's kind (claude), the
// conversation's id, the folder it runs in, and its model.
type Conversation struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Dir   string `json:"dir"`
	Model string `json:"model,omitempty"`
}

// ErrNoStep names a step the job does not have (PRGRS-E03).
var ErrNoStep = errors.New("progress: no such step")

// Step is a step of a job.
type Step struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State State  `json:"state"`
}

// Entry is something that happened in a job: its start, a step's change, a note, its end.
type Entry struct {
	At    time.Time `json:"at"`
	Kind  string    `json:"kind"`
	Step  string    `json:"step,omitempty"`
	State State     `json:"state,omitempty"`
	Text  string    `json:"text,omitempty"`
}

// Progress is an agent's job.
type Progress struct {
	Agent    string  `json:"agent"`
	Goal     string  `json:"goal"`
	Steps    []Step  `json:"steps"`
	Log      []Entry `json:"log"`
	Finished bool    `json:"finished"`
	// Conversation is the agent's, when it gave it at the start.
	Conversation *Conversation `json:"conversation,omitempty"`
	Questions    []Question    `json:"questions,omitempty"`
	Started      time.Time     `json:"started"`
	Updated      time.Time     `json:"updated"`
}

func (p Progress) clone() Progress {
	p.Steps = append([]Step(nil), p.Steps...)
	p.Log = append([]Entry(nil), p.Log...)
	p.Questions = append([]Question(nil), p.Questions...)
	if p.Conversation != nil {
		c := *p.Conversation
		p.Conversation = &c
	}
	return p
}

// Board keeps the jobs of the agents of a machine.
type Board struct {
	dir string

	mu      sync.Mutex
	jobs    map[string]*Progress
	version uint64
	changed chan struct{}
}

// NewBoard is the board of dir, with the jobs saved there (PRGRS-B07); a file that is not a job is
// left out with a warning (PRGRS-E06).
func NewBoard(dir string) (*Board, error) {
	b := &Board{dir: dir, jobs: map[string]*Progress{}, changed: make(chan struct{})}
	files, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, f.Name()))
		var p Progress
		if err == nil {
			err = json.Unmarshal(raw, &p)
		}
		if err == nil && check(p.Agent) != nil {
			err = errors.New("no agent")
		}
		if err != nil {
			// PRGRS-E06
			fmt.Fprintf(os.Stderr, "polvo: %s is not a job (%v); left out\n", filepath.Join(dir, f.Name()), err)
			continue
		}
		b.jobs[p.Agent] = &p
	}
	return b, nil
}

// check is PRGRS-E01.
func check(agent string) error {
	switch {
	case agent == "":
		return errors.New("progress: an agent needs a name")
	case len(agent) > AGENT_MAX:
		return fmt.Errorf("progress: the agent's name is longer than %d", AGENT_MAX)
	case strings.ContainsAny(agent, "/\\\x00") || agent == "." || agent == "..":
		return fmt.Errorf("progress: the agent's name %q has a character a name cannot have", agent)
	}
	return nil
}

func (b *Board) file(agent string) string {
	return filepath.Join(b.dir, url.PathEscape(agent)+".json")
}

// change applies f to the agent's job, logs entry, bumps the version, wakes the waiting and saves the
// job; the save's error goes back with the change kept in memory (PRGRS-E05).
func (b *Board) change(agent string, f func(*Progress) error) error {
	if err := check(agent); err != nil {
		// PRGRS-E01
		return err
	}
	b.mu.Lock()
	p := b.jobs[agent]
	if p == nil {
		b.mu.Unlock()
		// PRGRS-E02
		return ErrNoJob
	}
	next := p.clone()
	if err := f(&next); err != nil {
		b.mu.Unlock()
		return err
	}
	saved := b.commit(agent, &next)
	b.mu.Unlock()
	return b.save(saved)
}

// commit makes p the agent's job, with the log in its limit (PRGRS-B08), and tells the waiting
// (PRGRS-B06); it holds b.mu.
func (b *Board) commit(agent string, p *Progress) Progress {
	if over := len(p.Log) - LOG_LIMIT; over > 0 {
		p.Log = append([]Entry(nil), p.Log[over:]...)
	}
	p.Updated = time.Now()
	b.jobs[agent] = p
	b.version++
	close(b.changed)
	b.changed = make(chan struct{})
	return p.clone()
}

// save writes the job at once (PRGRS-B07).
func (b *Board) save(p Progress) error {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(b.dir, 0o700); err != nil {
		// PRGRS-E05
		return fmt.Errorf("progress: saving %s: %w", p.Agent, err)
	}
	tmp := b.file(p.Agent) + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("progress: saving %s: %w", p.Agent, err)
	}
	if err := os.Rename(tmp, b.file(p.Agent)); err != nil {
		return fmt.Errorf("progress: saving %s: %w", p.Agent, err)
	}
	return nil
}

// Start gives the agent a new job, with all its steps to do and its conversation, when it has one
// (PRGRS-B01).
func (b *Board) Start(agent, goal string, steps []string, conv *Conversation) (Progress, error) {
	if err := check(agent); err != nil {
		// PRGRS-E01
		return Progress{}, err
	}
	now := time.Now()
	p := &Progress{Agent: agent, Goal: goal, Started: now, Log: []Entry{{At: now, Kind: "start", Text: goal}}}
	if conv != nil {
		c := *conv
		p.Conversation = &c
	}
	for i, title := range steps {
		p.Steps = append(p.Steps, Step{ID: strconv.Itoa(i + 1), Title: title, State: Todo})
	}
	b.mu.Lock()
	saved := b.commit(agent, p)
	b.mu.Unlock()
	if err := b.save(saved); err != nil {
		return saved, err
	}
	return saved, nil
}

// Step moves a step, by its id or its title, to state, with an optional note (PRGRS-B02).
func (b *Board) Step(agent, step string, state State, note string) error {
	if !state.valid() {
		// PRGRS-E04
		return fmt.Errorf("progress: %q is not a state (todo, doing, done, blocked)", state)
	}
	return b.change(agent, func(p *Progress) error {
		for i := range p.Steps {
			if p.Steps[i].ID == step || p.Steps[i].Title == step {
				p.Steps[i].State = state
				p.Log = append(p.Log, Entry{At: time.Now(), Kind: "step", Step: p.Steps[i].ID, State: state, Text: note})
				return nil
			}
		}
		// PRGRS-E03
		return fmt.Errorf("%w: %q", ErrNoStep, step)
	})
}

// Note logs a note of the agent (PRGRS-B03).
func (b *Board) Note(agent, text string) error {
	return b.change(agent, func(p *Progress) error {
		p.Log = append(p.Log, Entry{At: time.Now(), Kind: "note", Text: text})
		return nil
	})
}

// Finish marks the job finished, with its summary (PRGRS-B04).
func (b *Board) Finish(agent, summary string) error {
	return b.change(agent, func(p *Progress) error {
		p.Finished = true
		p.Log = append(p.Log, Entry{At: time.Now(), Kind: "finish", Text: summary})
		return nil
	})
}

// Get gives the agent's job (PRGRS-B05).
func (b *Board) Get(agent string) (Progress, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.jobs[agent]
	if p == nil {
		return Progress{}, ErrNoJob
	}
	return p.clone(), nil
}

// Version is the board's version: it goes up with each change (PRGRS-B06).
func (b *Board) Version() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.version
}

// List gives every job, the most recently changed first, with the board's version (PRGRS-B05).
func (b *Board) List() (uint64, []Progress) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.version, b.list()
}

func (b *Board) list() []Progress {
	out := make([]Progress, 0, len(b.jobs))
	for _, p := range b.jobs {
		out = append(out, p.clone())
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Updated.Equal(out[j].Updated) {
			return out[i].Updated.After(out[j].Updated)
		}
		return out[i].Agent < out[j].Agent
	})
	return out
}

// Wait gives the list as soon as the board changed since `since`: at once when it already did, else
// at the next change, or when d or ctx end (PRGRS-B06).
func (b *Board) Wait(ctx context.Context, since uint64, d time.Duration) (uint64, []Progress) {
	b.mu.Lock()
	if b.version != since {
		defer b.mu.Unlock()
		return b.version, b.list()
	}
	changed := b.changed
	b.mu.Unlock()
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-changed:
	case <-timer.C:
	case <-ctx.Done():
	}
	return b.List()
}

// Ask records a question to the agent, waiting for its answer (PRGRS-B09).
func (b *Board) Ask(agent, text string) (Question, error) {
	var q Question
	err := b.change(agent, func(p *Progress) error {
		q = Question{ID: "q-" + strconv.Itoa(len(p.Questions)+1), Text: text, Asked: time.Now(), State: Pending}
		p.Questions = append(p.Questions, q)
		return nil
	})
	return q, err
}

// Answer gives the answer to a question waiting for it (PRGRS-B10).
func (b *Board) Answer(agent, question, answer string) error {
	return b.settle(agent, question, Answered, answer)
}

// Fail marks a question that could not be asked, with the reason (PRGRS-B10).
func (b *Board) Fail(agent, question, reason string) error {
	return b.settle(agent, question, Failed, reason)
}

func (b *Board) settle(agent, question string, state QuestionState, text string) error {
	return b.change(agent, func(p *Progress) error {
		for i := range p.Questions {
			if p.Questions[i].ID == question && p.Questions[i].State == Pending {
				p.Questions[i].State, p.Questions[i].Answer, p.Questions[i].Answered = state, text, time.Now()
				return nil
			}
		}
		// PRGRS-E07
		return fmt.Errorf("%w: %q", ErrNoQuestion, question)
	})
}
