package githubapp

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidPrivateKey = errors.New("githubapp: invalid private key")

// ParsePrivateKey accepts either a raw PEM-encoded RSA private key or a base64-encoded PEM.
// Returns the parsed *rsa.PrivateKey.
func ParsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	pemBytes := []byte(raw)
	if block, _ := pem.Decode(pemBytes); block == nil {
		// Not raw PEM — try base64.
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: not PEM or base64", ErrInvalidPrivateKey)
		}
		pemBytes = decoded
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("%w: no PEM block", ErrInvalidPrivateKey)
	}

	// GitHub provides PKCS#1 ("RSA PRIVATE KEY"); also handle PKCS#8 for flexibility.
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: not an RSA key", ErrInvalidPrivateKey)
	}
	return rsaKey, nil
}

// GenerateAppJWT returns a short-lived (≤10 min) JWT signed with the app's private key,
// as required to authenticate as a GitHub App.
func GenerateAppJWT(appID int64, key *rsa.PrivateKey, now time.Time) (string, error) {
	// iat slightly in the past tolerates minor clock skew per GitHub's recommendation.
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-30 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
		Issuer:    strconv.FormatInt(appID, 10),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return tok.SignedString(key)
}
