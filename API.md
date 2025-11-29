# Documentation de l'API REST

> **Version** : 1.0.0  
> **Dernière mise à jour** : Novembre 2025

Ce document décrit les endpoints disponibles dans l'API REST du Calculateur Fibonacci.

## Vue d'ensemble

L'API REST permet d'effectuer des calculs de nombres de Fibonacci via HTTP. Elle inclut des protections de sécurité (rate limiting, validation des entrées) et expose des métriques de performance.

### URL de base

```
http://localhost:8080
```

### Headers de Sécurité

Tous les endpoints retournent les headers de sécurité suivants :
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy: default-src 'none'`
- `Referrer-Policy: no-referrer`

### Rate Limiting

L'API implémente un rate limiter par IP :
- **Requêtes par seconde** : 10
- **Burst autorisé** : 20

Les requêtes excédant la limite reçoivent une réponse `429 Too Many Requests`.

---

## Endpoints

### 1. Calculer un nombre de Fibonacci

Effectue le calcul du Nième nombre de Fibonacci en utilisant l'algorithme spécifié.

**URL** : `/calculate`  
**Méthode** : `GET`

#### Paramètres de Requête (Query Parameters)

| Paramètre | Type   | Requis | Description |
|-----------|--------|--------|-------------|
| `n`       | uint64 | Oui    | L'index du nombre de Fibonacci à calculer (doit être positif, max: 1,000,000,000). |
| `algo`    | string | Non    | L'algorithme à utiliser. Défaut: `fast`. Valeurs possibles : `fast`, `matrix`, `fft`. |

#### Exemple de Requête

```bash
curl "http://localhost:8080/calculate?n=100&algo=fast"
```

#### Réponse de Succès (200 OK)

```json
{
  "n": 100,
  "result": 354224848179261915075,
  "duration": "125.5µs",
  "algorithm": "fast"
}
```

#### Schéma de Réponse

| Champ | Type | Description |
|-------|------|-------------|
| `n` | uint64 | L'index du nombre de Fibonacci demandé |
| `result` | string/number | Le nombre de Fibonacci calculé (peut être très grand) |
| `duration` | string | Durée du calcul formatée |
| `algorithm` | string | L'algorithme utilisé pour le calcul |
| `error` | string | Message d'erreur (si applicable) |

#### Réponse d'Erreur (400 Bad Request)

**Paramètre `n` manquant :**
```json
{
  "error": "Bad Request",
  "message": "Missing 'n' parameter"
}
```

**Paramètre `n` invalide :**
```json
{
  "error": "Bad Request",
  "message": "Invalid 'n' parameter: must be a positive integer"
}
```

**Valeur de `n` trop grande :**
```json
{
  "error": "Bad Request",
  "message": "Value of 'n' exceeds maximum allowed (1000000000). This limit prevents resource exhaustion."
}
```

**Algorithme invalide :**
```json
{
  "error": "Bad Request",
  "message": "Invalid 'algo' parameter: 'unknown' is not a valid algorithm"
}
```

#### Réponse Rate Limit (429 Too Many Requests)

```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded. Please slow down."
}
```

---

### 2. Vérification de Santé (Health Check)

Vérifie si le serveur est en ligne et fonctionnel.

**URL** : `/health`  
**Méthode** : `GET`

#### Exemple de Requête

```bash
curl "http://localhost:8080/health"
```

#### Réponse de Succès (200 OK)

```json
{
  "status": "healthy",
  "timestamp": 1732900800
}
```

#### Schéma de Réponse

| Champ | Type | Description |
|-------|------|-------------|
| `status` | string | État de santé du service ("healthy") |
| `timestamp` | int64 | Timestamp Unix de la réponse |

---

### 3. Lister les Algorithmes

Renvoie la liste des algorithmes de calcul disponibles sur le serveur.

**URL** : `/algorithms`  
**Méthode** : `GET`

#### Exemple de Requête

```bash
curl "http://localhost:8080/algorithms"
```

#### Réponse de Succès (200 OK)

```json
{
  "algorithms": [
    "fast",
    "fft",
    "matrix"
  ]
}
```

#### Schéma de Réponse

| Champ | Type | Description |
|-------|------|-------------|
| `algorithms` | []string | Liste des noms d'algorithmes disponibles |

---

### 4. Métriques du Serveur

Expose les métriques de performance du serveur pour le monitoring.

**URL** : `/metrics`  
**Méthode** : `GET`

#### Exemple de Requête

```bash
curl "http://localhost:8080/metrics"
```

#### Réponse de Succès (200 OK)

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

#### Schéma de Réponse

| Champ | Type | Description |
|-------|------|-------------|
| `uptime` | string | Temps depuis le démarrage du serveur |
| `total_requests` | int64 | Nombre total de requêtes HTTP reçues |
| `total_calculations` | int64 | Nombre total de calculs effectués |
| `calculations_by_algorithm` | object | Statistiques détaillées par algorithme |
| `rate_limit_hits` | int64 | Nombre de requêtes bloquées par rate limiting |
| `active_connections` | int | Nombre de connexions actives |

#### Statistiques par Algorithme

| Champ | Type | Description |
|-------|------|-------------|
| `count` | int64 | Nombre total de calculs |
| `success` | int64 | Nombre de calculs réussis |
| `errors` | int64 | Nombre de calculs en erreur |
| `total_duration` | string | Durée totale cumulée |
| `avg_duration` | string | Durée moyenne par calcul |

---

## Codes de Statut HTTP

| Code | Signification |
|------|---------------|
| `200 OK` | Requête réussie |
| `400 Bad Request` | Paramètres invalides |
| `405 Method Not Allowed` | Méthode HTTP non supportée |
| `429 Too Many Requests` | Rate limit dépassé |
| `500 Internal Server Error` | Erreur interne du serveur |

---

## Exemples avec cURL

### Calcul simple

```bash
curl "http://localhost:8080/calculate?n=50"
```

### Calcul avec algorithme spécifique

```bash
curl "http://localhost:8080/calculate?n=10000&algo=matrix"
```

### Calcul d'un très grand nombre

```bash
curl "http://localhost:8080/calculate?n=1000000&algo=fast"
```

### Pipeline avec jq

```bash
# Extraire uniquement le résultat
curl -s "http://localhost:8080/calculate?n=100&algo=fast" | jq '.result'

# Extraire la durée
curl -s "http://localhost:8080/calculate?n=100000&algo=fast" | jq '.duration'
```

---

## Configuration du Serveur

### Démarrage

```bash
# Port par défaut (8080)
./fibcalc --server

# Port personnalisé
./fibcalc --server --port 3000

# Avec auto-calibration
./fibcalc --server --port 8080 --auto-calibrate

# Avec timeout personnalisé
./fibcalc --server --port 8080 --timeout 10m
```

### Variables d'Environnement

| Variable | Description | Défaut |
|----------|-------------|--------|
| `FIBCALC_MAX_N` | Limite maximale pour N | 1,000,000,000 |
| `FIBCALC_RATE_LIMIT` | Requêtes par seconde | 10 |
| `FIBCALC_TIMEOUT` | Timeout des calculs | 5m |

### Timeouts

| Paramètre | Valeur | Description |
|-----------|--------|-------------|
| Request Timeout | 5 minutes | Timeout maximum par calcul |
| Read Timeout | 10 secondes | Timeout lecture requête |
| Write Timeout | 10 minutes | Timeout écriture réponse |
| Idle Timeout | 2 minutes | Timeout connexion inactive |
| Shutdown Timeout | 30 secondes | Timeout arrêt gracieux |

---

## Intégration

### Docker

```bash
# Démarrer le serveur
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080

# Tester
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

## Sécurité

### Protection DoS

- **Limite sur N** : La valeur maximale de N est limitée à 1 milliard.
- **Rate Limiting** : 10 requêtes/seconde par IP avec burst de 20.
- **Timeouts** : Tous les calculs ont un timeout configurable.

### Validation des Entrées

Toutes les entrées utilisateur sont strictement validées :
- Le paramètre `n` doit être un entier positif.
- Le paramètre `algo` doit correspondre à un algorithme enregistré.

### Logging

Le serveur journalise toutes les requêtes :
```
[SERVER] 2025/11/29 10:15:32 GET /calculate from 192.168.1.100
[SERVER] 2025/11/29 10:15:32 GET /calculate completed in 125.5ms
```

---

## Voir aussi

- [README.md](README.md) - Documentation principale
- [Docs/SECURITY.md](Docs/SECURITY.md) - Politique de sécurité complète
- [Docs/PERFORMANCE.md](Docs/PERFORMANCE.md) - Guide de performance
- [Docs/api/openapi.yaml](Docs/api/openapi.yaml) - Spécification OpenAPI
