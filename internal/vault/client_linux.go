//go:build linux

package vault

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"

	"github.com/xgbackup/xgbackup/internal/config"
)

func ListClients(pool string) ([]string, error) {
	dir := filepath.Join(pool, "clients")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if err := validateClient(name); err != nil {
			continue
		}
		if st, err := os.Stat(dropDir(pool, name)); err == nil && st.IsDir() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func AddClient(pool, name string) (string, error) {
	if err := requireRoot(); err != nil {
		return "", err
	}
	if err := validateClient(name); err != nil {
		return "", err
	}
	if _, err := os.Stat(pool); err != nil {
		return "", fmt.Errorf("pool %s is not available: %w", pool, err)
	}
	if _, err := user.Lookup(name); err != nil {
		if err := run("useradd", "--create-home", "--shell", "/usr/sbin/nologin", "--comment", "xgbackup client", name); err != nil {
			return "", err
		}
	}
	if err := mkdirRoot(clientDir(pool, name), 0o755); err != nil {
		return "", err
	}
	if err := mkdirRoot(jailDir(pool, name), 0o755); err != nil {
		return "", err
	}
	if err := mkdirRoot(snapsDir(pool, name), 0o700); err != nil {
		return "", err
	}
	drop := dropDir(pool, name)
	if _, err := os.Stat(drop); os.IsNotExist(err) {
		if err := run("btrfs", "subvolume", "create", drop); err != nil {
			return "", err
		}
	}
	u, err := user.Lookup(name)
	if err != nil {
		return "", err
	}
	if err := run("chown", u.Uid+":"+u.Gid, drop); err != nil {
		return "", err
	}
	if err := os.Chmod(drop, 0o755); err != nil {
		return "", err
	}
	keyPath := filepath.Join(config.KeysDir, name, "id_ed25519")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
			return "", err
		}
		if err := run("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-C", "xgbackup-"+name); err != nil {
			return "", err
		}
	}
	if err := os.Chmod(filepath.Dir(keyPath), 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return "", err
	}
	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", err
	}
	home := u.HomeDir
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", err
	}
	authPath := filepath.Join(sshDir, "authorized_keys")
	if err := os.WriteFile(authPath, pub, 0o600); err != nil {
		return "", err
	}
	if err := run("chown", "-R", u.Uid+":"+u.Gid, sshDir); err != nil {
		return "", err
	}
	if err := os.Chmod(home, 0o755); err != nil {
		return "", err
	}
	return keyPath, nil
}
