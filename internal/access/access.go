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
