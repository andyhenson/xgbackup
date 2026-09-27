package retain

import "testing"

func TestPruneNames(t *testing.T) {
	names := []string{"daily-20260103T000000Z", "daily-20260101T000000Z", "daily-20260102T000000Z"}
	got := PruneNames(names, 2)
	if len(got) != 1 || got[0] != "daily-20260101T000000Z" {
		t.Fatalf("got %v", got)
	}
	if PruneNames(names, 7) != nil {
		t.Fatal("expected nothing to prune")
	}
}

func TestFilterPrefix(t *testing.T) {
	names := []string{"daily-20260101T000000Z", "weekly-20260101T000000Z", "daily-note"}
	got := FilterPrefix(names, "daily")
	if len(got) != 1 || got[0] != "daily-20260101T000000Z" {
		t.Fatalf("got %v", got)
	}
}
