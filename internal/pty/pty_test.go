// @anchors
//   code: PTTST
//   ref: PTYXX

package pty

import (
	"bytes"
	"io"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func skipWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("ConPTY: @TBD na spec")
	}
}

func start(t *testing.T, size Size, argv ...string) *Session {
	t.Helper()
	s, err := Start(argv, size)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func readAll(t *testing.T, s *Session) string {
	t.Helper()
	b, err := io.ReadAll(s)
	if err != nil {
		t.Fatalf("want the normal end, got %v", err)
	}
	return strings.ReplaceAll(string(b), "\r", "")
}

// readUntil reads until want appears, so a write lands after the program asked for input.
func readUntil(t *testing.T, s *Session, want string) string {
	t.Helper()
	var buf bytes.Buffer
	b := make([]byte, 256)
	for !strings.Contains(buf.String(), want) {
		n, err := s.Read(b)
		buf.Write(b[:n])
		if err != nil {
			t.Fatalf("want %q before the end, got %q and %v", want, buf.String(), err)
		}
	}
	return buf.String()
}

func TestPty_start(t *testing.T) {
	skipWindows(t)
	t.Run("PTYXX-B01: A shell arranca num terminal de 30 linhas por 100 colunas", func(t *testing.T) {
		s := start(t, Size{Rows: 30, Cols: 100}, "sh", "-c", "tty >/dev/null && echo tem-tty; stty size")
		out := readAll(t, s)
		if !strings.Contains(out, "tem-tty") || !strings.Contains(out, "30 100") {
			t.Fatalf("want a tty of 30x100, got %q", out)
		}
	})
	t.Run("PTYXX-B02: O texto escrito na sessão volta pela leitura", func(t *testing.T) {
		s := start(t, Size{}, "sh", "-c", "echo pronto; read l; echo eco:$l")
		readUntil(t, s, "pronto")
		if _, err := s.Write([]byte("ola\n")); err != nil {
			t.Fatal(err)
		}
		if out := readAll(t, s); !strings.Contains(out, "eco:ola") {
			t.Fatalf("want eco:ola, got %q", out)
		}
	})
	t.Run("PTYXX-B03: O redimensionamento para 40 por 120 chega ao processo", func(t *testing.T) {
		s := start(t, Size{Rows: 24, Cols: 80}, "sh", "-c", "echo pronto; read l; stty size")
		readUntil(t, s, "pronto")
		if err := s.Resize(Size{Rows: 40, Cols: 120}); err != nil {
			t.Fatal(err)
		}
		s.Write([]byte("\n"))
		if out := readAll(t, s); !strings.Contains(out, "40 120") {
			t.Fatalf("want 40 120, got %q", out)
		}
	})
	t.Run("PTYXX-I01: O tamanho visto é o do último redimensionamento", func(t *testing.T) {
		s := start(t, Size{Rows: 10, Cols: 20}, "sh", "-c", "stty size; read l; stty size")
		first := readUntil(t, s, "10 20")
		if err := s.Resize(Size{Rows: 11, Cols: 21}); err != nil {
			t.Fatal(err)
		}
		s.Write([]byte("\n"))
		rest := readAll(t, s)
		if !strings.Contains(first, "10 20") || !strings.Contains(rest, "11 21") {
			t.Fatalf("want 10 20 then 11 21, got %q and %q", first, rest)
		}
	})
}

func TestPty_end(t *testing.T) {
	skipWindows(t)
	t.Run("PTYXX-B04: O código de saída 3 volta pelo fim do processo", func(t *testing.T) {
		s := start(t, Size{}, "sh", "-c", "exit 3")
		readAll(t, s)
		st, err := s.Wait()
		if err != nil || st.Code != 3 {
			t.Fatalf("want code 3, got %+v %v", st, err)
		}
	})
	t.Run("PTYXX-B04: Um processo morto por um sinal devolve -1 e o nome do sinal", func(t *testing.T) {
		s := start(t, Size{}, "sleep", "30")
		if err := s.Signal(syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		st, err := s.Wait()
		if err != nil || st.Code != -1 || st.Signal != "killed" {
			t.Fatalf("want -1 and killed, got %+v %v", st, err)
		}
	})
	t.Run("PTYXX-B05: A leitura acaba com o fim normal depois do processo", func(t *testing.T) {
		s := start(t, Size{}, "echo", "fim")
		if out := readAll(t, s); !strings.Contains(out, "fim") {
			t.Fatalf("want fim, got %q", out)
		}
	})
	t.Run("PTYXX-B06: Fechar a sessão termina a shell", func(t *testing.T) {
		s := start(t, Size{}, "sleep", "30")
		time.Sleep(50 * time.Millisecond)
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		st, err := s.Wait()
		if err != nil || st.Signal == "" {
			t.Fatalf("want the process ended by a signal, got %+v %v", st, err)
		}
	})
}

func TestPty_errors(t *testing.T) {
	skipWindows(t)
	t.Run("PTYXX-E01: Um programa que não existe falha com o nome dele", func(t *testing.T) {
		_, err := Start([]string{"nao-existe-polvo"}, Size{})
		if err == nil || !strings.Contains(err.Error(), "nao-existe-polvo") {
			t.Fatalf("want an error naming the program, got %v", err)
		}
	})
	t.Run("PTYXX-E02: Sem programa o arranque falha", func(t *testing.T) {
		if _, err := Start(nil, Size{}); err == nil {
			t.Fatal("want an error")
		}
	})
}
