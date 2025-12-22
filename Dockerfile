# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# -ldflags="-s -w" removes symbol table and DWARF generation for smaller binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o fibcalc ./cmd/fibcalc

# Stage 2: Create minimal production image
FROM alpine:3.19

# Create a non-root user
RUN adduser -D -g '' appuser

# Install CA certificates for HTTPS support
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/fibcalc .

# Use non-root user
USER appuser

# Expose default port
EXPOSE 8080

# Define entrypoint
ENTRYPOINT ["./fibcalc"]

# Default arguments
CMD ["--server", "--port", "8080"]
