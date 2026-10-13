//go:build linux

package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/xgbackup/xgbackup/internal/access"
	"github.com/xgbackup/xgbackup/internal/config"
)

const querySock = "/run/xgbackup/query.sock"

type connKey struct{}

type webSetup struct {
	Listen    string   `json:"listen"`
	Required  []string `json:"required"`
	AllowCIDR []string `json:"allow_cidr"`
	User      string   `json:"user"`
	Password  string   `json:"password"`
	Email     string   `json:"email"`
}

// Query serves local read-only status and listings, and web setup.
// It runs as root so people can use the menu without becoming root.
func Query() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := os.MkdirAll("/run/xgbackup", 0o755); err != nil {
		return err
	}
	_ = os.Remove(querySock)
	ln, err := net.Listen("unix", querySock)
	if err != nil {
		return err
	}
	if err := os.Chmod(querySock, 0o666); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/ls", handleLs)
	mux.HandleFunc("/web/setup", handleWebSetup)
	server := &http.Server{
		Handler: mux,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connKey{}, c)
		},
	}
	return server.Serve(ln)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	text, err := statusText(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, text)
}

func handleLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	names, err := Entries(cfg.Pool, r.URL.Query().Get("snapshot"), r.URL.Query().Get("client"), r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if names == nil {
		names = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Names []string `json:"names"`
	}{Names: names})
}

func handleWebSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	actor, err := peerName(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	var setup webSetup
	if err := json.NewDecoder(r.Body).Decode(&setup); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := applyWebSetup(actor, setup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "ok\n")
}

func peerName(r *http.Request) (string, error) {
	c, _ := r.Context().Value(connKey{}).(net.Conn)
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return "", fmt.Errorf("not a local connection")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return "", err
	}
	var cred *syscall.Ucred
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		cred, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return "", err
	}
	if controlErr != nil {
		return "", controlErr
	}
	u, err := user.LookupId(fmt.Sprint(cred.Uid))
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

func applyWebSetup(actor string, setup webSetup) error {
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	if !access.AllowConfigure(cfg.SetupUsers, actor, webConfigured(cfg.Web)) {
		return fmt.Errorf("%s cannot configure the web UI", actor)
	}
	listen := strings.TrimSpace(setup.Listen)
	if listen == "" {
		listen = "0.0.0.0:8787"
	}
	if _, _, err := net.SplitHostPort(listen); err != nil {
		return fmt.Errorf("listen address must be host:port")
	}
	required, err := normalChecks(setup.Required)
	if err != nil {
		return err
	}
	var cidrs []string
	for _, item := range setup.AllowCIDR {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, err := netip.ParsePrefix(item); err != nil {
			return fmt.Errorf("bad network %s", item)
		}
		cidrs = append(cidrs, item)
	}
	needs := map[string]bool{}
	for _, name := range required {
		needs[name] = true
	}
	if needs["ip"] && len(cidrs) == 0 {
		return fmt.Errorf("an IP check needs at least one network")
	}
	if setup.User == "" || setup.Password == "" {
		return fmt.Errorf("a web user and password are required")
	}
	if needs["email"] && strings.TrimSpace(setup.Email) == "" {
		return fmt.Errorf("an email check needs an address")
	}
	cfg.Web.Listen = listen
	cfg.Web.Required = required
	cfg.Web.AllowCIDR = cidrs
	addSetupUser(&cfg, actor)
	addSetupUser(&cfg, setup.User)
	if err := config.Save(config.ConfigPath, cfg); err != nil {
		return err
	}
	cert, key := webCertPaths(cfg.Web)
	if err := ensureWebCert(cert, key); err != nil {
		return err
	}
	if err := setWebPassword(setup.User, setup.Email, setup.Password); err != nil {
		return err
	}
	if err := enableUnit("xgbackup-web.service", true); err != nil {
		return err
	}
	return enableUnit("xgbackup-menu.service", false)
}

func webConfigured(w config.WebConfig) bool {
	return strings.TrimSpace(w.Listen) != ""
}

func normalChecks(in []string) ([]string, error) {
	known := map[string]bool{"ip": true, "cookie": true, "password": true, "totp": true, "email": true}
	var out []string
	seen := map[string]bool{}
	for _, name := range in {
		name = strings.Trim(strings.TrimSpace(name), `"`)
		if name == "" || seen[name] {
			continue
		}
		if !known[name] {
			return nil, fmt.Errorf("unknown check %s", name)
		}
		seen[name] = true
		out = append(out, name)
	}
	if len(out) == 0 {
		out = []string{"password"}
	}
	return out, nil
}

func addSetupUser(cfg *config.Config, name string) {
	if name == "" || name == "root" || strings.HasPrefix(name, "xgb-") {
		return
	}
	for _, have := range cfg.SetupUsers {
		if have == name {
			return
		}
	}
	cfg.SetupUsers = append(cfg.SetupUsers, name)
}

func enableUnit(name string, restart bool) error {
	if out, err := exec.Command("systemctl", "enable", name).CombinedOutput(); err != nil {
		return fmt.Errorf("enable %s: %s", name, strings.TrimSpace(string(out)))
	}
	if restart {
		if out, err := exec.Command("systemctl", "restart", name).CombinedOutput(); err != nil {
			return fmt.Errorf("start %s: %s", name, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if exec.Command("systemctl", "is-active", "--quiet", name).Run() == nil {
		return nil
	}
	if out, err := exec.Command("systemctl", "start", name).CombinedOutput(); err != nil {
		return fmt.Errorf("start %s: %s", name, strings.TrimSpace(string(out)))
	}
	return nil
}

func queryJSON(method, path string, body any, dest any) error {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, "http://xgbackup"+path, r)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", querySock)
		},
	}}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach the vault query service")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("%s", msg)
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(data, dest)
}

func queryNames(snapshot, client, rel string) ([]string, error) {
	q := "/ls?" + url.Values{"snapshot": {snapshot}, "client": {client}, "path": {rel}}.Encode()
	var out struct {
		Names []string `json:"names"`
	}
	if err := queryJSON(http.MethodGet, q, nil, &out); err != nil {
		return nil, err
	}
	return out.Names, nil
}

func queryStatusText() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://xgbackup/status", nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", querySock)
		},
	}}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach the vault query service")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(data)))
	}
	return string(data), nil
}
