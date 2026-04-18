# AUDIT_REPORT — FibGo / FibCalc

**Date** : 2026-04-18
**Mode d'exécution** : Agent Teams (PRD.md)
**Branche** : `main`
**Scope** : audit exhaustif **read-only** (aucune modification de code)
**Coordinateur** : Claude (consolidation 6 rapports `bench/TEAM_*.md`)

---

## 1. Résumé exécutif

FibGo est un codebase Go mature (~35 600 lignes, 108 sources, 97 tests, 22 packages, couverture pondérée **87.5 %**). Les 6 équipes confirment **une posture solide globale** : Clean Architecture respectée (zéro violation de couches), zéro vulnérabilité `govulncheck`, zéro panic non justifié en code prod, suite verte (22 packages OK), `golangci-lint` aligné sur CLAUDE.md (22 linters, seuils cyclo/cognitive/funlen identiques). Les findings sont **du polish chirurgical**, pas un redesign.

**Total : 60 findings — P0 = 9 · P1 = 25 · P2 = 26** (après dédup inter-équipes).

### Points saillants à corriger en priorité

1. **Performance (F-A1)** : `internal/bigfft/fft_poly.go` — buffers `sync.Pool` jamais relâchés ⇒ 64 % du trafic alloc FFT (~1 GB/calcul F(10⁶)). Patch ROI extrême.
2. **Contrat Go (F-F8)** : `go.mod` déclare `go 1.24.3` ; CLAUDE.md exige 1.25.0+. Aucune directive `toolchain`.
3. **Dépendances (F-F13)** : 6 directes obsolètes dont **majeure** `bubbles v0.21 → v1.0` (rupture API potentielle TUI), `golang.org/x/text` 28 versions de retard.
4. **Documentation cassée (F-D1/D2/D3/D4)** : 4 fichiers `.md` référencés mais **inexistants** (`docs/INNOVATION.md`, `docs/INNOVEPLAN.md`, `docs/architecture/patterns/design-patterns.md`, et 4 sous-fichiers `flows/*.md`).
5. **Sécurité errgroup (B-1/E-3)** : `g.Wait()` ignoré dans `orchestrator.go:67` — sémantique errgroup sous-utilisée, cancellation cross-calculator absente.
6. **Hygiène repo (E-8/F-9)** : 4 fichiers `*_err.txt`/`*_out.txt` trackés à la racine, non couverts par `.gitignore`.

### Verdict global

| Dimension | État | Commentaire |
|-----------|------|-------------|
| Architecture | ✅ Très bonne | 0 violation Clean Arch, ISP respecté partout |
| Performance | ⚠️ 1 P0 + 5 P1 | Pool leak FFT majeur, sinon optimisations marginales |
| Tests | ✅ Bonne | 87.5 % couverture, 0 FAIL ; améliorer `internal/tui` parallélisme |
| Documentation | ⚠️ Plusieurs liens cassés | 4 fichiers fantômes, 5 `doc.go` manquants |
| Sécurité | ✅ Bonne | 0 vuln, panics justifiés, mono-user CLI |
| Build local | ⚠️ Polish | `-trimpath` absent, PGO morte, deps obsolètes |

---

## 2. Tableau consolidé des findings (priorisé P0 → P2)

### P0 — Critique (9 findings)

| ID | Catégorie | Fichier | Effort | Impact | Source |
|----|-----------|---------|--------|--------|--------|
| **P0-01** | Performance | `internal/bigfft/fft_poly.go:158-400` (pool leak Transform/InvTransform/mul/sqr) | M | -60 % allocs FFT, -10 % CPU | A-1 |
| **P0-02** | Build/contrat | `go.mod:3` (`go 1.24.3` vs CLAUDE.md 1.25+, pas de `toolchain`) | S | dérive silencieuse Go 1.26 | F-8 |
| **P0-03** | Dépendances | `go.mod` (6 directes obsolètes, dont **`bubbles v0.21 → v1.0` majeure**) | M-L | rupture API TUI, sécurité | F-13 |
| **P0-04** | Doc cassée | `README.md:193,543` ; `docs/ARCH.md:3,766` (`INNOVATION.md`/`INNOVEPLAN.md` inexistants) | S | confusion lecteurs, ADR-010 fantôme | D-1 |
| **P0-05** | Doc cassée | `docs/TUI_GUIDE.md:7,391` ; `docs/algorithms/BIGFFT.md:694` ; validation-report.md (`patterns/design-patterns.md` inexistant) | S | doc orpheline | D-2 |
| **P0-06** | Doc cassée | `docs/architecture/validation/validation-report.md:148-156` (annonce 4 `flows/*.md` absents) | S | rapport trompeur | D-3 |
| **P0-07** | Doc cassée | `docs/architecture/validation/validation-report.md:170` (`docs/calibration/CALIBRATION.md` n'existe pas — réel : `docs/CALIBRATION.md`) | S | lien mort | D-4 |
| **P0-08** | Sécurité/erreur | `internal/orchestration/orchestrator.go:67` (`g.Wait()` ignoré, errgroup sous-utilisé) | S | cancellation cross-calculator absente | B-1 / E-3 |
| **P0-09** | Pool FFT (lié P0-01) | `internal/bigfft/fft_poly.go:158,209,339,377` (pas de `release*` non-test hors legacy) | M | confirmé par profiling pprof | A-1 |

### P1 — Important (25 findings)

| ID | Catégorie | Fichier | Effort | Impact | Source |
|----|-----------|---------|--------|--------|--------|
| P1-01 | Perf | `fibonacci/fft.go:106-113` (BumpAllocator non utilisé pour Transform) | S | -40 % allocs Transform séq | A-2 |
| P1-02 | Perf | `fibonacci/doubling_framework.go:230-245` (cache.Stats RLock × 24 itérations) | S | -3-5 % CPU | A-3 |
| P1-03 | Perf | `fibonacci/generator_iterative.go:113,115` (`new(big.Int)` × 2 par itération) | S | -75 % allocs `First1000` | A-4 |
| P1-04 | Perf | `fibonacci/fastdoubling.go:103` + `memory/arena.go` (arena non poolée, 13 % alloc) | S | -10 % B/op, -3-5 % CPU | A-5 |
| P1-05 | Refactor | `fibonacci/common.go:97-144` (triplication `executeParallel3`, ~40 LoC) | S | dette DRY | B-2 |
| P1-06 | Refactor | `fibonacci/fastdoubling.go:100-115` + `fft_based.go:53-66` (dup arena pre-size) | S | dette DRY | B-3 |
| P1-07 | Code mort | `calibration/microbench.go:169,208` ; `cli/presenter.go:92` ; `config/env.go:19` ; `fibonacci/fft.go:86` (5 `unparam`) | S×5 | clean-up signaux lint | B-5 |
| P1-08 | Refactor | `fibonacci/matrix_types.go:154-170` (`releaseMatrixState` cyclo 24 → 4 via helper) | S | passe seuil lint 15 | B-8 |
| P1-09 | Hygiène | ~80 fichiers (`gofmt -s` non passé, baseline lint:803-1099) | S | dette de format | B-9 |
| P1-10 | Tests | `internal/tui/{bridge,chart,footer,header,keymap,logs,metrics,model,sparkline}_test.go` (122 tests sans `t.Parallel()`) | M | durée suite, debug parallélisme | C-1 |
| P1-11 | Code mort | `internal/config/thresholds.go:17,33,74,99` (4 fonctions à 0 % cov, dup probable de `calibration/adaptive.go`) | S→M | suppression possible | C-3 |
| P1-12 | Doc impl | `docs/PERFORMANCE.md:159` (`FIBCALC_GC_CONTROL` documenté sans implémentation) | S (doc) ou M (impl) | UX trompeuse | D-5 |
| P1-13 | Doc env | `README.md` + `BUILD.md` + `.env.example` (`FIBCALC_TUI_THEME` implémenté non documenté) | S | feature cachée | D-6 |
| P1-14 | Doc env | `.env.example` (manque `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT`, `FIBCALC_TUI_THEME`) | S | source de vérité incohérente | D-7 |
| P1-15 | Doc obsolète | `docs/TESTING.md:281,283` (chemins post-extraction sous-packages incorrects) | S | navigation cassée | D-8 |
| P1-16 | Doc fausse | `docs/BUILD.md:207` ("24 linters" — réel 22) | S | chiffre faux | D-9 |
| P1-17 | Doc | `CHANGELOG.md:[Unreleased]` (8 commits perf récents non listés : FFT cache, NTransform, ExecutionConfig, sec fix, etc.) | S | release notes incomplètes | D-13 |
| P1-18 | Doc package | `internal/{format, metrics, progress, fibonacci/memory, fibonacci/threshold}/` (5 packages sans `doc.go`) | S | viole CLAUDE.md #5 | D-15 |
| P1-19 | Sécurité | `fibonacci/common.go:215,265,276` (`executeTasks`/`executeMixedTasks` sans `ctx.Err()` post-sem) | M | cancellation utilisateur retardée | E-6 |
| P1-20 | Hygiène repo | `build_err.txt`, `e2e_rich_out.txt`, `test_err.txt`, `test_out.txt` (trackés, non gitignorés) | S | fuite logs dev | E-8 / F-9 |
| P1-21 | Build | `Makefile:22-26` (`-trimpath` absent → chemins absolus dans binaires) | S | builds non reproductibles | F-4 |
| P1-22 | PGO | `cmd/fibcalc/default.pgo` absent + `.gitignore *.pgo` (branche PGO morte) | M | gain PGO 2-5 % perdu | A-8 / F-5 |
| P1-23 | Sécurité lint | `.golangci.yml:42` (gosec actif, 21 findings G115 ignorés silencieusement) | S | dette gosec invisible | F-11 |
| P1-24 | Dépendances | `golang.org/x/term v0.36→v0.42`, `x/text v0.8→v0.36` (28 versions !), `x/sync`, `x/sys`, `zerolog`, `gopsutil` | M | sécurité, perf | F-13 |
| P1-25 | Refactor | `internal/cli/presenter.go` (7 méthodes adapter à 0 % cov, wrappers triviaux) | S | F-B7 + F-C2 | B-7 / C-2 |
| P1-26 | Tests | `cmd/generate-golden` (29.4 % cov, `targets[]` codés en dur, manque cas FFT-bound n>700k) | S | golden ne couvre pas FFT | C-4 |

### P2 — Mineur (26 findings)

| ID | Catégorie | Source | Note |
|----|-----------|--------|------|
| P2-01 | Perf | A-6 | `WithOptimizedCache` 20 % plus lent que default — anomalie tuning |
| P2-02 | Perf | A-7 | Sémaphores indépendants → sur-souscription NumCPU×3 |
| P2-03 | Perf | A-9 | `wg`/`ec` escape to heap dans `executeParallel3` |
| P2-04 | Refactor | B-4 | `cmd/generate-golden/fibBig` ↔ `internal/fibonacci/calculateSmall` (justifier dup oracle) |
| P2-05 | Refactor | B-6 | `calibration/calibration.go:65` (`RunCalibrationWithOptions` 61 statements, cohésion mixte) |
| P2-06 | Tests | C-5 | `internal/fibonacci/testing.go` (TestFactory exporté, 0 % cov) |
| P2-07 | Tests | C-6 | `Makefile:271-276` (cibles `generate-mocks`/`install-mockgen` inertes) |
| P2-08 | Tests | C-8 | Aucun corpus seed persistant pour 5 fuzz targets |
| P2-09 | Sécurité | E-1 | 21 G115 (overflow conversions, faux positifs 64-bit) |
| P2-10 | Sécurité | E-2 | 2 G304 (path traversal, non exploitable mono-user CLI) |
| P2-11 | Sécurité | E-4 | `tw.Flush()` ignoré `calibration/io.go:38` |
| P2-12 | Sécurité | E-5 | `fourier()` retour erreur ignoré `fft_poly.go:297,315` |
| P2-13 | Sécurité | E-7 | `panic()` `MustGet`/`MustNewCalculator` (convention Go OK, garder) |
| P2-14 | Doc | D-10 | `Claude.md` stats périmées (103/89 → 108/97 fichiers) |
| P2-15 | Doc | D-11 | `Claude.md` arbre `internal/` ne reflète pas sous-packages |
| P2-16 | Doc | D-12 | `docs/ARCH.md:12` stats divergentes (19 packages → 22) |
| P2-17 | Doc | D-14 | `CONTRIBUTING.md` ↔ `TESTING.md` se contredisent sur mockgen |
| P2-18 | Doc | D-16 | `doc.go` triviaux (`bigfft`, `calibration`, `app`, `parallel`, `testutil`) |
| P2-19 | Doc | D-17 | Benchmarks README (Intel) ↔ PERFORMANCE.md (Ryzen) — ~30× écart F(10M) |
| P2-20 | Doc | D-18 | Badge coverage `80%` README hardcodé |
| P2-21 | Doc | D-19 | `docs/architecture/README.md` annonce `Patterns/` riche, dossier vide |
| P2-22 | Doc | D-20 | `dependency-graph.mermaid` manque `progress`, `memory`, `threshold` |
| P2-23 | Build | F-1 | `.PHONY` Makefile incomplet (10 cibles manquantes) |
| P2-24 | Build | F-6 | `make help` non portable Windows (`column`, `sed`) |
| P2-25 | Build | F-7 | Cross-compil `linux/arm64` et `windows/arm64` absents |
| P2-26 | Build | F-10 / F-12 | `bench-versioned`/`pgo-profile` POSIX-only ; `go.mod` non `tidy` (x/sync isolé) |

### P3 — Cosmétique (3 findings, hors quota P0/P1/P2)

| ID | Source | Note |
|----|--------|------|
| P3-01 | C-7 | `internal/tui/logs_test.go:283` (`t.Skip("viewport...")` peut devenir skip permanent) |
| P3-02 | C-9 | `internal/cli/ui_advanced_test.go:43` (sleep fixe 50 ms) |
| P3-03 | C-10 | `bench-versioned` ne benchmark que `BenchmarkFastDoubling` |

---

## 3. Roadmap proposée (3 phases)

### Phase 1 — Quick wins & hygiène (1-2 jours, faible risque)

Objectif : nettoyer les évidences avant les changements de fond.

1. **`gofmt -s -w ./...`** (P1-09) — commit isolé.
2. **`.gitignore` + `git rm`** des 4 `*_err.txt`/`*_out.txt` racine (P1-20).
3. **Compléter `.PHONY`** Makefile (P2-23).
4. **`-trimpath`** dans Makefile LDFLAGS (P1-21).
5. **Stats périmées doc** : `Claude.md` 103/89 → 108/97, `docs/ARCH.md`, `BUILD.md` "24 linters" → 22 (P1-16, P2-14, P2-15, P2-16).
6. **Liens cassés docs** P0 quick fixes (P0-04, P0-05, P0-06, P0-07).
7. **`CHANGELOG.md` `[Unreleased]`** : ajouter les 8 commits perf récents (P1-17).

**Livrables** : 2-3 PRs distinctes, pas de risque régression.

### Phase 2 — Sécurité, contrat Go & dépendances (3-5 jours)

Objectif : remettre à niveau le contrat Go et corriger les findings sécurité.

1. **`go.mod` 1.25 + `toolchain`** (P0-02) — revalider tests sur 1.25.
2. **Mises à jour deps mineures/patch** (P1-24) : `x/sync`, `x/sys`, `x/term`, `x/text`, `zerolog`, `gopsutil`.
3. **`bubbles v0.21 → v1.0`** (P0-03) : à isoler dans une PR dédiée + smoke-test TUI complet.
4. **`g.Wait()` orchestrator** (P0-08) : soit `_ = g.Wait()` documenté, soit refactor pour exploiter cancellation errgroup.
5. **`gosec` excludes G115** (P1-23) explicite + commentaire.
6. **Contexte propagé** `executeTasks`/`executeMixedTasks` (P1-19).
7. **5 `doc.go` manquants** (P1-18).

**Livrables** : 4-6 PRs.

### Phase 3 — Performance, refactor chirurgical & docs (5-7 jours)

Objectif : matérialiser les gains identifiés par Team A et nettoyer l'architecture.

1. **F-A1 pool leak FFT** (P0-01/P0-09) : ajout `Release()` sur `PolValues`/`Poly` + defers call-sites. **Bench A/B obligatoire `benchstat`** sur F(10⁶), F(10⁷).
2. **F-A4 IterativeGenerator + `NextInto`** (P1-03).
3. **F-A5 arena pool** (P1-04).
4. **F-A2 BumpAllocator dans `executeDoublingStepFFT`** (P1-01).
5. **F-A3 cache stats throttling** (P1-02).
6. **F-A8 PGO** : générer + commit `default.pgo`, ajuster `.gitignore` (P1-22).
7. **Refactos B** : `executeParallel3` dédup (P1-05), arena pre-size dédup (P1-06), `releaseMatrixState` cyclo (P1-08), 5 `unparam` (P1-07), `CLIProgressReporter` wrapper (P1-25).
8. **Tests** : `t.Parallel()` `internal/tui` (P1-10), suppression `internal/config/thresholds.go` si confirmé mort (P1-11), tests `cmd/generate-golden` (P1-26).
9. **Documentation** : env vars (P1-12, P1-13, P1-14), chemins TESTING.md (P1-15), condensation README (annexe Team D).

**Livrables** : 8-10 PRs, gains perf mesurés et documentés.

### Reste P2/P3

À traiter en *backlog continuous improvement* — ROI individuel modeste mais cumulés améliorent la qualité.

---

## 4. Annexes

### 4.1 Rapports partiels (sources)

- [bench/TEAM_A_PERFORMANCE.md](bench/TEAM_A_PERFORMANCE.md) — 9 findings, profiling pprof exécuté
- [bench/TEAM_B_ARCHITECTURE.md](bench/TEAM_B_ARCHITECTURE.md) — 9 refactos chirurgicales, schéma deps Mermaid
- [bench/TEAM_C_TESTS.md](bench/TEAM_C_TESTS.md) — matrice couverture 22 packages, 5 fuzz audités
- [bench/TEAM_D_DOCS.md](bench/TEAM_D_DOCS.md) — 19 .md + 9 .mermaid + matrice doc.go par package
- [bench/TEAM_E_SECURITY.md](bench/TEAM_E_SECURITY.md) — `govulncheck` 0, `gosec` 27, panic inventaire
- [bench/TEAM_F_TOOLING.md](bench/TEAM_F_TOOLING.md) — Makefile (34 cibles), `.golangci`, `go.mod`

### 4.2 Baseline capturée

- [bench/baseline/test.txt](bench/baseline/test.txt) — 22 OK / 0 FAIL (race off : pas de gcc Windows)
- [bench/baseline/coverage.txt](bench/baseline/coverage.txt) + `coverage.out` — 87.5 % pondéré
- [bench/baseline/benchmark.txt](bench/baseline/benchmark.txt) — 727 lignes
- [bench/baseline/lint.txt](bench/baseline/lint.txt) — 1 531 lignes, 201 findings

### 4.3 Métriques codebase (mesure 2026-04-18)

| Indicateur | Valeur | Source |
|------------|--------|--------|
| Lignes Go | ~35 560 | mesure post-extraction sous-packages |
| Fichiers source (non-test) | 108 | `find -name '*.go' -not -name '*_test.go'` |
| Fichiers test | 97 | `find -name '*_test.go'` |
| Packages Go | 22 | `go list ./...` |
| Couverture pondérée | 87.5 % | `go tool cover -func=coverage.out` |
| Tests passants | 1 553 sous-tests | baseline `test.txt` |
| `t.Parallel()` | 664 | grep |
| Fuzz targets | 5 | tous dans `internal/fibonacci/` |
| `panic()` prod | 13 | tous justifiés (Must* / invariants bigfft) |
| Linters actifs | 22 | `.golangci.yml` |
| Findings lint baseline | 201 | gocritic 66, revive 42, errcheck 29, gosec 21, ... |
| Vulnérabilités govulncheck | 0 | exit 0 |
| Cibles Makefile | 34 | dont 10 manquantes au `.PHONY` |
| Dépendances directes obsolètes | 6 (1 majeure) | `go list -u -m all` |

### 4.4 Limitations de cet audit

- **Race detector désactivé** sur la machine d'audit (Windows, pas de gcc/cgo) — à revalider sur une machine Linux/macOS lors de l'exécution Phase 2.
- **Bench en mode warm-up court** pour le profiling (300-500 ms) ; les chiffres définitifs A/B doivent être capturés via `benchstat` `-count=10 -benchtime=2s` sur machine dédiée.
- **Aucune modification de code** effectuée — toutes les actions sont à valider avant la Phase 4 d'exécution (PRD §4).

---

## 5. Validation requise

Conformément au PRD §3 (Coordinateur, Phase 3), **ce rapport doit être présenté à l'utilisateur pour validation explicite avant toute modification de code source**. Les phases d'exécution (Phase 4 du PRD) sont hors scope du livrable actuel.

**Prochaine action suggérée** : valider la roadmap (Phase 1/2/3) ou redéfinir les priorités, puis lancer la Phase 4 par sous-agent en isolation `worktree` pour les findings P0.
