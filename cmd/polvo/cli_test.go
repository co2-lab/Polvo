//go:build !windows

// @anchors
//   code: CLTSX
//   ref: CLIXX

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/sshx"
	"io"
	"net"
	"net/http"
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

	"golang.org/x/net/websocket"
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
	return startNodeWith(t, desktopDir, "")
}

// startNodeWith is startNode with the node.json conf, when it is not empty.
func startNodeWith(t *testing.T, desktopDir, conf string) string {
	t.Helper()
	nodeDir := t.TempDir()
	if conf != "" {
		os.WriteFile(filepath.Join(nodeDir, "node.json"), []byte(conf), 0o600)
	}
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
	t.Run("CLIXX-B02: as sessões do node sabem onde ele está por dentro da máquina", func(t *testing.T) {
		var got []string
		for _, a := range []string{"[::]:2222", "0.0.0.0:2222", "10.0.0.5:2222"} {
			addr, err := net.ResolveTCPAddr("tcp", a)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, loopback(addr))
		}
		if strings.Join(got, " ") != "127.0.0.1:2222 127.0.0.1:2222 10.0.0.5:2222" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("CLIXX-B02: node escreve o pedido de emparelhamento com o código", func(t *testing.T) {
		desktop := t.TempDir()
		runKeygen(t, desktop)
		r, w := io.Pipe()
		go Run([]string{"node", "--dir", t.TempDir(), "--listen", "127.0.0.1:0"}, Stdio{Stdout: w, Stderr: io.Discard})
		lines := bufio.NewReader(r)
		first, err := lines.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		addr := listening.FindStringSubmatch(first)[1]
		id, _, err := sshx.LoadOrCreateIdentity(filepath.Join(desktop, "id_ed25519"))
		if err != nil {
			t.Fatal(err)
		}
		code := regexp.MustCompile(`pairing request from portatil \(SHA256:[^)]+\): code (\d{6})`)
		typed := func() (string, error) {
			for {
				line, err := lines.ReadString('\n')
				if err != nil {
					return "", err
				}
				if m := code.FindStringSubmatch(line); m != nil {
					go io.Copy(io.Discard, r)
					return m[1], nil
				}
			}
		}
		cfg := hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(desktop, "known_hosts"))}
		if err := hub.Pair(addr, cfg, "portatil", typed); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("CLIXX-B02: node não arranca com um node.json estragado", func(t *testing.T) {
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "node.json"), []byte("isto não é json"), 0o600)
		var errb bytes.Buffer
		code := Run([]string{"node", "--dir", d, "--listen", "127.0.0.1:0"}, Stdio{Stdout: io.Discard, Stderr: &errb})
		if code != 1 || !strings.HasPrefix(errb.String(), "polvo: ") || !strings.Contains(errb.String(), "node.json") {
			t.Fatalf("want polvo: with node.json and 1, got %d %q", code, errb.String())
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

var forwarding = regexp.MustCompile(`polvo forward: (\S+) -> `)

func TestCli_forward(t *testing.T) {
	t.Run("CLIXX-B08: forward encaminha um socket pela máquina de um nó", func(t *testing.T) {
		d := t.TempDir()
		addr := startNodeWith(t, d, `{"forward":["*"]}`)
		target, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer target.Close()
		go func() {
			if c, err := target.Accept(); err == nil {
				io.Copy(c, c)
				c.Close()
			}
		}()
		var out syncBuffer
		go Run([]string{"forward", "--dir", d, "127.0.0.1:0", target.Addr().String(), addr}, Stdio{Stdout: &out, Stderr: io.Discard})
		waitFor(t, &out, " -> ")
		m := forwarding.FindStringSubmatch(out.String())
		c, err := net.Dial("tcp", m[1])
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(10 * time.Second))
		c.Write([]byte("ola"))
		b := make([]byte, 3)
		io.ReadFull(c, b)
		if string(b) != "ola" {
			t.Fatalf("want ola back, got %q", b)
		}
	})
}

func TestCli_mcp(t *testing.T) {
	t.Run("CLIXX-B06: mcp inicializa, lista as treze ferramentas e corre um comando num nó", func(t *testing.T) {
		d := t.TempDir()
		addr := startNode(t, d)
		in := strings.Join([]string{
			`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
			`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
			`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"polvo_run","arguments":{"node":"` + addr + `","command":"echo do-agente"}}}`,
		}, "\n") + "\n"
		var out syncBuffer
		code := Run([]string{"mcp", "--dir", d}, Stdio{Stdin: strings.NewReader(in), Stdout: &out, Stderr: io.Discard})
		if code != 0 {
			t.Fatalf("want 0, got %d", code)
		}
		answers := map[float64]map[string]any{}
		for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			var m map[string]any
			if err := json.Unmarshal([]byte(l), &m); err != nil {
				t.Fatalf("not a message: %q", l)
			}
			answers[m["id"].(float64)] = m["result"].(map[string]any)
		}
		info := answers[1]["serverInfo"].(map[string]any)
		if info["name"] != "polvo" {
			t.Fatalf("want the server polvo, got %v", info)
		}
		var names []string
		for _, tl := range answers[2]["tools"].([]any) {
			names = append(names, tl.(map[string]any)["name"].(string))
		}
		want := "polvo_run,polvo_read_file,polvo_write_file,polvo_list," +
			"polvo_sessions,polvo_screen,polvo_type,polvo_wait," +
			"polvo_progress_start,polvo_progress_step,polvo_progress_note,polvo_progress_answer,polvo_progress_finish"
		if strings.Join(names, ",") != want {
			t.Fatalf("want the thirteen tools, got %v", names)
		}
		text := answers[3]["content"].([]any)[0].(map[string]any)["text"].(string)
		if !strings.Contains(text, "do-agente") || answers[3]["isError"] != false {
			t.Fatalf("want do-agente, got %v", answers[3])
		}
	})
}

var uiURL = regexp.MustCompile(`(http://127\.0\.0\.1:\d+)/#t=(\w+)`)

// startUI runs the UI command with args and returns the base URL and the secret it printed.
func startUI(t *testing.T, args []string) (string, string, chan string) {
	t.Helper()
	opened := make(chan string, 1)
	old := openBrowser
	openBrowser = func(u string) error { opened <- u; return nil }
	t.Cleanup(func() { openBrowser = old })
	r, w := io.Pipe()
	go Run(args, Stdio{Stdout: w, Stderr: io.Discard})
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	go io.Copy(io.Discard, r)
	m := uiURL.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("want the UI address, got %q", line)
	}
	return m[1], m[2], opened
}

func TestCli_ui(t *testing.T) {
	t.Run("CLIXX-B07: ui escreve o endereço com o segredo, abre o browser e serve a UI e as chamadas", func(t *testing.T) {
		d := t.TempDir()
		addr := startNode(t, d)
		base, tok, opened := startUI(t, []string{"ui", "--dir", d, "--listen", "127.0.0.1:0"})
		select {
		case u := <-opened:
			if u != base+"/#t="+tok {
				t.Fatalf("want the browser on %s/#t=%s, got %s", base, tok, u)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the browser was not opened")
		}
		resp, err := http.Get(base + "/")
		if err != nil {
			t.Fatal(err)
		}
		page, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(page), "<title>Polvo</title>") {
			t.Fatalf("want the UI page, got %q", page)
		}
		body, _ := json.Marshal(map[string]any{"node": addr, "method": "fs.list", "params": map[string]string{"path": d}})
		req, _ := http.NewRequest(http.MethodPost, base+"/api/call", bytes.NewReader(body))
		req.Header.Set("X-Polvo-Token", tok)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(b), `"id_ed25519"`) {
			t.Fatalf("want the entries of the folder, got %s", b)
		}
	})
	t.Run("CLIXX-B07: o terminal local da UI corre a shell do SHELL", func(t *testing.T) {
		sh := filepath.Join(t.TempDir(), "shell")
		os.WriteFile(sh, []byte("#!/bin/sh\necho shell-do-SHELL\nexit 3\n"), 0o755)
		t.Setenv("SHELL", sh)
		base, tok, _ := startUI(t, []string{"ui", "--dir", t.TempDir(), "--no-browser"})
		c, err := websocket.Dial(strings.Replace(base, "http://", "ws://", 1)+"/ws/term?t="+tok+"&node=local", "", base)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(10 * time.Second))
		var out bytes.Buffer
		for {
			var frame []byte
			if err := websocket.Message.Receive(c, &frame); err != nil {
				break
			}
			out.Write(frame)
		}
		if !strings.Contains(out.String(), "shell-do-SHELL") || !strings.Contains(out.String(), `"exit":3`) {
			t.Fatalf("want the SHELL's output and exit 3, got %q", out.String())
		}
	})
	t.Run("CLIXX-B07: polvo sem subcomando é a UI, e --no-browser não abre o browser", func(t *testing.T) {
		_, _, opened := startUI(t, []string{"--dir", t.TempDir(), "--no-browser"})
		select {
		case u := <-opened:
			t.Fatalf("want no browser, got %s", u)
		case <-time.After(300 * time.Millisecond):
		}
	})
	t.Run("CLIXX-B07: ui recusa escutar num endereço que não é local", func(t *testing.T) {
		var errb bytes.Buffer
		code := Run([]string{"ui", "--dir", t.TempDir(), "--listen", "0.0.0.0:0"}, Stdio{Stdout: io.Discard, Stderr: &errb})
		if code != 1 || !strings.HasPrefix(errb.String(), "polvo: ") {
			t.Fatalf("want polvo: and 1, got %d %q", code, errb.String())
		}
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

var between = regexp.MustCompile(`reachable between networks at (\S+)`)

func TestCli_p2p(t *testing.T) {
	t.Run("CLIXX-B09: Entre redes", func(t *testing.T) {
		d, nodeDir := t.TempDir(), t.TempDir()
		os.WriteFile(filepath.Join(nodeDir, "authorized_keys"), []byte(runKeygen(t, d)), 0o600)
		var out syncBuffer
		go Run([]string{"node", "--dir", nodeDir, "--listen", "127.0.0.1:0", "--relay", "", "--p2p-port", "0"}, Stdio{Stdout: &out, Stderr: io.Discard})
		waitFor(t, &out, "listening on")
		time.Sleep(300 * time.Millisecond)
		m := between.FindStringSubmatch(out.String())
		if m == nil {
			t.Skip("built without the transport (make test-p2p runs these)")
		}
		var got syncBuffer
		code := Run([]string{"connect", "--dir", d, m[1]}, Stdio{Stdin: strings.NewReader("echo marca-$((40+2)); exit 4\n"), Stdout: &got, Stderr: io.Discard})
		if code != 4 || !strings.Contains(got.String(), "marca-42") {
			t.Fatalf("want marca-42 and code 4, got %d and %q", code, got.String())
		}
	})
}
