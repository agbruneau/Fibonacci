# Fast Doubling Algorithm

> **Complexity**: O(log n) arithmetic operations  
> **Actual Complexity**: O(log n × M(n)) where M(n) is the multiplication cost

## Introduction

The **Fast Doubling** algorithm is one of the most efficient methods for calculating Fibonacci numbers. It exploits the mathematical properties of the sequence to reduce the number of operations to O(log n).

## Mathematical Foundation

### Matrix Form of Fibonacci

The Fibonacci sequence can be expressed in matrix form:

```
[ F(n+1)  F(n)   ]   [ 1  1 ]^n
[                ] = [      ]
[ F(n)    F(n-1) ]   [ 1  0 ]
```

This relation is known as the **Fibonacci Q matrix**.

### Derivation of Doubling Formulae

By squaring the matrix for F(k), we obtain the matrix for F(2k):

```
[ F(k+1)  F(k)  ]²   [ F(k+1)² + F(k)²        F(k+1)F(k) + F(k)F(k-1) ]
[               ]  = [                                                 ]
[ F(k)    F(k-1)]    [ F(k)F(k+1) + F(k-1)F(k)   F(k)² + F(k-1)²       ]
```

Which corresponds to:

```
[ F(2k+1)  F(2k)   ]
[                  ]
[ F(2k)    F(2k-1) ]
```

From this equality, we extract the **Fast Doubling identities**:

```
F(2k)   = F(k) × [2×F(k+1) - F(k)]
F(2k+1) = F(k+1)² + F(k)²
```

### Proof

1. **For F(2k)**:
   - From the matrix: F(2k) = F(k) × F(k+1) + F(k) × F(k-1)
   - Now F(k-1) = F(k+1) - F(k) (Fibonacci definition)
   - Therefore: F(2k) = F(k) × [F(k+1) + F(k+1) - F(k)]
   - **F(2k) = F(k) × [2×F(k+1) - F(k)]**

2. **For F(2k+1)**:
   - From the matrix: F(2k+1) = F(k+1)² + F(k)²
   - This formula follows directly from element (1,1) of the squared matrix

## Algorithm

### Pseudocode

```
FastDoubling(n):
    if n == 0:
        return (0, 1)  // (F(0), F(1))
    
    (a, b) = FastDoubling(n // 2)  // (F(k), F(k+1)) where k = n/2
    
    c = a × (2×b - a)      // F(2k)
    d = a² + b²            // F(2k+1)
    
    if n is even:
        return (c, d)   // (F(n), F(n+1))
    else:
        return (d, c+d) // (F(n), F(n+1))
```

### Go Implementation (Simplified)

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

## Implemented Optimisations

### 1. Iterative Version

The recursive version is converted to iterative to avoid function call overhead:

```go
func (fd *OptimizedFastDoubling) CalculateCore(...) (*big.Int, error) {
    numBits := bits.Len64(n)
    
    for i := numBits - 1; i >= 0; i-- {
        // Doubling step
        t2.Lsh(f_k1, 1).Sub(t2, f_k)       // t2 = 2×F(k+1) - F(k)
        
        t3 = smartMultiply(t3, f_k, t2)    // F(2k) = F(k) × t2
        t1 = smartMultiply(t1, f_k1, f_k1) // F(k+1)²
        t4 = smartMultiply(t4, f_k, f_k)   // F(k)²
        t2.Add(t1, t4)                      // F(2k+1) = F(k+1)² + F(k)²
        
        f_k, f_k1 = t3, t2
        
        // Addition step (if bit = 1)
        if (n >> i) & 1 == 1 {
            t1.Add(f_k, f_k1)
            f_k, f_k1 = f_k1, t1
        }
    }
    
    return f_k, nil
}
```

### 2. Zero-Allocation with sync.Pool

Calculation states are recycled:

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

### 3. Multiplication Parallelism

The three multiplications are executed in parallel on multi-core:

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

### 4. Adaptive Multiplication

Automatic switching between Karatsuba and FFT:

```go
func smartMultiply(z, x, y *big.Int, threshold int) *big.Int {
    if threshold > 0 && x.BitLen() > threshold && y.BitLen() > threshold {
        return bigfft.MulTo(z, x, y)  // FFT: O(n log n)
    }
    return z.Mul(x, y)  // Karatsuba: O(n^1.585)
}
```

## Complexity Analysis

### Number of Operations

At each iteration of the main loop:
- 1 left shift (O(n) bits)
- 1 subtraction (O(n) bits)
- 3 large integer multiplications
- 1 addition (O(n) bits)
- Potentially 1 additional addition (if bit = 1)

Number of iterations: log₂(n)

### Multiplication Cost

The cost of each multiplication depends on the operand size:
- F(n) has approximately n × log₂(φ) ≈ 0.694 × n bits
- Karatsuba: O(n^1.585)
- FFT: O(n log n)

### Total Complexity

- **With Karatsuba**: O(log n × n^1.585)
- **With FFT**: O(log n × n log n)

## Comparison with Other Methods

| Method | Complexity | Multiplications/iteration | Advantage |
|--------|------------|---------------------------|-----------|
| Fast Doubling | O(log n × M(n)) | 3 | Fastest |
| Matrix Exp. | O(log n × M(n)) | 4-8 | More intuitive |
| Naive recursion | O(φⁿ) | 0 | Simple but impractical |
| Iteration | O(n) | 0 | Simple, slow for large n |

## Usage

```bash
# Calculation with Fast Doubling
./fibcalc -n 1000000 -algo fast -d

# With parallelism enabled (default)
./fibcalc -n 10000000 -algo fast --threshold 4096

# Force sequential mode
./fibcalc -n 1000000 -algo fast --threshold 0
```

## References

1. Knuth, D. E. (1997). *The Art of Computer Programming, Volume 2: Seminumerical Algorithms*. Section 4.6.3.
2. [Fast Fibonacci algorithms](https://www.nayuki.io/page/fast-fibonacci-algorithms) - Nayuki
3. [Project Nayuki - Fast Doubling](https://www.nayuki.io/res/fast-fibonacci-algorithms/FastFibonacci.java)
