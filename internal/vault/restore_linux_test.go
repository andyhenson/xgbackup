//go:build linux

package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTree(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "sample.txt"), []byte("vault-drill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "out")
	if err := copyTree(src, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "nested", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "vault-drill\n" {
		t.Fatalf("%q", got)
	}
	clean, err := cleanRel(".")
	if err != nil || clean != "" {
		t.Fatalf("dot %q %v", clean, err)
	}
	if _, err := cleanRel("../x"); err == nil {
		t.Fatal("expected invalid path")
	}
}
