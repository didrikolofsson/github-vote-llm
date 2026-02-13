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
	gogithub "github.com/google/go-github/v68/github"
)

// Runner orchestrates Claude Code to implement approved features.
type Runner struct {
	client *gh.Client
	cfg    *config.AgentConfig
	log    *logger.Logger
}

// NewRunner creates a new agent runner.
func NewRunner(client *gh.Client, cfg *config.AgentConfig, log *logger.Logger) *Runner {
	return &Runner{client: client, cfg: cfg, log: log.Named("agent")}
}

// claudeResult represents the JSON output from claude -p.
type claudeResult struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

// Run implements a feature request: clones repo, runs Claude Code, creates a PR.
func (r *Runner) Run(ctx context.Context, owner, repo string, issue *gogithub.Issue, repoCfg *config.RepoConfig) {
	issueNum := issue.GetNumber()
	r.log.Infow("starting work on issue", "issue", issueNum, "repo", owner+"/"+repo)

	// Mark issue as in-progress
	if err := r.client.AddLabel(ctx, owner, repo, issueNum, repoCfg.Labels.InProgress); err != nil {
		r.log.Warnw("failed to add in-progress label", "error", err)
	}

	// Remove approved label to prevent re-triggering
	if err := r.client.RemoveLabel(ctx, owner, repo, issueNum, repoCfg.Labels.Approved); err != nil {
		r.log.Warnw("failed to remove approved label", "error", err)
	}

	// Run with timeout
	timeout := time.Duration(r.cfg.TimeoutMinutes) * time.Minute
	implCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prURL, err := r.implement(implCtx, owner, repo, issue)
	if err != nil {
		r.log.Errorw("failed to implement issue", "issue", issueNum, "error", err)
		comment := fmt.Sprintf("**vote-llm**: Failed to implement this feature.\n\n```\n%s\n```", err)
		if cErr := r.client.CreateComment(ctx, owner, repo, issueNum, comment); cErr != nil {
			r.log.Warnw("failed to comment on issue", "error", cErr)
		}
		if err := r.client.RemoveLabel(ctx, owner, repo, issueNum, repoCfg.Labels.InProgress); err != nil {
			r.log.Warnw("failed to remove in-progress label", "error", err)
		}
		return
	}

	// Mark issue as done and comment with PR link
	if err := r.client.AddLabel(ctx, owner, repo, issueNum, repoCfg.Labels.Done); err != nil {
		r.log.Warnw("failed to add done label", "error", err)
	}
	if err := r.client.RemoveLabel(ctx, owner, repo, issueNum, repoCfg.Labels.InProgress); err != nil {
		r.log.Warnw("failed to remove in-progress label", "error", err)
	}

	comment := fmt.Sprintf("**vote-llm**: A PR has been created for this feature request: %s\n\nPlease review the changes.", prURL)
	if err := r.client.CreateComment(ctx, owner, repo, issueNum, comment); err != nil {
		r.log.Warnw("failed to comment on issue", "error", err)
	}

	r.log.Infow("successfully created PR", "issue", issueNum, "pr", prURL)
}

func (r *Runner) implement(ctx context.Context, owner, repo string, issue *gogithub.Issue) (string, error) {
	// Prepare workspace
	workDir := filepath.Join(r.cfg.WorkspaceDir, owner, repo)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}

	repoDir := filepath.Join(workDir, "repo")

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
	branchName := fmt.Sprintf("vote-llm/issue-%d-%s", issue.GetNumber(), slugify(issue.GetTitle()))
	if err := r.gitCheckoutNewBranch(ctx, repoDir, defaultBranch, branchName); err != nil {
		return "", fmt.Errorf("create branch: %w", err)
	}

	// Build prompt and run Claude Code
	prompt := r.buildPrompt(issue, owner, repo)
	if err := r.runClaude(ctx, repoDir, prompt); err != nil {
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
	if err := r.gitCommitAndPush(ctx, repoDir, branchName, issue.GetNumber(), issue.GetTitle()); err != nil {
		return "", fmt.Errorf("commit and push: %w", err)
	}

	// Create PR (or find existing one for this branch)
	prTitle := fmt.Sprintf("feat: %s (issue #%d)", issue.GetTitle(), issue.GetNumber())
	prBody := prBodyForIssue(issue.GetNumber())
	pr, err := r.client.CreatePullRequest(ctx, owner, repo, branchName, defaultBranch, prTitle, prBody)
	if err != nil {
		// If PR creation failed, check if one already exists for this branch
		existing, findErr := r.client.FindPullRequestByHead(ctx, owner, repo, branchName)
		if findErr != nil {
			return "", fmt.Errorf("create PR: %w (also failed to find existing: %v)", err, findErr)
		}
		if existing != nil {
			r.log.Infow("PR already exists for branch, reusing", "branch", branchName, "pr", existing.GetHTMLURL())
			return existing.GetHTMLURL(), nil
		}
		return "", fmt.Errorf("create PR: %w", err)
	}

	return pr.GetHTMLURL(), nil
}

func (r *Runner) cloneOrResetRepo(ctx context.Context, owner, repo, repoDir string) error {
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		// Repo exists, pull latest
		cmd := exec.CommandContext(ctx, "git", "fetch", "--all", "--prune")
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git fetch: %s: %w", out, err)
		}
		return nil
	}

	// Clone fresh
	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, repoDir)
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

func (r *Runner) runClaude(ctx context.Context, repoDir, prompt string) error {
	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--allowedTools", strings.Join(r.cfg.AllowedTools, ","),
		"--max-turns", strconv.Itoa(r.cfg.MaxTurns),
		"--max-budget-usd", fmt.Sprintf("%.2f", r.cfg.MaxBudgetUSD),
		"--no-session-persistence",
	}

	cmd := exec.CommandContext(ctx, r.cfg.Command, args...)
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

func (r *Runner) gitCommitAndPush(ctx context.Context, repoDir, branch string, issueNum int, title string) error {
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
