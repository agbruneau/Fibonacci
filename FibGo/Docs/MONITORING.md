# Monitoring Guide

> **Version**: 1.0.0  
> **Last Updated**: December 2025

This guide covers monitoring, observability, and alerting for the Fibonacci Calculator in production.

---

## Table of Contents

1. [Built-in Metrics](#built-in-metrics)
2. [Prometheus Integration](#prometheus-integration)
3. [Grafana Dashboards](#grafana-dashboards)
4. [Alerting Rules](#alerting-rules)
5. [Logging](#logging)
6. [Health Checks](#health-checks)

---

## Built-in Metrics

The server exposes metrics at the `/metrics` endpoint:

```bash
curl http://localhost:8080/metrics
```

### Available Metrics

| Metric                      | Type      | Description                            |
| --------------------------- | --------- | -------------------------------------- |
| `uptime`                    | Gauge     | Time since server startup              |
| `total_requests`            | Counter   | Total HTTP requests received           |
| `total_calculations`        | Counter   | Total Fibonacci calculations performed |
| `calculations_by_algorithm` | Counter   | Calculations per algorithm             |
| `rate_limit_hits`           | Counter   | Requests blocked by rate limiting      |
| `active_connections`        | Gauge     | Current active connections             |
| `calculation_duration_*`    | Histogram | Calculation duration distribution      |
| `errors_*`                  | Counter   | Error counts by type                   |

### Example Response

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
    }
  },
  "rate_limit_hits": 15,
  "active_connections": 3
}
```

---

## Prometheus Integration

### Prometheus Configuration

Add fibcalc to your `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "fibcalc"
    static_configs:
      - targets: ["fibcalc:8080"]
    metrics_path: "/metrics"
    scrape_interval: 15s
    scrape_timeout: 10s
```

### Kubernetes ServiceMonitor

For Prometheus Operator deployments:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: fibcalc
  namespace: fibcalc
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: fibcalc
  namespaceSelector:
    matchNames:
      - fibcalc
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
      scrapeTimeout: 10s
```

### Pod Annotations

Alternative to ServiceMonitor:

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

---

## Grafana Dashboards

### Importing the Dashboard

1. Open Grafana → Dashboards → Import
2. Use the JSON below or import from file

### FibCalc Dashboard JSON

```json
{
  "dashboard": {
    "title": "FibCalc Monitoring",
    "uid": "fibcalc-main",
    "panels": [
      {
        "title": "Requests per Second",
        "type": "graph",
        "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
        "targets": [
          {
            "expr": "rate(fibcalc_total_requests[5m])",
            "legendFormat": "Requests/s"
          }
        ]
      },
      {
        "title": "Calculation Duration (p95)",
        "type": "graph",
        "gridPos": { "h": 8, "w": 12, "x": 12, "y": 0 },
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(fibcalc_calculation_duration_bucket[5m]))",
            "legendFormat": "p95 {{algorithm}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "stat",
        "gridPos": { "h": 4, "w": 6, "x": 0, "y": 8 },
        "targets": [
          {
            "expr": "sum(rate(fibcalc_errors_total[5m])) / sum(rate(fibcalc_total_requests[5m])) * 100",
            "legendFormat": "Error %"
          }
        ]
      },
      {
        "title": "Active Connections",
        "type": "gauge",
        "gridPos": { "h": 4, "w": 6, "x": 6, "y": 8 },
        "targets": [
          {
            "expr": "fibcalc_active_connections",
            "legendFormat": "Connections"
          }
        ]
      },
      {
        "title": "Rate Limit Hits",
        "type": "stat",
        "gridPos": { "h": 4, "w": 6, "x": 12, "y": 8 },
        "targets": [
          {
            "expr": "rate(fibcalc_rate_limit_hits[5m]) * 60",
            "legendFormat": "Hits/min"
          }
        ]
      },
      {
        "title": "Uptime",
        "type": "stat",
        "gridPos": { "h": 4, "w": 6, "x": 18, "y": 8 },
        "targets": [
          {
            "expr": "fibcalc_uptime_seconds",
            "legendFormat": "Uptime"
          }
        ]
      }
    ]
  }
}
```

### Key Panels Recommendations

| Panel           | Purpose         | Query Example                      |
| --------------- | --------------- | ---------------------------------- |
| Request Rate    | Traffic volume  | `rate(total_requests[5m])`         |
| Latency p95     | Performance     | `histogram_quantile(0.95, ...)`    |
| Error Rate      | Reliability     | `errors / requests * 100`          |
| Algorithm Usage | Distribution    | `sum by (algorithm)(calculations)` |
| Rate Limits     | Abuse detection | `rate(rate_limit_hits[5m])`        |

---

## Alerting Rules

### PrometheusRule for Kubernetes

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: fibcalc-alerts
  namespace: fibcalc
spec:
  groups:
    - name: fibcalc.rules
      rules:
        # Service Down
        - alert: FibcalcDown
          expr: up{job="fibcalc"} == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "FibCalc service is down"
            description: "FibCalc has been unreachable for more than 2 minutes."

        # High Latency
        - alert: FibcalcHighLatency
          expr: |
            histogram_quantile(0.95, 
              rate(fibcalc_calculation_duration_seconds_bucket[5m])
            ) > 10
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "High calculation latency on FibCalc"
            description: "95th percentile latency is above 10 seconds."

        # High Error Rate
        - alert: FibcalcHighErrorRate
          expr: |
            sum(rate(fibcalc_errors_total[5m])) 
            / sum(rate(fibcalc_total_requests[5m])) > 0.05
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High error rate on FibCalc"
            description: "Error rate is above 5% for 5 minutes."

        # Rate Limiting Active
        - alert: FibcalcHighRateLimiting
          expr: rate(fibcalc_rate_limit_hits[5m]) > 10
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High rate limiting on FibCalc"
            description: "More than 10 requests/second being rate limited."

        # Pod Memory Usage
        - alert: FibcalcHighMemory
          expr: |
            container_memory_usage_bytes{container="fibcalc"} 
            / container_spec_memory_limit_bytes{container="fibcalc"} > 0.9
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "FibCalc high memory usage"
            description: "Memory usage is above 90% of limit."

        # Pod CPU Usage
        - alert: FibcalcHighCPU
          expr: |
            rate(container_cpu_usage_seconds_total{container="fibcalc"}[5m]) 
            / container_spec_cpu_quota{container="fibcalc"} * 100000 > 0.9
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "FibCalc high CPU usage"
            description: "CPU usage is above 90% of limit."
```

### Alert Severity Levels

| Severity   | Response Time | Examples                      |
| ---------- | ------------- | ----------------------------- |
| `critical` | Immediate     | Service down, data loss       |
| `warning`  | Within 1h     | High latency, high error rate |
| `info`     | Daily review  | Rate limiting, high usage     |

---

## Logging

### Log Format

The server logs in structured format:

```
[SERVER] 2025/12/22 10:15:32 GET /calculate from 192.168.1.100
[SERVER] 2025/12/22 10:15:32 GET /calculate completed in 125.5ms
```

### Log Levels

Configure via environment:

```bash
# Options: debug, info, warn, error
FIBCALC_LOG_LEVEL=info fibcalc --server
```

### Centralized Logging

#### Docker with Fluentd

```yaml
version: "3.8"
services:
  fibcalc:
    image: fibcalc:latest
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: fibcalc.{{.ID}}
```

#### Kubernetes with Loki

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: promtail-config
data:
  promtail.yaml: |
    scrape_configs:
      - job_name: fibcalc
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            regex: fibcalc
            action: keep
```

---

## Health Checks

### Endpoint

```bash
curl http://localhost:8080/health
```

### Response

```json
{
  "status": "healthy",
  "timestamp": 1703254800
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

### Docker Health Check

```dockerfile
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget -q --spider http://localhost:8080/health || exit 1
```

---

## Best Practices

1. **Set up alerts before going to production**
2. **Monitor error rates, not just uptime**
3. **Track latency percentiles (p50, p95, p99)**
4. **Correlate metrics with logs for debugging**
5. **Review dashboards weekly for trends**
6. **Test alerting rules regularly**

---

## See Also

- [Docs/PERFORMANCE.md](PERFORMANCE.md) - Performance tuning
- [Docs/SECURITY.md](SECURITY.md) - Security configuration
- [Docs/TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Troubleshooting guide
- [Docs/deployment/KUBERNETES.md](deployment/KUBERNETES.md) - K8s deployment
