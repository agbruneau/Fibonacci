# README - Calcul de Fibonacci avec Multiples Algorithmes en Go

## Introduction
Ce projet explore différentes implémentations du calcul de la suite de Fibonacci en utilisant le langage de programmation Golang. L'objectif est de comparer divers algorithmes afin de déterminer lesquels sont les plus performants pour le calcul de grands nombres de la suite de Fibonacci.

La suite de Fibonacci est une série où chaque terme est la somme des deux termes précédents, débutant par 0 et 1 :

```
F(0) = 0, F(1) = 1, F(n) = F(n-1) + F(n-2) pour n ≥ 2
```

Dans ce projet, nous analysons plusieurs techniques et algorithmes pour calculer ces termes, chacun ayant des caractéristiques distinctes en termes de complexité temporelle et de performance.

## Algorithmes Implémentés

### 1. Algorithme Récursif Naïf
Cet algorithme est une implémentation directe de la définition mathématique de la suite de Fibonacci. Toutefois, il est extrêmement inefficace pour de grands nombres en raison de sa complexité exponentielle, O(2^n).

**Code :**
```go
func FibonacciRecursif(n int) int {
    if n <= 1 {
        return n
    }
    return FibonacciRecursif(n-1) + FibonacciRecursif(n-2)
}
```

### 2. Algorithme Itératif
L'algorithme itératif évite la récursivité en utilisant une boucle simple pour accumuler les valeurs de Fibonacci. Cet algorithme a une complexité temporelle linéaire, O(n), et est beaucoup plus performant que la méthode récursive naïve.

**Code :**
```go
func FibonacciIteratif(n int) int {
    if n <= 1 {
        return n
    }
    a, b := 0, 1
    for i := 2; i <= n; i++ {
        a, b = b, a+b
    }
    return b
}
```

### 3. Algorithme par Mémoïsation (Récursif avec Cache)
Cet algorithme améliore la méthode récursive en utilisant la mémoïsation pour stocker les résultats intermédiaires dans un tableau. La complexité est ainsi réduite à O(n), rendant l'approche significativement plus efficace pour des calculs de grande envergure.

**Code :**
```go
func FibonacciMemo(n int, memo map[int]int) int {
    if n <= 1 {
        return n
    }
    if val, ok := memo[n]; ok {
        return val
    }
    memo[n] = FibonacciMemo(n-1, memo) + FibonacciMemo(n-2, memo)
    return memo[n]
}
```

### 4. Algorithme Utilisant la Matrice de Puissance
Cet algorithme utilise l'exponentiation par matrices pour calculer le terme n de la suite de Fibonacci en O(log n). Il repose sur la propriété que la multiplication répétée d'une matrice permet de générer les termes de Fibonacci de manière efficace.

L'algorithme de matrice de puissance est particulièrement intéressant pour des valeurs élevées de n, car il réduit considérablement la quantité de calculs requis par rapport aux approches naïves. L'idée fondamentale est de représenter la relation de récurrence de Fibonacci sous forme de multiplication matricielle. Plus spécifiquement, la suite de Fibonacci peut être exprimée comme suit :

```
| F(n)   | = | 1  1 |^(n-1) * | F(1) |
| F(n-1) |   | 1  0 |        | F(0) |
```

Cette représentation permet d'utiliser l'exponentiation rapide des matrices, réduisant ainsi la complexité temporelle à O(log n), car nous divisons exponentiellement la puissance à chaque étape. Cela fait de cet algorithme l'une des approches les plus efficaces pour des calculs de grande envergure.

**Code :**
```go
func FibonacciMatrix(n int) int {
    if n == 0 {
        return 0
    }
    F := [2][2]int{{1, 1}, {1, 0}}
    power(&F, n-1)
    return F[0][0]
}

func power(F *[2][2]int, n int) {
    if n == 0 || n == 1 {
        return
    }
    M := [2][2]int{{1, 1}, {1, 0}}
    power(F, n/2)
    multiply(F, F)
    if n%2 != 0 {
        multiply(F, &M)
    }
}

func multiply(F, M *[2][2]int) {
    x := F[0][0]*M[0][0] + F[0][1]*M[1][0]
    y := F[0][0]*M[0][1] + F[0][1]*M[1][1]
    z := F[1][0]*M[0][0] + F[1][1]*M[1][0]
    w := F[1][0]*M[0][1] + F[1][1]*M[1][1]

    F[0][0], F[0][1], F[1][0], F[1][1] = x, y, z, w
}
```

### 5. Algorithme par la Formule de Binet
Cet algorithme repose sur une formule fermée connue sous le nom de Formule de Binet pour calculer directement le n-ième terme de la suite de Fibonacci. Bien que théoriquement intéressant, il est limité par la précision des calculs à virgule flottante, surtout pour les grands n.

**Code :**
```go
import "math"

func FibonacciBinet(n int) int {
    phi := (1 + math.Sqrt(5)) / 2
    return int(math.Round(math.Pow(phi, float64(n)) / math.Sqrt(5)))
}
```

## Comparaison des Algorithmes
Pour le calcul de petits nombres de Fibonacci, tous les algorithmes fonctionnent de manière adéquate. Toutefois, pour des valeurs élevées (par exemple, F(50) ou plus), les performances diffèrent considérablement :

1. **Récursif Naïf** : Complexité exponentielle (O(2^n)), très inefficace pour n > 30.
2. **Itératif** : Complexité linéaire (O(n)), avec une bonne performance même pour de grands n.
3. **Mémoïsation** : Complexité linéaire (O(n)), offre une amélioration significative pour les calculs récursifs, notamment en réduisant la redondance.
4. **Matrice de Puissance** : Complexité logarithmique (O(log n)), constitue un choix idéal pour les très grands n en raison de sa rapidité et de sa précision.
5. **Formule de Binet** : Complexité constante (O(1)), bien que sujette aux erreurs de précision pour les très grands n.

## Conclusion
Pour le calcul de grands termes de la suite de Fibonacci (par exemple F(1000) et plus), l'**algorithme basé sur la matrice de puissance** est le plus efficace en raison de sa complexité logarithmique, combinant rapidité et précision.

L'**algorithme itératif** est également une option robuste pour des calculs où la lisibilité du code et la simplicité sont des priorités, tout en offrant des performances adéquates.

Pour des calculs constants et rapides, la **formule de Binet** peut être utilisée, mais uniquement dans des contextes où la précision n'est pas critique.

## Comment Contribuer
Les contributions à ce projet sont les bienvenues, qu'il s'agisse d'optimisations des algorithmes existants ou de nouvelles implémentations innovantes. Veuillez soumettre une pull request ou ouvrir une issue pour discuter de vos idées.

## Licence
Ce projet est sous licence MIT. Consultez le fichier [LICENSE](LICENSE) pour plus de détails.

---
Merci d'avoir exploré ce projet sur le calcul des nombres de Fibonacci en Go. Pour toute question ou suggestion, n'hésitez pas à nous contacter.
