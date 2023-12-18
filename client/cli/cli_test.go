package cli

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/davinci-std/kanvas/client"

	"github.com/stretchr/testify/require"
)

func TestCLIApply(t *testing.T) {
	cmd := setupTestCommand(t, "--config", "kanvas.yaml", "--env", "dev", "apply")

	cli := New()
	cli.Command = cmd

	got, err := cli.Apply(context.Background(), "testdata/kanvas.yaml", "dev", client.ApplyOptions{})
	require.NoError(t, err)

	require.Equal(t, &client.ApplyResult{Outputs: map[string]client.Output{}}, got)
}

func TestCLIDiff(t *testing.T) {
	cmd := setupTestCommand(t, "--config", "kanvas.yaml", "--env", "dev", "diff")

	cli := New()
	cli.Command = cmd

	got, err := cli.Diff(context.Background(), "testdata/kanvas.yaml", "dev", client.DiffOptions{})
	require.NoError(t, err)

	require.Equal(t, &client.DiffResult{}, got)
}

const (
	env      = "RUN_MAIN_FOR_TESTING"
	envValue = "1"
	argsEnv  = "RUN_MAIN_FOR_TESTING_ARGS"
	argsSep  = ","
)

// setupTestCommand sets up the test command, that
// fails with exit code 1 when the arguments passed to the command
// are not the same as the expected ones.
func setupTestCommand(t *testing.T, expectKanvasArgs ...string) []string {
	// This is a hack to run the main function in this file,
	// from the command that is run by the test.
	//
	// This originates from the hack I used in the helm project:
	//   https://github.com/helm/helm/pull/6790

	if os.Getenv(env) == envValue {
		main()
	}

	os.Setenv(env, envValue)
	os.Setenv(argsEnv, strings.Join(expectKanvasArgs, argsSep))

	return []string{os.Args[0], "-test.run=" + t.Name(), "--"}
}

// This is a mock of the kanvas command.
func main() {
	want := strings.Split(os.Getenv(argsEnv), argsSep)
	got := os.Args[3:]
	if !reflect.DeepEqual(want, got) {
		fmt.Fprintf(os.Stderr, "want: %v\n", want)
		fmt.Fprintf(os.Stderr, "got : %v\n", got)
		fmt.Fprintf(os.Stderr, "args : %v\n", os.Args)
		fmt.Fprintf(os.Stderr, "env : %v\n", os.Environ())
		os.Exit(1)
	}
	fmt.Println("{}")
	os.Exit(0)
}
