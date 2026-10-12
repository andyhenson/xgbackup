# Hardening checklist

Use this after `xgbackup init` and again when you add a client.

- The pool device is not mounted, and neither is any partition or swap on it, except the pool mount itself after init. `init` also refuses the partition that holds `/` and the whole disk that contains that partition. A spare unmounted partition on the same disk is allowed; `init` warns that one disk failure loses the operating system and the backups.
- `/etc/fstab` mounts the pool by UUID at `/var/lib/xgbackup/pool`.
- Each client is a user named `xgb-...` with shell `/usr/sbin/nologin` and is not in the `sudo` group.
- `/etc/ssh/sshd_config` ends with the `# BEGIN xgbackup` block: `ChrootDirectory` is the client jail, `ForceCommand` is `internal-sftp`, and forwarding, TTY, and passwords are off for `xgb-*`.
- `sshd -t` succeeds and `systemctl reload ssh` has been run after that block was installed.
- The jail directory and every parent up to `/` are owned by root and not group- or world-writable.
- The writable drop is only `jail/drop`. Snapshot directories are siblings of `jail`, mode `700`, owned by root.
- New snapshots are read-only (`btrfs property get ... ro` prints `ro=true`).
- Client private keys live in `/etc/xgbackup/clients/<name>.key`, mode `600`, owner root. The copy on the workstation is the one the sync client uses.
- Daily, weekly, monthly, and yearly timers are enabled. Hourly is enabled only if you opted in.
- `/var/lib/xgbackup/status.json` exists and matches [monitor-contract.md](monitor-contract.md).
- SSH from the public internet is closed. Offsite clients reach the vault through a VPN. See [vpn-offsite.md](vpn-offsite.md).
- Root on the vault can still delete snapshots. This is not a promise of immutability against the root account. It is read-only snapshots plus a separate client role.
- On AlmaLinux, if SELinux is enforcing, confirm `sshd` may enter the chroot and read the client home `.ssh` directory (`ausearch` / `restorecon` on `/home/xgb-*` and the pool). Debian does not apply that policy.

Prove the client boundary:

```sh
sudo tests/acceptance.sh
```
