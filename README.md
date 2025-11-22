# Calculateur Haute Performance pour la Suite de Fibonacci

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage Status](https://img.shields.io/badge/coverage-75.2%25-brightgreen)

## 📋 Table des matières

- [Calculateur Haute Performance pour la Suite de Fibonacci](#calculateur-haute-performance-pour-la-suite-de-fibonacci)
  - [📋 Table des matières](#-table-des-matières)
  - [⚡ Démarrage ultra-rapide (30 secondes)](#-démarrage-ultra-rapide-30-secondes)
  - [1. Objectif](#1-objectif)
  - [2. Démarrage](#2-démarrage)
    - [Prérequis](#prérequis)
    - [Installation](#installation)
    - [Démarrage Rapide](#démarrage-rapide)
      - [Test rapide (30 secondes)](#test-rapide-30-secondes)
      - [Utilisation CLI](#utilisation-cli)
      - [Mode Serveur API](#mode-serveur-api)
      - [Docker](#docker)
  - [3. Fonctionnalités](#3-fonctionnalités)
  - [4. Utilisation](#4-utilisation)
    - [Commandes Essentielles (Référence Rapide)](#commandes-essentielles-référence-rapide)
    - [Options CLI](#options-cli)
    - [Exemples](#exemples)
  - [5. Architecture Logicielle](#5-architecture-logicielle)
  - [6. Analyse Algorithmique et Complexité](#6-analyse-algorithmique-et-complexité)
    - [Dérivation des Formules de Fast Doubling](#dérivation-des-formules-de-fast-doubling)
  - [7. Optimisations de Performance](#7-optimisations-de-performance)
    - [Stratégie Zéro-Allocation](#stratégie-zéro-allocation)
    - [Parallélisme et Seuils](#parallélisme-et-seuils)
    - [Méthodologie de Calibration](#méthodologie-de-calibration)
  - [8. Tests](#8-tests)
    - [8.1 Exécuter les tests](#81-exécuter-les-tests)
      - [Tests unitaires](#tests-unitaires)
      - [Benchmarks de performance](#benchmarks-de-performance)
      - [Couverture de code](#couverture-de-code)
    - [8.2 Exécuter l'application](#82-exécuter-lapplication)
      - [Compilation](#compilation)
      - [Mode CLI - Exemples d'utilisation](#mode-cli---exemples-dutilisation)
      - [Mode Serveur HTTP](#mode-serveur-http)
      - [Mode Docker](#mode-docker)
      - [Mode production](#mode-production)
  - [9. Développement](#9-développement)
    - [Makefile](#makefile)
    - [Structure du projet](#structure-du-projet)
    - [CI/CD](#cicd)
  - [10. Déploiement](#10-déploiement)
    - [Docker](#docker-1)
    - [Docker Compose](#docker-compose)
  - [11. Ressources supplémentaires](#11-ressources-supplémentaires)
  - [12. Licence](#12-licence)

---

## ⚡ Démarrage ultra-rapide (30 secondes)

```bash
# 1. Compiler le projet
make build   # ou: go build -o build/fibcalc ./cmd/fibcalc

# 2. Premier calcul
./build/fibcalc -n 100 -algo fast -d

# 3. Lancer le serveur API
make run-server   # ou: ./build/fibcalc --server --port 8080

# 4. Tester l'API
curl "http://localhost:8080/calculate?n=1000&algo=fast"
```

**Commandes essentielles :**
```bash
make test          # Exécuter tous les tests
make benchmark     # Benchmarks de performance
make coverage      # Rapport de couverture HTML
make run-fast      # Test rapide avec n=1000
make help          # Voir toutes les commandes disponibles
```

---

## 1. Objectif

Ce projet est un calculateur de Fibonacci haute performance et une étude de cas en ingénierie logicielle avancée avec Go. Il est conçu pour explorer et implémenter des algorithmes efficaces pour la manipulation de très grands entiers, en appliquant des optimisations de bas niveau et des patrons de conception de haut niveau pour maximiser la performance.

Les objectifs principaux sont :

- Servir de référence pour l'implémentation d'algorithmes sophistiqués en Go.
- Démontrer les meilleures pratiques en architecture logicielle, y compris la modularité et la testabilité.
- Fournir un exemple pratique de techniques d'optimisation de la performance.
- Offrir une API REST production-ready avec graceful shutdown et monitoring.

## 2. Démarrage

Suivez ces étapes pour mettre en service le calculateur de Fibonacci sur votre machine locale.

### Prérequis

- Go 1.25 ou une version ultérieure
- Make (optionnel, pour utiliser le Makefile)

### Installation

1. Clonez le dépôt :

   ```bash
   git clone https://github.com/votre-nom-utilisateur/fibcalc.git
   cd fibcalc
   ```

2. Compilez l'exécutable :
   
   **Avec Make (recommandé):**
   ```bash
   make build
   ```
   
   **Sans Make:**
   ```bash
   go build -o build/fibcalc ./cmd/fibcalc
   ```
   
   Cela créera un binaire dans le dossier `build/`.

3. (Optionnel) Installer globalement :
   ```bash
   make install
   # ou
   go install ./cmd/fibcalc
   ```

### Démarrage Rapide

#### Test rapide (30 secondes)

```bash
# 1. Compiler le projet
make build

# 2. Premier calcul
./build/fibcalc -n 100 -algo fast -d

# Résultat attendu: F(100) = 354,224,848,179,261,915,075
```

#### Utilisation CLI

- **Calcul simple :**
  ```bash
  ./build/fibcalc -n 1000 -algo fast
  ```

- **Comparaison de tous les algorithmes :**
  ```bash
  ./build/fibcalc -n 10000000 -algo all
  ```

- **Calibration (recommandé) :**
  ```bash
  ./build/fibcalc --calibrate
  # ou
  make run-calibrate
  ```
  Cela déterminera les paramètres de performance optimaux pour votre système.

#### Mode Serveur API

```bash
# Démarrer le serveur
make run-server
# ou
./build/fibcalc --server --port 8080
```

Puis testez les endpoints :
```bash
curl "http://localhost:8080/calculate?n=1000&algo=fast"
curl "http://localhost:8080/health"
curl "http://localhost:8080/algorithms"
```

Voir [API.md](API.md) pour la documentation complète de l'API.

#### Docker

```bash
# Build et exécution
make docker-build
docker run -d -p 8080:8080 fibcalc:1.0.0 --server --port 8080

# Test
curl "http://localhost:8080/calculate?n=1000"
```

## 3. Fonctionnalités

- **Support des Grands Nombres** : Utilise `math/big` pour une arithmétique de précision arbitraire.
- **Algorithmes Multiples** : Implémente plusieurs algorithmes en O(log n) :
  - **Fast Doubling** (`fast`)
  - **Exponentiation Matricielle** (`matrix`)
  - **Fast Doubling Basé sur la FFT** (`fft`)
- **Nouveautés** :
  - **Mode Serveur HTTP** : Expose une API REST pour effectuer des calculs à la demande.
  - **Sortie JSON** : Formatage structuré pour une intégration facile avec d'autres outils.
- **Optimisations de Performance** :
  - **Stratégie Zéro-Allocation** : Emploie `sync.Pool` pour minimiser la surcharge du ramasse-miettes.
  - **Parallélisme** : Tire parti des cœurs multiples pour des performances améliorées.
  - **Multiplication FFT Adaptative** : Bascule vers la multiplication basée sur la FFT pour les très grands nombres.
- **Architecture Modulaire** :
  - **Séparation des Préoccupations** : Découple la logique, la présentation et l'orchestration.
  - **Arrêt Propre** : Gère le cycle de vie de l'application avec `context`.
  - **Concurrence Structurée** : Utilise `golang.org/x/sync/errgroup` pour l'orchestration.

## 4. Utilisation

Le calculateur est contrôlé via des drapeaux de ligne de commande :

```bash
./build/fibcalc [options]
```

### Commandes Essentielles (Référence Rapide)

| Commande | Description |
|----------|-------------|
| `make build` | Compiler le projet |
| `make test` | Exécuter tous les tests |
| `make run-fast` | Test rapide (n=1000) |
| `make run-server` | Démarrer le serveur HTTP |
| `make run-calibrate` | Calibrer les performances |
| `make coverage` | Rapport de couverture HTML |
| `make benchmark` | Exécuter les benchmarks |
| `make docker-build` | Construire l'image Docker |
| `make clean` | Nettoyer les artifacts |
| `make help` | Afficher toutes les commandes |

### Options CLI

| Drapeau                | Alias       | Description                                                | Défaut      |
| ---------------------- | ----------- | ---------------------------------------------------------- | ----------- |
| `-n`                   |             | Index du nombre de Fibonacci à calculer.                   | `250000000` |
| `-algo`                |             | Algorithme à utiliser : `fast`, `matrix`, `fft`, ou `all`. | `all`       |
| `-timeout`             |             | Temps d'exécution maximum (ex: `10s`, `1m30s`).            | `5m`        |
| `-threshold`           |             | Seuil en bits pour paralléliser les multiplications.       | `4096`      |
| `-fft-threshold`       |             | Seuil en bits pour activer la multiplication FFT.          | `20000`     |
| `--strassen-threshold` |             | Seuil en bits pour basculer vers l'algorithme de Strassen. | `256`       |
| `-d`                   | `--details` | Afficher les détails de performance et les métadonnées.    | `false`     |
| `-v`                   |             | Afficher le résultat complet (peut être très long).        | `false`     |
| `--calibrate`          |             | Calibrer le seuil de parallélisme optimal.                 | `false`     |
| `--auto-calibrate`     |             | Lancer une calibration rapide au démarrage.                | `false`     |
| `--json`               |             | Afficher les résultats au format JSON.                     | `false`     |
| `--server`             |             | Démarrer en mode serveur HTTP.                             | `false`     |
| `--port`               |             | Port d'écoute pour le mode serveur.                        | `8080`      |
| `--lang`               |             | Langue pour l'i18n (ex: `fr`, `en`).                       | `en`        |
| `--i18n-dir`           |             | Répertoire des fichiers de traduction (ex: `./locales`).   | `""`        |

### Exemples

- **Sortie JSON pour intégration :**

  ```bash
  ./fibcalc -n 1000 --json
  ```

- **Calculez F(250,000,000) avec un timeout de 10 minutes :**
  ```bash
  ./fibcalc -n 250000000 -algo fast -d --timeout 10m
  ```

## 5. Architecture Logicielle

Ce projet est structuré selon les meilleures pratiques de l'ingénierie logicielle Go, en mettant l'accent sur la **modularité** et la **séparation des préoccupations**.

L'architecture est organisée comme suit :

- **`cmd/fibcalc`**: Point d'entrée. Orchestre l'initialisation et délègue l'exécution.
- **`internal/config`**: Gestion de la configuration et validation des drapeaux.
- **`internal/fibonacci`**: Cœur de la logique mathématique. Contient les algorithmes (`fast`, `matrix`, `fft`) et les optimisations bas niveau.
- **`internal/calibration`**: Logique de calibration automatique et manuelle des performances.
- **`internal/orchestration`**: Gestion de l'exécution concurrente des calculs et agrégation des résultats.
- **`internal/server`**: Implémentation du serveur HTTP pour l'exposition API.
- **`internal/cli`**: Gestion de l'interface utilisateur (spinner, barres de progression, formatage).
- **`internal/i18n`**: Gestion de l'internationalisation.

Cette conception en couches assure un **faible couplage** et facilite l'ajout de nouvelles fonctionnalités (comme le mode serveur récemment ajouté) sans perturber la logique existante.

## 6. Analyse Algorithmique et Complexité

La complexité `O(log n)` souvent citée pour les algorithmes de Fibonacci rapides se réfère au nombre d'opérations arithmétiques. Cependant, lors de l'utilisation de l'arithmétique de précision arbitraire (`math/big`), le coût de la multiplication `M(k)` pour des nombres de `k` bits devient le facteur dominant. Le nombre de bits dans F(n) est proportionnel à `n`.

La complexité réelle est donc `O(log n * M(n))`.

- Avec la multiplication de Karatsuba (utilisée par `math/big`), `M(n) ≈ O(n^1.585)`.
- Avec la multiplication basée sur la FFT, `M(n) ≈ O(n log n)`.

### Dérivation des Formules de Fast Doubling

Les identités de _Fast Doubling_ sont dérivées de la forme matricielle :

```
[ F(2k+1) F(2k)   ] = [ F(k+1)²+F(k)²     F(k)(2F(k+1)-F(k)) ]
[ F(2k)   F(2k-1) ]   [ F(k)(2F(k+1)-F(k)) F(k)²+F(k-1)²     ]
```

De là, nous extrayons :

- `F(2k) = F(k) * (2*F(k+1) - F(k))`
- `F(2k+1) = F(k+1)² + F(k)²`

## 7. Optimisations de Performance

### Stratégie Zéro-Allocation

- **Pools d'Objets (`sync.Pool`)**: Pour minimiser la pression sur le ramasse-miettes, les états de calcul (`calculationState`, `matrixState`) sont recyclés. Cela élimine presque toutes les allocations de mémoire dans les boucles de calcul critiques.

### Parallélisme et Seuils

- **Parallélisme Multi-cœur**: Les multiplications de grands nombres sont exécutées en parallèle sur plusieurs goroutines.
- **Seuils Empiriques**:
  - `--threshold` (défaut `4096` bits) : Active le parallélisme.
  - `--fft-threshold` (défaut `20000` bits) : Active la multiplication FFT.
  - `--strassen-threshold` (défaut `256` bits) : Utilise l'algorithme de Strassen pour la multiplication de matrices.

### Méthodologie de Calibration

Le mode de calibration (`--calibrate`) permet d'ajuster finement les performances du calculateur à l'architecture de la machine hôte.

Le processus fonctionne comme suit :

1. **Benchmark Itératif** : Le calculateur exécute une série de calculs de Fibonacci pour une valeur fixe (par défaut N=10 000 000) en utilisant l'algorithme *Fast Doubling*.
2. **Variation du Seuil** : À chaque itération, le seuil de parallélisme (`--threshold`) varie parmi une liste prédéfinie de valeurs (séquentiel, 256, 512, ..., 16384 bits).
3. **Sélection de l'Optimum** : Le temps d'exécution est mesuré pour chaque seuil. Le seuil offrant le temps de calcul le plus court est identifié comme l'optimum pour la configuration matérielle actuelle.

## 8. Tests

Le projet inclut une suite de tests robuste pour garantir la correction et la stabilité.

- **Tests Unitaires**: Valident les cas limites et les petites valeurs de `n`.
- **Tests de Propriétés**: Utilisent `gopter` pour effectuer des tests basés sur les propriétés.
- **Tests d'Intégration**: Valident le serveur HTTP et ses endpoints.
- **Benchmarks**: Mesurent la performance des différents algorithmes.

### 8.1 Exécuter les tests

#### Tests unitaires

```bash
# Exécuter tous les tests (recommandé avec Make)
make test

# Ou sans Make
go test ./... -v

# Tests courts uniquement (sans les tests longs)
make test-short
# Ou
go test -v -short ./...

# Tests avec affichage de la couverture
go test -v -cover ./...

# Tests avec détection de race conditions
go test -race ./...
```

#### Benchmarks de performance

```bash
# Exécuter les benchmarks (recommandé avec Make)
make benchmark

# Ou sans Make
go test -bench=. -benchmem ./internal/fibonacci/

# Benchmarks complets avec statistiques détaillées
go test -bench=. -benchmem -benchtime=10s ./internal/fibonacci/

# Benchmark d'un algorithme spécifique
go test -bench=BenchmarkFastDoubling -benchmem ./internal/fibonacci/
```

#### Couverture de code

```bash
# Générer et visualiser le rapport de couverture HTML
make coverage

# Ou manuellement
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Afficher la couverture par fonction
go tool cover -func=coverage.out

# Couverture avec seuil minimum
go test -cover -coverprofile=coverage.out ./... && \
  go tool cover -func=coverage.out | grep total | \
  awk '{if ($3+0 < 70) {print "Coverage below 70%"; exit 1}}'
```

### 8.2 Exécuter l'application

#### Compilation

```bash
# Compilation pour la plateforme actuelle
make build

# Ou sans Make
go build -o build/fibcalc ./cmd/fibcalc

# Compilation optimisée (binaire plus léger)
go build -ldflags="-s -w" -o build/fibcalc ./cmd/fibcalc

# Compilation pour toutes les plateformes
make build-all

# Installation globale dans $GOPATH/bin
make install
# Ou
go install ./cmd/fibcalc
```

#### Mode CLI - Exemples d'utilisation

```bash
# Calcul simple avec l'algorithme Fast Doubling
./build/fibcalc -n 1000 -algo fast

# Calcul avec affichage des détails de performance
./build/fibcalc -n 1000 -algo fast -d

# Calcul avec affichage du résultat complet
./build/fibcalc -n 100 -algo fast -v

# Comparaison de tous les algorithmes
./build/fibcalc -n 10000000 -algo all

# Calcul avec un timeout personnalisé
./build/fibcalc -n 500000000 -algo fast -d --timeout 10m

# Sortie au format JSON
./build/fibcalc -n 1000 -algo fast --json

# Test rapide prédéfini (avec Make)
make run-fast

# Calibration des seuils de performance
./build/fibcalc --calibrate
# Ou avec Make
make run-calibrate

# Calibration automatique rapide au démarrage
./build/fibcalc -n 10000000 -algo fast --auto-calibrate -d
```

#### Mode Serveur HTTP

```bash
# Démarrer le serveur (recommandé avec Make)
make run-server

# Ou manuellement
./build/fibcalc --server --port 8080

# Avec configuration personnalisée
./build/fibcalc --server --port 8080 \
  --threshold 8192 \
  --fft-threshold 25000 \
  --strassen-threshold 512

# Avec auto-calibration
./build/fibcalc --server --port 8080 --auto-calibrate

# Tester les endpoints de l'API
curl "http://localhost:8080/calculate?n=1000&algo=fast"
curl "http://localhost:8080/health"
curl "http://localhost:8080/algorithms"

# Calcul avec jq pour formater le JSON
curl -s "http://localhost:8080/calculate?n=100" | jq .
```

#### Mode Docker

```bash
# Construire l'image Docker
make docker-build
# Ou
docker build -t fibcalc:1.0.0 .

# Exécution CLI dans Docker
docker run --rm fibcalc:1.0.0 -n 1000 -algo fast -d

# Exécution serveur dans Docker (daemon)
docker run -d -p 8080:8080 --name fibcalc-server \
  fibcalc:1.0.0 --server --port 8080

# Voir les logs du serveur
docker logs -f fibcalc-server

# Arrêter le serveur
docker stop fibcalc-server

# Supprimer le conteneur
docker rm fibcalc-server

# Test de l'API dans Docker
curl "http://localhost:8080/calculate?n=1000&algo=fast"
```

#### Mode production

```bash
# Construction optimisée pour production
CGO_ENABLED=0 go build -ldflags="-s -w" -o fibcalc ./cmd/fibcalc

# Avec variables d'environnement
export GOMAXPROCS=8
./fibcalc --server --port 8080 --threshold 8192

# Exécution en arrière-plan avec nohup
nohup ./fibcalc --server --port 8080 > server.log 2>&1 &

# Avec systemd (créer un fichier /etc/systemd/system/fibcalc.service)
```

## 9. Développement

### Makefile

Le projet inclut un Makefile complet pour faciliter le développement :

```bash
make help          # Afficher toutes les commandes disponibles
make build         # Compiler le projet
make test          # Exécuter les tests
make coverage      # Générer un rapport de couverture
make benchmark     # Exécuter les benchmarks
make lint          # Vérifier le code avec golangci-lint
make format        # Formater le code
make check         # Exécuter toutes les vérifications
make run-fast      # Test rapide avec n=1000
make run-server    # Démarrer le serveur
make docker-build  # Construire l'image Docker
```

### Structure du projet

```
.
├── cmd/
│   └── fibcalc/                   # Point d'entrée de l'application
│       ├── main.go                # Logique principale et orchestration
│       └── main_test.go           # Tests d'intégration du main
│
├── internal/                      # Packages internes de l'application
│   ├── calibration/               # Calibration automatique des performances
│   │   └── calibration.go
│   ├── cli/                       # Interface utilisateur CLI (spinner, barres)
│   │   ├── ui.go
│   │   └── ui_test.go
│   ├── config/                    # Gestion de la configuration et flags CLI
│   │   └── config.go
│   ├── errors/                    # Gestion centralisée des erreurs
│   │   └── errors.go
│   ├── fibonacci/                 # Cœur des algorithmes de calcul
│   │   ├── calculator.go          # Interface Calculator et wrapper
│   │   ├── fastdoubling.go        # Fast Doubling (O(log n), parallèle)
│   │   ├── matrix.go              # Exponentiation matricielle (Strassen)
│   │   ├── fft_based.go           # Calculateur basé FFT
│   │   ├── fft.go                 # Multiplication FFT pour grands nombres
│   │   ├── fibonacci_test.go      # Tests unitaires
│   │   └── fibonacci_property_test.go  # Property-based testing
│   ├── i18n/                      # Internationalisation (i18n)
│   │   └── messages.go
│   ├── orchestration/             # Orchestration des calculs concurrents
│   │   └── orchestrator.go
│   └── server/                    # Serveur HTTP REST API
│       ├── server.go              # Implémentation serveur avec graceful shutdown
│       └── server_test.go         # Tests des endpoints
│
├── build/                         # Binaires compilés (créé par make build)
│
├── Books/                         # Documentation technique (PDFs)
│
├── API.md                         # 📖 Documentation complète de l'API REST
├── CHANGELOG.md                   # 📝 Historique des versions et changements
├── CONTRIBUTING.md                # 🤝 Guide pour contribuer au projet
├── Dockerfile                     # 🐳 Configuration Docker multi-stage
├── .dockerignore                  # Fichiers exclus du contexte Docker
├── Makefile                       # 🔧 Commandes de développement
├── go.mod                         # 📦 Dépendances Go modules
├── go.sum                         # 🔒 Checksums des dépendances
├── LICENSE                        # ⚖️ Licence MIT
└── README.md                      # 📚 Ce fichier - Documentation principale
```

### CI/CD

Le projet est prêt pour l'intégration continue avec GitHub Actions. Configuration suggérée :

```yaml
# .github/workflows/ci.yml (exemple)
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.23', '1.24', '1.25']
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}
      - run: make test
      - run: make benchmark
      - run: make coverage
```

**Intégration recommandée :**
- Tests automatiques sur Go 1.23+
- Vérification du linting avec `golangci-lint`
- Build multi-plateforme (Linux, Windows, macOS)
- Génération de rapports de couverture
- Tests de build Docker

## 10. Déploiement

### Docker

```bash
# Build
docker build -t fibcalc:latest .

# Run CLI
docker run --rm fibcalc:latest -n 1000 -algo fast

# Run server
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
    command: ["--server", "--port", "8080", "--auto-calibrate"]
    restart: unless-stopped
```

## 11. Ressources supplémentaires

- **[API.md](API.md)** - Documentation complète de l'API REST avec exemples d'intégration
- **[CHANGELOG.md](CHANGELOG.md)** - Historique des versions et changements
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Guide pour contribuer au projet
- **[Makefile](Makefile)** - Commandes de développement disponibles (voir section 9)

## 12. Licence

Ce projet est sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.
