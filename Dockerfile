FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o vote-llm cmd/main/main.go

FROM node:22-alpine AS runtime

ENV GIN_MODE=release
ENV PORT=8080

RUN apk add --no-cache git && \
    npm install -g @anthropic-ai/claude-code && \
    npm cache clean --force

COPY --from=builder /app/vote-llm /usr/local/bin/vote-llm

CMD ["/usr/local/bin/vote-llm"]
