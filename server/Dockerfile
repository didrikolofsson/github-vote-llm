# Stage 1: Build frontend
# Generates TypeScript SDK from openapi.yaml, then builds the Vite app.
# Output lands in /build/server/web/dist/ (per client/vite.config.ts outDir).
FROM node:22-alpine AS frontend
WORKDIR /build
COPY client/package*.json ./client/
RUN cd client && npm ci
COPY client/ ./client/
COPY server/openapi.yaml ./server/
RUN cd client && npm run build

# Stage 2: Build Go binary
# Embeds the frontend dist via go:embed (server/web/embed.go).
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download && go mod verify
COPY server/ ./
COPY --from=frontend /build/server/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o vote-llm ./cmd/main/main.go

# Stage 3: Runtime
# Node is required for the claude CLI; git is required by the agent runner.
FROM node:22-alpine
ENV GIN_MODE=release
ENV PORT=8080
RUN apk add --no-cache git && \
    npm install -g @anthropic-ai/claude-code && \
    npm cache clean --force
COPY --from=builder /app/vote-llm /usr/local/bin/vote-llm
CMD ["/usr/local/bin/vote-llm"]
