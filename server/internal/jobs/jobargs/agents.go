package jobargs

type RunAgentArgs struct {
	Prompt  string `json:"prompt"`
	WorkDir string `json:"work_dir"`
	RunID   int64  `json:"run_id"`
}

func (RunAgentArgs) Kind() string {
	return "run_agent"
}
