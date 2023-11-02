package test

import (
	"fmt"
	"kanvas"
	"kanvas/app"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Config struct {
	Env   string
	Error string
}

type Option func(*Config)

func Error(err string) Option {
	return func(c *Config) {
		c.Error = err
	}
}

func Env(env string) Option {
	return func(c *Config) {
		c.Env = env
	}
}

func TestExport(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	run(t, "reference")
	run(t, "jsonnet")
	run(t, "unusedenv", Env("dev"), Error(`environment "dev" uses "missing" but it is not defined`))
}

func run(t *testing.T, sub string, opts ...Option) {
	t.Helper()

	var (
		config Config
	)

	for _, opt := range opts {
		opt(&config)
	}

	env := config.Env
	wantErr := config.Error

	var name string
	if env == "" {
		name = sub
	} else {
		name = fmt.Sprintf("%s-%s", sub, env)
	}

	t.Run(name, func(t *testing.T) {
		var (
			exportsDir = filepath.Join(name, "exports")
			destDir    = t.TempDir()

			exports = map[string]string{}
		)

		if wantErr == "" {
			files, err := os.ReadDir(exportsDir)
			require.NoError(t, err)

			for _, f := range files {
				fn := filepath.Join(exportsDir, f.Name())
				data, err := os.ReadFile(fn)
				require.NoError(t, err)
				exports[f.Name()] = string(data)
			}
		}

		wd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(sub))
		a, err := app.New(kanvas.Options{
			Env: env,
		})
		require.NoError(t, os.Chdir(wd))
		require.NoError(t, err)

		gotErr := a.Export("githubactions", destDir, "kanvas:example")
		if wantErr != "" {
			require.EqualError(t, gotErr, wantErr)
		} else {
			require.NoError(t, gotErr)
		}

		destFiles, err := os.ReadDir(destDir)
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
