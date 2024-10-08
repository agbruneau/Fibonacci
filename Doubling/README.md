# Calcul de Fibonacci par la Méthode du Doublement avec Mémoïsation et Benchmark

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/Doubling/Fibonacci%20Golang%20Sequence%20Diagram.jpeg)

## Introduction

Ce projet, implémenté en Go (Golang), présente un algorithme avancé pour le calcul des nombres de Fibonacci, basé sur la **méthode du doublement** couplée à la **mémoïsation**. L'objectif est de permettre le calcul de termes de Fibonacci extrêmement élevés tout en optimisant les performances grâce à une approche de **division et conquête**, l'utilisation de la **concurrence avec des goroutines**, ainsi qu'un **cache LRU** (Least Recently Used) pour minimiser les recalculs inutiles. Ce document vise à fournir une explication rigoureuse du fonctionnement de l'algorithme, en explorant les concepts mathématiques et informatiques sous-jacents, ainsi que ses avantages dans un contexte de calcul intensif. Ce projet s'adresse principalement aux chercheurs, aux développeurs avancés, ainsi qu'aux étudiants en informatique intéressés par l'optimisation des algorithmes numériques.

## Contexte et Justification

Le calcul des nombres de **Fibonacci** est un problème classique de l'algorithmique et des mathématiques appliquées. Si les méthodes itératives ou récursives naïves peuvent être suffisantes pour des valeurs relativement modestes de `n`, le calcul de termes pour des indices élevés exige des algorithmes plus sophistiqués afin de surmonter les limitations de performance inhérentes et d'éviter des temps de calcul exponentiellement longs. Le présent projet utilise l'**algorithme de doublement**, qui permet de calculer les termes de Fibonacci de manière efficace, même pour des indices très grands, en exploitant les propriétés binaires des entiers et une stratégie de division et conquête.

L'intégration de **goroutines** (unités d'exécution concurrente dans Go) associée à un **cache LRU** permet également de gérer simultanément plusieurs calculs, réduisant ainsi le temps de calcul global, particulièrement dans les contextes où des valeurs sont répétées ou peuvent être parallélisées.

## Structure du Code

### Bibliothèques Utilisées

- **math/big** : Gère les très grands entiers qui ne peuvent pas être représentés par les types natifs de Go tels que `int64`.
- **math/bits** : Utilisée pour obtenir la longueur en bits d'un entier, ce qui est crucial pour l'algorithme de doublement.
- **sync** : Assure la synchronisation entre les goroutines lors de l'exécution parallèle des tâches.
- **time** : Permet de mesurer la durée d'exécution des calculs de Fibonacci, facilitant ainsi l'analyse des performances.
- **github.com/hashicorp/golang-lru** : Bibliothèque implémentant un cache LRU afin d'optimiser le stockage des résultats calculés, minimisant la redondance.

### Méthode du Doublement

L'algorithme de **doublement** repose sur des propriétés spécifiques des nombres de Fibonacci, permettant de les calculer en **temps logarithmique** par rapport à l'indice `n`, ce qui s'avère significativement plus efficace que les méthodes itératives ou récursives classiques, dont les complexités sont linéaires ou exponentielles. La réduction de la complexité de l'algorithme est particulièrement avantageuse lorsque `n` est très élevé. Les formules fondamentales utilisées dans cet algorithme sont :

- `F(2k) = F(k) * [2 * F(k+1) - F(k)]`
- `F(2k + 1) = F(k)^2 + F(k+1)^2`

Ces relations permettent de diviser le problème en sous-problèmes plus petits, qui peuvent être combinés de manière optimale. Cela réduit considérablement le nombre total d'opérations requises, rendant le calcul bien plus efficace comparé aux approches récursives ou itératives traditionnelles.

### Mémoïsation avec Cache LRU

Pour réduire davantage le temps de calcul, une stratégie de **mémoïsation** est mise en œuvre via un **cache LRU**. Ce cache stocke les résultats précédemment calculés, permettant une récupération instantanée si la même valeur de Fibonacci est redemandée, évitant ainsi les recalculs. Cette approche réduit significativement les coûts temporels, en particulier pour des calculs répétitifs, comme ceux rencontrés dans des applications de simulation ou de modélisation financière. Dans ces contextes, l'utilisation d'un cache LRU améliore grandement la performance globale en évitant les calculs redondants.

### Parallélisation avec Goroutines

Le programme est conçu pour exploiter la **concurrence** grâce à l'utilisation des **goroutines** (unités légères de concurrence en Go) et d'un modèle de **workers**. L'exécution simultanée des calculs de Fibonacci pour différents indices, facilitée par les goroutines, maximise l'efficacité du programme, particulièrement sur des systèmes multicœurs. Cela permet non seulement de réduire les temps de calcul, mais aussi de mieux utiliser les ressources disponibles sur la machine hôte.

### Fonctionnalités Principales

1. **Calcul de Fibonacci pour des indices très élevés** : Jusqu'à une valeur maximale spécifiée par `MAX_FIB_VALUE` (500 000 001).
2. **Cache LRU** : Permet d'éviter les recalculs inutiles, augmentant ainsi l'efficacité et la rapidité du programme.
3. **Parallélisation** : Calculs concurrentiels grâce à des **goroutines** et un pool de **workers**, permettant une exécution simultanée et une réduction des temps de calcul.
4. **Benchmarking** : Mesure du temps d'exécution moyen pour différentes valeurs de `n`, fournissant une évaluation rigoureuse de l'efficacité de l'algorithme.

## Décomposition des Fonctions

### Fonction `fibDoubling(n int) (*big.Int, error)`

Cette fonction constitue le cœur de l'algorithme. Elle implémente la méthode du doublement pour calculer le nième nombre de Fibonacci.
- Pour `n < 2`, la fonction retourne directement `n` (F(0) = 0, F(1) = 1).
- Pour `n > MAX_FIB_VALUE`, la fonction retourne une erreur, indiquant que la valeur est trop grande pour être calculée raisonnablement.
- Pour des valeurs élevées de `n`, la fonction utilise une approche itérative de la méthode du doublement, en s'appuyant sur les propriétés mathématiques décrites précédemment.

### Fonction `fibInt64(n int) int64`

Pour les petites valeurs de `n` (inférieures à 93), cette fonction utilise un entier `int64` pour effectuer les calculs de Fibonacci de manière itérative. Cette approche est plus rapide et plus simple pour des valeurs relativement petites.

### Fonction `fibDoublingHelperIterative(n int) *big.Int`

Cette fonction réalise le calcul de Fibonacci par doublement en utilisant une boucle itérative sur les bits de `n`. Chaque bit est analysé afin d'appliquer les formules du doublement, permettant un calcul efficace de `F(2k)` et `F(2k + 1)`. Les calculs intermédiaires utilisent l'arithmétique sur grands entiers (`math/big`) pour éviter tout débordement.

### Benchmarking

La fonction `benchmarkFibWithWorkerPool(nValues []int, repetitions int, workerCount int)` permet d'effectuer un **benchmark** des calculs de Fibonacci pour différentes valeurs de `n`.
- **Canal de travaux** : Les tâches de calcul sont ajoutées à un canal, et les **workers** (lancés sous forme de goroutines) prennent chaque tâche pour la traiter.
- **Mesure du temps moyen** : Chaque calcul est répété un certain nombre de fois, et la durée moyenne d'exécution est mesurée pour évaluer les performances de l'algorithme de manière empirique.

## Exemple d'Utilisation

Pour exécuter le programme, vous pouvez lancer la commande suivante :
```sh
$ go run main.go
```
Cela calculera des valeurs de Fibonacci pour une liste donnée (`nValues`) en utilisant la parallélisation, et mesurera les performances moyennes pour chaque valeur.

### Commandes Principales

- **nValues** : Liste des valeurs de `n` pour lesquelles le calcul de Fibonacci est effectué.
- **repetitions** : Nombre de répétitions pour chaque calcul afin d'obtenir une mesure précise de la performance.
- **workerCount** : Nombre de goroutines utilisées pour le calcul parallèle.

## Conclusion

Ce programme, écrit en Go, représente une implémentation avancée et optimisée pour le calcul des nombres de Fibonacci, combinant des techniques telles que la **méthode du doublement**, la **mémoïsation** via un **cache LRU**, et la **parallélisation** avec des goroutines. Ces stratégies permettent de calculer de manière rapide et efficace des termes extrêmement élevés de la suite de Fibonacci, tout en optimisant l'utilisation des ressources grâce à la concurrence et au stockage intelligent des résultats intermédiaires.

Pour les chercheurs et développeurs intéressés par l'optimisation des algorithmes de calcul intensif, ce projet constitue une excellente base d'expérimentation. Il offre des perspectives d'optimisation supplémentaires, comme l'intégration de techniques de parallélisation plus avancées, l'utilisation de variantes de l'algorithme de doublement, ou l'implémentation de caches distribués afin de mieux gérer la mémoire dans des environnements distribués. N'hésitez pas à ajuster les paramètres tels que `MAX_FIB_VALUE`, `repetitions` et `workerCount` afin de tester l'impact de la parallélisation et de la mémoïsation sur les performances et ainsi explorer les limites de cette implémentation.
