package githubapp

import (
	"context"
	"errors"
	"testing"
)

func TestInstallStateToken_RoundTrip(t *testing.T) {
	ctx := context.Background()
	token, err := CreateInstallStateToken(ctx, 123, 456, "secret")
	if err != nil {
		t.Fatalf("create install state token: %v", err)
	}

	claims, err := VerifyInstallStateToken(ctx, token, "secret")
	if err != nil {
		t.Fatalf("verify install state token: %v", err)
	}
	if claims.OrgID != 123 {
		t.Fatalf("org id = %d, want 123", claims.OrgID)
	}
	if claims.UserID != 456 {
		t.Fatalf("user id = %d, want 456", claims.UserID)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expires at is nil")
	}
}

func TestInstallStateToken_InvalidSecret(t *testing.T) {
	ctx := context.Background()
	token, err := CreateInstallStateToken(ctx, 123, 456, "secret")
	if err != nil {
		t.Fatalf("create install state token: %v", err)
	}

	_, err = VerifyInstallStateToken(ctx, token, "other-secret")
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("verify error = %v, want %v", err, ErrInvalidState)
	}
}
