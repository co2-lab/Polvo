// @anchors
//   code: CLIMQ
//   ref: CLDAG

// Package agents asks the agents of the sessions about their work: the claude by its CLI, in a copy
// of its conversation.
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/co2-lab/polvo/internal/progress"
)

// CLAUDE_BIN is Claude Code's CLI.
const CLAUDE_BIN = "claude"

// ASK_TIMEOUT is the longest a question waits for the claude (CLDAG-E02).
const ASK_TIMEOUT = 5 * time.Minute

// TAIL_MAX is how much of the end of a conversation is read for its model (CLDAG-B01).
const TAIL_MAX = 1 << 20

var notInSlug = regexp.MustCompile(`[^A-Za-z0-9-]`)

// slug is the name Claude Code gives the folder of a project's conversations: the path with every
// character that is not a letter, a digit or a hyphen made a hyphen.
func slug(dir string) string { return notInSlug.ReplaceAllString(dir, "-") }

var modelField = regexp.MustCompile(`"model":"([^"]+)"`)

// modelOf is the model of the last assistant message of the conversation in file, read from its end.
func modelOf(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size() > TAIL_MAX {
		f.Seek(st.Size()-TAIL_MAX, io.SeekStart)
	}
	tail, _ := io.ReadAll(io.LimitReader(f, TAIL_MAX))
	lines := bytes.Split(tail, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if !bytes.Contains(lines[i], []byte(`"type":"assistant"`)) {
			continue
		}
		if m := modelField.FindSubmatch(lines[i]); m != nil {
			return string(m[1])
		}
	}
	return ""
}

// Find is the conversation of the claude that runs in dir: the one of id when it is there, else the
// most recently changed (CLDAG-B01).
func Find(home, dir, id string) (progress.Conversation, error) {
	folder := filepath.Join(home, ".claude", "projects", slug(dir))
	pick := ""
	if id != "" {
		if _, err := os.Stat(filepath.Join(folder, id+".jsonl")); err == nil {
			pick = id
		}
	}
	if pick == "" {
		files, _ := filepath.Glob(filepath.Join(folder, "*.jsonl"))
		var newest time.Time
		for _, f := range files {
			if st, err := os.Stat(f); err == nil && st.ModTime().After(newest) {
				newest, pick = st.ModTime(), strings.TrimSuffix(filepath.Base(f), ".jsonl")
			}
		}
	}
	if pick == "" {
		// CLDAG-E01
		return progress.Conversation{}, fmt.Errorf("agents: no conversation of claude for %s (in %s)", dir, folder)
	}
	model := modelOf(filepath.Join(folder, pick+".jsonl"))
	return progress.Conversation{Kind: "claude", ID: pick, Dir: dir, Model: model}, nil
}

// prompt frames the question for the copy of the conversation.
func prompt(question string) string {
	return "Polvo, the environment you work in, asks you about the work of this conversation. " +
		"Answer only this, in a few sentences, without tools, and change nothing:\n\n" + question
}

// Ask asks the claude of c, by its CLI, in a copy of the conversation (CLDAG-B02).
func Ask(bin string, c progress.Conversation, question string) (string, error) {
	args := []string{"-p", "--resume", c.ID, "--fork-session", "--output-format", "json", "--permission-mode", "plan"}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	args = append(args, prompt(question))
	ctx, cancel := context.WithTimeout(context.Background(), ASK_TIMEOUT)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = c.Dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	var r struct {
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}
	if jerr := json.Unmarshal(out.Bytes(), &r); jerr == nil && r.IsError {
		// CLDAG-E02: the claude's own reason
		return "", fmt.Errorf("agents: claude answered with an error: %s", r.Result)
	} else if jerr == nil && err == nil {
		return r.Result, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("agents: claude did not answer in %s", ASK_TIMEOUT)
	}
	if err != nil {
		return "", fmt.Errorf("agents: running %s: %w: %s", bin, err, strings.TrimSpace(errb.String()))
	}
	return "", fmt.Errorf("agents: claude answered what is not JSON: %q", strings.TrimSpace(out.String()))
}
