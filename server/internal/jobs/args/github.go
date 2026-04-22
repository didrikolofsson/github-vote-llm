package args

type CloneRepoArgs struct {
	UserID int64  `json:"user_id"`
	RunID  int64  `json:"run_id"`
	Owner  string `json:"owner"`
	Name   string `json:"name"`
}

func (CloneRepoArgs) Kind() string {
	return "clone_repo"
}
