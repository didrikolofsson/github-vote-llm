package agent

import (
	"strings"
	"testing"
)

func TestPRBodyForIssue_ContainsClosingKeyword(t *testing.T) {
	body := prBodyForIssue(42)

	if !strings.Contains(body, "Closes #42") {
		t.Errorf("PR body should contain closing keyword 'Closes #42', got: %s", body)
	}
}

func TestPRBodyForIssue_DifferentIssueNumbers(t *testing.T) {
	tests := []struct {
		issueNum int
		want     string
	}{
		{1, "Closes #1"},
		{100, "Closes #100"},
		{9999, "Closes #9999"},
	}
	for _, tt := range tests {
		body := prBodyForIssue(tt.issueNum)
		if !strings.Contains(body, tt.want) {
			t.Errorf("prBodyForIssue(%d) = %q, want it to contain %q", tt.issueNum, body, tt.want)
		}
	}
}

func TestCommitMessageForIssue_ContainsClosingKeyword(t *testing.T) {
	msg := commitMessageForIssue("Add dark mode", 10)

	if !strings.Contains(msg, "Closes #10") {
		t.Errorf("commit message should contain closing keyword 'Closes #10', got: %s", msg)
	}
}

func TestCommitMessageForIssue_ContainsTitle(t *testing.T) {
	msg := commitMessageForIssue("Add dark mode", 10)

	if !strings.Contains(msg, "feat: Add dark mode") {
		t.Errorf("commit message should contain title, got: %s", msg)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add dark mode", "add-dark-mode"},
		{"Fix Bug #123", "fix-bug-123"},
		{"Hello World!!!", "hello-world"},
		{"", ""},
		{"a", "a"},
		{strings.Repeat("a", 50), strings.Repeat("a", 40)},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
