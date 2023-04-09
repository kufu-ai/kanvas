package kanvas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopsort(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		got, err := topologicalSort(map[string][]string{
			"1": {"2", "3"},
			"2": {"3"},
			"3": {},
		})

		require.NoError(t, err)
		require.Equal(t, [][]string{{"3"}, {"2"}, {"1"}}, got)
	})

	t.Run("cycle", func(t *testing.T) {
		_, err := topologicalSort(map[string][]string{
			"1": {"2", "3"},
			"2": {"3"},
			"3": {"1"},
		})

		require.Error(t, err)
	})

	t.Run("multi", func(t *testing.T) {
		deps, err := topologicalSort(map[string][]string{
			"1": {"2", "3"},
			"2": {"4"},
			"3": {"5"},
			"4": {},
			"5": {},
		})

		require.NoError(t, err)
		require.Equal(t, [][]string{{"4", "5"}, {"2", "3"}, {"1"}}, deps)
	})

	t.Run("most dependeded and zero depended jobs form level 0", func(t *testing.T) {
		deps, err := topologicalSort(map[string][]string{
			"1": {"2", "3"},
			"2": {"4"},
			"3": {"5"},
			"4": {},
			"5": {},
			"6": {},
		})

		require.NoError(t, err)
		require.Equal(t, [][]string{{"4", "5", "6"}, {"2", "3"}, {"1"}}, deps)
	})
}
