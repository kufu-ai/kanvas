package kanvas

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderOrReadFile(t *testing.T) {
	v := os.Getenv("GITHUB_REPOSITORY")
	defer os.Setenv("GITHUB_REPOSITORY", v)

	os.Setenv("GITHUB_REPOSITORY", "owner/repo")

	f, err := RenderOrReadFile("testdata/render_or_read_file_test.template.jsonnet")
	require.NoError(t, err)

	require.Equal(t, `{
   "my_repo_name": "repo-suffix1",
   "my_repo_owner": "owner-suffix2"
}
`, string(f))
}

func TestRenderOrReadFileNoEnv(t *testing.T) {
	v := os.Getenv("GITHUB_REPOSITORY")
	defer os.Setenv("GITHUB_REPOSITORY", v)

	os.Setenv("GITHUB_REPOSITORY", "")

	_, err := RenderOrReadFile("testdata/render_or_read_file_test.template.jsonnet")
	require.Equal(t,
		".template.jsonnet requires GITHUB_REPOSITORY to be set to OWNER/REPO_NAME for the template to access `std.extVar(\"github_repo_name\")` and `std.extVar(\"github_repo_owner\")`",
		err.Error(),
	)
}
