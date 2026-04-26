package gitauth_app

import "github.com/jackc/pgx/v5/pgxpool"

type GithubAppClient struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *GithubAppClient {
	return &GithubAppClient{db: db}
}
