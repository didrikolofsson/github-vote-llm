dev:
	trap 'kill 0' EXIT; \
	ngrok http 8080 --log=stdout > /dev/null & \
	air

build:
	go build ./cmd/main/main.go

generate:
	sqlc generate -f db/sqlc.yaml

migrate-up:
	migrate -source file://db/migrations -database $$DATABASE_URL up

migrate-down:
	migrate -source file://db/migrations -database $$DATABASE_URL down 1