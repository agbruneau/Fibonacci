# Algorithme Fast Doubling

> **Complexité** : O(log n) opérations arithmétiques  
> **Complexité réelle** : O(log n × M(n)) où M(n) est le coût de multiplication

## Introduction

L'algorithme **Fast Doubling** (ou "doublement rapide") est l'une des méthodes les plus efficaces pour calculer les nombres de Fibonacci. Il exploite les propriétés mathématiques de la suite pour réduire le nombre d'opérations à O(log n).

## Fondement Mathématique

### Forme Matricielle de Fibonacci

La suite de Fibonacci peut être exprimée sous forme matricielle :

```
[ F(n+1)  F(n)   ]   [ 1  1 ]^n
[                ] = [      ]
[ F(n)    F(n-1) ]   [ 1  0 ]
```

Cette relation est connue sous le nom de **matrice Q de Fibonacci**.

### Dérivation des Formules de Doublement

En élevant au carré la matrice pour F(k), on obtient la matrice pour F(2k) :

```
[ F(k+1)  F(k)  ]²   [ F(k+1)² + F(k)²        F(k+1)F(k) + F(k)F(k-1) ]
[               ]  = [                                                 ]
[ F(k)    F(k-1)]    [ F(k)F(k+1) + F(k-1)F(k)   F(k)² + F(k-1)²       ]
```

Ce qui correspond à :

```
[ F(2k+1)  F(2k)   ]
[                  ]
[ F(2k)    F(2k-1) ]
```

De cette égalité, on extrait les **identités de Fast Doubling** :

```
F(2k)   = F(k) × [2×F(k+1) - F(k)]
F(2k+1) = F(k+1)² + F(k)²
```

### Démonstration

1. **Pour F(2k)** :
   - De la matrice : F(2k) = F(k) × F(k+1) + F(k) × F(k-1)
   - Or F(k-1) = F(k+1) - F(k) (définition de Fibonacci)
   - Donc : F(2k) = F(k) × [F(k+1) + F(k+1) - F(k)]
   - **F(2k) = F(k) × [2×F(k+1) - F(k)]**

2. **Pour F(2k+1)** :
   - De la matrice : F(2k+1) = F(k+1)² + F(k)²
   - Cette formule découle directement de l'élément (1,1) de la matrice carrée

## Algorithme

### Pseudocode

```
FastDoubling(n):
    si n == 0:
        retourner (0, 1)  // (F(0), F(1))
    
    (a, b) = FastDoubling(n // 2)  // (F(k), F(k+1)) où k = n/2
    
    c = a × (2×b - a)      // F(2k)
    d = a² + b²            // F(2k+1)
    
    si n est pair:
        retourner (c, d)   // (F(n), F(n+1))
    sinon:
        retourner (d, c+d) // (F(n), F(n+1))
```

### Implémentation Go (Simplifiée)

```go
func FastDoublingSimple(n uint64) (*big.Int, *big.Int) {
    if n == 0 {
        return big.NewInt(0), big.NewInt(1)
    }
    
    a, b := FastDoublingSimple(n / 2)
    
    // c = a × (2b - a) = F(2k)
    c := new(big.Int).Lsh(b, 1)     // 2b
    c.Sub(c, a)                      // 2b - a
    c.Mul(c, a)                      // a × (2b - a)
    
    // d = a² + b² = F(2k+1)
    a2 := new(big.Int).Mul(a, a)
    b2 := new(big.Int).Mul(b, b)
    d := new(big.Int).Add(a2, b2)
    
    if n%2 == 0 {
        return c, d
    }
    return d, new(big.Int).Add(c, d)
}
```

## Optimisations Implémentées

### 1. Version Itérative

La version récursive est convertie en itérative pour éviter le coût des appels de fonction :

```go
func (fd *OptimizedFastDoubling) CalculateCore(...) (*big.Int, error) {
    numBits := bits.Len64(n)
    
    for i := numBits - 1; i >= 0; i-- {
        // Étape de doublement
        t2.Lsh(f_k1, 1).Sub(t2, f_k)       // t2 = 2×F(k+1) - F(k)
        
        t3 = smartMultiply(t3, f_k, t2)    // F(2k) = F(k) × t2
        t1 = smartMultiply(t1, f_k1, f_k1) // F(k+1)²
        t4 = smartMultiply(t4, f_k, f_k)   // F(k)²
        t2.Add(t1, t4)                      // F(2k+1) = F(k+1)² + F(k)²
        
        f_k, f_k1 = t3, t2
        
        // Étape d'addition (si bit = 1)
        if (n >> i) & 1 == 1 {
            t1.Add(f_k, f_k1)
            f_k, f_k1 = f_k1, t1
        }
    }
    
    return f_k, nil
}
```

### 2. Zero-Allocation avec sync.Pool

Les états de calcul sont recyclés :

```go
type calculationState struct {
    f_k, f_k1, t1, t2, t3, t4 *big.Int
}

var statePool = sync.Pool{
    New: func() interface{} {
        return &calculationState{
            f_k:  new(big.Int),
            f_k1: new(big.Int),
            // ...
        }
    },
}
```

### 3. Parallélisme des Multiplications

Les trois multiplications sont exécutées en parallèle sur multi-cœur :

```go
func parallelMultiply3Optimized(s *calculationState, fftThreshold int) {
    var wg sync.WaitGroup
    wg.Add(2)
    go func() { s.t3 = smartMultiply(s.t3, s.f_k, s.t2, fftThreshold); wg.Done() }()
    go func() { s.t1 = smartMultiply(s.t1, s.f_k1, s.f_k1, fftThreshold); wg.Done() }()
    s.t4 = smartMultiply(s.t4, s.f_k, s.f_k, fftThreshold)
    wg.Wait()
}
```

### 4. Multiplication Adaptative

Basculement automatique entre Karatsuba et FFT :

```go
func smartMultiply(z, x, y *big.Int, threshold int) *big.Int {
    if threshold > 0 && x.BitLen() > threshold && y.BitLen() > threshold {
        return bigfft.MulTo(z, x, y)  // FFT: O(n log n)
    }
    return z.Mul(x, y)  // Karatsuba: O(n^1.585)
}
```

## Analyse de Complexité

### Nombre d'Opérations

À chaque itération de la boucle principale :
- 1 décalage à gauche (O(n) bits)
- 1 soustraction (O(n) bits)
- 3 multiplications de grands entiers
- 1 addition (O(n) bits)
- Potentiellement 1 addition supplémentaire (si bit = 1)

Nombre d'itérations : log₂(n)

### Coût de Multiplication

Le coût de chaque multiplication dépend de la taille des opérandes :
- F(n) a environ n × log₂(φ) ≈ 0.694 × n bits
- Karatsuba : O(n^1.585)
- FFT : O(n log n)

### Complexité Totale

- **Avec Karatsuba** : O(log n × n^1.585)
- **Avec FFT** : O(log n × n log n)

## Comparaison avec les Autres Méthodes

| Méthode | Complexité | Multiplications/itération | Avantage |
|---------|------------|---------------------------|----------|
| Fast Doubling | O(log n × M(n)) | 3 | Le plus rapide |
| Matrix Exp. | O(log n × M(n)) | 4-8 | Plus intuitif |
| Récursion naive | O(φⁿ) | 0 | Simple mais impraticable |
| Itération | O(n) | 0 | Simple, lent pour grand n |

## Utilisation

```bash
# Calcul avec Fast Doubling
./fibcalc -n 1000000 -algo fast -d

# Avec parallélisme activé (défaut)
./fibcalc -n 10000000 -algo fast --threshold 4096

# Forcer le mode séquentiel
./fibcalc -n 1000000 -algo fast --threshold 0
```

## Références

1. Knuth, D. E. (1997). *The Art of Computer Programming, Volume 2: Seminumerical Algorithms*. Section 4.6.3.
2. [Fast Fibonacci algorithms](https://www.nayuki.io/page/fast-fibonacci-algorithms) - Nayuki
3. [Project Nayuki - Fast Doubling](https://www.nayuki.io/res/fast-fibonacci-algorithms/FastFibonacci.java)

