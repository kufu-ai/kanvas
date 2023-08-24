package ghrepos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarizerr(t *testing.T) {
	s := Summarizer{
		GitHubToken: GitHubToken,
	}
	ws := ProjectRoot
	summary, err := s.Summarize(ws)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(summary.Repos), 2)

	for _, content := range summary.Contents {
		t.Log(content)
	}
}
