package retain

import (
	"regexp"
	"sort"
)

var stamp = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}(\.[0-9]{9})?Z$`)

// PruneNames returns the oldest names to delete so that keep names remain.
// Names must sort chronologically, which the snapshot timestamp format does.
func PruneNames(names []string, keep int) []string {
	if keep < 1 {
		keep = 1
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if len(sorted) <= keep {
		return nil
	}
	return sorted[:len(sorted)-keep]
}

// FilterPrefix keeps names that start with schedule plus a hyphen.
func FilterPrefix(names []string, schedule string) []string {
	prefix := schedule + "-"
	var out []string
	for _, name := range names {
		if len(name) <= len(prefix) || name[:len(prefix)] != prefix {
			continue
		}
		if stamp.MatchString(name[len(prefix):]) {
			out = append(out, name)
		}
	}
	return out
}
