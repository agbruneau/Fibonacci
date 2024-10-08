# **[Référence biblique de Genèse 1 : ](https://www.bible.com/bible/104/GEN.1.NBS?parallel=3345)** Calcul de la suite de Fibonacci par divers Algorithmes

## Introduction

Ce projet présente une analyse rigoureuse de plusieurs méthodes d'implémentation de la suite de Fibonacci en utilisant le langage de programmation Golang. L'objectif est d'évaluer systématiquement les performances de chaque méthode en fonction de la taille des termes à calculer, tout en examinant leur complexité algorithmique respective.

La suite de Fibonacci est définie comme une séquence dans laquelle chaque terme est la somme des deux précédents, les deux premiers termes étant fixés à 0 et 1 :

```
F(0) = 0, F(1) = 1, F(n) = F(n-1) + F(n-2) pour n ≥ 2
```

Les sections suivantes examinent diverses approches algorithmiques pour le calcul de cette suite, chacune ayant des caractéristiques distinctes en termes de performances et de complexité temporelle.

## Algorithmes Implémentés

### 1. Algorithme Récursif Naïf

Cet algorithme applique la définition mathématique de la suite de Fibonacci en utilisant une approche récursive directe. La complexité exponentielle, O(2^n), en fait une méthode inefficace pour les valeurs élevées de n, principalement en raison de la redondance des appels récursifs. Les recalculs multiples des mêmes sous-problèmes induisent une inefficience manifeste.

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

L'approche itérative évite la redondance des calculs récursifs en utilisant une simple boucle pour cumuler les valeurs nécessaires. Sa complexité temporelle linéaire, O(n), lui confère une meilleure efficacité que l'approche récursive naïve, tout en réduisant significativement la consommation de mémoire.

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

### 3. Algorithme avec Mémoïsation (Récursivité Optimisée)

Cet algorithme améliore l'efficacité de la récursivité en introduisant la mémoïsation, c'est-à-dire la mémorisation des résultats intermédiaires pour éviter les calculs redondants. Cette approche réduit la complexité temporelle à O(n), transformant la récursivité en une solution beaucoup plus efficace, adaptée aux valeurs élevées de n.

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

Cet algorithme repose sur l'exponentiation matricielle pour calculer le terme n de la suite de Fibonacci. En représentant la relation de récurrence sous forme matricielle, il est possible d'appliquer une exponentiation rapide qui réduit la complexité à O(log n). Cette méthode est particulièrement efficace pour les valeurs élevées de n, offrant une solution rapide et optimale.

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

### 5. Algorithme Utilisant la Formule de Binet

La formule de Binet offre une solution fermée pour calculer directement le n-ième terme de la suite de Fibonacci. La complexité est constante, O(1), ce qui la rend théoriquement très rapide. Cependant, en pratique, cette méthode est limitée par les erreurs d'arrondi liées aux nombres en virgule flottante, la rendant moins fiable pour des valeurs de n très élevées.

**Code :**

```go
import "math"

func FibonacciBinet(n int) int {
    phi := (1 + math.Sqrt(5)) / 2
    return int(math.Round(math.Pow(phi, float64(n)) / math.Sqrt(5)))
}
```

### 6. Algorithme de Calcul de Fibonacci par la Méthode du Doublement

L'algorithme du doublement (ou "doubling") permet de calculer efficacement le terme n de la suite de Fibonacci en utilisant une approche récursive qui divise le problème par deux à chaque étape. Cette méthode présente une complexité logarithmique, O(log n), similaire à l'exponentiation matricielle, et est particulièrement adaptée aux grandes valeurs de n, assurant une efficacité optimale tant en termes de temps de calcul que de consommation mémoire.

**Code :**

```go
func FibonacciDoubling(n int) (int, int) {
    if n == 0 {
        return 0, 1
    }
    a, b := FibonacciDoubling(n / 2)
    c := a * (2*b - a)
    d := a*a + b*b
    if n%2 == 0 {
        return c, d
    } else {
        return d, c + d
    }
}

func GetFibonacci(n int) int {
    result, _ := FibonacciDoubling(n)
    return result
}
```

## Comparaison des Algorithmes

Pour le calcul de petits termes de la suite de Fibonacci, tous les algorithmes sont adéquats. Toutefois, pour des valeurs élevées (par exemple, F(50) ou plus), les performances des différentes méthodes divergent significativement :

1. **Récursif Naïf** : Complexité exponentielle (O(2^n)), inefficace pour n > 30 en raison de la redondance excessive des appels.
2. **Itératif** : Complexité linéaire (O(n)), performant et économe en mémoire.
3. **Mémoïsation** : Complexité linéaire (O(n)), évite la redondance et améliore l'efficacité des calculs récursifs.
4. **Matrice de Puissance** : Complexité logarithmique (O(log n)), offre une excellente performance pour les grandes valeurs de n.
5. **Formule de Binet** : Complexité constante (O(1)), rapide mais sujette aux erreurs d'arrondi pour des valeurs élevées.
6. **Doublement** : Complexité logarithmique (O(log n)), alternative performante à l'exponentiation matricielle, optimisée pour les calculs récursifs.

## Conclusion

Pour le calcul de termes élevés de la suite de Fibonacci, l'**algorithme de la matrice de puissance** et l'**algorithme du doublement** sont les plus recommandés en raison de leur complexité logarithmique, qui offre un compromis optimal entre rapidité et précision.

L'**algorithme itératif** reste une option solide pour sa simplicité d'implémentation et sa constance en termes de performance.

La **formule de Binet** peut être envisagée pour des calculs rapides dans les cas où la précision absolue n'est pas essentielle.

## Licence

Ce projet est distribué sous la licence MIT. Pour plus de détails, veuillez consulter le fichier [LICENSE](LICENSE).
