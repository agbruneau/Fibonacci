# Calculateur Haute Performance pour la Suite de Fibonacci

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage Status](https://img.shields.io/badge/coverage-69.7%25-brightgreen)

## 1. Objectif

Ce projet est un calculateur de Fibonacci haute performance et une étude de cas en ingénierie logicielle avancée avec Go. Il est conçu pour explorer et implémenter des algorithmes efficaces pour la manipulation de très grands entiers, en appliquant des optimisations de bas niveau et des patrons de conception de haut niveau pour maximiser la performance.

Les objectifs principaux sont :
- Servir de référence pour l'implémentation d'algorithmes sophistiqués en Go.
- Démontrer les meilleures pratiques en architecture logicielle, y compris la modularité et la testabilité.
- Fournir un exemple pratique de techniques d'optimisation de la performance.

## 2. Démarrage

Suivez ces étapes pour mettre en service le calculateur de Fibonacci sur votre machine locale.

### Prérequis

- Go 1.25 ou une version ultérieure

### Installation

1. Clonez le dépôt :
   ```bash
   git clone https://github.com/votre-nom-utilisateur/fibcalc.git
   cd fibcalc
   ```

2. Compilez l'exécutable :
   ```bash
   go build -o fibcalc ./cmd/fibcalc
   ```
   Cela créera un binaire `fibcalc` (ou `fibcalc.exe` sur Windows) à la racine du projet.

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

## 3. Fonctionnalités

*   **Support des Grands Nombres** : Utilise `math/big` pour une arithmétique de précision arbitraire.
*   **Algorithmes Multiples** : Implémente plusieurs algorithmes en O(log n) :
    *   **Fast Doubling** (`fast`)
    *   **Exponentiation Matricielle** (`matrix`)
    *   **Fast Doubling Basé sur la FFT** (`fft`)
*   **Optimisations de Performance** :
    *   **Stratégie Zéro-Allocation** : Emploie `sync.Pool` pour minimiser la surcharge du ramasse-miettes.
    *   **Parallélisme** : Tire parti des cœurs multiples pour des performances améliorées.
    *   **Multiplication FFT Adaptative** : Bascule vers la multiplication basée sur la FFT pour les très grands nombres.
*   **Architecture Modulaire** :
    *   **Séparation des Préoccupations** : Découple la logique, la présentation et l'orchestration.
    *   **Arrêt Propre** : Gère le cycle de vie de l'application avec `context`.
    *   **Concurrence Structurée** : Utilise `golang.org/x/sync/errgroup` pour l'orchestration.
*   **CLI Conviviale** :
    *   Spinner et barre de progression pour un retour visuel.
    *   Modes pour la comparaison, la calibration et les résultats détaillés.
    *   Configuration et validation robustes.

## 4. Utilisation

Le calculateur est contrôlé via des drapeaux de ligne de commande :

```bash
./fibcalc [options]
```

### Options

| Drapeau              | Alias       | Description                                                 | Défaut       |
| -------------------- | ----------- | ----------------------------------------------------------- | ------------ |
| `-n`                 |             | Index du nombre de Fibonacci à calculer.                    | `250000000`  |
| `-algo`              |             | Algorithme à utiliser : `fast`, `matrix`, `fft`, ou `all`.  | `all`        |
| `-timeout`           |             | Temps d'exécution maximum (ex: `10s`, `1m30s`).             | `5m`         |
| `-threshold`         |             | Seuil en bits pour paralléliser les multiplications.        | `4096`       |
| `-fft-threshold`     |             | Seuil en bits pour activer la multiplication FFT.           | `20000`      |
| `--strassen-threshold` |           | Seuil en bits pour basculer vers l'algorithme de Strassen.  | `256`        |
| `-d`                 | `--details` | Afficher les détails de performance et les métadonnées.     | `false`      |
| `-v`                 |             | Afficher le résultat complet (peut être très long).         | `false`      |
| `--calibrate`        |             | Calibrer le seuil de parallélisme optimal.                  | `false`      |
| `--auto-calibrate`   |             | Lancer une calibration rapide au démarrage.                 | `false`      |
| `--lang`             |             | Langue pour l'i18n (ex: `fr`, `en`).                        | `en`         |
| `--i18n-dir`         |             | Répertoire des fichiers de traduction (ex: `./locales`).    | `""`         |

### Exemples

- **Calibrez pour votre machine :**
  ```bash
  ./fibcalc --calibrate
  ```

- **Comparez les algorithmes pour F(10,000,000) :**
  ```bash
  ./fibcalc -n 10000000 -algo all -d
  ```

- **Calculez F(250,000,000) avec un timeout de 10 minutes :**
  ```bash
  ./fibcalc -n 250000000 -algo fast -d --timeout 10m
  ```

## 5. Architecture Logicielle

Ce projet est structuré selon les meilleures pratiques de l'ingénierie logicielle Go, en mettant l'accent sur la **modularité** et la **séparation des préoccupations**. Cette approche garantit que la base de code est non seulement performante, mais aussi maintenable, testable et facile à faire évoluer.

L'architecture est organisée comme suit :

-   **`cmd/fibcalc`**: C'est le **point d'entrée** de l'application. Son rôle unique est d'analyser les arguments de la ligne de commande, de construire les dépendances nécessaires (comme les calculateurs et la configuration) et d'orchestrer le déroulement de l'exécution. Il agit comme la "racine de composition" (*composition root*) de l'application.

-   **`internal/config`**: Ce paquet est exclusivement dédié à la **gestion de la configuration**. Il définit la structure `AppConfig`, gère l'analyse des drapeaux de la ligne de commande et assure la validation des entrées utilisateur. En isolant la configuration, nous pouvons facilement modifier ou étendre les options de l'application sans impacter la logique métier.

-   **`internal/fibonacci`**: Le **cœur de la logique métier** réside ici. Ce paquet contient les implémentations des algorithmes de calcul de Fibonacci. Il expose une interface `Calculator` claire, ce qui permet au reste de l'application d'interagir avec les algorithmes de manière agnostique. C'est également là que se trouvent les optimisations de bas niveau (gestion de la mémoire, parallélisme).

-   **`internal/cli`**: La **couche de présentation** est entièrement gérée par ce paquet. Il est responsable de l'affichage du spinner, de la barre de progression et des résultats finaux. En séparant l'interface utilisateur de la logique de calcul, nous pourrions, par exemple, exposer la même logique via une API web sans modifier le paquet `fibonacci`.

-   **`internal/i18n`**: Ce paquet gère l'**internationalisation (i18n)** de l'application. Il permet de charger et d'utiliser des traductions pour les messages affichés à l'utilisateur, rendant l'application accessible à un public plus large.

Cette conception en couches assure un **faible couplage** entre les différents composants du système, ce qui est la clé d'un logiciel robuste et de haute qualité.

## 6. Patrons de Conception (Design Patterns)

Pour améliorer la modularité et la flexibilité, le projet met en œuvre le **patron de conception Décorateur (Decorator)**. Ce patron permet d'ajouter dynamiquement de nouvelles fonctionnalités à des objets sans en modifier la structure.

Dans le paquet `internal/fibonacci`, nous définissons deux interfaces :

1.  **`coreCalculator`**: C'est une interface interne simple qui définit le contrat de base pour tout algorithme de Fibonacci. Elle ne se préoccupe que de la logique de calcul pure.

    ```go
    type coreCalculator interface {
        CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, ...) (*big.Int, error)
        Name() string
    }
    ```

2.  **`Calculator`**: C'est l'interface publique utilisée par le reste de l'application.

Le type `FibCalculator` agit comme un **décorateur**. Il encapsule un `coreCalculator` et y ajoute des fonctionnalités transversales :

-   **Optimisation par Cache (Lookup Table)** : Pour les petites valeurs de `n`, le décorateur retourne un résultat depuis une table précalculée, court-circuitant l'algorithme de base.
-   **Gestion de la Progression** : Il adapte le canal de progression en une simple fonction de rappel (`ProgressReporter`), simplifiant ainsi l'implémentation des algorithmes de base.

```go
type FibCalculator struct {
    core coreCalculator
}

func (c *FibCalculator) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, ...) (*big.Int, error) {
    // 1. Vérifier la table de cache
    if n <= MaxFibUint64 {
        return lookupSmall(n), nil
    }
    // 2. Adapter le rapport de progression
    reporter := func(...) { /* ... */ }
    // 3. Déléguer à l'algorithme de base
    return c.core.CalculateCore(ctx, reporter, n, ...)
}
```

Cette approche permet de conserver des algorithmes de base simples et ciblés, tout en les enrichissant de manière flexible avec des fonctionnalités communes.

## 7. Optimisations de Performance

La performance est un objectif central de ce projet. Plusieurs techniques avancées sont utilisées pour garantir des temps de calcul minimaux, en particulier pour les très grands nombres.

### Stratégie Zéro-Allocation

Pour minimiser la pression sur le ramasse-miettes (Garbage Collector), qui peut être une source de latence, nous utilisons une **stratégie zéro-allocation** dans les boucles de calcul critiques.

-   **Pools d'Objets (`sync.Pool`)**: Les états de calcul complexes (comme `calculationState` et `matrixState`), qui contiennent de multiples `big.Int`, sont recyclés à l'aide de `sync.Pool`. Au lieu de créer de nouveaux objets pour chaque calcul, nous en réutilisons des anciens, ce qui élimine presque totalement les allocations de mémoire dans les parties les plus intensives du code.

### Parallélisme Multi-cœur

Les algorithmes comme le *Fast Doubling* impliquent plusieurs multiplications de grands nombres à chaque étape. Ces opérations sont indépendantes et peuvent donc être exécutées en parallèle.

-   **Goroutines et `sync.WaitGroup`**: Lorsque la taille des opérandes dépasse un certain seuil (`-threshold`), nous lançons les multiplications sur des **goroutines** distinctes. Un `sync.WaitGroup` est utilisé pour synchroniser leur achèvement avant de poursuivre. Cette approche permet de distribuer la charge de calcul sur tous les cœurs de processeur disponibles, réduisant ainsi le temps de calcul de manière significative.

### Multiplication Adaptative

La multiplication de très grands entiers est l'opération la plus coûteuse. L'algorithme de multiplication standard a une complexité d'environ O(n^1.585). Cependant, pour les nombres extrêmement grands, les algorithmes basés sur la **Transformation de Fourier Rapide (FFT)** sont asymptotiquement plus rapides (proche de O(n log n)).

-   **Seuil Dynamique (`-fft-threshold`)**: Le calculateur surveille la taille (en bits) des nombres à multiplier. S'ils dépassent le `-fft-threshold`, il bascule dynamiquement d'une multiplication standard `big.Int.Mul` à une implémentation basée sur la FFT, garantissant que l'algorithme le plus efficace est toujours utilisé pour la taille des données concernées.

## 8. Tests

Le projet inclut une suite de tests complète pour garantir la correction et la stabilité.

- **Lancer tous les tests :**
  ```bash
  go test ./... -v
  ```

- **Lancer les benchmarks :**
  ```bash
  go test -bench . ./...
  ```

## 7. Licence

Ce projet est sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.
