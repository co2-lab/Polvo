// @anchors
//   code: NCIND
//   ref: NDCNN

// Package nodeconf reads what the people who connect to a node may do in its sessions.
package nodeconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/co2-lab/polvo/internal/sessions"
)

// DEFAULT_GUEST is the permission of an authorized key when nothing says otherwise (NDCNN-B01): to
// authorize a key is to trust it; watching only is what the configuration restricts.
const DEFAULT_GUEST = sessions.Control

// Config is the node's configuration: the permission of the guests, of each named key, and the owner.
type Config struct {
	guest sessions.Perm
	keys  map[string]sessions.Perm
	owner string
}

type file struct {
	Guest string            `json:"guest"`
	Keys  map[string]string `json:"keys"`
}

// Load reads the configuration at path; owner is the key of this installation.
func Load(path string, owner ssh.PublicKey) (*Config, error) {
	c := &Config{guest: DEFAULT_GUEST, keys: map[string]sessions.Perm{}, owner: ssh.FingerprintSHA256(owner)}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// NDCNN-B01
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("nodeconf: %w", err)
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		// NDCNN-E01
		return nil, fmt.Errorf("nodeconf: %s is not a node configuration: %w", path, err)
	}
	if f.Guest != "" {
		if c.guest, err = perm(f.Guest); err != nil {
			return nil, fmt.Errorf("nodeconf: %s: guest: %w", path, err)
		}
	}
	for fp, p := range f.Keys {
		if c.keys[fp], err = perm(p); err != nil {
			return nil, fmt.Errorf("nodeconf: %s: key %s: %w", path, fp, err)
		}
	}
	return c, nil
}

func perm(s string) (sessions.Perm, error) {
	switch s {
	case "view":
		return sessions.View, nil
	case "control":
		return sessions.Control, nil
	}
	// NDCNN-E02
	return 0, fmt.Errorf("unknown permission %q (view or control)", s)
}

// PermFor is the permission of key (NDCNN-B02); the installation's own key is the owner (NDCNN-B03).
func (c *Config) PermFor(key ssh.PublicKey) sessions.Perm {
	fp := ssh.FingerprintSHA256(key)
	if fp == c.owner {
		return sessions.Owner
	}
	if p, ok := c.keys[fp]; ok {
		return p
	}
	return c.guest
}
