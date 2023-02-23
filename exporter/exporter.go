package exporter

import "kanvas"

type Exporter struct {
	wf *kanvas.Workflow
}

func New(wf *kanvas.Workflow) *Exporter {
	return &Exporter{
		wf: wf,
	}
}

func (e *Exporter) Export(dir string) error {
	return e.exportActionsWorkflows(dir)
}
