# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gpu-discovery ./cmd/gpu-discovery/

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    lshw \
    pciutils \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 gpudiscovery && \
    adduser -u 1000 -G gpudiscovery -s /bin/sh -D gpudiscovery

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/gpu-discovery .

# Copy configuration
COPY --from=builder /app/config/config.yaml ./config/

# Create necessary directories
RUN mkdir -p /var/log/gpu-discovery && \
    chown -R gpudiscovery:gpudiscovery /app /var/log/gpu-discovery

# Switch to non-root user
USER gpudiscovery

# Expose ports
EXPOSE 50051 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./gpu-discovery discover --output=json > /dev/null || exit 1

# Default command
CMD ["./gpu-discovery", "serve", "--config=/app/config/config.yaml"]