package kanvas

type WorkflowJob struct {
	Dir     string
	Needs   []string
	Outputs map[string]string
	Ran     bool

	Driver *driver
}

func (j *WorkflowJob) runWithExtraArgs(wf *Workflow, dir string, cmd []string) error {
	var c []string

	c = append(c, cmd...)

	j.Driver.args.Visit(func(str string) {
		c = append(c, str)
	}, func(out string) {
		// TODO Add the referenced output as an arg
	})

	return wf.exec(dir, c)
}

func (j *WorkflowJob) Diff(wf *Workflow) error {
	if j.Ran {
		return j.runWithExtraArgs(wf, j.Dir, j.Driver.diff)
	}

	j.Ran = true

	return nil
}

func (j *WorkflowJob) Apply(wf *Workflow) error {
	if j.Ran {
		return j.runWithExtraArgs(wf, j.Dir, j.Driver.apply)
	}

	j.Ran = true

	return nil
}
