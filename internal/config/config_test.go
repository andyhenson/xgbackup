package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := Default("vault-1")
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.VaultID != "vault-1" || !got.Schedules["daily"].Enabled || got.Schedules["hourly"].Enabled {
		t.Fatalf("%+v", got)
	}
	if _, err := got.Schedule("nope"); err == nil {
		t.Fatal("expected unknown schedule error")
	}
}
