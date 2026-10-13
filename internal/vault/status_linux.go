//go:build linux

package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xgbackup/xgbackup/internal/config"
)

type ClientStatus struct {
	Name             string `json:"name"`
	LastSnapshot     string `json:"last_snapshot"`
	LastSnapshotTime string `json:"last_snapshot_time,omitempty"`
	LastSnapshotOK   bool   `json:"last_snapshot_ok"`
	LastError        string `json:"last_error"`
}

type Status struct {
	VaultID        string         `json:"vault_id"`
	Time           string         `json:"time"`
	PoolFreeBytes  uint64         `json:"pool_free_bytes"`
	PoolTotalBytes uint64         `json:"pool_total_bytes"`
	Clients        []ClientStatus `json:"clients"`
}

func WriteStatus(cfg config.Config) error {
	st, err := BuildStatus(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(config.StatusPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := config.StatusPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, config.StatusPath)
}

func BuildStatus(cfg config.Config) (Status, error) {
	free, total, err := poolSpace(cfg.Pool)
	if err != nil {
		return Status{}, err
	}
	state, err := loadState()
	if err != nil {
		return Status{}, err
	}
	names, err := ListClients(cfg.Pool)
	if err != nil {
		return Status{}, err
	}
	out := Status{
		VaultID:        cfg.VaultID,
		Time:           time.Now().UTC().Format(time.RFC3339),
		PoolFreeBytes:  free,
		PoolTotalBytes: total,
		Clients:        []ClientStatus{},
	}
	for _, name := range names {
		cs := state.Clients[name]
		newest, err := newestSnapshotForClient(cfg.Pool, name)
		if err != nil {
			return Status{}, err
		}
		item := ClientStatus{
			Name:             name,
			LastSnapshot:     cs.LastSnapshot,
			LastSnapshotTime: cs.LastSnapshotTime,
			LastSnapshotOK:   cs.LastSnapshotOK,
			LastError:        cs.LastError,
		}
		if item.LastSnapshot == "" {
			item.LastSnapshot = newest
			if newest == "" && item.LastError == "" {
				item.LastSnapshotOK = false
				item.LastError = "no snapshot yet"
			}
		}
		out.Clients = append(out.Clients, item)
	}
	return out, nil
}

func PrintStatus(cfg config.Config) error {
	if os.Geteuid() != 0 {
		text, err := queryStatusText()
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	}
	text, err := statusText(cfg)
	if err != nil {
		return err
	}
	fmt.Print(text)
	return nil
}

func statusText(cfg config.Config) (string, error) {
	st, err := BuildStatus(cfg)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "vault %s\n", st.VaultID)
	fmt.Fprintf(&b, "pool %s\n", cfg.Pool)
	fmt.Fprintf(&b, "free %d bytes of %d\n", st.PoolFreeBytes, st.PoolTotalBytes)
	if len(st.Clients) == 0 {
		fmt.Fprintf(&b, "clients none\n")
	}
	for _, c := range st.Clients {
		ok := "ok"
		if !c.LastSnapshotOK {
			ok = "failed"
		}
		fmt.Fprintf(&b, "client %s last_snapshot=%s %s", c.Name, emptyDash(c.LastSnapshot), ok)
		if c.LastError != "" {
			fmt.Fprintf(&b, " error=%s", c.LastError)
		}
		fmt.Fprintf(&b, "\n")
	}
	if err := WriteStatus(cfg); err != nil {
		return "", err
	}
	return b.String(), nil
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func poolSpace(pool string) (uint64, uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(pool, &st); err != nil {
		return 0, 0, err
	}
	bsize := uint64(st.Bsize)
	return st.Bavail * bsize, st.Blocks * bsize, nil
}
