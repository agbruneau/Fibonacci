# 41 — Audit internal/bigfft & arènes

Date : 2026-04-28 — Périmètre : `internal/bigfft/` (33 fichiers `.go`, 7 611 LOC totales,
dont 3 327 LOC prod / 4 284 LOC tests).

## Cartographie

| Fichier | Rôle | LOC |
| --- | --- | --: |
| `doc.go` | Package doc (rôle, invariants, exemple) | 31 |
| `fft.go` | Entry points publics `Mul`/`MulTo`/`Sqr`/`SqrTo` (+ `recover`) | 226 |
| `fft_core.go` | Cœur FFT (orchestration mulFFT/sqrFFT) | 109 |
| `fft_poly.go` | Représentation polynomiale, transform/inverse | 533 |
| `fft_recursion.go` | Sémaphore concurrence FFT (NumCPU), parallélisme borné | 169 |
| `fft_cache.go` | Cache LRU thread-safe des plans FFT | 534 |
| `fermat.go` | Arithmétique anneau Fermat 2^n+1 | 309 |
| `pool.go` | `sync.Pool` (word slices, fermats, FFTState) | 467 |
| `pool_warming.go` | Pré-chauffage des pools | 120 |
| `bump.go` | Allocateur bump (arène) | 242 |
| `allocator.go` | Interface `TempAllocator` (Pool + Bump adapter) | 109 |
| `memory_est.go` | Estimation mémoire avant calcul | 78 |
| `scan.go` | Parsing décimal subquadratique | 88 |
| `arith_decl.go` | `go:linkname` vers `math/big` (addVV, subVV, …) | 65 |
| `arith_amd64.go` | Wrappers amd64 | 32 |
| `arith_generic.go` | Fallback `!amd64` | 36 |
| `cpu_amd64.go` | Détection AVX2/AVX-512/BMI2/ADX | 168 |

## Allocateur bump

- **Localisation** : `bump.go` (struct `BumpAllocator`), interface dans `allocator.go`
  (`TempAllocator`, adapter `BumpAllocatorAdapter`).
- **Implémentation** : `[]big.Word` + `offset int`. `Alloc(n)` = slicing + `clear` (Go 1.21).
  `AllocUnsafe` saute le zeroing. `AllocFermat`/`AllocFermatSlice` pour les coefficients FFT.
- **Pooling** : `bumpAllocatorPool sync.Pool` ; `Acquire/Release` réutilisent le buffer
  sous-jacent (resize uniquement si capacité insuffisante).
- **Invariants vérifiés** : (1) O(1) confirmé (slice + bump pointer, pas de mutex après
  acquisition) ; (2) zéro fragmentation (buffer contigu) ; (3) `Reset()` et `Release` remettent
  `offset = 0` mais conservent le buffer ; (4) **fallback `make()`** quand
  `offset+n > len(buffer)` → préserve la correction même si `EstimateBumpCapacity` sous-estime.
- **Tests dédiés** : `bump_test.go` — `TestBumpAllocatorAlloc`, `Fallback`, `Reset`,
  `AllocFermat`, `AllocFermatSlice` (vérifie non-recouvrement), `AllocUnsafe`,
  `EstimateBumpCapacity` ; benchmarks `AllocVsMake`, `FFTMulWithBump`, `FFTSqrWithBump`,
  `BumpAllocatorReuse`.
- **Risques** : (a) **Aliasing après `Reset`/`Release`** non détecté à l'exécution — un caller
  qui conserve un slice est silencieusement écrasé par la prochaine `Alloc`. La doc
  l'avertit (« should not be used »), mais aucun garde-fou (générations, marker). (b) Le
  fallback silencieux à `make()` masque les sous-estimations de capacité : pas de métrique
  exposée pour mesurer le taux de fallback en prod.

## Intrinsics amd64 vs fallback

| API | amd64 (`arith_amd64.go`) | !amd64 (`arith_generic.go`) | Parité |
| --- | --- | --- | --- |
| `AddVV` | wrapper `addVV` (linkname) | wrapper `addVV` (linkname) | OK |
| `SubVV` | wrapper `subVV` | wrapper `subVV` | OK |
| `AddMulVVW` | wrapper `addMulVVW` | wrapper `addMulVVW` | OK |

Note : les deux variantes appellent les **mêmes** symboles `go:linkname` déclarés dans
`arith_decl.go` (compilé sur toutes les archis). La séparation `_amd64.go` / `_generic.go` est
préparatoire mais aucun assembleur custom n'est présent — la détection AVX2/AVX-512 dans
`cpu_amd64.go` est utilisée par `fft_poly.go` (scan SIMD) mais **pas** dans les wrappers
arith. Surface API strictement identique → parité.

`cpu_amd64.go` n'a pas de pendant `cpu_generic.go` ; cohérent puisque ces variables
(`hasAVX2`, etc.) sont uniquement consommées par du code sous build tag amd64.

## recover() / panic

| Site | Justification | Wrapping |
| --- | --- | --- |
| `fft.go:42` `Mul` | Convertit panic interne en erreur applicative | `fmt.Errorf("panic in bigfft.Mul: %v\nStack: %s", r, debug.Stack())` |
| `fft.go:58` `MulTo` | idem | idem (verbe `%v`, pas `%w`) |
| `fft.go:85` `Sqr` | idem | idem |
| `fft.go:98` `SqrTo` | idem | idem |

**Pertinent** : ces sites sont les frontières publiques d'un package dont les `panic`
internes (assertions sur tailles fermat, divisions modulaires) seraient catastrophiques pour
les callers. **Bémol** : le formatage `%v` rebranché par audit 3.1 perd la chaîne d'erreur
(pas de `errors.Is/As`) et inclut systématiquement la stack — verbeux. Cf. finding F1.

## unsafe.Pointer

- Prod : **0** occurrence de `unsafe.Pointer`.
- Test : **0** occurrence.
- Usages indirects : `unsafe.Sizeof(big.Word(0))` dans `fft.go:13` (constante `_W`, calcul
  taille de mot — usage idiomatique sans risque) et `_ "unsafe"` dans `arith_decl.go` (requis
  par `go:linkname`).

→ Confirme l'audit 4.1 sur la prod ; corrige son chiffre « 2 en test » (réalité : 0).

## Tests

- **Fichiers `_test.go`** : 14 (`bump_test.go`, `pool_test.go`, `pool_extra_test.go`,
  `pool_warming_bench_test.go`, `fermat_test.go`, `fft_cache_test.go`, `fft_extra_test.go`,
  `fft_parallel_test.go`, `fft_poly_test.go`, `fft_precision_test.go`, `sqr_test.go`,
  `arith_amd64_test.go`, `cpu_amd64_extended_test.go`, `memory_est_test.go`, `scan_test.go`).
- **Benchmarks** : 36 `BenchmarkXxx` dans le package. Couvrent : bump vs make, FFT mul/sqr
  avec/sans bump, pool warming on/off, cache hit/miss/put, Mul small/medium/large,
  Sqr vs Mul, parallélisation FFT, AddVV/SubVV/AddMulVVW. **Manquant : pas de benchmark
  comparatif explicite `bigfft.Mul` vs `(*big.Int).Mul`** — la décision algorithmique
  (seuil 1800 mots) n'a pas de garde-fou de régression dans ce package (cf. F2).
- **Fuzz** : **0** fonction `FuzzXxx`. Cible naturelle : `Mul(x, y) == new(big.Int).Mul(x,y)`
  pour des entrées arbitraires au-dessus du seuil (cf. F3).

## Documentation interne

- `doc.go` : rôle, invariants, exemple, pointeur vers `docs/algorithms/FFT.md`.
- `docs/algorithms/BIGFFT.md` (701 lignes) : couvre Public API, Internal Data Flow, FFT Size
  Selection, Polynomial Representation, Fermat Ring, Transform, **Memory Management** (incl.
  bump allocator), FFT Cache, CPU Feature Detection, Subquadratic String Parsing, Memory
  Architecture Diagram. Très complet.
- `docs/algorithms/FFT.md` (221 lignes) : Mathematical Principle, Threshold, Complexity
  Analysis (O(N log N log log N)), Usage. Schönhage-Strassen explicitement traité.
- Commentaires `bump.go` détaillés (avantages vs `sync.Pool`, contrat d'usage avec `defer`).

→ Documentation interne **suffisante**, voire au-delà du standard du codebase.

## Findings

| # | Description | Sévérité |
| --- | --- | --- |
| F1 | Les 4 `recover` utilisent `%v` (pas `%w`) → la cause racine ne peut pas être inspectée via `errors.Is/As`. Suggérer un type d'erreur structuré (déjà dispo dans `internal/errors`) ou `%w` sur un `fmt.Errorf` enveloppant `errors.New(string)`. | Mineur |
| F2 | Aucun benchmark comparatif `bigfft.Mul` vs `(*big.Int).Mul` au sein du package → impossible de détecter une régression du seuil 1 800 mots ou un ralentissement absolu sans bench applicatif externe. | Mineur |
| F3 | Pas de cible `Fuzz` (correctness vs `math/big.Mul` est un fuzz idéal : oracle gratuit). | Mineur |
| F4 | `BumpAllocator.Reset/Release` n'invalide pas les slices précédemment retournés ; aucun mécanisme de génération/marker. Risque d'aliasing silencieux si un caller conserve un slice par mégarde. Tests ne couvrent pas ce scénario. | Mineur (doc le mentionne) |
| F5 | Le fallback silencieux `make()` dans `Alloc` quand `EstimateBumpCapacity` sous-estime n'expose aucune métrique → impossible de mesurer la qualité de l'estimation en prod. | Trivial |
| F6 | `arith_amd64.go` et `arith_generic.go` ont un corps quasi identique (les deux délèguent aux symboles `go:linkname`). La séparation par build tag est aujourd'hui un placeholder ; à fusionner ou justifier en commentaire. | Trivial |
| F7 | `audit 4.1` annonçait « 2 occurrences `unsafe.Pointer` en test » → réalité 0 ; corriger le rapport amont. | Info |

Aucun finding bloquant ou de sévérité majeure : le sous-système est mature, bien testé,
bien documenté.

## Synthèse

`internal/bigfft` est l'un des packages les plus disciplinés du dépôt. Les invariants
revendiqués (bump O(1), zéro fragmentation, parité amd64/fallback) sont **vérifiés par le
code et les tests**. La documentation (`doc.go` + `docs/algorithms/BIGFFT.md` + `FFT.md`)
explique correctement Schönhage-Strassen et les choix mémoire. `unsafe.Pointer` est
totalement absent ; les seuls usages d'`unsafe` sont `Sizeof` et le marqueur `go:linkname`,
tous idiomatiques.

Les axes d'amélioration sont incrémentaux : (1) wrapping d'erreur `%w` au lieu de `%v` dans
les 4 `recover`, (2) ajout d'un benchmark comparatif et d'une cible `Fuzz` contre
`math/big.Mul` pour verrouiller la non-régression de correction, (3) métriques sur le
fallback bump, (4) consolider `arith_amd64.go`/`arith_generic.go` aujourd'hui redondants.
Aucun risque mémoire ni concurrence détecté.
