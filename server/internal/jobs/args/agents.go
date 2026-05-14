package args

type RunAgentArgs struct {
	RunID int64 `json:"run_id"`
}

func (RunAgentArgs) Kind() string {
	return "run_agent"
}
