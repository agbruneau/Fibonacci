# Security Policy

> **Version**: 1.0.0  
> **Last Updated**: November 2025

## Overview

This document describes the security measures implemented in the Fibonacci Calculator and best practices for production deployment.

## Vulnerability Reporting

### Contact

If you discover a security vulnerability, please contact us responsibly:

- **Email**: agbru@users.noreply.github.com
- **GitHub Issues**: For non-sensitive issues, use GitHub Issues with the "security" label
- **PGP Key**: Available upon request

### Disclosure Process

1. **Report**: Send an email detailing the vulnerability
2. **Acknowledgement**: Response within 48 hours
3. **Assessment**: Analysis within 7 days
4. **Fix**: Patch development
5. **Publication**: Coordinated release with credit to the discoverer

### Information to Provide

- Detailed description of the vulnerability
- Reproduction steps
- Potential impact
- Fix suggestions (optional)

## Implemented Security Measures

### 1. Denial of Service (DoS) Attack Protection

#### Limit on N Value

The server limits the maximum value of N to prevent resource exhaustion:

```go
// SecurityConfig in internal/server/middleware.go
type SecurityConfig struct {
    MaxNValue uint64 // Default: 1_000_000_000
}
```

Requests with N too high return a 400 error:

```json
{
  "error": "Bad Request",
  "message": "Value of 'n' exceeds maximum allowed (1000000000)"
}
```

#### Rate Limiting

The server implements per-IP rate limiting:

```go
type RateLimiterConfig struct {
    RequestsPerSecond float64 // Default: 10
    BurstSize         int     // Default: 20
}
```

Requests exceeding the limit receive a 429 response:

```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded. Please slow down."
}
```

#### Timeouts

All calculations have a configurable timeout:

```go
const (
    DefaultRequestTimeout  = 5 * time.Minute
    DefaultReadTimeout     = 10 * time.Second
    DefaultWriteTimeout    = 10 * time.Minute
    DefaultIdleTimeout     = 2 * time.Minute
    DefaultShutdownTimeout = 30 * time.Second
)
```

### 2. HTTP Security Headers

The security middleware adds protective headers:

```go
func SecurityMiddleware(config SecurityConfig, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'none'")
        w.Header().Set("Referrer-Policy", "no-referrer")
        
        next(w, r)
    }
}
```

### 3. Input Validation

All user inputs are validated:

```go
// Validation of 'n' parameter
n, err := strconv.ParseUint(nStr, 10, 64)
if err != nil {
    s.writeErrorResponse(w, http.StatusBadRequest, 
        "Invalid 'n' parameter: must be a positive integer")
    return
}

// Validation of 'algo' parameter
calc, ok := s.registry[algo]
if !ok {
    s.writeErrorResponse(w, http.StatusBadRequest,
        fmt.Sprintf("Invalid 'algo' parameter: '%s' is not a valid algorithm", algo))
    return
}
```

### 4. Docker Isolation

The Dockerfile implements security best practices:

```dockerfile
# Non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Minimal image (Alpine)
FROM alpine:latest

# No interactive shell for user
ENTRYPOINT ["/app/fibcalc"]
```

### 5. Graceful Shutdown

The server properly handles shutdown signals to avoid abrupt interruptions:

```go
func (s *Server) Start() error {
    signal.Notify(s.shutdownSignal, os.Interrupt, syscall.SIGTERM)
    
    // ... server startup ...
    
    <-s.shutdownSignal
    s.logger.Println("Shutdown signal received...")
    
    ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
    defer cancel()
    
    return s.httpServer.Shutdown(ctx)
}
```

## Secure Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_MAX_N` | Maximum limit for N | 1,000,000,000 |
| `FIBCALC_RATE_LIMIT` | Requests per second | 10 |
| `FIBCALC_TIMEOUT` | Calculation timeout | 5m |

### Command-Line Flags

```bash
# Recommended secure configuration
./fibcalc --server \
    --port 8080 \
    --timeout 2m
```

## Deployment Recommendations

### 1. Reverse Proxy (Nginx)

Place the server behind a reverse proxy for:
- TLS termination
- Additional rate limiting
- Access logging
- DDoS protection

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Host $host;
    }
}
```

### 2. Kubernetes

Use NetworkPolicies to isolate the pod:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: fibcalc-policy
spec:
  podSelector:
    matchLabels:
      app: fibcalc
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              role: frontend
      ports:
        - port: 8080
  egress: []  # No egress needed
```

### 3. Resource Limits

Configure resource limits to prevent exhaustion:

```yaml
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "2000m"
    memory: "2Gi"
```

### 4. Logging and Audit

The server logs all requests:

```
[SERVER] 2025/11/29 10:15:32 GET /calculate from 192.168.1.100
[SERVER] 2025/11/29 10:15:32 GET /calculate completed in 125.5ms
```

For a complete audit, configure an external log collector (Fluentd, Loki, etc.).

## Security Checklist

### Before Deployment

- [ ] TLS configured (valid certificates)
- [ ] Rate limiting enabled
- [ ] Resource limits configured
- [ ] Non-root user in Docker
- [ ] NetworkPolicy applied (Kubernetes)
- [ ] Centralised logging
- [ ] Error monitoring

### In Production

- [ ] Regular dependency updates
- [ ] Log analysis for anomalies
- [ ] Periodic penetration testing
- [ ] Calibration profile backups
- [ ] Access review

## Supported Versions

| Version | Supported | End of Support |
|---------|-----------|----------------|
| 1.0.x | ✅ | December 2026 |
| < 1.0 | ❌ | N/A |

## Dependencies

Dependencies are regularly audited. Run:

```bash
# Check for known vulnerabilities
go list -m all | nancy sleuth

# Or with govulncheck
govulncheck ./...
```

## Compliance

This project follows best practices from:

- **OWASP**: Top 10 API Security Risks
- **CWE**: Common Weakness Enumeration
- **Go Security**: Official Go recommendations
