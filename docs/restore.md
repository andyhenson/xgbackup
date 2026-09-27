# Restore a file from a snapshot

Snapshots live under `/var/lib/xgbackup/pool/clients/<client>/snaps/` and are btrfs read-only subvolumes. Restoring is a root action on the vault. Client keys cannot read this directory.

List snapshot names:

```sh
sudo ls /var/lib/xgbackup/pool/clients/xgb-demo/snaps
```

Copy one file out. `--path` is relative to the snapshot root (the drop contents at that time):

```sh
sudo xgbackup restore \
  --client xgb-demo \
  --snapshot daily-20260927T170000Z \
  --path nested/sample.txt \
  --dest /tmp/sample.txt
```

Check the copy, then put it back on the workstation yourself. The restore command does not write into the live drop, so it does not change what the next snapshot will see.

## Drill

On a vault that already has `xgb-demo`:

```sh
sudo mkdir -p /tmp/xg-restore-src
echo vault-drill > /tmp/xg-restore-src/sample.txt
sudo xgbackup-sync -host 127.0.0.1 -user xgb-demo \
  -key /etc/xgbackup/clients/xgb-demo/id_ed25519 \
  -src /tmp/xg-restore-src -known-hosts /root/.ssh/known_hosts
sudo xgbackup snapshot --schedule daily
snap=$(sudo ls /var/lib/xgbackup/pool/clients/xgb-demo/snaps | grep '^daily-' | sort | tail -1)
echo changed-later | sudo tee /var/lib/xgbackup/pool/clients/xgb-demo/jail/drop/sample.txt
sudo xgbackup restore --client xgb-demo --snapshot "$snap" --path sample.txt --dest /tmp/restored-sample.txt
sudo grep -q vault-drill /tmp/restored-sample.txt
```

`/tmp/restored-sample.txt` contains `vault-drill`. The live drop contains `changed-later`. That is the snapshot boundary.

`tests/acceptance.sh` runs this drill, plus the check that the demo client cannot delete the snapshot.
