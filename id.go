package kanvas

import (
	"path/filepath"
	"strings"
)

// SiblingID calculates the ID of a component relative to the parent of the caller.
// If the caller is empty, the component is resolved relative to the root.
// If the caller is non-empty,
// the component is resolved relative to the parent of the caller.
func SiblingID(caller, component string) string {
	if component == "" {
		return ""
	}

	if component[0] == '/' {
		return normalize(component)
	}

	if caller == "" {
		return normalize("/" + component)
	}

	return normalize(filepath.Join(filepath.Dir(caller), component))
}

func ID(components ...string) string {
	var ns []string
	for i, n := range components {
		if n == "" {
			continue
		} else if n[0] == '/' {
			return normalize(n)
		} else if i == 0 {
			n = "/" + n
		}
		ns = append(ns, normalize(n))
	}
	return strings.Join(ns, "/")
}

func normalize(n string) string {
	return strings.ToLower(strings.ReplaceAll(n, " ", "-"))
}
