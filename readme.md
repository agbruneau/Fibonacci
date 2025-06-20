# Calculateur de Fibonacci Concurrent en Go

Ce projet est un outil en ligne de commande écrit en Go pour calculer le n-ième nombre de Fibonacci. Sa caractéristique principale est l'implémentation de plusieurs algorithmes distincts, les exécutant en parallèle pour comparer leurs performances, leur consommation mémoire et valider leurs résultats.

L'application est conçue à la fois comme un outil pratique et une démonstration de plusieurs concepts avancés de Go, tels que la concurrence, la gestion de contexte, l'optimisation de la mémoire et les tests de performance (benchmarking).

✨ Fonctionnalités

*   **Calcul de Très Grands Nombres**: Utilise le paquet `math/big` pour calculer des nombres de Fibonacci bien au-delà des limites des types entiers standards.
*   **Exécution Concurrente**: Lance plusieurs algorithmes en parallèle à l'aide de goroutines pour une comparaison directe des performances.
*   **Multi-Algorithme**: Implémente quatre méthodes de calcul différentes avec des caractéristiques de performance distinctes :
    *   Doublage Rapide (Fast Doubling)
    *   Exponentiation Matricielle (Matrix Exponentiation)
    *   Formule de Binet
    *   Méthode Itérative
*   **Affichage de la Progression**: Montre en temps réel la progression de chaque algorithme sur une seule ligne qui se met à jour.
*   **Gestion du Délai d'Attente (Timeout)**: Utilise `context.WithTimeout` pour assurer que le programme se termine proprement si les calculs prennent trop de temps.
*   **Optimisation de la Mémoire**: Emploie un `sync.Pool` pour recycler les objets `*big.Int`, réduisant la pression sur le Ramasse-Miettes (Garbage Collector).
*   **Suite de Tests Complète**: Inclut des tests unitaires pour valider la correction des algorithmes et des benchmarks pour mesurer formellement leurs performances.
*   **Exécution Sélective des Algorithmes**: Permet aux utilisateurs de spécifier quels algorithmes exécuter via une option en ligne de commande.

🛠️ Prérequis

*   Go (version 1.18+ recommandée, car le projet utilise `go fmt ./...` et certaines pratiques plus récentes. La logique principale pourrait fonctionner sur Go 1.16+ mais 1.18+ est conseillé pour une compatibilité totale avec l'outillage de l'environnement de développement.)
*   Git (pour cloner le projet)

🚀 Installation

1.  Clonez le dépôt sur votre machine locale :
    ```sh
    git clone https://github.com/votre-nom-utilisateur/votre-depot.git
    ```
    (Remplacez l'URL par l'URL réelle de votre dépôt)

2.  Naviguez vers le répertoire du projet :
    ```sh
    cd votre-depot
    ```

💻 Utilisation

L'outil est exécuté directement depuis la ligne de commande.

**Exécution Simple**

Pour exécuter le calcul avec les valeurs par défaut (n=100000, timeout=1m, tous les algorithmes) :
```sh
go run .
```

**Options en Ligne de Commande**

Vous pouvez personnaliser l'exécution avec les options suivantes :

*   `-n <nombre>` : Spécifie l'index `n` du nombre de Fibonacci à calculer (entier non-négatif). Défaut : `100000`.
*   `-timeout <durée>` : Spécifie le délai d'attente global pour l'exécution (ex: `30s`, `2m`, `1h`). Défaut : `1m`.
*   `-algorithms <liste>` : Liste d'algorithmes séparés par des virgules à exécuter. Disponibles : `Fast Doubling`, `Matrix 2x2`, `Binet`, `Iterative`. Utilisez `all` pour tous les exécuter. Les noms sont insensibles à la casse (ex: "fast doubling", "matrix 2x2", "binet", "iterative"). Défaut : `all`.

**Exemples**

Calculer F(500 000) avec un délai d'attente de 30 secondes, en exécutant uniquement les algorithmes Fast Doubling et Iterative :
```sh
go run . -n 500000 -timeout 30s -algorithms "fast doubling,iterative"
```

Calculer F(1 000 000) avec un délai d'attente de 5 minutes, en exécutant tous les algorithmes :
```sh
go run . -n 1000000 -timeout 5m -algorithms all
```
Ou simplement (puisque `all` est la valeur par défaut pour les algorithmes) :
```sh
go run . -n 1000000 -timeout 5m
```

**Exemple de Sortie**
```
2023/10/27 10:30:00 Calculating F(200000) with a timeout of 1m...
2023/10/27 10:30:00 Algorithms to run: Fast Doubling, Matrix 2x2, Binet, Iterative
2023/10/27 10:30:00 Launching concurrent calculations...
Fast Doubling:   100.00%   Matrix 2x2:      100.00%   Binet:           100.00%   Iterative:       100.00%
2023/10/27 10:30:01 Calculations finished.

--------------------------- ORDERED RESULTS ---------------------------
Fast Doubling    : 8.8475ms     [OK              ] Result: 25974...03125
Iterative        : 12.5032ms    [OK              ] Result: 25974...03125
Matrix 2x2       : 18.0673ms    [OK              ] Result: 25974...03125
Binet            : 43.1258ms    [OK              ] Result: 25974...03125
------------------------------------------------------------------------

🏆 Fastest algorithm (that succeeded): Fast Doubling (8.848ms)
Number of digits in F(200000): 41798
Value (scientific notation) ≈ 2.59740692e+41797
✅ All valid results produced are identical.
2023/10/27 10:30:01 Program finished.
```

🧠 Algorithmes Implémentés

1.  **Doublage Rapide (Fast Doubling)**
    L'un des algorithmes connus les plus rapides pour les grands entiers. Il utilise les identités :
    *   `F(2k) = F(k) * [2*F(k+1) – F(k)]`
    *   `F(2k+1) = F(k)² + F(k+1)²`
    pour réduire significativement le nombre d'opérations. Complexité : O(log n) opérations arithmétiques.

2.  **Exponentiation Matricielle (Matrix Exponentiation 2x2)**
    Une approche classique basée sur la propriété que l'élévation de la matrice `Q = [[1,1],[1,0]]` à la puissance `k` donne :
    ```
    Q^k  =  | F(k+1)  F(k)   |
           | F(k)    F(k-1) |
    ```
    Le calcul de Q^(n-1) (pour obtenir F(n) comme élément en haut à gauche) est optimisé en utilisant l'exponentiation par carrés. Complexité : O(log n) multiplications de matrices.

3.  **Formule de Binet**
    Une solution analytique utilisant le nombre d'or (φ) :
    `F(n) = (φ^n - ψ^n) / √5`, où `φ = (1+√5)/2` et `ψ = (1-√5)/2`.
    Elle est calculée en utilisant des nombres à virgule flottante de haute précision (`big.Float`). Bien qu'élégante, elle est généralement moins performante pour le calcul direct et peut souffrir d'erreurs de précision pour de très grandes valeurs de `n`.

4.  **Méthode Itérative**
    Calcule les nombres de Fibonacci en itérant depuis F(0)=0 et F(1)=1 jusqu'à F(n) en utilisant la définition fondamentale `F(k) = F(k-1) + F(k-2)`.
    Cette méthode est simple à comprendre et très efficace en mémoire (surtout lorsque `sync.Pool` est utilisé pour les objets `big.Int`). Cependant, avec O(n) opérations arithmétiques (chacune sur des nombres potentiellement grands), elle est significativement plus lente pour les grandes valeurs de `n` par rapport aux méthodes logarithmiques.

🏗️ Architecture du Code

La base de code est organisée en plusieurs fichiers Go pour une meilleure modularité :

*   `main.go`: Contient la logique principale de l'application, y compris l'analyse des options en ligne de commande, l'orchestration de l'exécution concurrente des algorithmes via des goroutines, et l'affichage final des résultats.
*   `algorithms.go`: Abrite les implémentations des différents algorithmes de calcul de Fibonacci. Cela inclut la définition du type `fibFunc` et ses implémentations concrètes (par ex., `fibFastDoubling`, `fibMatrix`, `fibBinet`, `fibIterative`).
*   `utils.go`: Fournit des fonctions utilitaires partagées à travers l'application. Les composants clés sont le `progressPrinter` pour l'affichage en temps réel de la progression et l'assistant `newIntPool` pour la gestion du `sync.Pool` d'objets `*big.Int`.
*   `main_test.go`: Contient une suite complète de tests unitaires pour vérifier la correction de chaque algorithme et des benchmarks pour mesurer leurs caractéristiques de performance (temps d'exécution et allocations mémoire).

La concurrence est gérée à l'aide d'un `sync.WaitGroup` pour s'assurer que toutes les goroutines de calcul se terminent avant que le programme ne procède à l'agrégation des résultats. Les mises à jour de progression de chaque tâche concurrente sont envoyées via un canal partagé (`progressAggregatorCh`) à la goroutine `progressPrinter`, qui les consolide et les affiche sur une seule ligne dans la console.

✅ Tests

Le projet inclut une suite complète de tests pour assurer la correction et mesurer les performances.

**Exécuter les Tests Unitaires**

Pour vérifier que tous les algorithmes implémentés produisent des nombres de Fibonacci corrects pour un ensemble de valeurs connues (y compris les cas limites) :
```sh
go test -v ./...
```
Cette commande exécute tous les tests dans le paquet courant et tous les sous-paquets.

**Exécuter les Benchmarks**

Pour mesurer et comparer les performances (temps d'exécution et allocations mémoire) de chaque algorithme :
```sh
go test -bench . ./...
```
Cette commande exécute tous les benchmarks dans le paquet courant et les sous-paquets. Le `.` indique tous les benchmarks.
Pour exécuter les benchmarks pour un algorithme spécifique ou un groupe, vous pouvez utiliser l'option `-bench` avec une expression régulière. Par exemple, pour bencher uniquement la méthode Itérative :
```sh
go test -bench=BenchmarkFibIterative ./...
```
Ou pour bencher tous les algorithmes Fibonacci :
```sh
go test -bench=Fib ./...
```

📜 Licence

Ce projet est distribué sous la Licence MIT. (Typiquement, un fichier `LICENSE` serait inclus dans le dépôt avec le texte intégral de la Licence MIT.)
