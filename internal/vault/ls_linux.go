//go:build linux

package vault

import (
	"fmt"
	"os"
	"sort"
)

// Entries lists snapshot names, clients in a snapshot, or one directory.
// rel empty or "." lists the client root. A trailing slash is added to directories.
func Entries(pool, snapshot, client, rel string) ([]string, error) {
	if snapshot == "" {
		names, err := snapshotNames(snapsRoot(pool))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		return names, nil
	}
	if err := validateSnapshot(snapshot); err != nil {
		return nil, err
	}
	if client == "" {
		entries, err := os.ReadDir(snapshotTree(pool, snapshot))
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() && validateClient(e.Name()) == nil {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		return names, nil
	}
	src, err := snapshotPath(pool, client, snapshot, rel)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{rel}, nil
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func List(pool, snapshot, client, rel string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	names, err := Entries(pool, snapshot, client, rel)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		if snapshot == "" {
			fmt.Println("no snapshots")
		} else if client == "" {
			fmt.Println("no clients")
		} else {
			fmt.Println("(empty)")
		}
		return nil
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}
