package test

import (
	"fmt"
	"io/ioutil"
	"kanvas"
	"kanvas/app"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	run(t, "reference", "")
}

func run(t *testing.T, sub, env string) {
	t.Helper()

	var name string
	if env == "" {
		name = sub
	} else {
		name = fmt.Sprintf("%s-%s", sub, env)
	}

	t.Run(name, func(t *testing.T) {
		var (
			configFile = filepath.Join(name, kanvas.DefaultConfigFileYAML)
			exportsDir = filepath.Join(name, "exports")
			destDir    = t.TempDir()
		)

		files, err := ioutil.ReadDir(exportsDir)
		require.NoError(t, err)

		exports := map[string]string{}
		for _, f := range files {
			fn := filepath.Join(exportsDir, f.Name())
			data, err := os.ReadFile(fn)
			require.NoError(t, err)
			exports[f.Name()] = string(data)
		}

		a, err := app.New(kanvas.Options{
			ConfigFile: configFile,
			Env:        env,
		})
		require.NoError(t, err)

		require.NoError(t, a.Export("githubactions", destDir, "kanvas:example"))

		destFiles, err := ioutil.ReadDir(destDir)
		require.NoError(t, err)

		for _, f := range destFiles {
			want, ok := exports[f.Name()]

			got, err := os.ReadFile(filepath.Join(destDir, f.Name()))
			assert.NoError(t, err)

			assert.True(t, ok, "Unexpected file %q has been exported: %s", f.Name(), string(got))
			assert.Equal(t, want, string(got))

			if t.Failed() {
				if os.Getenv("UPDATE_SNAPSHOT") == t.Name() {
					fn := filepath.Join(exportsDir, f.Name())
					require.Errorf(t, os.WriteFile(fn, got, 0666), "Saving snapshot at %s", fn)
				} else {
					t.Errorf("Rerun test with UPDATE_SNAPSHOT=%s in order to update the snapshot", t.Name())
				}
			}

			delete(exports, f.Name())
		}

		for f := range exports {
			t.Errorf("Expected %q to be exported", f)
		}
	})
}
