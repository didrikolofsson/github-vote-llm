package services

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

type Services struct {
	UserService         UserService
	AuthService         AuthService
	OrganizationService OrganizationService
	GithubService       GithubService
	RepositoriesService RepositoriesService
	MembersService      MembersService
	RunService          RunService
	FeaturesService     FeaturesService
	PortalService       PortalService
}

func New(
	db *pgxpool.Pool,
	q *store.Queries,
	env *config.Environment,
	githubOAuthCfg *oauth2.Config,
	rc *river.Client[pgx.Tx],
) *Services {
	return &Services{
		UserService:         NewUserService(db, q),
		AuthService:         NewAuthService(db, q, env.JWT_SECRET),
		OrganizationService: NewOrganizationService(db, q),
		GithubService: NewGithubService(db, q, &GithubServiceConfigParams{
			TokenEncryptionKey: env.TOKEN_ENCRYPTION_KEY,
			Config:             *githubOAuthCfg,
		}),
		RepositoriesService: NewRepositoriesService(db, q),
		MembersService:      NewMembersService(q),
		RunService:          NewRunService(db, q, rc),
		FeaturesService:     NewFeaturesService(db, q, hub.NewHub()),
		PortalService:       NewPortalService(db, q),
	}
}
