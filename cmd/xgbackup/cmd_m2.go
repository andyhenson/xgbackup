//go:build linux

package main

import (
	"flag"
	"fmt"

	"github.com/xgbackup/xgbackup/internal/config"
	"github.com/xgbackup/xgbackup/internal/vault"
)

func cmdLs(args []string) error {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	snapshot := fs.String("snapshot", "", "snapshot name")
	client := fs.String("client", "", "client name")
	rel := fs.String("path", "", "path inside the client snapshot")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	return vault.List(cfg.Pool, *snapshot, *client, *rel)
}

func cmdMenu() error {
	return vault.Menu()
}

func cmdQuery() error {
	return vault.Query()
}

func cmdAccess(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("access requires list, setup, or restore")
	}
	switch args[0] {
	case "list":
		return vault.ListAccess()
	case "setup":
		if len(args) != 3 || args[1] != "add" {
			return fmt.Errorf("access setup add USER")
		}
		return vault.AddSetupUser(args[2])
	case "restore":
		if len(args) < 2 || args[1] != "add" {
			return fmt.Errorf("access restore add USER [--client NAME] [--prefix PATH]")
		}
		fs := flag.NewFlagSet("access restore", flag.ContinueOnError)
		client := fs.String("client", "*", "client name, or * for every client")
		prefix := fs.String("prefix", "", "path prefix inside that client")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("access restore add USER")
		}
		return vault.AddRestoreUser(fs.Arg(0), *client, *prefix)
	default:
		return fmt.Errorf("unknown access command %s", args[0])
	}
}

func cmdWeb(args []string) error {
	if len(args) == 0 {
		return vault.Web()
	}
	switch args[0] {
	case "approve":
		return vault.ApproveWeb()
	case "totp":
		if len(args) != 2 {
			return fmt.Errorf("web totp USER")
		}
		return vault.SetWebTOTP(args[1])
	case "user":
		if len(args) < 3 || args[1] != "add" {
			return fmt.Errorf("web user add USER [--email ADDRESS]")
		}
		fs := flag.NewFlagSet("web user", flag.ContinueOnError)
		email := fs.String("email", "", "address used when email is a required check")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("web user add USER")
		}
		return vault.AddWebUser(fs.Arg(0), *email)
	default:
		return fmt.Errorf("unknown web command %s", args[0])
	}
}

func cmdSmb(args []string) error {
	if len(args) != 1 || args[0] != "sync" {
		return fmt.Errorf("smb sync")
	}
	return vault.SyncSamba()
}
