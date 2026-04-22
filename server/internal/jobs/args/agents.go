package args

type RunAgentArgs struct {
	UserID int64 `json:"user_id"`
	RunID  int64 `json:"run_id"`
}

func (RunAgentArgs) Kind() string {
	return "run_agent"
}
