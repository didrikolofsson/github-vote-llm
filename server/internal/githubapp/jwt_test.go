package githubapp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func genTestKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pemBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	return key, string(pem.EncodeToMemory(pemBlock))
}

func TestParsePrivateKey_RawPEM(t *testing.T) {
	_, raw := genTestKey(t)
	if _, err := ParsePrivateKey(raw); err != nil {
		t.Fatalf("parse raw PEM: %v", err)
	}
}

func TestParsePrivateKey_Base64PEM(t *testing.T) {
	_, raw := genTestKey(t)
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	if _, err := ParsePrivateKey(encoded); err != nil {
		t.Fatalf("parse base64 PEM: %v", err)
	}
}

func TestParsePrivateKey_Invalid(t *testing.T) {
	if _, err := ParsePrivateKey("not a key"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestGenerateAppJWT(t *testing.T) {
	key, _ := genTestKey(t)
	now := time.Now()

	tokenStr, err := GenerateAppJWT(42, key, now)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	parsed, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("jwt not valid")
	}

	claims := parsed.Claims.(*jwt.RegisteredClaims)
	if claims.Issuer != strconv.FormatInt(42, 10) {
		t.Fatalf("issuer = %q, want 42", claims.Issuer)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("missing exp")
	}
	if exp := claims.ExpiresAt.Sub(now); exp > 10*time.Minute {
		t.Fatalf("exp is %v, must be ≤ 10 min", exp)
	}
	if iat := now.Sub(claims.IssuedAt.Time); iat < 0 {
		t.Fatalf("iat is in the future relative to now")
	}
}
