# xgbackup

Open-source Linux backup vault. Windows, macOS, and Linux clients push files into a private drop area. The vault keeps read-only btrfs snapshots. A client key cannot delete that history.

XG Monitor (dashboard, billing, alerts) is a separate product and is not in this repository. The vault writes a status file that a monitor can read. See [docs/monitor-contract.md](docs/monitor-contract.md).

## Layout

- `cmd/xgbackup` — vault CLI (Linux)
- `cmd/xgbackup-sync` — push client (Windows, macOS, Linux)
- `docs/` — install, status, restore, menus and web UI, VPN, hardening, heartbeat contract
- `packaging/` — Debian package, AlmaLinux RPM, systemd timers

## License

MIT. See [LICENSE](LICENSE).
