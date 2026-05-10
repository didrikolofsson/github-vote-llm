.PHONY: dev build test generate generate-pkce migrate-new migrate-up migrate-down lint lint-fix

dev:
	air

build:
	go build ./...

test:
	go test ./...

generate:
	sqlc generate -f db/sqlc.yaml

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
