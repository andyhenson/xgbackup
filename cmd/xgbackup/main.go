//go:build linux

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xgbackup/xgbackup/internal/config"
	"github.com/xgbackup/xgbackup/internal/vault"
	"github.com/xgbackup/xgbackup/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "version", "-version", "--version":
		fmt.Println(version.Version)
		return
	case "init":
		err = cmdInit(os.Args[2:])
	case "client":
		err = cmdClient(os.Args[2:])
	case "snapshot":
		err = cmdSnapshot(os.Args[2:])
	case "status":
		err = cmdStatus()
	case "restore":
		err = cmdRestore(os.Args[2:])
	case "ls":
		err = cmdLs(os.Args[2:])
	case "menu":
		err = cmdMenu()
	case "access":
		err = cmdAccess(os.Args[2:])
	case "web":
		err = cmdWeb(os.Args[2:])
	case "query":
		err = cmdQuery()
	case "smb":
		err = cmdSmb(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "xgbackup: %s\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `xgbackup %s

Usage:
  xgbackup init --device /dev/nvme1n1
  xgbackup client add xgb-demo
  xgbackup client list
  xgbackup snapshot --schedule daily
  xgbackup status
  xgbackup ls [--snapshot NAME] [--client NAME] [--path PATH]
  xgbackup restore --client xgb-demo --snapshot NAME --path FILE --dest PATH
  xgbackup restore --all --snapshot NAME --dest DIR
  xgbackup menu
  xgbackup access list
  xgbackup web
  xgbackup smb sync
`, version.Version)
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	device := fs.String("device", "", "block device for a new btrfs pool")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *device == "" {
		return fmt.Errorf("init requires --device")
	}
	return vault.Init(*device)
}

func cmdClient(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("client requires add or list")
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		names, err := vault.ListClients(cfg.Pool)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Println("no clients")
			return nil
		}
		for _, name := range names {
			fmt.Println(name)
		}
		return nil
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("client add NAME")
		}
		key, err := vault.AddClient(cfg.Pool, args[1])
		if err != nil {
			return err
		}
		fmt.Printf("client %s\nkey %s\n", args[1], key)
		return nil
	default:
		return fmt.Errorf("unknown client command %s", args[0])
	}
}

func cmdSnapshot(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	schedule := fs.String("schedule", "", "hourly, daily, weekly, monthly, or yearly")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *schedule == "" {
		return fmt.Errorf("snapshot requires --schedule")
	}
	return vault.Snapshot(*schedule)
}

func cmdStatus() error {
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	return vault.PrintStatus(cfg)
}

func cmdRestore(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	client := fs.String("client", "", "client name")
	snapshot := fs.String("snapshot", "", "snapshot name")
	rel := fs.String("path", "", "path inside the snapshot; . restores the whole client")
	dest := fs.String("dest", "", "local destination path")
	all := fs.Bool("all", false, "restore every client in the snapshot")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *snapshot == "" || *dest == "" {
		return fmt.Errorf("restore requires --snapshot and --dest")
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	if *all {
		return vault.RestoreAll(cfg.Pool, *snapshot, *dest)
	}
	if *client == "" || *rel == "" {
		return fmt.Errorf("restore requires --client and --path, or --all")
	}
	return vault.Restore(cfg.Pool, *client, *snapshot, *rel, *dest)
}
