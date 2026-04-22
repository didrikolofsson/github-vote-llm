package agents

import "context"

type Runner interface {
	Run(ctx context.Context, prompt string) error
}
