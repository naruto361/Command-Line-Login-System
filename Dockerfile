# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /osto ./cmd/cli

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /osto /app/osto

# Runtime config is set in docker-compose.yml (or via -e flags).
# Only defaults needed for a plain `docker run` without compose:
ENV DATABASE_PATH=/app/data/app.db

VOLUME ["/app/data"]

ENTRYPOINT ["/app/osto"]
