# Méthode de la Matrice de Puissance pour Fibonacci - Implémentation en Go

![Diagramme de l'algorithme de Fibonacci](https://github.com/agbruneau/Fibonacci/blob/main/Parallel/Fibonacci%20Golang%20Sequence%20Diagram.jpeg)

## Description

Ce projet présente une implémentation sophistiquée en Go pour le calcul des nombres de Fibonacci en utilisant la **méthode de la matrice de puissance**. Cette technique, qui exploite l'exponentiation rapide des matrices, atteint une complexité temporelle de **O(log(n))**. L'algorithme surpasse significativement les méthodes itératives et récursives en réduisant le nombre de calculs nécessaires. Un cache de type **LRU (Least Recently Used)** est intégré pour optimiser les performances en mémorisant les valeurs calculées, ce qui est particulièrement avantageux dans le contexte de requêtes répétitives sur des valeurs similaires.

## Fonctionnalités

- **Calcul avancé des nombres de Fibonacci** : Implémentation de l'exponentiation matricielle pour des performances accrues.
- **Cache LRU** : Utilisation d'une mémoire cache pour conserver les résultats intermédiaires, minimisant le besoin de recalcul.
- **Traitement parallèle** : Utilisation intensive de **goroutines** pour une exécution concurrente efficace des calculs de Fibonacci.
- **Benchmark des performances** : Évaluation systématique des performances via des tests sur des valeurs prédéfinies avec un calcul du temps moyen d'exécution.

## Structure du Code

### Fichiers
- **FibonacciMatrix.go** : Fichier principal contenant l'implémentation du calcul de Fibonacci ainsi que le benchmark des performances.

### Fonctions Principales

- `fibMatrixPower(n int) (*big.Int, error)` : Calcule le nième nombre de Fibonacci par exponentiation matricielle.
- `matrixPower(matrix [2][2]*big.Int, n int) [2][2]*big.Int` : Effectue l'exponentiation rapide d'une matrice à la puissance `n`.
- `matrixMultiply(a, b [2][2]*big.Int) [2][2]*big.Int` : Multiplie deux matrices 2x2 de type `big.Int`.
- `benchmarkFibWithWorkerPool(ctx context.Context, nValues []int, repetitions int, workerCount int)` : Effectue des benchmarks sur une série de valeurs en exploitant la concurrence.

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
L'algorithme repose sur l'exponentiation d'une matrice de base pour obtenir le nième nombre de Fibonacci. La matrice fondamentale utilisée est :

\[
F = \begin{bmatrix} 1 & 1 \\ 1 & 0 \end{bmatrix}
\]

En élevant cette matrice à la puissance `(n-1)`, la valeur de Fibonacci `F(n)` est localisée dans l'entrée `[0][0]` de la matrice résultante. L'utilisation de l'exponentiation rapide réduit la complexité de calcul à **O(log(n))**, rendant cette méthode nettement plus performante que les approches itératives ou récursives classiques.

### Cache LRU
Pour accroître l'efficacité des calculs, un cache **LRU** est utilisé afin de mémoriser les résultats des calculs précédents. Cela permet de réduire les répétitions de calculs pour des valeurs déjà traitées, optimisant ainsi le temps d'exécution. Ce cache est mis en œuvre à l'aide de la bibliothèque `golang-lru`.

### Concurrence avec Goroutines
Le programme emploie un **pool de workers** pour exécuter les calculs en parallèle. Cette approche est particulièrement bénéfique pour le benchmarking, où les calculs sur de multiples valeurs de Fibonacci sont effectués simultanément, équilibrant ainsi la charge entre les différents threads et réduisant le temps global d'exécution.

## Exemples d'Utilisation
- **Calculer une valeur spécifique de Fibonacci** : La fonction `fibMatrixPower(n)` peut être utilisée pour obtenir la valeur de Fibonacci pour un entier `n`. Les résultats sont mémorisés dans le cache pour des performances améliorées lors d'appels répétitifs.
- **Tester les performances** : La fonction `benchmarkFibWithWorkerPool` permet de mesurer le temps moyen de calcul des nombres de Fibonacci sur une série de valeurs, en exploitant la parallélisation grâce aux goroutines.

## Limites
- Le programme est limité à une valeur maximale de `500,000,001` pour `n` en raison des contraintes de mémoire et de complexité du calcul.
- Les valeurs sont représentées en utilisant `*big.Int` afin d'éviter les dépassements de capacité des entiers primitifs, ce qui peut rendre les calculs plus lents pour les petites valeurs.

## Contributions
Les contributions sont encouragées. Pour contribuer :
1. **Forkez le projet** : Créez votre propre copie du projet.
2. **Créez une branche** : Pour vos modifications, utilisez la commande suivante : `git checkout -b feature/nouvelle-fonctionnalité`.
3. **Effectuez des changements** : Modifiez le code source selon vos besoins.
4. **Committez vos changements** : `git commit -am 'Ajouter une nouvelle fonctionnalité'`.
5. **Poussez votre branche** : Envoyez vos modifications sur GitHub : `git push origin feature/nouvelle-fonctionnalité`.
6. **Ouvrez une Pull Request** : Soumettez vos changements pour revue.

## Licence
Ce projet est distribué sous la licence MIT - consultez le fichier [LICENSE](LICENSE) pour plus de détails.

## Auteur
- **Votre Nom** - [Votre Profil GitHub](https://github.com/votre-utilisateur)
