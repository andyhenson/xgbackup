# Status

```sh
sudo xgbackup status
```

The command prints the vault id, free space, and the last snapshot for each client. It also rewrites `/var/lib/xgbackup/status.json`.

Snapshot timers call `xgbackup snapshot`, which updates the same file. A client that has never had a successful snapshot is reported with `last_snapshot_ok` false and `last_error` set to `no snapshot yet`.

Schedules:

| Schedule | Default | Keep |
|----------|---------|------|
| hourly | off | 24 |
| daily | on | 7 |
| weekly | on | 4 |
| monthly | on | 12 |
| yearly | on | 3 |

Run one schedule by hand:

```sh
sudo xgbackup snapshot --schedule daily
```

If hourly is still disabled, that command prints `hourly snapshots are disabled` and exits 0.

Free space in the status file is `pool_free_bytes` for the btrfs pool, not the operating-system disk.
