package args

type OpenRepoPullRequestArgs struct {
	UserID     int64  `json:"user_id"`
	RunID      int64  `json:"run_id"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	BranchName string `json:"branch_name"`
	Prompt     string `json:"prompt"`
}

func (OpenRepoPullRequestArgs) Kind() string {
	return "open_pr"
}
