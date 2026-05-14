package args

type CloneRepoArgs struct {
	RunID int64 `json:"run_id"`
}

func (CloneRepoArgs) Kind() string {
	return "clone_repo"
}
