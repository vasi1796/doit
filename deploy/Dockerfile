# Build the webhook binary
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/webhook .

# Final image with Docker CLI and git
FROM docker:27-cli
RUN apk add --no-cache git \
    && git config --global --add safe.directory /repo
COPY --from=builder /bin/webhook /usr/local/bin/webhook
ENTRYPOINT ["webhook"]
