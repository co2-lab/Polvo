// @anchors
//   code: CDTST
//   ref: CDCOE

package proto

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// reads every message of the stream, stopping at the first error.
func readAll(t *testing.T, d *Decoder) ([]*Message, error) {
	t.Helper()
	var ms []*Message
	for {
		m, err := d.Read()
		if err != nil {
			return ms, err
		}
		ms = append(ms, m)
	}
}

func wantReadError(t *testing.T, err error, code int) *ReadError {
	t.Helper()
	var re *ReadError
	if !errors.As(err, &re) {
		t.Fatalf("want a *ReadError with code %d, got %v", code, err)
	}
	if re.Code != code {
		t.Fatalf("want code %d, got %d (%s)", code, re.Code, re.Message)
	}
	return re
}

func TestCodec_oneMessagePerLine(t *testing.T) {
	t.Run("CDCOE-B01: Duas mensagens em duas linhas, com uma linha vazia no meio, chegam pela ordem", func(t *testing.T) {
		d := NewDecoder(strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"a\"}\n\n{\"jsonrpc\":\"2.0\",\"method\":\"b\"}\n"))
		ms, err := readAll(t, d)
		if err != io.EOF {
			t.Fatalf("want io.EOF at the end, got %v", err)
		}
		if len(ms) != 2 || ms[0].Method != "a" || ms[1].Method != "b" {
			t.Fatalf("want notifications a then b, got %+v", ms)
		}
	})
}

func TestCodec_kindFromFields(t *testing.T) {
	t.Run("CDCOE-B02: O pedido, a notificação e a resposta distinguem-se pelos campos", func(t *testing.T) {
		in := `{"jsonrpc":"2.0","id":1,"method":"fs.read","params":{"path":"/x"}}` + "\n" +
			`{"jsonrpc":"2.0","method":"pty.output","params":["oi"]}` + "\n" +
			`{"jsonrpc":"2.0","id":1,"result":"ok"}` + "\n"
		ms, err := readAll(t, NewDecoder(strings.NewReader(in)))
		if err != io.EOF || len(ms) != 3 {
			t.Fatalf("want 3 messages and io.EOF, got %d and %v", len(ms), err)
		}
		if ms[0].Kind() != Request || ms[1].Kind() != Notification || ms[2].Kind() != Response {
			t.Fatalf("want request, notification, response; got %v %v %v", ms[0].Kind(), ms[1].Kind(), ms[2].Kind())
		}
		if string(ms[2].Result) != `"ok"` {
			t.Fatalf("want the result \"ok\", got %s", ms[2].Result)
		}
	})
}

func TestCodec_idKeepsItsType(t *testing.T) {
	t.Run("CDCOE-B03: O identificador 7 e o identificador \"7\" são diferentes", func(t *testing.T) {
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		if err := e.Write(&Message{ID: IntID(7), Method: "a"}); err != nil {
			t.Fatal(err)
		}
		if err := e.Write(&Message{ID: StringID("7"), Method: "a"}); err != nil {
			t.Fatal(err)
		}
		ms, _ := readAll(t, NewDecoder(&buf))
		if len(ms) != 2 {
			t.Fatalf("want 2 messages, got %d", len(ms))
		}
		if n, ok := ms[0].ID.Int(); !ok || n != 7 {
			t.Fatalf("want the integer 7, got %s", ms[0].ID)
		}
		if s, ok := ms[1].ID.String(); !ok || s != "7" {
			t.Fatalf("want the text \"7\", got %s", ms[1].ID)
		}
		if ms[0].ID.Equal(ms[1].ID) {
			t.Fatal("the integer 7 and the text \"7\" must differ")
		}
	})
}

func TestCodec_concurrentWriters(t *testing.T) {
	t.Run("CDCOE-B04: Cem escritores ao mesmo tempo produzem cem linhas inteiras", func(t *testing.T) {
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		param := []byte(`"` + strings.Repeat("x", 4096) + `"`)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := e.Write(&Message{Method: "n", Params: param}); err != nil {
					t.Error(err)
				}
			}()
		}
		wg.Wait()
		ms, err := readAll(t, NewDecoder(&buf))
		if err != io.EOF || len(ms) != 100 {
			t.Fatalf("want 100 notifications and io.EOF, got %d and %v", len(ms), err)
		}
		for _, m := range ms {
			if !bytes.Equal(m.Params, param) {
				t.Fatal("a parameter arrived cut or mixed")
			}
		}
	})
}

func TestCodec_batchRefused(t *testing.T) {
	t.Run("CDCOE-B05: Um lote de dois pedidos é recusado como pedido inválido", func(t *testing.T) {
		d := NewDecoder(strings.NewReader(`[{"jsonrpc":"2.0","id":1,"method":"a"},{"jsonrpc":"2.0","id":2,"method":"b"}]` + "\n"))
		_, err := d.Read()
		wantReadError(t, err, CodeInvalidRequest)
	})
}

func TestCodec_errors(t *testing.T) {
	t.Run("CDCOE-E01: Uma linha que não é JSON dá erro de parse e a seguinte ainda é lida", func(t *testing.T) {
		d := NewDecoder(strings.NewReader("{\"jsonrpc\":\n{\"jsonrpc\":\"2.0\",\"method\":\"b\"}\n"))
		_, err := d.Read()
		wantReadError(t, err, CodeParseError)
		m, err := d.Read()
		if err != nil || m.Method != "b" {
			t.Fatalf("want the notification b, got %+v %v", m, err)
		}
	})
	t.Run("CDCOE-E02: Um objeto sem método nem resultado dá pedido inválido com o identificador lido", func(t *testing.T) {
		d := NewDecoder(strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":9}\n{\"jsonrpc\":\"2.0\",\"method\":\"b\"}\n"))
		_, err := d.Read()
		re := wantReadError(t, err, CodeInvalidRequest)
		if n, ok := re.ID.Int(); !ok || n != 9 {
			t.Fatalf("want the id 9 in the error, got %s", re.ID)
		}
		m, err := d.Read()
		if err != nil || m.Method != "b" {
			t.Fatalf("want the notification b, got %+v %v", m, err)
		}
	})
	t.Run("CDCOE-E02: Uma mensagem sem a versão 2.0 dá pedido inválido", func(t *testing.T) {
		d := NewDecoder(strings.NewReader("{\"jsonrpc\":\"1.0\",\"id\":3,\"method\":\"a\"}\n"))
		_, err := d.Read()
		re := wantReadError(t, err, CodeInvalidRequest)
		if n, ok := re.ID.Int(); !ok || n != 3 {
			t.Fatalf("want the id 3 in the error, got %s", re.ID)
		}
	})
	t.Run("CDCOE-E03: Uma linha maior do que o limite é descartada e a seguinte ainda é lida", func(t *testing.T) {
		in := strings.Repeat("y", 200) + "\n{\"jsonrpc\":\"2.0\",\"method\":\"b\"}\n"
		d := NewDecoderSize(strings.NewReader(in), 64)
		if _, err := d.Read(); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("want ErrTooLarge, got %v", err)
		}
		m, err := d.Read()
		if err != nil || m.Method != "b" {
			t.Fatalf("want the notification b, got %+v %v", m, err)
		}
	})
	t.Run("CDCOE-E04: O fluxo que acaba a meio de uma linha dá fim inesperado", func(t *testing.T) {
		_, err := NewDecoder(strings.NewReader(`{"jsonrpc":"2.0","met`)).Read()
		if err != io.ErrUnexpectedEOF {
			t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
		}
	})
	t.Run("CDCOE-E04: O fluxo que acaba entre linhas dá o fim normal", func(t *testing.T) {
		d := NewDecoder(strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"a\"}\n"))
		if m, err := d.Read(); err != nil || m.Method != "a" {
			t.Fatalf("want the notification a, got %+v %v", m, err)
		}
		if _, err := d.Read(); err != io.EOF {
			t.Fatalf("want io.EOF, got %v", err)
		}
	})
	t.Run("CDCOE-E05: Uma resposta com resultado e erro ao mesmo tempo não é escrita", func(t *testing.T) {
		var buf bytes.Buffer
		err := NewEncoder(&buf).Write(&Message{ID: IntID(1), Result: []byte(`"ok"`), Error: &Error{Code: -1, Message: "x"}})
		if err == nil {
			t.Fatal("want the write refused")
		}
		if buf.Len() != 0 {
			t.Fatalf("want an empty stream, got %q", buf.String())
		}
	})
}

func TestCodec_roundTrip(t *testing.T) {
	t.Run("CDCOE-I01: Escrever e ler devolve a mesma mensagem", func(t *testing.T) {
		req := &Message{ID: StringID("abc"), Method: "fs.write", Params: []byte(`{"path":"/tmp/x","data":"oi"}`)}
		resp := &Message{ID: IntID(5), Error: &Error{Code: CodeMethodNotFound, Message: "method not found"}}
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		for _, m := range []*Message{req, resp} {
			if err := e.Write(m); err != nil {
				t.Fatal(err)
			}
		}
		ms, err := readAll(t, NewDecoder(&buf))
		if err != io.EOF || len(ms) != 2 {
			t.Fatalf("want 2 messages and io.EOF, got %d and %v", len(ms), err)
		}
		r := ms[0]
		if r.Kind() != Request || !r.ID.Equal(req.ID) || r.Method != req.Method || !bytes.Equal(r.Params, req.Params) {
			t.Fatalf("the request changed: %+v", r)
		}
		s := ms[1]
		if s.Kind() != Response || !s.ID.Equal(resp.ID) || s.Error == nil ||
			s.Error.Code != resp.Error.Code || s.Error.Message != resp.Error.Message || s.Result != nil {
			t.Fatalf("the error response changed: %+v", s)
		}
	})
}
