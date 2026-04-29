package gitauth_account

import (
	"context"
	"fmt"
	"time"

	gitauth_client "github.com/didrikolofsson/github-vote-llm/internal/gitauth/client"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
)

type GithubAccountClient struct {
	q         *store.Queries
	clientID  string
	jwtSecret string
}

func New(q *store.Queries, clientID string, jwtSecret string) *GithubAccountClient {
	return &GithubAccountClient{q: q, clientID: clientID, jwtSecret: jwtSecret}
}

type AuthStateClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func (c *GithubAccountClient) CreateAuthURL(ctx context.Context, userID int64) (string, error) {
	claims := AuthStateClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	state, err := token.SignedString([]byte(c.jwtSecret))
	if err != nil {
		return "", err
	}
	authUrl := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s", c.clientID, state)
	return authUrl, nil
}

func (c *GithubAccountClient) VerifyAuthStateToken(ctx context.Context, token string) (AuthStateClaims, error) {
	claims := AuthStateClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return []byte(c.jwtSecret), nil
	})
	if err != nil {
		return AuthStateClaims{}, err
	}
	return claims, nil
}

func (c *GithubAccountClient) UpsertGithubAccountTokenByUserID(ctx context.Context, userID int64, token *oauth2.Token) error {
	accessTokenExpiry := gitauth_client.CalculateAccessTokenExpiry(time.Duration(token.ExpiresIn))
	_, err := c.q.UpsertGithubAccountTokenByUserID(ctx, store.UpsertGithubAccountTokenByUserIDParams{
		UserID:               userID,
		AccessToken:          token.AccessToken,
		AccessTokenExpiresAt: pgtype.Timestamptz{Time: accessTokenExpiry, Valid: true},
		RefreshToken:         token.RefreshToken,
	})
	if err != nil {
		return err
	}
	return nil
}
