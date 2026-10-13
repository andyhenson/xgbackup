package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	ConfigPath  = "/etc/xgbackup/config.json"
	StatePath   = "/var/lib/xgbackup/state.json"
	StatusPath  = "/var/lib/xgbackup/status.json"
	DefaultPool = "/var/lib/xgbackup/pool"
	KeysDir     = "/etc/xgbackup/clients"
)

var ScheduleNames = []string{"hourly", "daily", "weekly", "monthly", "yearly"}

type Schedule struct {
	Enabled bool `json:"enabled"`
	Keep    int  `json:"keep"`
}

type RestoreRule struct {
	User   string `json:"user"`
	Client string `json:"client"`
	Prefix string `json:"prefix"`
}

type WebConfig struct {
	Listen    string   `json:"listen,omitempty"`
	Required  []string `json:"required,omitempty"`
	AllowCIDR []string `json:"allow_cidr,omitempty"`
	Cert      string   `json:"cert,omitempty"`
	Key       string   `json:"key,omitempty"`
}

type Config struct {
	Pool       string              `json:"pool"`
	VaultID    string              `json:"vault_id"`
	Schedules  map[string]Schedule `json:"schedules"`
	SetupUsers []string            `json:"setup_users,omitempty"`
	Restore    []RestoreRule       `json:"restore,omitempty"`
	Web        WebConfig           `json:"web,omitempty"`
}

func (c Config) EffectiveWeb() WebConfig {
	w := c.Web
	if w.Listen == "" {
		w.Listen = "127.0.0.1:8787"
	}
	if len(w.Required) == 0 {
		w.Required = []string{"password"}
	}
	return w
}

func Default(vaultID string) Config {
	return Config{
		Pool:    DefaultPool,
		VaultID: vaultID,
		Schedules: map[string]Schedule{
			"hourly":  {Enabled: false, Keep: 24},
			"daily":   {Enabled: true, Keep: 7},
			"weekly":  {Enabled: true, Keep: 4},
			"monthly": {Enabled: true, Keep: 12},
			"yearly":  {Enabled: true, Keep: 3},
		},
	}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Pool == "" {
		cfg.Pool = DefaultPool
	}
	if cfg.Schedules == nil {
		cfg.Schedules = Default("").Schedules
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (c Config) Schedule(name string) (Schedule, error) {
	known := false
	for _, n := range ScheduleNames {
		if n == name {
			known = true
			break
		}
	}
	if !known {
		return Schedule{}, errUnknownSchedule(name)
	}
	s, ok := c.Schedules[name]
	if !ok {
		return Schedule{}, errUnknownSchedule(name)
	}
	return s, nil
}

type unknownSchedule string

func errUnknownSchedule(name string) error { return unknownSchedule(name) }

func (e unknownSchedule) Error() string { return "unknown schedule " + string(e) }
