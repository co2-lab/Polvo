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
	"os"
	"path/filepath"

	"github.com/co2-lab/polvo/internal/hub"
	"github.com/co2-lab/polvo/internal/node"
	"github.com/co2-lab/polvo/internal/pty"
	"github.com/co2-lab/polvo/internal/sshx"
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
`

// errUsage is CLIXX-E01: the usage on stderr and the exit code 2.
var errUsage = errors.New("usage")

// Run runs the polvo command line and returns the exit code.
func Run(args []string, std Stdio) int {
	err := errUsage
	code := 0
	if len(args) > 0 {
		switch args[0] {
		case "keygen":
			err = keygen(args[1:], std)
		case "node":
			err = serveNode(args[1:], std)
		case "connect":
			code, err = connect(args[1:], std)
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
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	defer l.Close()
	fmt.Fprintf(std.Stdout, "polvo node: listening on %s (%s)\n", l.Addr(), ssh.FingerprintSHA256(host.PublicKey()))
	cfg := node.Config{
		HostKey:    host,
		Authorized: sshx.NewAuthorizedKeys(filepath.Join(dir, "authorized_keys")),
		Control:    node.ServeControl,
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		cfg.Shell = []string{sh}
	}
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
