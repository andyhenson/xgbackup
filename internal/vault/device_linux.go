//go:build linux

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var deviceName = regexp.MustCompile(`^(nvme[0-9]+n[0-9]+(p[0-9]+)?|xvd[a-z][0-9]*|sd[a-z][0-9]*|vd[a-z][0-9]*)$`)

func resolveDevice(dev string) (string, error) {
	if !strings.HasPrefix(dev, "/dev/") {
		return "", fmt.Errorf("device must be an absolute path under /dev")
	}
	real, err := filepath.EvalSymlinks(dev)
	if err != nil {
		return "", err
	}
	base := filepath.Base(real)
	if !deviceName.MatchString(base) {
		return "", fmt.Errorf("unsupported device name %s", base)
	}
	return "/dev/" + base, nil
}

func parentDisk(dev string) (string, error) {
	name := filepath.Base(dev)
	part := "/sys/class/block/" + name + "/partition"
	if _, err := os.Stat(part); err != nil {
		if os.IsNotExist(err) {
			return "/dev/" + name, nil
		}
		return "", err
	}
	link, err := os.Readlink("/sys/class/block/" + name)
	if err != nil {
		return "", err
	}
	parent := filepath.Base(filepath.Dir(link))
	return "/dev/" + parent, nil
}

func rootSource() (string, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "/" {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("root filesystem not found in /proc/mounts")
}

func mountPoint(dev string) (string, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == dev {
			return fields[1], nil
		}
	}
	return "", nil
}

func assertSafePoolDevice(dev, pool string) error {
	real, err := resolveDevice(dev)
	if err != nil {
		return err
	}
	rootDev, err := rootSource()
	if err != nil {
		return err
	}
	rootReal, err := resolveDevice(rootDev)
	if err != nil {
		return fmt.Errorf("root device: %w", err)
	}
	rootDisk, err := parentDisk(rootReal)
	if err != nil {
		return err
	}
	poolDisk, err := parentDisk(real)
	if err != nil {
		return err
	}
	if poolDisk == rootDisk {
		return fmt.Errorf("refusing to use %s: it is on the root disk %s", real, rootDisk)
	}
	mp, err := mountPoint(real)
	if err != nil {
		return err
	}
	if mp != "" && mp != pool {
		return fmt.Errorf("refusing to use %s: mounted at %s", real, mp)
	}
	return nil
}

func blkid(dev, tag string) (string, error) {
	cmdOut, err := output("blkid", "-o", "value", "-s", tag, dev)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 2") || strings.Contains(strings.ToLower(err.Error()), "exit status") {
			return "", nil
		}
		return "", nil
	}
	return cmdOut, nil
}
