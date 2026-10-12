package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xgbackup/xgbackup/internal/syncer"
	"github.com/xgbackup/xgbackup/internal/version"
)

func main() {
	host := flag.String("host", "", "vault hostname or IP")
	port := flag.Int("port", 0, "vault SSH port")
	user := flag.String("user", "", "client user, for example xgb-demo")
	key := flag.String("key", "", "path to the client private key")
	src := flag.String("src", "", "local directory to push")
	dest := flag.String("dest", "/drop", "remote directory inside the client chroot")
	known := flag.String("known-hosts", "", "known_hosts file (default: ~/.xgbackup/known_hosts)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.Version)
		return
	}

	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		set[f.Name] = true
	})

	dir, err := syncer.ProfileDir()
	if err != nil {
		fail(err)
	}
	prof, err := syncer.LoadProfile(dir)
	if err != nil {
		fail(err)
	}

	if set["host"] && set["user"] && set["key"] {
		savePort := *port
		if savePort == 0 {
			savePort = 22
		}
		if err := syncer.SaveProfile(dir, syncer.Profile{
			Host: *host,
			Port: savePort,
			User: *user,
			Key:  *key,
		}); err != nil {
			fail(err)
		}
		if set["known-hosts"] {
			if err := syncer.CopyKnownHosts(dir, *known); err != nil {
				fail(err)
			}
		}
	}

	useHost := *host
	if useHost == "" {
		useHost = prof.Host
	}
	usePort := *port
	if usePort == 0 {
		usePort = prof.Port
	}
	if usePort == 0 {
		usePort = 22
	}
	useUser := *user
	if useUser == "" {
		useUser = prof.User
	}
	useKey := *key
	if useKey == "" {
		useKey = prof.Key
	}
	knownHosts := *known
	if !set["known-hosts"] {
		knownHosts = syncer.KnownHostsPath(dir)
	}
	if useHost == "" || useUser == "" || useKey == "" || *src == "" {
		fail(fmt.Errorf("host, user, key, and src are required (saved in %s)", dir))
	}

	result, err := syncer.Run(syncer.Options{
		Host:       useHost,
		Port:       usePort,
		User:       useUser,
		KeyPath:    useKey,
		Src:        *src,
		Dest:       *dest,
		KnownHosts: knownHosts,
	})
	if err != nil {
		fail(err)
	}
	fmt.Printf("uploaded=%d identical=%d deleted=%d\n", result.Uploaded, result.Identical, result.Deleted)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(syncer.ExitCode(err))
}
