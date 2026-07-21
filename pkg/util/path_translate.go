package utils

import "strings"

// TranslatePath replaces a container path prefix with the real host path. Used
// when a download client (e.g. Deluge in Docker) reports file paths from inside
// its container that differ from the host filesystem. Returns path unchanged when
// no translation is configured or the prefix does not match.
func TranslatePath(path, delugePath, hostPath string) string {
	if delugePath == "" || hostPath == "" || !strings.HasPrefix(path, delugePath) {
		return path
	}
	return hostPath + path[len(delugePath):]
}
