package app

import (
	"kanvas"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type args struct {
	a *App
}

func TestGenerateConfigData(t *testing.T) {
	run(t, "simple")
}

func run(t *testing.T, name string) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		var (
			testdataDir      = "testdata"
			wantedConfigFile = filepath.Join(testdataDir, name, kanvas.DefaultConfigFileYAML)
		)

		args := generateArgs{
			Dir: filepath.Join("testdata", name),
		}

		a := &App{}

		want, err := os.ReadFile(wantedConfigFile)
		require.NoError(t, err)

		got, err := a.generateConfigData(args)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got))

		if t.Failed() {
			if os.Getenv("UPDATE_SNAPSHOT") == t.Name() {
				fn := wantedConfigFile
				require.Errorf(t, os.WriteFile(fn, got, 0666), "Saving snapshot at %s", fn)
			} else {
				t.Errorf("Rerun test with UPDATE_SNAPSHOT=%s in order to update the snapshot", t.Name())
			}
		}
	})
}
