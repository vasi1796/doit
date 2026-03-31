# Build the webhook binary
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/webhook .

# Final image with Docker CLI and git
# Run as UID 1000 to match the host user so git pull doesn't create
# root-owned files. Added to GID 999 (ping group, reused for docker
# socket access since the host docker socket is GID 999).
FROM docker:27-cli
RUN apk add --no-cache git \
    && adduser -u 1000 -G ping -D deployer \
    && git config --global --add safe.directory /repo
COPY --from=builder /bin/webhook /usr/local/bin/webhook
USER deployer
ENTRYPOINT ["webhook"]
