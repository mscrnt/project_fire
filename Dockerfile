# Multi-stage build for F.I.R.E.
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" \
    -o bench ./cmd/fire

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    stress-ng \
    fio \
    bash \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 fire && \
    adduser -D -u 1000 -G fire fire

# Copy binary from builder
COPY --from=builder /build/bench /usr/local/bin/bench

# Create working directory
WORKDIR /home/fire
RUN chown -R fire:fire /home/fire

# Switch to non-root user
USER fire

# Set environment
ENV HOME=/home/fire
ENV FIRE_DATA_DIR=/home/fire/data

# Expose agent port (if running in agent mode)
EXPOSE 2223

# Default command
ENTRYPOINT ["bench"]
CMD ["--help"]