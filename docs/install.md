# Install the vault

The vault runs on Debian or AlmaLinux. The backup pool is a btrfs filesystem on a disk that is not the operating-system disk.

## Packages

Debian:

```sh
sudo apt-get install ./xgbackup_0.1.0-1_amd64.deb
```

AlmaLinux, from the RPM built with `make rpm`:

```sh
sudo dnf install ./xgbackup-0.1.0-1.*.x86_64.rpm
```

Both packages install `/usr/bin/xgbackup`, `/usr/bin/xgbackup-sync`, and systemd timers. Hourly snapshots stay disabled until you turn them on.

From a git checkout you can `make` and `sudo make install` instead of using a package. Building needs Go and `btrfs-progs`.

## Create the pool

Find the spare disk with `lsblk`. This example uses `/dev/nvme1n1`. `init` refuses to format the disk that holds `/`.

```sh
sudo xgbackup init --device /dev/nvme1n1
```

`init` formats that disk as btrfs if it has no filesystem, mounts it at `/var/lib/xgbackup/pool`, adds an fstab entry, installs an sshd rule for `xgb-*` users, creates the demo client `xgb-demo`, and enables the daily, weekly, monthly, and yearly timers.

The demo private key is `/etc/xgbackup/clients/xgb-demo/id_ed25519`. Copy that file to the client machine and keep it private. The vault copy stays mode `600` and owned by root.

## Another client

```sh
sudo xgbackup client add xgb-laptop
sudo xgbackup client list
```

Client names start with `xgb-` and use lowercase letters, digits, and hyphens.

## Push from a client

Build or unpack `xgbackup-sync` for the client operating system (`make dist` produces the archives). On the client:

```sh
ssh-keyscan -H VAULT > known_hosts
xgbackup-sync -host VAULT -user xgb-demo -key id_ed25519 -src /path/to/data -known-hosts known_hosts
```

The client writes only to `/drop` inside its chroot. See [vpn-offsite.md](vpn-offsite.md) when the vault is not on the same LAN.

## Optional hourly snapshots

Edit `/etc/xgbackup/config.json` and set `schedules.hourly.enabled` to `true`. Then:

```sh
sudo systemctl enable --now xgbackup-snapshot-hourly.timer
```

`hourly.keep` defaults to 24.
