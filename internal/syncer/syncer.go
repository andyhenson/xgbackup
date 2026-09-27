package syncer

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Options struct {
	Host       string
	Port       int
	User       string
	KeyPath    string
	Src        string
	Dest       string
	KnownHosts string
}

type Result struct {
	Uploaded int
	Skipped  int
	Deleted  int
}

func Run(opt Options) (Result, error) {
	if opt.Port == 0 {
		opt.Port = 22
	}
	if opt.Dest == "" {
		opt.Dest = "/drop"
	}
	if opt.Host == "" || opt.User == "" || opt.KeyPath == "" || opt.Src == "" {
		return Result{}, fmt.Errorf("host, user, key, and src are required")
	}
	info, err := os.Stat(opt.Src)
	if err != nil {
		return Result{}, err
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("src is not a directory")
	}

	signer, err := readKey(opt.KeyPath)
	if err != nil {
		return Result{}, err
	}
	kh, err := knownhosts.New(opt.KnownHosts)
	if err != nil {
		return Result{}, fmt.Errorf("known hosts: %w", err)
	}
	cfg := &ssh.ClientConfig{
		User:            opt.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: kh,
		Timeout:         15 * time.Second,
	}
	addr := net.JoinHostPort(opt.Host, fmt.Sprintf("%d", opt.Port))
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return Result{}, Classify(err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return Result{}, Classify(err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	sclient, err := sftp.NewClient(client)
	if err != nil {
		return Result{}, Classify(err)
	}
	defer sclient.Close()

	if _, err := sclient.Stat(opt.Dest); err != nil {
		return Result{}, fmt.Errorf("remote dest %s: %w", opt.Dest, err)
	}

	wantFiles := map[string]struct{}{}
	wantDirs := map[string]struct{}{path.Clean(opt.Dest): {}}
	var result Result

	err = filepath.WalkDir(opt.Src, func(local string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(opt.Src, local)
		if err != nil {
			return err
		}
		remote, err := remoteJoin(opt.Dest, filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			fmt.Fprintf(os.Stderr, "skipped symlink: %s\n", local)
			return nil
		}
		if d.IsDir() {
			wantDirs[remote] = struct{}{}
			if remote != path.Clean(opt.Dest) {
				if err := sclient.MkdirAll(remote); err != nil {
					return Classify(err)
				}
			}
			return nil
		}
		if !d.Type().IsRegular() {
			fmt.Fprintf(os.Stderr, "skipped non-regular file: %s\n", local)
			return nil
		}
		wantFiles[remote] = struct{}{}
		parent := path.Dir(remote)
		wantDirs[parent] = struct{}{}
		uploaded, err := upload(sclient, local, remote)
		if err != nil {
			return Classify(err)
		}
		if uploaded {
			result.Uploaded++
		} else {
			result.Skipped++
		}
		return nil
	})
	if err != nil {
		return result, err
	}

	deleted, err := deleteExtras(sclient, opt.Dest, wantFiles, wantDirs)
	if err != nil {
		return result, Classify(err)
	}
	result.Deleted = deleted
	return result, nil
}

func readKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}
	return signer, nil
}

func upload(c *sftp.Client, local, remote string) (bool, error) {
	st, err := os.Stat(local)
	if err != nil {
		return false, err
	}
	if remoteInfo, err := c.Stat(remote); err == nil {
		delta := remoteInfo.ModTime().Sub(st.ModTime())
		if delta < 0 {
			delta = -delta
		}
		if remoteInfo.Size() == st.Size() && delta < 2*time.Second {
			return false, nil
		}
	}
	in, err := os.Open(local)
	if err != nil {
		return false, err
	}
	defer in.Close()
	out, err := c.OpenFile(remote, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return false, err
	}
	if err := out.Close(); err != nil {
		return false, err
	}
	if err := c.Chtimes(remote, st.ModTime(), st.ModTime()); err != nil {
		return false, err
	}
	return true, nil
}

func deleteExtras(c *sftp.Client, dest string, files, dirs map[string]struct{}) (int, error) {
	var fileList []string
	var dirList []string
	walker := c.Walk(dest)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return 0, err
		}
		p := path.Clean(walker.Path())
		if walker.Stat().IsDir() {
			dirList = append(dirList, p)
			continue
		}
		fileList = append(fileList, p)
	}
	deleted := 0
	for _, p := range fileList {
		if _, ok := files[p]; ok {
			continue
		}
		if err := c.Remove(p); err != nil {
			return deleted, err
		}
		deleted++
	}
	for i := len(dirList) - 1; i >= 0; i-- {
		p := dirList[i]
		if p == path.Clean(dest) {
			continue
		}
		if _, ok := dirs[p]; ok {
			continue
		}
		if err := c.RemoveDirectory(p); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
