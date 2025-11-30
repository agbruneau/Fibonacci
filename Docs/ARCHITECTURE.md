# Architecture du Calculateur Fibonacci

> **Version** : 1.1.0  
> **Dernière mise à jour** : Novembre 2025

## Vue d'ensemble

Le Calculateur Fibonacci est conçu selon les principes de la **Clean Architecture**, avec une séparation stricte des responsabilités et un faible couplage entre les modules. Cette architecture permet une testabilité maximale, une évolutivité aisée et une maintenance simplifiée.

## Diagramme d'Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           POINTS D'ENTRÉE                               │
│                                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   CLI Mode  │  │ Server Mode │  │   Docker    │  │ REPL Mode   │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
│         │                │                │                │           │
│         └────────────────┼────────────────┼────────────────┘           │
│                          ▼                ▼                            │
│                    ┌───────────────┐ ┌────────────────┐                │
│                    │ cmd/fibcalc   │ │ internal/cli   │                │
│                    │   main.go     │ │   repl.go      │                │
│                    └───────┬───────┘ └───────┬────────┘                │
└────────────────────────────┼─────────────────┼──────────────────────────┘
                             │                 │
                             └────────┬────────┘
                                      │
┌─────────────────────────────────────┼───────────────────────────────────┐
│                   COUCHE ORCHESTRATION                                  │
│                                     ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    internal/orchestration                        │   │
│  │  • ExecuteCalculations() - Exécution parallèle des algorithmes  │   │
│  │  • AnalyzeComparisonResults() - Analyse et comparaison          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                            │                                           │
│  ┌─────────────────────────┼───────────────────────────────────────┐   │
│  │                         ▼                                        │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │   config    │  │ calibration │  │   server    │              │   │
│  │  │   Parsing   │  │   Tuning    │  │   HTTP API  │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────┼────────────────────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────────────┐
│                      COUCHE MÉTIER                                      │
│                            ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    internal/fibonacci                            │   │
│  │                                                                  │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐ │   │
│  │  │  Fast Doubling   │  │     Matrix       │  │    FFT-Based   │ │   │
│  │  │  O(log n)        │  │  Exponentiation  │  │    Doubling    │ │   │
│  │  │  Parallel        │  │  O(log n)        │  │    O(log n)    │ │   │
│  │  │  Zero-Alloc      │  │  Strassen        │  │    FFT Mul     │ │   │
│  │  └──────────────────┘  └──────────────────┘  └────────────────┘ │   │
│  │                            │                                     │   │
│  │                            ▼                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐│   │
│  │  │                    internal/bigfft                          ││   │
│  │  │  • Multiplication FFT pour très grands nombres              ││   │
│  │  │  • Complexité O(n log n) vs O(n^1.585) pour Karatsuba       ││   │
│  │  └─────────────────────────────────────────────────────────────┘│   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────┼────────────────────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────────────┐
│                   COUCHE PRÉSENTATION                                   │
│                            ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      internal/cli                                │   │
│  │  • Spinner et barre de progression avec ETA                     │   │
│  │  • Formatage des résultats                                       │   │
│  │  • Thèmes de couleur (dark/light/none)                          │   │
│  │  • Support NO_COLOR                                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Structure des Packages

### `cmd/fibcalc`

Point d'entrée de l'application. Responsabilités :
- Parsing des arguments de ligne de commande
- Initialisation des composants
- Routage vers le mode CLI ou serveur
- Gestion des signaux système

### `internal/fibonacci`

Cœur métier de l'application. Contient :
- **`calculator.go`** : Interface `Calculator` et wrapper générique
- **`fastdoubling.go`** : Algorithme Fast Doubling optimisé
- **`matrix.go`** : Exponentiation matricielle avec Strassen
- **`fft_based.go`** : Calculateur forçant la multiplication FFT
- **`fft.go`** : Logique de sélection de multiplication (standard vs FFT)
- **`constants.go`** : Seuils et constantes de configuration

### `internal/bigfft`

Implémentation de la multiplication FFT pour `big.Int` :
- **`fft.go`** : Algorithme FFT principal
- **`fermat.go`** : Arithmétique modulaire pour FFT
- **`pool.go`** : Pools d'objets pour réduction des allocations

### `internal/orchestration`

Gestion de l'exécution concurrente :
- Exécution parallèle de plusieurs algorithmes
- Agrégation et comparaison des résultats
- Gestion des erreurs et timeouts

### `internal/calibration`

Système de calibration automatique :
- Détection des seuils optimaux pour le matériel
- Persistance des profils de calibration
- Génération adaptative des seuils selon le CPU

### `internal/server`

Serveur HTTP REST :
- Endpoints `/calculate`, `/health`, `/algorithms`, `/metrics`
- Rate limiting et sécurité
- Middleware de logging et métriques
- Graceful shutdown

### `internal/cli`

Interface utilisateur en ligne de commande :
- Spinner animé avec barre de progression
- Estimation du temps restant (ETA)
- Système de thèmes de couleur (dark, light, none)
- Formatage des grands nombres
- **Mode REPL** (`repl.go`) : Session interactive pour calculs multiples
  - Commandes : `calc`, `algo`, `compare`, `list`, `hex`, `status`, `help`, `exit`
  - Changement d'algorithme à la volée
  - Comparaison temps réel des algorithmes
- Génération de scripts d'autocomplétion (bash, zsh, fish, powershell)
- Support de la variable d'environnement `NO_COLOR`

### `internal/config`

Gestion de la configuration :
- Parsing des flags CLI
- Validation des paramètres
- Valeurs par défaut

### `internal/errors`

Gestion centralisée des erreurs :
- Types d'erreurs personnalisés
- Codes de sortie standardisés

## Décisions d'Architecture (ADR)

### ADR-001 : Utilisation de `sync.Pool` pour les états de calcul

**Contexte** : Les calculs de Fibonacci pour de grands N nécessitent de nombreux objets `big.Int` temporaires.

**Décision** : Utiliser `sync.Pool` pour recycler les états de calcul (`calculationState`, `matrixState`).

**Conséquences** :
- ✅ Réduction drastique des allocations mémoire
- ✅ Diminution de la pression sur le GC
- ✅ Amélioration des performances de 20-30%
- ⚠️ Complexité accrue du code

### ADR-002 : Sélection dynamique de l'algorithme de multiplication

**Contexte** : La multiplication FFT est plus efficace que Karatsuba pour les très grands nombres, mais a un overhead significatif pour les petits nombres.

**Décision** : Implémenter une fonction `smartMultiply` qui sélectionne l'algorithme basé sur la taille des opérandes.

**Conséquences** :
- ✅ Performance optimale sur toute la plage de valeurs
- ✅ Configurable via `--fft-threshold`
- ⚠️ Nécessite une calibration pour chaque architecture

### ADR-003 : Architecture hexagonale pour le serveur

**Contexte** : Le serveur doit être testable et extensible.

**Décision** : Utiliser des interfaces et l'injection de dépendances via des options fonctionnelles.

**Conséquences** :
- ✅ Tests unitaires facilités
- ✅ Middleware facilement composable
- ✅ Configuration flexible

### ADR-004 : Parallélisme adaptatif

**Contexte** : Le parallélisme a un coût de synchronisation qui peut dépasser les gains pour de petits calculs.

**Décision** : Activer le parallélisme uniquement au-dessus d'un seuil configurable (`--threshold`).

**Conséquences** :
- ✅ Performance optimale selon la taille du calcul
- ✅ Évite la saturation CPU pour les petits N
- ⚠️ Désactivation du parallélisme quand FFT est utilisé (FFT sature déjà le CPU)

## Flux de Données

### Mode CLI

```
1. main() parse les arguments → config.AppConfig
2. Si --calibrate : calibration.RunCalibration() et exit
3. Si --auto-calibrate : calibration.AutoCalibrate() met à jour config
4. getCalculatorsToRun() sélectionne les algorithmes
5. orchestration.ExecuteCalculations() lance les calculs en parallèle
   - Chaque Calculator.Calculate() s'exécute dans une goroutine
   - Les mises à jour de progression sont envoyées sur un channel
   - cli.DisplayProgress() affiche la progression
6. orchestration.AnalyzeComparisonResults() compare et affiche les résultats
```

### Mode Serveur

```
1. main() détecte --server et appelle server.NewServer()
2. Server.Start() démarre le serveur HTTP avec graceful shutdown
3. Pour chaque requête /calculate :
   a. SecurityMiddleware vérifie les en-têtes
   b. RateLimitMiddleware applique le rate limiting
   c. loggingMiddleware journalise la requête
   d. metricsMiddleware enregistre les métriques
   e. handleCalculate() exécute le calcul
4. Le résultat est retourné en JSON
```

### Mode Interactif (REPL)

```
1. main() détecte --interactive et appelle cli.NewREPL()
2. REPL.Start() affiche la bannière et l'aide
3. Boucle principale :
   a. Affiche le prompt "fib> "
   b. Lit l'entrée utilisateur
   c. Parse et exécute la commande :
      - calc <n> : Calcul avec l'algorithme courant
      - algo <name> : Change l'algorithme actif
      - compare <n> : Compare tous les algorithmes
      - list : Liste les algorithmes
      - hex : Toggle format hexadécimal
      - status : Affiche la configuration
      - exit : Termine la session
4. Répète jusqu'à exit ou EOF
```

## Considérations de Performance

1. **Zero-Allocation** : Les pools d'objets évitent les allocations dans les boucles critiques
2. **Parallélisme intelligent** : Activé uniquement quand bénéfique
3. **FFT adaptatif** : Utilisé pour les très grands nombres uniquement
4. **Strassen** : Activé pour les matrices avec grands éléments
5. **Mise au carré symétrique** : Optimisation spécifique réduisant les multiplications

## Extensibilité

Pour ajouter un nouvel algorithme :

1. Créer une structure implémentant l'interface `coreCalculator` dans `internal/fibonacci`
2. Enregistrer le calculateur dans `calculatorRegistry` dans `main.go`
3. Ajouter les tests correspondants

Pour ajouter un nouveau endpoint API :

1. Ajouter le handler dans `internal/server/server.go`
2. Enregistrer la route dans `NewServer()`
3. Mettre à jour la documentation OpenAPI

