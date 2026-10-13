package access

import "strings"

// AllowSetup reports whether user may configure the vault.
// Root is always allowed. Backup client accounts are not setup users.
func AllowSetup(setupUsers []string, user string) bool {
	if user == "root" {
		return true
	}
	for _, name := range setupUsers {
		if name == user {
			return true
		}
	}
	return false
}

// AllowConfigure reports whether a local user may change web settings.
// Backup accounts cannot. Before the web UI is configured, any other local
// user may do it once. After that, only setup users and the console account.
func AllowConfigure(setupUsers []string, user string, webConfigured bool) bool {
	if user == "" || strings.HasPrefix(user, "xgb-") {
		return false
	}
	if user == "root" {
		return true
	}
	if user == "xgbackup" {
		return true
	}
	for _, name := range setupUsers {
		if name == user {
			return true
		}
	}
	return !webConfigured
}

// RestoreRule is who may restore which part of a snapshot.
// Client "*" means every client. An empty prefix means the whole client tree.
type RestoreRule struct {
	User   string
	Client string
	Prefix string
}

// AllowRestore reports whether user may restore rel from client.
// Setup users may restore everything. Prefix matches the path or anything under it.
func AllowRestore(setupUsers []string, rules []RestoreRule, user, client, rel string) bool {
	if AllowSetup(setupUsers, user) {
		return true
	}
	rel = strings.TrimPrefix(rel, "/")
	for _, rule := range rules {
		if rule.User != user {
			continue
		}
		if rule.Client != "*" && rule.Client != client {
			continue
		}
		prefix := strings.Trim(rule.Prefix, "/")
		if prefix == "" || rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return false
}
