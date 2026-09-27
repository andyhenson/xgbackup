package syncer

import (
	"fmt"
	"path"
	"strings"
)

func remoteJoin(dest, rel string) (string, error) {
	dest = path.Clean(dest)
	if !path.IsAbs(dest) {
		return "", fmt.Errorf("dest must be an absolute path")
	}
	rel = path.Clean(strings.TrimPrefix(rel, "/"))
	if rel == ".." || strings.HasPrefix(rel, "../") || rel == "." && dest == "" {
		return "", fmt.Errorf("invalid relative path")
	}
	if rel == "." {
		return dest, nil
	}
	full := path.Join(dest, rel)
	if full != dest && !strings.HasPrefix(full, dest+"/") {
		return "", fmt.Errorf("path escapes dest")
	}
	return full, nil
}
