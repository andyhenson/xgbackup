//go:build linux

package vault

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func requireRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must be root")
	}
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
	}
	return nil
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), text)
	}
	return text, nil
}
