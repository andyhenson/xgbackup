#!/bin/sh
# Proves the demo client can push, cannot delete snapshots, and that a file
# restores from a snapshot after the live drop has changed.
set -eu

KEY=/etc/xgbackup/clients/xgb-demo/id_ed25519
POOL=/var/lib/xgbackup/pool/clients/xgb-demo
DROP=$POOL/jail/drop
SNAPS=$POOL/snaps
KNOWN=/tmp/xg-known-hosts
SRC=/tmp/xg-src

if [ "$(id -u)" -ne 0 ]; then
	echo "run as root" >&2
	exit 1
fi

mode=$(stat -c %a "$KEY")
test "$mode" = "600"

if groups xgb-demo | grep -q '\bsudo\b'; then
	echo "xgb-demo is in the sudo group" >&2
	exit 1
fi

rm -rf "$SRC" /tmp/xg-badkey /tmp/xg-badkey.pub
mkdir -p "$SRC/nested"
printf 'vault-drill\n' > "$SRC/nested/sample.txt"
printf 'extra\n' > "$SRC/extra.txt"
ssh-keyscan -H 127.0.0.1 > "$KNOWN"

sync_ok() {
	xgbackup-sync -host 127.0.0.1 -user xgb-demo -key "$KEY" -src "$SRC" -known-hosts "$KNOWN"
}

out=$(sync_ok)
printf '%s\n' "$out" | grep -q 'uploaded=[1-9]'
test -f "$DROP/nested/sample.txt"
out=$(sync_ok)
printf '%s\n' "$out" | grep -q 'skipped=[1-9]'

rm -f "$SRC/extra.txt"
out=$(sync_ok)
printf '%s\n' "$out" | grep -q 'deleted=[1-9]'
test ! -f "$DROP/extra.txt"

xgbackup snapshot --schedule daily
snap=$(ls "$SNAPS" | grep '^daily-' | sort | tail -n 1)
test -n "$snap"
prop=$(btrfs property get "$SNAPS/$snap" ro)
printf '%s\n' "$prop" | grep -q 'ro=true'
if sh -c 'printf "%s\n" nope > "$1"' sh "$SNAPS/$snap/nested/sample.txt" >/dev/null 2>&1; then
	echo "snapshot is writable" >&2
	exit 1
fi

printf 'changed-later\n' > "$DROP/nested/sample.txt"
xgbackup restore --client xgb-demo --snapshot "$snap" --path nested/sample.txt --dest /tmp/restored-sample.txt
grep -q 'vault-drill' /tmp/restored-sample.txt
grep -q 'changed-later' "$DROP/nested/sample.txt"

cat > /tmp/xg-sftp.batch <<EOF
rm /snaps/$snap/nested/sample.txt
rm /drop/../../snaps/$snap/nested/sample.txt
EOF
sftp -b /tmp/xg-sftp.batch -i "$KEY" -o BatchMode=yes -o IdentitiesOnly=yes \
	-o StrictHostKeyChecking=yes -o UserKnownHostsFile="$KNOWN" \
	xgb-demo@127.0.0.1 >/tmp/xg-sftp.out 2>&1 || true
test -f "$SNAPS/$snap/nested/sample.txt"
if grep -q 'vault-drill' "$SNAPS/$snap/nested/sample.txt"; then
	:
else
	echo "snapshot file was damaged" >&2
	exit 1
fi

if ssh -i "$KEY" -o BatchMode=yes -o IdentitiesOnly=yes \
	-o StrictHostKeyChecking=yes -o UserKnownHostsFile="$KNOWN" \
	xgb-demo@127.0.0.1 "rm -rf $SNAPS/$snap"; then
	echo "client opened a shell" >&2
	exit 1
fi
test -d "$SNAPS/$snap"

ssh-keygen -t ed25519 -N "" -f /tmp/xg-badkey -q
set +e
xgbackup-sync -host 127.0.0.1 -user xgb-demo -key /tmp/xg-badkey -src "$SRC" -known-hosts "$KNOWN" >/tmp/xg-auth.out 2>&1
auth_code=$?
xgbackup-sync -host 127.0.0.1 -port 1 -user xgb-demo -key "$KEY" -src "$SRC" -known-hosts "$KNOWN" >/tmp/xg-unreach.out 2>&1
unreach_code=$?
set -e
test "$auth_code" = "3"
grep -q 'auth failed' /tmp/xg-auth.out
test "$unreach_code" = "2"
grep -q 'unreachable' /tmp/xg-unreach.out

xgbackup snapshot --schedule hourly | grep -q 'disabled'

i=0
while [ "$i" -lt 8 ]; do
	xgbackup snapshot --schedule daily >/dev/null
	i=$((i + 1))
done
count=$(ls "$SNAPS" | grep -c '^daily-' || true)
test "$count" -le 7
test "$count" -ge 1

test -f /var/lib/xgbackup/status.json
grep -q '"vault_id"' /var/lib/xgbackup/status.json
grep -q '"pool_free_bytes"' /var/lib/xgbackup/status.json
grep -q '"last_snapshot_ok"' /var/lib/xgbackup/status.json

if systemctl is-enabled xgbackup-snapshot-hourly.timer >/dev/null 2>&1; then
	echo "hourly timer should stay disabled" >&2
	exit 1
fi
systemctl is-enabled xgbackup-snapshot-daily.timer >/dev/null
systemctl is-enabled xgbackup-snapshot-weekly.timer >/dev/null
systemctl is-enabled xgbackup-snapshot-monthly.timer >/dev/null
systemctl is-enabled xgbackup-snapshot-yearly.timer >/dev/null

echo "acceptance ok"
