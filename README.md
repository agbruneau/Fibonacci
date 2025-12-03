# Calculateur de Suite de Fibonacci Haute Performance

<div align="center">

![Go version](https://img.shields.io/badge/Go-1.23+-blue.svg?style=for-the-badge&logo=go)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg?style=for-the-badge)
![Coverage Status](https://img.shields.io/badge/coverage-75.2%25-brightgreen?style=for-the-badge)

</div>

---

## 📋 Table des Matières

- [Calculateur de Suite de Fibonacci Haute Performance](#calculateur-de-suite-de-fibonacci-haute-performance)
  - [📋 Table des Matières](#-table-des-matières)
  - [⚡ Démarrage Rapide](#-démarrage-rapide)
  - [🎥 Démo](#-démo)
  - [🚀 Performance vs Baseline](#-performance-vs-baseline)
  - [1. Objectif](#1-objectif)
  - [2. Pour Commencer](#2-pour-commencer)
    - [Prérequis](#prérequis)
    - [Installation](#installation)
    - [Vérification](#vérification)
  - [3. Fonctionnalités](#3-fonctionnalités)
  - [4. Utilisation](#4-utilisation)
  - [5. Architecture Logicielle](#5-architecture-logicielle)
  - [6. Algorithmes](#6-algorithmes)
  - [7. Optimisations de Performance](#7-optimisations-de-performance)
  - [8. Tests](#8-tests)
  - [9. Développement](#9-développement)
  - [10. Déploiement](#10-déploiement)
  - [11. Documentation](#11-documentation)
  - [12. Licence](#12-licence)

---

## ⚡ Démarrage Rapide

Profitez de la puissance de Go moderne sans installation complexe.

```bash
# 🚀 Lancer immédiatement (Nécessite Go installé)
go run ./cmd/fibcalc -n 100000 -algo fast

# 🛠️ Ou compiler pour des performances maximales
make build
./build/fibcalc -n 1000000
```

> **Pas de Go ?** Utilisez Docker :
> `docker run --rm fibcalc -n 1000`

---

## 🎥 Démo

Voyez `fibcalc` en action, calculant le 1 000 000ème nombre de Fibonacci en moins de 100ms.

```console
$ ./build/fibcalc -n 1000000 --algo fast
🚀 Calculating Fibonacci for n=1000000...
Algorithm: Fast Doubling (O(log n), Parallel)

✅ F(1000000) calculated in 85ms
   Bits:      694,242
   Digits:    208,988
   Value:     19532821287077577316320149475962563... (truncated)
```

---

## 🚀 Performance vs Baseline

Comparaison du temps d'exécution sur un processeur standard (Ryzen 9 5900X).
L'algorithme **Fast Doubling** surpasse significativement l'approche matricielle standard pour les grands nombres.

| N (Index) | Fast Doubling | Matrix Exp. | Accélération |
|-----------|---------------|-------------|--------------|
| 1,000 | **15µs** | 18µs | 1.2x |
| 100,000 | **3.2ms** | 4.1ms | 1.3x |
| 1,000,000 | **85ms** | 110ms | 1.3x |
| 10,000,000 | **2.1s** | 2.8s | 1.35x |
| 100,000,000 | **45s** | 62s | **1.4x** |

> **Note :** Une implémentation itérative naïve (O(n)) prendrait des **années** pour calculer F(100,000,000). Nos algorithmes logarithmiques (O(log n)) le font en moins d'une minute.

---

## 1. Objectif

Ce projet est un calculateur de Fibonacci haute performance et une étude de cas en ingénierie logicielle avancée avec Go. Il est conçu pour explorer et implémenter des algorithmes efficaces pour gérer de très grands entiers, en appliquant des optimisations bas niveau et des modèles de conception haut niveau pour maximiser les performances.

Les principaux objectifs sont :

- **Référence Technique** : Servir d'implémentation de référence pour des algorithmes mathématiques complexes (Fast Doubling, Strassen, FFT).
- **Architecture Propre** : Démontrer une architecture modulaire, testable et découplée (Clean Architecture).
- **Performance Extrême** : Illustrer des techniques d'optimisation avancées telles que le recyclage de mémoire (`sync.Pool`), la concurrence fine et l'arithmétique adaptée au matériel.
- **Prêt pour la Production** : Offrir une CLI robuste, un mode REPL interactif et une API REST avec arrêt gracieux, surveillance et configuration dynamique.

## 2. Pour Commencer

Suivez ces étapes pour configurer le calculateur de Fibonacci sur votre machine locale.

### Prérequis

- Go 1.23 ou ultérieur
- Make (optionnel, pour utiliser le Makefile)

### Installation

1. Cloner le dépôt :

   ```bash
   git clone https://github.com/votre-nom-utilisateur/fibcalc.git
   cd fibcalc
   ```

2. Compiler l'exécutable :

   **Avec Make (recommandé) :**

   ```bash
   make build
   ```

   **Sans Make :**

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

## 3. Fonctionnalités

- **Support des Grands Nombres** : Utilise `math/big` pour l'arithmétique à précision arbitraire, capable de calculer des nombres de Fibonacci avec des millions de chiffres.
- **Algorithmes Multiples** :
  - **Fast Doubling (`fast`)** : L'algorithme par défaut. Combine complexité logarithmique, parallélisme et multiplication hybride (Karatsuba/FFT).
  - **Matrix Exponentiation (`matrix`)** : Utilise la décomposition binaire de l'exposant et l'algorithme de Strassen pour les grandes matrices.
  - **FFT-Based Doubling (`fft`)** : Force l'utilisation de la multiplication FFT pour tous les calculs.
- **Modes d'Exécution Multiples** :
  - **CLI** : Calculs ponctuels via ligne de commande.
  - **Mode Interactif (REPL)** : Session interactive pour plusieurs calculs.
  - **Mode Serveur HTTP** : API REST haute performance pour calculs à la demande.
  - **Docker** : Déploiement conteneurisé prêt pour la production.
- **Sortie Flexible** :
  - Format JSON (`--json`) pour intégration dans des pipelines.
  - Export vers fichier (`-o, --output`).
  - Affichage hexadécimal (`--hex`).
  - Mode silencieux (`-q, --quiet`) pour les scripts.
- **Optimisations de Performance** :
  - **Stratégie Zéro-Allocation** : Utilise `sync.Pool` pour recycler les objets `big.Int`.
  - **Allocateur d'Arène** : Allocation mémoire adaptative avec pré-estimation et préchauffage de pool.
  - **Architecture Modulaire** : Cadres réutilisables et stratégies de multiplication interchangeables.
  - **Parallélisme Multi-niveaux** : Parallélisation au niveau algorithme et au niveau FFT interne.
  - **Algorithme de Strassen** : Réduit la complexité de multiplication matricielle.
  - **Calibration Automatique** : Détection des seuils optimaux pour le matériel.
- **Sécurité** : Limitation de débit, validation des entrées, en-têtes de sécurité HTTP, protection DoS.

## 4. Utilisation

Le calculateur est contrôlé via des drapeaux en ligne de commande :

```bash
./build/fibcalc [options]
```

### Commandes Essentielles

| Commande             | Description                    |
| -------------------- | ------------------------------ |
| `make build`         | Compiler le projet             |
| `make test`          | Exécuter tous les tests        |
| `make run-fast`      | Test rapide (n=1000)           |
| `make run-server`    | Démarrer le serveur HTTP       |
| `make run-calibrate` | Calibrer les performances      |
| `make coverage`      | Rapport de couverture HTML     |
| `make benchmark`     | Exécuter les benchmarks        |
| `make docker-build`  | Construire l'image Docker      |
| `make clean`         | Nettoyer les artefacts de build|
| `make help`          | Afficher toutes les commandes  |

### Options CLI Complètes

| Drapeau                 | Alias       | Description                                                          | Défaut                        |
| ----------------------- | ----------- | -------------------------------------------------------------------- | ----------------------------- |
| `-n`                    |             | Index du nombre de Fibonacci à calculer.                             | `250000000`                   |
| `-algo`                 |             | Algorithme : `fast`, `matrix`, `fft`, ou `all`.                      | `all`                         |
| `-timeout`              |             | Temps d'exécution maximum (ex: `10s`, `1m30s`).                      | `5m`                          |
| `-threshold`            |             | Seuil en bits pour paralléliser les multiplications.                 | `4096`                        |
| `-fft-threshold`        |             | Seuil en bits pour activer la multiplication FFT.                    | `1000000`                     |
| `--strassen-threshold`  |             | Seuil en bits pour l'algorithme de Strassen.                         | `3072`                        |
| `-d`                    | `--details` | Afficher les détails de performance.                                 | `false`                       |
| `-v`                    |             | Afficher le résultat complet (peut être très long).                  | `false`                       |
| `--calibrate`           |             | Calibrer le seuil de parallélisme optimal.                           | `false`                       |
| `--auto-calibrate`      |             | Calibration rapide au démarrage.                                     | `false`                       |
| `--calibration-profile` |             | Chemin vers le fichier de profil de calibration.                     | `~/.fibcalc_calibration.json` |
| `--json`                |             | Sortie au format JSON.                                               | `false`                       |
| `--server`              |             | Démarrer en mode serveur HTTP.                                       | `false`                       |
| `--port`                |             | Port d'écoute pour le mode serveur.                                  | `8080`                        |
| `--interactive`         |             | Démarrer en mode interactif (REPL).                                  | `false`                       |
| `-o`                    | `--output`  | Sauvegarder le résultat dans un fichier.                             | `""`                          |
| `-q`                    | `--quiet`   | Mode silencieux (sortie minimale).                                   | `false`                       |
| `--hex`                 |             | Afficher le résultat en hexadécimal.                                 | `false`                       |
| `--no-color`            |             | Désactiver les couleurs (respecte aussi `NO_COLOR`).                 | `false`                       |
| `--completion`          |             | Générer un script d'autocomplétion (bash, zsh, fish, powershell).    | `""`                          |
| `--version`             | `-V`        | Afficher la version du programme.                                    |                               |

### Configuration via Variables d'Environnement

En plus des drapeaux CLI, `fibcalc` peut être configuré via des variables d'environnement. C'est particulièrement utile pour les déploiements Docker et Kubernetes, suivant les meilleures pratiques [12-Factor App](https://12factor.net/config).

**Priorité de Configuration :** Drapeaux CLI > Variables d'Environnement > Valeurs par Défaut

| Variable                      | Type     | Description                           | Défaut      |
| ----------------------------- | -------- | ------------------------------------- | ----------- |
| `FIBCALC_N`                   | uint64   | Index du nombre de Fibonacci          | `250000000` |
| `FIBCALC_ALGO`                | string   | Algorithme (fast, matrix, fft, all)   | `all`       |
| `FIBCALC_PORT`                | string   | Port du serveur HTTP                  | `8080`      |
| `FIBCALC_TIMEOUT`             | duration | Timeout (ex: "5m", "30s")             | `5m`        |
| `FIBCALC_THRESHOLD`           | int      | Seuil de parallélisme (bits)          | `4096`      |
| `FIBCALC_FFT_THRESHOLD`       | int      | Seuil FFT (bits)                      | `1000000`   |
| `FIBCALC_STRASSEN_THRESHOLD`  | int      | Seuil Strassen (bits)                 | `3072`      |
| `FIBCALC_SERVER`              | bool     | Mode serveur (true/false)             | `false`     |
| `FIBCALC_JSON`                | bool     | Sortie JSON                           | `false`     |
| `FIBCALC_VERBOSE`             | bool     | Mode verbeux                          | `false`     |
| `FIBCALC_QUIET`               | bool     | Mode silencieux                       | `false`     |
| `FIBCALC_HEX`                 | bool     | Sortie hexadécimale                   | `false`     |
| `FIBCALC_INTERACTIVE`         | bool     | Mode REPL                             | `false`     |
| `FIBCALC_NO_COLOR`            | bool     | Désactiver les couleurs               | `false`     |
| `FIBCALC_OUTPUT`              | string   | Fichier de sortie                     | `""`        |
| `FIBCALC_CALIBRATION_PROFILE` | string   | Fichier de calibration                | `""`        |

**Exemples :**

```bash
# Calcul simple via variable d'environnement
FIBCALC_N=1000 FIBCALC_ALGO=fast ./build/fibcalc

# Serveur avec configuration par environnement
export FIBCALC_SERVER=true
export FIBCALC_PORT=9090
export FIBCALC_THRESHOLD=8192
./build/fibcalc

# Les drapeaux CLI sont toujours prioritaires
FIBCALC_N=99999 ./build/fibcalc -n 100  # Utilisera n=100
```

**Docker Compose :**

```yaml
services:
  fibcalc:
    image: fibcalc:latest
    ports:
      - "8080:8080"
    environment:
      - FIBCALC_SERVER=true
      - FIBCALC_PORT=8080
      - FIBCALC_THRESHOLD=8192
      - FIBCALC_FFT_THRESHOLD=500000
      - FIBCALC_TIMEOUT=10m
```

### Mode Interactif (REPL)

Le mode interactif vous permet d'effectuer plusieurs calculs dans une session :

```bash
./build/fibcalc --interactive
```

**Commandes disponibles dans le REPL :**

| Commande                    | Description                              |
| --------------------------- | ---------------------------------------- |
| `calc <n>` ou `c <n>`       | Calculer F(n) avec l'algorithme actuel   |
| `algo <nom>` ou `a <nom>`   | Changer l'algorithme (fast, matrix, fft) |
| `compare <n>` ou `cmp <n>`  | Comparer tous les algorithmes pour F(n)  |
| `list` ou `ls`              | Lister les algorithmes disponibles       |
| `hex`                       | Basculer l'affichage hexadécimal         |
| `status` ou `st`            | Afficher la configuration actuelle       |
| `help` ou `h`               | Afficher l'aide                          |
| `exit` ou `quit`            | Quitter le mode interactif               |

**Exemple de session REPL :**

```
fib> calc 1000
Calculating F(1000) with Fast Doubling (O(log n), Parallel, Zero-Alloc)...

Result:
  Time: 15.2µs
  Bits:  693
  Digits: 209
  F(1000) = 43466...03811 (truncated)

fib> algo matrix
Algorithm changed to: Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)

fib> compare 10000
Comparison for F(10000):
─────────────────────────────────────────────
  fast                : 180.5µs ✓
  matrix              : 220.3µs ✓
  fft                 : 350.1µs ✓
─────────────────────────────────────────────

fib> exit
Goodbye!
```

### Mode Serveur API

```bash
# Démarrer le serveur
make run-server
# ou
./build/fibcalc --server --port 8080
```

**Points de terminaison disponibles :**

| Endpoint      | Méthode | Description                              |
| ------------- | ------- | ---------------------------------------- |
| `/calculate`  | GET     | Calculer F(n) avec l'algorithme spécifié |
| `/health`     | GET     | Vérification de l'état du serveur        |
| `/algorithms` | GET     | Lister les algorithmes disponibles       |
| `/metrics`    | GET     | Métriques de performance du serveur      |

**Exemples de requêtes :**

```bash
# Calcul simple
curl "http://localhost:8080/calculate?n=1000&algo=fast"

# Vérification de santé
curl "http://localhost:8080/health"

# Lister les algorithmes
curl "http://localhost:8080/algorithms"

# Métriques
curl "http://localhost:8080/metrics"
```

Voir [API.md](API.md) pour la documentation complète de l'API.

### Exemples d'Utilisation

**Sortie JSON pour intégration :**

```bash
./build/fibcalc -n 1000 --json
```

**Calcul avec export vers fichier :**

```bash
./build/fibcalc -n 100000 -algo fast -o result.txt
```

**Calcul silencieux pour scripts :**

```bash
./build/fibcalc -n 1000 -q
```

**Affichage hexadécimal :**

```bash
./build/fibcalc -n 1000 --hex -d
```

**Calculer F(250,000,000) avec un timeout de 10 minutes :**

```bash
./build/fibcalc -n 250000000 -algo fast -d --timeout 10m
```

**Générer l'autocomplétion Bash :**

```bash
./build/fibcalc --completion bash > /etc/bash_completion.d/fibcalc
```

**Utilisation avec Docker :**

```bash
# Construire et exécuter
make docker-build
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080

# Tester
curl "http://localhost:8080/calculate?n=1000"
```

## 5. Architecture Logicielle

Ce projet est structuré selon les meilleures pratiques d'ingénierie logicielle Go, avec un accent sur la **modularité** et la **séparation des préoccupations**.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           POINTS D'ENTRÉE                               │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌──────────┐ │
│  │   Mode CLI  │    │ Mode Serveur│    │   Docker    │    │   REPL   │ │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    └────┬─────┘ │
└─────────┼──────────────────┼──────────────────┼────────────────┼───────┘
          └──────────────────┼──────────────────┘                │
                             ▼                                   ▼
                     ┌───────────────┐                  ┌────────────────┐
                     │ cmd/fibcalc   │                  │ internal/cli   │
                     │   main.go     │                  │   repl.go      │
                     └───────┬───────┘                  └────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────────────┐
│                  COUCHE D'ORCHESTRATION                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   config    │  │ calibration │  │   server    │  │orchestration│    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└────────────────────────────┼────────────────────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────────────┐
│                      COUCHE MÉTIER                                      │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    internal/fibonacci                             │  │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐  │  │
│  │  │  Fast Doubling   │  │     Matrix       │  │    FFT-Based   │  │  │
│  │  │  O(log n)        │  │  Exponentiation  │  │    Doubling    │  │  │
│  │  └──────────────────┘  └──────────────────┘  └────────────────┘  │  │
│  │                            │                                      │  │
│  │                    ┌───────┴───────────────────────────────────┐  │  │
│  │                    │           internal/bigfft                 │  │  │
│  │                    │  Multiplication FFT pour très grand N     │  │  │
│  │                    └───────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

**Packages principaux :**

- **`cmd/fibcalc`** : Point d'entrée. Orchestre l'initialisation et délègue l'exécution.
- **`internal/fibonacci`** : Logique mathématique centrale (Fast Doubling, Matrix, FFT).
  - `strategy.go` : Interface et implémentations des stratégies de multiplication.
  - `doubling_framework.go` : Cadre réutilisable pour Fast Doubling.
  - `matrix_framework.go` : Cadre pour l'exponentiation matricielle.
- **`internal/calibration`** : Calibration de performance automatique et manuelle.
- **`internal/orchestration`** : Gestion de l'exécution concurrente des calculs.
- **`internal/server`** : Serveur API REST HTTP avec sécurité et métriques.
- **`internal/cli`** : Interface utilisateur (spinner, barres, thèmes, REPL).
- **`internal/bigfft`** : Multiplication FFT pour les très grands nombres.
  - `arena.go` : Allocateur d'arène et estimation mémoire.
  - `pool.go` : Système de pool avec préchauffage.
  - `fft.go` : Implémentation FFT avec parallélisation interne.
- **`internal/config`** : Gestion de la configuration et validation des drapeaux.
- **`internal/errors`** : Gestion centralisée des erreurs.

Voir [Docs/ARCHITECTURE.md](Docs/ARCHITECTURE.md) pour les détails complets.

## 6. Algorithmes

| Algorithme                | Drapeau        | Complexité         | Description                                                                                                    |
| ------------------------- | -------------- | ------------------ | -------------------------------------------------------------------------------------------------------------- |
| **Fast Doubling**         | `-algo fast`   | O(log n × M(n))    | Le plus performant. 3 multiplications par itération. Utilise DoublingFramework avec stratégie adaptative.      |
| **Matrix Exponentiation** | `-algo matrix` | O(log n × M(n))    | Approche matricielle avec optimisation Strassen. Utilise MatrixFramework.                                      |
| **FFT-Based**             | `-algo fft`    | O(log n × n log n) | Force l'utilisation de la multiplication FFT pour tous les calculs. Utilise DoublingFramework avec stratégie FFT-only. |

**Note** : Tous les algorithmes partagent désormais des cadres communs qui éliminent la duplication de code et facilitent la maintenance. Les stratégies de multiplication peuvent être interchangées dynamiquement.

### Dérivation de la Formule Fast Doubling

Les identités _Fast Doubling_ sont dérivées de la forme matricielle :

```math
F(2k)   = F(k) × [2×F(k+1) - F(k)]
F(2k+1) = F(k+1)^2 + F(k)^2
```

## 7. Optimisations de Performance

Le projet intègre plusieurs niveaux d'optimisations avancées pour maximiser les performances :

### Stratégie Zéro-Allocation

- **Pools d'Objets (`sync.Pool`)** : Les états de calcul sont recyclés pour minimiser la pression sur le GC.
- **Allocateur d'Arène** : Système d'allocation mémoire adaptatif qui pré-estime les besoins mémoire basés sur N et préchauffe les pools globaux pour réduire les allocations durant le calcul.
- **Mise au Carré Symétrique** : Réduit le nombre de multiplications à 4 (comparé à 8 avec la méthode naïve).

### Optimisation PGO (Profile-Guided Optimization)

Le projet supporte l'optimisation guidée par profil (PGO), disponible depuis Go 1.20.
- **Principe** : Le compilateur utilise un profil d'exécution réel (`default.pgo`) pour optimiser les chemins de code critiques (inlining, dévirtualisation).
- **Gain** : **~5-10%** d'amélioration de performance sur les grands calculs.
- **Utilisation** : `make build-pgo` utilise automatiquement le profil inclus.

### Architecture Modulaire avec Stratégies

- **MultiplicationStrategy** : Abstraction permettant le choix dynamique entre différentes méthodes de multiplication (Adaptatif, FFT-only, Karatsuba).
- **DoublingFramework** : Cadre réutilisable qui élimine la duplication de code entre les implémentations Fast Doubling et FFT-Based.
- **MatrixFramework** : Cadre similaire pour l'exponentiation matricielle, facilitant la maintenance et l'extension.

### Parallélisme Multi-niveaux

- **Parallélisme Multi-cœur** : Les multiplications sont exécutées en parallèle au niveau de l'algorithme.
- **Parallélisation FFT Interne** : La récursion FFT est parallélisée pour les grandes transformations, exploitant efficacement plusieurs cœurs CPU durant les calculs FFT.
- **Seuils Configurables** :
  - `--threshold` (défaut `4096` bits) : Active le parallélisme au niveau algorithme.
  - `--fft-threshold` (défaut `1000000` bits) : Active la multiplication FFT.
  - `--strassen-threshold` (défaut `3072` bits) : Active l'algorithme de Strassen.

### Optimisations Mémoire Avancées

- **Estimation Mémoire Préalable** : Le système estime les besoins mémoire avant le calcul basé sur F(n) taille ≈ n × 0.694 bits.
- **Préchauffage de Pool** : Les pools mémoire sont préchauffés avec des tampons optimaux selon les besoins estimés, réduisant les allocations à chaud.
- **Réutilisation de Tampon** : Les tampons temporaires sont réutilisés efficacement via le système de pooling.

### Calibration

```bash
# Calibration complète (recommandé)
./build/fibcalc --calibrate

# Calibration rapide au démarrage
./build/fibcalc --auto-calibrate -n 100000000
```

### Gains de Performance Attendus

Les optimisations récentes apportent les améliorations suivantes :

- **Réduction d'Allocation** : 10-20% de réduction de la pression GC grâce à l'allocateur d'arène et au préchauffage.
- **Amélioration de Maintenabilité** : Code plus modulaire et extensible grâce aux cadres et stratégies.
- **Parallélisation FFT** : Gains significatifs pour N > 100M où la FFT domine les calculs.

Voir [Docs/PERFORMANCE.md](Docs/PERFORMANCE.md) pour le guide de réglage complet.

## 8. Tests

Le projet inclut une suite de tests robuste :

```bash
# Exécuter tous les tests
make test

# Tests unitaires courts
go test -v -short ./...

# Tests de propriété (gopter) et benchmarks
go test -bench=. -benchmem ./internal/fibonacci/

# Vérification de couverture
make coverage

# Tests de fuzzing
go test -fuzz=FuzzFastDoublingConsistency ./internal/fibonacci/
```

**Types de tests inclus :**

- Tests unitaires
- Tests de propriété (gopter)
- Tests de fuzzing (Go 1.18+)
- Benchmarks
- Tests d'intégration HTTP
- Tests de charge/stress

## 9. Développement

### Makefile

```bash
make help          # Afficher toutes les commandes
make build         # Compiler le projet
make build-all     # Compiler pour toutes les plateformes
make test          # Exécuter les tests
make coverage      # Générer le rapport de couverture
make benchmark     # Exécuter les benchmarks
make lint          # Vérifier le code avec golangci-lint
make format        # Formater le code
make check         # Exécuter toutes les vérifications
make tidy          # Nettoyer go.mod et go.sum
make deps          # Télécharger les dépendances
make upgrade       # Mettre à jour les dépendances
```

### Structure du Projet

```
.
├── cmd/
│   └── fibcalc/                   # Point d'entrée de l'application
│       ├── main.go                # Logique principale
│       └── main_test.go           # Tests d'intégration
│
├── internal/                      # Packages internes
│   ├── bigfft/                    # Multiplication FFT pour big.Int
│   ├── calibration/               # Calibration automatique
│   ├── cli/                       # Interface CLI (spinner, REPL, thèmes)
│   ├── config/                    # Configuration et drapeaux
│   ├── errors/                    # Gestion centralisée des erreurs
│   ├── fibonacci/                 # Algorithmes de calcul
│   ├── orchestration/             # Orchestration des calculs
│   ├── server/                    # Serveur REST HTTP
│   └── testutil/                  # Utilitaires de test
│
├── Docs/                          # Documentation détaillée
│   ├── algorithms/                # Documentation des algorithmes
│   │   ├── COMPARISON.md
│   │   ├── FAST_DOUBLING.md
│   │   ├── FFT.md
│   │   ├── MATRIX.md
│   ├── api/                       # Documentation API
│   │   ├── openapi.yaml
│   │   └── postman_collection.json
│   ├── deployment/                # Guides de déploiement
│   │   ├── DOCKER.md
│   │   └── KUBERNETES.md
│   ├── ARCHITECTURE.md            # Architecture du projet
│   ├── PERFORMANCE.md             # Guide de performance
│   └── SECURITY.md                # Politique de sécurité
│
├── API.md                         # 📖 Documentation API REST
├── CONTRIBUTING.md                # 🤝 Guide de contribution
├── Dockerfile                     # 🐳 Configuration Docker
├── go.mod                         # Dépendances Go
├── go.sum                         # Checksums des dépendances
├── LICENSE                        # Licence Apache 2.0
├── Makefile                       # 🔧 Commandes de développement
└── README.md                      # 📚 Documentation principale
```

## 10. Déploiement

### Docker

```bash
# Construire l'image
docker build -t fibcalc:latest .

# Exécuter en mode CLI
docker run --rm fibcalc:latest -n 1000 -algo fast -d

# Exécuter en mode serveur
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080
```

### Docker Compose

```yaml
version: "3.8"

services:
  fibcalc:
    build: .
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080", "--auto-calibrate"]
    deploy:
      resources:
        limits:
          cpus: "4"
          memory: 2G
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped
```

### Kubernetes

Voir [Docs/deployment/KUBERNETES.md](Docs/deployment/KUBERNETES.md) pour les manifestes Kubernetes complets.

### Recommandations de Ressources

| Usage             | CPU      | RAM    |
| ----------------- | -------- | ------ |
| Petit (N < 100K)  | 1 cœur   | 512 MB |
| Moyen (N < 10M)   | 2 cœurs  | 1 GB   |
| Grand (N > 10M)   | 4+ cœurs | 2+ GB  |

## 11. Documentation

| Document                                     | Description                |
| -------------------------------------------- | -------------------------- |
| [README.md](README.md)                       | Documentation principale   |
| [API.md](API.md)                             | Documentation API REST     |
| [CONTRIBUTING.md](CONTRIBUTING.md)           | Guide de contribution      |
| [Docs/ARCHITECTURE.md](Docs/ARCHITECTURE.md) | Architecture du projet     |
| [Docs/PERFORMANCE.md](Docs/PERFORMANCE.md)   | Guide de performance       |
| [Docs/SECURITY.md](Docs/SECURITY.md)         | Politique de sécurité      |
| [Docs/algorithms/](Docs/algorithms/)         | Documentation algorithmes  |
| [Docs/deployment/](Docs/deployment/)         | Guides de déploiement      |

## 12. Licence

Ce projet est licencié sous la licence Apache 2.0. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

---

_Développé avec ❤️ en Go - Novembre 2025_
