//go:build linux

package vault

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Restore(pool, client, snapshot, rel, dest string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	src, err := snapshotPath(pool, client, snapshot, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := copyTree(src, dest); err != nil {
			return err
		}
		fmt.Printf("restored %s to %s\n", src, dest)
		return nil
	}
	if err := copyFile(src, dest); err != nil {
		return err
	}
	fmt.Printf("restored %s to %s\n", src, dest)
	return nil
}

// RestoreAll copies every client in one snapshot into dest/<client>/.
func RestoreAll(pool, snapshot, dest string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := validateSnapshot(snapshot); err != nil {
		return err
	}
	root := snapshotTree(pool, snapshot)
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		client := e.Name()
		if err := validateClient(client); err != nil {
			continue
		}
		target := filepath.Join(dest, client)
		if err := copyTree(filepath.Join(root, client), target); err != nil {
			return err
		}
		fmt.Printf("restored %s to %s\n", client, target)
		n++
	}
	if n == 0 {
		return fmt.Errorf("snapshot %s has no clients", snapshot)
	}
	return nil
}

func snapshotPath(pool, client, snapshot, rel string) (string, error) {
	if err := validateClient(client); err != nil {
		return "", err
	}
	if err := validateSnapshot(snapshot); err != nil {
		return "", err
	}
	clean, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	return path.Join(clientSnapDir(pool, snapshot, client), clean), nil
}

func validateSnapshot(snapshot string) error {
	if snapshot == "" || strings.Contains(snapshot, "/") || snapshot == "." || snapshot == ".." {
		return fmt.Errorf("invalid snapshot name")
	}
	return nil
}

func cleanRel(rel string) (string, error) {
	if rel == "" || rel == "." {
		return "", nil
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return "", fmt.Errorf("invalid path")
		}
	}
	c := path.Clean("/" + rel)
	c = strings.TrimPrefix(c, "/")
	if c == "" || c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return "", fmt.Errorf("invalid path")
	}
	return c, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyTree(src, dest string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFile(p, target)
	})
}
