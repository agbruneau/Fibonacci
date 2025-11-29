# Calculateur Haute Performance pour la Suite de Fibonacci

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
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
    - [Vérification](#vérification)
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
    - [Optimisation de Strassen](#optimisation-de-strassen)
  - [7. Optimisations de Performance](#7-optimisations-de-performance)
    - [Stratégie Zéro-Allocation](#stratégie-zéro-allocation)
    - [Parallélisme et Seuils Adaptatifs](#parallélisme-et-seuils-adaptatifs)
    - [Méthodologie de Calibration](#méthodologie-de-calibration)
  - [8. Tests](#8-tests)
    - [8.1 Exécuter les tests](#81-exécuter-les-tests)
    - [8.2 Exécuter l'application](#82-exécuter-lapplication)
  - [9. Développement](#9-développement)
    - [Makefile](#makefile)
    - [Structure du projet](#structure-du-projet)
    - [Documentation](#documentation)
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

- **Référence Technique** : Servir d'implémentation de référence pour des algorithmes mathématiques complexes (Fast Doubling, Strassen, FFT).
- **Architecture Propre** : Démontrer une architecture modulaire, testable et découplée (Clean Architecture).
- **Performance Extrême** : Illustrer des techniques d'optimisation avancées comme le recyclage de mémoire (`sync.Pool`), la concurrence fine et l'arithmétique adaptée au matériel.
- **Production-Ready** : Offrir une CLI robuste et une API REST avec gestion gracieuse de l'arrêt, monitoring et configuration dynamique.

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

### Vérification

Une fois le projet installé, il est recommandé de vérifier que tout fonctionne correctement en exécutant la suite de tests :

```bash
make test
# ou si Make n'est pas disponible :
go test ./...
```

Cette étape validera que votre environnement est correctement configuré et que le code est fonctionnel sur votre architecture.

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

- **Support des Grands Nombres** : Utilise `math/big` pour une arithmétique de précision arbitraire, capable de calculer des nombres de Fibonacci avec des millions de chiffres.
- **Algorithmes Multiples** :
  - **Fast Doubling (`fast`)** : L'algorithme par défaut. Combine complexité logarithmique, parallélisme et multiplication hybride (Karatsuba/FFT).
  - **Exponentiation Matricielle (`matrix`)** : Utilise la décomposition binaire de la puissance et l'algorithme de Strassen pour les grandes matrices.
  - **FFT-Based Doubling (`fft`)** : Force l'utilisation de la multiplication FFT pour tous les calculs (utile pour les benchmarks sur très grands N).
- **Mode Serveur HTTP** : Expose une API REST performante pour effectuer des calculs à la demande.
- **Sortie Structurée** : Support natif du format JSON (`--json`) pour l'intégration dans des pipelines de données.
- **Optimisations de Performance** :
  - **Stratégie Zéro-Allocation** : Emploie `sync.Pool` pour recycler les objets `big.Int` et minimiser la pression sur le GC.
  - **Parallélisme Adaptatif** : Utilisation intelligente des cœurs CPU basée sur des seuils configurables ou auto-calibrés.
  - **Algorithme de Strassen** : Réduit la complexité de la multiplication matricielle pour les grandes dimensions.
- **Architecture Modulaire** : Conception robuste avec séparation stricte des couches (Configuration, Calcul, Orchestration, Présentation).

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
| `-fft-threshold`       |             | Seuil en bits pour activer la multiplication FFT.          | `1000000`   |
| `--strassen-threshold` |             | Seuil en bits pour basculer vers l'algorithme de Strassen. | `3072`      |
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

### Optimisation de Strassen

Pour l'algorithme d'Exponentiation Matricielle, nous employons l'algorithme de Strassen lorsque la taille des nombres dépasse un certain seuil (`--strassen-threshold`).
Bien que la complexité standard de la multiplication matricielle soit $O(N^3)$ (ici N=2, donc constant), l'impact se situe au niveau des opérations sur les `big.Int`. Strassen réduit le nombre de multiplications scalaires de 8 à 7, ce qui, pour de très grands nombres, compense largement le coût des additions supplémentaires.

## 7. Optimisations de Performance

### Stratégie Zéro-Allocation

- **Pools d'Objets (`sync.Pool`)**: Pour minimiser la pression sur le ramasse-miettes, les états de calcul (`calculationState`, `matrixState`) sont recyclés. Cela élimine presque toutes les allocations de mémoire dans les boucles de calcul critiques.
- **Mise au Carré Symétrique** : Dans l'algorithme matriciel, une fonction spécialisée `squareSymmetricMatrix` est utilisée pour réduire le nombre de multiplications à seulement 4 (contre 8 en méthode naïve) lors de l'élévation au carré de matrices symétriques.

### Parallélisme et Seuils Adaptatifs

- **Parallélisme Multi-cœur**: Les multiplications de grands nombres sont exécutées en parallèle sur plusieurs goroutines.
- **Limitation FFT** : Une heuristique interne désactive le parallélisme si la multiplication FFT est utilisée sur des nombres inférieurs à **10 millions de bits**, afin d'éviter la saturation des ressources CPU (contention).
- **Seuils Empiriques**:
  - `--threshold` (défaut `4096` bits) : Active le parallélisme pour les multiplications classiques.
  - `--fft-threshold` (défaut `1000000` bits) : Active la multiplication FFT.
  - `--strassen-threshold` (défaut `3072` bits) : Bascule vers l'algorithme de Strassen.

### Méthodologie de Calibration

Le mode de calibration (`--calibrate`) permet d'ajuster finement les performances du calculateur à l'architecture de la machine hôte.

Le processus fonctionne comme suit :

1. **Benchmark Itératif** : Le calculateur exécute une série de calculs de Fibonacci pour une valeur fixe (par défaut N=10 000 000) en utilisant l'algorithme *Fast Doubling*.
2. **Variation du Seuil** : À chaque itération, le seuil de parallélisme (`--threshold`) varie parmi une liste prédéfinie de valeurs (séquentiel, 256, 512, ..., 16384 bits).
3. **Sélection de l'Optimum** : Le temps d'exécution est mesuré pour chaque seuil. Le seuil offrant le temps de calcul le plus court est identifié comme l'optimum pour la configuration matérielle actuelle.

## 8. Tests

Le projet inclut une suite de tests robuste pour garantir la correction et la stabilité.

### 8.1 Exécuter les tests

```bash
# Exécuter tous les tests (recommandé)
make test

# Tests unitaires courts
go test -v -short ./...

# Tests de propriété (gopter) et benchmarks
go test -bench=. -benchmem ./internal/fibonacci/

# Vérification de la couverture
make coverage
```

### 8.2 Exécuter l'application

Voir la section [Utilisation](#4-utilisation) pour les détails complets. Les modes principaux incluent :
- **CLI** : Pour des calculs ponctuels ou scripts.
- **Serveur** : Pour une utilisation en tant que micro-service.
- **Docker** : Pour le déploiement conteneurisé.

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
│   ├── cli/                       # Interface utilisateur CLI (spinner, barres)
│   ├── config/                    # Gestion de la configuration et flags CLI
│   ├── errors/                    # Gestion centralisée des erreurs
│   ├── fibonacci/                 # Cœur des algorithmes de calcul (Fast Doubling, Matrix, FFT)
│   ├── i18n/                      # Internationalisation (i18n)
│   ├── orchestration/             # Orchestration des calculs concurrents
│   └── server/                    # Serveur HTTP REST API
│
├── build/                         # Binaires compilés
├── API.md                         # 📖 Documentation complète de l'API REST
├── CHANGELOG.md                   # 📝 Historique des versions
├── Dockerfile                     # 🐳 Configuration Docker
├── Makefile                       # 🔧 Commandes de développement
└── README.md                      # 📚 Documentation principale
```

### Documentation

Le code source est documenté selon les conventions GoDoc. Utilisez `go doc ./...` pour lire la documentation des packages.

## 10. Déploiement

### Docker

```bash
docker build -t fibcalc:latest .
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080
```

### Docker Compose

```yaml
version: '3.8'
services:
  fibcalc:
    build: .
    ports: ["8080:8080"]
    command: ["--server", "--port", "8080", "--auto-calibrate"]
```

## 11. Ressources supplémentaires

- **[API.md](API.md)** - Documentation complète de l'API REST
- **[CHANGELOG.md](CHANGELOG.md)** - Historique des changements
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Guide de contribution

## 12. Licence

Ce projet est sous licence Apache 2.0. Voir le fichier `LICENSE` pour plus de détails.
