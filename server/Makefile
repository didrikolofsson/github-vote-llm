dev:
	trap 'kill 0' EXIT; \
	ngrok http 8080 --log=stdout > /dev/null & \
	air

build:
	go build ./...

test:
	go test ./...

generate:
	sqlc generate -f db/sqlc.yaml

migrate-new:
	migrate create -ext sql -dir db/migrations -seq $(name)

migrate-up:
	migrate -source file://db/migrations -database $$DATABASE_URL up

migrate-down:
	migrate -source file://db/migrations -database $$DATABASE_URL down 1

lint-openapi:
	npx --yes @redocly/cli@latest lint openapi.yaml
