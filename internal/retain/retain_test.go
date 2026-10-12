package retain

import "testing"

func TestPruneNames(t *testing.T) {
	names := []string{"daily-20260103", "daily-20260101", "daily-20260102"}
	got := PruneNames(names, 2)
	if len(got) != 1 || got[0] != "daily-20260101" {
		t.Fatalf("got %v", got)
	}
	if PruneNames(names, 7) != nil {
		t.Fatal("expected nothing to prune")
	}
}

func TestFilterPrefix(t *testing.T) {
	names := []string{
		"daily-20260101",
		"weekly-20260101",
		"daily-note",
		"hourly-20260101T15",
		"daily-20260101T000000Z",
	}
	got := FilterPrefix(names, "daily")
	if len(got) != 1 || got[0] != "daily-20260101" {
		t.Fatalf("daily %v", got)
	}
	hourly := FilterPrefix(names, "hourly")
	if len(hourly) != 1 || hourly[0] != "hourly-20260101T15" {
		t.Fatalf("hourly %v", hourly)
	}
}
