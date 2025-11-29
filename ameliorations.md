# 📋 Document d'Améliorations - Calculateur Fibonacci Haute Performance

> **Version analysée :** 1.0.0  
> **Date d'analyse :** Novembre 2025  
> **Auteur :** Analyse automatisée du code source

---

## 🏗️ Vue d'ensemble du projet

Ce projet est un calculateur de Fibonacci haute performance en Go, implémentant plusieurs algorithmes (Fast Doubling, Matrix Exponentiation, FFT) avec des optimisations avancées (sync.Pool, parallélisme, FFT). Le code est bien structuré et suit les bonnes pratiques Go.

### Points forts actuels
- ✅ Architecture modulaire et découplée (Clean Architecture)
- ✅ Stratégie zéro-allocation avec `sync.Pool`
- ✅ Parallélisme adaptatif multi-cœurs
- ✅ API REST avec rate limiting et sécurité
- ✅ Support Docker et Makefile complet
- ✅ Tests unitaires et benchmarks
- ✅ Système i18n basique

---

## DONE 1. 🔧 Améliorations Techniques

### DONE 1.1 Gestion de la mémoire et performances

| Amélioration | Description | Priorité | Fichier(s) concerné(s) |
|--------------|-------------|----------|------------------------|
| **Pool FFT** | Implémenter un `sync.Pool` pour les structures FFT (`poly`, `polValues`) pour réduire les allocations dans les calculs FFT répétés | Haute | `internal/bigfft/fft.go` |
| **Pré-allocation intelligente** | Optimiser l'allocation de `p.a` dans `polyFromNat()` basée sur la taille réelle nécessaire | Moyenne | `internal/bigfft/fft.go:151` |
| **Cache LRU pour résultats** | Ajouter un cache optionnel pour les calculs fréquents (utile en mode serveur) | Moyenne | Nouveau : `internal/cache/` |

#### Exemple d'implémentation - Pool FFT

```go
// internal/bigfft/fft.go

var polyPool = sync.Pool{
    New: func() interface{} {
        return &poly{a: make([]nat, 0, 64)}
    },
}

func acquirePoly() *poly {
    p := polyPool.Get().(*poly)
    p.a = p.a[:0]
    return p
}

func releasePoly(p *poly) {
    polyPool.Put(p)
}
```

### DONE 1.2 Amélioration des algorithmes

| Amélioration | Description | Priorité |
|--------------|-------------|----------|
| **Algorithme de Lucas** | Implémenter la méthode de Lucas comme 4ème algorithme pour comparaison | Basse |
| **Multiplication de Toom-Cook** | Ajouter Toom-3 comme intermédiaire entre Karatsuba et FFT | Moyenne |
| **Optimisation SIMD** | Utiliser les instructions SIMD (AVX2/AVX-512) via assembly pour les opérations critiques sur `big.Int` | Haute |

#### Exemple d'implémentation - Algorithme de Lucas

```go
// internal/fibonacci/lucas.go

package fibonacci

import (
    "context"
    "math/big"
)

// LucasCalculator implémente la méthode de Lucas pour calculer F(n).
// Utilise les suites de Lucas U_n et V_n.
type LucasCalculator struct{}

func (c *LucasCalculator) Name() string {
    return "Lucas Sequence (O(log n))"
}

func (c *LucasCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
    // U_n = F(n), V_n = L(n) où L(n) est le n-ème nombre de Lucas
    // Utilise les identités:
    // U_2n = U_n * V_n
    // V_2n = V_n² - 2*(-1)^n
    // Implementation...
}
```

### DONE 1.3 Calibration améliorée

**Problème actuel :** Les seuils de calibration sont codés en dur dans `internal/calibration/calibration.go`.

```go
// Ligne 60 - Actuel
thresholdsToTest := []int{0, 256, 512, 1024, 2048, 4096, 8192, 16384}

// Ligne 168 - Actuel  
parallelCandidates := []int{0, 2048, 4096, 8192, 16384}
```

#### Améliorations proposées

1. **Calibration adaptative basée sur la détection CPU**
2. **Seuils dynamiques selon le nombre de cœurs**
3. **Persistance des résultats de calibration**

```go
// internal/calibration/adaptive.go

package calibration

import (
    "encoding/json"
    "os"
    "path/filepath"
    "runtime"
)

// CalibrationProfile stocke les résultats de calibration
type CalibrationProfile struct {
    CPUModel          string `json:"cpu_model"`
    NumCores          int    `json:"num_cores"`
    OptimalThreshold  int    `json:"optimal_threshold"`
    OptimalFFT        int    `json:"optimal_fft_threshold"`
    OptimalStrassen   int    `json:"optimal_strassen_threshold"`
    CalibratedAt      string `json:"calibrated_at"`
}

// GetCalibrationPath retourne le chemin du fichier de calibration
func GetCalibrationPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".fibcalc_calibration.json")
}

// LoadProfile charge le profil de calibration existant
func LoadProfile() (*CalibrationProfile, error) {
    // Implementation...
}

// SaveProfile sauvegarde le profil de calibration
func SaveProfile(profile *CalibrationProfile) error {
    // Implementation...
}

// GenerateAdaptiveThresholds génère des seuils basés sur le CPU
func GenerateAdaptiveThresholds() []int {
    numCPU := runtime.NumCPU()
    base := []int{0, 256, 512, 1024, 2048, 4096}
    
    // Ajouter des seuils plus élevés pour les machines puissantes
    if numCPU >= 8 {
        base = append(base, 8192, 16384, 32768)
    }
    if numCPU >= 16 {
        base = append(base, 65536)
    }
    
    return base
}
```

---

## DONE 2. 🌐 Améliorations du Serveur HTTP

### DONE 2.1 Nouveaux endpoints

| Endpoint | Méthode | Description | Priorité |
|----------|---------|-------------|----------|
| `/batch` | POST | Calcul de plusieurs valeurs en une seule requête | Haute |
| `/range` | GET | Calcul d'une plage de Fibonacci (F(a) à F(b)) | Basse |
| `/ws/progress` | WebSocket | Streaming du progrès pour les calculs longs | Moyenne |
| `/docs` | GET | Documentation OpenAPI/Swagger | Haute |

#### Exemple d'implémentation - Endpoint Batch

```go
// internal/server/handlers_batch.go

package server

import (
    "encoding/json"
    "net/http"
    "sync"
)

// BatchRequest représente une requête de calcul par lots
type BatchRequest struct {
    Values []uint64 `json:"values"`
    Algo   string   `json:"algo,omitempty"`
}

// BatchResponse représente la réponse d'un calcul par lots
type BatchResponse struct {
    Results []BatchResultItem `json:"results"`
    Total   int               `json:"total"`
}

type BatchResultItem struct {
    N        uint64 `json:"n"`
    Result   string `json:"result,omitempty"`
    Duration string `json:"duration"`
    Error    string `json:"error,omitempty"`
}

// handleBatch traite les requêtes de calcul par lots
func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    var req BatchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body")
        return
    }

    // Limiter le nombre de valeurs par requête
    if len(req.Values) > 100 {
        s.writeErrorResponse(w, http.StatusBadRequest, "Maximum 100 values per batch request")
        return
    }

    results := make([]BatchResultItem, len(req.Values))
    var wg sync.WaitGroup

    for i, n := range req.Values {
        wg.Add(1)
        go func(idx int, val uint64) {
            defer wg.Done()
            // Calcul individuel...
        }(i, n)
    }

    wg.Wait()

    s.writeJSONResponse(w, http.StatusOK, BatchResponse{
        Results: results,
        Total:   len(results),
    })
}
```

### DONE 2.2 Sécurité et observabilité

| Amélioration | Description | Priorité | Fichier(s) |
|--------------|-------------|----------|------------|
| **JWT Authentication** | Authentification optionnelle pour API publique | Moyenne | Nouveau : `internal/server/auth.go` |
| **OpenTelemetry** | Support tracing distribué | Moyenne | `internal/server/tracing.go` |
| **Prometheus metrics** | Métriques compatibles Prometheus (améliorer l'existant) | Haute | `internal/server/metrics.go` |
| **Request ID** | ID unique par requête pour debugging | Basse | `internal/server/middleware.go` |

#### Exemple - Métriques Prometheus améliorées

```go
// internal/server/prometheus.go

package server

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    requestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "fibcalc_request_duration_seconds",
            Help:    "Duration of HTTP requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"endpoint", "method", "status"},
    )

    calculationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "fibcalc_calculation_duration_seconds",
            Help:    "Duration of Fibonacci calculations",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
        },
        []string{"algorithm"},
    )

    activeCalculations = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "fibcalc_active_calculations",
        Help: "Number of currently running calculations",
    })

    cacheHits = promauto.NewCounter(prometheus.CounterOpts{
        Name: "fibcalc_cache_hits_total",
        Help: "Total number of cache hits",
    })

    cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
        Name: "fibcalc_cache_misses_total",
        Help: "Total number of cache misses",
    })
)
```

### DONE 2.3 Rate Limiter amélioré

**Fichier concerné :** `internal/server/middleware.go`

Le rate limiter actuel utilise un simple token bucket par IP. Améliorations proposées :

```go
// internal/server/ratelimit_advanced.go

package server

import (
    "sync"
    "time"
)

// AdvancedRateLimiterConfig configuration avancée du rate limiter
type AdvancedRateLimiterConfig struct {
    // Limite de base par minute
    RequestsPerMinute int
    // Taille du burst autorisé
    BurstSize int
    // Utiliser une fenêtre glissante
    SlidingWindow bool
    // Limites différentes par endpoint
    PerEndpointLimits map[string]int
    // Limites par tier d'utilisateur (avec JWT)
    TierLimits map[string]int
    // Liste blanche d'IPs
    Whitelist []string
}

// AdvancedRateLimiter implémente un rate limiter plus sophistiqué
type AdvancedRateLimiter struct {
    config   AdvancedRateLimiterConfig
    mu       sync.RWMutex
    clients  map[string]*slidingWindowCounter
    stopChan chan struct{}
}

type slidingWindowCounter struct {
    counts      []int
    windowStart time.Time
    totalCount  int
}

// AllowWithContext vérifie si la requête est autorisée avec contexte
func (rl *AdvancedRateLimiter) AllowWithContext(clientIP, endpoint, tier string) bool {
    // Vérifier la whitelist
    if rl.isWhitelisted(clientIP) {
        return true
    }

    // Obtenir la limite applicable
    limit := rl.getApplicableLimit(endpoint, tier)

    // Vérifier avec fenêtre glissante
    return rl.checkSlidingWindow(clientIP, limit)
}
```

---

## 3. 📊 Améliorations CLI

### 3.1 Nouvelles fonctionnalités

| Fonctionnalité | Flag | Description | Priorité |
|----------------|------|-------------|----------|
| **Export fichier** | `-o, --output FILE` | Sauvegarder le résultat dans un fichier | Haute |
| **Mode interactif** | `--interactive` | REPL pour calculs multiples | Moyenne |
| **Format hexadécimal** | `--hex` | Affichage en hexadécimal | Basse |
| **Mode silencieux** | `-q, --quiet` | Sortie minimale pour scripts | Haute |
| **Autocomplétion** | `--completion bash/zsh/fish` | Génération scripts autocomplétion | Moyenne |

#### Exemple d'implémentation - Export fichier

```go
// Dans internal/config/config.go, ajouter :

type AppConfig struct {
    // ... champs existants ...
    
    // OutputFile, si spécifié, sauvegarde le résultat dans ce fichier
    OutputFile string
    // Quiet mode - sortie minimale
    Quiet bool
    // HexOutput - affichage hexadécimal
    HexOutput bool
}

// Dans cmd/fibcalc/main.go, ajouter le flag :
fs.StringVar(&config.OutputFile, "output", "", "Output file path for the result")
fs.StringVar(&config.OutputFile, "o", "", "Output file path (shorthand)")
fs.BoolVar(&config.Quiet, "quiet", false, "Quiet mode - minimal output")
fs.BoolVar(&config.Quiet, "q", false, "Quiet mode (shorthand)")
```

### 3.2 Mode interactif REPL

```go
// internal/cli/repl.go

package cli

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

// StartREPL démarre le mode interactif
func StartREPL(calculatorRegistry map[string]fibonacci.Calculator) {
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("Fibonacci Calculator REPL")
    fmt.Println("Commands: calc <n>, algo <name>, compare <n>, help, exit")
    fmt.Println()

    for {
        fmt.Print("fib> ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)

        if input == "exit" || input == "quit" {
            fmt.Println("Goodbye!")
            break
        }

        processCommand(input, calculatorRegistry)
    }
}

func processCommand(input string, registry map[string]fibonacci.Calculator) {
    parts := strings.Fields(input)
    if len(parts) == 0 {
        return
    }

    switch parts[0] {
    case "calc":
        // Calculer F(n)
    case "algo":
        // Changer d'algorithme
    case "compare":
        // Comparer tous les algorithmes
    case "help":
        printHelp()
    default:
        fmt.Printf("Unknown command: %s\n", parts[0])
    }
}
```

### 3.3 Amélioration du système i18n

**Fichier concerné :** `internal/i18n/messages.go`

```go
// internal/i18n/catalog.go

package i18n

import (
    "fmt"
    "sync"
)

// MessageCatalog gère les messages multilingues
type MessageCatalog struct {
    mu       sync.RWMutex
    messages map[string]map[string]string // lang -> key -> value
    fallback string
    current  string
}

// NewCatalog crée un nouveau catalogue de messages
func NewCatalog(fallback string) *MessageCatalog {
    return &MessageCatalog{
        messages: make(map[string]map[string]string),
        fallback: fallback,
        current:  fallback,
    }
}

// Get récupère un message avec support de formatage
func (c *MessageCatalog) Get(key string, args ...interface{}) string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // Chercher dans la langue courante
    if msgs, ok := c.messages[c.current]; ok {
        if msg, ok := msgs[key]; ok {
            if len(args) > 0 {
                return fmt.Sprintf(msg, args...)
            }
            return msg
        }
    }

    // Fallback
    if msgs, ok := c.messages[c.fallback]; ok {
        if msg, ok := msgs[key]; ok {
            if len(args) > 0 {
                return fmt.Sprintf(msg, args...)
            }
            return msg
        }
    }

    return key
}

// GetPlural gère la pluralisation
func (c *MessageCatalog) GetPlural(key string, count int) string {
    suffix := "_plural"
    if count == 1 {
        suffix = "_singular"
    }
    return c.Get(key + suffix)
}
```

**Nouvelles langues à supporter :**
- `locales/fr.json` - Français (déjà partiel)
- `locales/es.json` - Espagnol
- `locales/de.json` - Allemand
- `locales/zh.json` - Chinois simplifié
- `locales/ja.json` - Japonais

---

## 4. 🧪 Améliorations des Tests

### 4.1 Objectifs de couverture

| Package | Couverture actuelle | Cible | Actions requises |
|---------|---------------------|-------|------------------|
| `fibonacci` | ~80% | 90% | Tests edge cases, overflow |
| `server` | ~70% | 85% | Tests d'intégration HTTP |
| `bigfft` | ~60% | 80% | Tests précision numérique |
| `calibration` | ~65% | 80% | Mocking du temps, tests parallèles |
| `cli` | ~75% | 85% | Tests output formatting |
| `config` | ~85% | 90% | Tests validation exhaustifs |

### 4.2 Tests de Fuzzing (Go 1.18+)

```go
// internal/fibonacci/fibonacci_fuzz_test.go

package fibonacci

import (
    "context"
    "testing"
)

func FuzzFastDoublingConsistency(f *testing.F) {
    // Seeds
    f.Add(uint64(0))
    f.Add(uint64(1))
    f.Add(uint64(93)) // Max uint64
    f.Add(uint64(100))
    f.Add(uint64(1000))

    f.Fuzz(func(t *testing.T, n uint64) {
        // Limiter pour éviter les timeouts
        if n > 50000 {
            return
        }

        ctx := context.Background()
        opts := Options{ParallelThreshold: DefaultParallelThreshold}

        // Calculer avec Fast Doubling
        fd := &OptimizedFastDoubling{}
        resultFD, err := fd.CalculateCore(ctx, func(float64) {}, n, opts)
        if err != nil {
            t.Fatalf("FastDoubling failed for n=%d: %v", n, err)
        }

        // Calculer avec Matrix
        mx := &MatrixExponentiation{}
        resultMX, err := mx.CalculateCore(ctx, func(float64) {}, n, opts)
        if err != nil {
            t.Fatalf("Matrix failed for n=%d: %v", n, err)
        }

        // Vérifier la cohérence
        if resultFD.Cmp(resultMX) != 0 {
            t.Errorf("Inconsistent results for n=%d: FD=%s, MX=%s", 
                n, resultFD.String(), resultMX.String())
        }
    })
}

func FuzzFFTMultiplication(f *testing.F) {
    f.Add([]byte{1, 2, 3}, []byte{4, 5, 6})
    
    f.Fuzz(func(t *testing.T, a, b []byte) {
        // Convertir en big.Int et vérifier FFT vs standard
    })
}
```

### 4.3 Tests de charge automatisés

```go
// internal/server/server_stress_test.go

package server

import (
    "net/http"
    "sync"
    "testing"
    "time"
)

func TestServerUnderLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test in short mode")
    }

    srv := setupTestServer(t)
    defer srv.Shutdown(context.Background())

    concurrency := 100
    requestsPerClient := 50
    var wg sync.WaitGroup
    errors := make(chan error, concurrency*requestsPerClient)

    start := time.Now()

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(clientID int) {
            defer wg.Done()
            client := &http.Client{Timeout: 30 * time.Second}

            for j := 0; j < requestsPerClient; j++ {
                n := uint64((clientID*requestsPerClient + j) % 10000)
                resp, err := client.Get(fmt.Sprintf("http://localhost:8080/calculate?n=%d", n))
                if err != nil {
                    errors <- err
                    continue
                }
                resp.Body.Close()
                if resp.StatusCode != http.StatusOK {
                    errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
                }
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    duration := time.Since(start)
    totalRequests := concurrency * requestsPerClient
    rps := float64(totalRequests) / duration.Seconds()

    t.Logf("Completed %d requests in %v (%.2f req/s)", totalRequests, duration, rps)

    errorCount := 0
    for err := range errors {
        t.Logf("Error: %v", err)
        errorCount++
    }

    if errorCount > totalRequests/100 { // Plus de 1% d'erreurs
        t.Errorf("Too many errors: %d/%d", errorCount, totalRequests)
    }
}
```

### 4.4 Benchmarks de régression CI

```yaml
# .github/workflows/benchmark.yml

name: Benchmark

on:
  pull_request:
    branches: [main]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - name: Run benchmarks
        run: go test -bench=. -benchmem ./internal/fibonacci/ | tee benchmark.txt
      
      - name: Compare with main
        uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: benchmark.txt
          fail-on-alert: true
          alert-threshold: '150%'  # Alerte si >50% de régression
```

---

## DONE 5. 📦 Améliorations DevOps & Infrastructure

### DONE 5.1 Dockerfile amélioré

```dockerfile
# Dockerfile.improved

# Multi-stage build for optimal image size
# ========================================

# Stage 1: Build
FROM golang:1.25-alpine AS builder

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with version information
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w \
        -X main.Version=${VERSION} \
        -X main.Commit=${COMMIT} \
        -X main.BuildDate=${BUILD_DATE}" \
    -o /app/fibcalc \
    ./cmd/fibcalc

# Stage 2: Runtime
FROM alpine:3.19

# OCI Labels
LABEL org.opencontainers.image.title="Fibonacci Calculator"
LABEL org.opencontainers.image.description="High-performance Fibonacci number calculator"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/your-repo/fibcalc"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

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

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["/app/fibcalc"]
CMD ["--help"]
```

### DONE 5.2 Docker Compose complet

```yaml
# docker-compose.yml

version: '3.8'

services:
  fibcalc:
    build:
      context: .
      args:
        VERSION: ${VERSION:-dev}
        COMMIT: ${COMMIT:-unknown}
        BUILD_DATE: ${BUILD_DATE:-unknown}
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080", "--auto-calibrate"]
    environment:
      - GOMAXPROCS=4
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
      start_period: 10s
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    depends_on:
      - fibcalc

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
    depends_on:
      - prometheus

volumes:
  grafana-data:
```

### DONE 5.3 Kubernetes Manifests

```yaml
# kubernetes/deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: fibcalc
  labels:
    app: fibcalc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: fibcalc
  template:
    metadata:
      labels:
        app: fibcalc
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
        - name: fibcalc
          image: fibcalc:latest
          args: ["--server", "--port", "8080"]
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "2000m"
              memory: "2Gi"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: fibcalc
spec:
  selector:
    app: fibcalc
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: fibcalc
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: fibcalc
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### DONE 5.4 GitHub Actions CI/CD

```yaml
# .github/workflows/ci.yml

name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out

  build:
    needs: [lint, test]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Build
        run: make build-all
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: build/

  docker:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.sha }}
```

---

## 6. 📚 Améliorations Documentation

### 6.1 Nouveaux documents à créer

| Document | Description | Priorité |
|----------|-------------|----------|
| `ARCHITECTURE.md` | Diagrammes et décisions d'architecture ADR | Haute |
| `PERFORMANCE.md` | Guide de tuning et résultats de benchmarks | Haute |
| `SECURITY.md` | Politique de sécurité et signalement | Moyenne |
| `docs/algorithms/FAST_DOUBLING.md` | Explication mathématique détaillée | Moyenne |
| `docs/algorithms/MATRIX.md` | Explication de l'exponentiation matricielle | Moyenne |
| `docs/algorithms/FFT.md` | Explication de la multiplication FFT | Moyenne |

### 6.2 Structure documentation proposée

```
docs/
├── ARCHITECTURE.md           # Architecture globale et ADRs
├── PERFORMANCE.md            # Guide de performance
├── SECURITY.md               # Politique de sécurité
├── algorithms/
│   ├── FAST_DOUBLING.md     # Algorithme Fast Doubling
│   ├── MATRIX.md            # Exponentiation matricielle
│   ├── FFT.md               # Multiplication FFT
│   └── COMPARISON.md        # Comparaison des algorithmes
├── api/
│   ├── openapi.yaml         # Spécification OpenAPI 3.0
│   └── postman_collection.json
└── deployment/
    ├── DOCKER.md            # Guide Docker
    └── KUBERNETES.md        # Guide Kubernetes
```

### 6.3 Génération documentation API

```yaml
# docs/api/openapi.yaml

openapi: 3.0.3
info:
  title: Fibonacci Calculator API
  description: High-performance Fibonacci number calculator REST API
  version: 1.0.0
  license:
    name: Apache 2.0
    url: https://www.apache.org/licenses/LICENSE-2.0

servers:
  - url: http://localhost:8080
    description: Local development server

paths:
  /calculate:
    get:
      summary: Calculate a Fibonacci number
      description: Computes the n-th Fibonacci number using the specified algorithm
      parameters:
        - name: n
          in: query
          required: true
          schema:
            type: integer
            format: int64
            minimum: 0
            maximum: 1000000000
          description: The index of the Fibonacci number to calculate
        - name: algo
          in: query
          schema:
            type: string
            enum: [fast, matrix, fft]
            default: fast
          description: The algorithm to use
      responses:
        '200':
          description: Successful calculation
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CalculationResponse'
        '400':
          description: Invalid parameters
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '429':
          description: Rate limit exceeded

  /health:
    get:
      summary: Health check
      responses:
        '200':
          description: Service is healthy

  /algorithms:
    get:
      summary: List available algorithms
      responses:
        '200':
          description: List of algorithms

  /metrics:
    get:
      summary: Prometheus metrics
      responses:
        '200':
          description: Metrics in Prometheus format

components:
  schemas:
    CalculationResponse:
      type: object
      properties:
        n:
          type: integer
        result:
          type: string
        duration:
          type: string
        algorithm:
          type: string
          
    ErrorResponse:
      type: object
      properties:
        error:
          type: string
        message:
          type: string
```

---

## 7. DONE 🔐 Améliorations Sécurité

### DONE 7.1 Validations et protections

| Amélioration | Description | Priorité |
|--------------|-------------|----------|
| **Rate limiting adaptatif** | Basé sur la complexité du calcul demandé | Haute |
| **Timeout dynamique** | Ajuster automatiquement selon la valeur de N | Haute |
| **Sandboxing mémoire** | Limite mémoire par calcul individuel | Moyenne |
| **Audit logging** | Journalisation structurée des accès | Moyenne |
| **Input sanitization** | Validation stricte des entrées | Haute |

### DONE 7.2 Timeout dynamique

```go
// internal/server/timeout.go

package server

import (
    "math"
    "time"
)

// DynamicTimeout calcule un timeout approprié basé sur N
// La complexité est O(log n * M(n)) où M(n) est la complexité de multiplication
func DynamicTimeout(n uint64, baseTimeout time.Duration) time.Duration {
    if n <= 1000 {
        return time.Second * 5 // Minimum 5 secondes
    }

    // Estimation basée sur la complexité
    // F(n) a environ n * log2(φ) ≈ n * 0.694 bits
    estimatedBits := float64(n) * 0.694
    
    // La complexité est environ O(log(n) * n^1.585) pour Karatsuba
    // ou O(log(n) * n * log(n)) pour FFT
    logN := math.Log2(float64(n))
    complexityFactor := logN * math.Pow(estimatedBits/1000, 1.2)
    
    // Calculer le timeout
    estimated := time.Duration(complexityFactor) * time.Millisecond
    
    // Appliquer des limites
    minTimeout := time.Second * 5
    maxTimeout := baseTimeout
    
    if estimated < minTimeout {
        return minTimeout
    }
    if estimated > maxTimeout {
        return maxTimeout
    }
    
    return estimated
}
```

### DONE 7.3 Audit Logging

```go
// internal/server/audit.go

package server

import (
    "encoding/json"
    "log"
    "os"
    "time"
)

// AuditEntry représente une entrée d'audit
type AuditEntry struct {
    Timestamp   time.Time `json:"timestamp"`
    RequestID   string    `json:"request_id"`
    ClientIP    string    `json:"client_ip"`
    Method      string    `json:"method"`
    Path        string    `json:"path"`
    QueryParams string    `json:"query_params"`
    StatusCode  int       `json:"status_code"`
    Duration    string    `json:"duration"`
    UserAgent   string    `json:"user_agent"`
    Error       string    `json:"error,omitempty"`
}

// AuditLogger gère les logs d'audit
type AuditLogger struct {
    logger *log.Logger
    file   *os.File
}

// NewAuditLogger crée un nouveau logger d'audit
func NewAuditLogger(path string) (*AuditLogger, error) {
    file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    return &AuditLogger{
        logger: log.New(file, "", 0),
        file:   file,
    }, nil
}

// Log enregistre une entrée d'audit
func (a *AuditLogger) Log(entry AuditEntry) {
    data, _ := json.Marshal(entry)
    a.logger.Println(string(data))
}

// Close ferme le fichier d'audit
func (a *AuditLogger) Close() error {
    return a.file.Close()
}
```

---

## DONE 8. 🎨 Améliorations UX/UI CLI

### DONE 8.1 Barre de progression avec ETA

```go
// internal/cli/progress_eta.go

package cli

import (
    "fmt"
    "time"
)

// ProgressWithETA étend ProgressState avec estimation du temps restant
type ProgressWithETA struct {
    *ProgressState
    startTime    time.Time
    lastUpdate   time.Time
    progressRate float64 // progrès par seconde
}

// NewProgressWithETA crée un nouveau tracker avec ETA
func NewProgressWithETA(numCalculators int) *ProgressWithETA {
    return &ProgressWithETA{
        ProgressState: NewProgressState(numCalculators),
        startTime:     time.Now(),
        lastUpdate:    time.Now(),
    }
}

// UpdateWithETA met à jour le progrès et calcule l'ETA
func (p *ProgressWithETA) UpdateWithETA(index int, value float64) (progress float64, eta time.Duration) {
    p.Update(index, value)
    progress = p.CalculateAverage()
    
    now := time.Now()
    elapsed := now.Sub(p.startTime)
    
    if progress > 0 && progress < 1 {
        // Estimation linéaire simple
        totalEstimated := time.Duration(float64(elapsed) / progress)
        eta = totalEstimated - elapsed
        
        // Lissage exponentiel du taux de progression
        if p.progressRate > 0 {
            instantRate := (value - p.progresses[index]) / now.Sub(p.lastUpdate).Seconds()
            p.progressRate = 0.7*p.progressRate + 0.3*instantRate
        } else {
            p.progressRate = progress / elapsed.Seconds()
        }
    }
    
    p.lastUpdate = now
    return
}

// FormatProgressBar génère une barre avec ETA
func FormatProgressBarWithETA(progress float64, eta time.Duration, width int) string {
    bar := progressBar(progress, width)
    etaStr := "calculating..."
    
    if eta > 0 {
        if eta < time.Minute {
            etaStr = fmt.Sprintf("%ds", int(eta.Seconds()))
        } else if eta < time.Hour {
            etaStr = fmt.Sprintf("%dm%ds", int(eta.Minutes()), int(eta.Seconds())%60)
        } else {
            etaStr = fmt.Sprintf("%dh%dm", int(eta.Hours()), int(eta.Minutes())%60)
        }
    }
    
    return fmt.Sprintf("%6.2f%% [%s] ETA: %s", progress*100, bar, etaStr)
}
```

### DONE 8.2 Thèmes de couleur

```go
// internal/cli/themes.go

package cli

// Theme définit un schéma de couleurs
type Theme struct {
    Name      string
    Primary   string
    Secondary string
    Success   string
    Warning   string
    Error     string
    Info      string
    Reset     string
}

var (
    // DarkTheme pour terminaux sombres
    DarkTheme = Theme{
        Name:      "dark",
        Primary:   "\033[38;5;39m",  // Bleu clair
        Secondary: "\033[38;5;245m", // Gris
        Success:   "\033[38;5;82m",  // Vert vif
        Warning:   "\033[38;5;220m", // Jaune
        Error:     "\033[38;5;196m", // Rouge
        Info:      "\033[38;5;141m", // Violet
        Reset:     "\033[0m",
    }

    // LightTheme pour terminaux clairs
    LightTheme = Theme{
        Name:      "light",
        Primary:   "\033[38;5;27m",  // Bleu foncé
        Secondary: "\033[38;5;240m", // Gris foncé
        Success:   "\033[38;5;28m",  // Vert foncé
        Warning:   "\033[38;5;130m", // Orange
        Error:     "\033[38;5;124m", // Rouge foncé
        Info:      "\033[38;5;54m",  // Violet foncé
        Reset:     "\033[0m",
    }

    // NoColorTheme pour sortie sans couleur
    NoColorTheme = Theme{
        Name:      "none",
        Primary:   "",
        Secondary: "",
        Success:   "",
        Warning:   "",
        Error:     "",
        Info:      "",
        Reset:     "",
    }
)

// CurrentTheme est le thème actif
var CurrentTheme = DarkTheme

// SetTheme change le thème actif
func SetTheme(name string) {
    switch name {
    case "dark":
        CurrentTheme = DarkTheme
    case "light":
        CurrentTheme = LightTheme
    case "none":
        CurrentTheme = NoColorTheme
    }
}
```

---

## DONE 9. 🔄 Refactoring suggéré

### DONE 9.1 Code à refactorer

| Fichier | Ligne(s) | Problème | Solution |
|---------|----------|----------|----------|
| `main.go` | 44-48 | Registry hardcodé | Factory pattern |
| `fastdoubling.go` | 116-137 | Logique parallèle complexe | Extraire en fonction |
| `matrix.go` | 198-217 | Duplication séquentiel/parallèle | Template method |
| `calibration.go` | 168-213 | Fonction trop longue | Découper en sous-fonctions |
| `middleware.go` | 169-188 | Parsing IP complexe | Utiliser `net.SplitHostPort` |

### DONE 9.2 Factory Pattern pour le Registry

```go
// internal/fibonacci/registry.go

package fibonacci

// CalculatorFactory crée des calculateurs
type CalculatorFactory interface {
    Create(name string) (Calculator, error)
    List() []string
}

// DefaultFactory est la factory par défaut
type DefaultFactory struct {
    creators map[string]func() coreCalculator
}

// NewDefaultFactory crée une nouvelle factory
func NewDefaultFactory() *DefaultFactory {
    f := &DefaultFactory{
        creators: make(map[string]func() coreCalculator),
    }
    
    // Enregistrer les calculateurs par défaut
    f.Register("fast", func() coreCalculator { return &OptimizedFastDoubling{} })
    f.Register("matrix", func() coreCalculator { return &MatrixExponentiation{} })
    f.Register("fft", func() coreCalculator { return &FFTBasedCalculator{} })
    
    return f
}

// Register enregistre un nouveau type de calculateur
func (f *DefaultFactory) Register(name string, creator func() coreCalculator) {
    f.creators[name] = creator
}

// Create crée un calculateur par nom
func (f *DefaultFactory) Create(name string) (Calculator, error) {
    creator, ok := f.creators[name]
    if !ok {
        return nil, fmt.Errorf("unknown calculator: %s", name)
    }
    return NewCalculator(creator()), nil
}

// List retourne la liste des calculateurs disponibles
func (f *DefaultFactory) List() []string {
    names := make([]string, 0, len(f.creators))
    for name := range f.creators {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

---

## 📅 Roadmap suggérée

### Version 1.1 (Court terme - 1-2 mois)

- [ ] Endpoint `/batch` pour calculs multiples
- [ ] Documentation OpenAPI/Swagger
- [ ] CI/CD avec GitHub Actions
- [ ] Amélioration couverture tests à 85%
- [ ] Export résultats vers fichier (`-o`)
- [ ] Mode silencieux (`-q`)

### Version 1.2 (Moyen terme - 3-4 mois)

- [ ] Pool FFT pour optimisation mémoire
- [ ] Prometheus metrics complet
- [ ] Helm chart Kubernetes
- [ ] Mode interactif CLI (REPL)
- [ ] Calibration persistante
- [ ] Thèmes de couleur CLI
- [ ] ETA dans la barre de progression

### Version 2.0 (Long terme - 6+ mois)

- [ ] WebSocket pour streaming progress
- [ ] Algorithme de Lucas
- [ ] Optimisations SIMD (assembly)
- [ ] SDK clients (Python, JavaScript, Rust)
- [ ] Interface web de démonstration
- [ ] Cache LRU pour résultats
- [ ] JWT Authentication optionnelle
- [ ] Tracing distribué (OpenTelemetry)

---

## 📊 Métriques de succès

| Métrique | Actuel | Cible v1.2 | Cible v2.0 |
|----------|--------|------------|------------|
| Couverture tests | 75.2% | 85% | 90% |
| Temps F(10M) | ~80ms | ~60ms | ~40ms |
| Requêtes/sec (API) | ~500 | ~1000 | ~2000 |
| Taille Docker | ~15MB | ~12MB | ~10MB |
| Temps démarrage | ~100ms | ~50ms | ~30ms |

---

## 🤝 Contribution

Pour contribuer à ces améliorations :

1. Choisir une amélioration de la liste
2. Créer une issue GitHub avec le label `enhancement`
3. Forker le projet et créer une branche `feature/nom-amelioration`
4. Implémenter avec tests
5. Soumettre une Pull Request

Voir [CONTRIBUTING.md](CONTRIBUTING.md) pour les détails.

---

*Document généré automatiquement - Novembre 2025*

