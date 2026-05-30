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
| WASI / js | — | — | — | non supporté (`unsafe.Pointer`, `runtime/debug`) |

## 2. Fallbacks plate-forme

### 2.1 `internal/bigfft/arith_*.go`

- `arith_amd64.go` (`//go:build amd64`) : implémentation assembleur optimisée
  pour les routines arithmétiques de bas niveau (`addVV`, `subVV`, `addVW`,
  `subVW`, `shlVU`).
- `arith_generic.go` (`//go:build !amd64`) : implémentation Go pure, prise
  par défaut sur toute autre architecture (arm64, riscv64, ppc64le, etc.).

**Conséquence** : le binaire compilé pour `linux/arm64` ou `darwin/arm64`
est fonctionnellement équivalent ; la performance arithmétique pure est
légèrement moindre (5-10 % d'écart attendu sur les très grands `big.Int`,
non profilé formellement à ce jour).

### 2.2 `internal/bigfft/cpu_amd64.go`

- `cpu_amd64.go` (`//go:build amd64`) : détection runtime AVX2 pour activer
  un dispatch optimisé dans les routines FFT.
- Sur les architectures non-amd64, aucune détection CPU avancée ;
  l'implémentation generic est utilisée systématiquement.

### 2.3 Backend GMP (`build tag gmp`)

- Activé via `go build -tags gmp` ; nécessite `libgmp-dev` à la compilation
  et à l'exécution.
- Désactivé par défaut. Le `Dockerfile` et `.devcontainer/devcontainer.json`
  installent `libgmp-dev` mais ne posent pas le tag — il faut l'ajouter
  explicitement à la commande de build.

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
- **WebAssembly** : non supporté. La codebase utilise `unsafe.Pointer`,
  `runtime/debug.Stack`, et des assertions de panic post-condition qui
  rendent un portage non trivial.
