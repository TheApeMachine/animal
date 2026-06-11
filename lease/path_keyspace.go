package lease

import (
	"fmt"
	"path/filepath"
	"strings"
)

/*
PathKeySpace applies workspace-style path normalization and prefix matching.
*/
type PathKeySpace struct{}

/*
Normalize cleans a workspace-relative path for lease lookup.
*/
func (pathKeySpace PathKeySpace) Normalize(key string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(key))))
	clean = strings.TrimPrefix(clean, "./")

	if clean == "" {
		return "", fmt.Errorf("lease: path is required")
	}

	return clean, nil
}

/*
Covers reports whether prefix contains key in the path-prefix sense.
*/
func (pathKeySpace PathKeySpace) Covers(prefix, key string) bool {
	normalizedPrefix, err := pathKeySpace.Normalize(prefix)
	if err != nil {
		return false
	}

	normalizedKey, err := pathKeySpace.Normalize(key)
	if err != nil {
		return false
	}

	return strings.HasPrefix(normalizedKey, normalizedPrefix)
}
