# Troubleshooting Guide

> **Version**: 1.0.0  
> **Last Updated**: December 2025

This guide helps diagnose and resolve common issues with the Fibonacci Calculator.

---

## Table of Contents

1. [CLI Issues](#cli-issues)
2. [Server Issues](#server-issues)
3. [Docker Issues](#docker-issues)
4. [Kubernetes Issues](#kubernetes-issues)
5. [Performance Issues](#performance-issues)
6. [Build Issues](#build-issues)

---

## CLI Issues

### Command not found

**Symptom**: `fibcalc: command not found`

**Solution**:

```bash
# Verify the binary is in your PATH
which fibcalc

# If installed via go install, add GOBIN to PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Or run from build directory
./build/fibcalc -n 100
```

### Calculation takes too long

**Symptom**: Calculation hangs or takes hours

**Solutions**:

1. Use `--timeout` to set a limit:

   ```bash
   fibcalc -n 100000000 --timeout 10m
   ```

2. Run calibration for optimal settings:

   ```bash
   fibcalc --calibrate
   ```

3. For very large N, consider using FFT:
   ```bash
   fibcalc -n 250000000 --algo fft --fft-threshold 500000
   ```

### Out of memory

**Symptom**: `runtime: out of memory` or process killed

**Solutions**:

1. Reduce N value
2. Increase system memory/swap
3. Check result size: F(1 billion) requires ~25 GB RAM

**Memory estimates**:
| N | Approximate Memory |
|---|-------------------|
| 10,000,000 | ~50 MB |
| 100,000,000 | ~500 MB |
| 250,000,000 | ~1.5 GB |
| 1,000,000,000 | ~25 GB |

### Progress not showing

**Symptom**: No spinner or progress bar

**Solutions**:

1. Check if `NO_COLOR` is set:

   ```bash
   unset NO_COLOR
   ```

2. Verify terminal supports ANSI:

   ```bash
   echo $TERM  # Should be xterm-256color or similar
   ```

3. For piped output, progress is automatically disabled

---

## Server Issues

### Server won't start

**Symptom**: `listen tcp :8080: bind: address already in use`

**Solution**:

```bash
# Find process using port
lsof -i :8080  # Linux/macOS
netstat -ano | findstr :8080  # Windows

# Use different port
fibcalc --server --port 9090
```

### Connection refused

**Symptom**: `curl: (7) Failed to connect`

**Solutions**:

1. Verify server is running:

   ```bash
   ps aux | grep fibcalc
   ```

2. Check server logs for errors

3. Verify firewall allows the port:
   ```bash
   sudo ufw allow 8080/tcp  # Ubuntu
   ```

### Rate limiting triggered

**Symptom**: `429 Too Many Requests`

**Solutions**:

1. Slow down requests (max 10/second per IP)
2. Wait for rate limit to reset
3. Adjust rate limit if you control the server:
   ```bash
   FIBCALC_RATE_LIMIT=50 fibcalc --server
   ```

### Request timeout

**Symptom**: `504 Gateway Timeout` or calculation never returns

**Solutions**:

1. Reduce N value
2. Increase timeout:
   ```bash
   fibcalc --server --timeout 30m
   ```
3. Use smaller calculations for health checks

---

## Docker Issues

### Container won't start

**Symptom**: Container exits immediately

**Diagnosis**:

```bash
# Check logs
docker logs fibcalc-server

# Run interactively
docker run --rm -it fibcalc:latest --help
```

**Common causes**:

- Invalid command arguments
- Missing required environment variables
- Port conflicts

### Permission denied

**Symptom**: `permission denied` errors

**Solution**: The container runs as non-root. Ensure mounted volumes are accessible:

```bash
# Fix permissions on host
chmod 755 /path/to/data

# Or run with specific user ID
docker run --user $(id -u):$(id -g) fibcalc:latest
```

### High memory usage

**Symptom**: Container uses excessive memory

**Solutions**:

1. Set memory limits:

   ```bash
   docker run --memory=2g --memory-swap=2g fibcalc:latest --server
   ```

2. Limit N value via environment:
   ```bash
   docker run -e FIBCALC_MAX_N=10000000 fibcalc:latest --server
   ```

### Image build fails

**Symptom**: `go build` fails during Docker build

**Solutions**:

1. Ensure sufficient resources for Docker:

   ```bash
   # Docker Desktop: Settings > Resources > Memory: 4GB+
   ```

2. Clear Docker cache:
   ```bash
   docker builder prune
   docker build --no-cache -t fibcalc .
   ```

---

## Kubernetes Issues

### Pod CrashLoopBackOff

**Diagnosis**:

```bash
# Check pod status
kubectl describe pod -l app=fibcalc -n fibcalc

# Check previous logs
kubectl logs -l app=fibcalc -n fibcalc --previous
```

**Common causes**:

- Readiness probe failing
- Insufficient resources
- ConfigMap misconfiguration

### Pod pending (Insufficient resources)

**Symptom**: `Insufficient cpu` or `Insufficient memory`

**Solutions**:

1. Reduce resource requests:

   ```yaml
   resources:
     requests:
       cpu: "250m"
       memory: "256Mi"
   ```

2. Scale cluster or add nodes

### Service not accessible

**Diagnosis**:

```bash
# Check endpoints
kubectl get endpoints fibcalc -n fibcalc

# Test from within cluster
kubectl run test --rm -it --image=busybox -n fibcalc -- wget -qO- http://fibcalc/health

# Port forward for local testing
kubectl port-forward svc/fibcalc 8080:80 -n fibcalc
```

### HPA not scaling

**Diagnosis**:

```bash
# Check HPA status
kubectl describe hpa fibcalc -n fibcalc

# Verify metrics-server is running
kubectl top pods -n fibcalc
```

**Common causes**:

- metrics-server not installed
- Resource limits not defined
- Target utilization not reached

---

## Performance Issues

### Calculation slower than expected

**Diagnosis**:

```bash
# Compare algorithms
fibcalc -n 1000000 --algo all --details

# Run calibration
fibcalc --calibrate
```

**Solutions**:

1. Use auto-calibration:

   ```bash
   fibcalc -n 10000000 --auto-calibrate
   ```

2. Tune thresholds manually:

   ```bash
   fibcalc -n 10000000 --threshold 2048 --fft-threshold 500000
   ```

3. Limit GOMAXPROCS on high-core systems:
   ```bash
   GOMAXPROCS=8 fibcalc -n 10000000
   ```

### GC pauses

**Symptom**: Periodic slowdowns during calculation

**Solution**: The calculator uses `sync.Pool` to minimize allocations. If GC pressure persists:

```bash
# Increase GOGC threshold
GOGC=200 fibcalc -n 100000000
```

### FFT not being used

**Symptom**: Large calculations not using FFT

**Solution**: Lower the FFT threshold:

```bash
fibcalc -n 50000000 --fft-threshold 100000
```

---

## Build Issues

### CGO disabled errors

**Symptom**: `CGO_ENABLED=0` causing build issues

**Solution**: The standard build doesn't require CGO:

```bash
CGO_ENABLED=0 go build -o fibcalc ./cmd/fibcalc
```

### GMP build failures

**Symptom**: Errors when building with `-tags gmp`

**Solutions**:

1. Install GMP development headers:

   ```bash
   # Ubuntu/Debian
   sudo apt-get install libgmp-dev

   # macOS
   brew install gmp
   ```

2. Set CGO flags if needed:
   ```bash
   CGO_CFLAGS="-I/usr/local/include" \
   CGO_LDFLAGS="-L/usr/local/lib" \
   go build -tags gmp ./cmd/fibcalc
   ```

### Module errors

**Symptom**: `go: module not found` errors

**Solution**:

```bash
# Download dependencies
go mod download

# Tidy modules
go mod tidy

# Verify modules
go mod verify
```

---

## Getting Help

If this guide doesn't resolve your issue:

1. **Check existing issues**: [GitHub Issues](https://github.com/agbru/fibcalc/issues)
2. **Open a new issue** with:
   - Go version (`go version`)
   - Operating system
   - Steps to reproduce
   - Error messages/logs
3. **Security issues**: See [SECURITY.md](SECURITY.md) for responsible disclosure

---

## See Also

- [Docs/PERFORMANCE.md](PERFORMANCE.md) - Performance tuning
- [Docs/SECURITY.md](SECURITY.md) - Security configuration
- [Docs/deployment/DOCKER.md](deployment/DOCKER.md) - Docker guide
- [Docs/deployment/KUBERNETES.md](deployment/KUBERNETES.md) - Kubernetes guide
