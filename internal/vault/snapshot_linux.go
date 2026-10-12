//go:build linux

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xgbackup/xgbackup/internal/config"
	"github.com/xgbackup/xgbackup/internal/retain"
)

func Snapshot(schedule string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	sch, err := cfg.Schedule(schedule)
	if err != nil {
		return err
	}
	if !sch.Enabled {
		fmt.Printf("%s snapshots are disabled\n", schedule)
		return nil
	}
	if sch.Keep < 1 {
		return fmt.Errorf("keep for %s must be at least 1", schedule)
	}
	clients, err := ListClients(cfg.Pool)
	if err != nil {
		return err
	}
	if err := mkdirRoot(snapsRoot(cfg.Pool), 0o700); err != nil {
		return err
	}
	if len(clients) == 0 {
		fmt.Println("no clients")
		return nil
	}
	snap, err := snapshotName(schedule, time.Now().UTC())
	if err != nil {
		return err
	}
	destRoot := snapshotTree(cfg.Pool, snap)
	if _, err := os.Stat(destRoot); err == nil {
		if err := deleteSnapshotTree(destRoot); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := mkdirRoot(destRoot, 0o700); err != nil {
		return err
	}
	st, err := loadState()
	if err != nil {
		return err
	}
	failed := 0
	for _, name := range clients {
		dest := clientSnapDir(cfg.Pool, snap, name)
		if err := run("btrfs", "subvolume", "snapshot", "-r", dropDir(cfg.Pool, name), dest); err != nil {
			failed++
			prev := st.Clients[name]
			prev.LastSnapshotOK = false
			prev.LastError = err.Error()
			st.Clients[name] = prev
			fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
			continue
		}
		st.Clients[name] = ClientState{
			LastSnapshot:     snap,
			LastSnapshotTime: time.Now().UTC().Format(time.RFC3339),
			LastSnapshotOK:   true,
			LastError:        "",
		}
		fmt.Printf("%s snapshot %s\n", name, snap)
	}
	if failed == len(clients) {
		_ = os.Remove(destRoot)
	}
	if err := pruneSchedule(cfg.Pool, schedule, sch.Keep); err != nil {
		return err
	}
	if err := saveState(st); err != nil {
		return err
	}
	if err := WriteStatus(cfg); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d client snapshot(s) failed", failed)
	}
	return nil
}

func snapshotName(schedule string, now time.Time) (string, error) {
	switch schedule {
	case "hourly":
		return schedule + "-" + now.UTC().Format("20060102T15"), nil
	case "daily", "weekly", "monthly", "yearly":
		return schedule + "-" + now.UTC().Format("20060102"), nil
	default:
		return "", fmt.Errorf("unknown schedule %s", schedule)
	}
}

func pruneSchedule(pool, schedule string, keep int) error {
	names, err := snapshotNames(snapsRoot(pool))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, old := range retain.PruneNames(retain.FilterPrefix(names, schedule), keep) {
		if err := deleteSnapshotTree(snapshotTree(pool, old)); err != nil {
			return err
		}
		fmt.Printf("pruned %s\n", old)
	}
	return nil
}

func snapshotNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func deleteSnapshot(path string) error {
	err := run("btrfs", "subvolume", "delete", path)
	if err == nil {
		return nil
	}
	if setErr := run("btrfs", "property", "set", path, "ro", "false"); setErr != nil {
		return err
	}
	return run("btrfs", "subvolume", "delete", path)
}

func deleteSnapshotTree(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := deleteSnapshot(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return os.Remove(dir)
}

func newestSnapshotForClient(pool, client string) (string, error) {
	names, err := snapshotNames(snapsRoot(pool))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var snaps []string
	for _, name := range names {
		if !strings.Contains(name, "-") {
			continue
		}
		st, err := os.Stat(clientSnapDir(pool, name, client))
		if err == nil && st.IsDir() {
			snaps = append(snaps, name)
		}
	}
	if len(snaps) == 0 {
		return "", nil
	}
	return snaps[len(snaps)-1], nil
}
