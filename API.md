# API Documentation

## Overview

Le serveur HTTP expose une API REST pour calculer des nombres de Fibonacci. Le serveur supporte plusieurs algorithmes optimisés et inclut un système de logging, des timeouts, et un graceful shutdown.

## Endpoints

### 1. POST /calculate

Calcule le n-ième nombre de Fibonacci.

**Paramètres de requête:**
- `n` (obligatoire): L'indice du nombre de Fibonacci à calculer (entier positif)
- `algo` (optionnel): L'algorithme à utiliser. Valeurs possibles: `fast`, `matrix`, `fft`. Défaut: `fast`

**Exemple de requête:**
```bash
curl "http://localhost:8080/calculate?n=1000&algo=fast"
```

**Réponse (succès):**
```json
{
  "n": 1000,
  "result": "43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875",
  "duration": "123.456µs",
  "algorithm": "fast"
}
```

**Réponse (erreur de calcul):**
```json
{
  "n": 1000000000,
  "duration": "5m0s",
  "algorithm": "fast",
  "error": "context deadline exceeded"
}
```

**Réponse (erreur de paramètres):**
```json
{
  "error": "Bad Request",
  "message": "Invalid 'n' parameter: must be a positive integer"
}
```

**Codes de statut:**
- `200 OK`: Calcul réussi (même si une erreur de timeout s'est produite)
- `400 Bad Request`: Paramètres invalides
- `405 Method Not Allowed`: Méthode HTTP non autorisée

---

### 2. GET /health

Vérifie l'état de santé du serveur.

**Exemple de requête:**
```bash
curl "http://localhost:8080/health"
```

**Réponse:**
```json
{
  "status": "healthy",
  "timestamp": 1732204800
}
```

**Codes de statut:**
- `200 OK`: Le serveur est opérationnel

---

### 3. GET /algorithms

Liste tous les algorithmes disponibles.

**Exemple de requête:**
```bash
curl "http://localhost:8080/algorithms"
```

**Réponse:**
```json
{
  "algorithms": [
    "fast",
    "fft",
    "matrix"
  ]
}
```

**Codes de statut:**
- `200 OK`: Liste retournée avec succès

---

## Configuration du serveur

Le serveur utilise les seuils de configuration suivants (configurables via les flags CLI):

- **Threshold** (parallélisation): Par défaut 4096 bits
- **FFT Threshold**: Par défaut 20000 bits
- **Strassen Threshold**: Par défaut 256 bits

Ces valeurs peuvent être ajustées au démarrage :

```bash
./fibcalc --server --port 8080 --threshold 8192 --fft-threshold 25000
```

## Timeouts

- **Timeout de requête**: 5 minutes maximum par calcul
- **Read timeout**: 10 secondes
- **Write timeout**: 10 minutes
- **Idle timeout**: 2 minutes
- **Shutdown timeout**: 30 secondes

## Exemples d'utilisation

### 1. Calcul basique

```bash
curl "http://localhost:8080/calculate?n=10"
```

### 2. Calcul avec algorithme spécifique

```bash
curl "http://localhost:8080/calculate?n=1000000&algo=fft"
```

### 3. Calcul avec jq pour formater

```bash
curl -s "http://localhost:8080/calculate?n=100" | jq .
```

### 4. Mesure du temps de réponse

```bash
time curl "http://localhost:8080/calculate?n=10000000&algo=fast"
```

### 5. Vérifier la santé du serveur

```bash
curl "http://localhost:8080/health"
```

### 6. Obtenir la liste des algorithmes

```bash
curl "http://localhost:8080/algorithms" | jq '.algorithms[]'
```

## Démarrage du serveur

### Mode standard

```bash
./fibcalc --server --port 8080
```

### Avec auto-calibration

```bash
./fibcalc --server --port 8080 --auto-calibrate
```

### Avec configuration personnalisée

```bash
./fibcalc --server \
  --port 8080 \
  --threshold 8192 \
  --fft-threshold 25000 \
  --strassen-threshold 512
```

## Déploiement

### Docker

```bash
# Build
docker build -t fibcalc:latest .

# Run
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080
```

### Docker Compose

```yaml
version: '3.8'
services:
  fibcalc:
    build: .
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080"]
    restart: unless-stopped
```

## Graceful Shutdown

Le serveur supporte un arrêt gracieux. Lorsqu'il reçoit un signal SIGTERM ou SIGINT :

1. Il arrête d'accepter de nouvelles connexions
2. Il attend que les requêtes en cours se terminent (max 30 secondes)
3. Il s'arrête proprement

```bash
# Arrêt gracieux
kill -SIGTERM <pid>
# ou
Ctrl+C
```

## Monitoring

Le serveur log toutes les requêtes avec les informations suivantes :
- Méthode HTTP
- Chemin
- IP du client
- Durée de traitement

Exemple de log :
```
[SERVER] 2024/11/22 10:30:00 GET /calculate from 127.0.0.1:52345
[SERVER] 2024/11/22 10:30:05 GET /calculate completed in 5.123s
```

## Benchmarking

### ApacheBench

```bash
ab -n 100 -c 10 "http://localhost:8080/calculate?n=1000&algo=fast"
```

### wrk

```bash
wrk -t4 -c100 -d30s "http://localhost:8080/calculate?n=1000&algo=fast"
```

## Intégration

### Python

```python
import requests

response = requests.get('http://localhost:8080/calculate', params={
    'n': 1000,
    'algo': 'fast'
})
data = response.json()
print(f"F({data['n']}) = {data['result']}")
```

### JavaScript/Node.js

```javascript
const axios = require('axios');

async function fibonacci(n, algo = 'fast') {
  const response = await axios.get('http://localhost:8080/calculate', {
    params: { n, algo }
  });
  return response.data;
}

fibonacci(1000).then(data => {
  console.log(`F(${data.n}) = ${data.result}`);
});
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Response struct {
    N        uint64 `json:"n"`
    Result   string `json:"result"`
    Duration string `json:"duration"`
    Algorithm string `json:"algorithm"`
}

func main() {
    resp, _ := http.Get("http://localhost:8080/calculate?n=1000&algo=fast")
    defer resp.Body.Close()
    
    var result Response
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Printf("F(%d) = %s\n", result.N, result.Result)
}
```
