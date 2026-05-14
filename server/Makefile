.PHONY: dev build test generate mocks generate-pkce migrate-new migrate-up migrate-down lint lint-fix

dev:
	air

build:
	go build ./...

test:
	go test ./...

generate:
	sqlc generate -f db/sqlc.yaml

mocks:
	mockgen -destination=internal/store/mock_querier.go -package=store github.com/didrikolofsson/github-vote-llm/internal/store Querier
	mockgen -destination=internal/hub/mock_hub.go -package=hub github.com/didrikolofsson/github-vote-llm/internal/hub Hub
	mockgen -destination=internal/github/mock_app_client.go -package=github github.com/didrikolofsson/github-vote-llm/internal/github AppClientIface

generate-pkce:
	node scripts/pkce.js

migrate-new:
	migrate create -ext sql -dir db/migrations -seq $(name)

migrate-up:
	migrate -source file://db/migrations -database $$DATABASE_URL up

migrate-down:
	migrate -source file://db/migrations -database $$DATABASE_URL down 1

lint:
	go build ./...
	golangci-lint run

lint-fix:
	go build ./...
	golangci-lint run --fix

ngrok:
	ngrok http 8080
