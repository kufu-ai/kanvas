package kanvas_test

import (
	"path/filepath"
	"testing"

	"github.com/davinci-std/kanvas"
	"github.com/davinci-std/kanvas/interpreter"
	"github.com/mumoshu/kargo"
	"github.com/stretchr/testify/require"
)

func newComponent() kanvas.Component {
	testData := filepath.Join("testdata", "workflow")
	kanvasYAMLPath := filepath.Join(testData, "kanvas.yaml")
	return kanvas.Component{
		Dir: filepath.Dir(kanvasYAMLPath),
		Components: map[string]kanvas.Component{
			"prereq": {
				AWS: &kanvas.AWS{
					Account: "12345",
				},
			},
			"image": {
				Needs: []string{"git", "prereq"},
				Docker: &kanvas.Docker{
					Image:    "myaccount.dkr.ecr.ap-northeast-1.amazonaws.com/myapp",
					File:     "Dockerfile",
					TagsFrom: []string{"git.sha"},
				},
			},
			"deploy": {
				Needs: []string{"image"},
				Kubernetes: &kanvas.Kubernetes{
					Config: kargo.Config{
						Kustomize: &kargo.Kustomize{
							Git: kargo.KustomizeGit{
								Path: "path/to/dir",
							},
							Images: kargo.KustomizeImages{
								{
									Name:       "myaccount.dkr.ecr.ap-northeast-1.amazonaws.com/myapp",
									NewTagFrom: "image.tag",
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestWorkflowLoad(t *testing.T) {
	plan := [][]string{
		{"git", "prereq"},
		{"image"},
		{"deploy"},
	}
	c := newComponent()
	o := kanvas.Options{
		TempDir: t.TempDir(),
	}

	w, err := kanvas.NewWorkflow(c, o)
	require.NoError(t, err)

	require.Equal(t, plan, w.Plan)
}

func TestWorkflowLoad_SkipImage(t *testing.T) {
	plan := [][]string{
		{"image", "prereq"},
		{"deploy"},
	}

	opts := func(o *kanvas.Options) {
		o.Skip = []string{"image"}
		o.SkippedJobsOutputs = map[string]map[string]string{
			"image": {
				"id": "sha256:12356935acbd6a67d3fed10512d89450330047f3ae6fb3a62e9bf4f229529387",
			},
		}
	}

	c := newComponent()
	o := kanvas.Options{
		TempDir: t.TempDir(),
	}

	opts(&o)

	w, err := kanvas.NewWorkflow(c, o)
	require.NoError(t, err)

	require.Equal(t, plan, w.Plan)
}

func TestWorkflowLoad_SkipTransitive(t *testing.T) {
	plan := [][]string{
		{"image", "prereq"},
		{"deploy"},
	}

	opts := func(o *kanvas.Options) {
		o.Skip = []string{"image", "prereq"}
		o.SkippedJobsOutputs = map[string]map[string]string{
			"image": {
				"tag": "foobar",
			},
			// "git": {
			// 	"sha": "123d3c1a9a669f45c17a25cf5222c0cc0b630738",
			// },
			"prereq": {},
		}
	}

	c := newComponent()
	o := kanvas.Options{
		TempDir: t.TempDir(),
	}

	opts(&o)

	w, err := kanvas.NewWorkflow(c, o)
	require.NoError(t, err)

	require.Equal(t, plan, w.Plan)

	i := interpreter.New(w, kanvas.NewRuntime())

	require.NoError(t, i.Diff())
}

func TestWorkflowLoad_TryingToSkipUnneededGit(t *testing.T) {
	opts := func(o *kanvas.Options) {
		o.Skip = []string{"image", "prereq"}
		o.SkippedJobsOutputs = map[string]map[string]string{
			"image": {
				"tag": "foobar",
			},
			"git": {
				"sha": "123d3c1a9a669f45c17a25cf5222c0cc0b630738",
			},
			"prereq": {},
		}
	}

	c := newComponent()
	o := kanvas.Options{
		TempDir: t.TempDir(),
	}

	opts(&o)

	_, err := kanvas.NewWorkflow(c, o)
	require.Error(t, err)
	require.Equal(t, `loading "" "testdata/workflow": the number of skipped jobs (2) doesn't match the number of skipped jobs outputs (3)`, err.Error())
}
