package githubapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const SignatureHeader = "X-Hub-Signature-256"
const EventHeader = "X-GitHub-Event"
const DeliveryHeader = "X-GitHub-Delivery"

// VerifySignature checks the X-Hub-Signature-256 header against the payload using the shared secret.
// sigHeader must include the "sha256=" prefix as GitHub sends it.
// Returns true only if the signature is present, well-formed, and matches (constant-time).
func VerifySignature(payload []byte, sigHeader, secret string) bool {
	if sigHeader == "" || secret == "" {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	sigHex := sigHeader[len(prefix):]
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	return hmac.Equal(sig, expected)
}
