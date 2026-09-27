# Offsite access over VPN

The first setup is on the same network as the clients. The sync client talks SSH to the vault and does not need any other port.

When the vault sits somewhere else (a second site, a small VPS, a friend's house), do not leave SSH open to the whole internet. Put the vault and the clients on a VPN, and point `xgbackup-sync -host` at the vault's VPN address.

A typical WireGuard layout:

1. Install WireGuard on the vault and on each client.
2. Give the vault a tunnel address such as `10.8.0.1` and each client its own address in `10.8.0.0/24`.
3. On the vault firewall, allow TCP 22 only from that tunnel, plus the WireGuard UDP port from the clients.
4. Sync with `-host 10.8.0.1` and a `known_hosts` entry for that address.

OpenVPN or another VPN is the same pattern: the backup channel is still SSH, and the path to it is the tunnel. The vault does not need a public DNS name.

An SSH key for `xgb-demo` can upload into that client's drop. It cannot open a shell, forward ports, or see snapshot directories. Compromising a laptop still cannot delete history. The VPN keeps that SSH port off the public internet.

This development host may have a public address so it can be built and tested. A vault you rely on should follow the firewall rule above.
