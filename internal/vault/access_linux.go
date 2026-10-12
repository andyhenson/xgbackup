//go:build linux

package vault

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xgbackup/xgbackup/internal/access"
	"github.com/xgbackup/xgbackup/internal/config"
	"golang.org/x/crypto/bcrypt"
)

const webUsersPath = "/etc/xgbackup/web-users.json"

type webUser struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	TOTP     string `json:"totp,omitempty"`
}

func loadWebUsers() ([]webUser, error) {
	data, err := os.ReadFile(webUsersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var users []webUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func saveWebUsers(users []webUser) error {
	if err := os.MkdirAll("/etc/xgbackup", 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(webUsersPath, data, 0o600)
}

func AddWebUser(name, email string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, ":") {
		return fmt.Errorf("user name is required")
	}
	fmt.Fprint(os.Stderr, "password: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	pw := strings.TrimSpace(line)
	if len(pw) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	users, err := loadWebUsers()
	if err != nil {
		return err
	}
	updated := false
	for i := range users {
		if users[i].Name == name {
			users[i].Password = string(hash)
			if email != "" {
				users[i].Email = email
			}
			updated = true
		}
	}
	if !updated {
		users = append(users, webUser{Name: name, Password: string(hash), Email: email})
	}
	if err := saveWebUsers(users); err != nil {
		return err
	}
	fmt.Printf("web user %s\n", name)
	return nil
}

func SetWebTOTP(name string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	users, err := loadWebUsers()
	if err != nil {
		return err
	}
	secret, err := access.NewTOTPSecret()
	if err != nil {
		return err
	}
	found := false
	for i := range users {
		if users[i].Name == name {
			users[i].TOTP = secret
			found = true
		}
	}
	if !found {
		return fmt.Errorf("web user %s does not exist", name)
	}
	if err := saveWebUsers(users); err != nil {
		return err
	}
	fmt.Printf("otpauth://totp/xgbackup:%s?secret=%s&issuer=xgbackup\n", name, secret)
	fmt.Println("Add that URL to Google Authenticator.")
	return nil
}

func AddSetupUser(name string) error {
	return editConfig(func(cfg *config.Config) error {
		for _, have := range cfg.SetupUsers {
			if have == name {
				return nil
			}
		}
		cfg.SetupUsers = append(cfg.SetupUsers, name)
		fmt.Printf("setup %s\n", name)
		return nil
	})
}

func AddRestoreUser(name, client, prefix string) error {
	if client == "" {
		client = "*"
	}
	return editConfig(func(cfg *config.Config) error {
		cfg.Restore = append(cfg.Restore, config.RestoreRule{User: name, Client: client, Prefix: prefix})
		fmt.Printf("restore %s client %s prefix %s\n", name, client, prefix)
		return nil
	})
}

func ListAccess() error {
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	fmt.Println("backup: xgb-* client keys, write only to that client's drop")
	if len(cfg.SetupUsers) == 0 {
		fmt.Println("setup: root")
	} else {
		fmt.Printf("setup: root %s\n", strings.Join(cfg.SetupUsers, " "))
	}
	if len(cfg.Restore) == 0 {
		fmt.Println("restore: setup users")
		return nil
	}
	for _, rule := range cfg.Restore {
		prefix := rule.Prefix
		if prefix == "" {
			prefix = "(all)"
		}
		fmt.Printf("restore: %s client %s path %s\n", rule.User, rule.Client, prefix)
	}
	return nil
}

func editConfig(fn func(*config.Config) error) error {
	if err := requireRoot(); err != nil {
		return err
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	if err := fn(&cfg); err != nil {
		return err
	}
	return config.Save(config.ConfigPath, cfg)
}
