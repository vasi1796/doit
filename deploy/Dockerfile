# Build the webhook binary
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/webhook .

# Final image with Docker CLI and git
# Run as UID 1000 to match the host user so git pull doesn't create
# root-owned files. GID 999 grants access to the Docker socket.
FROM docker:27-cli
RUN apk add --no-cache git \
    && addgroup -g 999 dockerhost \
    && adduser -u 1000 -G dockerhost -D deployer \
    && git config --global --add safe.directory /repo
COPY --from=builder /bin/webhook /usr/local/bin/webhook
USER deployer
ENTRYPOINT ["webhook"]
