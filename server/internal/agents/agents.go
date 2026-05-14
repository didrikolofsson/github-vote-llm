package agents

import "context"

type Runner interface {
	Run(ctx context.Context, prompt, workDir string, onStart func(pid int), onLine func(stream, line string)) error
}
