package agents

import "context"

type Event struct {
	Chunk string `json:"chunk"`
	Err   error  `json:"error"`
}
type AgentRunner interface {
	Run(ctx context.Context, prompt string) (<-chan Event, error)
}
