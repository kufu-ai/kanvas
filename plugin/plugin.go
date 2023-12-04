package plugin

import (
	"fmt"

	"github.com/davinci-std/kanvas"
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

func (e *Plugin) Export(format string, dir, kanvasContainerImage string) error {
	if format != FormatGitHubActions {
		return fmt.Errorf("unsupported format %q", format)
	}

	return e.exportActionsWorkflows(dir, kanvasContainerImage)
}

func (e *Plugin) Output(op kanvas.Op, format, target string) error {
	if format != FormatGitHubActions {
		return fmt.Errorf("unsupported format %q", format)
	}

	return e.outputActionsWorkflows(op, target)
}
