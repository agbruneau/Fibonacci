# Programme Go : Calcul de Fibonacci par la Méthode du Doublement avec Mémoïsation et Benchmark

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/Fibonacci%20Golang%20Sequence%20Diagram.jpeg)


## Introduction

Ce projet en Go (Golang) implémente un algorithme avancé pour le calcul des nombres de Fibonacci, en utilisant la **méthode du doublement** combinée à la **mémoïsation**. L'objectif est de calculer des valeurs de Fibonacci extrêmement élevées tout en optimisant les performances via une approche basée sur la division et la conquête, la **concurrence avec des goroutines**, et un **cache LRU** (Least Recently Used) pour éviter les recalculs inutiles. Ce document vise à fournir une explication rigoureuse du fonctionnement du programme, en exposant ses concepts sous-jacents et ses avantages dans un contexte de calcul intensif. Il s'adresse principalement aux développeurs avancés, aux étudiants en informatique et aux chercheurs intéressés par l'optimisation des algorithmes de calcul.

## Contexte et Justification

Le calcul des nombres de **Fibonacci** est une problématique classique en algorithmique et en mathématiques appliquées. Bien que des méthodes itératives ou récursives naïves puissent être efficaces pour des valeurs relativement faibles de `n`, les calculs pour de grands indices nécessitent des algorithmes plus sophistiqués pour surmonter les limitations de performance et éviter des temps de calcul exponentiellement longs. Ce programme utilise l'**algorithme de doublement**, qui permet de calculer des valeurs de Fibonacci de manière efficace, même pour des indices extrêmement grands, en exploitant la nature binaire des nombres et la méthode de division et conquête.

En outre, l'intégration de **goroutines** (unités d'exécution concurrentes) combinée à un **cache LRU** permet au programme de gérer efficacement de nombreux calculs simultanés, minimisant ainsi le temps de calcul global, en particulier pour des valeurs répétées ou des charges de travail parallélisables.

## Structure du Code

### Bibliothèques Utilisées

- **math/big** : Pour gérer les très grands entiers qui ne peuvent être représentés par les types natifs de Go comme `int64`.
- **math/bits** : Pour obtenir la longueur en bits d'un entier, ce qui est crucial pour l'algorithme de doublement.
- **sync** : Pour la synchronisation des goroutines lors de l'exécution concurrente des tâches.
- **time** : Pour mesurer la durée d'exécution des calculs de Fibonacci, facilitant ainsi l'analyse des performances.
- **github.com/hashicorp/golang-lru** : Bibliothèque de cache LRU permettant d'optimiser le stockage des résultats précédemment calculés, réduisant ainsi la redondance.

### Méthode du Doublement

L'algorithme de **doublement** exploite des propriétés mathématiques spécifiques des nombres de Fibonacci pour les calculer en temps logarithmique par rapport à l'indice `n`, ce qui est significativement plus efficace que les méthodes itératives ou récursives classiques qui nécessitent un temps linéaire ou exponentiel. Cette réduction du temps de calcul est particulièrement avantageuse lorsque `n` est très grand, car elle permet de réduire la complexité globale de l'algorithme. Les formules fondamentales employées sont :
- `F(2k) = F(k) * [2 * F(k+1) - F(k)]`
- `F(2k + 1) = F(k)^2 + F(k+1)^2`

Ces formules permettent de réduire le nombre total d'opérations nécessaires en divisant le problème en sous-problèmes plus petits, puis en combinant les résultats de manière optimale, rendant ainsi le calcul bien plus efficace que les approches récursives ou itératives traditionnelles.

### Mémoïsation avec Cache LRU

Afin de réduire encore le temps de calcul, une stratégie de **mémoïsation** est mise en place via un **cache LRU**. Ce cache stocke les résultats précédemment calculés, de sorte que si la même valeur de Fibonacci est demandée ultérieurement, elle peut être récupérée instantanément sans recalcul. Cela réduit considérablement le coût en temps, surtout lorsque des valeurs identiques ou similaires sont demandées de manière répétée. Par exemple, dans des applications de simulation ou de modélisation financière où les mêmes calculs de Fibonacci sont effectués plusieurs fois, l'utilisation du cache permet d'éviter des recalculs coûteux, ce qui améliore grandement la performance globale.

### Parallélisation avec Goroutines

Le programme est conçu pour tirer parti de la **concurrence** en utilisant des **goroutines** (unités légères d'exécution concurrente dans Go qui facilitent le multitâche) et un modèle de **workers**. Grâce à plusieurs goroutines, il est possible de paralléliser les calculs de Fibonacci pour différents indices, augmentant ainsi l'efficacité globale du programme, particulièrement sur des systèmes multicœurs. Cela permet non seulement de réduire les temps d'attente, mais aussi de maximiser l'utilisation des ressources disponibles sur la machine hôte.

### Fonctionnalités Principales

1. **Calcul de Fibonacci pour des indices très élevés** : Jusqu'à une valeur maximale définie par `MAX_FIB_VALUE` (500 000 001 dans le code).
2. **Cache LRU** : Utilisation d'un cache pour éviter les recalculs, augmentant ainsi la rapidité et l'efficacité du programme.
3. **Parallélisation** : Calculs concurrentiels avec des **goroutines** et des **workers**, permettant une exécution simultanée et une réduction des temps de calcul.
4. **Benchmarking** : Mesure du temps d'exécution moyen pour différentes valeurs de `n`, permettant une évaluation précise de l'efficacité de l'algorithme.

## Décomposition des Fonctions

### Fonction `fibDoubling(n int) (*big.Int, error)`

Cette fonction constitue le cœur du programme. Elle implémente l'algorithme de doublement pour calculer le nième nombre de Fibonacci.
- Pour `n < 2`, la fonction retourne directement la valeur de `n` (F(0) = 0, F(1) = 1).
- Pour `n > MAX_FIB_VALUE`, la fonction retourne une erreur, car la valeur est considérée trop grande pour être calculée de manière raisonnable.
- Pour des valeurs élevées de `n`, la fonction utilise une version itérative de la méthode du doublement, se basant sur les propriétés mathématiques décrites précédemment.

### Fonction `fibInt64(n int) int64`

Pour les petites valeurs de `n` (inférieures à 93), cette fonction utilise un entier `int64` pour effectuer le calcul de Fibonacci de manière itérative, car cette approche est plus rapide pour ces valeurs limitées.

### Fonction `fibDoublingHelperIterative(n int) *big.Int`

Cette fonction réalise le calcul des nombres de Fibonacci en utilisant une boucle itérative sur les bits de `n`. Chaque bit est analysé pour appliquer les formules de doublement, permettant ainsi de calculer efficacement `F(2k)` et `F(2k + 1)`. Les calculs intermédiaires sont effectués à l'aide de l'arithmétique des grands entiers pour éviter les débordements.

### Benchmarking

La fonction `benchmarkFibWithWorkerPool(nValues []int, repetitions int, workerCount int)` effectue un **benchmark** des calculs de Fibonacci pour différentes valeurs d'indice.
- **Canal de travaux** : Les calculs sont ajoutés à un canal, et des **workers** (lancés sous forme de goroutines) prennent chacun une tâche pour la traiter.
- **Mesure du temps moyen** : Chaque calcul est répété un certain nombre de fois, et le temps d'exécution moyen est mesuré pour évaluer les performances de l'algorithme de manière empirique.

## Exemple d'Utilisation

Pour exécuter le programme, vous pouvez lancer la commande suivante :
```sh
$ go run main.go
```
Cela calculera des valeurs de Fibonacci pour une liste donnée (`nValues`) en utilisant la parallélisation et mesurera les performances moyennes pour chaque valeur.

### Commandes Principales

- **nValues** : Liste des valeurs de `n` pour lesquelles on souhaite effectuer le calcul de Fibonacci.
- **repetitions** : Nombre de répétitions pour chaque calcul afin d'obtenir une mesure de performance précise.
- **workerCount** : Nombre de goroutines utilisées pour le calcul concurrent.

## Conclusion

Ce programme Go est une implémentation sophistiquée et optimisée pour le calcul des nombres de Fibonacci, utilisant des techniques avancées telles que la **méthode du doublement**, la **mémoïsation** avec un **cache LRU**, et la **parallélisation** avec des goroutines. Ces approches combinées permettent de calculer rapidement et efficacement des valeurs très élevées de Fibonacci, tout en optimisant l'utilisation des ressources disponibles grâce au cache et à la concurrence.

Pour les chercheurs et développeurs intéressés par l'optimisation des algorithmes de calcul intensif avec Go, ce projet constitue une excellente base pour l'étude et l'expérimentation. Vous pouvez envisager d'explorer des optimisations supplémentaires, comme l'utilisation de techniques de parallélisation plus fines, des variantes d'algorithmes de doublement, ou l'intégration de caches distribués pour une meilleure gestion de la mémoire sur des systèmes distribués. N'hésitez pas à ajuster les paramètres tels que `MAX_FIB_VALUE`, `repetitions` et `workerCount` pour tester l'impact de la parallélisation et de la mémoïsation sur les performances, et pour explorer les limites de l'implémentation.