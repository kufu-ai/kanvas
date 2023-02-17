package kanvas

import "strings"

func id(components ...string) string {
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
