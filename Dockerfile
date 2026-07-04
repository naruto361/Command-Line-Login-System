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

ENV DATABASE_PATH=/app/data/app.db \
    MAX_FAILED_LOGIN_ATTEMPTS=5 \
    MAX_TOTP_ATTEMPTS=3 \
    SESSION_TIMEOUT_MINUTES=30 \
    SESSION_WARNING_MINUTES=5 \
    PASSWORD_RESET_TOKEN_EXPIRY_MINUTES=60 \
    DEV_MODE=true \
    APP_NAME=OSTO

VOLUME ["/app/data"]

ENTRYPOINT ["/app/osto"]
