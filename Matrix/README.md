# Calcul de Fibonacci par la Méthode de la Matrice de Puissance avec Mémoïsation et Benchmark

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/Matrix/Fibonacci%20Golang%20Sequence%20Diagram.jpeg)

## Description

Ce projet propose une implémentation avancée du calcul des nombres de Fibonacci en utilisant la **méthode de la matrice de puissance** avec le langage Go. Cette approche repose sur l'exponentiation rapide des matrices, permettant d'atteindre une complexité temporelle de **O(log(n))**. En tirant parti de cette technique, l'algorithme réduit de manière significative le nombre d'opérations requises, surpassant largement les méthodes itératives et récursives traditionnelles. De plus, un cache de type **LRU (Least Recently Used)** est intégré pour optimiser les performances en mémorisant les valeurs calculées, ce qui est particulièrement avantageux dans le cadre de requêtes répétées sur des valeurs similaires.

## Fonctionnalités

- **Calcul avancé des nombres de Fibonacci** : Utilisation de l'exponentiation matricielle pour optimiser le calcul des valeurs de Fibonacci, offrant des gains substantiels de performance.
- **Cache LRU** : Implémentation d'une mémoire cache pour stocker les résultats intermédiaires et minimiser les besoins de recalcul, améliorant ainsi l'efficacité globale.
- **Traitement parallèle** : Exploitation intensive de **goroutines** pour exécuter des calculs en concurrence, augmentant la performance sur des systèmes multicœurs.
- **Benchmark des performances** : Évaluation systématique des performances par des tests sur des valeurs prédéfinies, permettant de mesurer le temps moyen d'exécution.

## Structure du Code

### Fichiers
- **FibonacciMatrix.go** : Fichier principal contenant l'implémentation de l'algorithme de calcul de Fibonacci ainsi que les fonctions de benchmarking.

### Fonctions Principales

- `fibMatrixPower(n int) (*big.Int, error)` : Calcule le nième nombre de Fibonacci en utilisant l'exponentiation matricielle.
- `matrixPower(matrix [2][2]*big.Int, n int) [2][2]*big.Int` : Effectue l'exponentiation rapide d'une matrice de base à la puissance `n`.
- `matrixMultiply(a, b [2][2]*big.Int) [2][2]*big.Int` : Multiplie deux matrices 2x2 de type `big.Int`.
- `benchmarkFibWithWorkerPool(ctx context.Context, nValues []int, repetitions int, workerCount int)` : Exécute des benchmarks sur une série de valeurs en utilisant la concurrence.

## Installation et Utilisation

### Prérequis

- **Go 1.16 ou supérieur**
- **Modules Go** pour la gestion des dépendances

### Installation

1. Clonez le dépôt :
   ```sh
   git clone https://github.com/votre-utilisateur/Fibonacci-Matrix.git
   ```
2. Naviguez dans le répertoire du projet :
   ```sh
   cd Fibonacci-Matrix
   ```
3. Initialisez le module Go :
   ```sh
   go mod init fibonacci-matrix
   ```
4. Installez les dépendances nécessaires :
   ```sh
   go get github.com/hashicorp/golang-lru
   ```

### Exécution

Pour exécuter le programme, utilisez la commande suivante :

```sh
go run FibonacciMatrix.go
```

Cette commande exécute le benchmark des performances sur une liste prédéfinie de valeurs de Fibonacci.

## Explications de l'Algorithme

### Méthode de la Matrice de Puissance

L'algorithme repose sur l'exponentiation d'une matrice fondamentale pour obtenir le nième nombre de Fibonacci. La matrice de base utilisée est :

\[
F = \begin{bmatrix} 1 & 1 \\ 1 & 0 \end{bmatrix}
\]

En élevant cette matrice à la puissance `(n-1)`, la valeur de Fibonacci `F(n)` est localisée dans l'entrée `[0][0]` de la matrice résultante. L'utilisation de l'exponentiation rapide réduit la complexité de calcul à **O(log(n))**, rendant cette méthode considérablement plus performante que les approches itératives ou récursives classiques. Cette efficacité découle de la réduction exponentielle du nombre d'opérations nécessaires, en tirant parti des propriétés binaires de l'indice `n`.

### Cache LRU

Pour accroître l'efficacité des calculs, un cache **LRU** est utilisé pour mémoriser les résultats des calculs précédents. Cela permet de réduire les répétitions de calculs pour des valeurs déjà traitées, optimisant ainsi le temps d'exécution, particulièrement dans les scénarios où des requêtes répétées sont effectuées. Ce cache est mis en œuvre à l'aide de la bibliothèque `golang-lru`, offrant une solution efficace pour minimiser la redondance et améliorer les performances globales.

### Concurrence avec Goroutines

Le programme emploie un **pool de workers** pour exécuter les calculs en parallèle, exploitant les **goroutines** de Go, qui sont des unités légères de concurrence. Cette approche est particulièrement bénéfique pour le benchmarking, où les calculs sur de multiples valeurs de Fibonacci peuvent être effectués simultanément, équilibrant la charge entre les différents threads. Cela conduit à une réduction significative du temps global d'exécution, surtout sur des architectures multicœurs, maximisant l'efficacité des ressources matérielles disponibles.

## Exemples d'Utilisation

- **Calculer une valeur spécifique de Fibonacci** : La fonction `fibMatrixPower(n)` peut être utilisée pour obtenir la valeur de Fibonacci pour un entier `n`. Les résultats sont mémorisés dans le cache, permettant des performances accrues lors de requêtes répétées.
- **Tester les performances** : La fonction `benchmarkFibWithWorkerPool` permet de mesurer le temps moyen de calcul des nombres de Fibonacci sur une série de valeurs, en exploitant la parallélisation par le biais des goroutines. Cela permet d'évaluer la scalabilité et l'efficacité de l'algorithme dans un environnement concurrent.

## Limites

- Le programme est limité à une valeur maximale de `500,000,001` pour `n`, en raison des contraintes de mémoire et de la complexité du calcul. Ce seuil est fixé pour garantir que les ressources matérielles nécessaires restent dans des limites raisonnables.
- Les valeurs sont représentées en utilisant `*big.Int` afin d'éviter les dépassements de capacité des entiers primitifs, ce qui peut rendre les calculs plus lents pour des valeurs relativement petites. L'utilisation de `*big.Int` permet toutefois de traiter des valeurs extrêmement grandes, au prix d'une légère dégradation des performances sur des entrées plus modestes.

## Contributions

Les contributions à ce projet sont les bienvenues. Pour contribuer, suivez les étapes suivantes :

1. **Forkez le projet** : Créez votre propre copie du projet.
2. **Créez une branche** : Pour vos modifications, utilisez la commande suivante : `git checkout -b feature/nouvelle-fonctionnalité`.
3. **Effectuez des changements** : Modifiez le code source selon vos besoins, en documentant vos ajouts et modifications.
4. **Committez vos changements** : `git commit -am 'Ajouter une nouvelle fonctionnalité'`.
5. **Poussez votre branche** : Envoyez vos modifications sur GitHub : `git push origin feature/nouvelle-fonctionnalité`.
6. **Ouvrez une Pull Request** : Soumettez vos changements pour revue. Décrivez clairement les modifications apportées et leurs impacts potentiels sur le projet.

## Licence

Ce projet est distribué sous la licence MIT. Pour plus de détails, veuillez consulter le fichier [LICENSE](LICENSE).
