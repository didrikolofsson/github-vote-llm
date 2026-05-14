package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercases", "MyOrg", "myorg"},
		{"replaces spaces with hyphens", "My Organization", "my-organization"},
		{"strips leading and trailing hyphens", "  My Org  ", "my-org"},
		{"collapses multiple separators", "My---Org", "my-org"},
		{"removes special characters", "My @Org!", "my-org"},
		{"handles numeric names", "Org 42", "org-42"},
		{"preserves existing slug", "already-slugified", "already-slugified"},
		{"empty string", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, slugify(tc.input))
		})
	}
}
