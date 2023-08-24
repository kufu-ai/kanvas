package configai

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuggest(t *testing.T) {
	if APIKey == "" {
		t.Logf("You must provide an OPENAI_API_KEY to run this test")
		t.SkipNow()
	}

	var buf bytes.Buffer

	contents, err := os.ReadFile("testdata/contents.txt")
	require.NoError(t, err)

	repos, err := os.ReadFile("testdata/repos.txt")
	require.NoError(t, err)

	c := &ConfigRecommender{APIKey: APIKey}

	sseGeneratedYAML, err := c.Suggest(string(contents), string(repos), WithUseFun(true), WithSSE(true), WithLog(&buf))
	require.NoError(t, err)

	logs := buf.String()
	require.Contains(t, logs, "finish_reason\":\"function_call\"")

	require.Contains(t, *sseGeneratedYAML, "components:")

	generatedYAML, err := c.Suggest(string(contents), string(repos), WithUseFun(true), WithLog(&buf))
	require.NoError(t, err)

	require.Contains(t, *generatedYAML, "components:")
}
