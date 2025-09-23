# syntax=docker/dockerfile:1.7
# --- Build stage ---
FROM golang:1.24-alpine AS builder
WORKDIR /app
# Install build deps
RUN apk add --no-cache git build-base bash
# Pre-copy go.mod for caching
COPY go.mod ./
# Tidy (in case) and download modules
RUN go mod download
# Copy source
COPY . .
# Generate swagger docs if swag is present (ignore failure)
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6 && \
    if [ -f Makefile ]; then make swag || true; fi
# Build binary (static)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o echome-api ./cmd/api

# --- Runtime stage ---
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
# Copy config example as default config (can be overridden by mount / env)
COPY --from=builder /app/config/etc ./config/etc
COPY --from=builder /app/echome-api ./echome-api
USER nonroot:nonroot
EXPOSE 8081
ENTRYPOINT ["/app/echome-api","-f","config/etc/config.yaml"]
