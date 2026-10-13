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

// Menu is the whiptail console. A logged-in user can look at the vault
// without becoming root. Restore and snapshots stay off this menu:
// snapshots are taken by the timers, and a restore is done on another machine.
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
			"web", "Configure the web UI",
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
		if os.Geteuid() != 0 {
			text, err := queryStatusText()
			if err != nil {
				return err
			}
			whiptailMsg("Status", text)
			return nil
		}
		text, err := statusText(cfg)
		if err != nil {
			return err
		}
		whiptailMsg("Status", text)
		fmt.Print(text)
		return nil
	case "ls":
		names, err := menuNames(cfg.Pool, "", "", "")
		if err != nil {
			return err
		}
		if len(names) == 0 {
			whiptailMsg("Snapshots", "no snapshots")
			return nil
		}
		whiptailMsg("Snapshots", strings.Join(names, "\n"))
		return nil
	case "browse":
		return menuBrowse(cfg.Pool)
	case "web":
		return menuWeb()
	default:
		return nil
	}
}

func menuNames(pool, snapshot, client, rel string) ([]string, error) {
	if os.Geteuid() == 0 {
		return Entries(pool, snapshot, client, rel)
	}
	return queryNames(snapshot, client, rel)
}

func menuBrowse(pool string) error {
	snaps, err := menuNames(pool, "", "", "")
	if err != nil {
		return err
	}
	snap, err := pick("Snapshot", snaps)
	if err != nil || snap == "" {
		return err
	}
	clients, err := menuNames(pool, snap, "", "")
	if err != nil {
		return err
	}
	client, err := pick("Client", clients)
	if err != nil || client == "" {
		return err
	}
	rel := ""
	for {
		names, err := menuNames(pool, snap, client, rel)
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
		whiptailMsg("File", client+"/"+joinMenu(rel, choice))
		return nil
	}
}

func menuWeb() error {
	listen, err := whiptailInputDefault("HTTPS listen address. This is reachable without WireGuard.", "0.0.0.0:8787")
	if err != nil || listen == "" {
		return err
	}
	checks, err := whiptailChecks()
	if err != nil || checks == nil {
		return err
	}
	var cidrs []string
	for _, name := range checks {
		if name == "ip" {
			raw, err := whiptailInput("Allowed networks, comma-separated, for example 10.4.1.0/24")
			if err != nil || raw == "" {
				return err
			}
			for _, item := range strings.Split(raw, ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					cidrs = append(cidrs, item)
				}
			}
		}
	}
	user, err := whiptailInput("Web user name")
	if err != nil || user == "" {
		return err
	}
	email := ""
	for _, name := range checks {
		if name == "email" {
			email, err = whiptailInput("Email address for " + user)
			if err != nil || email == "" {
				return err
			}
		}
	}
	password, err := whiptailPassword("Password for " + user)
	if err != nil || password == "" {
		return err
	}
	setup := webSetup{
		Listen:    listen,
		Required:  checks,
		AllowCIDR: cidrs,
		User:      user,
		Password:  password,
		Email:     email,
	}
	if err := queryJSON("POST", "/web/setup", setup, nil); err != nil {
		return err
	}
	whiptailMsg("Web", "The web UI is https://"+listen+"/.\nThe console menu and the web service are enabled.\nThe certificate is created on this machine.")
	return nil
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
	return whiptailInputDefault(prompt, "")
}

func whiptailInputDefault(prompt, init string) (string, error) {
	args := []string{"--title", "XG Backup", "--inputbox", prompt, "10", "70"}
	if init != "" {
		args = append(args, init)
	}
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

func whiptailPassword(prompt string) (string, error) {
	cmd := exec.Command("whiptail", "--title", "XG Backup", "--passwordbox", prompt, "10", "70")
	cmd.Stdin = os.Stdin
	var buf bytes.Buffer
	cmd.Stderr = &buf
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", nil
	}
	return strings.TrimSpace(buf.String()), nil
}

func whiptailChecks() ([]string, error) {
	cmd := exec.Command("whiptail", "--title", "Web", "--checklist", "Required checks. Password is used when none are selected.", "20", "76", "6",
		"password", "Username and password", "on",
		"ip", "Limit by IP range", "off",
		"cookie", "Approved browser cookie", "off",
		"totp", "Authenticator code", "off",
		"email", "Email code", "off")
	cmd.Stdin = os.Stdin
	var buf bytes.Buffer
	cmd.Stderr = &buf
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return nil, nil
	}
	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		return []string{"password"}, nil
	}
	var out []string
	for _, field := range strings.Fields(raw) {
		out = append(out, strings.Trim(field, `"`))
	}
	return out, nil
}

func whiptailMsg(title, text string) {
	if text == "" {
		text = "(empty)"
	}
	cmd := exec.Command("whiptail", "--title", title, "--scrolltext", "--msgbox", text, "22", "76")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
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
