//go:build linux

package vault

import "testing"

func TestMountConflict(t *testing.T) {
	mounts := []mountedFS{{device: "/dev/sda1", mountpoint: "/"}, {device: "/dev/sda5", mountpoint: "/home"}}
	swaps := []string{"/dev/sda2"}
	if err := mountConflict("/dev/sda6", nil, mounts, swaps, "/var/lib/xgbackup/pool"); err != nil {
		t.Fatal(err)
	}
	if err := mountConflict("/dev/sda1", nil, mounts, swaps, "/var/lib/xgbackup/pool"); err == nil {
		t.Fatal("expected mounted root to be refused")
	}
	if err := mountConflict("/dev/sda5", nil, mounts, swaps, "/var/lib/xgbackup/pool"); err == nil {
		t.Fatal("expected mounted home to be refused")
	}
	if err := mountConflict("/dev/sda", []string{"/dev/sda1"}, mounts, nil, "/var/lib/xgbackup/pool"); err == nil {
		t.Fatal("expected a disk with a mounted partition to be refused")
	}
	if err := mountConflict("/dev/sda2", nil, nil, swaps, "/var/lib/xgbackup/pool"); err == nil {
		t.Fatal("expected active swap to be refused")
	}
	if err := mountConflict("/dev/nvme1n1", nil, []mountedFS{{device: "/dev/nvme1n1", mountpoint: "/var/lib/xgbackup/pool"}}, nil, "/var/lib/xgbackup/pool"); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyPoolDevice(t *testing.T) {
	cases := []struct {
		name     string
		real     string
		rootReal string
		rootDisk string
		poolDisk string
		want     poolDeviceAction
	}{
		{"spare partition on root disk", "/dev/sda5", "/dev/sda1", "/dev/sda", "/dev/sda", poolDeviceWarnSameDisk},
		{"root partition", "/dev/sda1", "/dev/sda1", "/dev/sda", "/dev/sda", poolDeviceRefuseRoot},
		{"whole root disk", "/dev/sda", "/dev/sda1", "/dev/sda", "/dev/sda", poolDeviceRefuseWholeDisk},
		{"root is the whole disk", "/dev/sda", "/dev/sda", "/dev/sda", "/dev/sda", poolDeviceRefuseRoot},
		{"separate disk", "/dev/nvme1n1", "/dev/nvme0n1p1", "/dev/nvme0n1", "/dev/nvme1n1", poolDeviceOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyPoolDevice(tc.real, tc.rootReal, tc.rootDisk, tc.poolDisk)
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
