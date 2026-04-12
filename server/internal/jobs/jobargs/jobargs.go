package jobargs

import "github.com/didrikolofsson/github-vote-llm/internal/api/dtos"

// RunClaudeArgs must stay JSON-serializable: River persists args and reloads them
// in the worker process, so non-serializable types (*store.Queries, *oauth2.Config)
// must not be stored here—inject those on RunClaudeWorker instead.
type RunAgentArgs struct {
	Prompt             string           `json:"prompt"`
	TokenEncryptionKey string           `json:"token_encryption_key"`
	Repository         *dtos.Repository `json:"repository"`
	Workspace          string           `json:"workspace"`
	ApiKey             string           `json:"api_key"`
}

func (RunAgentArgs) Kind() string {
	return "run_agent"
}

type CloneRepoArgs struct {
	UserID    int64
	Owner     string
	Name      string
	Workspace string
}

func (CloneRepoArgs) Kind() string {
	return "clone_repo"
}
