dev:
	trap 'kill 0' EXIT; \
	ngrok http 8080 --log=stdout > /dev/null & \
	cd server && air

client-install:
	cd client && npm install

client-generate:
	cd client && npm run generate

client-build: client-generate
	cd client && npm run build

build: client-build
	cd server && go build ./...

test:
	cd server && go test ./...

generate:
	sqlc generate -f server/db/sqlc.yaml

migrate-new:
	migrate create -ext sql -dir server/db/migrations -seq $(name)

migrate-up:
	migrate -source file://server/db/migrations -database $$DATABASE_URL up

migrate-down:
	migrate -source file://server/db/migrations -database $$DATABASE_URL down 1

lint-openapi:
	npx --yes @redocly/cli@latest lint server/openapi.yaml
