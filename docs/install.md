# Install the vault

The vault runs on Debian or AlmaLinux. The backup pool is a btrfs filesystem on its own disk, or on a spare partition when the machine has only one disk.

## Packages

Debian:

Copy the package somewhere the `_apt` user can read it. Installing a file under `/root` makes apt report an unsandboxed download and `Permission denied`.

```sh
sudo cp xgbackup_0.2.0-1_amd64.deb /tmp/
sudo apt-get install /tmp/xgbackup_0.2.0-1_amd64.deb
```

The package masks `sleep.target`, `suspend.target`, `hibernate.target`, and `hybrid-sleep.target` so the vault does not suspend.

AlmaLinux, from the RPM built with `make rpm`:

```sh
sudo dnf install ./xgbackup-0.2.0-1.*.x86_64.rpm
```

Both packages install `/usr/bin/xgbackup`, `/usr/bin/xgbackup-sync`, and systemd timers. Hourly snapshots stay disabled until you turn them on.

From a git checkout you can `make` and `sudo make install` instead of using a package. Building needs Go and `btrfs-progs`.

## Create the pool

Find the spare disk or partition with `lsblk`. This example uses `/dev/nvme1n1`. `init` refuses a device that is mounted, including any mounted partition or active swap on a whole disk, so it will not format the running system. It also refuses the partition that holds `/` and the whole disk that contains that partition. Another partition on that same disk is allowed when it is not mounted. In that case `init` warns that one disk failure loses the operating system and the backups.

```sh
sudo xgbackup init --device /dev/nvme1n1
```

`init` formats that disk as btrfs if it has no filesystem, mounts it at `/var/lib/xgbackup/pool`, adds an fstab entry, installs an sshd rule for `xgb-*` users, creates the demo client `xgb-demo`, and enables the daily, weekly, monthly, and yearly timers.

The demo private key is `/etc/xgbackup/clients/xgb-demo.key`. Copy that file to the client machine and keep it private. The vault copy stays mode `600` and owned by root. An older install keeps a symlink at `xgb-demo/id_ed25519`.

## Another client

```sh
sudo xgbackup client add xgb-laptop
sudo xgbackup client list
```

Client names start with `xgb-` and use lowercase letters, digits, and hyphens.

## Push from a client

Build or unpack `xgbackup-sync` for the client operating system (`make dist` produces the archives). On the client:

Copy the key into `~/.xgbackup` and remember the vault on the first push. On Windows that directory is `%USERPROFILE%\.xgbackup`.

```sh
mkdir -p ~/.xgbackup
cp xgb-demo.key ~/.xgbackup/xgb-demo.key
chmod 600 ~/.xgbackup/xgb-demo.key
ssh-keyscan -H VAULT > ~/.xgbackup/known_hosts
xgbackup-sync -host VAULT -user xgb-demo -key ~/.xgbackup/xgb-demo.key -src /path/to/data -known-hosts ~/.xgbackup/known_hosts
```

The next push only needs the source directory:

```sh
xgbackup-sync -src /path/to/data
```

Unchanged files are reported as `identical`. The client writes only to `/drop` inside its chroot. See [vpn-offsite.md](vpn-offsite.md) when the vault is not on the same LAN.

Snapshots are taken under `/var/lib/xgbackup/pool/snaps/<snapshot>/<client>/`. Daily, weekly, monthly, and yearly names keep the date (`daily-20261012`). Hourly names keep the hour (`hourly-20261012T23`). A second run on the same day replaces that snapshot.

## Optional hourly snapshots

Edit `/etc/xgbackup/config.json` and set `schedules.hourly.enabled` to `true`. Then:

```sh
sudo systemctl enable --now xgbackup-snapshot-hourly.timer
```

`hourly.keep` defaults to 24.
