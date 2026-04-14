package services

import (
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
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
) *Services {
	return &Services{
		UserService:         NewUserService(db, q),
		AuthService:         NewAuthService(db, q, env.JWT_SECRET),
		OrganizationService: NewOrganizationService(db, q),
		GithubService:       NewGithubService(db, q, env),
		RepositoriesService: NewRepositoriesService(db, q),
		MembersService:      NewMembersService(q),
		RunService:          NewRunService(db, q),
		FeaturesService:     NewFeaturesService(db, q, hub.NewHub()),
		PortalService:       NewPortalService(db, q),
	}
}
