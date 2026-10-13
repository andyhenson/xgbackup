# Menus, web UI, and Samba

The CLI stays. People use the menu and the web UI without logging in as root.

## List

```sh
xgbackup ls
xgbackup ls --snapshot daily-20261012
xgbackup ls --snapshot daily-20261012 --client xgb-demo --path nested
xgbackup status
```

These work for a logged-in user. The vault query service reads the snapshots. The user does not become root, and a backup key still cannot delete history.

Restore is not done on the vault. A restore is made onto another machine.

## Console menu

```sh
xgbackup menu
```

The menu shows status, lists snapshots, browses files, and configures the web UI. It does not take snapshots and it does not restore files onto this machine. Snapshots are taken by the timers.

The first time the web UI is configured from this menu, the console menu service and the web service are enabled.

## Web UI

The web UI is HTTPS, on the address chosen in the menu, so it can be opened without WireGuard. The certificate is created on the vault. A browser will warn until that certificate is trusted.

The menu asks which checks to require. `required` is any mix of `ip`, `cookie`, `password`, and a second factor: `totp` (Google Authenticator) or `email`. If none are selected, password is required.

```sh
xgbackup web totp USER
xgbackup web approve
```

`web approve` prints a code. Enter it in the browser to store the cookie that permits logon.

## Permissions

- **Backup.** An `xgb-*` key can write only to that client's drop. It cannot read snapshots and it cannot configure the web UI.
- **Setup.** A user who configures the web UI is recorded as a setup user. Later changes are limited to setup users.
- **Restore.** Restore permissions still apply when a restore is run for another machine. They are not a menu item on the vault.

```sh
xgbackup access list
```

## Samba

```sh
sudo xgbackup smb sync
sudo smbpasswd -a xgb-demo
```

Each client gets a writable share of its drop. `xgbackup-snaps` is a read-only share of the snapshot tree for setup and restore users. On the LAN, map the share. When the path crosses the internet, connect WireGuard first and map the share at the vault's tunnel address. See [vpn-offsite.md](vpn-offsite.md).
