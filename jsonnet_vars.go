package kanvas

import (
	"os"
	"strings"
)

func getJsonnetVars() map[string]string {
	vars := make(map[string]string)

	if v := os.Getenv("GITHUB_REPOSITORY"); v != "" {
		splits := strings.Split(v, "/")
		vars["github_repo_owner"] = splits[0]
		vars["github_repo_name"] = splits[1]
	}

	return vars
}
