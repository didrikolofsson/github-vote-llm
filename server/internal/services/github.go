package services

import (
	"context"
	"fmt"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gitauth_account "github.com/didrikolofsson/github-vote-llm/internal/gitauth/account"
	gitauth_client "github.com/didrikolofsson/github-vote-llm/internal/gitauth/client"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type GithubService struct {
	db            *pgxpool.Pool
	q             *store.Queries
	accountClient *gitauth_account.GithubAccountClient
	env           *config.Environment
	cfg           *oauth2.Config
}

type GithubServiceDeps struct {
	DB            *pgxpool.Pool
	Queries       *store.Queries
	Env           *config.Environment
	AccountClient *gitauth_account.GithubAccountClient
	Config        *oauth2.Config
}

func NewGithubService(deps GithubServiceDeps) *GithubService {
	return &GithubService{
		db:            deps.DB,
		q:             deps.Queries,
		accountClient: deps.AccountClient,
		env:           deps.Env,
		cfg:           deps.Config,
	}
}

func (s *GithubService) FrontendURL() string {
	return s.env.FRONTEND_URL
}

func (s *GithubService) CreateAuthURL(ctx context.Context, userID int64) (string, error) {
	authUrl, err := s.accountClient.CreateAuthURL(ctx, userID)
	if err != nil {
		return "", err
	}
	return authUrl, nil
}

func (s *GithubService) ExchangeCode(ctx context.Context, code, state string) (*oauth2.Token, error) {
	config := gitauth_client.NewOauthConfig(gitauth_client.OauthConfigParams{
		ClientID:     s.env.GITHUB_CLIENT_ID,
		ClientSecret: s.env.GITHUB_CLIENT_SECRET,
		Scopes:       []string{"user:email", "read:org"},
		RedirectURL:  fmt.Sprintf("%s/github/auth/callback", s.env.SERVER_URL),
	})

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *GithubService) VerifyAuthStateToken(ctx context.Context, token string) (gitauth_account.AuthStateClaims, error) {
	claims, err := s.accountClient.VerifyAuthStateToken(ctx, token)
	if err != nil {
		return gitauth_account.AuthStateClaims{}, err
	}
	return claims, nil
}

func (s *GithubService) UpsertGithubAccountTokenByUserID(ctx context.Context, userID int64, token *oauth2.Token) error {
	return s.accountClient.UpsertGithubAccountTokenByUserID(ctx, userID, token)
}

func (s *GithubService) GetAccountByUserID(ctx context.Context, userID int64) (string, error) {
	ts := gitauth_client.NewGithubTokenSource(gitauth_client.GithubTokenSourceDeps{
		DB:      s.db,
		Queries: s.q,
		UserID:  userID,
		Config:  s.cfg,
	})
	client := gitauth_client.NewGithubClient(ctx, ts)
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}
