package plugin

import (
	"fmt"
	"kanvas"
)

type Plugin struct {
	wf *kanvas.Workflow
	r  *kanvas.Runtime
}

func New(wf *kanvas.Workflow, r *kanvas.Runtime) *Plugin {
	return &Plugin{
		wf: wf,
		r:  r,
	}
}

func (e *Plugin) Export(format string, dir string) error {
	if format != FormatGitHubActions {
		return fmt.Errorf("unsupported format %q", format)
	}

	return e.exportActionsWorkflows(dir)
}

func (e *Plugin) Output(format, target string) error {
	if format != FormatGitHubActions {
		return fmt.Errorf("unsupported format %q", format)
	}

	return e.outputActionsWorkflows(target)
}
