# xgbackup — Cursor build brief (initial build)

**Owner:** Andy Henson  
**Open-source repo name:** `xgbackup`  
**Product name (working):** XG Backup  

## Scope split

| Piece | Licence | Repo | When |
|--------|---------|------|------|
| Vault + sync clients + ingest tools | Open source | **`xgbackup`** | **Initial build (now)** |
| XG Monitor (dashboard, billing, alerts SaaS) | Proprietary | Separate repo (name TBD) | Before first *public* release; **not** in this initial build |

Monitoring remains essential to the business model and ships for the first public release, but it is developed separately so `xgbackup` stays a clean open-source vault project.

## Product in one paragraph

XG Backup is a hardened **Linux backup vault** (Debian for now; AlmaLinux only around M5 if it still seems necessary). Windows, Mac, and Linux clients **push** data into per-client drop areas on the vault (the vault does not crawl the network with broad credentials). Ingest is a sync tool and, as core, a **per-client Samba share** (LAN, or WireGuard when the path crosses the internet). Once data lands, **btrfs** (or similar) takes read-only snapshots on optional hourly plus daily / weekly / monthly / yearly schedules with user-configurable retention. People configure and use the vault from the CLI, from whiptail-style menus on the box, or from a locked-down local web UI — listing files and snapshots and restoring one path or everything — without needing to be a Linux guru. Admin access is tightly controlled so compromised LAN machines or malicious agents cannot destroy historical snapshots. We do **not** host customer backup data in “our cloud.” Optional proprietary **XG Monitor** (later repo) will watch vault health for a low annual fee.

## Decisions locked

| Topic | Decision |
|--------|----------|
| Open-source repo | **`xgbackup`** |
| Pricing (Monitor, later) | ~£35 / €40 / US$50 per year **per vault machine** |
| Vault software | Fully open source |
| Vault OS | Linux only. **Debian for now** — it covers everything we need. AlmaLinux only around M5 if it still seems necessary |
| Endpoints | Windows, Mac, Linux clients push into the vault |
| Ingest | **Drop area + sync tool**, and **per-client Samba** (core, around M2). Samba on the LAN; WireGuard (or the VPN-supported pattern) when the share is reached over the internet. Windows gets `xgbackup-sync` as well as the share |
| Sync profile | Host, user, key, and known_hosts live in a known profile so they are not flags on every call: `~/.xgbackup` on Linux; `%USERPROFILE%\.xgbackup` on Windows once that client exists |
| Keys | File is `<name>.key` (for example `xgb-demo.key`), not `<name>/id_ed25519`, so the name survives when the file is copied. Preferred: secret key generated on the client, public key copied to the vault. Vault-side generation also allowed |
| Sync report | Unchanged files are **identical** / **already copied**, not `skipped` |
| Snapshots | btrfs or similar; hourly (optional) + D/W/M/Y; configurable retention. Names drop the long timestamp: D/W/M/Y keep the date; only hourly keeps `Thh`. Layout is snapshot-first: `snaps/<snapshot>/<client>/...` so one snapshot holds every client and a full restore is one tree |
| Interfaces | CLI stays. Whiptail-style menus for a logged-in user on the vault, and web pages for a browser. The web UI configures every feature and lists files and snapshots. Web access uses the checks in **Web setup access** |
| Web setup access | Four checks. The operator chooses how many are required, including all four for maximum security: **(a)** client IP in a limited range, **(b)** a cookie on that PC which permits logon (the cookie is approved by logging into the system), **(c)** user and password, **(d)** a second factor, Google Authenticator or email. Access from the machine console itself can require fewer of these, because the person is already a local user |
| Permissions | Three categories: **backup** (write for an automated task; usually per source machine, may be shared), **setup** (typically one person, on the box or via the web UI under the checks above), **restore** (default restore-all; can be limited to part of the archive). Restore is not the same key as backup |
| Import | One vault can be copied to another, preserving all client directories. Do not copy snapshots |
| Vault install | Standard install masks `sleep.target`, `suspend.target`, `hibernate.target`, and `hybrid-sleep.target` so the box does not suspend |
| Local apt | A `.deb` installed from `/root` makes `_apt` fail (`Permission denied`, unsandboxed-as-root notice). Install from a path `_apt` can read (for example `/tmp`). Do not treat that notice as normal |
| Security | Separate write-to-partition from admin; protect read-only snapshots. Web setup access as above |
| Offsite | VPN-supported pattern (WireGuard) |
| Our cloud storage | Out |
| Cloud repo backup | **Core**, around M4 — tools that pull cloud repos into the vault. Not “later” |
| Packaging | **M5**, after the product works well. Not the next step |
| Monitor in initial build | **No** — separate proprietary repo |
| Monitor in first public release | **Yes** — essential to the business model |

## Suggested `xgbackup` repo layout

```
xgbackup/
  vault/           # Linux vault: drop areas, snapshots, admin CLI, hardening
  sync-client/     # Cross-platform sync into drop areas (Win/Mac/Linux)
  smb/             # M2: per-client Samba ingest
  ui/              # M2: whiptail-style menus and locked-down web UI
  cloud-repos/     # M4: cloud repo backup tools
  docs/            # Install, harden, restore, VPN offsite guide
  packaging/       # M5: Debian packages; client installers; Alma only if still needed
  # later: nas/    # NAS-oriented tooling
```

Do **not** put Monitor server/web/billing in this repo.

## Suggested stack (defaults)

- **Vault:** Bash + small Go or Python control plane; btrfs subvolumes; systemd timers for snapshots
- **Sync client:** Go or Rust; Win/Mac/Linux → drop area over SSH or a minimal authenticated channel; profile in `~/.xgbackup` (Linux) so host, user, key, and known_hosts are not repeated on every call; VPN documented for offsite
- **Menus:** whiptail-style textual UI on the vault console. Web UI with the four configurable checks: IP range, approved cookie, user/password, and a second factor (Google Authenticator or email). Console access can require fewer of those checks
- **SMB:** per-client shares for drop-in copies; WireGuard when the share is not on the LAN
- **Hooks for Monitor (later):** emit local status / optional heartbeat endpoint the proprietary Monitor can consume — keep the interface small and documented, no Monitor code here

## Security principles (non-negotiable)

1. Everyday clients can **write only** to their assigned drop area (backup permission). Backup keys are for backup tasks, usually per source machine, and may be shared across tasks.
2. Snapshot history is **read-only** to clients; delete/alter requires high-friction admin (setup permission). Setup is typically one person, on the box or via the web UI. Web setup can require any combination of: an IP in a limited range, an approved cookie on that PC, a user and password, and a second factor (Google Authenticator or email). The operator chooses how many, up to all four. The cookie permits logon and is approved by logging into the system.
3. Restore permission is separate from backup. Default is restore-all; it can be limited to part of the archive.
4. Compromising an endpoint must not yield vault admin or setup.
5. No requirement to expose the vault to the public internet; offsite via VPN (WireGuard). The web UI is not a public service. Console access on the machine itself can require fewer of the web checks, because the person is already a local user.

## Milestones (`xgbackup` only)

**Status:** M0 and M1 are largely proven on Debian (xgbackup 0.1.2: init, demo client, scheduled snapshots, `xgbackup-sync` of a home directory, a manual restore). Stranger-ready packaging and a usable non-guru workflow are not done. AlmaLinux is out of scope until M5, and only if it still seems necessary. The CLI stays; it is not enough on its own.

### M0 — Skeleton (largely done)

- Create **`xgbackup`** repo
- Vault boots on Debian with btrfs pool + one demo client subvolume (packaging is M5; AlmaLinux only then, if it still seems necessary)
- Docs stub: install, status, restore

**Corrections still owed** (not a reopen of the milestone):

- Standard install masks `sleep.target`, `suspend.target`, `hibernate.target`, and `hybrid-sleep.target`
- Local `.deb` install uses a path `_apt` can read (for example `/tmp`), so apt does not report an unsandboxed download and `Permission denied` for a file under `/root`

### M1 — Vault core (largely done)

- Create/list client drop areas
- Snapshot scheduler + retention (D/W/M/Y; hourly optional)
- Local status: last snapshot, free space, failures
- Prove a normal client write path cannot delete snapshots

**Corrections still owed:**

- Key file is `<name>.key` (for example `xgb-demo.key`), not `<name>/id_ed25519`
- Sync profile in `~/.xgbackup`: host, user, key, and known_hosts, so `xgbackup-sync` does not need them on every call
- Report unchanged files as **identical** / **already copied**, not `skipped`
- Shorter snapshot names: D/W/M/Y keep the date; only hourly keeps `Thh`
- Snapshot-first layout: `snaps/<snapshot>/<client>/...` (today: `pool/clients/<client>/snaps/...`) so one snapshot contains every client

### M2 — Friendly configuration and SMB (next priority)

Everyday use without guru flags. This is the next work.

- Whiptail-style menus for a logged-in user on the vault
- Web UI: configure every feature; list files and snapshots. Require any chosen mix of the four checks (IP range, approved cookie, user/password, Google Authenticator or email), including all four. Console menus can require fewer checks
- List snapshots and browse files, then restore one path or restore all (`xgbackup ls`, the web UI, or Samba)
- Per-client Samba drop: LAN, or WireGuard when the path crosses the internet
- Three permission categories: backup, setup, restore
- Browse via `xgbackup ls`, the web UI, or Samba (over VPN for Windows)

### M3 — Sync clients and vault copy

- Windows and Mac `xgbackup-sync`, with the same remembered profile (`%USERPROFILE%\.xgbackup` on Windows)
- Clear failure modes (unreachable, disk full, auth failed)
- On-LAN first; document VPN / WireGuard for offsite
- Import: copy one vault to another, preserving all client directories; do not copy snapshots

### M4 — Cloud repo backup (core)

- Tools to pull cloud repos into the vault
- Same drop areas, snapshots, and permissions as other clients

### M5 — Package and stranger-ready

Only once M2–M4 work well.

- Debian packages. AlmaLinux only if it still seems necessary by then
- Client installers
- Restore drill documented and tested
- Hardening checklist
- Document a minimal **status/heartbeat contract** for future XG Monitor (no Monitor implementation)
- `_apt` local-deb install path, if that M0 correction is still open

## Later (still in `xgbackup`, after initial build)

- NAS-oriented tooling

## Separate track (not this repo)

- XG Monitor backend, web, trial, annual billing
- Consumes the documented heartbeat/status contract from the vault

## Definition of done (initial `xgbackup` build)

A stranger can install the vault on a cheap Debian machine and configure it from the menus or the locked-down web UI. They can push from Windows (sync client or Samba), see snapshots on schedule, list snapshots and files, and restore one file or everything — following the docs, without being a Linux guru.

## Open items (not blocking M0)

- Final product brand (repo stays `xgbackup`)
- Exact break-glass admin mechanism
- Monitor repo name and payment provider (when that track starts)

---

*Living brief — update when product decisions change.*
