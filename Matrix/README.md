# Calculateur de Nombres de Fibonacci avec Méthode Matricielle et Benchmark Concurrentiel

![Diagramme de Séquence](SequenceDiagram.jpeg)

## Description du Projet

Ce projet est un programme écrit en **Golang** permettant de calculer les nombres de Fibonacci à l'aide de la **méthode de la matrice de puissance**, une technique optimisée qui permet de réduire la complexité temporelle à `O(log n)`. Le programme utilise la **mémoïsation** grâce à un **cache LRU** (Least Recently Used) pour accélérer les calculs en évitant les recalculs inutiles et en tirant parti des valeurs précédemment calculées.

Le projet comprend aussi une fonctionnalité de **benchmark concurrentiel** qui évalue les performances du calcul pour différentes valeurs de `n`, en utilisant un **pool de workers** exécutant les calculs en parallèle. Cette approche permet d'optimiser l'utilisation des systèmes multi-cœurs modernes.

## Fonctionnalités Principales

- **Calcul des nombres de Fibonacci par la méthode de la matrice de puissance** : Cette approche utilise l'exponentiation rapide pour réduire le nombre d'opérations nécessaires au calcul des nombres de Fibonacci.

- **Mémoïsation avec Cache LRU** : L'utilisation d'un **cache LRU thread-safe**, basé sur la bibliothèque "github.com/hashicorp/golang-lru", permet de réutiliser les valeurs précédemment calculées et d'améliorer les performances, en particulier lors de calculs répétés.

- **Concurrence et Pool de Workers** : Un **pool de workers** est implémenté pour effectuer les calculs de manière parallèle, étant ainsi capable de tirer profit des systèmes à multiples cœurs. Chaque worker peut traiter des valeurs distinctes, réduisant ainsi le temps global de calcul.

- **Gestion du Contexte et Benchmarking** : Le programme utilise des contextes pour limiter le temps d'exécution des benchmarks et éviter des exécutions prolongées. Les benchmarks évaluent les performances de calcul pour des valeurs croissantes de `n`, permettant ainsi d'analyser la scalabilité de l'algorithme.

- **Gestion des Entiers de Grande Taille** : Le calcul des nombres de Fibonacci devient rapidement très exigeant en termes de taille des nombres. Ainsi, la bibliothèque `math/big` est utilisée pour manipuler des entiers de très grande taille sans dépasser les limites des types d'entiers natifs.

## Bibliothèques Utilisées

- **"math/big"** : Utilisée pour gérer des entiers de grande taille.
- **"github.com/hashicorp/golang-lru"** : Utilisée pour implémenter un **cache LRU** permettant une mémoïsation efficace des calculs précédents.
- **"sync" et "context"** : Utilisées pour la gestion de la concurrence et la synchronisation des goroutines, ainsi que pour la gestion des contextes lors des exécutions parallèles.

## Structure du Code

1. **Initialisation du Cache** : Le cache LRU est initialisé à l'aide de la fonction `init()`, avec un maximum de `1000` entrées pour préserver la mémoire et améliorer la performance.

2. **Fonction `fibMatrixPower(n int)`** : Implémente le calcul des nombres de Fibonacci en utilisant l'exponentiation matricielle. Avant de procéder au calcul, le cache est vérifié afin de récupérer une valeur préalablement calculée si elle est disponible.

3. **Fonction `matrixPower(matrix [2][2]*big.Int, n int)`** : Calcule la puissance d'une matrice par l'exponentiation rapide, avec une complexité de `O(log n)`.

4. **Fonctions `getFromCache()` et `addToCache()`** : Assurent la gestion thread-safe du cache, en utilisant des verrous pour empêcher les conflits lors des accès concurrents.

5. **Benchmarking avec `benchmarkFibWithWorkerPool()`** : Cette fonction crée un pool de workers et assigne les travaux de calcul de Fibonacci à chaque worker. Un contexte est utilisé pour limiter la durée des calculs afin d'éviter des exécutions trop longues.

## Utilisation

Pour exécuter le programme, il suffit de compiler et exécuter le fichier principal `main.go`. Assurez-vous d'avoir installé toutes les dépendances nécessaires, en particulier le cache LRU qui peut être installé via Go Modules.

Commandes à exécuter :

```bash
# Installation des dépendances
$ go mod tidy

# Compilation et exécution
$ go run main.go
```

Le programme effectuera les calculs pour des valeurs croissantes de `n` et affichera le temps moyen de calcul obtenu pour chaque valeur, grâce à un pool de `16` workers.

## Améliorations Possibles

1. **Ajustement Dynamique de la Taille du Cache** : Permettre une taille adaptative du cache en fonction de la mémoire disponible pourrait améliorer les performances pour des valeurs très élevées de `n`.

2. **Instrumentation** : Intégrer des outils de **profiling** pour surveiller la performance du programme et identifier les goulots d'étranglement.

3. **Gestion des Échecs** : Ajouter des mécanismes de gestion des échecs pour permettre une reprise automatique ou un ajustement adaptatif des paramètres afin d'éviter les erreurs liées aux ressources.

## Conclusion

Ce projet met en œuvre des techniques avancées de calcul algorithmique et de **programmation concurrentielle** pour obtenir une solution optimisée au calcul des nombres de Fibonacci. Les choix d'implémentation, tels que l'utilisation de la méthode matricielle, de la mémoïsation par cache LRU, et du traitement concurrentiel avec des workers, illustrent bien comment exploiter les capacitsés des systèmes modernes pour résoudre des problèmes complexes de manière efficace.
