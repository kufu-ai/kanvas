package kanvas

type WorkflowJob struct {
	Dir     string
	Needs   []string
	Outputs map[string]string
	Ran     bool
}

func (j *WorkflowJob) Diff() error {
	if j.Ran {
		return nil
	}

	j.Ran = true

	return nil
}

func (j *WorkflowJob) Apply() error {
	if j.Ran {
		return nil
	}

	j.Ran = true

	return nil
}
