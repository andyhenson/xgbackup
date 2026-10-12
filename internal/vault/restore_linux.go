//go:build linux

package vault

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

func Restore(pool, client, snapshot, rel, dest string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := validateClient(client); err != nil {
		return err
	}
	if snapshot == "" || strings.Contains(snapshot, "/") || snapshot == "." || snapshot == ".." {
		return fmt.Errorf("invalid snapshot name")
	}
	clean, err := cleanRel(rel)
	if err != nil {
		return err
	}
	src := path.Join(clientSnapDir(pool, snapshot, client), clean)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory; restore a file")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(path.Dir(dest), 0o755); err != nil {
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
	if err := out.Close(); err != nil {
		return err
	}
	fmt.Printf("restored %s to %s\n", src, dest)
	return nil
}

func cleanRel(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("path is required")
	}
	c := path.Clean("/" + rel)
	c = strings.TrimPrefix(c, "/")
	if c == "" || c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return "", fmt.Errorf("invalid path")
	}
	return c, nil
}
