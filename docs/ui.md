# Menus, web UI, and Samba

The CLI stays. These are the everyday ways to list snapshots and restore.

## List and restore

```sh
sudo xgbackup ls
sudo xgbackup ls --snapshot daily-20261012
sudo xgbackup ls --snapshot daily-20261012 --client xgb-demo --path nested
sudo xgbackup restore --client xgb-demo --snapshot daily-20261012 --path nested/sample.txt --dest /tmp/sample.txt
sudo xgbackup restore --client xgb-demo --snapshot daily-20261012 --path . --dest /tmp/xgb-demo
sudo xgbackup restore --all --snapshot daily-20261012 --dest /tmp/all-clients
```

`--path .` restores that client's whole tree. `--all` restores every client into `dest/<client>/`.

## Console menu

On the vault, as root:

```sh
sudo xgbackup menu
```

The menu uses whiptail. It does not ask for the web checks, because the person is already logged in on the machine.

## Permissions

- **Backup.** An `xgb-*` key can write only to that client's drop. It cannot read snapshots.
- **Setup.** Root, plus names added with `sudo xgbackup access setup add andy`. Setup can use the web UI for every feature.
- **Restore.** `sudo xgbackup access restore add sam --client xgb-demo --prefix docs` lets sam restore that prefix through the web UI. Setup users can restore everything.

```sh
sudo xgbackup access list
```

## Web UI

The web UI listens on `127.0.0.1:8787` unless `/etc/xgbackup/config.json` says otherwise. It is not a public service. Reach it through SSH or WireGuard, not by opening it to the internet.

```json
"web": {
  "listen": "127.0.0.1:8787",
  "required": ["ip", "cookie", "password", "totp"],
  "allow_cidr": ["127.0.0.1/32", "10.4.1.0/24"]
}
```

`required` is any mix of `ip`, `cookie`, `password`, and a second factor: `totp` (Google Authenticator) or `email`. If `required` is omitted, password is required.

```sh
sudo xgbackup access setup add andy
printf '%s\n' 'a-long-password' | sudo xgbackup web user add andy --email andy@example.com
sudo xgbackup web totp andy
sudo systemctl enable --now xgbackup-web.service
sudo xgbackup web approve
```

`web approve` prints a code. Enter it in the browser to store the cookie that permits logon. The cookie is approved by someone who can log into the vault.

A web restore is written under `/var/lib/xgbackup/restores/`.

## Samba

```sh
sudo xgbackup smb sync
sudo smbpasswd -a xgb-demo
```

Each client gets a writable share of its drop. `xgbackup-snaps` is a read-only share of the snapshot tree for setup and restore users. On the LAN, map the share. When the path crosses the internet, connect WireGuard first and map the share at the vault's tunnel address. See [vpn-offsite.md](vpn-offsite.md).
