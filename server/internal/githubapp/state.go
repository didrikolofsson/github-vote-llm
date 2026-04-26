package githubapp

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const StateTTL = 10 * time.Minute

var (
	ErrInvalidState      = errors.New("githubapp: invalid or expired install state")
	ErrStateUserMismatch = errors.New("githubapp: install state user mismatch")
)

type InstallStateClaims struct {
	OrgID  int64 `json:"org_id"`
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// CreateInstallState creates a state token for a given organization ID.
func CreateInstallStateToken(ctx context.Context, orgID, userID int64, jwtSecret string) (*jwt.Token, error) {
	now := time.Now()
	claims := InstallStateClaims{
		OrgID:  orgID,
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(StateTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token, nil
}

func VerifyInstallStateToken(ctx context.Context, token string, jwtSecret string) (*InstallStateClaims, error) {
	claims := &InstallStateClaims{}
	parsedToken, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !parsedToken.Valid {
		return nil, ErrInvalidState
	}
	return claims, nil
}
