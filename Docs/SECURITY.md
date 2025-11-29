# Politique de Sécurité

> **Version** : 1.0.0  
> **Dernière mise à jour** : Novembre 2025

## Vue d'ensemble

Ce document décrit les mesures de sécurité implémentées dans le Calculateur Fibonacci et les bonnes pratiques pour son déploiement en production.

## Signalement de Vulnérabilités

### Contact

Si vous découvrez une vulnérabilité de sécurité, veuillez nous contacter de manière responsable :

- **Email** : security@example.com
- **PGP Key** : Disponible sur demande

### Processus de Divulgation

1. **Signalement** : Envoyez un email détaillant la vulnérabilité
2. **Accusé de réception** : Réponse sous 48 heures
3. **Évaluation** : Analyse sous 7 jours
4. **Correction** : Développement d'un correctif
5. **Publication** : Release coordonnée avec crédit au découvreur

### Informations à Fournir

- Description détaillée de la vulnérabilité
- Étapes de reproduction
- Impact potentiel
- Suggestions de correction (optionnel)

## Mesures de Sécurité Implémentées

### 1. Protection contre les Attaques par Déni de Service (DoS)

#### Limite sur la valeur de N

Le serveur limite la valeur maximale de N pour prévenir l'épuisement des ressources :

```go
// SecurityConfig dans internal/server/middleware.go
type SecurityConfig struct {
    MaxNValue uint64 // Défaut: 1_000_000_000
}
```

Requêtes avec N trop élevé retournent une erreur 400 :

```json
{
  "error": "Bad Request",
  "message": "Value of 'n' exceeds maximum allowed (1000000000)"
}
```

#### Rate Limiting

Le serveur implémente un rate limiter par IP :

```go
type RateLimiterConfig struct {
    RequestsPerSecond float64 // Défaut: 10
    BurstSize         int     // Défaut: 20
}
```

Les requêtes excédant la limite reçoivent une réponse 429 :

```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded. Please slow down."
}
```

#### Timeouts

Tous les calculs ont un timeout configurable :

```go
const (
    DefaultRequestTimeout  = 5 * time.Minute
    DefaultReadTimeout     = 10 * time.Second
    DefaultWriteTimeout    = 10 * time.Minute
    DefaultIdleTimeout     = 2 * time.Minute
    DefaultShutdownTimeout = 30 * time.Second
)
```

### 2. En-têtes de Sécurité HTTP

Le middleware de sécurité ajoute des en-têtes protecteurs :

```go
func SecurityMiddleware(config SecurityConfig, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // En-têtes de sécurité
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'none'")
        w.Header().Set("Referrer-Policy", "no-referrer")
        
        next(w, r)
    }
}
```

### 3. Validation des Entrées

Toutes les entrées utilisateur sont validées :

```go
// Validation du paramètre 'n'
n, err := strconv.ParseUint(nStr, 10, 64)
if err != nil {
    s.writeErrorResponse(w, http.StatusBadRequest, 
        "Invalid 'n' parameter: must be a positive integer")
    return
}

// Validation du paramètre 'algo'
calc, ok := s.registry[algo]
if !ok {
    s.writeErrorResponse(w, http.StatusBadRequest,
        fmt.Sprintf("Invalid 'algo' parameter: '%s' is not a valid algorithm", algo))
    return
}
```

### 4. Isolation Docker

Le Dockerfile implémente les bonnes pratiques de sécurité :

```dockerfile
# Utilisateur non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Image minimale (Alpine)
FROM alpine:latest

# Pas de shell pour l'utilisateur
ENTRYPOINT ["/app/fibcalc"]
```

### 5. Graceful Shutdown

Le serveur gère proprement les signaux d'arrêt pour éviter les interruptions brutales :

```go
func (s *Server) Start() error {
    signal.Notify(s.shutdownSignal, os.Interrupt, syscall.SIGTERM)
    
    // ... démarrage du serveur ...
    
    <-s.shutdownSignal
    s.logger.Println("Shutdown signal received...")
    
    ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
    defer cancel()
    
    return s.httpServer.Shutdown(ctx)
}
```

## Configuration Sécurisée

### Variables d'Environnement

| Variable | Description | Défaut |
|----------|-------------|--------|
| `FIBCALC_MAX_N` | Limite maximale pour N | 1,000,000,000 |
| `FIBCALC_RATE_LIMIT` | Requêtes par seconde | 10 |
| `FIBCALC_TIMEOUT` | Timeout des calculs | 5m |

### Flags de Ligne de Commande

```bash
# Configuration sécurisée recommandée
./fibcalc --server \
    --port 8080 \
    --timeout 2m
```

## Recommandations de Déploiement

### 1. Reverse Proxy (Nginx)

Placez le serveur derrière un reverse proxy pour :
- Terminaison TLS
- Rate limiting additionnel
- Logging des accès
- Protection DDoS

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

Utilisez des NetworkPolicies pour isoler le pod :

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
  egress: []  # Pas de sortie nécessaire
```

### 3. Ressources Limitées

Configurez des limites de ressources pour prévenir l'épuisement :

```yaml
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "2000m"
    memory: "2Gi"
```

### 4. Logging et Audit

Le serveur log toutes les requêtes :

```
[SERVER] 2025/11/29 10:15:32 GET /calculate from 192.168.1.100
[SERVER] 2025/11/29 10:15:32 GET /calculate completed in 125.5ms
```

Pour un audit complet, configurez un collecteur de logs externe (Fluentd, Loki, etc.).

## Checklist de Sécurité

### Avant le déploiement

- [ ] TLS configuré (certificats valides)
- [ ] Rate limiting activé
- [ ] Limites de ressources configurées
- [ ] Utilisateur non-root dans Docker
- [ ] NetworkPolicy appliquée (Kubernetes)
- [ ] Logs centralisés
- [ ] Monitoring des erreurs

### En production

- [ ] Mises à jour régulières des dépendances
- [ ] Analyse des logs pour anomalies
- [ ] Tests de pénétration périodiques
- [ ] Sauvegardes des profils de calibration
- [ ] Révision des accès

## Versions Supportées

| Version | Supportée | Fin de support |
|---------|-----------|----------------|
| 1.0.x | ✅ | Décembre 2026 |
| < 1.0 | ❌ | N/A |

## Dépendances

Les dépendances sont régulièrement auditées. Exécutez :

```bash
# Vérifier les vulnérabilités connues
go list -m all | nancy sleuth

# Ou avec govulncheck
govulncheck ./...
```

## Conformité

Ce projet suit les bonnes pratiques de :

- **OWASP** : Top 10 API Security Risks
- **CWE** : Common Weakness Enumeration
- **Go Security** : Recommandations officielles Go

