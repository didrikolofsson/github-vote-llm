package githubapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	secret := "topsecret"
	if !VerifySignature(payload, sign(payload, secret), secret) {
		t.Fatal("expected valid signature to verify")
	}
}

func TestVerifySignature_TamperedPayload(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	secret := "topsecret"
	sig := sign(payload, secret)

	tampered := []byte(`{"hello":"worlf"}`)
	if VerifySignature(tampered, sig, secret) {
		t.Fatal("tampered payload should fail")
	}
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	sig := sign(payload, "topsecret")
	if VerifySignature(payload, sig, "othersecret") {
		t.Fatal("wrong secret should fail")
	}
}

func TestVerifySignature_MissingHeader(t *testing.T) {
	if VerifySignature([]byte(`x`), "", "secret") {
		t.Fatal("empty header should fail")
	}
}

func TestVerifySignature_MissingSecret(t *testing.T) {
	if VerifySignature([]byte(`x`), "sha256=deadbeef", "") {
		t.Fatal("empty secret should fail")
	}
}

func TestVerifySignature_WrongPrefix(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	secret := "topsecret"
	sig := sign(payload, secret)
	// Strip "sha256=" — should no longer be accepted.
	raw := sig[len("sha256="):]
	if VerifySignature(payload, raw, secret) {
		t.Fatal("signature without sha256= prefix should fail")
	}
	// And with a different algo prefix.
	if VerifySignature(payload, "sha1="+raw, secret) {
		t.Fatal("wrong algo prefix should fail")
	}
}

func TestVerifySignature_MalformedHex(t *testing.T) {
	if VerifySignature([]byte(`x`), "sha256=zzzz", "secret") {
		t.Fatal("non-hex signature should fail")
	}
}
