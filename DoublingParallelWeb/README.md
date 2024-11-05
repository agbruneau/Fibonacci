# README

![Diagramme de Séquence](SequenceDiagram.jpeg)

## Introduction

Ce programme en Go calcule la **somme des n premiers nombres de Fibonacci** de manière **parallélisée**. Il utilise des techniques avancées de **concurrence** en Go et gère les **grands nombres** entiers grâce au package `math/big`.

L'objectif de ce document est d'expliquer en détail le fonctionnement du code, en décrivant chaque composant et en vulgarisant les concepts utilisés.

## Table des matières

1. [Structure générale du programme](#structure-générale-du-programme)
2. [Importations](#importations)
3. [Configuration](#configuration)
4. [Métriques de performance](#métriques-de-performance)
5. [Calcul des nombres de Fibonacci](#calcul-des-nombres-de-fibonacci)
6. [Gestion des workers](#gestion-des-workers)
7. [Calcul des segments](#calcul-des-segments)
8. [Formatage des grands nombres](#formatage-des-grands-nombres)
9. [Fonction principale `main`](#fonction-principale-main)
10. [Concepts clés](#concepts-clés)
11. [Instructions pour l'exécution](#instructions-pour-lexécution)
12. [Conclusion](#conclusion)
13. [Références](#références)

## Structure générale du programme

Le programme est structuré comme suit :

- **Importations** : Inclusion des packages nécessaires.
- **Types personnalisés** : Définition des structures pour la configuration, les métriques et le calcul des nombres de Fibonacci.
- **Fonctions** : Implémentation des fonctions pour le calcul, la gestion des workers et le formatage.
- **Fonction `main`** : Orchestration du processus global.

## Importations

Le programme utilise les packages suivants :

- **Packages standards** :
  - `context` : Gestion des contextes pour les annulations et les timeouts.
  - `fmt` : Formatage des entrées/sorties.
  - `log` : Journalisation des erreurs.
  - `math/big` : Manipulation des grands nombres entiers.
  - `runtime` : Informations sur l'environnement d'exécution (par exemple, le nombre de cœurs CPU disponibles).
  - `strings` : Manipulation des chaînes de caractères.
  - `sync` : Synchronisation des goroutines.
  - `time` : Gestion du temps et des délais.
- **Package tiers** :
  - `github.com/pkg/errors` : Enrichissement des erreurs avec des messages supplémentaires.

## Configuration

### Type `Configuration`

La structure `Configuration` centralise tous les paramètres configurables du programme :

- `M int` : Limite supérieure (exclue) du calcul des nombres de Fibonacci.
- `NumWorkers int` : Nombre de workers parallèles.
- `SegmentSize int` : Taille des segments de calcul pour chaque worker.
- `Timeout time.Duration` : Durée maximale autorisée pour le calcul complet.

### Fonction `DefaultConfig`

Cette fonction retourne une configuration par défaut avec des valeurs raisonnables :

- `M` : 100000 (calcul jusqu'à F(99 999)).
- `NumWorkers` : Nombre de cœurs CPU disponibles.
- `SegmentSize` : 1000 (chaque worker traite 1000 nombres à la fois).
- `Timeout` : 5 minutes.

```go
func DefaultConfig() Configuration {
    return Configuration{
        M:           100000,
        NumWorkers:  runtime.NumCPU(),
        SegmentSize: 1000,
        Timeout:     5 * time.Minute,
    }
}
```

## Métriques de performance

### Type `Metrics`

La structure `Metrics` garde trace des performances pendant l'exécution :

- `StartTime time.Time` : Heure de début du calcul.
- `EndTime time.Time` : Heure de fin du calcul.
- `TotalCalculations int64` : Nombre total de calculs effectués.
- `mutex sync.Mutex` : Mutex pour protéger les modifications concurrentes.

### Fonctions associées

- `NewMetrics()` : Crée une nouvelle instance de `Metrics` avec l'heure actuelle.
- `IncrementCalculations(count int64)` : Incrémente le compteur de calculs de manière thread-safe.

```go
func NewMetrics() *Metrics {
    return &Metrics{StartTime: time.Now()}
}

func (m *Metrics) IncrementCalculations(count int64) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.TotalCalculations += count
}
```

## Calcul des nombres de Fibonacci

### Type `FibCalculator`

La structure `FibCalculator` encapsule la logique de calcul des nombres de Fibonacci en réutilisant des variables `big.Int` pour éviter les allocations mémoire répétées :

- `fk, fk1 *big.Int` : Stockent F(k) et F(k+1).
- `temp1, temp2, temp3 *big.Int` : Variables temporaires pour les calculs.
- `mutex sync.Mutex` : Protection pour l'accès concurrent.

### Fonction `NewFibCalculator`

Crée une nouvelle instance de `FibCalculator` avec les variables initialisées.

```go
func NewFibCalculator() *FibCalculator {
    return &FibCalculator{
        fk:    new(big.Int),
        fk1:   new(big.Int),
        temp1: new(big.Int),
        temp2: new(big.Int),
        temp3: new(big.Int),
    }
}
```

### Méthode `Calculate`

Calcule le n-ième nombre de Fibonacci en utilisant l'**algorithme de doublement**, qui a une complexité de O(log n).

#### Étapes de la méthode `Calculate`

1. **Validation des entrées** :
   - Vérifie que `n` est non négatif.
   - Vérifie que `n` n'est pas trop grand pour éviter des calculs coûteux.

2. **Cas de base** :
   - Si `n <= 1`, retourne `n`.

3. **Initialisation** :
   - Initialise `F(0)` et `F(1)`.

4. **Boucle principale** :
   - Parcourt les bits de `n` de haut en bas.
   - Utilise les formules de doublement :
     - `F(2k) = F(k)[2F(k+1) - F(k)]`
     - `F(2k+1) = F(k+1)^2 + F(k)^2`
   - Si le bit est à 1, effectue un pas supplémentaire.

5. **Retourne le résultat** :
   - Retourne une copie de `fk`.

```go
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
    // Validation et initialisation omises pour la concision

    for i := 63; i >= 0; i-- {
        // Calcul de F(2k) et F(2k+1)

        if (n & (1 << uint(i))) != 0 {
            // Pas supplémentaire si le bit est à 1
        }
    }

    return new(big.Int).Set(fc.fk), nil
}
```

## Gestion des workers

### Type `WorkerPool`

Le `WorkerPool` gère un pool de calculateurs réutilisables :

- `calculators []*FibCalculator` : Tableau des calculateurs disponibles.
- `current int` : Index du prochain calculateur à utiliser.
- `mutex sync.Mutex` : Protection pour l'accès concurrent.

### Fonction `NewWorkerPool`

Crée un nouveau pool avec le nombre spécifié de calculateurs.

```go
func NewWorkerPool(size int) *WorkerPool {
    calculators := make([]*FibCalculator, size)
    for i := range calculators {
        calculators[i] = NewFibCalculator()
    }
    return &WorkerPool{
        calculators: calculators,
    }
}
```

### Méthode `GetCalculator`

Retourne le prochain calculateur disponible de manière circulaire.

```go
func (wp *WorkerPool) GetCalculator() *FibCalculator {
    wp.mutex.Lock()
    defer wp.mutex.Unlock()
    calc := wp.calculators[wp.current]
    wp.current = (wp.current + 1) % len(wp.calculators)
    return calc
}
```

## Calcul des segments

### Type `Result`

Structure pour encapsuler le résultat d'un calcul avec une potentielle erreur :

- `Value *big.Int` : Résultat du calcul.
- `Error error` : Erreur éventuelle.

### Fonction `computeSegment`

Calcule la somme des nombres de Fibonacci pour un segment donné.

#### Étapes de `computeSegment`

1. **Récupération d'un calculateur** depuis le `WorkerPool`.
2. **Initialisation** de la somme partielle.
3. **Boucle de calcul** :
   - Pour chaque `i` dans le segment :
     - Vérifie si le contexte est annulé (timeout).
     - Calcule `F(i)` et l'ajoute à la somme partielle.
4. **Mise à jour des métriques**.
5. **Retourne le résultat**.

```go
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics) Result {
    // Code de la fonction
}
```

## Formatage des grands nombres

### Fonction `formatBigIntSci`

Formate un grand nombre en notation scientifique pour un affichage plus lisible.

```go
func formatBigIntSci(n *big.Int) string {
    // Code de la fonction
}
```

**Exemple** : `123456789` devient `"1.2345e8"`.

## Fonction principale `main`

La fonction `main` orchestre tout le processus de calcul.

### Étapes de `main`

1. **Initialisation** :
   - Charge la configuration par défaut.
   - Initialise les métriques.

2. **Création du contexte** avec timeout.

3. **Initialisation du `WorkerPool`** et des canaux.

4. **Distribution du travail** :
   - Divise le calcul en segments.
   - Lance des goroutines pour chaque segment.

5. **Collecte des résultats** :
   - Agrège les sommes partielles.
   - Gère les erreurs éventuelles.

6. **Calcul des métriques finales**.

7. **Affichage des résultats** :
   - Configuration utilisée.
   - Performances.
   - Résultat final.

```go
func main() {
    // Code de la fonction
}
```

## Concepts clés

### Concurrence et parallélisme

- **Goroutines** : Légères unités d'exécution concurrentes.
- **WaitGroup** : Synchronisation des goroutines pour attendre la fin des tâches.
- **Mutex** : Protection des ressources partagées contre les accès concurrents.

### Gestion des grands nombres

- **Package `math/big`** : Permet de manipuler des entiers de taille arbitraire.
- **`big.Int`** : Type pour les entiers grands.
- **Opérations arithmétiques** : Méthodes associées pour les opérations (+, -, *, etc.).

### Algorithme de doublement pour Fibonacci

- **Complexité** : O(log n).
- **Formules utilisées** :
  - `F(2k) = F(k) * [2 * F(k+1) - F(k)]`
  - `F(2k+1) = F(k+1)^2 + F(k)^2`
- **Avantages** : Beaucoup plus efficace que l'approche récursive ou itérative classique.

### Gestion des timeouts avec `context`

- **Contexte avec timeout** : Permet d'annuler les opérations si elles prennent trop de temps.
- **Propagation de l'annulation** : Les goroutines vérifient régulièrement si le contexte est annulé.

## Instructions pour l'exécution

### Prérequis

- **Go** : Assurez-vous que Go est installé sur votre système (version 1.13 ou supérieure recommandée).
- **Packages tiers** : Installez le package `github.com/pkg/errors` en exécutant :

```bash
go get github.com/pkg/errors
```

### Compilation

Compilez le programme avec la commande :

```bash
go build -o fibonacci_sum
```

### Exécution

Exécutez le programme compilé :

```bash
./fibonacci_sum
```

### Personnalisation

Pour modifier les paramètres du programme, ajustez les valeurs dans la fonction `DefaultConfig` :

- **Limite supérieure `M`** : Changez la valeur pour calculer jusqu'à un autre nombre de Fibonacci.
- **Nombre de workers `NumWorkers`** : Ajustez en fonction du nombre de cœurs CPU souhaités.
- **Taille des segments `SegmentSize`** : Modifiez pour contrôler la charge de travail de chaque goroutine.
- **Timeout `Timeout`** : Changez la durée maximale autorisée pour le calcul.

## Conclusion

Ce programme démontre comment utiliser efficacement la **concurrence en Go** pour effectuer des calculs intensifs. En combinant l'algorithme de doublement pour le calcul des nombres de Fibonacci et la gestion des grands nombres avec `math/big`, il est possible de calculer rapidement la somme des n premiers nombres de Fibonacci, même pour de grandes valeurs de n.

Les techniques utilisées, telles que les goroutines, les mutex et les contextes avec timeout, sont essentielles pour écrire des programmes Go performants et robustes.

## Références

- [Documentation officielle de Go](https://golang.org/doc/)
- [Package `math/big`](https://pkg.go.dev/math/big)
- [Concurrence en Go](https://tour.golang.org/concurrency/1)
- [Algorithme de doublement pour Fibonacci](https://www.nayuki.io/page/fast-fibonacci-algorithms)
- [Gestion des contextes en Go](https://blog.golang.org/context)