# Documentation de l'API Interne

> **Version**: 1.0.0  
> **Dernière mise à jour**: Décembre 2025

## Vue d'ensemble

Ce document décrit les interfaces et types internes de l'application Fibonacci Calculator. Il fournit une référence complète pour les développeurs souhaitant étendre ou intégrer le système.

L'architecture suit les principes de **Clean Architecture** avec une séparation claire des responsabilités et un faible couplage entre les modules. Les interfaces principales sont conçues pour permettre l'injection de dépendances et faciliter les tests.

## Table des matières

1. [Interfaces principales](#interfaces-principales)
2. [Patterns de design](#patterns-de-design)
3. [Types et structures](#types-et-structures)
4. [Exemples d'utilisation](#exemples-dutilisation)
5. [Relations entre composants](#relations-entre-composants)
6. [Génération de documentation GoDoc](#génération-de-documentation-godoc)

---

## Interfaces principales

### `Calculator`

L'interface principale pour calculer les nombres de Fibonacci. Elle abstrait les différents algorithmes (Fast Doubling, Matrix Exponentiation, FFT-based) et permet leur utilisation interchangeable.

**Localisation**: `internal/fibonacci/calculator.go`

```go
type Calculator interface {
    // Calculate exécute le calcul du n-ième nombre de Fibonacci
    // Conçu pour une exécution concurrente sûre et supporte l'annulation via context
    Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, 
              calcIndex int, n uint64, opts Options) (*big.Int, error)
    
    // Name retourne le nom d'affichage de l'algorithme (ex: "Fast Doubling")
    Name() string
}
```

**Implémentations**:
- `FibCalculator` : Wrapper générique qui encapsule un `coreCalculator`
- Les algorithmes concrets implémentent `coreCalculator` (interface interne)

**Méthodes avancées**:
- `CalculateWithObservers()` : Version avec support du pattern Observer pour le suivi de progression

**Exemple d'utilisation**:
```go
factory := fibonacci.NewDefaultFactory()
calc, _ := factory.Get("fast")
result, err := calc.Calculate(ctx, progressChan, 0, 1000000, opts)
```

---

### `coreCalculator`

Interface interne pour les algorithmes de calcul purs. Les implémentations concrètes incluent :

- `OptimizedFastDoubling` : Algorithme Fast Doubling optimisé (O(log n), parallèle, zero-allocation)
- `MatrixExponentiation` : Exponentiation matricielle avec Strassen (O(log n))
- `FFTBasedCalculator` : Calcul basé sur FFT (O(log n), multiplication FFT)

```go
type coreCalculator interface {
    CalculateCore(ctx context.Context, reporter ProgressReporter, 
                  n uint64, opts Options) (*big.Int, error)
    Name() string
}
```

---

### `ProgressReporter`

Type fonctionnel pour le reporting de progression. Permet aux algorithmes de calcul de rapporter leur progression sans être couplés au mécanisme de communication.

**Localisation**: `internal/fibonacci/progress.go`

```go
type ProgressReporter func(progress float64)
```

**Paramètres**:
- `progress` : Valeur de progression normalisée (0.0 à 1.0)

**Utilisation**:
```go
reporter := func(progress float64) {
    fmt.Printf("Progression: %.2f%%\n", progress*100)
}
```

---

### `ProgressObserver`

Interface pour observer les événements de progression. Implémente le pattern Observer pour permettre un traitement découplé des mises à jour de progression.

**Localisation**: `internal/fibonacci/observer.go`

```go
type ProgressObserver interface {
    // Update est appelé lorsque la progression change
    Update(calcIndex int, progress float64)
}
```

**Implémentations fournies**:
- `ChannelObserver` : Adapte le pattern Observer aux canaux (compatibilité ascendante)
- `LoggingObserver` : Enregistre les mises à jour avec zerolog
- `MetricsObserver` : Exporte les métriques vers Prometheus
- `NoOpObserver` : Pattern Null Object pour les tests

**Exemple**:
```go
subject := fibonacci.NewProgressSubject()
subject.Register(fibonacci.NewChannelObserver(progressChan))
subject.Register(fibonacci.NewLoggingObserver(logger, 0.1))
```

---

### `ProgressSubject`

Gère l'enregistrement et la notification des observateurs de progression. Implémente la partie Subject du pattern Observer.

**Localisation**: `internal/fibonacci/observer.go`

```go
type ProgressSubject struct {
    observers []ProgressObserver
    mu        sync.RWMutex  // Thread-safe
}
```

**Méthodes principales**:
- `Register(observer ProgressObserver)` : Ajoute un observateur
- `Unregister(observer ProgressObserver)` : Retire un observateur
- `Notify(calcIndex int, progress float64)` : Notifie tous les observateurs
- `AsProgressReporter(calcIndex int) ProgressReporter` : Convertit en ProgressReporter pour compatibilité

---

### `CalculatorFactory`

Interface pour créer et gérer des instances de `Calculator`. Permet l'injection de dépendances et facilite les tests.

**Localisation**: `internal/fibonacci/registry.go`

```go
type CalculatorFactory interface {
    // Create crée une nouvelle instance Calculator par nom
    Create(name string) (Calculator, error)
    
    // Get retourne une instance Calculator existante (avec cache)
    Get(name string) (Calculator, error)
    
    // List retourne une liste triée des noms de calculateurs enregistrés
    List() []string
    
    // Register ajoute un nouveau type de calculateur à la factory
    Register(name string, creator func() coreCalculator) error
    
    // GetAll retourne une map de tous les calculateurs enregistrés
    GetAll() map[string]Calculator
}
```

**Implémentation**: `DefaultFactory`

**Calculateurs pré-enregistrés**:
- `"fast"` : OptimizedFastDoubling
- `"matrix"` : MatrixExponentiation
- `"fft"` : FFTBasedCalculator

**Exemple**:
```go
factory := fibonacci.NewDefaultFactory()
calc, err := factory.Get("fast")
if err != nil {
    log.Fatal(err)
}
```

---

### `MultiplicationStrategy`

Interface pour les opérations de multiplication et de mise au carré utilisées dans les calculs de Fibonacci. Permet de choisir entre Karatsuba, FFT ou d'autres algorithmes.

**Localisation**: `internal/fibonacci/strategy.go`

```go
type MultiplicationStrategy interface {
    // Multiply calcule x * y et stocke le résultat dans z (qui peut être réutilisé)
    Multiply(z, x, y *big.Int, opts Options) (*big.Int, error)
    
    // Square calcule x * x (optimisé par rapport à la multiplication générale)
    Square(z, x *big.Int, opts Options) (*big.Int, error)
    
    // Name retourne un nom descriptif pour la stratégie
    Name() string
    
    // ExecuteStep effectue une étape de doublage complète
    ExecuteStep(s *CalculationState, opts Options, inParallel bool) error
}
```

**Implémentations**:
- `AdaptiveStrategy` : Choisit adaptativement entre Karatsuba et FFT selon la taille des opérandes
- `FFTOnlyStrategy` : Force la multiplication FFT pour toutes les opérations
- `KaratsubaStrategy` : Force la multiplication Karatsuba (via math/big)

---

### `Service`

Interface pour les services de calcul de Fibonacci. Abstraction de haut niveau utilisée par la couche serveur HTTP.

**Localisation**: `internal/service/calculator_service.go`

```go
type Service interface {
    // Calculate effectue le calcul de Fibonacci pour l'algorithme et l'index donnés
    Calculate(ctx context.Context, algoName string, n uint64) (*big.Int, error)
}
```

**Implémentation**: `CalculatorService`

**Fonctionnalités**:
- Validation des entrées (limite maxN)
- Récupération de l'algorithme via la factory
- Application centralisée des options de configuration

---

## Patterns de design

### Pattern Observer

Le pattern Observer est utilisé pour le reporting de progression, permettant un découplage entre les calculateurs et les consommateurs de progression.

**Composants**:
- `ProgressSubject` : Sujet observable
- `ProgressObserver` : Interface des observateurs
- `ChannelObserver`, `LoggingObserver`, `MetricsObserver` : Implémentations concrètes

**Flux**:
```
Calculator → ProgressReporter → ProgressSubject → Observers
```

**Avantages**:
- Découplage : Les calculateurs ne connaissent pas leurs observateurs
- Extensibilité : Facile d'ajouter de nouveaux types d'observateurs
- Testabilité : Facile de mocker les observateurs

---

### Pattern Factory

Le pattern Factory est utilisé pour créer et gérer les instances de calculateurs.

**Composants**:
- `CalculatorFactory` : Interface de la factory
- `DefaultFactory` : Implémentation avec cache et thread-safety

**Avantages**:
- Injection de dépendances
- Réutilisation d'instances (cache)
- Enregistrement dynamique de nouveaux calculateurs

---

### Pattern Strategy

Le pattern Strategy est utilisé pour les opérations de multiplication, permettant de choisir dynamiquement l'algorithme (Karatsuba, FFT, etc.).

**Composants**:
- `MultiplicationStrategy` : Interface de la stratégie
- `AdaptiveStrategy`, `FFTOnlyStrategy`, `KaratsubaStrategy` : Implémentations

**Avantages**:
- Flexibilité : Changement d'algorithme à l'exécution
- Testabilité : Facile de tester différents algorithmes
- Performance : Choix optimal selon la taille des opérandes

---

### Pattern Decorator

Le pattern Decorator est utilisé pour ajouter des préoccupations transversales (cross-cutting concerns) aux calculateurs.

**Composants**:
- `FibCalculator` : Décorateur qui enveloppe un `coreCalculator`
- Fonctionnalités ajoutées :
  - Optimisation pour petits n (lookup table)
  - Adaptation du mécanisme de progression
  - Métriques Prometheus
  - Tracing OpenTelemetry

---

## Types et structures

### `Options`

Structure de configuration pour les calculs de Fibonacci.

**Localisation**: `internal/fibonacci/options.go`

```go
type Options struct {
    ParallelThreshold      int  // Seuil (bits) pour paralléliser les multiplications
    FFTThreshold          int  // Seuil (bits) pour utiliser la multiplication FFT
    KaratsubaThreshold    int  // Seuil (bits) pour Karatsuba optimisé
    StrassenThreshold     int  // Seuil (bits) pour l'algorithme de Strassen
    FFTCacheMinBitLen     int  // Longueur minimale (bits) pour mettre en cache les transforms FFT
    FFTCacheMaxEntries    int  // Nombre maximum d'entrées dans le cache FFT
    FFTCacheEnabled       *bool // Active/désactive le cache FFT
    EnableDynamicThresholds bool // Active l'ajustement dynamique des seuils
    DynamicAdjustmentInterval int // Intervalle entre les vérifications de seuil
}
```

**Valeurs par défaut**:
- `ParallelThreshold`: 4096 bits
- `FFTThreshold`: 500000 bits
- `StrassenThreshold`: 3072 bits

---

### `ProgressUpdate`

DTO (Data Transfer Object) qui encapsule l'état de progression d'un calcul.

**Localisation**: `internal/fibonacci/progress.go`

```go
type ProgressUpdate struct {
    CalculatorIndex int     // Identifiant unique du calculateur
    Value           float64 // Progression normalisée (0.0 à 1.0)
}
```

---

### `CalculationState`

Agrège les variables temporaires pour l'algorithme "Fast Doubling", permettant une gestion efficace via un pool d'objets.

**Localisation**: `internal/fibonacci/fastdoubling.go`

```go
type CalculationState struct {
    FK, FK1, T1, T2, T3, T4 *big.Int
}
```

**Méthodes**:
- `Reset()` : Prépare l'état pour un nouveau calcul

**Pool d'objets**:
- `AcquireState()` : Obtient un état du pool
- `ReleaseState(s *CalculationState)` : Libère un état vers le pool

---

### `CalculationResult`

Encapsule le résultat d'un calcul de Fibonacci, facilitant la comparaison et le reporting.

**Localisation**: `internal/orchestration/orchestrator.go`

```go
type CalculationResult struct {
    Name     string        // Identifiant de l'algorithme
    Result   *big.Int      // Nombre de Fibonacci calculé (nil si erreur)
    Duration time.Duration // Temps d'exécution
    Err      error         // Erreur éventuelle
}
```

---

## Exemples d'utilisation

### Exemple 1 : Calcul simple avec un algorithme

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
)

func main() {
    // Créer une factory
    factory := fibonacci.NewDefaultFactory()
    
    // Obtenir un calculateur
    calc, err := factory.Get("fast")
    if err != nil {
        panic(err)
    }
    
    // Configurer les options
    opts := fibonacci.Options{
        ParallelThreshold: 4096,
        FFTThreshold:     500000,
    }
    
    // Calculer F(1000000)
    ctx := context.Background()
    result, err := calc.Calculate(ctx, nil, 0, 1000000, opts)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("F(1000000) = %s\n", result.String())
}
```

---

### Exemple 2 : Calcul avec suivi de progression

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
)

func main() {
    factory := fibonacci.NewDefaultFactory()
    calc, _ := factory.Get("fast")
    
    // Créer un sujet de progression
    subject := fibonacci.NewProgressSubject()
    
    // Enregistrer un observateur de canal
    progressChan := make(chan fibonacci.ProgressUpdate, 10)
    subject.Register(fibonacci.NewChannelObserver(progressChan))
    
    // Lire les mises à jour de progression
    go func() {
        for update := range progressChan {
            fmt.Printf("Calculateur %d: %.2f%%\n", 
                       update.CalculatorIndex, update.Value*100)
        }
    }()
    
    // Calculer avec observateurs
    ctx := context.Background()
    opts := fibonacci.Options{}
    result, err := calc.CalculateWithObservers(ctx, subject, 0, 1000000, opts)
    
    close(progressChan)
    
    if err != nil {
        panic(err)
    }
    fmt.Printf("Résultat: %s\n", result.String())
}
```

---

### Exemple 3 : Comparaison de plusieurs algorithmes

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
    "github.com/agbru/fibcalc/internal/orchestration"
    "github.com/agbru/fibcalc/internal/config"
    "os"
)

func main() {
    factory := fibonacci.NewDefaultFactory()
    cfg := config.AppConfig{
        N: 1000000,
    }
    
    // Obtenir tous les calculateurs
    calculators := []fibonacci.Calculator{}
    for _, name := range factory.List() {
        calc, _ := factory.Get(name)
        calculators = append(calculators, calc)
    }
    
    // Exécuter les calculs en parallèle
    ctx := context.Background()
    results := orchestration.ExecuteCalculations(ctx, calculators, cfg, os.Stdout)
    
    // Analyser les résultats
    exitCode := orchestration.AnalyzeComparisonResults(results, cfg, os.Stdout)
    os.Exit(exitCode)
}
```

---

### Exemple 4 : Enregistrement d'un calculateur personnalisé

```go
package main

import (
    "context"
    "fmt"
    "math/big"
    "github.com/agbru/fibcalc/internal/fibonacci"
)

// Implémentation personnalisée
type CustomCalculator struct{}

func (c *CustomCalculator) Name() string {
    return "Custom Algorithm"
}

func (c *CustomCalculator) CalculateCore(ctx context.Context, 
                                        reporter fibonacci.ProgressReporter,
                                        n uint64, 
                                        opts fibonacci.Options) (*big.Int, error) {
    // Implémentation personnalisée
    reporter(0.5) // 50% de progression
    result := big.NewInt(int64(n)) // Exemple simplifié
    reporter(1.0) // 100% de progression
    return result, nil
}

func main() {
    factory := fibonacci.NewDefaultFactory()
    
    // Enregistrer le calculateur personnalisé
    factory.Register("custom", func() fibonacci.coreCalculator {
        return &CustomCalculator{}
    })
    
    // Utiliser le calculateur personnalisé
    calc, _ := factory.Get("custom")
    result, _ := calc.Calculate(context.Background(), nil, 0, 100, fibonacci.Options{})
    fmt.Println(result)
}
```

---

### Exemple 5 : Utilisation du service

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
    "github.com/agbru/fibcalc/internal/service"
    "github.com/agbru/fibcalc/internal/config"
)

func main() {
    factory := fibonacci.NewDefaultFactory()
    cfg := config.AppConfig{}
    
    // Créer le service
    svc := service.NewCalculatorService(factory, cfg, 0) // 0 = pas de limite
    
    // Utiliser le service
    ctx := context.Background()
    result, err := svc.Calculate(ctx, "fast", 1000000)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Résultat: %s\n", result.String())
}
```

---

## Relations entre composants

### Diagramme de dépendances

```
┌─────────────────────────────────────────────────────────────┐
│                    Entry Points                             │
│  (CLI, Server, REPL)                                        │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Orchestration Layer                            │
│  • ExecuteCalculations()                                    │
│  • AnalyzeComparisonResults()                               │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                  Service Layer                              │
│  • CalculatorService                                        │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Fibonacci Package                              │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │  Calculator  │───▶│   Factory    │───▶│  Algorithms │ │
│  │  Interface   │    │              │    │             │ │
│  └──────────────┘    └──────────────┘    └─────────────┘ │
│         │                    │                   │         │
│         │                    │                   │         │
│         ▼                    ▼                   ▼         │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │   Progress   │    │   Strategy    │    │   Options   │ │
│  │   Observer   │    │   Pattern    │    │             │ │
│  └──────────────┘    └──────────────┘    └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Flux de données

1. **Initialisation**:
   ```
   Factory → Register Calculators → Cache Instances
   ```

2. **Calcul**:
   ```
   Calculator.Calculate() 
   → ProgressReporter 
   → ProgressSubject 
   → Observers (Channel, Logging, Metrics)
   ```

3. **Orchestration**:
   ```
   ExecuteCalculations() 
   → Multiple Calculators (concurrent)
   → Collect Results
   → AnalyzeComparisonResults()
   ```

---

## Génération de documentation GoDoc

### Accès via pkg.go.dev

La documentation GoDoc est automatiquement générée et hébergée sur [pkg.go.dev](https://pkg.go.dev) pour les packages publics. Pour les packages internes, vous pouvez générer la documentation localement.

### Génération locale

```bash
# Générer la documentation HTML pour tous les packages
godoc -http=:6060

# Accéder à la documentation via navigateur
# http://localhost:6060/pkg/github.com/agbru/fibcalc/internal/fibonacci/
```

### Structure de la documentation

Les commentaires GoDoc suivent les conventions standard :

- **Package comment** : Description du package (première ligne du fichier)
- **Type comments** : Description des types et interfaces
- **Method comments** : Description des méthodes avec paramètres et retours
- **Example functions** : Exemples d'utilisation (fonctions `Example*`)

### Exemple de commentaire GoDoc

```go
// Calculator définit l'interface publique pour un calculateur de Fibonacci.
// C'est l'abstraction principale utilisée par la couche d'orchestration
// pour interagir avec les différents algorithmes de calcul.
type Calculator interface {
    // Calculate exécute le calcul du n-ième nombre de Fibonacci.
    // Conçu pour une exécution concurrente sûre et supporte l'annulation
    // via le context fourni.
    //
    // Paramètres:
    //   - ctx: Le context pour gérer l'annulation et les délais.
    //   - progressChan: Le canal pour envoyer les mises à jour de progression.
    //   - calcIndex: Un index unique pour l'instance du calculateur.
    //   - n: L'index du nombre de Fibonacci à calculer.
    //   - opts: Options de configuration pour le calcul.
    //
    // Retourne:
    //   - *big.Int: Le nombre de Fibonacci calculé.
    //   - error: Une erreur si une erreur s'est produite.
    Calculate(ctx context.Context, progressChan chan<- ProgressUpdate,
              calcIndex int, n uint64, opts Options) (*big.Int, error)
}
```

---

## Bonnes pratiques

### 1. Utilisation des interfaces

- Préférez les interfaces aux types concrets pour les paramètres de fonction
- Utilisez l'injection de dépendances via les factories
- Évitez de créer des dépendances circulaires

### 2. Gestion de la progression

- Utilisez `ProgressSubject` pour enregistrer plusieurs observateurs
- Utilisez `CalculateWithObservers()` pour un contrôle fin
- Utilisez `Calculate()` pour la compatibilité avec les canaux

### 3. Gestion des erreurs

- Vérifiez toujours les erreurs retournées
- Utilisez `context.Context` pour l'annulation
- Respectez les timeouts configurés

### 4. Performance

- Réutilisez les instances de calculateurs via `Factory.Get()`
- Configurez les seuils selon votre cas d'usage
- Utilisez le pool d'objets pour les calculs répétés

### 5. Tests

- Utilisez les mocks générés (`mockgen`) pour les tests
- Testez les interfaces, pas les implémentations
- Utilisez `NoOpObserver` pour les tests sans progression

---

## Ressources supplémentaires

- [Architecture générale](./ARCHITECTURE.md)
- [Documentation de l'API REST](./api/API.md)
- [Guide de performance](./PERFORMANCE.md)
- [Documentation des algorithmes](./algorithms/)

---

## Contribution

Pour contribuer à cette documentation :

1. Mettez à jour ce fichier avec les nouvelles interfaces/types
2. Ajoutez des exemples d'utilisation pour les nouvelles fonctionnalités
3. Maintenez la cohérence avec les commentaires GoDoc existants
4. Testez que les exemples fonctionnent correctement

---

**Note**: Cette documentation est maintenue manuellement. Pour la documentation générée automatiquement à partir des commentaires GoDoc, consultez la documentation générée avec `godoc` ou sur pkg.go.dev.
