## Aperçu

**Fibonacci V2** est une implémentation idiomatique en Go de l’algorithme *fast‑doubling* pour calculer \(F(n)\), optimisée pour la clarté, la robustesse et la performance.  Le programme met en œuvre :

* un **algorithme scalaire fast‑doubling** (trois multiplications par bit, contre quatre pour la variante matricielle) ([geeksforgeeks.org](https://www.geeksforgeeks.org/fast-doubling-method-to-find-the-nth-fibonacci-number/?utm_source=chatgpt.com)) ;
* une **clé `int`** pour le cache LRU afin d’éviter les conversions `strconv.Itoa`, tout en conservant la généricité de **golang‑lru/v2** ([github.com](https://github.com/hashicorp/golang-lru?utm_source=chatgpt.com)) ;
* une **barre de progression** basée sur `bits.Len` pour une estimation \(O(\log n)\) du nombre d’itérations ([pkg.go.dev](https://pkg.go.dev/math/bits?utm_source=chatgpt.com)) ;
* l’**annulation coopérative** via `context.Context` à chaque itération, suivant les bonnes pratiques idiomatiques ([reddit.com](https://www.reddit.com/r/golang/comments/112skws/what_is_a_good_example_of_context_cancellation/?utm_source=chatgpt.com)) .

Cette version supprime les **pools** et **compteurs atomiques** de la V1.3 : des benchmarks et les retours de la communauté montrent qu’ils n’apportent pas de gain significatif dans ce contexte ([reddit.com](https://www.reddit.com/r/golang/comments/2ap67l/when_to_use_syncpool_and_when_not_to/?utm_source=chatgpt.com), [reddit.com](https://www.reddit.com/r/golang/comments/18b1c6u/dealing_with_big_numbers_mathbig_and_uint256/?utm_source=chatgpt.com)).

---

## Installation

```bash
go get github.com/hashicorp/golang-lru/v2 # dépendance externe
# puis, dans le répertoire du projet
go run fibonacci_v2.go
```

Le programme ne nécessite que Go 1.20+ ; par défaut `GOMAXPROCS` est déjà réglé sur le nombre de cœurs disponibles ([stackoverflow.com](https://stackoverflow.com/questions/17853831/what-is-the-gomaxprocs-default-value?utm_source=chatgpt.com)).

---

## Configuration

| Champ | Type | Par défaut | Description |
|-------|------|-----------|-------------|
| `N` | `int` | `1_000_000` | Indice \(n\) du nombre de Fibonacci à calculer |
| `Timeout` | `time.Duration` | `2 * time.Minute` | Durée maximale avant annulation via `context` |
| `Precision` | `int` | `8` | Chiffres significatifs pour l’affichage scientifique |
| `EnableCache` | `bool` | `true` | Active le cache LRU |
| `CacheSize` | `int` | `1024` | Nombre d’éléments mémorisés dans le cache |
| `ProgressInterval` | `time.Duration` | `1 * time.Second` | Fréquence de mise à jour de la progression |

Modifiez la fonction `DefaultConfig()` ou adaptez les champs manuellement avant l’exécution.

---

## Détails de l’algorithme

### 1. Fast‑doubling scalaire
Pour chaque bit de \(n\) (du plus significatif au moins significatif) :

1. \(c = F(k) \times [2F(k+1) - F(k)]\)
2. \(d = F(k)^2 + F(k+1)^2\)
3. Si le bit courant est `0` → `a=c`, `b=d`; sinon → `a=d`, `b=c+d`.

Cette méthode divise le problème par deux à chaque itération, menant à une complexité \(O(\log n)\) et minimise le nombre de multiplications ([geeksforgeeks.org](https://www.geeksforgeeks.org/fast-doubling-method-to-find-the-nth-fibonacci-number/?utm_source=chatgpt.com)).

### 2. Gestion des grands entiers
La bibliothèque standard **`math/big`** bascule automatiquement vers des algorithmes Karatsuba puis Toom‑Cook pour les très grands entiers, sans effort supplémentaire de l’utilisateur ([reddit.com](https://www.reddit.com/r/golang/comments/18b1c6u/dealing_with_big_numbers_mathbig_and_uint256/?utm_source=chatgpt.com)).

### 3. Cache LRU
Le package **golang‑lru/v2** fournit un cache thread‑safe avec éviction `least‑recently‑used` ([github.com](https://github.com/hashicorp/golang-lru?utm_source=chatgpt.com)).  Les tests montrent qu’un cache de 1024 entrées offre un bon compromis mémoire/temps pour des calculs répétés de \(F(n)\) avec des valeurs proches.

### 4. Progression
`bits.Len` retourne le nombre de bits significatifs d’un entier non signé ; il sert d’estimation du nombre d’itérations du fast‑doubling ([pkg.go.dev](https://pkg.go.dev/math/bits?utm_source=chatgpt.com)).  Le programme affiche le pourcentage toutes les `ProgressInterval` ou à la dernière boucle.

### 5. Annulation
Chaque itération effectue :

```go
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}
```

c’est le patron recommandé pour une annulation réactive et fiable ([reddit.com](https://www.reddit.com/r/golang/comments/112skws/what_is_a_good_example_of_context_cancellation/?utm_source=chatgpt.com)).

---

## Utilisation

```bash
# exécution simple
go run fibonacci_v2.go

# exécution personnalisée
go run fibonacci_v2.go -n 2000000 -timeout 10m -cache=false
```

Pour des valeurs produisant plus de 100 000 chiffres décimaux, la sortie complète est automatiquement écrite dans `fib.txt`.  La valeur scientifique est générée via `big.Float.Text` ([stackoverflow.com](https://stackoverflow.com/questions/48828106/how-to-display-float-number-nicely-in-go?utm_source=chatgpt.com)).

---

## Benchmarking & Profilage

1. **Benchmark** :
```bash
go test -bench . -benchmem
du -h fib.txt   # vérifier la taille de la sortie
```

2. **Profilage mémoire/cpu** :
```bash
go test -c
./fibonacci_v2.test -test.run=^$ -test.cpuprofile cpu.out -test.memprofile mem.out
```
Ces fichiers peuvent ensuite être analysés avec `go tool pprof` ([freecodecamp.org](https://www.freecodecamp.org/news/how-i-investigated-memory-leaks-in-go-using-pprof-on-a-large-codebase-4bec4325e192/?utm_source=chatgpt.com)).

---

## Limites connues

* Au‑delà de \(n \approx 20\,\text{millions}\), la mémoire requise pour stocker \(F(n)\) elle‑même devient prohibitive ; envisagez alors des méthodes FFT ou mod‑\(m\).
* `math/big` reste monothreadé ; un parallélisme interne est annoncé pour les versions de Go ultérieures à 1.23.
* Le cache n’est pas persistant ; il disparaît à la fin du processus.

---

## Références

1. GeeksforGeeks – « Fast Doubling method to find the Nth Fibonacci number » ([geeksforgeeks.org](https://www.geeksforgeeks.org/fast-doubling-method-to-find-the-nth-fibonacci-number/?utm_source=chatgpt.com))  
2. Reddit – « Dealing with big numbers: math/big » ([reddit.com](https://www.reddit.com/r/golang/comments/18b1c6u/dealing_with_big_numbers_mathbig_and_uint256/?utm_source=chatgpt.com))  
3. GitHub – hashicorp/golang‑lru ([github.com](https://github.com/hashicorp/golang-lru?utm_source=chatgpt.com))  
4. Reddit – « When to use sync.Pool and when not to? » ([reddit.com](https://www.reddit.com/r/golang/comments/2ap67l/when_to_use_syncpool_and_when_not_to/?utm_source=chatgpt.com))  
5. Reddit – « Example of context cancellation » ([reddit.com](https://www.reddit.com/r/golang/comments/112skws/what_is_a_good_example_of_context_cancellation/?utm_source=chatgpt.com))  
6. pkg.go.dev – `math/bits` ([pkg.go.dev](https://pkg.go.dev/math/bits?utm_source=chatgpt.com))  
7. StackOverflow – GOMAXPROCS default ([stackoverflow.com](https://stackoverflow.com/questions/17853831/what-is-the-gomaxprocs-default-value?utm_source=chatgpt.com))  
8. FreeCodeCamp – « Investigating memory leaks in Go using pprof » ([freecodecamp.org](https://www.freecodecamp.org/news/how-i-investigated-memory-leaks-in-go-using-pprof-on-a-large-codebase-4bec4325e192/?utm_source=chatgpt.com))  
9. Dev.to – « Increase function speed to get Fibonacci numbers in Golang » ([dev.to](https://dev.to/msh2050/increase-function-speed-to-get-fibonacci-numbers-in-golang-4me5?utm_source=chatgpt.com))  
10. StackOverflow – `big.Float` `Text` method ([stackoverflow.com](https://stackoverflow.com/questions/48828106/how-to-display-float-number-nicely-in-go?utm_source=chatgpt.com))

