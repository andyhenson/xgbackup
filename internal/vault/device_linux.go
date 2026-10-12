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

type poolDeviceAction int

const (
	poolDeviceOK poolDeviceAction = iota
	poolDeviceWarnSameDisk
	poolDeviceRefuseRoot
	poolDeviceRefuseWholeDisk
)

// classifyPoolDevice decides whether init may format real.
// rootReal is the device mounted at /. rootDisk is that device's whole disk.
// poolDisk is real's whole disk.
func classifyPoolDevice(real, rootReal, rootDisk, poolDisk string) poolDeviceAction {
	if real == rootReal {
		return poolDeviceRefuseRoot
	}
	if real == rootDisk {
		return poolDeviceRefuseWholeDisk
	}
	if poolDisk == rootDisk {
		return poolDeviceWarnSameDisk
	}
	return poolDeviceOK
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
	switch classifyPoolDevice(real, rootReal, rootDisk, poolDisk) {
	case poolDeviceRefuseRoot:
		return fmt.Errorf("refusing to use %s: it holds the root filesystem", real)
	case poolDeviceRefuseWholeDisk:
		return fmt.Errorf("refusing to use %s: formatting that disk would erase the root filesystem on %s", real, rootReal)
	case poolDeviceWarnSameDisk:
		fmt.Fprintf(os.Stderr, "warning: %s is on %s, the same disk as /. One disk failure loses the operating system and the backups.\n", real, rootDisk)
	}
	mounts, err := mountedFilesystems()
	if err != nil {
		return err
	}
	swaps, err := swapDevices()
	if err != nil {
		return err
	}
	children, err := childDevices(real)
	if err != nil {
		return err
	}
	return mountConflict(real, children, mounts, swaps, pool)
}

type mountedFS struct {
	device     string
	mountpoint string
}

func mountConflict(real string, children []string, mounts []mountedFS, swaps []string, pool string) error {
	for _, m := range mounts {
		if m.device == real {
			if m.mountpoint != pool {
				return fmt.Errorf("refusing to use %s: mounted at %s", real, m.mountpoint)
			}
			continue
		}
		for _, child := range children {
			if m.device == child {
				return fmt.Errorf("refusing to use %s: %s is mounted at %s", real, child, m.mountpoint)
			}
		}
	}
	for _, sw := range swaps {
		if sw == real {
			return fmt.Errorf("refusing to use %s: it is active swap", real)
		}
		for _, child := range children {
			if sw == child {
				return fmt.Errorf("refusing to use %s: %s is active swap", real, child)
			}
		}
	}
	return nil
}

func mountedFilesystems() ([]mountedFS, error) {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	var out []mountedFS
	for _, line := range strings.Split(string(data), "\n") {
		sep := strings.Index(line, " - ")
		if sep < 0 {
			continue
		}
		fields := strings.Fields(line[:sep])
		if len(fields) < 5 {
			continue
		}
		dev, err := deviceForMajorMinor(fields[2])
		if err != nil {
			continue
		}
		out = append(out, mountedFS{device: dev, mountpoint: unescapeMount(fields[4])})
	}
	return out, nil
}

func deviceForMajorMinor(majmin string) (string, error) {
	link, err := os.Readlink("/sys/dev/block/" + majmin)
	if err != nil {
		return "", err
	}
	return "/dev/" + filepath.Base(link), nil
}

func childDevices(disk string) ([]string, error) {
	name := filepath.Base(disk)
	entries, err := os.ReadDir("/sys/class/block/" + name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		part := "/sys/class/block/" + name + "/" + e.Name() + "/partition"
		if _, err := os.Stat(part); err != nil {
			continue
		}
		out = append(out, "/dev/"+e.Name())
	}
	return out, nil
}

func swapDevices() ([]string, error) {
	data, err := os.ReadFile("/proc/swaps")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "/dev/") {
			continue
		}
		real, err := filepath.EvalSymlinks(fields[0])
		if err != nil {
			out = append(out, fields[0])
			continue
		}
		out = append(out, "/dev/"+filepath.Base(real))
	}
	return out, nil
}

func unescapeMount(path string) string {
	return strings.ReplaceAll(path, `\040`, " ")
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
