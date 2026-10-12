//go:build linux

package vault

import (
	"testing"
	"time"
)

func TestSnapshotName(t *testing.T) {
	when := time.Date(2026, 10, 12, 23, 40, 1, 0, time.UTC)
	daily, err := snapshotName("daily", when)
	if err != nil {
		t.Fatal(err)
	}
	if daily != "daily-20261012" {
		t.Fatalf("daily %s", daily)
	}
	hourly, err := snapshotName("hourly", when)
	if err != nil {
		t.Fatal(err)
	}
	if hourly != "hourly-20261012T23" {
		t.Fatalf("hourly %s", hourly)
	}
	for _, schedule := range []string{"weekly", "monthly", "yearly"} {
		got, err := snapshotName(schedule, when)
		if err != nil {
			t.Fatal(err)
		}
		if got != schedule+"-20261012" {
			t.Fatalf("%s -> %s", schedule, got)
		}
	}
	if _, err := snapshotName("nope", when); err == nil {
		t.Fatal("expected unknown schedule")
	}
}
