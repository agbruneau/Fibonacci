# Calculateur Haute Performance pour la Suite de Fibonacci

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage Status](https://img.shields.io/badge/coverage-75.2%25-brightgreen)

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

- **Calibrez pour votre machine :**

  ```bash
  ./fibcalc --calibrate
  ```

  Cela déterminera les paramètres de performance optimaux pour votre système.

- **Lancez une comparaison de tous les algorithmes :**

  ```bash
  ./fibcalc -n 10000000 -algo all
  ```

- **Mode Serveur API :**
  ```bash
  ./fibcalc --server --port 8080
  ```
  Puis testez :
  ```bash
  curl "http://localhost:8080/calculate?n=1000&algo=fast"
  curl "http://localhost:8080/health"
  curl "http://localhost:8080/algorithms"
  ```
  Voir [API.md](API.md) pour la documentation complète de l'API.

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
./fibcalc [options]
```

### Options

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

**Exécuter les tests :**

```bash
# Avec Make (recommandé)
make test

# Sans Make
go test ./... -v

# Exécuter les benchmarks
make benchmark
# ou
go test -bench=. -benchmem ./internal/fibonacci/

# Générer un rapport de couverture
make coverage
# ou
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
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
│   └── fibcalc/           # Point d'entrée de l'application
├── internal/
│   ├── calibration/       # Calibration automatique
│   ├── cli/               # Interface utilisateur CLI
│   ├── config/            # Gestion de la configuration
│   ├── errors/            # Gestion centralisée des erreurs
│   ├── fibonacci/         # Algorithmes de calcul
│   ├── i18n/              # Internationalisation
│   ├── orchestration/     # Orchestration des calculs
│   └── server/            # Serveur HTTP REST
├── API.md                 # Documentation de l'API REST
├── CHANGELOG.md           # Historique des changements
├── CONTRIBUTING.md        # Guide de contribution
├── Dockerfile             # Configuration Docker
├── Makefile               # Commandes de développement
└── README.md              # Ce fichier
```

### CI/CD

Le projet utilise GitHub Actions pour l'intégration continue :
- Tests automatiques sur Go 1.22, 1.23, et 1.25
- Vérification du linting
- Build multi-plateforme (Linux, Windows, macOS)
- Génération de rapports de couverture

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

- [API Documentation](API.md) - Documentation complète de l'API REST
- [Changelog](CHANGELOG.md) - Historique des versions et changements
- [Contributing Guidelines](CONTRIBUTING.md) - Guide pour contribuer au projet

## 12. Licence

Ce projet est sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.
