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
	port := flag.Int("port", 22, "vault SSH port")
	user := flag.String("user", "", "client user, for example xgb-demo")
	key := flag.String("key", "", "path to the client private key")
	src := flag.String("src", "", "local directory to push")
	dest := flag.String("dest", "/drop", "remote directory inside the client chroot")
	known := flag.String("known-hosts", "", "known_hosts file (default: $HOME/.ssh/known_hosts)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.Version)
		return
	}
	knownHosts := *known
	if knownHosts == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fail(err)
		}
		knownHosts = home + "/.ssh/known_hosts"
	}
	result, err := syncer.Run(syncer.Options{
		Host:       *host,
		Port:       *port,
		User:       *user,
		KeyPath:    *key,
		Src:        *src,
		Dest:       *dest,
		KnownHosts: knownHosts,
	})
	if err != nil {
		fail(err)
	}
	fmt.Printf("uploaded=%d skipped=%d deleted=%d\n", result.Uploaded, result.Skipped, result.Deleted)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(syncer.ExitCode(err))
}
