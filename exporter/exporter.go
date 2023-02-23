package exporter

import (
	"fmt"
	"kanvas"
)

// TODO Probably we'd better rename this to "Plugin"
// if we are going to consolidate Export and Output features into this package and struct
type Exporter struct {
	wf *kanvas.Workflow
	r  *kanvas.Runtime
}

func New(wf *kanvas.Workflow, r *kanvas.Runtime) *Exporter {
	return &Exporter{
		wf: wf,
		r:  r,
	}
}

func (e *Exporter) Export(format string, dir string) error {
	if format != FormatGitHubActions {
		return fmt.Errorf("unsupported format %q", format)
	}

	return e.exportActionsWorkflows(dir)
}

func (e *Exporter) Output(format, target string) error {
	if format != FormatGitHubActions {
		return fmt.Errorf("unsupported format %q", format)
	}

	return e.outputActionsWorkflows(target)
}
