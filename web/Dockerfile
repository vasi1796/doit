# =============================================================================
# Multi-stage build for the DoIt React frontend
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build
# ---------------------------------------------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

# Cache dependency install
COPY package.json package-lock.json ./
RUN npm ci

# Copy source and build
COPY . .
RUN npm run build

# ---------------------------------------------------------------------------
# Stage 2: Output — just the built assets
# The resulting image contains only the static files in /app/dist.
# In production, these are volume-mounted or copied into the Caddy container.
# ---------------------------------------------------------------------------
FROM alpine:3.20

COPY --from=builder /app/dist /app/dist
