package client

import (
	"testing"

	"github.com/mumoshu/kargo/tools"
	"github.com/stretchr/testify/require"
)

func TestApplyResultUnmarshal(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var got ApplyResult
		if err := got.UnmarshalJSON([]byte(`{}`)); err != nil {
			t.Fatal(err)
		}
		want := ApplyResult{Outputs: map[string]Output{}}
		require.Equal(t, want, got)
	})
	t.Run("with outputs", func(t *testing.T) {
		var got ApplyResult
		jsonBytes := []byte(`{
			"foo": {
				"pullRequest": {
					"number": 1
				}
			}
		}`)
		if err := got.UnmarshalJSON(jsonBytes); err != nil {
			t.Fatal(err)
		}
		want := ApplyResult{
			Outputs: map[string]Output{
				"foo": {
					PullRequest: &tools.PullRequest{
						Number: 1,
					},
				},
			},
		}
		require.Equal(t, want, got)
	})
}
