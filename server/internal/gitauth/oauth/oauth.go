package gitauth_oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

type GithubOauthClient struct {
	q         *store.Queries
	clientID  string
	jwtSecret string
}

func New(q *store.Queries, clientID string, jwtSecret string) *GithubOauthClient {
	return &GithubOauthClient{q: q, clientID: clientID, jwtSecret: jwtSecret}
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type authStateClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func createAuthStateToken(ctx context.Context, userID int64, jwtSecret string) (string, error) {
	claims := authStateClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func verifyAuthStateToken(ctx context.Context, token string, jwtSecret string) (authStateClaims, error) {
	claims := authStateClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return authStateClaims{}, err
	}
	return claims, nil
}

func (c *GithubOauthClient) CreateAuthURL(ctx context.Context, userID int64) (string, error) {
	state, err := createAuthStateToken(ctx, userID, c.jwtSecret)
	if err != nil {
		return "", err
	}
	authUrl := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s", c.clientID, state)
	return authUrl, nil
}
