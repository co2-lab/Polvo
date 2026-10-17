//go:build !windows

// @anchors
//   code: CLTSX
//   ref: CLIXX

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	cpty "github.com/creack/pty"
)

// syncBuffer is a bytes.Buffer safe for the writer goroutine and the test reading it.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitFor(t *testing.T, b *syncBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !strings.Contains(b.String(), want) {
		if time.Now().After(deadline) {
			t.Fatalf("want %q, got %q", want, b.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func runKeygen(t *testing.T, dir string) string {
	t.Helper()
	var out, errb bytes.Buffer
	if code := Run([]string{"keygen", "--dir", dir}, Stdio{Stdout: &out, Stderr: &errb}); code != 0 {
		t.Fatalf("keygen: %d %s", code, errb.String())
	}
	return out.String()
}

var listening = regexp.MustCompile(`listening on (\S+) \((SHA256:\S+)\)`)

// startNode runs `polvo node` authorizing the desktop of desktopDir, and returns its address.
func startNode(t *testing.T, desktopDir string) string {
	t.Helper()
	nodeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(nodeDir, "authorized_keys"), []byte(runKeygen(t, desktopDir)), 0o600); err != nil {
		t.Fatal(err)
	}
	r, w := io.Pipe()
	go Run([]string{"node", "--dir", nodeDir, "--listen", "127.0.0.1:0"}, Stdio{Stdout: w, Stderr: io.Discard})
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	go io.Copy(io.Discard, r)
	m := listening.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("want the listening line, got %q", line)
	}
	return m[1]
}

func TestCli_keygen(t *testing.T) {
	t.Run("CLIXX-B01: keygen cria a identidade e mostra a mesma linha duas vezes", func(t *testing.T) {
		d := t.TempDir()
		a, b := runKeygen(t, d), runKeygen(t, d)
		if !strings.HasPrefix(a, "ssh-ed25519 ") || a != b {
			t.Fatalf("want the same ssh-ed25519 line, got %q and %q", a, b)
		}
	})
}

func TestCli_node(t *testing.T) {
	t.Run("CLIXX-B02: node escuta no endereço pedido e mostra a impressão digital", func(t *testing.T) {
		addr := startNode(t, t.TempDir())
		if !strings.HasPrefix(addr, "127.0.0.1:") || strings.HasSuffix(addr, ":0") {
			t.Fatalf("want the real address, got %q", addr)
		}
	})
}

func TestCli_connect(t *testing.T) {
	t.Run("CLIXX-B03: connect sem terminal corre a shell com 24 por 80 e devolve o código dela", func(t *testing.T) {
		d := t.TempDir()
		addr := startNode(t, d)
		var out syncBuffer
		code := Run([]string{"connect", "--dir", d, addr}, Stdio{Stdin: strings.NewReader("stty size; exit 4\n"), Stdout: &out, Stderr: io.Discard})
		if code != 4 || !strings.Contains(out.String(), "24 80") {
			t.Fatalf("want 24 80 and code 4, got %d and %q", code, out.String())
		}
	})
	t.Run("CLIXX-B04: o terminal local fica sem eco durante a ligação e volta ao que era", func(t *testing.T) {
		d := t.TempDir()
		addr := startNode(t, d)
		master, tty := openTerminal(t, 24, 80)
		if !echo(t, tty) {
			t.Fatal("want the local terminal with echo before")
		}
		var out syncBuffer
		done := make(chan int, 1)
		go func() {
			done <- Run([]string{"connect", "--dir", d, addr}, Stdio{Stdin: tty, Stdout: &out, Stderr: io.Discard})
		}()
		master.Write([]byte("stty size\n"))
		waitFor(t, &out, "24 80")
		if echo(t, tty) {
			t.Fatal("want no local echo during the session")
		}
		master.Write([]byte("exit 5\n"))
		if code := <-done; code != 5 {
			t.Fatalf("want code 5, got %d", code)
		}
		if !echo(t, tty) {
			t.Fatal("want the echo back after the session")
		}
	})
	t.Run("CLIXX-B05: a janela local de 33 por 99 passa a 44 por 111 e a shell vê-o", func(t *testing.T) {
		d := t.TempDir()
		addr := startNode(t, d)
		master, tty := openTerminal(t, 33, 99)
		var out syncBuffer
		done := make(chan int, 1)
		go func() {
			done <- Run([]string{"connect", "--dir", d, addr}, Stdio{Stdin: tty, Stdout: &out, Stderr: io.Discard})
		}()
		master.Write([]byte("stty size\n"))
		waitFor(t, &out, "33 99")
		cpty.Setsize(master, &cpty.Winsize{Rows: 44, Cols: 111})
		syscall.Kill(os.Getpid(), syscall.SIGWINCH)
		time.Sleep(300 * time.Millisecond)
		master.Write([]byte("stty size; exit\n"))
		waitFor(t, &out, "44 111")
		<-done
	})
}

func TestCli_errors(t *testing.T) {
	t.Run("CLIXX-E01: um subcomando desconhecido mostra o uso e dá 2", func(t *testing.T) {
		var errb bytes.Buffer
		if code := Run([]string{"voar"}, Stdio{Stdout: io.Discard, Stderr: &errb}); code != 2 || !strings.Contains(errb.String(), "usage") {
			t.Fatalf("want usage and 2, got %d %q", code, errb.String())
		}
	})
	t.Run("CLIXX-E01: connect sem endereço mostra o uso e dá 2", func(t *testing.T) {
		var errb bytes.Buffer
		if code := Run([]string{"connect", "--dir", t.TempDir()}, Stdio{Stdout: io.Discard, Stderr: &errb}); code != 2 || !strings.Contains(errb.String(), "usage") {
			t.Fatalf("want usage and 2, got %d %q", code, errb.String())
		}
	})
	t.Run("CLIXX-E02: connect a um endereço sem nó dá o erro e 1", func(t *testing.T) {
		var errb bytes.Buffer
		code := Run([]string{"connect", "--dir", t.TempDir(), "127.0.0.1:1"}, Stdio{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &errb})
		if code != 1 || !strings.HasPrefix(errb.String(), "polvo: ") {
			t.Fatalf("want polvo: and 1, got %d %q", code, errb.String())
		}
	})
}

// openTerminal is a local terminal of the given size: the test types on master, polvo reads tty.
func openTerminal(t *testing.T, rows, cols uint16) (*os.File, *os.File) {
	t.Helper()
	master, tty, err := cpty.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close(); tty.Close() })
	if err := cpty.Setsize(master, &cpty.Winsize{Rows: rows, Cols: cols}); err != nil {
		t.Fatal(err)
	}
	go io.Copy(io.Discard, master)
	return master, tty
}

// echo reads the ECHO bit of the terminal's line discipline (Linux termios: c_lflag at byte 12).
func echo(t *testing.T, tty *os.File) bool {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("the termios layout read here is Linux's")
	}
	var buf [60]byte
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, tty.Fd(), 0x5401, uintptr(unsafe.Pointer(&buf))); e != 0 {
		t.Fatal(e)
	}
	return binary.LittleEndian.Uint32(buf[12:16])&0x8 != 0
}
