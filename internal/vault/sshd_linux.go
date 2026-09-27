//go:build linux

package vault

import (
	"fmt"
	"os"
	"strings"
)

const sshdPath = "/etc/ssh/sshd_config"

func installSSHD(pool string) error {
	data, err := os.ReadFile(sshdPath)
	if err != nil {
		return err
	}
	block := fmt.Sprintf(`# BEGIN xgbackup
Match User xgb-*
    ChrootDirectory %s/clients/%%u/jail
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
    PermitTunnel no
    PermitTTY no
    AllowAgentForwarding no
    PasswordAuthentication no
# END xgbackup
`, pool)
	content := string(data)
	const begin = "# BEGIN xgbackup\n"
	const end = "# END xgbackup\n"
	if i := strings.Index(content, begin); i >= 0 {
		j := strings.Index(content[i:], end)
		if j < 0 {
			return fmt.Errorf("sshd config has BEGIN xgbackup without END")
		}
		j = i + j + len(end)
		content = content[:i] + block + content[j:]
	} else {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + block
	}
	backup := sshdPath + ".xgbackup.bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if err := os.WriteFile(backup, data, 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(sshdPath, []byte(content), 0o644); err != nil {
		return err
	}
	if err := run("sshd", "-t"); err != nil {
		_ = os.WriteFile(sshdPath, data, 0o644)
		return fmt.Errorf("sshd config check failed, previous file restored: %w", err)
	}
	if err := run("systemctl", "reload", "ssh"); err != nil {
		return err
	}
	return nil
}
