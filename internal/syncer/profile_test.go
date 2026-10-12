package syncer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := Profile{Host: "10.4.1.33", Port: 22, User: "xgb-demo", Key: filepath.Join(dir, "xgb-demo.key")}
	if err := SaveProfile(dir, in); err != nil {
		t.Fatal(err)
	}
	got, err := LoadProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("got %+v", got)
	}
	missing, err := LoadProfile(filepath.Join(dir, "absent"))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Port != 22 || missing.Host != "" {
		t.Fatalf("missing %+v", missing)
	}
	kh := filepath.Join(dir, "src_known_hosts")
	if err := os.WriteFile(kh, []byte("vault ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyKnownHosts(dir, kh); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(KnownHostsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Fatal("empty known_hosts")
	}
}
