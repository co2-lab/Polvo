// @anchors
//   code: KYIMP
//   ref: KYSEX

// Package sshx is the SSH transport of Polvo: the identities, who enters a node, the nodes a
// desktop knows, and the server and client. The files follow the OpenSSH formats.
package sshx

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// PermissionError refuses a private key that others can read or write (KYSEX-E01).
type PermissionError struct {
	Path string
	Mode fs.FileMode
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("sshx: %s has permissions %04o; the private key must be readable only by its owner (0600)", e.Path, e.Mode)
}

// LoadOrCreateIdentity returns the installation's key pair at path, creating an ed25519 pair
// there the first time (KYSEX-B01). created tells whether it was just born.
func LoadOrCreateIdentity(path string) (signer ssh.Signer, created bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		s, err := createIdentity(path)
		return s, err == nil, err
	}
	if err != nil {
		return nil, false, fmt.Errorf("sshx: reading the identity %s: %w", path, err)
	}
	// KYSEX-E01: on Unix a key others can read or write is refused, as OpenSSH does.
	if runtime.GOOS != "windows" {
		st, err := os.Stat(path)
		if err != nil {
			return nil, false, fmt.Errorf("sshx: reading the identity %s: %w", path, err)
		}
		if m := st.Mode().Perm(); m&0o077 != 0 {
			return nil, false, &PermissionError{Path: path, Mode: m}
		}
	}
	// KYSEX-E02: a file that is not a key is an error naming the file.
	s, err := ssh.ParsePrivateKey(b)
	if err != nil {
		return nil, false, fmt.Errorf("sshx: the identity %s is not a private key: %w", path, err)
	}
	return s, false, nil
}

// RawKey is the ed25519 private key of the identity at path, as LoadOrCreateIdentity made it, for what
// needs the key itself and not an SSH signer: the transport between networks (KYSEX-B08).
func RawKey(path string) (ed25519.PrivateKey, error) {
	if _, _, err := LoadOrCreateIdentity(path); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sshx: reading the identity %s: %w", path, err)
	}
	raw, err := ssh.ParseRawPrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("sshx: the identity %s is not a private key: %w", path, err)
	}
	k, ok := raw.(*ed25519.PrivateKey)
	if !ok {
		// KYSEX-E03
		return nil, fmt.Errorf("sshx: the identity %s is not an ed25519 key", path)
	}
	return *k, nil
}

// createIdentity writes a new ed25519 pair readable only by its owner (KYSEX-B02).
func createIdentity(path string) (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("sshx: generating the identity: %w", err)
	}
	blk, err := ssh.MarshalPrivateKey(priv, "polvo")
	if err != nil {
		return nil, fmt.Errorf("sshx: encoding the identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("sshx: creating the folder of %s: %w", path, err)
	}
	// O_EXCL: two processes born at once never overwrite each other's key.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("sshx: creating the identity %s: %w", path, err)
	}
	if _, err := f.Write(pem.EncodeToMemory(blk)); err != nil {
		f.Close()
		return nil, fmt.Errorf("sshx: writing the identity %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("sshx: writing the identity %s: %w", path, err)
	}
	return ssh.NewSignerFromKey(priv)
}

// AuthorizedLine is the authorized_keys line for key, with an optional comment.
func AuthorizedLine(key ssh.PublicKey, comment string) string {
	// KYSEX-I01: the line is the OpenSSH form Allows parses back to the same key bytes.
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
	if comment != "" {
		line += " " + comment
	}
	return line + "\n"
}

// AuthorizedKeys is a node's list of the keys that may enter, in the authorized_keys format.
type AuthorizedKeys struct {
	path string
	// Warn receives each line that is not a key (KYSEX-B04); nil drops them.
	Warn func(line int, err error)
}

// NewAuthorizedKeys reads the list at path.
func NewAuthorizedKeys(path string) *AuthorizedKeys { return &AuthorizedKeys{path: path} }

// Allows tells whether key is in the list. The file is read again on each call, so an edit
// counts for the next connection (KYSEX-B03).
func (a *AuthorizedKeys) Allows(key ssh.PublicKey) (bool, error) {
	b, err := os.ReadFile(a.path)
	// KYSEX-E03: without the list nobody enters, and it is not an error.
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("sshx: reading the authorized keys %s: %w", a.path, err)
	}
	want := key.Marshal()
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		k, _, _, _, err := ssh.ParseAuthorizedKey(line)
		if err != nil {
			// KYSEX-B04: the bad line is skipped and reported; the others still count.
			if a.Warn != nil {
				a.Warn(n, err)
			}
			continue
		}
		if bytes.Equal(k.Marshal(), want) {
			return true, nil
		}
	}
	if err := sc.Err(); err != nil {
		return false, fmt.Errorf("sshx: reading the authorized keys %s: %w", a.path, err)
	}
	return false, nil
}

// HostKeyChangedError refuses a known node whose key is not the recorded one (KYSEX-B07).
type HostKeyChangedError struct {
	Host string
}

func (e *HostKeyChangedError) Error() string {
	return fmt.Sprintf("sshx: the key of %s changed; if the change is expected, remove its line from the known hosts", e.Host)
}

// KnownHosts is the desktop's list of the nodes it has met, in the known_hosts format.
type KnownHosts struct {
	path string
	mu   sync.Mutex
}

// NewKnownHosts keeps the list at path.
func NewKnownHosts(path string) *KnownHosts { return &KnownHosts{path: path} }

// Check is an ssh.HostKeyCallback: it accepts and records a node met for the first time
// (KYSEX-B05), accepts the recorded key (KYSEX-B06) and refuses a changed one (KYSEX-B07).
func (k *KnownHosts) Check(hostport string, remote net.Addr, key ssh.PublicKey) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(k.path), 0o700); err != nil {
		return fmt.Errorf("sshx: creating the folder of %s: %w", k.path, err)
	}
	f, err := os.OpenFile(k.path, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("sshx: opening the known hosts %s: %w", k.path, err)
	}
	f.Close()
	cb, err := knownhosts.New(k.path)
	if err != nil {
		return fmt.Errorf("sshx: reading the known hosts %s: %w", k.path, err)
	}
	err = cb(hostport, remote, key)
	var ke *knownhosts.KeyError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ke) && len(ke.Want) > 0:
		return &HostKeyChangedError{Host: hostport}
	case errors.As(err, &ke):
		// KYSEX-B05: first contact, so the key is recorded and the connection accepted.
		return k.record(hostport, key)
	}
	return fmt.Errorf("sshx: checking the key of %s: %w", hostport, err)
}

func (k *KnownHosts) record(hostport string, key ssh.PublicKey) error {
	f, err := os.OpenFile(k.path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("sshx: recording %s in %s: %w", hostport, k.path, err)
	}
	// KYSEX-I02: the line is recorded under the same normalized host knownhosts reads back.
	line := knownhosts.Line([]string{knownhosts.Normalize(hostport)}, key) + "\n"
	if _, err := f.WriteString(line); err != nil {
		f.Close()
		return fmt.Errorf("sshx: recording %s in %s: %w", hostport, k.path, err)
	}
	return f.Close()
}
