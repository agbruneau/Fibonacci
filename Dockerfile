# Multi-stage build for optimal image size
# ========================================

# Stage 1: Build
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/fibcalc \
    ./cmd/fibcalc

# Stage 2: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/fibcalc .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port for server mode
EXPOSE 8080

# ─────────────────────────────────────────────────────────────────────────────
# Environment Variables (all optional, CLI flags take precedence)
# ─────────────────────────────────────────────────────────────────────────────
# FIBCALC_N             - Index of Fibonacci number (default: 250000000)
# FIBCALC_ALGO          - Algorithm: fast, matrix, fft, all (default: all)
# FIBCALC_PORT          - Server port (default: 8080)
# FIBCALC_TIMEOUT       - Calculation timeout, e.g., "5m", "30s" (default: 5m)
# FIBCALC_THRESHOLD     - Parallelism threshold in bits (default: 4096)
# FIBCALC_FFT_THRESHOLD - FFT multiplication threshold (default: 1000000)
# FIBCALC_SERVER        - Enable server mode: true/false (default: false)
# FIBCALC_JSON          - JSON output: true/false (default: false)
# FIBCALC_VERBOSE       - Verbose output: true/false (default: false)
# FIBCALC_QUIET         - Quiet mode: true/false (default: false)

# Default command (can be overridden)
ENTRYPOINT ["/app/fibcalc"]
CMD ["--help"]

# ═══════════════════════════════════════════════════════════════════════════════
# EXAMPLE USAGE
# ═══════════════════════════════════════════════════════════════════════════════
#
# BUILD:
#   docker build -t fibcalc:latest .
#
# RUN CLI (with flags):
#   docker run --rm fibcalc:latest -n 1000 -algo fast
#
# RUN CLI (with environment variables):
#   docker run --rm -e FIBCALC_N=1000 -e FIBCALC_ALGO=fast fibcalc:latest
#
# RUN SERVER (with flags):
#   docker run --rm -p 8080:8080 fibcalc:latest --server --port 8080
#
# RUN SERVER (with environment variables):
#   docker run --rm -p 8080:8080 \
#     -e FIBCALC_SERVER=true \
#     -e FIBCALC_PORT=8080 \
#     -e FIBCALC_THRESHOLD=8192 \
#     fibcalc:latest
#
# DOCKER COMPOSE EXAMPLE:
#   services:
#     fibcalc:
#       image: fibcalc:latest
#       ports:
#         - "8080:8080"
#       environment:
#         - FIBCALC_SERVER=true
#         - FIBCALC_PORT=8080
#         - FIBCALC_THRESHOLD=8192
#         - FIBCALC_FFT_THRESHOLD=500000
#         - FIBCALC_TIMEOUT=10m
