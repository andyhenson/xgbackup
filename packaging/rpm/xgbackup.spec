Name:           xgbackup
Version:        0.4.0
Release:        1%{?dist}
Summary:        Linux backup vault
License:        MIT
BuildArch:      x86_64

%define stage %{_topdir}/STAGE
%define _build_id_links none
%global debug_package %{nil}
%global _missing_build_ids_terminate_build 0

%description
Clients push into per-client drop areas. The vault keeps read-only btrfs
snapshots. Everyday client keys cannot delete snapshot history.
This package targets AlmaLinux and other RPM systems. The vault OS is Linux.

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/lib/systemd/system
mkdir -p %{buildroot}/usr/share/doc/xgbackup
cp -a %{stage}/usr/bin/xgbackup %{buildroot}/usr/bin/xgbackup
cp -a %{stage}/usr/bin/xgbackup-sync %{buildroot}/usr/bin/xgbackup-sync
cp -a %{stage}/usr/lib/systemd/system/. %{buildroot}/usr/lib/systemd/system/
cp -a %{stage}/usr/share/doc/xgbackup/. %{buildroot}/usr/share/doc/xgbackup/

%files
/usr/bin/xgbackup
/usr/bin/xgbackup-sync
/usr/lib/systemd/system/xgbackup-snapshot@.service
/usr/lib/systemd/system/xgbackup-snapshot-hourly.timer
/usr/lib/systemd/system/xgbackup-snapshot-daily.timer
/usr/lib/systemd/system/xgbackup-snapshot-weekly.timer
/usr/lib/systemd/system/xgbackup-snapshot-monthly.timer
/usr/lib/systemd/system/xgbackup-snapshot-yearly.timer
/usr/lib/systemd/system/xgbackup-web.service
/usr/lib/systemd/system/xgbackup-query.service
/usr/lib/systemd/system/xgbackup-menu.service
/usr/share/doc/xgbackup

%post
if [ $1 -eq 1 ] || [ $1 -eq 2 ]; then
    systemctl daemon-reload || true
    systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target || true
    systemctl enable xgbackup-snapshot-daily.timer || true
    systemctl enable xgbackup-snapshot-weekly.timer || true
    systemctl enable xgbackup-snapshot-monthly.timer || true
    systemctl enable xgbackup-snapshot-yearly.timer || true
    getent group xgbackup >/dev/null || groupadd --system xgbackup || true
    id xgbackup >/dev/null 2>&1 || useradd --system --gid xgbackup --home-dir /var/lib/xgbackup --no-create-home --shell /usr/sbin/nologin xgbackup || true
    systemctl enable --now xgbackup-query.service || true
fi

%preun
if [ $1 -eq 0 ]; then
    systemctl disable --now xgbackup-snapshot-hourly.timer || true
    systemctl disable --now xgbackup-snapshot-daily.timer || true
    systemctl disable --now xgbackup-snapshot-weekly.timer || true
    systemctl disable --now xgbackup-snapshot-monthly.timer || true
    systemctl disable --now xgbackup-snapshot-yearly.timer || true
fi
