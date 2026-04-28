# 99 — Backlog d'améliorations

Backlog priorisé agrégé à partir des 22 rapports d'audit (`01-…` à `72-…`) produits le 2026-04-28.
Méthode : extraction de toutes les actions/recommandations, dédoublonnage, scoring Impact × Effort.

## Légende

- **Impact** : Critique (C) / Élevé (É) / Moyen (M) / Faible (F)
- **Effort** : XS (<2 h) / S (2-8 h) / M (1-3 j) / L (>3 j)
- **Priorité** :
  - **P0** sprint immédiat — Critique + XS/S, ou Élevé + XS
  - **P1** mois en cours — Critique + M/L, Élevé + S/M, Moyen + XS
  - **P2** trimestre — Élevé + L, Moyen + S/M
  - **P3** backlog — reste

## Synthèse

- Total items : **47**
- P0 : **6** | P1 : **15** | P2 : **15** | P3 : **11**

3 actions P0 absolues (rang 1-3) :
1. **Mettre en place une CI GitHub Actions** (`.github/workflows/ci.yml`) — débloque la quasi-totalité des audits qui constatent l'absence de garde-fou automatique.
2. **Réinstaller un `golangci-lint` v2.x compatible Go 1.26.2** et le pinner — sans cela aucune des règles `.golangci.yml` n'est vérifiée localement.
3. **Sceller le `GCController` global** (`memory/gc_control.go`) — seul finding correctness pouvant corrompre l'état GC du process en exécution concurrente.

---

## P0 — Sprint immédiat

| # | Action | Domaine | Impact | Effort | Source(s) | Notes |
|---|---|---|---|---|---|---|
| 1 | Créer `.github/workflows/ci.yml` (matrix lint + test `-race -cover` Linux/Windows/macOS, Go 1.25/1.26, gosec hebdo, govulncheck, codecov) | CI | C | S | `50-tests-coverage.md` Top 1 ; `70-makefile.md` Top 1 ; `30-security.md` L-1 ; `51-race-detector.md` action 2 ; `71-lint.md` action 3 ; `61-doc-tech.md` §Maintenance | Débloque ~6 actions dépendantes |
| 2 | Réinstaller golangci-lint v2.x compatible Go 1.26.2 et le pinner (Go `tool` directive ou tools.go) | Outillage | C | XS | `71-lint.md` Top 1+2 ; `20-code-quality.md` §préambule | Prérequis avant ré-audit lint |
| 3 | Sérialiser `GCController.Begin/End` via mutex package-level (ou refuser calculs concurrents avec GC contrôlé) | Perf/correctness | C | S | `40-fibonacci-perf.md` F1 + Top 5 #1 ; `42-concurrency.md` §GC controller | Risque corruption `originalGCPercent` en multi-tenant |
| 4 | Régénérer `coverage.out` (référence `generator_iterative.go` introuvable) et l'ajouter au `.gitignore` (artefact) | Tests | É | XS | `50-tests-coverage.md` Top 2 + §préambule | `go tool cover -func` plante sinon |
| 5 | Documenter dépendance CGO de `make test -race` dans README/CLAUDE.md (instructions MSYS2 UCRT64 / TDM-GCC) + cible `make test-race-check` qui échoue explicitement | Doc/Outillage | É | XS | `51-race-detector.md` Top 1+3 | Évite faux négatifs concurrence en dev Windows |
| 6 | Supprimer/migrer `internal/fibonacci/testing.go` (mock exporté en code prod) vers `_test.go` ou `fibonaccitest/` | Code quality | É | XS | `12-interfaces.md` action 3 ; `21-dead-code.md` Top 5 #2 | Anti-pattern : pollue API publique du package cœur |

## P1 — Mois en cours

| # | Action | Domaine | Impact | Effort | Source(s) | Notes |
|---|---|---|---|---|---|---|
| 7 | Supprimer `internal/fibonacci/progress_aliases.go` (12/14 symboles morts) ; migrer consommateurs vers `internal/progress` directement | Architecture | É | S | `11-layering.md` Top 3 #1+#2 ; `21-dead-code.md` Top 5 #1 | Casse la fuite domaine→infra (1 seule violation Clean Architecture) |
| 8 | Borne par défaut sur `-n` (ex. `MaxN=1e10`) ou `--memory-limit` obligatoire au-delà d'un seuil | Sécurité | É | S | `30-security.md` M-1 + Top 5 #1 | DoS local mémoire/CPU |
| 9 | Assainir `-output` et `-calibration-profile` (`filepath.Clean`, contrainte $HOME/cwd, ou justifier) | Sécurité | É | S | `30-security.md` M-2 + Top 5 #2 | Path traversal théorique (G304) |
| 10 | Centraliser doubles `Calculator` dans `fibonaccitest/` (consolider 6 mocks dispersés) | Tests | M | S | `12-interfaces.md` action 1 ; `21-dead-code.md` Top 5 #3 | Base : `MockCalculator` de `orchestration_test.go` |
| 11 | Mettre à jour `docs/architecture/dependency-graph.mermaid` (retirer arêtes `cli→fib`, `calib→cli` ; sortir `errors` de leaves) + re-dater `validation-report.md` | Doc | M | XS | `10-architecture.md` recommandation 1 ; `61-doc-tech.md` Top 3 #1 | 4 écarts diagramme |
| 12 | Synchroniser `container-diagram.mermaid` (retirer `calib→cli`, ajouter `tui→sysmon`) | Doc | M | XS | `10-architecture.md` recommandation 2 | Cohérence C4 |
| 13 | CHANGELOG : ajouter entrées `[2.0.0]`, `[3.0.0]`, corriger lien compare `v3.0.0...HEAD` | Doc | É | XS | `60-doc-user.md` Top 5 #1 ; `03-git-snapshot.md` obs 4 | Versioning cassé (CHANGELOG bloqué à 1.0.0) |
| 14 | Documenter `--gc-control` / `FIBCALC_GC_CONTROL` dans README + `.env.example` | Doc | É | XS | `60-doc-user.md` Top 5 #2 | Flag actif non documenté |
| 15 | Corriger `CONTRIBUTING.md` : snippet `CoreCalculator` (majuscule), ajouter `make build-pgo`/`build-all`, mettre à jour File Organization (`memory/`, `threshold/`) + section License (Apache 2.0/DCO) | Doc | M | XS | `60-doc-user.md` Top 5 #3 ; `31-licenses.md` action P3 #3 | |
| 16 | BUILD.md : aligner cibles cross-build (`build-linux-arm64`, `build-windows-arm64`), retirer cibles fantômes `generate-mocks`/`install-mockgen`, aligner compteur linters (22 vs 24) | Doc | M | XS | `60-doc-user.md` Top 5 #4 ; `70-makefile.md` Top 5 #2 ; `72-cross-build.md` action 1 | |
| 17 | README : ajouter section explicite Build Tag `gmp` (procédure `go build -tags=gmp`) | Doc | M | XS | `60-doc-user.md` Top 5 #5 ; `70-makefile.md` Top 5 #5 | |
| 18 | Migrer `fmt.Errorf("literal")` vers `errors.New` (5 sites : `modular.go:19`, `memory/budget.go:45`, `registry.go:104,146`, `completion.go:78`) | Code quality | M | XS | `20-code-quality.md` Top 5 #4 | gocritic `errorf` |
| 19 | Wrapper les 4 `recover()` de `bigfft/fft.go` en `%w` quand `r` est `error` (sites :43,59,86,99) | Code quality | M | XS | `20-code-quality.md` Top 5 #2 ; `41-bigfft-arena.md` F1 | Permet `errors.Is/As` |
| 20 | Compléter `//nolint` sur `ExecuteDoublingLoop` (`doubling_framework.go:141`) avec `funlen` ou extraire helper | Code quality | M | XS | `20-code-quality.md` Top 5 #1 | 1 violation funlen stricte |
| 21 | Fusionner `formatBytesInternal` (`memory/budget.go`) avec `format.FormatBytes` | Code quality | M | XS | `21-dead-code.md` Top 5 #4 | Duplication exacte |

## P2 — Trimestre

| # | Action | Domaine | Impact | Effort | Source(s) | Notes |
|---|---|---|---|---|---|---|
| 22 | Ajouter `NOTICE` racine + générer `THIRD_PARTY_LICENSES/` via `go-licenses save` (cible `make licenses`) | Outillage/Licences | É | M | `31-licenses.md` action P2 #1 | Indispensable avant release binaire publique |
| 23 | Étendre `cmd/generate-golden` avec N ≥ 200 000 pour couvrir le palier FFT | Tests | M | S | `50-tests-coverage.md` Top 3 | Borne actuelle N=10000 ne stresse pas FFT |
| 24 | Allocations dans `threshold.getActiveMetrics` : pré-allouer buffer membre (éviter 3 slices par appel) | Perf | M | S | `40-fibonacci-perf.md` F2 + Top 5 #2 | Aligne avec philosophie zero-alloc |
| 25 | Tasks Strassen sur la stack (`[8]multiplicationTask` + slice non-capturant) | Perf | M | S | `40-fibonacci-perf.md` F4/F5 + Top 5 #3 | Évite 7-8 escapes/iter Matrix Exp |
| 26 | Pool `gmp.Int` pour 4 temporaires du backend GMP | Perf | M | S | `40-fibonacci-perf.md` F8 + Top 5 #4 | Bénéfice multi-runs/serveur |
| 27 | Right-sizing `CalculationArena` : compter exactement les buffers pré-sized (5 vs ×15 actuel) | Perf | M | S | `40-fibonacci-perf.md` F9 + Top 5 #5 | Économie plusieurs MB à n=10M+ |
| 28 | Ajouter benchmark comparatif `bigfft.Mul` vs `(*big.Int).Mul` ; ajouter cible `Fuzz` contre `math/big.Mul` | Tests/Perf | M | S | `41-bigfft-arena.md` F2/F3 ; `43-bench-baseline.md` §Limites | Verrouille seuil 1800 mots et oracle gratuit |
| 29 | Ré-exécuter benchmarks avec `-benchtime=3s -count=5` + `benchstat` ; archiver baseline exploitable | Tests/Perf | M | S | `43-bench-baseline.md` action recommandée | Baseline actuelle = 1× itération non fiable |
| 30 | Auditer `RWMutex` du `TransformCache` (LRU mute à chaque hit — Mutex simple ou design CLOCK ?) | Perf | M | S | `42-concurrency.md` finding non-bloquant 1 + Top 5 #3 | Probablement sub-optimal |
| 31 | Mesurer oversubscription cross-package (`fibonacci.globalSem` + `bigfft.concurrencySemaphore`) — ajouter métrique ou unifier | Perf | M | M | `42-concurrency.md` Top 5 #1 | Jusqu'à 2×NumCPU goroutines actives possible |
| 32 | Aligner `make test-short` avec `-race` (ou documenter explicitement la divergence dev/CI) | Tests | M | XS | `42-concurrency.md` Top 5 #4 ; `51-race-detector.md` couplage | |
| 33 | Documenter `ChannelObserver` drop silencieux côté API publique | Doc | F | XS | `42-concurrency.md` Top 5 #5 | Sends en `select default` |
| 34 | Patterns design : aligner `patterns/design-patterns.md` (12) avec ARCH.md (14) — ajouter Bump Allocator, Arena, GC Controller, Dynamic Threshold, Zero-Copy, Generics | Doc | F | S | `61-doc-tech.md` Top 3 #2 | |
| 35 | Renvois `file:line` (style BIGFFT.md) dans FAST_DOUBLING/MATRIX/FFT/GMP/COMPARISON.md + `docs/MAINTAINING.md` | Doc | F | M | `61-doc-tech.md` Top 3 #3 ; `62-doc-go.md` action mineure | |
| 36 | Étoffer `cmd/fibcalc/doc.go` (extraire de `main.go`), `internal/cli/doc.go`, `internal/orchestration/doc.go` (modèle `parallel`) | Doc | F | S | `62-doc-go.md` Top 3 #1+#2 ; `01-inventory.md` ; `22-conventions.md` Top 5 #4 | 21/21 packages avec doc.go (forme) |

## P3 — Backlog

| # | Action | Domaine | Impact | Effort | Source(s) | Notes |
|---|---|---|---|---|---|---|
| 37 | Introduire Mage (`magefile.go`) ou Taskfile pour portabilité Windows native du build | Outillage | M | M | `70-makefile.md` Top 5 #3 | Makefile inutilisable hors WSL/MSYS |
| 38 | Renommer `GOFLAGS`→`LDFLAGS_FLAGS` dans Makefile (collision sémantique env Go) | Outillage | F | XS | `70-makefile.md` Top 5 #4 | |
| 39 | Ajouter cible Makefile `build-gmp` (`-tags gmp`) | Outillage | F | XS | `70-makefile.md` Top 5 #5 ; `02-dependencies.md` build tag gmp | |
| 40 | Étendre `build-pgo-all` à linux/arm64 et windows/arm64 (parité avec `build-all`) | Outillage | F | XS | `72-cross-build.md` action 2 | |
| 41 | Découper `internal/cli/completion.go` (520 LOC) en sous-package `internal/cli/completion/` | Architecture | F | M | `10-architecture.md` recommandation 4 | Pas d'urgence |
| 42 | Évaluer sous-package `internal/fibonacci/framework` (regrouper `doubling_framework.go`, `matrix_framework.go`, `strategy.go`) | Architecture | F | M | `10-architecture.md` recommandation 5 | Non urgent |
| 43 | Évaluer découpage `internal/bigfft/{fft,arith,pool,cache}` (3 316 LOC) | Architecture | F | L | `10-architecture.md` recommandation 6 ; `41-bigfft-arena.md` F6 | Aucune urgence |
| 44 | Documenter convention `errors → format` dans `docs/ARCH.md` | Doc | F | XS | `10-architecture.md` recommandation 7 ; `11-layering.md` Test 5 | Évite régression perçue lors d'audits futurs |
| 45 | Ajouter `t.Parallel()` aux tests des 5 packages en retrait : `cmd/generate-golden`, `internal/sysmon`, `internal/ui`, `internal/metrics`, `test/e2e` | Tests | F | S | `22-conventions.md` Top 5 #1+#2+#3 | Couverture 94 % → 100 % top-level |
| 46 | Convertir blocs `# Example` pseudo-code de `bigfft`, `calibration`, `parallel` en `Example*` exécutables | Doc/Tests | F | S | `62-doc-go.md` Top 3 #3 ; `22-conventions.md` Top 5 #5 | Vérifiés par `go test` |
| 47 | Hygiène git/historique : aligner `user.name`/`user.email` (`.mailmap`) ; nettoyer 4 branches distantes orphelines ; envisager `git filter-repo` pour ~100 Mo de PDFs hérités | Outillage | F | M | `03-git-snapshot.md` obs 3, 5, 6, 8 | Action destructive sur historique — à valider |

---

## Index par domaine

- **CI / Outillage** : 1, 2, 22, 37, 38, 39, 40, 47
- **Sécurité** : 8, 9
- **Architecture** : 7, 11, 12, 41, 42, 43, 44
- **Performance / correctness** : 3, 24, 25, 26, 27, 30, 31
- **Tests** : 4, 5, 10, 23, 28, 29, 32, 45
- **Code quality** : 6, 18, 19, 20, 21
- **Documentation** : 13, 14, 15, 16, 17, 33, 34, 35, 36, 46
- **Licences** : 22 (croisée Outillage)

## Notes

- **Items non retenus** : doublons fusionnés (CI mentionnée dans `50/51/61/70/30` → 1 seule entrée), recommandations purement informatives (audit 1/2 inventaires, audit 41 F7 « corriger rapport amont »).
- **Effort indicatif** : XS (changement chirurgical, 1 fichier), S (revue + tests, 2-8 h), M (refactor multi-package), L (architecture).
- **Dépendances** : action 1 (CI) débloque la vérification de 4, 28, 29, 32, 45 ; action 2 débloque l'exécution effective de la config `.golangci.yml` (24 linters).
