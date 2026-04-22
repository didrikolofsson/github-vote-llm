package services

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type ServicesDeps struct {
	DB        *pgxpool.Pool
	Queries   *store.Queries
	Env       *config.Environment
	JobClient *river.Client[pgx.Tx]
	Hub       hub.Hub
}

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
	deps ServicesDeps,
) *Services {
	return &Services{
		UserService:         NewUserService(deps.DB, deps.Queries),
		AuthService:         NewAuthService(deps.DB, deps.Queries, deps.Env.JWT_SECRET),
		OrganizationService: NewOrganizationService(deps.DB, deps.Queries),
		GithubService:       NewGithubService(deps.DB, deps.Queries, deps.Env),
		RepositoriesService: NewRepositoriesService(deps.DB, deps.Queries),
		MembersService:      NewMembersService(deps.Queries),
		RunService:          NewRunService(deps.DB, deps.Queries, deps.JobClient),
		FeaturesService:     NewFeaturesService(deps.DB, deps.Queries, deps.Hub),
		PortalService:       NewPortalService(deps.DB, deps.Queries),
	}
}
