xgbackup-sync pushes a local directory into your drop area on an XG Backup vault.

Example:
  xgbackup-sync -host VAULT -user xgb-demo -key id_ed25519 -src C:\Users\me\Documents -known-hosts known_hosts

The remote dest defaults to /drop, which is the only directory the client can write.
Create known_hosts with: ssh-keyscan -H VAULT

Exit codes:
  0 success
  2 unreachable
  3 auth failed
  4 disk full
  1 other error

A failed push does not delete read-only snapshots already taken on the vault.
Offsite vaults should be reached through a VPN. See the vault docs/vpn-offsite.md.
