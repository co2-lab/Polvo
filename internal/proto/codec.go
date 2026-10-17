// @anchors
//   code: CDIMC
//   ref: CDCOE

// Package proto is the control protocol between the desktop, the node and the local agent
// bridge: JSON-RPC 2.0 messages, one per line, over any ordered and reliable byte stream.
package proto

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// MaxLine is the largest line the decoder accepts (CDCOE constants).
const MaxLine = 16 << 20

// The JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// ErrTooLarge is returned for a line beyond the decoder's limit (CDCOE-E03).
var ErrTooLarge = errors.New("proto: message too large")

// ID identifies a request: a JSON string or a JSON integer. The zero ID is "absent".
type ID struct{ raw string }

// IntID is the integer id n.
func IntID(n int64) ID { return ID{strconv.FormatInt(n, 10)} }

// StringID is the text id s.
func StringID(s string) ID {
	b, _ := json.Marshal(s)
	return ID{string(b)}
}

// NullID is the null id of an error response to a message whose id could not be read
// (CDCOE-B06).
func NullID() ID { return ID{"null"} }

// IsNull reports whether the id is the null id.
func (id ID) IsNull() bool { return id.raw == "null" }

// IsZero reports whether the id is absent.
func (id ID) IsZero() bool { return id.raw == "" }

// Int is the integer value of an integer id.
func (id ID) Int() (int64, bool) {
	n, err := strconv.ParseInt(id.raw, 10, 64)
	return n, err == nil
}

// String is the text value of a text id.
func (id ID) String() (string, bool) {
	var s string
	if len(id.raw) == 0 || id.raw[0] != '"' || json.Unmarshal([]byte(id.raw), &s) != nil {
		return "", false
	}
	return s, true
}

// Equal compares value and type: 7 is not "7" (CDCOE-B03).
func (id ID) Equal(o ID) bool { return id.raw == o.raw }

// Format prints the id as it travels.
func (id ID) Format(f fmt.State, _ rune) { io.WriteString(f, id.raw) }

// parseID reads an id from its JSON text; ok is false for anything but a string or an integer.
func parseID(raw json.RawMessage) (ID, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ID{}, false
	}
	if string(raw) == "null" {
		return NullID(), true
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return ID{}, false
		}
		return StringID(s), true
	}
	n, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return ID{}, false
	}
	return IntID(n), true
}

// Kind is what a message is, from the fields it has (CDCOE-B02).
type Kind int

const (
	Invalid Kind = iota
	Request
	Notification
	Response
)

func (k Kind) String() string {
	switch k {
	case Request:
		return "request"
	case Notification:
		return "notification"
	case Response:
		return "response"
	}
	return "invalid"
}

// Error is the error object of a response.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string { return fmt.Sprintf("proto: %d %s", e.Code, e.Message) }

// Message is a request, a notification or a response.
type Message struct {
	ID     ID
	Method string
	Params json.RawMessage
	Result json.RawMessage
	Error  *Error
}

// Kind classifies the message by its fields (CDCOE-B02).
func (m *Message) Kind() Kind {
	switch {
	// CDCOE-B06: the null id belongs only to an error response.
	case m.ID.IsNull() && (m.Method != "" || m.Error == nil):
		return Invalid
	case m.Method != "" && m.Result == nil && m.Error == nil:
		if m.ID.IsZero() {
			return Notification
		}
		return Request
	case m.Method == "" && !m.ID.IsZero() && (m.Result == nil) != (m.Error == nil):
		return Response
	}
	return Invalid
}

// ReadError is a line that was read but is not a message (CDCOE-E01, CDCOE-E02). The
// decoder stays usable: the next Read starts at the next line.
type ReadError struct {
	ID      ID
	Code    int
	Message string
}

func (e *ReadError) Error() string { return fmt.Sprintf("proto: %d %s", e.Code, e.Message) }

// Decoder reads messages from a stream, one per line.
type Decoder struct {
	r   *bufio.Reader
	max int
}

// NewDecoder reads r with the MaxLine limit.
func NewDecoder(r io.Reader) *Decoder { return NewDecoderSize(r, MaxLine) }

// NewDecoderSize reads r refusing lines longer than max bytes.
func NewDecoderSize(r io.Reader, max int) *Decoder {
	return &Decoder{r: bufio.NewReader(r), max: max}
}

// line returns the next line without its end. A line beyond the limit is discarded up to its
// end and reported as ErrTooLarge (CDCOE-E03).
func (d *Decoder) line() ([]byte, error) {
	var buf []byte
	tooLarge := false
	for {
		chunk, err := d.r.ReadSlice('\n')
		if !tooLarge {
			if len(buf)+len(chunk) > d.max+1 {
				tooLarge, buf = true, nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		switch {
		case err == nil:
			if tooLarge {
				return nil, ErrTooLarge
			}
			return bytes.TrimSuffix(buf[:len(buf)-1], []byte("\r")), nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case err == io.EOF:
			// CDCOE-E04: the end between lines is the normal end; inside one, it is not.
			if len(buf) == 0 && !tooLarge {
				return nil, io.EOF
			}
			return nil, io.ErrUnexpectedEOF
		default:
			return nil, err
		}
	}
}

// Read returns the next message. Errors: io.EOF at the normal end, io.ErrUnexpectedEOF at an
// end inside a line, ErrTooLarge, or a *ReadError for a line that is not a message.
func (d *Decoder) Read() (*Message, error) {
	for {
		ln, err := d.line()
		if err != nil {
			return nil, err
		}
		// CDCOE-B01: an empty line is not a message.
		if len(bytes.TrimSpace(ln)) == 0 {
			continue
		}
		return decode(ln)
	}
}

func decode(ln []byte) (*Message, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(ln, &fields); err != nil {
		// CDCOE-E01 for bytes that are not JSON; CDCOE-B05 and CDCOE-E02 for JSON that is not
		// an object (a batch is a list).
		if !json.Valid(ln) {
			return nil, &ReadError{Code: CodeParseError, Message: "parse error"}
		}
		return nil, &ReadError{Code: CodeInvalidRequest, Message: "invalid request: not an object"}
	}
	invalid := func(id ID, why string) error {
		return &ReadError{ID: id, Code: CodeInvalidRequest, Message: "invalid request: " + why}
	}
	var id ID
	if raw, ok := fields["id"]; ok {
		var good bool
		if id, good = parseID(raw); !good {
			return nil, invalid(ID{}, "the id is neither a string nor an integer")
		}
	}
	var version string
	if json.Unmarshal(fields["jsonrpc"], &version) != nil || version != "2.0" {
		return nil, invalid(id, `the version is not "2.0"`)
	}
	m := &Message{ID: id}
	if raw, ok := fields["method"]; ok {
		if json.Unmarshal(raw, &m.Method) != nil || m.Method == "" {
			return nil, invalid(id, "the method is not a text")
		}
	}
	if raw, ok := fields["params"]; ok {
		m.Params = raw
	}
	if raw, ok := fields["result"]; ok {
		m.Result = raw
	}
	if raw, ok := fields["error"]; ok {
		m.Error = &Error{}
		if json.Unmarshal(raw, m.Error) != nil {
			return nil, invalid(id, "the error is not an error object")
		}
	}
	if m.Kind() == Invalid {
		return nil, invalid(id, "neither a request, a notification nor a response")
	}
	return m, nil
}

// Encoder writes messages to a stream, one per line. It is safe for concurrent use: each
// message reaches the stream in a single write (CDCOE-B04).
type Encoder struct {
	mu sync.Mutex
	w  io.Writer
}

// NewEncoder writes to w.
func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w} }

type wire struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Write sends m. A message outside the domain is refused and nothing is written (CDCOE-E05).
func (e *Encoder) Write(m *Message) error {
	if m.Kind() == Invalid {
		return fmt.Errorf("proto: refusing to write an invalid message (method %q, id %v)", m.Method, m.ID)
	}
	// CDCOE-I01: the wire form carries every field decode reads back, and the id as its raw
	// JSON text, so its type survives the round trip.
	w := wire{JSONRPC: "2.0", Method: m.Method, Params: m.Params, Result: m.Result, Error: m.Error}
	if !m.ID.IsZero() {
		w.ID = json.RawMessage(m.ID.raw)
	}
	b, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("proto: encoding the message: %w", err)
	}
	b = append(b, '\n')
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err = e.w.Write(b)
	return err
}
