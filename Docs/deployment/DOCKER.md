# Docker Deployment Guide

> **Version**: 1.0.0  
> **Last Updated**: November 2025

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+ (optional)
- 512 MB RAM minimum (2 GB recommended for large calculations)

## Building the Image

### Standard Build

```bash
# Build the image with default tag
docker build -t fibcalc:latest .

# Build with a specific version tag
docker build -t fibcalc:1.0.0 .
```

### Build with Arguments

```bash
# Build with injected version information
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t fibcalc:1.0.0 .
```

### Multi-architecture

```bash
# Build for multiple architectures (AMD64 + ARM64)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t fibcalc:1.0.0 \
  --push .
```

## Running

### CLI Mode

```bash
# Simple calculation
docker run --rm fibcalc:latest -n 1000 -algo fast -d

# Calculation with all algorithms
docker run --rm fibcalc:latest -n 10000 -algo all

# JSON output
docker run --rm fibcalc:latest -n 1000 --json

# Calibration
docker run --rm fibcalc:latest --calibrate
```

### Server Mode

```bash
# Start the server on port 8080
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080

# Verify the server is running
curl http://localhost:8080/health

# View logs
docker logs -f fibcalc-server

# Stop the server
docker stop fibcalc-server
docker rm fibcalc-server
```

### Advanced Options

```bash
# With auto-calibration at startup
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080 --auto-calibrate

# With resource limits
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  --memory=2g \
  --cpus=4 \
  fibcalc:latest --server --port 8080

# With custom timeout
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080 --timeout 10m
```

## Docker Compose

### Simple Configuration

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  fibcalc:
    build: .
    image: fibcalc:latest
    container_name: fibcalc-server
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080", "--auto-calibrate"]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

```bash
# Start
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Configuration with Monitoring

```yaml
version: '3.8'

services:
  fibcalc:
    build: .
    image: fibcalc:latest
    container_name: fibcalc-server
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080", "--auto-calibrate"]
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 512M
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped
    networks:
      - monitoring

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    depends_on:
      - fibcalc
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana-data:/var/lib/grafana
    depends_on:
      - prometheus
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge

volumes:
  prometheus-data:
  grafana-data:
```

Corresponding `prometheus.yml` file:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'fibcalc'
    static_configs:
      - targets: ['fibcalc:8080']
    metrics_path: '/metrics'
```

## Dockerfile Explained

```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder

# Build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Cache Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Optimised build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/fibcalc \
    ./cmd/fibcalc

# Stage 2: Runtime (minimal image)
FROM alpine:latest

# Certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Non-root user (security)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/fibcalc .

# Permissions
RUN chown -R appuser:appgroup /app

# Run as non-root
USER appuser

# Exposed port
EXPOSE 8080

# Entry point
ENTRYPOINT ["/app/fibcalc"]
CMD ["--help"]
```

## Best Practices

### 1. Image Size

The final image is approximately 15 MB thanks to:
- Multi-stage build
- Alpine base image
- Static Go binary (CGO_ENABLED=0)
- Symbol stripping (-ldflags="-s -w")

### 2. Security

- Non-root user (`appuser`)
- Minimal base image (Alpine)
- No interactive shell needed
- Built-in healthcheck

### 3. Performance

```bash
# Resource recommendations
# - Small usage: 1 CPU, 512 MB RAM
# - Medium usage: 2 CPUs, 1 GB RAM
# - Large calculations: 4+ CPUs, 2+ GB RAM

docker run -d \
  --cpus=4 \
  --memory=2g \
  --memory-swap=2g \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080
```

### 4. Calibration Persistence

```bash
# Mount a volume to persist the calibration profile
docker run -d \
  -v fibcalc-data:/home/appuser \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080 --auto-calibrate
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs fibcalc-server

# Run in interactive mode
docker run --rm -it fibcalc:latest --help
```

### Degraded Performance

```bash
# Check resources
docker stats fibcalc-server

# Increase limits
docker update --cpus=8 --memory=4g fibcalc-server
```

### Port Already in Use

```bash
# Use a different port
docker run -d -p 9090:8080 fibcalc:latest --server --port 8080
```

## Supported Registries

```bash
# Docker Hub
docker tag fibcalc:latest username/fibcalc:latest
docker push username/fibcalc:latest

# GitHub Container Registry
docker tag fibcalc:latest ghcr.io/username/fibcalc:latest
docker push ghcr.io/username/fibcalc:latest

# AWS ECR
docker tag fibcalc:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/fibcalc:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/fibcalc:latest
```
