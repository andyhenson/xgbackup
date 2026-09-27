# Status file for a future monitor

XG Monitor is not part of this repository. The vault writes a local JSON file that a monitor, or any other watcher, can read. Nothing here phones home.

Path: `/var/lib/xgbackup/status.json`

Updated by `xgbackup status` and by every `xgbackup snapshot` run.

```json
{
  "vault_id": "6b9c1c0e-3e1a-4f2b-9a1d-0c5e8a7b6d11",
  "time": "2026-09-27T17:00:00Z",
  "pool_free_bytes": 19327352832,
  "pool_total_bytes": 21474836480,
  "clients": [
    {
      "name": "xgb-demo",
      "last_snapshot": "daily-20260927T170000.123456789Z",
      "last_snapshot_time": "2026-09-27T17:00:01Z",
      "last_snapshot_ok": true,
      "last_error": ""
    }
  ]
}
```

Required fields:

- `vault_id` — stable id created by `init`
- `time` — RFC3339 UTC time when this file was written
- `pool_free_bytes` — bytes available on the btrfs pool
- `clients` — one object per client
- `clients[].name`
- `clients[].last_snapshot` — snapshot directory name, or empty when none exist
- `clients[].last_snapshot_ok` — false when the last attempt failed or no snapshot exists yet
- `clients[].last_error` — empty on success; `no snapshot yet` before the first snapshot

Additional fields a reader can ignore:

- `pool_total_bytes`
- `clients[].last_snapshot_time` — RFC3339 UTC time of the last recorded attempt

A quiet failure is `last_snapshot_ok: false` or a `time` that stops advancing. Low space is a falling `pool_free_bytes`. This file is the whole contract.
