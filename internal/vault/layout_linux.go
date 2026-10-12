//go:build linux

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var clientName = regexp.MustCompile(`^xgb-[a-z0-9]([a-z0-9-]{0,22}[a-z0-9])?$`)

func validateClient(name string) error {
	if !clientName.MatchString(name) {
		return fmt.Errorf("client name must look like xgb-demo (lowercase, start with xgb-)")
	}
	return nil
}

func clientDir(pool, name string) string {
	return filepath.Join(pool, "clients", name)
}

func jailDir(pool, name string) string {
	return filepath.Join(clientDir(pool, name), "jail")
}

func dropDir(pool, name string) string {
	return filepath.Join(jailDir(pool, name), "drop")
}

func snapsRoot(pool string) string {
	return filepath.Join(pool, "snaps")
}

func snapshotTree(pool, snap string) string {
	return filepath.Join(snapsRoot(pool), snap)
}

func clientSnapDir(pool, snap, client string) string {
	return filepath.Join(snapshotTree(pool, snap), client)
}

func mkdirRoot(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	if err := os.Chown(path, 0, 0); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}
