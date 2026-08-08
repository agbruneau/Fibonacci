# Portabilité — FibCalc

Document de référence pour la matrice OS/arch supportée, les *fallbacks*
plate-forme, et la chaîne de compilation par cible. Audit-PRD P1-09 /
E10-R5 / Sprint S4-T5.

## 1. Matrice supportée

| OS | Architecture | CGO | Race detector | Vérification locale |
|---|---|---|---|---|
| Linux | amd64 | ✅ requis pour `-race` et `gmp` | ✅ via gcc | `make test` (race natif) |
| Linux | arm64 | ❌ désactivé | ❌ (cross-compile only) | `make build-all` (build only) |
| Windows | amd64 | ✅ requis pour `-race` (via MinGW) | ✅ via MinGW gcc si installé | `make test` ou `go test -race` |
| Windows | arm64 | ❌ désactivé en cross-compile | ❌ (cross-compile only) | `make build-windows-arm64` (build only) |
| macOS | amd64 | ✅ pour `-race` | ✅ via clang | `make test` (race natif) |
| macOS | arm64 (Apple Silicon) | ❌ désactivé en cross-compile | ❌ (cross-compile only) | `make build-all` (build only) |
| WASI / js | — | — | — | non supporté (`runtime/debug`, assertions de post-condition) |

**64 bits uniquement.** Aucune cible 32 bits n'est supportée et aucune n'est
construite par `make build-all`. La compilation **échoue sur toute cible où
`int` fait 32 bits** — `386`, `arm`, `mips`, `mipsle` — et l'échec est le même
dans tous les cas : `maxReasonableWords = 1 << 60` (déclaré
`internal/fibonacci/memory/arena.go:maxReasonableWords`, utilisé trois fois dans
`arenaTotalWords`) déborde un `int` 32 bits. Vérifié le 2026-08-07 :
`GOOS=linux GOARCH=386 go build ./...` et `GOOS=linux GOARCH=arm go build ./...`
produisent tous deux, mot pour mot, `cannot use maxReasonableWords … (overflows)`
en `arena.go:26,30` et `maxReasonableWords / 10 … overflows int` en
`arena.go:29`. Ces trois numéros sont la **sortie citée du compilateur**, pas un
renvoi de lecture : ils doivent rester exacts et être revérifiés en relançant les
deux commandes ci-dessus, pas corrigés par recherche de symbole. La mention `386` en
§2.2 décrit uniquement la branche
`runtime.GOARCH` de `DetectHardwareHeuristic()` ; elle ne constitue pas une
déclaration de support.

## 2. Fallbacks plate-forme

### 2.1 `internal/bigfft/arith.go`

- `arith.go` (portable, sans build tag — fusion de l'ancien split
  `arith_amd64.go`/`arith_generic.go`, audit FFT-06) : wrappers exportés
  `AddVV`, `SubVV`, `AddMulVVW`, qui délèguent aux routines internes de
  `math/big` via `go:linkname`. Les déclarations `go:linkname` vivent dans
  `arith_decl.go` (commun à toutes les architectures), qui couvre aussi
  `addVW`, `subVW`, `shlVU`. Aucun assembleur original dans ce dépôt :
  l'assembleur optimisé exploité est celui de `math/big`, pour toutes les
  architectures (amd64, arm64, riscv64, ppc64le, etc.).

**Conséquence** : le binaire compilé pour `linux/arm64` ou `darwin/arm64`
est fonctionnellement équivalent ; la performance arithmétique pure est
légèrement moindre (5-10 % d'écart attendu sur les très grands `big.Int`,
non profilé formellement à ce jour).

### 2.2 `internal/config/hardware.go`

- Sans `//go:build` : `DetectHardwareHeuristic()` branche à l'exécution sur
  `runtime.GOARCH` et ne consulte `golang.org/x/sys/cpu` (`HasAVX512F`,
  `HasAVX2`) que pour `amd64`/`386` ; toute autre architecture reste en
  `SIMDNone`. **Aucun chemin de code de `internal/bigfft` n'en dépend** : le
  dispatch FFT ne consulte jamais `SIMDKind`. Le résultat a deux
  consommateurs, tous deux dans `internal/config` :
  - `HeuristicKey()` (`hardware.go:HardwareHeuristic.HeuristicKey`) /
    `CurrentHardwareHeuristicKey()` (`hardware.go:CurrentHardwareHeuristicKey`) —
    invalidation de profil de calibration (un profil calibré sur une classe
    SIMD différente est rejeté) ;
  - **les trois estimateurs de seuils adaptatifs**, qui branchent directement
    sur `h.SIMD` : `thresholds.go:estimateParallelThresholdForHeuristic`
    (parallèle : −512 / −256 bits si NumCPU ≥ 8),
    `thresholds.go:estimateFFTThresholdForHeuristic` (FFT : 460 000 / 480 000 /
    500 000 bits selon AVX512 / AVX2 / autre),
    `thresholds.go:estimateStrassenThresholdForHeuristic` (Strassen : 224 / 240 /
    256 bits si NumCPU ≥ 4).
- Sur les architectures non-amd64/386, aucune détection SIMD avancée ;
  seule la classification `NumCPU`/`GOARCH` s'applique.

### 2.3 Backend GMP (`build tag gmp`)

- Activé via `go build -tags gmp` ; nécessite `libgmp-dev` à la compilation
  et à l'exécution.
- Désactivé par défaut. Le `Dockerfile` (image de production) est
  `CGO_ENABLED=0` sans paquet `apt` : il ne peut pas construire ce backend.
  Seul `.devcontainer/devcontainer.json` installe `libgmp-dev` (poste de
  développement) ; le tag `gmp` doit alors être ajouté explicitement à la
  commande de build locale.

## 3. Race detector

Le race detector Go nécessite CGO. La cible canonique `make test` lance
`go test -race`, qui exige donc un compilateur C :

- **Linux + macOS** : CGO via gcc/clang natif. `make test` (avec `-race`)
  fonctionne directement.
- **Windows** : CGO via MinGW si le contributeur l'installe localement.
  Sinon, `make test` échoue faute de gcc. Sur un poste Windows pur sans
  gcc, utiliser la cible sans `-race` **`make test-win`** (équivalent
  `go test -v -cover ./...`) ou le script de garde-fou local
  **`scripts/check.ps1`**. Le `-race` reste **recommandé** : l'exécuter via
  WSL ou un poste Linux/macOS.

> Résumé : `make test` = suite complète avec `-race` (CGO requis, donc
> Linux/macOS ou WSL sous Windows) ; `make test-win` / `scripts/check.ps1`
> = repli Windows sans `-race`.

Pour les builds cross-compile (`linux/arm64`, `darwin/arm64`), le race
detector n'est pas exécuté — seule la **compilabilité** sans CGO est
vérifiée par `make build-all`.

## 4. Procédure de build par cible

### Linux/amd64 (cible principale)

```bash
make build              # standard
make build-pgo          # avec profil PGO
```

### Linux/arm64 (croisé)

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/fibcalc-linux-arm64 ./cmd/fibcalc
```

### macOS/arm64 (Apple Silicon, croisé)

```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o build/fibcalc-darwin-arm64 ./cmd/fibcalc
```

### Windows/amd64

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/fibcalc.exe ./cmd/fibcalc
```

### Container reproductible

```bash
docker build -t fibcalc:local .
docker run --rm fibcalc:local --help
```

L'image distroless ne ship que le binaire linké statiquement (pas de
runtime libc, pas de shell). Pour debug : remplacer la stage finale par
`gcr.io/distroless/base-debian12:debug`.

## 5. Vérification locale

`make build-all` exécute `go build` pour les cibles suivantes et doit
passer sans erreur avant tout commit touchant `internal/bigfft/` ou
`internal/fibonacci/` :

- `linux/amd64`
- `linux/arm64`
- `windows/amd64`
- `windows/arm64`
- `darwin/amd64`
- `darwin/arm64`

Une régression introduisant une dépendance amd64-exclusive non gardée par
`//go:build` fera échouer immédiatement l'une des cibles ci-dessus.

## 6. Limitations connues

- **Pas de bench cross-arch** : les chiffres dans
  `docs/audits/bench-baseline.txt` proviennent d'un host amd64. Un
  benchmark arm64 demande un poste Apple Silicon ou ARM Linux.
- **GMP non testé en cross-compile** : `gmp` build tag requiert CGO,
  donc seul `linux/amd64` exerce cette branche.
- **WebAssembly** : non supporté. La codebase importe `runtime/debug`
  (blocs `import` de `internal/bigfft/fft.go` et de
  `internal/fibonacci/memory/gc_control.go`) et
  s'appuie sur des assertions de panic post-condition, ce qui rend un portage
  non trivial. Le code de production n'utilise **pas** `unsafe.Pointer` : le
  seul usage de `unsafe` est `unsafe.Sizeof` (`internal/bigfft/fft.go:_W`), plus
  l'import muet requis par `go:linkname` (bloc `import` de `internal/bigfft/arith_decl.go`) ;
  l'unique `unsafe.Pointer` du dépôt est dans un test
  (`internal/fibonacci/memory/arena_test.go`, `TestCalculationArena_MultipleAllocs_NoAliasing`).
- **32 bits** : non supporté, voir §1.
