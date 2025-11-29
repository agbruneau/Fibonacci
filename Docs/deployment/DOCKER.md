# Guide de Déploiement Docker

> **Version** : 1.0.0  
> **Dernière mise à jour** : Novembre 2025

## Prérequis

- Docker 20.10+
- Docker Compose 2.0+ (optionnel)
- 512 MB RAM minimum (2 GB recommandé pour grands calculs)

## Construction de l'Image

### Build Standard

```bash
# Build l'image avec le tag par défaut
docker build -t fibcalc:latest .

# Build avec un tag de version spécifique
docker build -t fibcalc:1.0.0 .
```

### Build avec Arguments

```bash
# Build avec informations de version injectées
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t fibcalc:1.0.0 .
```

### Multi-architecture

```bash
# Build pour multiple architectures (AMD64 + ARM64)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t fibcalc:1.0.0 \
  --push .
```

## Exécution

### Mode CLI

```bash
# Calcul simple
docker run --rm fibcalc:latest -n 1000 -algo fast -d

# Calcul avec tous les algorithmes
docker run --rm fibcalc:latest -n 10000 -algo all

# Sortie JSON
docker run --rm fibcalc:latest -n 1000 --json

# Calibration
docker run --rm fibcalc:latest --calibrate
```

### Mode Serveur

```bash
# Démarrer le serveur sur le port 8080
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080

# Vérifier que le serveur fonctionne
curl http://localhost:8080/health

# Voir les logs
docker logs -f fibcalc-server

# Arrêter le serveur
docker stop fibcalc-server
docker rm fibcalc-server
```

### Options Avancées

```bash
# Avec auto-calibration au démarrage
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080 --auto-calibrate

# Avec limites de ressources
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  --memory=2g \
  --cpus=4 \
  fibcalc:latest --server --port 8080

# Avec timeout personnalisé
docker run -d \
  --name fibcalc-server \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080 --timeout 10m
```

## Docker Compose

### Configuration Simple

Créez un fichier `docker-compose.yml` :

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
# Démarrer
docker-compose up -d

# Voir les logs
docker-compose logs -f

# Arrêter
docker-compose down
```

### Configuration avec Monitoring

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

Fichier `prometheus.yml` correspondant :

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'fibcalc'
    static_configs:
      - targets: ['fibcalc:8080']
    metrics_path: '/metrics'
```

## Dockerfile Expliqué

```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder

# Dépendances de build
RUN apk add --no-cache git make

WORKDIR /app

# Cache des dépendances Go
COPY go.mod go.sum ./
RUN go mod download

# Copie du code source
COPY . .

# Build optimisé
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/fibcalc \
    ./cmd/fibcalc

# Stage 2: Runtime (image minimale)
FROM alpine:latest

# Certificats pour HTTPS
RUN apk --no-cache add ca-certificates

# Utilisateur non-root (sécurité)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copie du binaire depuis le builder
COPY --from=builder /app/fibcalc .

# Permissions
RUN chown -R appuser:appgroup /app

# Exécution en non-root
USER appuser

# Port exposé
EXPOSE 8080

# Point d'entrée
ENTRYPOINT ["/app/fibcalc"]
CMD ["--help"]
```

## Bonnes Pratiques

### 1. Taille de l'Image

L'image finale fait environ 15 MB grâce à :
- Multi-stage build
- Image de base Alpine
- Binaire Go statique (CGO_ENABLED=0)
- Stripping des symboles (-ldflags="-s -w")

### 2. Sécurité

- Utilisateur non-root (`appuser`)
- Image de base minimale (Alpine)
- Pas de shell interactif nécessaire
- Healthcheck intégré

### 3. Performances

```bash
# Recommandations de ressources
# - Petit usage : 1 CPU, 512 MB RAM
# - Usage moyen : 2 CPUs, 1 GB RAM
# - Grands calculs : 4+ CPUs, 2+ GB RAM

docker run -d \
  --cpus=4 \
  --memory=2g \
  --memory-swap=2g \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080
```

### 4. Persistance de la Calibration

```bash
# Monter un volume pour persister le profil de calibration
docker run -d \
  -v fibcalc-data:/home/appuser \
  -p 8080:8080 \
  fibcalc:latest --server --port 8080 --auto-calibrate
```

## Dépannage

### Le conteneur ne démarre pas

```bash
# Vérifier les logs
docker logs fibcalc-server

# Exécuter en mode interactif
docker run --rm -it fibcalc:latest --help
```

### Performances dégradées

```bash
# Vérifier les ressources
docker stats fibcalc-server

# Augmenter les limites
docker update --cpus=8 --memory=4g fibcalc-server
```

### Port déjà utilisé

```bash
# Utiliser un autre port
docker run -d -p 9090:8080 fibcalc:latest --server --port 8080
```

## Intégration CI/CD

### GitHub Actions

```yaml
name: Docker Build

on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.ref_name }}
          build-args: |
            VERSION=${{ github.ref_name }}
            COMMIT=${{ github.sha }}
            BUILD_DATE=${{ github.event.head_commit.timestamp }}
```

## Registres Supportés

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

