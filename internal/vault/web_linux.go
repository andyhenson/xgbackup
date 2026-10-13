//go:build linux

package vault

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xgbackup/xgbackup/internal/access"
	"github.com/xgbackup/xgbackup/internal/config"
	"golang.org/x/crypto/bcrypt"
)

const (
	deviceCookie  = "xgbackup_device"
	sessionCookie = "xgbackup_session"
	devicePath    = "/var/lib/xgbackup/web-devices.json"
	approvePath   = "/var/lib/xgbackup/web-approve.json"
	restoreRoot   = "/var/lib/xgbackup/restores"
)

type webServer struct {
	cfg      config.Config
	web      config.WebConfig
	mu       sync.Mutex
	sessions map[string]string
	email    map[string]emailCode
}

type emailCode struct {
	Code string
	Exp  time.Time
}

func Web() error {
	if err := requireRoot(); err != nil {
		return err
	}
	cfg, err := config.Load(config.ConfigPath)
	if err != nil {
		return err
	}
	s := &webServer{
		cfg:      cfg,
		web:      cfg.EffectiveWeb(),
		sessions: map[string]string{},
		email:    map[string]emailCode{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serve)
	cert, key := webCertPaths(s.web)
	if err := ensureWebCert(cert, key); err != nil {
		return err
	}
	fmt.Printf("web listening on https://%s/\n", s.web.Listen)
	server := &http.Server{
		Addr:              s.web.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return server.ListenAndServeTLS(cert, key)
}

func ApproveWeb() error {
	if err := requireRoot(); err != nil {
		return err
	}
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	code := hex.EncodeToString(buf)
	body, err := json.Marshal(map[string]any{"code": code, "exp": time.Now().Add(10 * time.Minute).Unix()})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(approvePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(approvePath, append(body, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Printf("approval code %s\n", code)
	fmt.Println("Enter it on the web UI within 10 minutes. That approves the browser cookie.")
	return nil
}

func (s *webServer) serve(w http.ResponseWriter, r *http.Request) {
	if !s.allowIP(r) {
		http.Error(w, "address not allowed", http.StatusForbidden)
		return
	}
	if r.URL.Path == "/approve" {
		s.approve(w, r)
		return
	}
	if s.needs("cookie") && !s.deviceOK(r) {
		s.page(w, "Approve this browser", `<p>This browser needs a cookie approved from the vault console.</p>
<form method="post" action="/approve"><label>Approval code <input name="code"></label>
<button type="submit">Approve</button></form>`)
		return
	}
	if r.URL.Path == "/login" {
		s.login(w, r)
		return
	}
	user, ok := s.sessionUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.URL.Path == "/logout" {
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.URL.Path == "/restore" && r.Method == http.MethodPost {
		s.restore(w, r, user)
		return
	}
	s.browse(w, r, user)
}

func (s *webServer) allowIP(r *http.Request) bool {
	if !s.needs("ip") {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	for _, cidr := range s.web.AllowCIDR {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (s *webServer) needs(name string) bool {
	for _, item := range s.web.Required {
		if item == name {
			return true
		}
	}
	return false
}

func (s *webServer) deviceOK(r *http.Request) bool {
	c, err := r.Cookie(deviceCookie)
	if err != nil || c.Value == "" {
		return false
	}
	tokens, err := loadStringMap(devicePath)
	if err != nil {
		return false
	}
	_, ok := tokens[c.Value]
	return ok
}

func (s *webServer) approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(approvePath)
	if err != nil {
		http.Error(w, "no approval is waiting", http.StatusForbidden)
		return
	}
	var pending struct {
		Code string `json:"code"`
		Exp  int64  `json:"exp"`
	}
	if err := json.Unmarshal(data, &pending); err != nil || pending.Code == "" || time.Now().Unix() > pending.Exp || r.FormValue("code") != pending.Code {
		http.Error(w, "approval code was not accepted", http.StatusForbidden)
		return
	}
	token, err := randomToken()
	if err != nil {
		http.Error(w, "token", http.StatusInternalServerError)
		return
	}
	tokens, _ := loadStringMap(devicePath)
	if tokens == nil {
		tokens = map[string]string{}
	}
	tokens[token] = "approved"
	if err := saveStringMap(devicePath, tokens); err != nil {
		http.Error(w, "save", http.StatusInternalServerError)
		return
	}
	_ = os.Remove(approvePath)
	http.SetCookie(w, &http.Cookie{Name: deviceCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: true})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *webServer) login(w http.ResponseWriter, r *http.Request) {
	msg := ""
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user, err := s.checkLogin(r.FormValue("user"), r.FormValue("password"), r.FormValue("totp"), r.FormValue("email"))
		if err != nil {
			msg = err.Error()
		} else {
			token, err := randomToken()
			if err != nil {
				http.Error(w, "token", http.StatusInternalServerError)
				return
			}
			s.mu.Lock()
			s.sessions[token] = user
			s.mu.Unlock()
			http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: true})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	extra := ""
	if s.needs("totp") {
		extra += `<label>Authenticator code <input name="totp"></label>`
	}
	if s.needs("email") {
		extra += `<label>Email code <input name="email"></label>`
	}
	s.page(w, "Log in", fmt.Sprintf(`<p>%s</p>
<form method="post" action="/login">
<label>User <input name="user"></label>
<label>Password <input name="password" type="password"></label>
%s
<button type="submit">Log in</button>
</form>`, html.EscapeString(msg), extra))
}

func (s *webServer) checkLogin(name, password, totp, email string) (string, error) {
	users, err := loadWebUsers()
	if err != nil {
		return "", err
	}
	var match *webUser
	for i := range users {
		if users[i].Name == name {
			match = &users[i]
		}
	}
	if match == nil || bcrypt.CompareHashAndPassword([]byte(match.Password), []byte(password)) != nil {
		return "", fmt.Errorf("user or password was not accepted")
	}
	if !access.AllowSetup(s.cfg.SetupUsers, name) && !s.restoreUser(name) {
		return "", fmt.Errorf("user has no setup or restore permission")
	}
	if s.needs("totp") && !access.VerifyTOTP(match.TOTP, totp, time.Now()) {
		return "", fmt.Errorf("authenticator code was not accepted")
	}
	if s.needs("email") {
		if err := s.checkEmail(match, email); err != nil {
			return "", err
		}
	}
	return name, nil
}

func (s *webServer) restoreUser(name string) bool {
	for _, rule := range s.cfg.Restore {
		if rule.User == name {
			return true
		}
	}
	return false
}

func (s *webServer) checkEmail(user *webUser, code string) error {
	s.mu.Lock()
	pending, ok := s.email[user.Name]
	s.mu.Unlock()
	if code != "" && ok && time.Now().Before(pending.Exp) && code == pending.Code {
		return nil
	}
	if user.Email == "" {
		return fmt.Errorf("user has no email address")
	}
	next, err := randomToken()
	if err != nil {
		return err
	}
	next = next[:6]
	s.mu.Lock()
	s.email[user.Name] = emailCode{Code: next, Exp: time.Now().Add(10 * time.Minute)}
	s.mu.Unlock()
	msg := "To: " + user.Email + "\nSubject: xgbackup code\n\n" + next + "\n"
	cmd := exec.Command("/usr/sbin/sendmail", "-t")
	cmd.Stdin = strings.NewReader(msg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not send email: %w", err)
	}
	return fmt.Errorf("email code sent")
}

func (s *webServer) sessionUser(r *http.Request) (string, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.sessions[c.Value]
	return user, ok
}

func (s *webServer) browse(w http.ResponseWriter, r *http.Request, user string) {
	q := r.URL.Query()
	snap := q.Get("snapshot")
	client := q.Get("client")
	rel := q.Get("path")
	names, err := Entries(s.cfg.Pool, snap, client, rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var b strings.Builder
	b.WriteString(`<p><a href="/">snapshots</a>`)
	if snap != "" {
		fmt.Fprintf(&b, ` / <a href="/?snapshot=%s">%s</a>`, html.EscapeString(snap), html.EscapeString(snap))
	}
	if client != "" {
		fmt.Fprintf(&b, ` / <a href="/?snapshot=%s&amp;client=%s">%s</a>`, html.EscapeString(snap), html.EscapeString(client), html.EscapeString(client))
	}
	fmt.Fprintf(&b, `</p><p>Signed in as %s. <a href="/logout">Log out</a></p><ul>`, html.EscapeString(user))
	for _, name := range names {
		nextPath := rel
		linkName := name
		if snap == "" {
			fmt.Fprintf(&b, `<li><a href="/?snapshot=%s">%s</a></li>`, html.EscapeString(name), html.EscapeString(name))
			continue
		}
		if client == "" {
			fmt.Fprintf(&b, `<li><a href="/?snapshot=%s&amp;client=%s">%s</a></li>`, html.EscapeString(snap), html.EscapeString(name), html.EscapeString(name))
			continue
		}
		if strings.HasSuffix(name, "/") {
			dir := strings.TrimSuffix(name, "/")
			if nextPath == "" || nextPath == "." {
				nextPath = dir
			} else {
				nextPath = nextPath + "/" + dir
			}
			fmt.Fprintf(&b, `<li><a href="/?snapshot=%s&amp;client=%s&amp;path=%s">%s</a></li>`, html.EscapeString(snap), html.EscapeString(client), html.EscapeString(nextPath), html.EscapeString(linkName))
			continue
		}
		fileRel := name
		if rel != "" && rel != "." {
			fileRel = rel + "/" + name
		}
		if !s.canRestore(user, client, fileRel) {
			fmt.Fprintf(&b, `<li>%s</li>`, html.EscapeString(name))
			continue
		}
		fmt.Fprintf(&b, `<li>%s <form method="post" action="/restore">
<input type="hidden" name="snapshot" value="%s">
<input type="hidden" name="client" value="%s">
<input type="hidden" name="path" value="%s">
<button type="submit">Restore</button></form></li>`, html.EscapeString(name), html.EscapeString(snap), html.EscapeString(client), html.EscapeString(fileRel))
	}
	b.WriteString(`</ul>`)
	if snap != "" && client != "" && s.canRestore(user, client, rel) {
		fmt.Fprintf(&b, `<form method="post" action="/restore">
<input type="hidden" name="snapshot" value="%s">
<input type="hidden" name="client" value="%s">
<input type="hidden" name="path" value="%s">
<button type="submit">Restore this folder</button></form>`, html.EscapeString(snap), html.EscapeString(client), html.EscapeString(blankPath(rel)))
	}
	if snap != "" && access.AllowSetup(s.cfg.SetupUsers, user) {
		fmt.Fprintf(&b, `<form method="post" action="/restore">
<input type="hidden" name="snapshot" value="%s">
<input type="hidden" name="all" value="yes">
<button type="submit">Restore every client</button></form>`, html.EscapeString(snap))
	}
	s.page(w, "XG Backup", b.String())
}

func blankPath(rel string) string {
	if rel == "" {
		return "."
	}
	return rel
}

func (s *webServer) canRestore(user, client, rel string) bool {
	rules := make([]access.RestoreRule, len(s.cfg.Restore))
	for i, rule := range s.cfg.Restore {
		rules[i] = access.RestoreRule{User: rule.User, Client: rule.Client, Prefix: rule.Prefix}
	}
	return access.AllowRestore(s.cfg.SetupUsers, rules, user, client, rel)
}

func (s *webServer) restore(w http.ResponseWriter, r *http.Request, user string) errorWriter {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return nil
	}
	snap := r.FormValue("snapshot")
	dest := filepath.Join(restoreRoot, user, time.Now().UTC().Format("20060102T150405"))
	if r.FormValue("all") == "yes" {
		if !access.AllowSetup(s.cfg.SetupUsers, user) {
			http.Error(w, "setup permission is required to restore every client", http.StatusForbidden)
			return nil
		}
		if err := RestoreAll(s.cfg.Pool, snap, dest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
	} else {
		client := r.FormValue("client")
		rel := r.FormValue("path")
		if !s.canRestore(user, client, rel) {
			http.Error(w, "restore not allowed for this path", http.StatusForbidden)
			return nil
		}
		if err := Restore(s.cfg.Pool, client, snap, rel, filepath.Join(dest, client)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
	}
	s.page(w, "Restored", fmt.Sprintf(`<p>Restored to %s</p><p><a href="/">Back</a></p>`, html.EscapeString(dest)))
	return nil
}

type errorWriter = error

func (s *webServer) page(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html><head><title>%s</title></head><body><h1>%s</h1>%s</body></html>", html.EscapeString(title), html.EscapeString(title), body)
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func loadStringMap(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var out map[string]string
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func saveStringMap(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
