//go:build linux

package vault

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xgbackup/xgbackup/internal/config"
)

// Menu is the whiptail console for a logged-in user. It does not apply the
// web UI's IP, cookie, password, or second-factor checks.
func Menu() error {
	if _, err := exec.LookPath("whiptail"); err != nil {
		return fmt.Errorf("whiptail is not installed")
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return fmt.Errorf("menu needs a terminal")
	}
	for {
		choice, err := whiptail("XG Backup", "Choose", []string{
			"status", "Show vault status",
			"ls", "List snapshots",
			"browse", "Browse files in a snapshot",
			"restore", "Restore one path or a whole client",
			"all", "Restore every client in a snapshot",
			"add", "Add a backup client",
			"snap", "Take a snapshot now",
			"quit", "Leave the menu",
		})
		if err != nil || choice == "" || choice == "quit" {
			return nil
		}
		if err := menuAct(choice); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			_, _ = whiptail("XG Backup", err.Error(), []string{"ok", "Continue"})
		}
	}
}

func menuAct(choice string) error {
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	switch choice {
	case "status":
		return PrintStatus(cfg)
	case "ls":
		return List(cfg.Pool, "", "", "")
	case "browse":
		return menuBrowse(cfg.Pool)
	case "restore":
		return menuRestore(cfg.Pool, false)
	case "all":
		return menuRestore(cfg.Pool, true)
	case "add":
		name, err := whiptailInput("Client name, for example xgb-laptop")
		if err != nil || name == "" {
			return err
		}
		key, err := AddClient(cfg.Pool, name)
		if err != nil {
			return err
		}
		fmt.Printf("key %s\n", key)
		return nil
	case "snap":
		schedule, err := whiptail("Snapshot", "Schedule", []string{
			"daily", "Daily",
			"weekly", "Weekly",
			"monthly", "Monthly",
			"yearly", "Yearly",
			"hourly", "Hourly",
		})
		if err != nil || schedule == "" {
			return err
		}
		return Snapshot(schedule)
	default:
		return nil
	}
}

func menuBrowse(pool string) error {
	snaps, err := Entries(pool, "", "", "")
	if err != nil {
		return err
	}
	snap, err := pick("Snapshot", snaps)
	if err != nil || snap == "" {
		return err
	}
	clients, err := Entries(pool, snap, "", "")
	if err != nil {
		return err
	}
	client, err := pick("Client", clients)
	if err != nil || client == "" {
		return err
	}
	rel := ""
	for {
		names, err := Entries(pool, snap, client, rel)
		if err != nil {
			return err
		}
		items := []string{"..", "Up"}
		for _, name := range names {
			items = append(items, name, name)
		}
		choice, err := whiptail("Browse "+client, rel, items)
		if err != nil || choice == "" {
			return err
		}
		if choice == ".." {
			if rel == "" || rel == "." {
				return nil
			}
			rel = pathDir(rel)
			continue
		}
		if strings.HasSuffix(choice, "/") {
			next := strings.TrimSuffix(choice, "/")
			if rel == "" || rel == "." {
				rel = next
			} else {
				rel = rel + "/" + next
			}
			continue
		}
		fmt.Printf("%s/%s\n", client, joinMenu(rel, choice))
		return nil
	}
}

func menuRestore(pool string, all bool) error {
	snaps, err := Entries(pool, "", "", "")
	if err != nil {
		return err
	}
	snap, err := pick("Restore snapshot", snaps)
	if err != nil || snap == "" {
		return err
	}
	dest, err := whiptailInput("Destination directory")
	if err != nil || dest == "" {
		return err
	}
	if all {
		return RestoreAll(pool, snap, dest)
	}
	clients, err := Entries(pool, snap, "", "")
	if err != nil {
		return err
	}
	client, err := pick("Client", clients)
	if err != nil || client == "" {
		return err
	}
	rel, err := whiptailInput("Path inside the snapshot, or . for the whole client")
	if err != nil || rel == "" {
		return err
	}
	return Restore(pool, client, snap, rel, dest)
}

func pick(title string, names []string) (string, error) {
	if len(names) == 0 {
		return "", fmt.Errorf("nothing to choose")
	}
	items := make([]string, 0, len(names)*2)
	for _, name := range names {
		items = append(items, name, name)
	}
	return whiptail(title, "Choose", items)
}

func whiptail(title, prompt string, pairs []string) (string, error) {
	args := []string{"--title", title, "--menu", prompt, "22", "76", "12"}
	args = append(args, pairs...)
	cmd := exec.Command("whiptail", args...)
	cmd.Stdin = os.Stdin
	var buf bytes.Buffer
	cmd.Stderr = &buf
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", nil
	}
	return strings.TrimSpace(buf.String()), nil
}

func whiptailInput(prompt string) (string, error) {
	cmd := exec.Command("whiptail", "--title", "XG Backup", "--inputbox", prompt, "10", "70")
	cmd.Stdin = os.Stdin
	var buf bytes.Buffer
	cmd.Stderr = &buf
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", nil
	}
	return strings.TrimSpace(buf.String()), nil
}

func pathDir(rel string) string {
	i := strings.LastIndex(rel, "/")
	if i < 0 {
		return ""
	}
	return rel[:i]
}

func joinMenu(rel, name string) string {
	if rel == "" || rel == "." {
		return name
	}
	return rel + "/" + name
}
