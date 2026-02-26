package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/spinner"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	gogithub "github.com/google/go-github/v68/github"
)

var agentAllowedTools = []string{"Read", "Edit", "Write", "Bash", "Glob", "Grep"}

// Runner orchestrates Claude Code to implement approved features.
type Runner struct {
	client       gh.ClientAPI
	log          *logger.Logger
	workspaceDir string
	store        store.Store
}

// NewRunner creates a new agent runner. workspaceDir is the base directory for
// repo clones; use DefaultWorkspaceDir as the fallback.
func NewRunner(client gh.ClientAPI, log *logger.Logger, workspaceDir string, st store.Store) *Runner {
	return &Runner{client: client, log: log.Named("agent"), workspaceDir: workspaceDir, store: st}
}

// claudeResult represents the JSON output from claude -p.
type claudeResult struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

// Run implements a feature request: clones repo, runs Claude Code, creates a PR.
func (r *Runner) Run(ctx context.Context, owner, repo string, issue *gogithub.Issue, executionID int64) {
	issueNum := issue.GetNumber()
	r.log.Infow("starting work on issue", "issue", issueNum, "repo", owner+"/"+repo)

	branchName := fmt.Sprintf("vote-llm/issue-%d-%s", issueNum, slugify(issue.GetTitle()))

	// Mark in-progress in DB
	if _, err := r.store.SetInProgress(ctx, executionID, branchName); err != nil {
		r.log.Warnw("failed to set execution in-progress", "error", err)
	}

	// Mark issue as in-progress on GitHub
	if err := r.client.AddLabel(ctx, owner, repo, issueNum, config.LabelInProgress); err != nil {
		r.log.Warnw("failed to add in-progress label", "error", err)
	}

	// Remove approved label to prevent re-triggering
	if err := r.client.RemoveLabel(ctx, owner, repo, issueNum, config.LabelApproved); err != nil {
		r.log.Warnw("failed to remove approved label", "error", err)
	}

	// Fetch per-repo config to resolve timeout and budget overrides
	repoConfig, err := r.store.GetRepoConfig(ctx, owner, repo)
	if err != nil {
		r.log.Warnw("failed to fetch repo config, using defaults", "error", err)
	}

	// Run with timeout
	timeout := resolveTimeout(repoConfig)
	implCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prURL, err := r.implement(implCtx, owner, repo, issue, branchName, repoConfig)
	if err != nil {
		r.log.Errorw("failed to implement issue", "issue", issueNum, "error", err)

		if _, dbErr := r.store.SetFailed(ctx, executionID, err.Error()); dbErr != nil {
			r.log.Warnw("failed to set execution failed", "error", dbErr)
		}

		comment := fmt.Sprintf("**vote-llm**: Failed to implement this feature.\n\n```\n%s\n```", err)
		if cErr := r.client.CreateComment(ctx, owner, repo, issueNum, comment); cErr != nil {
			r.log.Warnw("failed to comment on issue", "error", cErr)
		}
		if err := r.client.RemoveLabel(ctx, owner, repo, issueNum, config.LabelInProgress); err != nil {
			r.log.Warnw("failed to remove in-progress label", "error", err)
		}
		if err := r.client.AddLabel(ctx, owner, repo, issueNum, config.LabelFailed); err != nil {
			r.log.Warnw("failed to add failed label", "error", err)
		}
		return
	}

	if _, dbErr := r.store.SetSuccess(ctx, executionID, prURL); dbErr != nil {
		r.log.Warnw("failed to set execution success", "error", dbErr)
	}

	// Mark issue as done and comment with PR link
	if err := r.client.AddLabel(ctx, owner, repo, issueNum, config.LabelDone); err != nil {
		r.log.Warnw("failed to add done label", "error", err)
	}
	if err := r.client.RemoveLabel(ctx, owner, repo, issueNum, config.LabelInProgress); err != nil {
		r.log.Warnw("failed to remove in-progress label", "error", err)
	}

	comment := fmt.Sprintf("**vote-llm**: A PR has been created for this feature request: %s\n\nPlease review the changes.", prURL)
	if err := r.client.CreateComment(ctx, owner, repo, issueNum, comment); err != nil {
		r.log.Warnw("failed to comment on issue", "error", err)
	}

	r.log.Infow("successfully created PR", "issue", issueNum, "pr", prURL)
}

func (r *Runner) implement(ctx context.Context, owner, repo string, issue *gogithub.Issue, branchName string, repoConfig *store.RepoConfig) (string, error) {
	// Prepare workspace
	workDir := filepath.Join(r.workspaceDir, owner, repo)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}

	repoDir := filepath.Join(workDir, "repo")

	spinner := spinner.NewSpinner()
	spinner.Start(fmt.Sprintf("Preparing workspace for issue #%d...", issue.GetNumber()))

	// Clone or update repo
	if err := r.cloneOrResetRepo(ctx, owner, repo, repoDir); err != nil {
		return "", fmt.Errorf("ensure repo: %w", err)
	}

	// Get default branch
	defaultBranch, err := r.client.GetDefaultBranch(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("get default branch: %w", err)
	}

	// Create feature branch
	if err := r.gitCheckoutNewBranch(ctx, repoDir, defaultBranch, branchName); err != nil {
		return "", fmt.Errorf("create branch: %w", err)
	}
	spinner.UpdateMessage(fmt.Sprintf("Running Claude Code on issue #%d...", issue.GetNumber()))

	// Build prompt and run Claude Code
	prompt := r.buildPrompt(issue, owner, repo)
	if err := r.runClaude(ctx, repoDir, prompt, repoConfig); err != nil {
		return "", fmt.Errorf("claude code: %w", err)
	}

	// Check if there are any changes
	hasChanges, err := r.gitHasChanges(ctx, repoDir)
	if err != nil {
		return "", fmt.Errorf("check changes: %w", err)
	}
	if !hasChanges {
		return "", fmt.Errorf("claude code produced no changes")
	}

	// Stage, commit, push
	spinner.UpdateMessage(fmt.Sprintf("Pushing changes for issue #%d...", issue.GetNumber()))
	if err := r.gitCommitAndPush(ctx, repoDir, branchName, issue.GetNumber(), issue.GetTitle(), owner, repo); err != nil {
		return "", fmt.Errorf("commit and push: %w", err)
	}

	// Create PR (or find existing one for this branch)
	spinner.UpdateMessage(fmt.Sprintf("Creating pull request for issue #%d...", issue.GetNumber()))
	prTitle := fmt.Sprintf("feat: %s (issue #%d)", issue.GetTitle(), issue.GetNumber())
	prBody := prBodyForIssue(issue.GetNumber())
	pr, err := r.client.CreatePullRequest(ctx, owner, repo, branchName, defaultBranch, prTitle, prBody)
	if err != nil {
		// If PR creation failed, check if one already exists for this branch
		existing, findErr := r.client.FindPullRequestByHead(ctx, owner, repo, branchName)
		if findErr != nil {
			spinner.Stop(fmt.Sprintf("PR creation failed for issue #%d!", issue.GetNumber()))
			return "", fmt.Errorf("create PR: %w (also failed to find existing: %v)", err, findErr)
		}
		if existing != nil {
			r.log.Infow("PR already exists for branch, reusing", "branch", branchName, "pr", existing.GetHTMLURL())
			spinner.Stop(fmt.Sprintf("PR created successfully for issue #%d!", issue.GetNumber()))
			return existing.GetHTMLURL(), nil
		}
		spinner.Stop(fmt.Sprintf("PR creation failed for issue #%d!", issue.GetNumber()))
		return "", fmt.Errorf("create PR: %w", err)
	}

	spinner.Stop(fmt.Sprintf("PR created successfully for issue #%d!", issue.GetNumber()))
	return pr.GetHTMLURL(), nil
}

func (r *Runner) cloneOrResetRepo(ctx context.Context, owner, repo, repoDir string) error {
	// Get token for authenticated git operations
	token, err := r.client.GetInstallationToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}
	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		// Repo exists — update remote URL with fresh token, then fetch
		cmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", cloneURL)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git set-url: %s: %w", out, err)
		}

		cmd = exec.CommandContext(ctx, "git", "fetch", "--all", "--prune")
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git fetch: %s: %w", out, err)
		}
		return nil
	}

	// Clone fresh
	cmd := exec.CommandContext(ctx, "git", "clone", "--filter=blob:none", cloneURL, repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s: %w", out, err)
	}
	return nil
}

func (r *Runner) gitCheckoutNewBranch(ctx context.Context, repoDir, baseBranch, newBranch string) error {
	// Clean up any leftover files from previous failed runs
	cmd := exec.CommandContext(ctx, "git", "clean", "-fd")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clean: %s: %w", out, err)
	}

	cmd = exec.CommandContext(ctx, "git", "checkout", baseBranch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("checkout %s: %s: %w", baseBranch, out, err)
	}

	cmd = exec.CommandContext(ctx, "git", "reset", "--hard", "origin/"+baseBranch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reset to origin/%s: %s: %w", baseBranch, out, err)
	}

	// Create and checkout new branch
	cmd = exec.CommandContext(ctx, "git", "checkout", "-B", newBranch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create branch %s: %s: %w", newBranch, out, err)
	}

	return nil
}

func (r *Runner) buildPrompt(issue *gogithub.Issue, owner, repo string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are implementing a feature request for the repository %s/%s.\n\n", owner, repo)
	fmt.Fprintf(&b, "## Feature Request (GitHub Issue #%d)\n", issue.GetNumber())
	fmt.Fprintf(&b, "**Title:** %s\n", issue.GetTitle())
	fmt.Fprintf(&b, "**Description:**\n%s\n\n", issue.GetBody())
	fmt.Fprintf(&b, "## Instructions\n")
	fmt.Fprintf(&b, "- Read the codebase to understand the project structure and conventions\n")
	fmt.Fprintf(&b, "- Implement the requested feature following existing patterns\n")
	fmt.Fprintf(&b, "- Write tests for the new functionality\n")
	fmt.Fprintf(&b, "- Ensure all existing tests still pass\n")
	fmt.Fprintf(&b, "- Keep changes minimal and focused on the request\n")
	fmt.Fprintf(&b, "- Do NOT commit changes — just edit the files\n")
	return b.String()
}

func (r *Runner) runClaude(ctx context.Context, repoDir, prompt string, repoConfig *store.RepoConfig) error {
	maxBudget := config.AgentMaxBudgetUSD
	if repoConfig != nil && repoConfig.MaxBudgetUsd.Valid {
		if f, err := repoConfig.MaxBudgetUsd.Float64Value(); err == nil && f.Valid {
			maxBudget = f.Float64
		}
	}

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--allowedTools", strings.Join(agentAllowedTools, ","),
		"--max-turns", strconv.Itoa(config.AgentMaxTurns),
		"--max-budget-usd", fmt.Sprintf("%.2f", maxBudget),
		"--no-session-persistence",
	}

	cmd := exec.CommandContext(ctx, config.AgentCommand, args...)
	cmd.Dir = repoDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	r.log.Infow("running claude", "dir", repoDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude exited with error: %w\nstderr: %s", err, stderr.String())
	}

	// Parse output to check for success
	var result claudeResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		r.log.Warnw("could not parse claude output as JSON (may still have made changes)", "error", err)
	} else {
		r.log.Infow("claude finished", "session", result.SessionID)
	}

	return nil
}

func (r *Runner) gitHasChanges(ctx context.Context, repoDir string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

func (r *Runner) gitCommitAndPush(ctx context.Context, repoDir, branch string, issueNum int, title, owner, repo string) error {
	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", out, err)
	}

	// Commit
	commitMsg := commitMessageForIssue(title, issueNum)
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", commitMsg)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s: %w", out, err)
	}

	// Ensure remote URL has fresh token before push
	token, err := r.client.GetInstallationToken(ctx)
	if err != nil {
		return fmt.Errorf("get token for push: %w", err)
	}
	pushURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)
	cmd = exec.CommandContext(ctx, "git", "remote", "set-url", "origin", pushURL)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git set-url for push: %s: %w", out, err)
	}

	// Push (force-with-lease: safe for bot-owned vote-llm/* branches, handles remote branch conflicts)
	cmd = exec.CommandContext(ctx, "git", "push", "--force-with-lease", "-u", "origin", branch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %s: %w", out, err)
	}

	return nil
}

// prBodyForIssue returns a PR body that uses a GitHub closing keyword so that
// merging the PR automatically closes the linked issue.
func prBodyForIssue(issueNum int) string {
	return fmt.Sprintf("Closes #%d\n\n---\n*Automatically generated by vote-llm using Claude Code*", issueNum)
}

// commitMessageForIssue returns a commit message that references the issue with
// a GitHub closing keyword.
func commitMessageForIssue(title string, issueNum int) string {
	return fmt.Sprintf("feat: %s\n\nCloses #%d\n\nGenerated by vote-llm using Claude Code", title, issueNum)
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	slug := strings.ToLower(s)
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.TrimRight(slug, "-")
	}
	return slug
}

// resolveTimeout returns the agent timeout duration, using the per-repo override
// if available, otherwise falling back to the global default.
func resolveTimeout(cfg *store.RepoConfig) time.Duration {
	if cfg != nil && cfg.TimeoutMinutes != nil {
		return time.Duration(*cfg.TimeoutMinutes) * time.Minute
	}
	return time.Duration(config.AgentTimeoutMinutes) * time.Minute
}
