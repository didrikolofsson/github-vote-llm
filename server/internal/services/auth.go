package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/helpers"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidAuthCode     = errors.New("invalid or expired authorization code")
	ErrInvalidPKCE         = errors.New("invalid code verifier")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

type AuthService interface {
	Authorize(ctx context.Context, email, password, codeChallenge, redirectURI string) (string, error)
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (accessToken, refreshToken string, err error)
	Refresh(ctx context.Context, refreshToken string) (accessToken string, err error)
	Revoke(ctx context.Context, refreshToken string) error
}

type AuthServiceImpl struct {
	db        *pgxpool.Pool
	q         *store.Queries
	jwtSecret []byte
}

func NewAuthService(db *pgxpool.Pool, q *store.Queries, jwtSecret string) AuthService {
	return &AuthServiceImpl{db: db, q: q, jwtSecret: []byte(jwtSecret)}
}

func (s *AuthServiceImpl) Authorize(ctx context.Context, email, password, codeChallenge, redirectURI string) (string, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	user, err := qtx.GetUserByEmailWithPassword(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if !helpers.VerifyPassword(user.Password, password) {
		return "", ErrInvalidCredentials
	}

	code, err := generateRandomToken(32)
	if err != nil {
		return "", err
	}

	_, err = qtx.CreateAuthCode(ctx, store.CreateAuthCodeParams{
		Code:          code,
		UserID:        user.ID,
		CodeChallenge: codeChallenge,
		RedirectUri:   redirectURI,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(config.AuthCodeTTL), Valid: true},
	})
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return code, nil
}

func (s *AuthServiceImpl) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (string, string, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	authCode, err := qtx.GetAuthCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrInvalidAuthCode
	}
	if err != nil {
		return "", "", err
	}
	if authCode.Used || time.Now().After(authCode.ExpiresAt.Time) || authCode.RedirectUri != redirectURI {
		return "", "", ErrInvalidAuthCode
	}
	if !verifyPKCE(authCode.CodeChallenge, codeVerifier) {
		return "", "", ErrInvalidPKCE
	}

	if err := qtx.UseAuthCode(ctx, authCode.ID); err != nil {
		return "", "", err
	}

	user, err := qtx.GetUserByID(ctx, authCode.UserID)
	if err != nil {
		return "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}

	accessToken, err := s.issueAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := s.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthServiceImpl) Refresh(ctx context.Context, refreshToken string) (string, error) {
	rt, err := s.q.GetRefreshToken(ctx, hashToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidRefreshToken
	}
	if err != nil {
		return "", err
	}
	if time.Now().After(rt.ExpiresAt.Time) {
		return "", ErrInvalidRefreshToken
	}

	user, err := s.q.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return "", err
	}

	return s.issueAccessToken(user.ID, user.Email)
}

func (s *AuthServiceImpl) Revoke(ctx context.Context, refreshToken string) error {
	return s.q.DeleteRefreshToken(ctx, hashToken(refreshToken))
}

func (s *AuthServiceImpl) issueAccessToken(userID int64, email string) (string, error) {
	claims := dtos.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AccessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthServiceImpl) issueRefreshToken(ctx context.Context, userID int64) (string, error) {
	raw, err := generateRandomToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.q.CreateRefreshToken(ctx, store.CreateRefreshTokenParams{
		TokenHash: hashToken(raw),
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(config.RefreshTokenTTL), Valid: true},
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func verifyPKCE(challenge, verifier string) bool {
	h := sha256.New()
	h.Write([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return computed == challenge
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
