package syncer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Profile is the remembered vault connection in ~/.xgbackup/profile.
type Profile struct {
	Host string
	Port int
	User string
	Key  string
}

func ProfileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".xgbackup"), nil
}

func KnownHostsPath(dir string) string {
	return filepath.Join(dir, "known_hosts")
}

func LoadProfile(dir string) (Profile, error) {
	p := Profile{Port: 22}
	f, err := os.Open(filepath.Join(dir, "profile"))
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return Profile{}, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return Profile{}, fmt.Errorf("profile line %q needs key=value", line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "host":
			p.Host = val
		case "port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return Profile{}, fmt.Errorf("profile port: %w", err)
			}
			p.Port = n
		case "user":
			p.User = val
		case "key":
			p.Key = val
		default:
			return Profile{}, fmt.Errorf("unknown profile field %s", key)
		}
	}
	if err := sc.Err(); err != nil {
		return Profile{}, err
	}
	if p.Port == 0 {
		p.Port = 22
	}
	return p, nil
}

func SaveProfile(dir string, p Profile) error {
	if p.Port == 0 {
		p.Port = 22
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data := fmt.Sprintf("host=%s\nport=%d\nuser=%s\nkey=%s\n", p.Host, p.Port, p.User, p.Key)
	return os.WriteFile(filepath.Join(dir, "profile"), []byte(data), 0o600)
}

// CopyKnownHosts stores a known_hosts file in the profile directory.
func CopyKnownHosts(dir, src string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	dest := KnownHostsPath(dir)
	if filepath.Clean(src) == filepath.Clean(dest) {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}
