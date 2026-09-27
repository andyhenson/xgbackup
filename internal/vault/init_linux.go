//go:build linux

package vault

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/xgbackup/xgbackup/internal/config"
)

func Init(device string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	dev, err := resolveDevice(device)
	if err != nil {
		return err
	}
	pool := config.DefaultPool
	if err := assertSafePoolDevice(dev, pool); err != nil {
		return err
	}
	if err := os.MkdirAll("/var/lib/xgbackup", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(pool, 0o755); err != nil {
		return err
	}
	mounted, err := mountPoint(dev)
	if err != nil {
		return err
	}
	fsType, _ := blkid(dev, "TYPE")
	switch {
	case mounted == pool:
		fmt.Printf("pool already mounted at %s\n", pool)
	case mounted != "":
		return fmt.Errorf("%s is mounted at %s", dev, mounted)
	case fsType == "":
		if err := run("mkfs.btrfs", "-L", "xgbackup", dev); err != nil {
			return err
		}
		if err := mountPool(dev, pool); err != nil {
			return err
		}
	case fsType == "btrfs":
		label, _ := blkid(dev, "LABEL")
		if label != "" && label != "xgbackup" {
			return fmt.Errorf("refusing to mount %s: btrfs label is %q", dev, label)
		}
		if err := mountPool(dev, pool); err != nil {
			return err
		}
	default:
		return fmt.Errorf("refusing to format %s: existing filesystem %s", dev, fsType)
	}
	if err := mkdirRoot(pool+"/clients", 0o755); err != nil {
		return err
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		id, err := newVaultID()
		if err != nil {
			return err
		}
		cfg = config.Default(id)
	}
	cfg.Pool = pool
	if cfg.VaultID == "" {
		id, err := newVaultID()
		if err != nil {
			return err
		}
		cfg.VaultID = id
	}
	if err := config.Save(config.ConfigPath, cfg); err != nil {
		return err
	}
	if err := installSSHD(pool); err != nil {
		return err
	}
	key, err := AddClient(pool, "xgb-demo")
	if err != nil {
		return err
	}
	if err := enableTimers(); err != nil {
		return err
	}
	fmt.Printf("vault %s\n", cfg.VaultID)
	fmt.Printf("pool %s on %s\n", pool, dev)
	fmt.Printf("demo client xgb-demo\n")
	fmt.Printf("demo key %s\n", key)
	return nil
}

func mountPool(dev, pool string) error {
	if err := run("mount", dev, pool); err != nil {
		return err
	}
	uuid, err := blkid(dev, "UUID")
	if err != nil {
		return err
	}
	if uuid == "" {
		return fmt.Errorf("blkid did not return a UUID for %s", dev)
	}
	line := fmt.Sprintf("UUID=%s %s btrfs defaults,noatime 0 2\n", uuid, pool)
	fstab, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return err
	}
	if strings.Contains(string(fstab), pool) {
		return nil
	}
	f, err := os.OpenFile("/etc/fstab", os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func enableTimers() error {
	if err := run("systemctl", "daemon-reload"); err != nil {
		return err
	}
	for _, name := range []string{"daily", "weekly", "monthly", "yearly"} {
		unit := "xgbackup-snapshot-" + name + ".timer"
		if _, err := os.Stat("/usr/lib/systemd/system/" + unit); err != nil {
			return fmt.Errorf("missing %s (install the xgbackup package or run make install)", unit)
		}
		if err := run("systemctl", "enable", "--now", unit); err != nil {
			return err
		}
	}
	return nil
}

func newVaultID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
