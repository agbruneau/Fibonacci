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

# Default command (can be overridden)
ENTRYPOINT ["/app/fibcalc"]
CMD ["--help"]

# Example usage:
# Build: docker build -t fibcalc:latest .
# Run CLI: docker run --rm fibcalc:latest -n 1000 -algo fast
# Run server: docker run --rm -p 8080:8080 fibcalc:latest --server --port 8080
