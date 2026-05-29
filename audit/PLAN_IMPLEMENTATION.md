# PLAN D'IMPLÉMENTATION — Remédiation des 45 constats d'audit FibGo

> Source : [`RAPPORT_FINAL.md`](RAPPORT_FINAL.md) + détails par axe (`01..05_*.md`). Plan généré via workflow dynamique multi-agents (5 agents axe + 1 critique de complétude/conflits), puis consolidé.
> Branche : `chore/audit-remediation` (livraison : commits conventionnels groupés, **sans push/PR** — révision locale).
> Date : 2026-05-28.

---

## 0. Cadre & contraintes d'environnement

- **Toolchain** : Go 1.26.3, `CGO_ENABLED=0`, **pas de gcc/clang**, **pas de `make`** (Windows). `golangci-lint` v1.64.8 + `staticcheck` 2026.1 présents (recompilés go1.26).
- **Conséquences** : `go test -race` et `-tags gmp` **non exécutables ici** → certains constats sont implémentés mais leur vérification dynamique est **différée à Linux/WSL** (cf. §6). On utilise les équivalents `go` au lieu de `make`.
- **Décisions de cadrage utilisateur** :
  1. **A5-02 (CI)** → NE PAS réintroduire de CI GitHub Actions. `no-fix-documented` + garde-fous **locaux** (cibles Makefile `coverage-check`/`test-win`, `scripts/check.ps1|.sh`, doc).
  2. **A5-06 (module path)** → **RENOMMER** `github.com/agbru/fibcalc` → `github.com/agbruneau/FibGo` partout (code + doc user-facing), **hors** artefacts générés/historiques.
  3. **Constats INFORMATIF/faux-positifs sans correctif sensé** → `no-fix-documented` + justification. Ne pas dégrader la lisibilité (annotations Strassen/Fast Doubling conservées).
  4. **Livraison** → branche locale + commits, sans push/PR.
- **Invariants à préserver** (CLAUDE.md) : `finalizeStateRelease`→`clearStateAliases` inconditionnel ; `WithGC` panic-safe ; `putByKey` backing frais (E1-R4) ; atomics privés bigfft (ADR-0003) ; `recover()` re-propage les sentinels fermat ; routage `releaseWordSlice` par `cap` ; `threshold/manager` n'importe pas `internal/config` ; golden **immuable**.

---

## 1. Corrections issues de la revue de complétude

1. **A1-03 (et A1-01)** : la fonction libre `FastDoubling(n)` **n'existe pas**. Utiliser `MustNewCalculator(&FastDoublingCalculator{}).Calculate(ctx, nil, 0, n, opts)`. `FastDoublingMod(n, m)` existe (`modular.go:17`) et `MustNewCalculator`/`FastDoublingCalculator`/`MatrixExponentiationCalculator` existent.
2. **A5-06** : décompte réel ≈ **183 occurrences dans 87 fichiers `.go`** (les 392/105 incluaient docs+audit non touchés). La vérif `grep` doit viser 0 occurrence sur `cmd/ internal/ test/ go.mod`.
3. **Conflits de fichiers à sérialiser** (cf. §2).

---

## 2. Carte des conflits de fichiers (sérialiser, ne pas paralléliser)

| Fichier | Constats | Ordre |
|---|---|---|
| **Tous les `.go`** (préfixe import) | **A5-06** + tout spec ajoutant/éditant un `.go` | **A5-06 EN PREMIER** |
| `internal/bigfft/fermat.go` | A4-04, A4-05, A4-08(ifElseChain), A4-11(no-fix) | A4-04 (fix isolé) → A4-05 → A4-08 → A4-11(rien supprimer) |
| `internal/bigfft/fft_cache.go` | A2-02(code), A2-06(no-fix), A3-01(doc), A3-02(comment) | A2-02 seul édite le code |
| `internal/bigfft/pool.go` | A4-01, A2-05 | A4-01 → A2-05 |
| `internal/bigfft/pool_warming.go` | A4-01, A4-08(ifElseChain l.57) | même commit/sérialiser |
| `internal/bigfft/fft_recursion.go` | A2-03(code), A2-07/A3-05(no-fix) | A2-03 seul édite |
| `cmd/generate-golden/main.go` | A4-06, A4-08(octal), A4-13(nosec) | **1 seul commit** |
| `internal/fibonacci/matrix_ops.go` | A4-08(paramTypeCombine), A4-11(no-fix) | chirurgical |
| `internal/fibonacci/doubling_framework.go` | A1-06(code), A4-12(no-fix) | compatible |
| `internal/fibonacci/strategy.go` | A3-03 (import bigfft → dépend de A5-06) | après A5-06 |
| `.golangci.yml` | A4-07, A4-09, A5-07, (A4-13 option) | **1 commit `chore(lint)`** |
| `Makefile` | A5-02, A5-05, A5-06, A5-10 | A5-06 d'abord, puis cibles ; coordonner `.PHONY` |
| `README.md` | A5-02, A5-03, A5-04, A5-05, A5-06 | regrouper edits doc |
| `CLAUDE.md` | A3-01, A5-03, A5-06 | regrouper |
| `docs/PERFORMANCE.md` | A3-01, A3-04, A3-06 | regrouper |
| `docs/TESTING.md` | A5-02, A5-04, A5-08, A5-09, A5-10 | regrouper |
| `internal/calibration/profile.go` | A5-01, A4-08(octal l.151), A5-06(import) | A5-06 → A5-01 → A4-08 |
| `CONTRIBUTING.md`, `docs/ARCH.md`, `docs/BUILD.md` | A5-06 | mono |

---

## 3. Ordonnancement par vagues

- **Vague 0 — Rename module** : A5-06 (commit isolé, prérequis absolu). Vérif : `go build ./...` + `TestArchitectureLayering`.
- **Vague 1 — Config/lint** : A4-07 + A4-09 + A5-07 (`.golangci.yml`) ; A4-10 (`.gitattributes`).
- **Vague 2 — Tests purs** : A1-01, A1-03, A1-08 (fibonacci) ; A1-02 (gmp, écrit, vérif différée WSL) ; A5-01 (flaky calibration).
- **Vague 3 — Robustesse non perf-critique** : A1-04 (budget saturant), A1-05 (arena clamp), A2-01 (GC refcount + **ADR-0005**), A4-02 (dead code + nolint), A4-03 (gocyclo TUI), A4-06+A4-08(octal)+A4-13 (generate-golden, 1 commit).
- **Vague 4 — bigfft/fibonacci perf-sensible (benchmark avant/après OBLIGATOIRE, régression >5% = blocage)** : A4-04 → A4-05 → A4-08(fermat) ; A4-01 (+**ADR-0007**) → A2-05 → A4-08(pool_warming) ; A2-02 ; A2-03 (+**ADR-0006**) ; A3-03 ; A1-06 ; A4-08(matrix_ops).
- **Vague 5 — Doc + garde-fous locaux** : A5-02/A5-05/A5-10 (Makefile+scripts), A5-03/A5-04 (README/CONTRIBUTING), A3-01 (CLAUDE.md+PERFORMANCE), A5-08/A5-09 (TESTING), A3-04/A3-06 (PERFORMANCE notes).
- **Vague 6 — `no-fix-documented` / `blocked-env`** : A1-07, A2-04, A2-06, A2-07, A3-02, A3-05, A3-06, A3-07, A4-11, A4-12 (+A5-07 traité V1, A3-04 traité V5). Annotations/commentaires + journal de décision (ce fichier + CHANGELOG).

---

## 4. Tableau récapitulatif des 45 constats

Légende décision : `code`=correctif code · `test`=ajout/renfort test · `doc`=documentation · `cfg`=config/outillage · `nofix`=no-fix documenté · `env`=bloqué environnement.
Flags : **P**=perf-sensible (benchmark) · **R**=needs `-race` (WSL) · **G**=needs `gmp` (WSL) · **A**=ADR.

| ID | Sév | Déc | Vague | Fichiers | Changement (résumé) | Flags |
|---|---|---|---|---|---|---|
| A1-01 | MAJ | test | 2 | `fibonacci/fft_crossval_test.go` (nouveau) | `TestFFTRegimeCrossValidation` : F(1M) FastDoubling vs Matrix au seuil FFT défaut + oracle `FastDoublingMod` mod 10^50. Voie `MustNewCalculator`. | |
| A1-02 | MAJ | test | 2 | `fibonacci/calculator_gmp_test.go` | Cross-val GMP vs FastDoubling à N∈{1000,100k,1M} sous `//go:build gmp`. | G |
| A1-03 | MIN | test | 2 | `fibonacci/fibonacci_fuzz_test.go` | Oracle dans `FuzzFastDoublingMod` : si n≤10000, comparer à F(n) (via Calculator) mod m. | |
| A1-04 | MIN | code | 3 | `fibonacci/memory/budget.go` (+test) | Arithmétique **saturante** (`satMul`/`satAdd`) dans `EstimateMemoryUsage`. | |
| A1-05 | MIN | code | 3 | `fibonacci/fastdoubling.go`, `memory/arena.go` | Clamp float→int + produit ×15 (helper `EstimateWordsForN`/`clampMul15` dans `memory`). | P |
| A1-06 | INF | code | 4 | `fibonacci/doubling_framework.go` | Aligner métrique `usedFFT` sur `fk1BitLen` (déjà calculé l.181). Neutre perf. | P |
| A1-07 | INF | nofix | 6 | `fibonacci/fft.go` | Commentaire d'intention sur le `&&` (correctness : math/big exact). Pas de changement logique. | |
| A1-08 | INF | test | 2 | `fibonacci/fft_test.go` | Asserter T1/T2/T3 vs `math/big` dans `TestExecuteDoublingStepFFT`. | |
| A2-01 | **CRIT** | code | 3 | `memory/gc_control.go` (+test, **ADR-0005**) | **Refcount package-level** (mutex+depth) : 1er `Begin` désactive/sauve l'original, dernier `End` restaure. Préserve `WithGC` panic-safe. | R A |
| A2-02 | MAJ | code | 4 | `bigfft/fft_cache.go` | `logger` → `atomic.Pointer[zerolog.Logger]` (write `SetCacheLogger`, read `logPeriodicStats`). | P R |
| A2-03 | MAJ | code | 4 | `bigfft/fft_recursion.go`, `fibonacci/fft.go` (**ADR-0006**) | Annulation best-effort : `atomic.Bool` privé + `Set/ClearFFTCancellation` + watcher ctx côté fibonacci ; check en tête de récursion. Sans changer la signature hot path. | P R A |
| A2-04 | MIN | nofix | 6 | `fibonacci/threshold/manager.go` | Commentaire d'invariant single-writer-before-use au-dessus des blocs `var`. Pas de migration atomic. | |
| A2-05 | MIN | code | 4 | `bigfft/pool.go` | `releaseFFTState` relâche `tmp/tmp2` si `cap > maxPooledFFTTmpCap` (anti-bloat). | P |
| A2-06 | INF | nofix | 6 | `bigfft/fft_cache.go` | Correct par construction ; préserver invariant backing-frais. Aucun code. | R |
| A2-07 | INF | nofix | 6 | `fibonacci/common.go`, `bigfft/fft_recursion.go` | Séparation couches OK ; aucun code. | |
| A3-01 | MAJ | doc | 5 | `CLAUDE.md`, `docs/PERFORMANCE.md` | Cantonner le gain « 15-30 % » aux chemins `bigfft.Mul/Sqr`/FFTOnly ; le défaut (FastDoubling) ne consulte pas le cache. Pas de câblage (won't-fix sans benchmark). | |
| A3-02 | MIN | nofix | 6 | `bigfft/fft_cache.go` | Gate `cacheGate` déjà avant `computePolyKey` ; commentaire O(n) optionnel. | P |
| A3-03 | MIN | code | 4 | `fibonacci/strategy.go` | `FFTOnlyStrategy.Multiply/Square` → `bigfft.MulTo/SqrTo` quand `z!=nil` (supprime alloc+copie). | P |
| A3-04 | INF | env | 5 | `docs/PERFORMANCE.md` | Note de prudence (inversion temps dépend du CPU) ; reconfirmation Ryzen/Linux impossible ici. | |
| A3-05 | INF | nofix | 6 | `bigfft/fft_recursion.go` | Compromis pool sur branche parallèle intentionnel ; aucun code. | P R |
| A3-06 | INF | nofix | 5 | `docs/PERFORMANCE.md` | Note : sous le seuil FFT, coût dominé par Karatsuba `math/big`. | |
| A3-07 | INF | nofix | 6 | `docs/audits/bench-baseline.txt` | Baseline actuelle = borne conservatrice ; rafraîchissement ns/op fiable seulement sur Linux idle → optionnel/différé. | P |
| A4-01 | MAJ | code | 4 | `bigfft/pool.go`, `pool_warming.go` (**ADR-0007**) | Pools de slices → **pointeurs** (`*[]big.Word`, `*fermat`, `*[]nat`, `*[]fermat`) ; routage par `cap` inchangé. Benchmark décide (si neutre → annotation `//lint:ignore SA6002` + justif). | P R A |
| A4-02 | MIN | code | 3 | `cli/completion/registry.go`, `bigfft/context.go` | Supprimer `formatAlgoList` (+import `strings` orphelin) ; annoter `defaultContext` `//nolint:unused` (ADR-0004 §B1). | |
| A4-03 | MIN | code | 3 | `tui/sparkline.go` | Extraire helper `plotBrailleValue` (cyclo 17→<15) **ou** `//nolint:gocyclo`. | |
| A4-04 | MIN | code | 4 | `bigfft/fermat.go` | Retirer `c = 1` mort (`norm()` l.61). Commit isolé. | P R |
| A4-05 | MIN | code | 4 | `bigfft/fermat.go` | Récepteur `String()` `n`→`z` (1 ligne ; les 12 autres déjà `z`). | P |
| A4-06 | MIN | code | 3 | `cmd/generate-golden/main.go` | Pattern `run() error` pour exécuter `defer Close()`. | |
| A4-07 | MIN | cfg | 1 | `.golangci.yml` | Exclusion revive ciblée (stutter) + justification. Pas de rename d'API. | |
| A4-08 | MIN | code | 3/4 | generate-golden, display, profile, matrix_ops, fermat, memory_est, pool_warming, completion fish/zsh | octalLiteral `0o`, paramTypeCombine, ifElseChain→switch (hors fermat sensible) ; **NE PAS** `%q` sur bash.go (annoter `//nolint`). | P (sites bigfft/fib) |
| A4-09 | INF | cfg | 1 | `.golangci.yml` | `misspell.locale: US`→`UK` (graphie britannique cohérente). | |
| A4-10 | INF | cfg | 1 | `.gitattributes` (nouveau) | `*.go text eol=lf` (faux positifs gofmt CRLF). | |
| A4-11 | INF | nofix | 6 | `matrix_ops.go`, `fermat.go`, `fft.go` | Annotations math (Strassen/Fast Doubling) : NE PAS supprimer. | |
| A4-12 | INF | nofix | 6 | `doubling_framework.go` + lot | Shadow `err` hot path **confirmé bénin** ; reste = nettoyage opportuniste hors sprint. | R |
| A4-13 | INF | code | 3 | `cmd/generate-golden/main.go` | `//nosec G304` sur `OpenFile` (cohérence modèle de menace). | |
| A5-01 | MAJ | test | 2 | `calibration/profile.go`, `profile_test.go` | `renameAtomic` : backoff exponentiel borné + maxAttempts 10→40 ; test tolère l'erreur de partage Windows côté écrivain. | |
| A5-02 | MAJ | nofix | 5 | `Makefile`, `scripts/check.ps1\|.sh` (nouveaux), `README.md`, `docs/TESTING.md` | NE PAS recréer `.github/`. Garde-fous locaux + doc de la décision assumée. | |
| A5-03 | MIN | doc | 5 | `README.md`, `CLAUDE.md`, `CONTRIBUTING.md` | Go 1.25→1.26.0, toolchain 1.26.2→1.26.3 (hors mentions historiques). | |
| A5-04 | MIN | doc | 5 | `README.md`, `docs/TESTING.md`, (CHANGELOG immuable) | Ne pas figer de nouveau chiffre ; pointer `make coverage(-check)`. | |
| A5-05 | MIN | cfg | 5 | `Makefile`, `README.md`, `docs/PORTABILITY.md` | Cible `test-win` (sans `-race`) + doc que `make test` (-race) requiert CGO/WSL. | |
| A5-06 | MIN | code | 0 | go.mod + 87 `.go` + doc user-facing | Rename module path. **Prérequis**. | A(commit body) |
| A5-07 | INF | nofix | 1 | `.golangci.yml` | Commentaire de verrou schéma v1 (pas de migration v2). | |
| A5-08 | INF | doc | 5 | `docs/TESTING.md` | Documenter angles morts e2e (`[no statements]`) + gmp. | |
| A5-09 | INF | doc | 5 | `docs/TESTING.md` | Documenter `generate-golden` peu couvert (outil dev). | |
| A5-10 | INF | cfg | 5 | `Makefile`, `docs/TESTING.md` | Cible `coverage-check` (échoue si total <80 %). | |

---

## 5. Nouveaux ADR à rédiger

- **ADR-0005** — Contrôle GC concurrent sérialisé par refcount package-level (extension de l'invariant `gc_control.go` au cas concurrent, A2-01).
- **ADR-0006** — Annulation best-effort de la récursion FFT via `atomic.Bool` sans changement de signature hot path (A2-03). *Selon résultat : si l'approche atomic-globale est jugée incorrecte (cross-talk entre FFT concurrents), documenter le **report** vers FFTContext (ADR-0004 §B1).*
- **ADR-0007** — Pooling par pointeur de slice dans `bigfft` (A4-01), avec benchmark avant/après ; ou justification de l'annotation si le gain est neutre.

> ⚠ **A2-03 — réserve technique** : un `atomic.Bool` *package-global* de cancel provoquerait un cross-talk entre multiplications FFT concurrentes (`executeParallel3`, `--algo all`). Décision d'implémentation à l'exécution : soit annulation **scopée** (acceptable car `errgroup.WithContext` annule déjà tous les frères ensemble — à documenter), soit **report** au refactor FFTContext (ADR-0004 §B1) avec test de la granularité grossière existante. Tranché en Vague 4 avec preuve.

---

## 6. Vérifications différées (Linux/WSL — non exécutables ici)

- **`-race`** (A2-01, A2-02, A2-03, A2-06, A4-04, A4-12, A3-05) : `CGO_ENABLED=1 go test -race ./...`.
- **`-tags gmp`** (A1-02) : `CGO_ENABLED=1 go test -tags gmp ./internal/fibonacci/`.
- **ns/op fiables** (A3-04, A3-07) : Ryzen/Linux idle, `-count≥10` + `benchstat`.

Ces items sont **implémentés** (code/test/doc) ; seule leur **confirmation dynamique** est différée. Documenté dans CHANGELOG + ce plan.

---

## 7. Portes de vérification par vague

À chaque vague (barrière exécutée par l'orchestrateur) : `go build ./...` → `go vet ./...` → `go test ./...` (sans `-race`) → `golangci-lint run ./...` (ciblé). **Vague 4** ajoute : benchmark `allocs/op`/`B/op` avant/après (`go test -run=^$ -bench=... -benchmem ./internal/bigfft/ ./internal/fibonacci/`) comparé à `docs/audits/` ; golden tests obligatoires. Commit conventionnel par groupe.
