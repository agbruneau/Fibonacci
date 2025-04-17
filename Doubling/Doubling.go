// fibonacci_v2.go
// Version 2.0 – Implémentation scalaire fast‑doubling optimisée
// Auteur : André‑Guy Bruneau (refonte 2025‑04‑17)
//
// Description :
//   - Algorithme fast‑doubling itératif sans matrice (≈ 2× moins d’opérations)
//   - Cache LRU optionnel avec clé int (évite la conversion strconv.Itoa)
//   - Barre de progression et annulation via context.Context
//   - Conception minimaliste ; le profilage peut être ajouté via go test/pprof
package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/bits"
	"os"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type Config struct {
	N                int
	Timeout          time.Duration
	Precision        int
	EnableCache      bool
	CacheSize        int
	ProgressInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		N:                1_000_000,       // Valeur par défaut ; ajustez selon vos besoins
		Timeout:          2 * time.Minute, // Durée maximale d’exécution
		Precision:        8,               // Chiffres significatifs pour l’affichage scientifique
		EnableCache:      true,            // Active le cache LRU
		CacheSize:        1024,            // Taille maxi du cache
		ProgressInterval: 1 * time.Second, // Fréquence de mise à jour de la progression
	}
}

type FibCalculator struct {
	cache *lru.Cache[int, *big.Int]
	cfg   Config
}

func NewFibCalculator(cfg Config) *FibCalculator {
	var c *lru.Cache[int, *big.Int]
	if cfg.EnableCache {
		cache, err := lru.New[int, *big.Int](cfg.CacheSize)
		if err != nil {
			log.Fatalf("impossible de créer le cache LRU : %v", err)
		}
		cache.Add(0, big.NewInt(0))
		cache.Add(1, big.NewInt(1))
		c = cache
	}
	return &FibCalculator{cache: c, cfg: cfg}
}

// fastDoubling calcule F(n) avec l’algorithme fast‑doubling itératif.
// Il vérifie ctx.Done() à chaque itération et renvoie ctx.Err() si nécessaire.
func (fc *FibCalculator) fastDoubling(ctx context.Context, n int) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}
	a := big.NewInt(0) // F(0)
	b := big.NewInt(1) // F(1)

	totalBits := bits.Len(uint(n))
	lastReport := time.Now()

	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// c = a * (2*b − a)
		twoB := new(big.Int).Lsh(b, 1)
		twoB.Sub(twoB, a)
		c := new(big.Int).Mul(a, twoB)

		// d = a² + b²
		aSq := new(big.Int).Mul(a, a)
		bSq := new(big.Int).Mul(b, b)
		d := new(big.Int).Add(aSq, bSq)

		if ((n >> i) & 1) == 0 {
			a, b = c, d
		} else {
			a, b = d, new(big.Int).Add(c, d)
		}

		if time.Since(lastReport) >= fc.cfg.ProgressInterval || i == 0 {
			progress := float64(totalBits-i) / float64(totalBits) * 100
			fmt.Printf("\rProgression : %.2f%% ", progress)
			lastReport = time.Now()
		}
	}
	fmt.Println()
	return a, nil
}

func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être ≥ 0, reçu %d", n)
	}
	if fc.cfg.EnableCache && fc.cache != nil {
		if v, ok := fc.cache.Get(n); ok {
			return new(big.Int).Set(v), nil
		}
	}
	res, err := fc.fastDoubling(ctx, n)
	if err != nil {
		return nil, err
	}
	if fc.cfg.EnableCache && fc.cache != nil {
		fc.cache.Add(n, new(big.Int).Set(res))
	}
	return res, nil
}

func fmtScientific(x *big.Int, precision int) string {
	if x.Sign() == 0 {
		return "0.0e+0"
	}
	f := new(big.Float).SetInt(x)
	return f.Text('e', precision)
}

func main() {
	cfg := DefaultConfig()
	log.Printf("Calcul de F(%d)…", cfg.N)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	fc := NewFibCalculator(cfg)
	start := time.Now()
	res, err := fc.Calculate(ctx, cfg.N)
	if err != nil {
		log.Fatalf("échec du calcul : %v", err)
	}
	elapsed := time.Since(start)

	fmt.Printf("Durée totale : %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("F(%d) ≈ %s (scientifique)\n", cfg.N, fmtScientific(res, cfg.Precision))

	full := res.Text(10)
	if len(full) > 100_000 {
		if err := os.WriteFile("fib.txt", []byte(full), 0o644); err == nil {
			fmt.Println("Valeur complète enregistrée dans fib.txt")
		}
	}
}
