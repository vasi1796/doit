# =============================================================================
# Multi-stage build for the DoIt Go backend
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

ARG CMD_PATH=cmd/api

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/app ./${CMD_PATH}

# ---------------------------------------------------------------------------
# Stage 2: Final minimal image
# ---------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

# Non-root user
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /bin/app /usr/local/bin/app
COPY --from=builder /src/migrations /app/migrations

USER app

EXPOSE 8080

ENTRYPOINT ["app"]
