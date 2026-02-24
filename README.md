# github-vote-llm

Community-driven feature development powered by GitHub reactions and Claude Code. Users vote on feature requests with thumbs-up reactions, and when a request is approved, an AI agent automatically implements it and opens a pull request.

## How it works

1. A feature request is filed as a GitHub issue with the `feature-request` label
2. A maintainer labels the issue `approved-for-dev`
3. vote-llm receives the webhook event, adds an `llm-in-progress` label, clones the repo, and runs Claude Code against the issue
4. Claude Code implements the feature; vote-llm commits, pushes, and opens a PR
5. The issue is labeled `llm-pr-created` and a comment links to the PR

## Project structure

```
cmd/main/           HTTP server entry point
internal/
  agent/            Claude Code orchestration (clone, run, commit, PR)
  github/           GitHub App auth and API client
  handlers/         Webhook event handler
  logger/           Structured logging (zap)
  middleware/       Webhook signature validation
  spinner/          Terminal progress spinner
```

## Requirements

- Go 1.25+
- A [GitHub App](https://docs.github.com/en/apps/creating-github-apps) installed on the target repo
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and available on `$PATH`

## Configuration

All config is via environment variables:

| Variable             | Required | Description                                   |
| -------------------- | -------- | --------------------------------------------- |
| `GITHUB_APP_ID`      | yes      | GitHub App numeric ID                         |
| `GITHUB_PRIVATE_KEY` | yes      | PEM bytes as a string                         |
| `WEBHOOK_SECRET`     | yes      | HMAC secret matching the App's webhook config |
| `ANTHROPIC_API_KEY`  | yes      | API key passed to the `claude` CLI            |
| `PORT`               | no       | HTTP listen port (default: `8080`)            |

For local development, put these in `.env.development` — they are loaded automatically when `GIN_MODE=debug`.

## Running

```bash
go build -o vote-llm ./cmd/main
export GITHUB_APP_ID=123456
export GITHUB_PRIVATE_KEY=pem-string
export WEBHOOK_SECRET=your-secret
./vote-llm
```

## Endpoints

| Path                   | Description                              |
| ---------------------- | ---------------------------------------- |
| `POST /github/webhook` | Receives GitHub webhook events           |
| `GET /health`          | Health check (returns `{"status":"ok"}`) |

## Labels

| Label              | Meaning                                                 |
| ------------------ | ------------------------------------------------------- |
| `feature-request`  | Marks an issue as eligible for automated implementation |
| `approved-for-dev` | Triggers the agent (also requires `feature-request`)    |
| `llm-in-progress`  | Agent is currently working on this issue                |
| `llm-pr-created`   | PR has been opened successfully                         |
| `llm-failed`       | Agent run failed; error details in issue comment        |

## License

See [LICENSE](LICENSE) for details.
