package kanvas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	t.Run("top-level component ID", func(t *testing.T) {
		require.Equal(t, "/foo", ID("foo"))
	})

	t.Run("Sub component ID", func(t *testing.T) {
		require.Equal(t, "/foo/bar", ID("foo", "bar"))
	})

	t.Run("Fully-qualifid component ID", func(t *testing.T) {
		require.Equal(t, "/bar/baz", ID("foo", "/bar/baz"))
	})
}
