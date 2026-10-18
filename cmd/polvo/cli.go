// @anchors
//   code: CLIMX
//   ref: CLIXX

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/mcp"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/nodeconf"
	"github.com/co2-lab/polvo/internal/progress"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sessions"
	"github.com/co2-lab/polvo/internal/sshx"
	"github.com/co2-lab/polvo/internal/web"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Stdio is where a run reads and writes.
type Stdio struct {
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

const usage = `usage:
  polvo keygen  [--dir DIR]                     create this installation's identity
  polvo node    [--dir DIR] [--listen ADDR]     serve this machine as a node
  polvo connect [--dir DIR] ADDR                open a shell on a node
  polvo mcp     [--dir DIR]                     serve the MCP bridge of the local agent on stdin/stdout
  polvo [ui]    [--dir DIR] [--listen ADDR] [--no-browser]
                                                open the desktop application (the default)
`

// version is this executable's version, the one the MCP server reports (CLIXX-B06).
const version = "0.1.0"

// errUsage is CLIXX-E01: the usage on stderr and the exit code 2.
var errUsage = errors.New("usage")

// Run runs the polvo command line and returns the exit code.
func Run(args []string, std Stdio) int {
	err := errUsage
	code := 0
	// CLIXX-B07: no subcommand (only options, or nothing) is the desktop application.
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		args = append([]string{"ui"}, args...)
	}
	if len(args) > 0 {
		switch args[0] {
		case "ui":
			err = serveUI(args[1:], std)
		case "keygen":
			err = keygen(args[1:], std)
		case "node":
			err = serveNode(args[1:], std)
		case "connect":
			code, err = connect(args[1:], std)
		case "mcp":
			err = serveMCP(args[1:], std)
		}
	}
	switch {
	case errors.Is(err, errUsage):
		fmt.Fprint(std.Stderr, usage)
		return 2
	case err != nil:
		// CLIXX-E02
		fmt.Fprintf(std.Stderr, "polvo: %v\n", err)
		return 1
	}
	return code
}

// flags parses the options of a subcommand, with the --dir every one has.
func flags(name string, args []string, more func(*flag.FlagSet)) (*flag.FlagSet, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "", "the configuration folder")
	if more != nil {
		more(fs)
	}
	if err := fs.Parse(args); err != nil {
		return nil, "", errUsage
	}
	if *dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, "", err
		}
		*dir = filepath.Join(base, "polvo")
	}
	return fs, *dir, nil
}

// keygen is CLIXX-B01.
func keygen(args []string, std Stdio) error {
	fs, dir, err := flags("keygen", args, nil)
	if err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errUsage
	}
	id, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(std.Stdout, sshx.AuthorizedLine(id.PublicKey(), "polvo"))
	return err
}

// serveNode is CLIXX-B02.
func serveNode(args []string, std Stdio) error {
	var listen string
	fs, dir, err := flags("node", args, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", ":2222", "the address to listen on")
	})
	if err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errUsage
	}
	host, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "host_ed25519"))
	if err != nil {
		return err
	}
	// the installation's own key is the owner of the node's sessions
	own, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return err
	}
	perms, err := nodeconf.Load(filepath.Join(dir, "node.json"), own.PublicKey())
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	defer l.Close()
	fmt.Fprintf(std.Stdout, "polvo node: listening on %s (%s)\n", l.Addr(), ssh.FingerprintSHA256(host.PublicKey()))
	cfg := node.Config{
		HostKey:    host,
		Authorized: sshx.NewAuthorizedKeys(filepath.Join(dir, "authorized_keys")),
		Sessions:   sessions.NewManager(),
		Perms:      perms.PermFor,
		Owner:      own.PublicKey(),
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		cfg.Shell = []string{sh}
	}
	board, err := progress.NewBoard(filepath.Join(dir, "progress"))
	if err != nil {
		return err
	}
	cfg.Control = node.Controller{Sessions: cfg.Sessions, Shell: cfg.Shell, Progress: board}.Serve
	// an agent in a session reports to this node, from inside the machine
	cfg.Sessions.SetEnv([]string{"POLVO_NODE=" + loopback(l.Addr())})
	return node.Serve(l, cfg)
}

// connect is CLIXX-B03.
func connect(args []string, std Stdio) (int, error) {
	fs, dir, err := flags("connect", args, nil)
	if err != nil {
		return 0, err
	}
	if fs.NArg() != 1 {
		return 0, errUsage
	}
	id, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return 0, err
	}
	n, err := hub.Dial(fs.Arg(0), hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(dir, "known_hosts"))})
	if err != nil {
		return 0, err
	}
	defer n.Close()

	size := pty.Size{Rows: 24, Cols: 80}
	local, isTerm := std.Stdin.(*os.File)
	isTerm = isTerm && term.IsTerminal(int(local.Fd()))
	if isTerm {
		if cols, rows, err := term.GetSize(int(local.Fd())); err == nil {
			size = pty.Size{Rows: uint16(rows), Cols: uint16(cols)}
		}
		// CLIXX-B04: raw for the session, and the mode before it restored at the end.
		old, err := term.MakeRaw(int(local.Fd()))
		if err != nil {
			return 0, err
		}
		defer term.Restore(int(local.Fd()), old)
	}
	t, err := n.Shell(size)
	if err != nil {
		return 0, err
	}
	defer t.Close()
	if isTerm {
		// CLIXX-B05
		stop := watchResize(local, t)
		defer stop()
	}
	go io.Copy(t, std.Stdin)
	if _, err := io.Copy(std.Stdout, t); err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	st, err := t.Wait()
	if err != nil {
		return 0, err
	}
	if st.Signal != "" {
		return 255, nil
	}
	return st.Code, nil
}

// serveMCP is CLIXX-B06: the MCP bridge on stdin and stdout, with the tools over the nodes.
func serveMCP(args []string, std Stdio) error {
	fs, dir, err := flags("mcp", args, nil)
	if err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errUsage
	}
	id, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return err
	}
	pool := mcp.NewPool(hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(dir, "known_hosts"))})
	defer pool.Close()
	// the session of Polvo the agent runs in tells it where its node is
	pool.SetHere(mcp.Here{Node: os.Getenv("POLVO_NODE"), Session: os.Getenv("POLVO_SESSION")})
	s := &mcp.Server{Name: "polvo", Version: version, Tools: pool.Tools()}
	return s.Serve(std.Stdin, std.Stdout)
}

// openBrowser opens url in the system's browser.
var openBrowser = func(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// serveUI is CLIXX-B07: the local UI server, its address with the session secret, and the browser.
func serveUI(args []string, std Stdio) error {
	var listen string
	var noBrowser bool
	fs, dir, err := flags("ui", args, func(fs *flag.FlagSet) {
		fs.StringVar(&listen, "listen", "127.0.0.1:0", "the local address to listen on")
		fs.BoolVar(&noBrowser, "no-browser", false, "only print the address")
	})
	if err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errUsage
	}
	// only the machine itself opens the UI: it gives shells on the nodes
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("the UI only listens on a local address, not %s", listen)
	}
	id, _, err := sshx.LoadOrCreateIdentity(filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return err
	}
	cfg := web.Config{
		Hub:   hub.Config{Identity: id, KnownHosts: sshx.NewKnownHosts(filepath.Join(dir, "known_hosts"))},
		Files: web.UI(),
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		cfg.Shell = []string{sh}
	}
	s := web.New(cfg)
	defer s.Close()
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	defer l.Close()
	url := fmt.Sprintf("http://%s/#t=%s", l.Addr(), s.Token())
	fmt.Fprintf(std.Stdout, "polvo: the UI is at %s\n", url)
	if !noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(std.Stderr, "polvo: could not open the browser (%v); open the address above\n", err)
		}
	}
	return http.Serve(l, s.Handler())
}

// loopback is where the node at addr answers from inside the machine: an address that listens on every
// interface is reached at the loopback.
func loopback(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}
