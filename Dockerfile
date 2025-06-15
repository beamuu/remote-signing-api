# ─────────────────────────────────────────────
# 1️⃣ Build stage
# ─────────────────────────────────────────────
FROM golang:1.24.2-alpine AS builder
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first for caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the entire source tree
COPY . .

# Build only the cmd/api package (the directory with main.go)
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o signer-api ./cmd/api

# ─────────────────────────────────────────────
# 2️⃣ Runtime stage
# ─────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates

RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /home/appuser

COPY --from=builder /app/signer-api .

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["./signer-api"]
