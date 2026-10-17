// @anchors
//   code: CNIMP
//   ref: CNTRL

package node

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"syscall"

	"github.com/co2-lab/polvo/internal/proto"
)

// MaxFile is the largest file fs.read returns (CNTRL-E04).
const MaxFile = 8 << 20

// CodeOperation is the error of an operation the system refused (CNTRL-E03).
const CodeOperation = -32000

type opError struct{ error }

type paramsError struct{ error }

// ServeControl speaks the control protocol over rw until it ends.
func ServeControl(rw io.ReadWriteCloser) {
	defer rw.Close()
	enc, dec := proto.NewEncoder(rw), proto.NewDecoder(rw)
	for {
		m, err := dec.Read()
		var re *proto.ReadError
		switch {
		case errors.As(err, &re):
			// CNTRL-E05: a line that is not a message is answered, with its id when there is one,
			// and the session goes on.
			id := re.ID
			if id.IsZero() {
				id = proto.NullID()
			}
			enc.Write(&proto.Message{ID: id, Error: &proto.Error{Code: re.Code, Message: re.Message}})
			continue
		case errors.Is(err, proto.ErrTooLarge):
			enc.Write(&proto.Message{ID: proto.NullID(), Error: &proto.Error{Code: proto.CodeInvalidRequest, Message: err.Error()}})
			continue
		case err != nil:
			return
		}
		if m.Kind() != proto.Request {
			// CNTRL-B06: a notification has no answer, not even an error.
			continue
		}
		// CNTRL-B05: each request is answered on its own, so a slow one holds nobody.
		go func(m *proto.Message) { enc.Write(Answer(m)) }(m)
	}
}

// Answer is the answer to one request, made here, without a session (the UI's local node).
func Answer(m *proto.Message) *proto.Message {
	var res any
	var err error
	switch m.Method {
	case "fs.read":
		res, err = fsRead(m.Params)
	case "fs.write":
		res, err = fsWrite(m.Params)
	case "fs.list":
		res, err = fsList(m.Params)
	case "proc.run":
		res, err = procRun(m.Params)
	default:
		// CNTRL-E01
		return &proto.Message{ID: m.ID, Error: &proto.Error{Code: proto.CodeMethodNotFound, Message: "method not found: " + m.Method}}
	}
	var pe paramsError
	var oe opError
	switch {
	case errors.As(err, &pe):
		return &proto.Message{ID: m.ID, Error: &proto.Error{Code: proto.CodeInvalidParams, Message: pe.Error()}}
	case errors.As(err, &oe):
		return &proto.Message{ID: m.ID, Error: &proto.Error{Code: CodeOperation, Message: oe.Error()}}
	case err != nil:
		return &proto.Message{ID: m.ID, Error: &proto.Error{Code: proto.CodeInternalError, Message: err.Error()}}
	}
	b, err := json.Marshal(res)
	if err != nil {
		return &proto.Message{ID: m.ID, Error: &proto.Error{Code: proto.CodeInternalError, Message: err.Error()}}
	}
	return &proto.Message{ID: m.ID, Result: b}
}

// params decodes the request's params into v; a wrong shape is CNTRL-E02.
func params(raw json.RawMessage, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return paramsError{fmt.Errorf("invalid params: %w", err)}
	}
	return nil
}

func decode64(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, paramsError{fmt.Errorf("invalid params: the data is not base64: %w", err)}
	}
	return b, nil
}

type pathParams struct {
	Path string `json:"path"`
	Data string `json:"data"`
}

func pathOf(raw json.RawMessage) (pathParams, error) {
	var p pathParams
	if err := params(raw, &p); err != nil {
		return p, err
	}
	if p.Path == "" {
		return p, paramsError{errors.New("invalid params: no path")}
	}
	return p, nil
}

// fsRead is CNTRL-B01.
func fsRead(raw json.RawMessage) (any, error) {
	p, err := pathOf(raw)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(p.Path)
	if err != nil {
		return nil, opError{err}
	}
	// CNTRL-E04: a file beyond the limit is refused before it is read.
	if st.Size() > MaxFile {
		return nil, opError{fmt.Errorf("%s: too large (%d bytes, the limit is %d)", p.Path, st.Size(), MaxFile)}
	}
	b, err := os.ReadFile(p.Path)
	if err != nil {
		return nil, opError{err}
	}
	return map[string]any{"data": base64.StdEncoding.EncodeToString(b), "size": len(b)}, nil
}

// fsWrite is CNTRL-B02.
func fsWrite(raw json.RawMessage) (any, error) {
	p, err := pathOf(raw)
	if err != nil {
		return nil, err
	}
	b, err := decode64(p.Data)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(p.Path, b, 0o644); err != nil {
		return nil, opError{err}
	}
	return struct{}{}, nil
}

type entry struct {
	Name string `json:"name"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
}

// fsList is CNTRL-B03.
func fsList(raw json.RawMessage) (any, error) {
	p, err := pathOf(raw)
	if err != nil {
		return nil, err
	}
	des, err := os.ReadDir(p.Path)
	if err != nil {
		return nil, opError{err}
	}
	es := make([]entry, 0, len(des))
	for _, d := range des {
		e := entry{Name: d.Name(), Dir: d.IsDir()}
		if info, err := d.Info(); err == nil && !d.IsDir() {
			e.Size = info.Size()
		}
		es = append(es, e)
	}
	sort.Slice(es, func(i, j int) bool { return es[i].Name < es[j].Name })
	return map[string]any{"entries": es}, nil
}

type runParams struct {
	Argv  []string `json:"argv"`
	Dir   string   `json:"dir"`
	Stdin string   `json:"stdin"`
}

// procRun is CNTRL-B04.
func procRun(raw json.RawMessage) (any, error) {
	var p runParams
	if err := params(raw, &p); err != nil {
		return nil, err
	}
	if len(p.Argv) == 0 {
		return nil, paramsError{errors.New("invalid params: no program")}
	}
	in, err := decode64(p.Stdin)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(p.Argv[0], p.Argv[1:]...)
	cmd.Dir = p.Dir
	var out, errb bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(in), &out, &errb
	err = cmd.Run()
	code, signal := 0, ""
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			code, signal = -1, ws.Signal().String()
		} else {
			code = ee.ExitCode()
		}
	default:
		// CNTRL-E03: the program did not start.
		return nil, opError{err}
	}
	return map[string]any{
		"code": code, "signal": signal,
		"stdout": base64.StdEncoding.EncodeToString(out.Bytes()),
		"stderr": base64.StdEncoding.EncodeToString(errb.Bytes()),
	}, nil
}
