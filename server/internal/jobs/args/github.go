package args

type CloneRepoArgs struct {
	UserID int64 `json:"user_id"`
	RunID  int64 `json:"run_id"`
}

func (CloneRepoArgs) Kind() string {
	return "clone_repo"
}
