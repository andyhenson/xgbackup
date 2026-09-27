//go:build linux

package vault

import (
	"fmt"
	"os"
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
	st, err := loadState()
	if err != nil {
		return err
	}
	failed := 0
	for _, name := range clients {
		snap, err := snapshotClient(cfg.Pool, name, schedule)
		if err != nil {
			failed++
			prev := st.Clients[name]
			prev.LastSnapshotOK = false
			prev.LastError = err.Error()
			st.Clients[name] = prev
			fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
			continue
		}
		if err := pruneClient(cfg.Pool, name, schedule, sch.Keep); err != nil {
			failed++
			st.Clients[name] = ClientState{
				LastSnapshot:     snap,
				LastSnapshotTime: time.Now().UTC().Format(time.RFC3339),
				LastSnapshotOK:   false,
				LastError:        err.Error(),
			}
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

func snapshotClient(pool, name, schedule string) (string, error) {
	var last error
	for try := 0; try < 5; try++ {
		stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
		destName := schedule + "-" + stamp
		dest := snapsDir(pool, name) + "/" + destName
		if _, err := os.Stat(dest); err == nil {
			time.Sleep(time.Millisecond)
			last = fmt.Errorf("snapshot %s already exists", destName)
			continue
		}
		if err := run("btrfs", "subvolume", "snapshot", "-r", dropDir(pool, name), dest); err != nil {
			return "", err
		}
		return destName, nil
	}
	if last == nil {
		last = fmt.Errorf("could not allocate a snapshot name")
	}
	return "", last
}

func pruneClient(pool, name, schedule string, keep int) error {
	names, err := snapshotNames(snapsDir(pool, name))
	if err != nil {
		return err
	}
	for _, old := range retain.PruneNames(retain.FilterPrefix(names, schedule), keep) {
		if err := deleteSnapshot(snapsDir(pool, name) + "/" + old); err != nil {
			return err
		}
		fmt.Printf("%s pruned %s\n", name, old)
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

func newestSnapshot(dir string) (string, error) {
	names, err := snapshotNames(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var snaps []string
	for _, name := range names {
		if strings.Contains(name, "-") {
			snaps = append(snaps, name)
		}
	}
	if len(snaps) == 0 {
		return "", nil
	}
	return snaps[len(snaps)-1], nil
}
