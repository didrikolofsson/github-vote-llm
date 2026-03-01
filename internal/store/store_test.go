package store_test

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrAlreadyExists_IsSentinel(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", errors.New("execution already exists for this issue"))
	if !errors.Is(wrapped, errors.New("execution already exists for this issue")) {
		t.Error("errors.Is should unwrap to execution already exists for this issue")
	}
}
