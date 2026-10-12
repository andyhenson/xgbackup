xgbackup-sync pushes a local directory into your drop area on an XG Backup vault.

The first run can store the vault address, user, key, and known_hosts in %USERPROFILE%\.xgbackup (Linux: ~/.xgbackup). Later runs only need the source directory.

Example:
  ssh-keyscan -H VAULT > %USERPROFILE%\.xgbackup\known_hosts
  xgbackup-sync -host VAULT -user xgb-demo -key xgb-demo.key -src C:\Users\me\Documents -known-hosts %USERPROFILE%\.xgbackup\known_hosts
  xgbackup-sync -src C:\Users\me\Documents

The remote dest defaults to /drop, which is the only directory the client can write.
Unchanged files are reported as identical.

Exit codes:
  0 success
  2 unreachable
  3 auth failed
  4 disk full
  1 other error

A failed push does not delete read-only snapshots already taken on the vault.
Offsite vaults should be reached through a VPN. See the vault docs/vpn-offsite.md.
