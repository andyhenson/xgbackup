package syncer

import "testing"

func TestRemoteJoin(t *testing.T) {
	got, err := remoteJoin("/drop", "a/b.txt")
	if err != nil || got != "/drop/a/b.txt" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := remoteJoin("/drop", "../snaps/x"); err == nil {
		t.Fatal("expected escape to fail")
	}
	if _, err := remoteJoin("drop", "a"); err == nil {
		t.Fatal("expected relative dest to fail")
	}
}
