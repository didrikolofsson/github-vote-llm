package jobargs

type RunAgentArgs struct {
	Prompt  string `json:"prompt"`
	WorkDir string `json:"work_dir"`
	ApiKey  string `json:"api_key"`
}

func (RunAgentArgs) Kind() string {
	return "run_agent"
}

type CloneRepoArgs struct {
	UserID    int64
	RunID     int64
	Owner     string
	Name      string
	Workspace string
}

func (CloneRepoArgs) Kind() string {
	return "clone_repo"
}
