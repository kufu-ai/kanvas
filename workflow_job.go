package kanvas

type WorkflowJob struct {
	Dir     string
	Needs   []string
	Outputs map[string]string
	Ran     bool

	driver *driver
}

func (j *WorkflowJob) runWithExtraArgs(wf *Workflow, cmd []string) error {
	var c []string

	c = append(c, cmd...)

	j.driver.args.Visit(func(str string) {
		c = append(c, str)
	}, func(out string) {
		// TODO Add the referenced output as an arg
	})

	return wf.exec(j.Dir, c)
}

func (j *WorkflowJob) Diff(wf *Workflow) error {
	if j.Ran {
		return nil
	}

	if err := j.runWithExtraArgs(wf, j.driver.diff); err != nil {
		return err
	}

	j.Ran = true

	return nil
}

func (j *WorkflowJob) Apply(wf *Workflow) error {
	if j.Ran {
		return nil
	}

	if err := j.runWithExtraArgs(wf, j.driver.apply); err != nil {
		return err
	}

	j.Ran = true

	return nil
}
