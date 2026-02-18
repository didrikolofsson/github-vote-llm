dev:
	trap 'kill 0' EXIT; \
	ngrok http 8080 --log=stdout > /dev/null & \
	air

build:
	go build ./cmd/main/main.go