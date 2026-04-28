# 00 — Tableau détaillé des résultats d'audit

> Synthèse agrégée des 24 rapports `01-*.md` à `99-*.md` produits le 2026-04-28.
> Score global : **7,9 / 10** — codebase mature, outillage à industrialiser.

---

## Vue d'ensemble

| Indicateur | Valeur |
| --- | --- |
| Phases | 9 / 9 ✅ |
| Tâches | 24 / 24 ✅ |
| Rapports livrés | 24 dans `docs/audits/` |
| Findings critiques | **0** |
| Findings élevés | **0** |
| Findings modérés | 6 |
| Items backlog | 47 (P0=6, P1=15, P2=15, P3=11) |
| Race detector | ⚠️ non exécuté (CGO indisponible sur hôte Windows) |
| `golangci-lint` | ⚠️ non exécuté (v1.64.8 < toolchain Go 1.26.2) |
| `govulncheck` | ⚠️ non exécuté (pas de réseau) |

---

## Phase 1 — Reconnaissance

| #   | Tâche               | Verdict       | Métriques clés                                                                                                  | Top findings                                                                                              | Source              |
| --- | ------------------- | ------------- | --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ------------------- |
| 1.1 | Cartographie        | ✅ Exhaustif  | 21 packages · 14 949 LOC prod / 21 242 LOC test · ratio test/prod 1,42 · doc.go 20/21                            | Top packages : bigfft 3316 / fibonacci 3179 / tui 1668. Absents : `.github/`, `Dockerfile`, `SECURITY.md`. | `01-inventory.md`   |
| 1.2 | Dépendances         | ✅ Saines     | Go 1.25.0 + toolchain 1.26.2 · 10 directes / 26 indirectes · `go.sum` 692 lignes                                  | Aucun module retired. Build tags `gmp`, `amd64/!amd64`. `ncw/gmp v1.0.5` peu maintenu.                    | `02-dependencies.md`|
| 1.3 | Snapshot Git        | ⚠️ À nettoyer | Working tree propre · 318 commits sur 6 mois · tag unique `v3.0.0`                                              | Bloat historique ~100 MB (PDFs/PNGs purgés du HEAD). 4 alias d'auteur. 4 branches orphelines.             | `03-git-snapshot.md`|

---

## Phase 2 — Architecture

| #   | Tâche                | Verdict      | Métriques clés                                                              | Top findings                                                                                              | Source                |
| --- | -------------------- | ------------ | --------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | --------------------- |
| 2.1 | Clean Architecture   | ✅ Conforme  | 0 cycle (tri topologique sur 21 paquets) · 0 god package strict             | 5 écarts diagrammes vs code dans `dependency-graph.mermaid` (cli→fib, calib→cli, etc.).                   | `10-architecture.md`  |
| 2.2 | Étanchéité couches   | ⚠️ 1 violation | cmd/ → 1 import unique (`internal/app`) · 0 fuite inverse                  | `internal/fibonacci/progress_aliases.go` importe `internal/progress` (domaine → infra).                   | `11-layering.md`      |
| 2.3 | Interfaces ISP       | ✅ Saine     | 13 interfaces · médiane 2 méthodes · 0 interface > 5 méthodes               | `MockCalculator` dupliqué dans 4 packages. `internal/fibonacci/testing.go` pollue l'API publique.         | `12-interfaces.md`    |

---

## Phase 3 — Qualité du code

| #   | Tâche               | Verdict      | Métriques clés                                                                | Top findings                                                                                                  | Source             |
| --- | ------------------- | ------------ | ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------ |
| 3.1 | Qualité (lint stat) | ✅ Très haute | 1 violation funlen · 4 `fmt.Errorf` sans `%w` · 13 panics tous justifiés       | `ExecuteDoublingLoop` 136 lignes (`doubling_framework.go:141`). 0 TODO/FIXME/HACK (signal de maturité).        | `20-code-quality.md` |
| 3.2 | Code mort           | ⚠️ ~80 LOC   | 17 symboles candidats à suppression                                            | `progress_aliases.go` : 12/14 symboles morts. `MockCalculator` dupliqué × 6. `formatBytes` dupliqué.           | `21-dead-code.md`  |
| 3.3 | Conventions FibGo   | ✅ 6,5/7    | t.Parallel() 94 % (620 tests / 802 calls) · 47 Benchmarks / 5 Examples / 5 Fuzz | Couverture `t.Parallel()` faible : `metrics` 15 %, `e2e` 33 %, nulle dans `sysmon`/`ui`/`generate-golden`.    | `22-conventions.md`|

---

## Phase 4 — Sécurité & licences

| #   | Tâche             | Verdict          | Métriques clés                                                          | Top findings                                                                                              | Source            |
| --- | ----------------- | ---------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------- |
| 4.1 | Sécurité OWASP    | ✅ Solide        | 0 critique · 0 élevé · 2 modérés · 0 secret commité · 0 `unsafe.Pointer` prod | M-1 : `-n` (uint64) sans borne sup → DoS local. M-2 : path traversal théorique sur `-output`.            | `30-security.md`  |
| 4.2 | Licences & SBOM   | ✅ 100 % whitelistées | 10/10 directes + 26/26 indirectes vérifiées (`GOMODCACHE`)               | Aucun en-tête de licence dans fichiers Go. `NOTICE` absent. `-tags gmp` lie statiquement libgmp (LGPL).   | `31-licenses.md`  |

---

## Phase 5 — Performance & concurrence

| #   | Tâche                  | Verdict      | Métriques clés                                                          | Top findings                                                                                                                | Source                |
| --- | ---------------------- | ------------ | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| 5.1 | `internal/fibonacci/`  | ✅ 9/10      | 2 `sync.Pool` (5 + 23 big.Int) · stealing pattern · 0 alloc hot paths    | F1 : `GCController.Begin/End` modifient l'état GC global **sans mutex** → corruption en multi-tenant. F2/F4/F8/F9 mineurs.  | `40-fibonacci-perf.md`|
| 5.2 | `internal/bigfft/`     | ✅ Mature    | Bump allocator 242 LOC O(1) · 36 benchmarks · parité amd64/!amd64 OK    | F1 : 4 sites `recover()` utilisent `%v` au lieu de `%w`. Manque bench comparatif vs `math/big.Mul`. Aucune cible Fuzz.       | `41-bigfft-arena.md`  |
| 5.3 | Concurrence            | ✅ Conforme  | 9 spawn sites tous bornés · 3 sémaphores NumCPU · 8 `sync.Pool` équilibrés | `_ = g.Wait()` (`orchestrator.go:78`) vérifié non bloquant (documenté). Risque oversubscription 2×NumCPU sous double sémaphore. | `42-concurrency.md`   |
| 5.4 | Baseline benchmarks    | ✅ 86 mesures | 24 benchs à 0 alloc/op · convergence FFT/FastDoubling à N=10M (~60 ms)    | Top hot : `FFTMulWithBump/1M_words` 307 ms / 695 MB / 304 allocs. Top alloc : `Fibonacci/FFTBased/10M` 4 051 allocs.        | `43-bench-baseline.md`|

---

## Phase 6 — Tests

| #   | Tâche                 | Verdict        | Métriques clés                                                       | Top findings                                                                                              | Source                  |
| --- | --------------------- | -------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------------- |
| 6.1 | Couverture & golden   | ✅ 88,9 %      | 3 722 / 4 185 stmts · 4 packages 100 % · 0 package < 50 %             | `coverage.out` obsolète (`generator_iterative.go` introuvable). Golden ne stresse pas le palier FFT (N max 10 k). | `50-tests-coverage.md`  |
| 6.2 | Race detector         | ⚠️ Non exécuté | `-race` exige CGO ; pas de gcc sur l'hôte Windows · fallback CGO=0 : 22/22 PASS | Documenter dépendance gcc/MSYS2 ; vérifier que la CI Linux exécute bien `-race`.                          | `51-race-detector.md`   |

---

## Phase 7 — Documentation

| #   | Tâche                      | Verdict      | Métriques clés                                                                 | Top findings                                                                                              | Source            |
| --- | -------------------------- | ------------ | ------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------- | ----------------- |
| 7.1 | Doc utilisateur            | ⚠️ 7/10      | README 8,5/10 · CHANGELOG 5/10 · CONTRIBUTING 7/10                              | Flag `--gc-control` absent du README. CHANGELOG s'arrête à `[1.0.0]` alors que tag git = `v3.0.0`. `BUILD.md` mentionne cibles fantômes (`generate-mocks`). | `60-doc-user.md`  |
| 7.2 | Doc technique              | ✅ 8/10      | ARCH.md 785 L · 7 docs algorithmes (2 284 L) · C4 niveaux 1-3                   | `validation-report.md` obsolète (2026-02-08). Patterns 12 vs 14 entre `design-patterns.md` et `ARCH.md`. Renvois `file:line` rares (sauf BIGFFT.md). | `61-doc-tech.md`  |
| 7.3 | `doc.go` par package       | ⚠️ 20/21     | 21/21 ont une doc package-level mais 1 viole la convention                      | `cmd/fibcalc/` met la doc dans `main.go` au lieu d'un `doc.go` dédié. 5 Examples concentrés dans `fibonacci`. | `62-doc-go.md`    |

---

## Phase 8 — Outillage & CI

| #   | Tâche             | Verdict       | Métriques clés                                                  | Top findings                                                                                              | Source            |
| --- | ----------------- | ------------- | --------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------- |
| 8.1 | Makefile & CI     | ⚠️ 7/10       | 34 cibles · `.PHONY` complet · 9/9 cibles CLAUDE.md alignées     | `.github/` **absent** → aucune CI. `BUILD.md` cite `generate-mocks` / `install-mockgen` fantômes. Pas portable hors POSIX. | `70-makefile.md`  |
| 8.2 | Lint              | ❌ Bloquant   | `.golangci.yml` 22 linters · `go vet` 0 finding                  | `golangci-lint v1.64.8` (Go 1.25.5) refuse la config Go 1.26.2 → cible "zéro warning CI" non vérifiable. | `71-lint.md`      |
| 8.3 | Cross-compilation | ✅ 6/6        | linux/windows/darwin × amd64/arm64 · binaire hôte 6,96 MB        | `build-pgo-all` ne couvre pas arm64. `BUILD.md` omet linux/arm64 + windows/arm64 du tableau cross-build. | `72-cross-build.md`|

---

## Phase 9 — Synthèse

| #   | Tâche                  | Verdict      | Métriques clés                                                                  | Top findings                                                                                              | Source                    |
| --- | ---------------------- | ------------ | ------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ------------------------- |
| 9.1 | Synthèse exécutive     | ✅ Score 7,9/10 | Tableau de bord 8 phases · 5 forces · 5 risques · 7 reco 30 j                  | Forte tension code mature ↔ outillage faible. Tous les findings critiques sont d'industrialisation, pas algorithmiques. | `99-executive-summary.md` |
| 9.2 | Backlog priorisé       | ✅ 47 items   | P0 = 6 · P1 = 15 · P2 = 15 · P3 = 11                                            | Top P0 : créer `.github/workflows/ci.yml` ; mettre à jour `golangci-lint` ; sceller `GCController.Begin/End`. | `99-backlog.md`           |

---

## Top 10 forces consolidées

1. **Architecture** propre, sans cycle, alignée sur la doctrine Clean Architecture (`10-architecture.md`).
2. **Performance** : 24 benchmarks à 0 alloc/op, pooling discipliné, arena bump O(1) (`40` + `41` + `43`).
3. **Couverture tests** : 88,9 %, 4 packages à 100 %, aucun < 50 % (`50-tests-coverage.md`).
4. **Sécurité** : 0 critique / 0 élevé, 0 secret commité, supply chain saine (`30-security.md`).
5. **Licences** : 36/36 modules whitelistés Apache 2.0 compatibles (`31-licenses.md`).
6. **Conventions** : t.Parallel() à 94 %, 0 TODO/FIXME, observer pattern complet (`22` + `42`).
7. **Docs algorithmes** : `BIGFFT.md` (701 L), `FFT.md` (221 L), notation rigoureuse (`61-doc-tech.md`).
8. **Cross-build** : 6/6 cibles compilent (`72-cross-build.md`).
9. **Concurrence** : 100 % des goroutines bornées par WG ou errgroup (`42-concurrency.md`).
10. **Maturité** : 0 TODO/FIXME/HACK dans la base prod (`20-code-quality.md`).

---

## Top 10 risques consolidés (par priorité)

| # | Risque | Sévérité | Effort | Source(s) |
| - | ------ | -------- | ------ | --------- |
| 1 | Aucune CI (`.github/` absent) — 4 vérifications dépendantes bloquées | Élevé | M | `70`, `50`, `51`, `71` |
| 2 | `golangci-lint` cassé localement (incompatibilité Go 1.26) | Élevé | XS | `71-lint.md` |
| 3 | `GCController.Begin/End` modifient le GC process-global sans mutex | Moyen | S | `40-fibonacci-perf.md` F1 |
| 4 | `-race` non exécuté (CGO indisponible sur l'hôte d'audit) | Moyen | S | `51-race-detector.md` |
| 5 | DoS local : `-n` non borné supérieurement | Moyen | XS | `30-security.md` M-1 |
| 6 | `progress_aliases.go` viole la couche domaine | Faible | S | `11-layering.md` |
| 7 | `MockCalculator` dupliqué × 6 (dont 1 dans la prod) | Faible | M | `21-dead-code.md`, `12` |
| 8 | CHANGELOG désynchronisé (`[1.0.0]` vs tag `v3.0.0`) | Faible | XS | `60-doc-user.md` |
| 9 | Bloat historique git ~100 MB | Faible | M | `03-git-snapshot.md` |
| 10 | 4 sites `recover()` perdent l'inspection `errors.Is` (`%v` vs `%w`) | Faible | XS | `41-bigfft-arena.md`, `20` |

---

## Top 6 actions P0 (sprint immédiat)

1. **Créer `.github/workflows/ci.yml`** — matrix `{ubuntu, macos, windows} × {go 1.26.2}`, exécutant `go test -race -cover`, `golangci-lint run`, `gosec`, `govulncheck`. Débloque audits 6.1, 6.2, 7.1, 8.1, 8.2.
2. **Mettre à jour `golangci-lint`** vers v2.x compatible Go 1.26.2 et le pinner via `tools.go`.
3. **Sceller `GCController.Begin/End`** : un mutex global sur l'enregistrement de l'ancien `GCPercent`.
4. **Borner l'argument `-n`** côté `internal/config/Validate` (par exemple ≤ 100 M par défaut).
5. **Régénérer `coverage.out`** après suppression du fichier obsolète et l'ajouter à `.gitignore`.
6. **Synchroniser `CHANGELOG.md`** avec le tag `v3.0.0` (entrée datée 2026-04-18).

> Détail complet et items P1/P2/P3 → `99-backlog.md` (47 items au total).

---

## Limites de l'audit (hors-scope)

| Vérification non exécutée | Raison                                                       | Comment compenser                          |
| ------------------------- | ------------------------------------------------------------ | ------------------------------------------ |
| `go test -race`           | CGO requis ; pas de gcc sur l'hôte Windows                  | CI Linux ou MSYS2/UCRT64                   |
| `golangci-lint run` complet | Version 1.64.8 incompatible Go 1.26.2                        | Mise à jour v2.x puis rerun                |
| `govulncheck`             | Pas d'accès réseau pendant l'audit                           | Lancer dans pipeline CI                    |
| Benchmarks fiables        | `-benchtime=1x` (mode rapide d'audit)                        | Ré-exécuter `-benchtime=3s -count=5` + `benchstat` |
| Build `-tags gmp`         | Toolchain C cross-compile non installée                      | À tester sur Linux ou via Docker           |

---

*Rapport généré automatiquement par 24 agents `general-purpose` orchestrés selon `PLAN.md`. Voir le journal d'exécution §3 du plan pour la traçabilité complète.*
