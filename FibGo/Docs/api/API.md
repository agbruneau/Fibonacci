# REST API Documentation

> **Version**: 1.0.0  
> **Last Updated**: November 2025

This document describes the endpoints available in the Fibonacci Calculator REST API.

## Overview

The REST API allows you to perform Fibonacci number calculations via HTTP. It includes security protections (rate limiting, input validation) and exposes performance metrics.

### Base URL

```
http://localhost:8080
```

### Security Headers

All endpoints return the following security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: default-src 'none'`
- `Referrer-Policy: no-referrer`

### Rate Limiting

The API implements per-IP rate limiting:
- **Requests per second**: 10
- **Allowed burst**: 20

Requests exceeding the limit receive a `429 Too Many Requests` response.

---

## Endpoints

### 1. Calculate a Fibonacci Number

Calculates the Nth Fibonacci number using the specified algorithm.

**URL**: `/calculate`  
**Method**: `GET`

#### Query Parameters

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `n`       | uint64 | Yes      | The index of the Fibonacci number to calculate (must be positive, max: 1,000,000,000). |
| `algo`    | string | No       | The algorithm to use. Default: `fast`. Possible values: `fast`, `matrix`, `fft`. |

#### Request Example

```bash
curl "http://localhost:8080/calculate?n=100&algo=fast"
```

#### Success Response (200 OK)

```json
{
  "n": 100,
  "result": 354224848179261915075,
  "duration": "125.5µs",
  "algorithm": "fast"
}
```

#### Response Schema

| Field | Type | Description |
|-------|------|-------------|
| `n` | uint64 | The requested Fibonacci number index |
| `result` | string/number | The calculated Fibonacci number (can be very large) |
| `duration` | string | Formatted calculation duration |
| `algorithm` | string | The algorithm used for the calculation |
| `error` | string | Error message (if applicable) |

#### Error Response (400 Bad Request)

**Missing `n` parameter:**
```json
{
  "error": "Bad Request",
  "message": "Missing 'n' parameter"
}
```

**Invalid `n` parameter:**
```json
{
  "error": "Bad Request",
  "message": "Invalid 'n' parameter: must be a positive integer"
}
```

**`n` value too large:**
```json
{
  "error": "Bad Request",
  "message": "Value of 'n' exceeds maximum allowed (1000000000). This limit prevents resource exhaustion."
}
```

**Invalid algorithm:**
```json
{
  "error": "Bad Request",
  "message": "Invalid 'algo' parameter: 'unknown' is not a valid algorithm"
}
```

#### Rate Limit Response (429 Too Many Requests)

```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded. Please slow down."
}
```

---

### 2. Health Check

Checks if the server is online and operational.

**URL**: `/health`  
**Method**: `GET`

#### Request Example

```bash
curl "http://localhost:8080/health"
```

#### Success Response (200 OK)

```json
{
  "status": "healthy",
  "timestamp": 1732900800
}
```

#### Response Schema

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Service health status ("healthy") |
| `timestamp` | int64 | Unix timestamp of the response |

---

### 3. List Algorithms

Returns the list of calculation algorithms available on the server.

**URL**: `/algorithms`  
**Method**: `GET`

#### Request Example

```bash
curl "http://localhost:8080/algorithms"
```

#### Success Response (200 OK)

```json
{
  "algorithms": [
    "fast",
    "fft",
    "matrix"
  ]
}
```

#### Response Schema

| Field | Type | Description |
|-------|------|-------------|
| `algorithms` | []string | List of available algorithm names |

---

### 4. Server Metrics

Exposes server performance metrics for monitoring.

**URL**: `/metrics`  
**Method**: `GET`

#### Request Example

```bash
curl "http://localhost:8080/metrics"
```

#### Success Response (200 OK)

```json
{
  "uptime": "2h15m32s",
  "total_requests": 1542,
  "total_calculations": 1230,
  "calculations_by_algorithm": {
    "fast": {
      "count": 850,
      "success": 848,
      "errors": 2,
      "total_duration": "125.5s",
      "avg_duration": "147.6ms"
    },
    "matrix": {
      "count": 280,
      "success": 280,
      "errors": 0,
      "total_duration": "52.3s",
      "avg_duration": "186.8ms"
    },
    "fft": {
      "count": 100,
      "success": 100,
      "errors": 0,
      "total_duration": "18.7s",
      "avg_duration": "187ms"
    }
  },
  "rate_limit_hits": 15,
  "active_connections": 3
}
```

#### Response Schema

| Field | Type | Description |
|-------|------|-------------|
| `uptime` | string | Time since server startup |
| `total_requests` | int64 | Total number of HTTP requests received |
| `total_calculations` | int64 | Total number of calculations performed |
| `calculations_by_algorithm` | object | Detailed statistics per algorithm |
| `rate_limit_hits` | int64 | Number of requests blocked by rate limiting |
| `active_connections` | int | Number of active connections |

#### Per-Algorithm Statistics

| Field | Type | Description |
|-------|------|-------------|
| `count` | int64 | Total number of calculations |
| `success` | int64 | Number of successful calculations |
| `errors` | int64 | Number of failed calculations |
| `total_duration` | string | Cumulative total duration |
| `avg_duration` | string | Average duration per calculation |

---

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| `200 OK` | Successful request |
| `400 Bad Request` | Invalid parameters |
| `405 Method Not Allowed` | HTTP method not supported |
| `429 Too Many Requests` | Rate limit exceeded |
| `500 Internal Server Error` | Internal server error |

---

## cURL Examples

### Simple Calculation

```bash
curl "http://localhost:8080/calculate?n=50"
```

### Calculation with Specific Algorithm

```bash
curl "http://localhost:8080/calculate?n=10000&algo=matrix"
```

### Calculation of a Very Large Number

```bash
curl "http://localhost:8080/calculate?n=1000000&algo=fast"
```

### Pipeline with jq

```bash
# Extract only the result
curl -s "http://localhost:8080/calculate?n=100&algo=fast" | jq '.result'

# Extract the duration
curl -s "http://localhost:8080/calculate?n=100000&algo=fast" | jq '.duration'
```

---

## Server Configuration

### Startup

```bash
# Default port (8080)
./fibcalc --server

# Custom port
./fibcalc --server --port 3000

# With auto-calibration
./fibcalc --server --port 8080 --auto-calibrate

# With custom timeout
./fibcalc --server --port 8080 --timeout 10m
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_MAX_N` | Maximum limit for N | 1,000,000,000 |
| `FIBCALC_RATE_LIMIT` | Requests per second | 10 |
| `FIBCALC_TIMEOUT` | Calculation timeout | 5m |

### Timeouts

| Parameter | Value | Description |
|-----------|-------|-------------|
| Request Timeout | 5 minutes | Maximum timeout per calculation |
| Read Timeout | 10 seconds | Request read timeout |
| Write Timeout | 10 minutes | Response write timeout |
| Idle Timeout | 2 minutes | Inactive connection timeout |
| Shutdown Timeout | 30 seconds | Graceful shutdown timeout |

---

## Integration

### Docker

```bash
# Start the server
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080

# Test
curl "http://localhost:8080/health"
```

### Docker Compose

```yaml
version: '3.8'
services:
  fibcalc:
    image: fibcalc:latest
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080"]
```

### Kubernetes

```yaml
apiVersion: v1
kind: Service
metadata:
  name: fibcalc
spec:
  ports:
    - port: 80
      targetPort: 8080
  selector:
    app: fibcalc
```

---

## Security

### DoS Protection

- **Limit on N**: The maximum value of N is limited to 1 billion.
- **Rate Limiting**: 10 requests/second per IP with burst of 20.
- **Timeouts**: All calculations have a configurable timeout.

### Input Validation

All user inputs are strictly validated:
- The `n` parameter must be a positive integer.
- The `algo` parameter must match a registered algorithm.

### Logging

The server logs all requests:
```
[SERVER] 2025/11/29 10:15:32 GET /calculate from 192.168.1.100
[SERVER] 2025/11/29 10:15:32 GET /calculate completed in 125.5ms
```

---

## See Also

- [README.md](README.md) - Main documentation
- [Docs/SECURITY.md](Docs/SECURITY.md) - Complete security policy
- [Docs/PERFORMANCE.md](Docs/PERFORMANCE.md) - Performance guide
- [Docs/api/openapi.yaml](Docs/api/openapi.yaml) - OpenAPI specification
