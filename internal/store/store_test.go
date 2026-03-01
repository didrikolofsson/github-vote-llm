package store_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
)

func TestErrAlreadyExists_IsSentinel(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", store.ErrAlreadyExists)
	if !errors.Is(wrapped, store.ErrAlreadyExists) {
		t.Error("errors.Is should unwrap to store.ErrAlreadyExists")
	}
}
