VERSION := 0.4.0
STAGE := packaging/out/stage
DESTDIR ?=

.PHONY: all build test install stage deb rpm dist clean

all: build

build:
	CGO_ENABLED=0 go build -o bin/xgbackup ./cmd/xgbackup
	CGO_ENABLED=0 go build -o bin/xgbackup-sync ./cmd/xgbackup-sync

test:
	go test ./...

install: build
	install -d $(DESTDIR)/usr/bin
	install -m 755 bin/xgbackup bin/xgbackup-sync $(DESTDIR)/usr/bin
	install -d $(DESTDIR)/usr/lib/systemd/system
	install -m 644 packaging/systemd/xgbackup-snapshot@.service $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-snapshot-hourly.timer $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-snapshot-daily.timer $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-snapshot-weekly.timer $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-snapshot-monthly.timer $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-snapshot-yearly.timer $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-web.service $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-query.service $(DESTDIR)/usr/lib/systemd/system/
	install -m 644 packaging/systemd/xgbackup-menu.service $(DESTDIR)/usr/lib/systemd/system/
	install -d $(DESTDIR)/usr/share/doc/xgbackup
	install -m 644 README.md LICENSE docs/install.md docs/status.md docs/restore.md docs/vpn-offsite.md docs/hardening.md docs/monitor-contract.md docs/ui.md $(DESTDIR)/usr/share/doc/xgbackup/

stage: build
	rm -rf $(STAGE)
	$(MAKE) install DESTDIR=$(CURDIR)/$(STAGE)

deb: stage
	rm -rf packaging/out/deb
	mkdir -p packaging/out/deb/xgbackup
	cp -a $(STAGE)/. packaging/out/deb/xgbackup/
	mkdir -p packaging/out/deb/xgbackup/DEBIAN
	cp packaging/debian/control packaging/out/deb/xgbackup/DEBIAN/control
	cp packaging/debian/postinst packaging/out/deb/xgbackup/DEBIAN/postinst
	cp packaging/debian/prerm packaging/out/deb/xgbackup/DEBIAN/prerm
	chmod 755 packaging/out/deb/xgbackup/DEBIAN/postinst packaging/out/deb/xgbackup/DEBIAN/prerm
	dpkg-deb --root-owner-group -Zgzip --build packaging/out/deb/xgbackup packaging/out/xgbackup_$(VERSION)-1_amd64.deb
	@echo "Install from a path _apt can read, for example: sudo cp packaging/out/xgbackup_$(VERSION)-1_amd64.deb /tmp/ && sudo apt-get install /tmp/xgbackup_$(VERSION)-1_amd64.deb"

rpm: stage
	rm -rf packaging/rpm/build
	mkdir -p packaging/rpm/build/BUILD packaging/rpm/build/RPMS packaging/rpm/build/SOURCES packaging/rpm/build/SPECS packaging/rpm/build/SRPMS
	cp -a $(STAGE) packaging/rpm/build/STAGE
	rpmbuild -bb packaging/rpm/xgbackup.spec \
		--define "_topdir $(CURDIR)/packaging/rpm/build" \
		--define "stage $(CURDIR)/packaging/rpm/build/STAGE"
	mkdir -p packaging/out
	cp packaging/rpm/build/RPMS/x86_64/xgbackup-$(VERSION)-1.*.rpm packaging/out/

dist: build
	rm -rf dist
	mkdir -p dist
	$(MAKE) client-archive GOOS=linux GOARCH=amd64 EXT=
	$(MAKE) client-archive GOOS=windows GOARCH=amd64 EXT=.exe
	$(MAKE) client-archive GOOS=darwin GOARCH=amd64 EXT=
	$(MAKE) client-archive GOOS=darwin GOARCH=arm64 EXT=

client-archive:
	rm -rf dist/work
	mkdir -p dist/work
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o dist/work/xgbackup-sync$(EXT) ./cmd/xgbackup-sync
	cp packaging/client/README.txt dist/work/README.txt
	tar -C dist/work -czf dist/xgbackup-sync_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz xgbackup-sync$(EXT) README.txt
	rm -rf dist/work

clean:
	rm -rf bin dist packaging/out packaging/rpm/build
