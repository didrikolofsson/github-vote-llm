package githubapp

import "testing"

func TestGenerateNonce_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		n, err := GenerateNonce()
		if err != nil {
			t.Fatalf("generate nonce: %v", err)
		}
		if len(n) < 32 {
			t.Fatalf("nonce too short: %d", len(n))
		}
		if _, dup := seen[n]; dup {
			t.Fatalf("duplicate nonce: %q", n)
		}
		seen[n] = struct{}{}
	}
}
