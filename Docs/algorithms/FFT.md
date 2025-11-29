# Multiplication FFT pour Grands Entiers

> **Complexité** : O(n log n) pour la multiplication de deux nombres de n bits  
> **Utilisée par** : Fast Doubling et Matrix Exp. pour très grands nombres

## Introduction

La **Transformée de Fourier Rapide (FFT)** permet de multiplier deux grands entiers en O(n log n) au lieu de O(n²) pour la multiplication naïve ou O(n^1.585) pour Karatsuba. Cette optimisation devient cruciale pour les nombres dépassant environ 1 million de bits.

## Principe Mathématique

### Convolution et Multiplication

La multiplication de deux entiers peut être vue comme une **convolution** de leurs chiffres :

```
A = Σᵢ aᵢ × B^i
B = Σⱼ bⱼ × B^j

A × B = Σₖ cₖ × B^k  où  cₖ = Σᵢ aᵢ × b(k-i)
```

Le terme cₖ est la **convolution discrète** des séquences {aᵢ} et {bⱼ}.

### Théorème de Convolution

Le théorème de convolution stipule que :

```
DFT(a * b) = DFT(a) × DFT(b)  (multiplication point par point)
```

Où `*` est la convolution et DFT est la Transformée de Fourier Discrète.

Donc :
```
a * b = IDFT(DFT(a) × DFT(b))
```

### Algorithme de Multiplication FFT

1. **Padding** : Étendre les nombres à une longueur puissance de 2
2. **DFT** : Calculer la FFT des deux séquences de chiffres
3. **Multiplication** : Multiplier point par point dans le domaine fréquentiel
4. **IDFT** : Calculer la FFT inverse
5. **Propagation** : Gérer les retenues

## Implémentation dans FibCalc

### Module `internal/bigfft`

Le module bigfft implémente une multiplication FFT spécialisée pour `big.Int` :

```go
// mulFFT effectue x × y via FFT
func mulFFT(x, y *big.Int) *big.Int {
    return bigfft.Mul(x, y)
}

// smartMultiply choisit la méthode optimale
func smartMultiply(z, x, y *big.Int, threshold int) *big.Int {
    if threshold > 0 {
        bx := x.BitLen()
        by := y.BitLen()
        if bx > threshold && by > threshold {
            return bigfft.MulTo(z, x, y)  // FFT
        }
    }
    return z.Mul(x, y)  // Karatsuba (math/big)
}
```

### Structure du Code

```
internal/bigfft/
├── fft.go      # Algorithme FFT principal
├── fermat.go   # Arithmétique modulaire pour FFT
├── scan.go     # Conversion entre big.Int et représentation FFT
├── pool.go     # Pools d'objets pour performance
└── arith_decl.go  # Déclarations d'arithmétique bas niveau
```

### FFT de Fermat

L'implémentation utilise une **FFT de Fermat** qui opère dans l'anneau Z/(2^k + 1) :

- Les racines de l'unité sont des puissances de 2
- Les multiplications deviennent des décalages de bits
- Plus efficace que FFT à nombres complexes pour les entiers

## Seuil d'Activation

### Configuration

```bash
# Seuil par défaut : 1,000,000 bits
./fibcalc -n 100000000 --fft-threshold 1000000

# Forcer FFT plus tôt (nombres > 500,000 bits)
./fibcalc -n 100000000 --fft-threshold 500000

# Désactiver FFT
./fibcalc -n 100000000 --fft-threshold 0
```

### Choix du Seuil

Le seuil optimal dépend de plusieurs facteurs :

| Facteur | Impact |
|---------|--------|
| Taille du cache L3 | Cache plus grand → seuil plus bas |
| Fréquence CPU | Plus rapide → seuil légèrement plus haut |
| Nombre de cœurs | Plus de cœurs → FFT moins avantageux (car saturant) |

Pour déterminer le seuil optimal :

```bash
./fibcalc --calibrate
```

## Interaction avec le Parallélisme

### Problème de Contention

L'algorithme FFT a tendance à **saturer les ressources CPU** car il effectue beaucoup d'opérations mémoire parallèles en interne. Exécuter plusieurs multiplications FFT en parallèle cause de la contention.

### Solution Implémentée

```go
// Désactiver le parallélisme externe quand FFT est utilisé
// sauf pour de très très grands nombres
if opts.FFTThreshold > 0 {
    minBitLen := s.f_k.BitLen()
    if minBitLen > opts.FFTThreshold {
        // FFT va être utilisé - désactiver parallélisme
        // sauf si nombres > ParallelFFTThreshold (10M bits)
        return minBitLen > ParallelFFTThreshold
    }
}
```

## Calculateur FFT-Based

Le calculateur `fft` force l'utilisation de FFT pour toutes les multiplications :

```go
type FFTBasedCalculator struct{}

func (c *FFTBasedCalculator) Name() string {
    return "FFT-Based Doubling (O(log n), FFT Mul)"
}

func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter,
    n uint64, opts Options) (*big.Int, error) {
    
    // Force FFT en mettant threshold très bas
    opts.FFTThreshold = 1
    
    // Utilise le même algorithme que Fast Doubling
    fd := &OptimizedFastDoubling{}
    return fd.CalculateCore(ctx, reporter, n, opts)
}
```

Ce calculateur est principalement utilisé pour :
- Benchmarking de la performance FFT
- Tests de régression
- Comparaison des algorithmes de multiplication

## Analyse de Complexité

### Multiplication de deux nombres de n bits

| Algorithme | Complexité | Constante cachée |
|------------|------------|------------------|
| Naïf | O(n²) | Faible |
| Karatsuba | O(n^1.585) | Moyenne |
| Toom-Cook 3 | O(n^1.465) | Élevée |
| FFT | O(n log n) | Très élevée |

### Point de Croisement

```
                    │
    Temps           │     /
     de             │    /  ← Karatsuba O(n^1.585)
   calcul           │   /
                    │  /
                    │ /          ← FFT O(n log n)
                    │/     _______
                    └─────────────────────
                          ~1M bits      Taille (bits)
```

### Overhead FFT

L'overhead de FFT provient de :
1. Conversion big.Int → représentation FFT
2. Padding à la puissance de 2 suivante
3. FFT aller et retour
4. Propagation des retenues

## Utilisation

```bash
# Forcer l'utilisation de FFT pour toutes les multiplications
./fibcalc -n 10000000 -algo fft -d

# Ajuster le seuil FFT pour Fast Doubling
./fibcalc -n 100000000 -algo fast --fft-threshold 800000

# Benchmark comparatif
./fibcalc -n 50000000 -algo all -d
```

## Références

1. Cooley, J. W., & Tukey, J. W. (1965). "An algorithm for the machine calculation of complex Fourier series". *Mathematics of Computation*.
2. Schönhage, A., & Strassen, V. (1971). "Schnelle Multiplikation großer Zahlen". *Computing*.
3. [GMP Library - FFT Multiplication](https://gmplib.org/manual/FFT-Multiplication)
4. [Go bigfft package documentation](https://pkg.go.dev/github.com/ncw/gmp)

