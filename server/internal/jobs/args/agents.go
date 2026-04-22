package args

type RunAgentArgs struct {
	UserID     int64  `json:"user_id"`
	RunID      int64  `json:"run_id"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	BranchName string `json:"branch_name"`
	Prompt     string `json:"prompt"`
	WorkDir    string `json:"work_dir"`
}

func (RunAgentArgs) Kind() string {
	return "run_agent"
}
