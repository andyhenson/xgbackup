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
	if err := migrateKeys(); err != nil {
		return nil, err
	}
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

func keyFile(name string) string {
	return filepath.Join(config.KeysDir, name+".key")
}

// migrateKeys renames <name>/id_ed25519 to <name>.key and leaves a symlink
// at the old path. It is a no-op for a non-root caller.
func migrateKeys() error {
	if os.Geteuid() != 0 {
		return nil
	}
	entries, err := os.ReadDir(config.KeysDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if err := validateClient(name); err != nil {
			continue
		}
		if err := migrateKey(name); err != nil {
			return err
		}
	}
	return nil
}

func migrateKey(name string) error {
	newPath := keyFile(name)
	oldDir := filepath.Join(config.KeysDir, name)
	oldPath := filepath.Join(oldDir, "id_ed25519")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
			if _, err := os.Stat(oldPath + ".pub"); err == nil {
				if err := os.Rename(oldPath+".pub", newPath+".pub"); err != nil {
					return err
				}
			}
		}
	} else if err != nil {
		return err
	}
	if _, err := os.Stat(newPath); err != nil {
		return nil
	}
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		return err
	}
	if _, err := os.Lstat(oldPath); os.IsNotExist(err) {
		if err := os.Symlink(newPath, oldPath); err != nil {
			return err
		}
	}
	if _, err := os.Stat(newPath + ".pub"); err == nil {
		if _, err := os.Lstat(oldPath + ".pub"); os.IsNotExist(err) {
			if err := os.Symlink(newPath+".pub", oldPath+".pub"); err != nil {
				return err
			}
		}
	}
	return nil
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
	if err := os.MkdirAll(config.KeysDir, 0o755); err != nil {
		return "", err
	}
	if err := migrateKey(name); err != nil {
		return "", err
	}
	keyPath := keyFile(name)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		if err := run("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-C", "xgbackup-"+name); err != nil {
			return "", err
		}
	}
	if err := migrateKey(name); err != nil {
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
