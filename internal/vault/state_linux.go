//go:build linux

package vault

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xgbackup/xgbackup/internal/config"
)

type ClientState struct {
	LastSnapshot     string `json:"last_snapshot"`
	LastSnapshotTime string `json:"last_snapshot_time"`
	LastSnapshotOK   bool   `json:"last_snapshot_ok"`
	LastError        string `json:"last_error"`
}

type State struct {
	Clients map[string]ClientState `json:"clients"`
}

func loadState() (State, error) {
	data, err := os.ReadFile(config.StatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Clients: map[string]ClientState{}}, nil
		}
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, err
	}
	if st.Clients == nil {
		st.Clients = map[string]ClientState{}
	}
	return st, nil
}

func saveState(st State) error {
	if st.Clients == nil {
		st.Clients = map[string]ClientState{}
	}
	if err := os.MkdirAll(filepath.Dir(config.StatePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := config.StatePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, config.StatePath)
}
