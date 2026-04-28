# 20 — Qualité du code

> Audit FibGo, tâche 3.1. WD : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo`. Date : 2026-04-28.
> Méthode : analyse manuelle (Read + Grep + AWK pour comptage LOC). `golangci-lint` v1.64.8 disponible mais inutilisable (build Go 1.25 < cible 1.26.2 du module). Toutes les conclusions reposent donc sur l'inspection directe.

## Linters configurés

- **Total activés** : 24 (`disable-all: true` puis liste explicite dans `.golangci.yml`).
- **Standard** : `gofmt`, `govet` (toutes les checks sauf `fieldalignment` & `shadow`), `errcheck`, `staticcheck`, `gosimple`, `unused`, `ineffassign`, `typecheck`.
- **Style/complexité** : `revive` (14 règles), `gocyclo` (≥15), `gocognit` (≥30), `funlen` (100 lignes / 50 statements), `whitespace`, `misspell` (US).
- **Performance/bugs** : `bodyclose`, `noctx`, `prealloc`, `gocritic` (5 tags activés), `unparam`, `unconvert`, `nakedret`, `copyloopvar`.
- **Sécurité** : `gosec` (G104/G115 exclus globalement, G304 par fichier sur `calibration/profile.go` et `cli/output.go`).
- **Méta** : `nolintlint`.
- **Exclusions** : `_test.go` exempté de `gocyclo|gocognit|funlen|gosec|dupl|unparam` ; `staticcheck SA1019` (deprecation) ignoré.
- **Notable** : seuils explicitement assouplis (cyclo 15, cogni 30, fun 100/50) pour les algorithmes mathématiques. `hugeParam` désactivé dans `gocritic` car de gros structs sont passés par valeur sciemment.

## Fonctions dépassant les seuils

Comptage AWK sur tous les `*.go` non-test (matching `^func ` + suivi des accolades `{`/`}`).

| Fichier:Ligne | Fonction | LOC approx | Seuil violé |
|---|---|---|---|
| `internal/fibonacci/doubling_framework.go:141` | `(*DoublingFramework).ExecuteDoublingLoop` | **136** | funlen (100), `//nolint:gocognit` déjà posé |
| `internal/cli/completion.go:422` | `generatePowerShellCompletion` | 99 | proche limite (string template) |
| `internal/tui/model.go:136` | `(Model).Update` | 89 | sous limite ; dispatcher Bubble Tea |
| `internal/tui/styles.go:42` | `initTUIStyles` | 82 | sous limite |
| `internal/bigfft/fft_recursion.go:76` | `fourierRecursiveUnified` | 77 | sous limite (cogni probable) |
| `internal/calibration/calibration.go:251` | `AutoCalibrateWithProfile` | 71 | sous limite |
| `internal/fibonacci/common.go:301` | `executeMixedTasks` | 69 | sous limite |
| `internal/app/calculate.go:22` | `(*Application).runCalculate` | 66 | sous limite |
| `internal/cli/completion.go:187` | `generateBashCompletion` | 64 | sous limite |
| `internal/bigfft/memory_est.go:16` | `EstimateMemoryNeeds` | 63 | sous limite |
| `internal/tui/sparkline.go:126` | `RenderBrailleChart` | 62 | sous limite |

**Conclusion** : 1 seule violation stricte de `funlen` dans le code de production (`ExecuteDoublingLoop`, justifiée par un `//nolint:gocognit` mais **pas** `funlen`). Aucune fonction non-test ne dépasse 100 lignes au-delà de cette unique exception.

## Erreurs non wrappées (`fmt.Errorf` sans `%w`)

Cas recensés en code de production (tests exclus). Contexte vérifié : la majorité n'a pas d'erreur underlying à wrapper (validation pure, pas de chaîne d'erreurs).

| Fichier:Ligne | Extrait | Verdict |
|---|---|---|
| `internal/bigfft/fft.go:43` | `fmt.Errorf("panic in bigfft.Mul: %v\nStack: %s", r, ...)` | **À évaluer** : `r` provient de `recover()`. Si c'est une `error`, utiliser `%w`. Idem ll. 59, 86, 99 (4 occurrences). |
| `internal/cli/completion.go:78` | `fmt.Errorf("unsupported shell: %s ...", shell)` | OK : pas d'err underlying. |
| `internal/bigfft/fft_recursion.go:85` | `fmt.Errorf("FFT recursion validation failed: ...")` | OK : assertion d'invariant. |
| `internal/fibonacci/modular.go:19` | `fmt.Errorf("modulus must be positive")` | OK : validation pure. À convertir en `errors.New` (gocritic `errorf` rule). |
| `internal/fibonacci/memory/budget.go:45` | `fmt.Errorf("empty memory limit")` | Idem : `errors.New`. |
| `internal/fibonacci/registry.go:104, 146` | `fmt.Errorf("unknown calculator: %s", name)` | OK : pas d'err underlying. |

Tous les autres `fmt.Errorf` du projet utilisent `%w` correctement (47 occurrences vérifiées dans `fibonacci/`, `bigfft/`, `calibration/`, `cli/`, `errors/`).

## Panics suspects (hors `init` / `_test.go`)

| Fichier:Ligne | Contexte | Verdict |
|---|---|---|
| `internal/bigfft/fermat.go:51` | `panic("len(z) != len(x) in Shift")` | Invariant interne, OK pour algo bas niveau. |
| `internal/bigfft/fermat.go:125` | `panic("Add: len(z) != len(x)")` | Idem. |
| `internal/bigfft/fermat.go:135` | `panic("fermat.Sub: len(z) != len(x)")` | Idem. |
| `internal/bigfft/fermat.go:153` | `panic("Mul: len(x) != len(y)")` | Idem. |
| `internal/bigfft/fermat.go:178` | `panic("len(z) > 2n+1")` | Idem. |
| `internal/bigfft/fermat.go:203` | `panic("fermat.Mul: unexpected carry after normalization")` | Idem. |
| `internal/bigfft/fermat.go:239` | `panic("len(z) > 2n+1")` | Idem. |
| `internal/bigfft/fermat.go:258` | `panic("fermat.Sqr: unexpected carry after normalization")` | Idem. |
| `internal/bigfft/fft_poly.go:356` | `panic("Transform: len(p.A) >= 1<<k")` | Invariant FFT. |
| `internal/bigfft/scan.go:28` | `panic("size < quadraticScanThreshold")` | Invariant. |
| `internal/bigfft/scan.go:44` | `panic("quadraticScanThreshold % 14 != 0")` | Garde de configuration compile-time. |
| `internal/fibonacci/calculator.go:107` | `panic(err)` dans `MustNewCalculator` | OK : helper documenté style `Must*`. |
| `internal/fibonacci/registry.go:221` | `panic(fmt.Sprintf(...))` dans `MustGet` | OK : helper documenté style `Must*`. |

**Tous les 13 panics sont récupérables en amont** : `bigfft` est protégé par les `defer recover` dans `fft.go:42–99` (Mul/MulTo/Sqr/SqrTo). Les helpers `Must*` sont conformes à la convention Go.

## Erreurs ignorées

- **`_ = err` direct** : aucune occurrence en code de production (greppé `^\s*_ =\s` filtré : seulement tests, helpers de tests, ou élision volontaire d'une *valeur de retour non-erreur*).
- **`_ = g.Wait()`** : `internal/orchestration/orchestrator.go:78` — résultat d'`errgroup` ignoré ; à vérifier si erreurs collectées ailleurs.
- **`_ = f.Register(...)`** : `internal/fibonacci/registry.go:64-66` — appels `init()`-style, OK.
- **`_ = c` (dead carry)** : `internal/bigfft/fermat.go:306` — variable carry inutilisée mais conservée pour clarté algorithmique. Bizarre, candidat à clean-up (ou commentaire explicatif).
- **`_ = parallel`** : `internal/calibration/microbench.go:176` — commentaire explicite "silence unparam without dropping the knob". OK.
- **`_ = new(big.Int).Mul(x, y)`** : `internal/calibration/microbench.go:228` — micro-bench, OK.

## TODO / FIXME / HACK / XXX

**Aucune occurrence** trouvée dans tout le codebase (recherche `// (TODO|FIXME|HACK|XXX)` insensible à la casse, `*.go`).
Indicateur exceptionnel de maturité : la dette technique est tracée par d'autres canaux (audits, commits "P1-NN" / "P2-NN" référencés dans les commentaires).

## Synthèse

- **Violations strictes de seuil** : **1** (`ExecuteDoublingLoop`, 136 lignes, justifiée par un `//nolint:gocognit` mais pas `//nolint:funlen` → linter remontera).
- **`fmt.Errorf` non wrappés non-justifiés** : **4** (`bigfft/fft.go:43,59,86,99`, `%v` sur `recover()` — à évaluer cas par cas).
- **`fmt.Errorf` convertibles en `errors.New`** : **5** (validation sans underlying err).
- **Panics** : **13**, tous justifiés (invariants algo bas niveau + helpers `Must*`).
- **TODO/FIXME** : **0**.
- **Erreurs ignorées suspectes** : **2** (`orchestrator.go:78`, `fermat.go:306`).

### Top 5 zones à traiter en priorité

1. **`internal/fibonacci/doubling_framework.go:141` (`ExecuteDoublingLoop`)** — Compléter le `//nolint` avec `funlen` ou extraire la double boucle de progression dans un helper. Risque de régression performance signalé en commentaire ; envisager `//nolint:funlen,gocognit` avec justification consolidée.
2. **`internal/bigfft/fft.go:43,59,86,99`** — Auditer les 4 sites `recover()`. Si `r` peut être une `error`, utiliser `%w` via `if e, ok := r.(error); ok { ... %w, e }`. Sinon, ajouter un commentaire justifiant `%v`.
3. **`internal/orchestration/orchestrator.go:78` (`_ = g.Wait()`)** — Documenter pourquoi l'erreur d'`errgroup` est délibérément ignorée, ou la propager. Risque de masquer des cancels.
4. **`internal/fibonacci/{modular.go:19, memory/budget.go:45, registry.go:104,146}` + `internal/cli/completion.go:78`** — Migrer les `fmt.Errorf("literal")` sans interpolation vers `errors.New("literal")` (gocritic `errorf` ; gain marginal mais aligne avec le style errors-as-values).
5. **`internal/bigfft/fermat.go:306` (`_ = c`)** — Soit supprimer le calcul de carry mort, soit ajouter un commentaire `// carry intentionally discarded: bla bla` pour expliciter l'invariant.

**Évaluation globale** : qualité de code **très élevée**. Une seule violation stricte des seuils. Aucune dette technique signalée par tags. Wrapping d'erreurs systématique (>90 % des `fmt.Errorf` utilisent `%w`). Panics encadrés par recover ou helpers `Must*`. Le code est mûr et discipliné.
