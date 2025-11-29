# Exponentiation Matricielle

> **Complexité** : O(log n) opérations matricielles  
> **Complexité réelle** : O(log n × M(n)) où M(n) est le coût de multiplication

## Introduction

L'**exponentiation matricielle** est une méthode élégante pour calculer les nombres de Fibonacci basée sur la représentation matricielle de la suite. Cette approche exploite l'exponentiation rapide (squaring) pour réduire le nombre d'opérations à O(log n).

## Fondement Mathématique

### Matrice Q de Fibonacci

La suite de Fibonacci satisfait la relation matricielle :

```
[ F(n+1) ]   [ 1  1 ]   [ F(n)   ]
[        ] = [      ] × [        ]
[ F(n)   ]   [ 1  0 ]   [ F(n-1) ]
```

En appliquant cette relation n fois depuis les conditions initiales F(1) = 1, F(0) = 0 :

```
[ F(n+1)  F(n)   ]   [ 1  1 ]^n
[                ] = [      ]
[ F(n)    F(n-1) ]   [ 1  0 ]
```

La matrice `Q = [[1,1], [1,0]]` est appelée **matrice Q de Fibonacci**.

### Propriétés de Q

1. **Déterminant** : det(Q^n) = (-1)^n
2. **Symétrie** : Q^n est toujours une matrice symétrique (Q^n[0][1] = Q^n[1][0])
3. **Relation de Cassini** : F(n+1)×F(n-1) - F(n)² = (-1)^n

## Algorithme

### Exponentiation Rapide (Binary Exponentiation)

L'idée clé est d'utiliser la décomposition binaire de l'exposant :

```
n = Σ bᵢ × 2^i  (où bᵢ ∈ {0, 1})
```

Alors :
```
Q^n = Q^(Σ bᵢ × 2^i) = Π Q^(bᵢ × 2^i)
```

### Pseudocode

```
MatrixFibonacci(n):
    si n == 0:
        retourner 0
    
    result = matrice identité I
    base = Q = [[1,1], [1,0]]
    
    exposant = n - 1
    
    tant que exposant > 0:
        si exposant est impair:
            result = result × base
        base = base × base  // Élévation au carré
        exposant = exposant / 2
    
    retourner result[0][0]  // C'est F(n)
```

### Implémentation Go

```go
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, 
    n uint64, opts Options) (*big.Int, error) {
    
    if n == 0 {
        return big.NewInt(0), nil
    }
    
    state := acquireMatrixState()
    defer releaseMatrixState(state)
    
    exponent := n - 1
    numBits := bits.Len64(exponent)
    
    // state.res = matrice identité
    // state.p = matrice Q = [[1,1],[1,0]]
    
    for i := 0; i < numBits; i++ {
        if (exponent >> i) & 1 == 1 {
            multiplyMatrices(state.tempMatrix, state.res, state.p, state, ...)
            state.res, state.tempMatrix = state.tempMatrix, state.res
        }
        
        if i < numBits - 1 {
            squareSymmetricMatrix(state.tempMatrix, state.p, state, ...)
            state.p, state.tempMatrix = state.tempMatrix, state.p
        }
    }
    
    return new(big.Int).Set(state.res.a), nil
}
```

## Optimisations Implémentées

### 1. Algorithme de Strassen

Pour les matrices 2×2 avec de grands éléments, l'algorithme de Strassen réduit le nombre de multiplications de 8 à 7 :

```
Multiplication classique 2×2:
  C[0][0] = A[0][0]×B[0][0] + A[0][1]×B[1][0]  (2 mult)
  C[0][1] = A[0][0]×B[0][1] + A[0][1]×B[1][1]  (2 mult)
  C[1][0] = A[1][0]×B[0][0] + A[1][1]×B[1][0]  (2 mult)
  C[1][1] = A[1][0]×B[0][1] + A[1][1]×B[1][1]  (2 mult)
  Total: 8 multiplications

Strassen 2×2:
  P1 = A[0][0] × (B[0][1] - B[1][1])
  P2 = (A[0][0] + A[0][1]) × B[1][1]
  P3 = (A[1][0] + A[1][1]) × B[0][0]
  P4 = A[1][1] × (B[1][0] - B[0][0])
  P5 = (A[0][0] + A[1][1]) × (B[0][0] + B[1][1])
  P6 = (A[0][1] - A[1][1]) × (B[1][0] + B[1][1])
  P7 = (A[0][0] - A[1][0]) × (B[0][0] + B[0][1])
  
  C[0][0] = P5 + P4 - P2 + P6
  C[0][1] = P1 + P2
  C[1][0] = P3 + P4
  C[1][1] = P5 + P1 - P3 - P7
  Total: 7 multiplications + 18 additions
```

L'implémentation bascule vers Strassen quand les éléments dépassent `--strassen-threshold` (défaut: 3072 bits).

### 2. Élévation au Carré de Matrices Symétriques

Pour une matrice symétrique (b = c), le carré peut être calculé avec seulement 4 multiplications :

```
[ a  b ]²   [ a²+b²    b(a+d) ]
[      ] = [                  ]
[ b  d ]    [ b(a+d)   b²+d²  ]
```

```go
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, 
    inParallel bool, fftThreshold int) {
    
    ad := new(big.Int).Add(mat.a, mat.d)  // a + d
    
    a2 = smartMultiply(a2, mat.a, mat.a)  // a²
    b2 = smartMultiply(b2, mat.b, mat.b)  // b²
    d2 = smartMultiply(d2, mat.d, mat.d)  // d²
    b_ad = smartMultiply(b_ad, mat.b, ad) // b(a+d)
    
    dest.a.Add(a2, b2)    // a² + b²
    dest.b.Set(b_ad)      // b(a+d)
    dest.c.Set(b_ad)      // symétrique
    dest.d.Add(b2, d2)    // b² + d²
}
```

### 3. Zero-Allocation avec sync.Pool

```go
type matrixState struct {
    res, p, tempMatrix *matrix
    // Temporaires pour Strassen
    p1, p2, p3, p4, p5, p6, p7 *big.Int
    s1, s2, s3, s4, s5, s6, s7, s8, s9, s10 *big.Int
    // Temporaires pour carré symétrique
    t1, t2, t3, t4, t5 *big.Int
}

var matrixStatePool = sync.Pool{
    New: func() interface{} {
        return &matrixState{
            res: newMatrix(),
            p: newMatrix(),
            // ...
        }
    },
}
```

### 4. Parallélisme

Les multiplications indépendantes sont parallélisées :

```go
if inParallel {
    var wg sync.WaitGroup
    wg.Add(7)  // Strassen: 7 multiplications parallèles
    go func() { p1 = smartMultiply(p1, m1.a, s1); wg.Done() }()
    go func() { p2 = smartMultiply(p2, s2, m2.d); wg.Done() }()
    // ...
    wg.Wait()
}
```

## Analyse de Complexité

### Opérations par Itération

| Opération | Classique | Strassen | Carré Symétrique |
|-----------|-----------|----------|------------------|
| Multiplications | 8 | 7 | 4 |
| Additions | 4 | 18 | 4 |

### Nombre d'Itérations

- log₂(n) itérations
- À chaque itération : 1 élévation au carré + potentiellement 1 multiplication

### Complexité Totale

- **Avec Karatsuba** : O(log n × n^1.585)
- **Avec FFT** : O(log n × n log n)

## Comparaison avec Fast Doubling

| Critère | Matrix Exp. | Fast Doubling |
|---------|-------------|---------------|
| Multiplications/iter (base) | 8 | 3 |
| Multiplications/iter (optimisé) | 4-7 | 3 |
| Complexité mathématique | Plus intuitive | Plus compacte |
| Performance pratique | Plus lent | Plus rapide |

## Utilisation

```bash
# Calcul avec Matrix Exponentiation
./fibcalc -n 1000000 -algo matrix -d

# Ajuster le seuil Strassen
./fibcalc -n 10000000 -algo matrix --strassen-threshold 2048

# Désactiver Strassen (multiplication classique uniquement)
./fibcalc -n 1000000 -algo matrix --strassen-threshold 999999999
```

## Références

1. Erickson, J. (2019). *Algorithms*. Chapter on Recursion and Backtracking.
2. Cormen, T. H. et al. (2009). *Introduction to Algorithms*. Section 31.2: Matrix Exponentiation.
3. Strassen, V. (1969). "Gaussian Elimination is not Optimal". *Numerische Mathematik*.

