# EXECUTION_PLAN — Implémentation des 60 findings FibGo (Mode Agent Teams)

**Projet** : FibGo / FibCalc
**Date** : 2026-04-18
**Branche mère** : `main`
**Branche d'exécution** : `claude/audit-execution-20260418`
**Source** : [AUDIT_REPORT.md](AUDIT_REPORT.md) · [PRD.md](PRD.md)
**Mode** : Agent Teams parallèles (6 équipes) + Coordinateur
**Scope** : 60 findings (9 P0 · 25 P1 · 26 P2) + 3 P3 cosmétiques + mise à jour README + nettoyage repo

---

## ⚡ Instructions d'exécution (à lire avant toute action)

### Instruction A — Exécution via Agent Teams dans Claude Code

**Ce plan DOIT être exécuté intégralement via les Agent Teams de Claude Code** (outil `Agent` avec sous-agents). Le Coordinateur (conversation principale) orchestre, chaque team est déléguée à un sous-agent parallèle ou séquentiel selon les dépendances (cf. §2).

**Procédure de lancement (à suivre phase par phase)** :

1. **Phase 0 (Coordinateur, séquentiel)** : créer la branche `claude/audit-execution-20260418`, figer la baseline (`make test`, `make coverage`, `make benchmark`, `make lint`) dans `Audit/bench/exec-baseline/`.
2. **Phases 1 → 6 (délégation Agent)** : pour chaque sous-phase, le Coordinateur envoie **un seul message multi-Agent** contenant une invocation `Agent` par team indépendante. Exemple Phase 1 :
   ```
   Agent(subagent_type="general-purpose", description="Team F Phase 1.1-1.7", prompt="<template §5.2 pour Team F>")
   Agent(subagent_type="general-purpose", description="Team D Phase 1.8-1.11", prompt="<template §5.2 pour Team D>")
   ```
3. **Findings risqués (P0, `bigfft`, `go.mod`, `bubbles v1.0`)** : lancement avec `isolation: "worktree"` obligatoire.
4. **Findings séquentiels (Phase 3 perf)** : un Agent à la fois, avec `benchstat` committé avant la PR suivante.
5. **Chaque Agent produit** : code modifié + tests + commit conforme à §5.2 + résumé ≤ 20 lignes renvoyé au Coordinateur.
6. **Coordinateur vérifie** avant merge : `make test`, `make lint`, diff conforme au finding, et marque la ligne à ✅ dans le tableau ci-dessous.

**Règle stricte** : aucune modification de code en dehors du cadre Agent Teams. Aucune action manuelle hors délégation / revue.

### Instruction B — Mise à jour du suivi à chaque tâche terminée

**À chaque fois qu'une tâche (ligne) du tableau §A ci-dessous est exécutée avec succès** (PR mergée, tests verts, gate de phase passée), le Coordinateur **DOIT** :

1. Passer le statut de `⏸️` à `✅` dans le tableau de suivi global (§A) **ET** dans la matrice de traçabilité §4.
2. Ajouter une entrée datée au **Journal d'exécution §8** : `| YYYY-MM-DD HH:MM | <phase> | PR#<n> | <finding-id> | Team <X> | Merged: <sha court> |`.
3. Si un finding est reporté ou abandonné : statut `⏭️` (skip justifié) ou `❌` (bloqué), avec justification dans le journal.
4. Committer ces mises à jour dans `Audit/EXECUTION_PLAN.md` (même PR que le finding ou PR suivante dédiée au suivi — au plus tard en fin de phase).

**Légende statuts** : `⏸️` En attente · `⏳` En cours · `✅` Terminé · `❌` Bloqué · `⏭️` Skipped (justifié) · `⚠️` Partiel

---

## A. Tableau de suivi global des phases

Ce tableau est **la source de vérité** de l'avancement. À maintenir à jour selon l'Instruction B.

### Phase 0 — Préparation

| Tâche | Description | Owner | Statut | Horodatage |
|-------|-------------|-------|--------|------------|
| 0.1 | Créer branche `claude/audit-execution-20260418` | Coord. | ⏸️ | — |
| 0.2 | Baseline `make test` → `Audit/bench/exec-baseline/test.txt` | Coord. | ⏸️ | — |
| 0.3 | Baseline `make coverage` → `coverage.txt` + `coverage.out` | Coord. | ⏸️ | — |
| 0.4 | Baseline `make benchmark -count=10 -benchtime=2s` | Coord. | ⏸️ | — |
| 0.5 | Baseline `make lint` → `lint.txt` | Coord. | ⏸️ | — |
| 0.6 | `go install golang.org/x/perf/cmd/benchstat@latest` | Coord. | ⏸️ | — |
| 0.7 | Commit initial `chore: open execution branch` | Coord. | ⏸️ | — |

### Phase 1 — Quick wins & hygiène (PR#1 à #5)

| Tâche | Finding | Action | Team | PR | Statut | Horodatage |
|-------|---------|--------|------|----|--------|------------|
| 1.1 | P1-09 | `gofmt -s -w ./... && goimports -w ./...` | F | #1 | ⏸️ | — |
| 1.2 | P1-20 | `.gitignore` + `git rm` 4 fichiers `*_err/*_out.txt` | F | #2 | ⏸️ | — |
| 1.3 | P2-23 | `.PHONY` Makefile (10 cibles manquantes) | F | #3 | ⏸️ | — |
| 1.4 | P1-21 | `-trimpath` dans LDFLAGS Makefile | F | #3 | ⏸️ | — |
| 1.5 | P2-24 | `make help` portable (awk POSIX) | F | #3 | ⏸️ | — |
| 1.6 | P2-25 | `build-all` : `linux/arm64`, `windows/arm64` | F | #3 | ⏸️ | — |
| 1.7 | P2-26 | `go mod tidy` + tag POSIX-only targets | F | #4 | ⏸️ | — |
| 1.8 | P0-04/05/06/07 | Liens Markdown cassés corrigés | D | #5 | ⏸️ | — |
| 1.9 | P1-16 | `BUILD.md` "24 → 22 linters" | D | #5 | ⏸️ | — |
| 1.10 | P2-14/15/16 | Stats `Claude.md`, `docs/ARCH.md` | D | #5 | ⏸️ | — |
| 1.11 | P2-20 | Badge coverage README dynamique/87.5% | D | #5 | ⏸️ | — |

### Phase 2 — Contrat Go, sécurité & deps (PR#6 à #14)

| Tâche | Finding | Action | Team | PR | Statut | Horodatage |
|-------|---------|--------|------|----|--------|------------|
| 2a.1 | P0-02 | `go.mod` Go 1.25 + `toolchain` | F | #6 | ⏸️ | — |
| 2b.1 | P1-24 | Deps mineures (`x/sync`, `x/sys`, `x/term`, `x/text`, `zerolog`, `gopsutil`) | F | #7 | ⏸️ | — |
| 2b.2 | P0-03 | `bubbles v0.21 → v1.0` + TUI smoke-test | F | #8 | ⏸️ | — |
| 2c.1 | P0-08 | `orchestrator.go:67` errgroup cancellation | E | #9 | ⏸️ | — |
| 2c.2 | P1-19 | `ctx.Err()` post-sémaphore `common.go` | E | #10 | ⏸️ | — |
| 2c.3 | P1-23 | `.golangci.yml` gosec G115 whitelist explicite | E+F | #11 | ⏸️ | — |
| 2c.4 | P2-11 | `calibration/io.go:38` `tw.Flush()` err | E | #12 | ⏸️ | — |
| 2c.5 | P2-12 | `fft_poly.go:297,315` `fourier()` err | E | #12 | ⏸️ | — |
| 2c.6 | P2-09 | 21 G115 casts audités individuellement | E | #13 | ⏸️ | — |
| 2c.7 | P2-10 | 2 G304 documentés en excludes | E | #13 | ⏸️ | — |
| 2c.8 | P2-13 | Convention `Must*` documentée (doc.go) | E | #14 | ⏸️ | — |
| 2c.9 | P1-18 | 5 `doc.go` manquants créés | D+E | #14 | ⏸️ | — |

### Phase 3 — Performance (PR#15 à #23)

| Tâche | Finding | Action | Team | PR | Statut | Horodatage |
|-------|---------|--------|------|----|--------|------------|
| 3.1 | P0-01 + P0-09 | FFT pool leak : `Release()` sur `PolValues`/`Poly` | A | #15 | ⏸️ | — |
| 3.2 | P1-01 | BumpAllocator pour `executeDoublingStepFFT` | A | #16 | ⏸️ | — |
| 3.3 | P1-02 | Throttling `cache.Stats()` doubling_framework | A | #17 | ⏸️ | — |
| 3.4 | P1-03 | `NextInto(dst)` generator_iterative | A | #18 | ⏸️ | — |
| 3.5 | P1-04 | Arena pool `fastdoubling.go` | A | #19 | ⏸️ | — |
| 3.6 | P1-22 | PGO : `default.pgo` committé + cible CI | A | #20 | ⏸️ | — |
| 3.7 | P2-01 | Anomalie `WithOptimizedCache` -20% | A | #21 | ⏸️ | — |
| 3.8 | P2-02 | Sémaphore global partagé NumCPU×1 | A | #22 | ⏸️ | — |
| 3.9 | P2-03 | Escape analysis `executeParallel3` | A | #23 | ⏸️ | — |

### Phase 4 — Refactor & tests (PR#24 à #35)

| Tâche | Finding | Action | Team | PR | Statut | Horodatage |
|-------|---------|--------|------|----|--------|------------|
| 4a.1 | P1-05 | Helper `executeParallel3` dédup | B | #24 | ⏸️ | — |
| 4a.2 | P1-06 | Helper arena pre-size dédup | B | #24 | ⏸️ | — |
| 4a.3 | P1-08 | `releaseMatrixState` cyclo 24 → 4 | B | #25 | ⏸️ | — |
| 4a.4 | P1-07 | 5 `unparam` nettoyés | B | #26 | ⏸️ | — |
| 4a.5 | P1-25 | `presenter.go` wrappers triviaux | B | #27 | ⏸️ | — |
| 4a.6 | P2-04 | Doc duplication volontaire oracle | B | #28 | ⏸️ | — |
| 4a.7 | P2-05 | `RunCalibrationWithOptions` split | B | #29 | ⏸️ | — |
| 4b.1 | P1-10 | 122 × `t.Parallel()` TUI tests | C | #30 | ⏸️ | — |
| 4b.2 | P1-11 | `thresholds.go` code mort supprimé/testé | C | #31 | ⏸️ | — |
| 4b.3 | P1-26 | Tests `generate-golden` FFT-bound n>700k | C | #32 | ⏸️ | — |
| 4b.4 | P2-06 | `TestFactory` privé ou couvert | C | #33 | ⏸️ | — |
| 4b.5 | P2-07 | `generate-mocks`/`install-mockgen` supprimés | C | #33 | ⏸️ | — |
| 4b.6 | P2-08 | `testdata/fuzz-seed/` corpus | C | #34 | ⏸️ | — |
| 4b.7 | P3-01 | `logs_test.go:283` skip viewport traité | C | #35 | ⏸️ | — |
| 4b.8 | P3-02 | `ui_advanced_test.go:43` sleep → sync | C | #35 | ⏸️ | — |
| 4b.9 | P3-03 | `bench-versioned` étendu | C | #35 | ⏸️ | — |

### Phase 5 — Documentation & README (PR#36 à #43)

| Tâche | Finding | Action | Team | PR | Statut | Horodatage |
|-------|---------|--------|------|----|--------|------------|
| 5a.1 | P1-12 | `FIBCALC_GC_CONTROL` implémenté ou retiré | D | #36 | ⏸️ | — |
| 5a.2 | P1-13 | `FIBCALC_TUI_THEME` documenté partout | D | #37 | ⏸️ | — |
| 5a.3 | P1-14 | `.env.example` complété (3 vars) | D | #37 | ⏸️ | — |
| 5a.4 | P1-15 | `docs/TESTING.md` chemins mis à jour | D | #38 | ⏸️ | — |
| 5a.5 | P2-17 | `CONTRIBUTING.md` ↔ `TESTING.md` mockgen | D | #38 | ⏸️ | — |
| 5a.6 | P2-18 | 5 `doc.go` triviaux enrichis | D | #39 | ⏸️ | — |
| 5a.7 | P2-19 | Benchmarks README ↔ PERFORMANCE.md | D | #40 | ⏸️ | — |
| 5a.8 | P2-21 | `docs/architecture/patterns/` créé ou retiré | D | #41 | ⏸️ | — |
| 5a.9 | P2-22 | `dependency-graph.mermaid` enrichi | D | #42 | ⏸️ | — |
| 5b.1 | P1-17 | `CHANGELOG.md [Unreleased]` récap complet | D | #43 | ⏸️ | — |
| 5b.2 | README | Réécriture condensée (≤ 22 Ko) | D | #43 | ⏸️ | — |
| 5b.3 | P2-19/20 | Benchs + badge cohérents dans README | D | #43 | ⏸️ | — |

### Phase 6 — Nettoyage final (PR#44 à #45)

| Tâche | Description | Team | PR | Statut | Horodatage |
|-------|-------------|------|----|--------|------------|
| 6.1 | Exécution 5 commandes détection fichiers orphelins | Coord. | — | ⏸️ | — |
| 6.2 | `git rm` fichiers orphelins confirmés | Coord. | #44 | ⏸️ | — |
| 6.3 | `.gitignore` final complet | Coord. | #44 | ⏸️ | — |
| 6.4 | Dossier `Audit/` → `docs/audits/2026-04/` (Option A) | Coord. | #45 | ⏸️ | — |
| 6.5 | Tag release `v3.0.0` sur merge final | Coord. | — | ⏸️ | — |

### Synthèse — Tableau de bord

| Phase | Tâches | Terminées | Statut global |
|-------|--------|-----------|---------------|
| 0 — Préparation | 7 | 0 / 7 | ⏸️ |
| 1 — Quick wins | 11 | 0 / 11 | ⏸️ |
| 2 — Go/Sécu/Deps | 12 | 0 / 12 | ⏸️ |
| 3 — Performance | 9 | 0 / 9 | ⏸️ |
| 4 — Refactor/Tests | 16 | 0 / 16 | ⏸️ |
| 5 — Docs/README | 12 | 0 / 12 | ⏸️ |
| 6 — Nettoyage | 5 | 0 / 5 | ⏸️ |
| **TOTAL** | **72** | **0 / 72** | **⏸️** |

---

## 0. Principes directeurs

1. **Modifications chirurgicales** (CLAUDE.md #7) — pas de refactor au-delà du finding.
2. **Zéro régression** — chaque PR doit laisser `make test` / `make lint` verts.
3. **Bench A/B obligatoire** pour tout finding perf (P0-01, P0-09, P1-01 à P1-04, P1-22).
4. **Isolation worktree** pour tout finding à risque (P0, modifications de `bigfft`, `orchestrator`, `go.mod`).
5. **Une PR = un finding** (ou un groupe cohérent de findings triviaux).
6. **Commit convention** : `<type>(<scope>): <finding-id> — <résumé>` (ex : `perf(bigfft): P0-01 — release PolValues buffers`).
7. **Validation humaine** avant merge pour toute PR P0 ou touchant l'API publique.
8. **Audit read-only interdit en Phase 4** : chaque team modifie son périmètre et valide par tests.

---

## 1. Organisation des équipes d'exécution

Reprise stricte des 6 teams d'audit, chacune devient team d'exécution sur son périmètre. Le **Coordinateur** (conversation principale) orchestre, lance les agents en parallèle quand les findings sont indépendants, séquentiel quand ils partagent des fichiers.

| Team | Sous-agent suggéré | Périmètre Phase 4 | Findings assignés |
|------|--------------------|-------------------|-------------------|
| **A — Performance** | `general-purpose` (isolation worktree) | `internal/bigfft/`, `internal/fibonacci/`, `internal/parallel/`, `internal/orchestration/` | P0-01, P0-09, P1-01, P1-02, P1-03, P1-04, P1-22, P2-01, P2-02, P2-03 |
| **B — Architecture/Refactor** | `general-purpose` | `internal/fibonacci/`, `internal/cli/`, `internal/calibration/`, `internal/config/` | P1-05, P1-06, P1-07, P1-08, P1-25, P2-04, P2-05 |
| **C — Tests/Qualité** | `general-purpose` | `*_test.go`, `cmd/generate-golden/`, `internal/testutil/` | P1-10, P1-11, P1-26, P2-06, P2-07, P2-08, P3-01, P3-02, P3-03 |
| **D — Documentation** | `general-purpose` | `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `docs/**`, `doc.go`, `.env.example` | P0-04, P0-05, P0-06, P0-07, P1-12, P1-13, P1-14, P1-15, P1-16, P1-17, P1-18, P2-14 → P2-22 |
| **E — Sécurité** | `general-purpose` | `internal/orchestration/`, `internal/fibonacci/common.go`, `internal/calibration/io.go`, `internal/bigfft/fft_poly.go` | P0-08, P1-19, P1-23, P2-09, P2-10, P2-11, P2-12, P2-13 |
| **F — Tooling/Build** | `general-purpose` | `Makefile`, `go.mod`, `.gitignore`, `.golangci.yml`, `.github/workflows/**`, `cmd/fibcalc/*.pgo` | P0-02, P0-03, P1-09, P1-20, P1-21, P1-24, P2-23, P2-24, P2-25, P2-26 |

**Coordinateur** : consolide, merge PRs, arbitre conflits inter-teams (ex : P1-22 PGO dépend de P0-03 deps upgrade), met à jour `PDRTask.md`.

---

## 2. Dépendances & ordonnancement inter-findings

Graphe des dépendances critiques (à respecter sous peine de rework) :

```
P0-02 (go.mod 1.25 + toolchain)
  └── P0-03 (bubbles v1.0 upgrade) ────┐
        └── P1-24 (deps mineures)      │
                                        ├── P1-22 (PGO rebuild avec nouvelle toolchain)
P1-09 (gofmt -s)                        │
  └── [tout patch touchant fichiers reformattés]
P0-01/P0-09 (pool leak FFT)             │
  ├── P1-01 (BumpAllocator Transform) <─┘
  ├── P2-12 (fourier() err check)  ← mêmes fichiers, à bundler
  └── benchmarks de validation
P1-20 (gitignore + rm *_err/*_out.txt)
  └── [tous les commits ultérieurs ne peuvent plus régénérer ces fichiers]
P1-17 (CHANGELOG Unreleased)
  └── à mettre à jour EN FIN de chaque phase, pas une seule fois
```

**Règle** : les findings bold **doivent** être mergés avant leurs dépendants.

---

## 3. Roadmap en 6 phases

### Phase 0 — Préparation (Coordinateur, ~30 min)

**Objectif** : préparer le terrain, geler la baseline, créer la branche.

1. Créer branche `claude/audit-execution-20260418` depuis `main`.
2. Capturer baseline de validation :
   - `make test -race` (ou sans race si Windows sans gcc) → `Audit/bench/exec-baseline/test.txt`
   - `make coverage` → `.../coverage.txt` + `coverage.out`
   - `make benchmark -count=10 -benchtime=2s` (BenchmarkFastDoubling F(10⁶), F(10⁷)) → `.../benchmark.txt` (référence pour `benchstat`)
   - `make lint` → `.../lint.txt`
3. Installer `benchstat` (`go install golang.org/x/perf/cmd/benchstat@latest`) si absent.
4. Ouvrir `Audit/PDRTask.md` Phase 4 et créer tableau de suivi exécution.
5. Commit initial vide documentant la branche : `chore: open execution branch for AUDIT_REPORT findings`.

**Critères de sortie** : branche créée, baseline figée, `PDRTask.md` mis à jour.

---

### Phase 1 — Quick wins & hygiène (Team F principalement, ~1 jour, faible risque)

**Objectif** : livrer rapidement les findings triviaux qui débloquent les phases suivantes.

| Ordre | Finding | Team | Action | PR |
|-------|---------|------|--------|----|
| 1.1 | **P1-09** | F | `gofmt -s -w ./... && goimports -w ./...` | PR#1 (commit isolé, aucun autre changement) |
| 1.2 | **P1-20** | F | Ajouter `*_err.txt`, `*_out.txt`, `fibonacci.test.exe` au `.gitignore` + `git rm --cached` des 4 fichiers racine | PR#2 |
| 1.3 | **P2-23** | F | Compléter `.PHONY` Makefile (10 cibles manquantes) | PR#3 |
| 1.4 | **P1-21** | F | Ajouter `-trimpath` dans LDFLAGS Makefile | PR#3 (même PR, même périmètre Makefile) |
| 1.5 | **P2-24** | F | `make help` portable : remplacer `column`/`sed` par awk POSIX portable | PR#3 |
| 1.6 | **P2-25** | F | Ajouter `linux/arm64` et `windows/arm64` à `build-all` | PR#3 |
| 1.7 | **P2-26** | F | `go mod tidy` pour x/sync + marquer `bench-versioned`/`pgo-profile` POSIX-only | PR#4 |
| 1.8 | **P0-04/05/06/07** | D | Corriger les 4 groupes de liens Markdown cassés (supprimer les refs fantômes OU créer stubs minimaux si vraiment nécessaires) | PR#5 |
| 1.9 | **P1-16** | D | `BUILD.md` : "24 linters" → "22 linters" | PR#5 |
| 1.10 | **P2-14/15/16** | D | Rafraîchir stats `Claude.md` (108/97, arbre `internal/`) + `docs/ARCH.md` (22 packages) | PR#5 |
| 1.11 | **P2-20** | D | Remplacer badge coverage `80%` README par badge dynamique (codecov.io) ou valeur 87.5% factuelle | PR#5 |

**Gating** :
- `make test` vert après chaque PR.
- `make lint` ne doit pas créer de nouveaux findings.
- Revue humaine rapide, merge direct.

**Livrable** : 5 PRs, aucune régression, repo plus propre.

---

### Phase 2 — Contrat Go, sécurité & dépendances (Teams E + F, 3-5 jours, risque modéré)

**Objectif** : remettre à niveau Go 1.25 + toolchain et appliquer les correctifs sécurité.

#### Phase 2a — Go toolchain (séquentiel, prérequis pour 2b)

| Ordre | Finding | Team | Action | PR |
|-------|---------|------|--------|----|
| 2a.1 | **P0-02** | F | `go.mod` : `go 1.25.0` + `toolchain go1.25.3`. Revalider `make test`, `make lint`, `make benchmark` sur 1.25. | PR#6 (**isolation worktree**) |

**Gate 2a** : tous tests verts sur Go 1.25 avant de merger. Si régression → rollback immédiat, investigation, blocage Phase 2b.

#### Phase 2b — Dépendances (parallèle après 2a mergée)

| Ordre | Finding | Team | Action | PR |
|-------|---------|------|--------|----|
| 2b.1 | **P1-24** | F | Upgrade mineur/patch : `x/sync`, `x/sys`, `x/term`, `x/text`, `zerolog`, `gopsutil` (groupé) | PR#7 |
| 2b.2 | **P0-03** | F | **Upgrade majeur `bubbles v0.21 → v1.0`** — inspecter breaking changes, adapter `internal/tui/`, smoke-test TUI manuel (`./fibcalc --tui 100`) | PR#8 (**isolation worktree**, review humaine obligatoire) |

**Gate 2b** : smoke-test TUI manuel documenté dans PR#8. Snapshot tests `internal/tui/*_test.go` verts.

#### Phase 2c — Sécurité & doc.go (parallèle entre eux)

| Ordre | Finding | Team | Action | PR |
|-------|---------|------|--------|----|
| 2c.1 | **P0-08** | E | `orchestrator.go:67` : refactor pour exploiter cancellation errgroup (préférer vraie propagation plutôt que `_ = g.Wait()`) | PR#9 |
| 2c.2 | **P1-19** | E | Ajouter `ctx.Err()` post-sémaphore dans `executeTasks`/`executeMixedTasks` (`common.go:215,265,276`) | PR#10 |
| 2c.3 | **P1-23** | E+F | `.golangci.yml` : whitelister explicitement les 21 G115 en faux positifs avec commentaire justificatif OU refactor les casts | PR#11 |
| 2c.4 | **P2-11** | E | `calibration/io.go:38` : gérer retour `tw.Flush()` | PR#12 (groupe) |
| 2c.5 | **P2-12** | E | `fft_poly.go:297,315` : gérer retour `fourier()` | PR#12 |
| 2c.6 | **P2-09** | E | Évaluer les 21 G115 un par un, supprimer casts inutiles, documenter les casts sûrs | PR#13 |
| 2c.7 | **P2-10** | E | Documenter les 2 G304 comme faux positifs (mono-user CLI) dans `.golangci.yml` excludes | PR#13 |
| 2c.8 | **P2-13** | E | Aucune action code — documenter convention `Must*` dans `doc.go` du package concerné | groupé avec P1-18 |
| 2c.9 | **P1-18** | D+E | Créer 5 `doc.go` manquants (`format`, `metrics`, `progress`, `fibonacci/memory`, `fibonacci/threshold`) | PR#14 |

**Gate 2c** : `govulncheck` = 0 ; `gosec` count stable ou en baisse ; `make test -race` vert.

**Livrables Phase 2** : 9 PRs, Go 1.25 en vigueur, deps à jour, TUI validée, errgroup propre.

---

### Phase 3 — Performance (Team A, 5-7 jours, risque élevé, ROI très élevé)

**Objectif** : matérialiser les gains identifiés par Team A, chaque PR accompagnée de `benchstat`.

**Protocole de validation perf** (obligatoire pour chaque finding) :
1. Avant le patch : `go test -bench=Benchmark{FastDoubling,Matrix,FFT} -benchmem -count=10 -benchtime=2s ./... > before.txt`
2. Appliquer le patch.
3. Après le patch : même commande → `after.txt`.
4. `benchstat before.txt after.txt > delta.txt` — **doit montrer amélioration significative (p < 0.05)** sur les benchmarks ciblés, pas de régression > 2 % sur les autres.
5. `delta.txt` committé dans `bench/perf-results/<finding-id>/`.

| Ordre | Finding | Action | Bench cible | PR |
|-------|---------|--------|-------------|----|
| 3.1 | **P0-01 + P0-09** (bundled) | `internal/bigfft/fft_poly.go` : ajouter `Release()` sur `PolValues`/`Poly`, defer aux call-sites `Transform`/`InvTransform`/`mul`/`sqr` | `BenchmarkFFT_F1M`, `BenchmarkFFT_F10M` | PR#15 (**worktree**, review) |
| 3.2 | **P1-01** | `fibonacci/fft.go:106-113` : utiliser BumpAllocator pour `executeDoublingStepFFT` | `BenchmarkFastDoubling_F1M` séquentiel | PR#16 |
| 3.3 | **P1-02** | `fibonacci/doubling_framework.go:230-245` : throttler `cache.Stats()` (1 appel tous les N itérations au lieu de 24) | `BenchmarkFastDoubling_F100k` | PR#17 |
| 3.4 | **P1-03** | `generator_iterative.go:113,115` : remplacer `new(big.Int)` par pool ou pré-allocation avec `NextInto(dst *big.Int)` | `BenchmarkFirst1000` | PR#18 |
| 3.5 | **P1-04** | `fastdoubling.go:103` + `memory/arena.go` : pooler l'arena | `BenchmarkFastDoubling_F500k` | PR#19 |
| 3.6 | **P1-22** | Générer `cmd/fibcalc/default.pgo` (workflow `make pgo-profile`), commit, retirer `*.pgo` du `.gitignore`, ajouter cible `make build-pgo-ci` | `BenchmarkFastDoubling_F1M` PGO vs non-PGO | PR#20 |
| 3.7 | **P2-01** | Investiguer pourquoi `WithOptimizedCache` est 20% plus lent — soit le corriger, soit documenter + déprécier | selon résultat | PR#21 |
| 3.8 | **P2-02** | Remplacer les 3 sémaphores indépendants par 1 sémaphore global partagé (NumCPU×1) | `BenchmarkParallel3_F1M` | PR#22 |
| 3.9 | **P2-03** | `executeParallel3` : empêcher `wg`/`ec` d'échapper au heap (passage par pointeur via struct stack-allouée) | escape analysis `-gcflags="-m=2"` | PR#23 |

**Gate Phase 3** : `benchstat` global comparant baseline Phase 0 vs fin Phase 3 — **amélioration cumulée ≥ 15 % allocations FFT, ≥ 5 % CPU** sur F(10⁶). Sinon justifier ou rollback.

**Livrables** : 9 PRs perf, dossier `bench/perf-results/` avec preuves `benchstat`.

---

### Phase 4 — Refactor chirurgical, tests & code mort (Teams B + C, 3-4 jours, risque faible)

**Objectif** : régler la dette de structure et tests identifiée.

#### 4a — Refactor (Team B)

| Ordre | Finding | Action | PR |
|-------|---------|--------|----|
| 4a.1 | **P1-05** | Extraire helper commun de `executeParallel3` (~40 LoC dédup) dans `fibonacci/common.go` | PR#24 |
| 4a.2 | **P1-06** | Extraire helper arena pre-size (`fastdoubling.go` + `fft_based.go`) | PR#24 |
| 4a.3 | **P1-08** | `matrix_types.go:154-170` : réduire cyclo 24 → 4 via extraction de sous-fonctions | PR#25 |
| 4a.4 | **P1-07** | Supprimer/adapter les 5 `unparam` signalés (microbench, presenter, env, fft) | PR#26 |
| 4a.5 | **P1-25** | `internal/cli/presenter.go` : supprimer les 7 wrappers triviaux ou ajouter tests pour couverture | PR#27 |
| 4a.6 | **P2-04** | Documenter dans `cmd/generate-golden/doc.go` la duplication volontaire `fibBig` ↔ `calculateSmall` (oracle indépendant) | PR#28 |
| 4a.7 | **P2-05** | `calibration.go:65` : extraire `configureHardwareDetection` et `runPassSequence` de `RunCalibrationWithOptions` | PR#29 |

#### 4b — Tests (Team C, peut paralléliser avec 4a)

| Ordre | Finding | Action | PR |
|-------|---------|--------|----|
| 4b.1 | **P1-10** | Ajouter `t.Parallel()` aux 122 tests manquants dans `internal/tui/*_test.go` | PR#30 |
| 4b.2 | **P1-11** | Confirmer que `internal/config/thresholds.go` (4 fonctions 0% cov) est bien dupliqué de `calibration/adaptive.go` → supprimer, sinon tester | PR#31 |
| 4b.3 | **P1-26** | Ajouter cas test FFT-bound n>700k à `cmd/generate-golden/main_test.go`, remplacer `targets[]` codés en dur par génération | PR#32 |
| 4b.4 | **P2-06** | `internal/fibonacci/testing.go` : couvrir `TestFactory` exporté ou le rendre privé | PR#33 |
| 4b.5 | **P2-07** | Supprimer cibles Makefile `generate-mocks`/`install-mockgen` inertes | PR#33 (même PR Makefile) |
| 4b.6 | **P2-08** | Créer `testdata/fuzz-seed/` avec corpus persistant pour les 5 fuzz targets | PR#34 |
| 4b.7 | **P3-01** | `logs_test.go:283` : enquêter sur le `t.Skip("viewport...")` — réparer ou supprimer | PR#35 |
| 4b.8 | **P3-02** | `cli/ui_advanced_test.go:43` : remplacer `time.Sleep(50ms)` par synchro déterministe | PR#35 |
| 4b.9 | **P3-03** | `bench-versioned` : étendre au-delà de `BenchmarkFastDoubling` | PR#35 |

**Gate Phase 4** : couverture globale `≥ 87.5 %`, `make lint` findings en baisse nette, temps suite test `internal/tui` en baisse grâce à parallélisation.

---

### Phase 5 — Documentation & README (Team D, 2 jours, risque nul)

**Objectif** : rendre la doc cohérente, à jour, et rafraîchir le README comme vitrine du projet post-audit.

#### 5a — Findings doc restants

| Ordre | Finding | Action | PR |
|-------|---------|--------|----|
| 5a.1 | **P1-12** | Choix : soit implémenter `FIBCALC_GC_CONTROL` (effort M), soit retirer de `docs/PERFORMANCE.md` — **décision requise coordinateur** | PR#36 |
| 5a.2 | **P1-13** | Documenter `FIBCALC_TUI_THEME` dans README + BUILD + .env.example | PR#37 (groupe env) |
| 5a.3 | **P1-14** | Ajouter `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT`, `FIBCALC_TUI_THEME` à `.env.example` | PR#37 |
| 5a.4 | **P1-15** | `docs/TESTING.md:281,283` : mettre à jour chemins post-extraction sous-packages | PR#38 |
| 5a.5 | **P2-17** | Aligner `CONTRIBUTING.md` ↔ `TESTING.md` sur mockgen (référence unique) | PR#38 |
| 5a.6 | **P2-18** | Enrichir les 5 `doc.go` triviaux (`bigfft`, `calibration`, `app`, `parallel`, `testutil`) — exemples d'usage, invariants | PR#39 |
| 5a.7 | **P2-19** | Harmoniser benchmarks README (Intel) vs PERFORMANCE.md (Ryzen) — choisir une plateforme de référence + annexe comparative | PR#40 |
| 5a.8 | **P2-21** | Créer `docs/architecture/patterns/` réel (au moins ADR-010, design-patterns.md) OU retirer mention dans `docs/architecture/README.md` | PR#41 |
| 5a.9 | **P2-22** | `dependency-graph.mermaid` : ajouter `progress`, `memory`, `threshold` | PR#42 |

#### 5b — Mise à jour README (livrable final de documentation)

**Action** : réécriture condensée du `README.md` pour refléter l'état post-audit. Team D produit un nouveau README qui :

1. **Résumé ≤ 1 écran** : qu'est-ce que FibGo, pour qui, algorithmes, perf.
2. **Section "Quick start"** : installation (`go install`, build local), première commande (`fibcalc 1000`).
3. **Section "Architecture"** : lien vers `docs/architecture/` + dépendance-graph Mermaid embarqué (ou lien).
4. **Section "Performance"** : benchmarks actualisés **post-Phase 3** (avec gains mesurés — ex : "FFT pool leak corrigé : -60 % allocs F(10⁶) vs v2.x").
5. **Section "Configuration"** : table des 10+ variables `FIBCALC_*` avec valeurs par défaut, y compris `FIBCALC_TUI_THEME`.
6. **Section "Développement"** : `make all`, `make lint`, `make benchmark`, `make build-pgo`, liens CLAUDE.md et CONTRIBUTING.md.
7. **Section "Tests & couverture"** : badge dynamique, pointeur vers rapport de couverture, fuzz targets.
8. **Section "Changelog"** : lien vers `CHANGELOG.md` mis à jour (P1-17).
9. **Retirer** : sections dupliquant `docs/PERFORMANCE.md`, `docs/BUILD.md`, `docs/TUI_GUIDE.md` — remplacer par liens.
10. **Target taille** : 18-22 Ko (vs 38 Ko actuel) — compression ~45 %.

| Ordre | Finding | Action | PR |
|-------|---------|--------|----|
| 5b.1 | **P1-17 final** | `CHANGELOG.md [Unreleased]` : récap toutes les PRs Phase 1-5 regroupées par type (Performance, Security, Docs, Build) | PR#43 |
| 5b.2 | **README** (global) | Réécriture selon checklist 1-10 ci-dessus | PR#43 |
| 5b.3 | **P2-19 / P2-20** | Benchmarks + badge coverage cohérents dans README refactoré | PR#43 |

**Gate Phase 5** : README relu par humain, tous liens markdown valides (`markdown-link-check`), `CHANGELOG.md` passe validateur Keep-a-Changelog.

---

### Phase 6 — Nettoyage final du dépôt (Coordinateur, 0.5 jour, risque nul)

**Objectif** : éliminer tous les fichiers inutiles identifiés durant l'audit + détection complémentaire.

#### 6a — Fichiers à supprimer (identifiés par l'audit)

| Fichier | Raison | Action |
|---------|--------|--------|
| `build_err.txt` | Log de build dev, trackerd par erreur (F-8 / E-8) | `git rm` + gitignore (déjà fait Phase 1.2) |
| `test_err.txt` | idem | `git rm` |
| `test_out.txt` | idem | `git rm` |
| `e2e_rich_out.txt` | idem | `git rm` |
| `fibonacci.test.exe` | Binaire test Windows, ne doit jamais être commit | `git rm` + gitignore `*.test.exe`, `*.test` |
| `Audit/PDRTask.md` | Document d'exécution transitoire (doublon de `PRD.md` + ce plan) | À déplacer dans `Audit/archive/` ou supprimer après merge final |
| Documents fantômes référencés | `docs/INNOVATION.md`, `docs/INNOVEPLAN.md` — déjà traités P0-04 | — |

#### 6b — Détection complémentaire (avant merge final)

Commandes d'audit à passer pour détecter les fichiers oubliés :

```bash
# 1. Fichiers binaires trackés
git ls-files | xargs -I{} file {} | grep -E "executable|binary" | grep -v -E "\\.(png|jpg|ico|pdf)$"

# 2. Fichiers > 1 Mo (hors LICENSE, testdata justifiés)
git ls-files | xargs -I{} du -h {} 2>/dev/null | sort -rh | head -20

# 3. Fichiers jamais référencés dans le code ou la doc
git ls-files | grep -v -E "(\\.git|testdata|\\.golden|LICENSE|go\\.(mod|sum))" | while read f; do
  name=$(basename "$f")
  refs=$(grep -r "$name" --exclude-dir=.git | wc -l)
  [ "$refs" -le "1" ] && echo "ORPHAN: $f"
done

# 4. Artifacts CI résiduels
find . -type f \( -name "*.pgo.old" -o -name "*.tmp" -o -name "*.bak" -o -name "*~" \) -not -path "./.git/*"

# 5. Coverage locaux oubliés
find . -type f \( -name "coverage.out" -o -name "coverage.html" -o -name "cover.out" \) -not -path "./.git/*"
```

#### 6c — Dossier `Audit/`

Après merge complet des Phases 1-5, le dossier `Audit/` devient historique. Deux options :

**Option A (recommandée)** : Déplacer `Audit/` → `docs/audits/2026-04/` pour préserver le rapport dans l'arborescence documentaire (utile pour audit futur, traçabilité).

**Option B** : Taguer `audit-2026-04-18` sur le commit contenant `Audit/` puis supprimer le dossier (plus propre, mais perte de la visibilité immédiate).

**Décision coordinateur requise** avant Phase 6c.

#### 6d — `.gitignore` final

Ajouter si manquant :
```
# Logs dev
build_err.txt
test_err.txt
test_out.txt
e2e_rich_out.txt
*_err.txt
*_out.txt

# Binaires de test
*.test
*.test.exe
fibonacci.test.exe

# Coverage locaux
coverage.out
coverage.html
cover.out

# Profiling
*.prof
!cmd/fibcalc/default.pgo  # exception PGO committé

# Editeurs
*~
*.bak
*.swp
.vscode/
.idea/
```

| Ordre | Action | PR |
|-------|--------|----|
| 6.1 | Exécution des 5 commandes 6b, reporter findings au coordinateur | — (investigation) |
| 6.2 | `git rm` des fichiers orphelins confirmés | PR#44 |
| 6.3 | Mise à jour `.gitignore` selon 6d | PR#44 |
| 6.4 | Décision + exécution 6c (dossier Audit/) | PR#45 |

**Gate Phase 6** : `git status` propre, `git ls-files | wc -l` en baisse, `make all` vert.

---

## 4. Matrice de traçabilité complète (60 findings → PRs)

| Finding | Phase | PR | Team | Statut |
|---------|-------|----|----|--------|
| P0-01 | 3.1 | #15 | A | ⏸️ |
| P0-02 | 2a.1 | #6 | F | ⏸️ |
| P0-03 | 2b.2 | #8 | F | ⏸️ |
| P0-04 | 1.8 | #5 | D | ⏸️ |
| P0-05 | 1.8 | #5 | D | ⏸️ |
| P0-06 | 1.8 | #5 | D | ⏸️ |
| P0-07 | 1.8 | #5 | D | ⏸️ |
| P0-08 | 2c.1 | #9 | E | ⏸️ |
| P0-09 | 3.1 | #15 | A | ⏸️ |
| P1-01 | 3.2 | #16 | A | ⏸️ |
| P1-02 | 3.3 | #17 | A | ⏸️ |
| P1-03 | 3.4 | #18 | A | ⏸️ |
| P1-04 | 3.5 | #19 | A | ⏸️ |
| P1-05 | 4a.1 | #24 | B | ⏸️ |
| P1-06 | 4a.2 | #24 | B | ⏸️ |
| P1-07 | 4a.4 | #26 | B | ⏸️ |
| P1-08 | 4a.3 | #25 | B | ⏸️ |
| P1-09 | 1.1 | #1 | F | ⏸️ |
| P1-10 | 4b.1 | #30 | C | ⏸️ |
| P1-11 | 4b.2 | #31 | C | ⏸️ |
| P1-12 | 5a.1 | #36 | D | ⏸️ |
| P1-13 | 5a.2 | #37 | D | ⏸️ |
| P1-14 | 5a.3 | #37 | D | ⏸️ |
| P1-15 | 5a.4 | #38 | D | ⏸️ |
| P1-16 | 1.9 | #5 | D | ⏸️ |
| P1-17 | 5b.1 | #43 | D | ⏸️ |
| P1-18 | 2c.9 | #14 | D+E | ⏸️ |
| P1-19 | 2c.2 | #10 | E | ⏸️ |
| P1-20 | 1.2 | #2 | F | ⏸️ |
| P1-21 | 1.4 | #3 | F | ⏸️ |
| P1-22 | 3.6 | #20 | A | ⏸️ |
| P1-23 | 2c.3 | #11 | E+F | ⏸️ |
| P1-24 | 2b.1 | #7 | F | ⏸️ |
| P1-25 | 4a.5 | #27 | B | ⏸️ |
| P1-26 | 4b.3 | #32 | C | ⏸️ |
| P2-01 | 3.7 | #21 | A | ⏸️ |
| P2-02 | 3.8 | #22 | A | ⏸️ |
| P2-03 | 3.9 | #23 | A | ⏸️ |
| P2-04 | 4a.6 | #28 | B | ⏸️ |
| P2-05 | 4a.7 | #29 | B | ⏸️ |
| P2-06 | 4b.4 | #33 | C | ⏸️ |
| P2-07 | 4b.5 | #33 | C | ⏸️ |
| P2-08 | 4b.6 | #34 | C | ⏸️ |
| P2-09 | 2c.6 | #13 | E | ⏸️ |
| P2-10 | 2c.7 | #13 | E | ⏸️ |
| P2-11 | 2c.4 | #12 | E | ⏸️ |
| P2-12 | 2c.5 | #12 | E | ⏸️ |
| P2-13 | 2c.8 | #14 | E | ⏸️ |
| P2-14 | 1.10 | #5 | D | ⏸️ |
| P2-15 | 1.10 | #5 | D | ⏸️ |
| P2-16 | 1.10 | #5 | D | ⏸️ |
| P2-17 | 5a.5 | #38 | D | ⏸️ |
| P2-18 | 5a.6 | #39 | D | ⏸️ |
| P2-19 | 5a.7 | #40 | D | ⏸️ |
| P2-20 | 1.11 | #5 | D | ⏸️ |
| P2-21 | 5a.8 | #41 | D | ⏸️ |
| P2-22 | 5a.9 | #42 | D | ⏸️ |
| P2-23 | 1.3 | #3 | F | ⏸️ |
| P2-24 | 1.5 | #3 | F | ⏸️ |
| P2-25 | 1.6 | #3 | F | ⏸️ |
| P2-26 | 1.7 | #4 | F | ⏸️ |
| P3-01 | 4b.7 | #35 | C | ⏸️ |
| P3-02 | 4b.8 | #35 | C | ⏸️ |
| P3-03 | 4b.9 | #35 | C | ⏸️ |
| README | 5b.2 | #43 | D | ⏸️ |
| Cleanup | 6.2-6.4 | #44, #45 | Coord. | ⏸️ |

**Total** : 63 findings + README + Cleanup → **45 PRs** réparties sur 6 phases.

---

## 5. Stratégie d'exécution par le Coordinateur

### 5.1 Lancement des agents en parallèle

Le Coordinateur délègue à des sous-agents dans un seul message multi-Agent quand :
- Les findings appartiennent à des teams différentes **et** touchent des fichiers disjoints.
- Aucune dépendance explicite (cf. §2).

**Exemple Phase 1** : `gofmt` (F), `gitignore` (F), liens doc (D) → 3 agents en parallèle.

**Exemple Phase 3** : toutes les PRs perf sont séquentielles (même fichiers, besoin de bench propre à chaque fois).

### 5.2 Prompt type pour un agent d'exécution

```
Tu es Team <X>, agent d'exécution du plan AUDIT_EXECUTION (voir Audit/EXECUTION_PLAN.md §3.<phase>).

Ton périmètre : <liste fichiers>
Findings assignés : <P0-01, ...>
Branche de travail : claude/audit-execution-20260418-<finding-id> (worktree isolée)
Contraintes :
  1. Modifications chirurgicales strictes (CLAUDE.md #7)
  2. `make test -race` et `make lint` doivent rester verts
  3. Pour findings perf : benchstat before/after obligatoire dans bench/perf-results/<id>/
  4. Commit : "<type>(<scope>): <finding-id> — <résumé>"
  5. Produire un résumé ≤ 20 lignes à la fin (fichiers modifiés, tests ajoutés, preuve de non-régression)

Lis AUDIT_REPORT.md pour le détail du finding, puis exécute.
```

### 5.3 Rollback

Règle : tout merge qui casse `make test` ou introduit une régression bench > 2 % est **rollback immédiat** (`git revert <sha>`), investigation dans une issue, PR corrigée.

---

## 6. Critères d'acceptation globaux (PRD §5 étendus)

- [ ] 60 findings + 3 P3 traités et documentés (ou justifiés s'ils ne sont pas applicables).
- [ ] README condensé (≤ 22 Ko) et à jour post-Phase 3.
- [ ] CHANGELOG.md `[Unreleased]` complet, Keep-a-Changelog.
- [ ] `make test -race` vert (ou test sans race documenté pour Windows sans gcc).
- [ ] `make coverage` ≥ 87.5 % pondéré.
- [ ] `make lint` : findings en baisse ≥ 50 % vs baseline.
- [ ] `make benchmark` : `benchstat` montre amélioration sur F(10⁶) allocations > 15 %.
- [ ] `govulncheck ./...` exit 0.
- [ ] `git status` propre, 4 fichiers `*_err/*_out` supprimés, `.gitignore` complet.
- [ ] Tous liens Markdown valides (pas de 404 interne).
- [ ] 5 `doc.go` manquants créés.
- [ ] `go.mod` déclare `go 1.25.0` + `toolchain`.
- [ ] `cmd/fibcalc/default.pgo` committé, build-pgo fonctionne.
- [ ] 45 PRs mergées sur `main` via `claude/audit-execution-20260418`.
- [ ] Tag release `v3.0.0` (ou version convenue) sur merge final.

---

## 7. Risques & mitigations

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| Upgrade `bubbles v1.0` casse TUI | Élevée | Moyen | Isolation worktree, smoke-test manuel obligatoire, possibilité rollback |
| Pool leak FFT patch introduit double-free | Moyenne | Élevé | Tests `-race` + golden tests FFT (F(10⁶)) + revue humaine |
| Regression perf cumulée > 2 % | Faible | Élevé | `benchstat` à chaque PR perf, gate bloquant |
| Upgrade Go 1.25 exposé bug toolchain | Faible | Moyen | Utiliser `toolchain go1.25.3` stable, tester sur Linux + Windows |
| Fichiers nettoyés contenaient données utiles | Faible | Faible | Phase 6b investigation avant suppression, commit isolé réversible |
| Agents produisent PRs non-conformes CLAUDE.md | Moyenne | Moyen | Prompt template §5.2 strict + revue coordinateur avant merge |
| Contexte agent dépassé sur grosse PR | Moyenne | Moyen | 1 PR = 1 finding (ou findings triviaux groupés), limite ~400 LoC diff |

---

## 8. Journal d'exécution (à mettre à jour au fur et à mesure)

| Horodatage | Phase | PR | Finding | Team | Événement |
|------------|-------|----|--------| -----|-----------|
| — | — | — | — | — | À démarrer après validation utilisateur |

---

## 9. Prochaines actions

1. **Valider ce plan** avec l'utilisateur (scope, priorités, décisions 5a.1 GC_CONTROL + 6c dossier Audit).
2. **Phase 0 immédiate** : créer branche + baseline exec.
3. **Lancer Phase 1** via un message multi-Agent (Teams D + F en parallèle).
4. Suivre progression dans §4 (matrice traçabilité) et §8 (journal).

---

## 10. Annexes

- [AUDIT_REPORT.md](AUDIT_REPORT.md) — rapport source, détails des 60 findings
- [PRD.md](PRD.md) — cadre méthodologique Agent Teams
- [PDRTask.md](PDRTask.md) — historique audit read-only (Phases 1-3 closes)
- [CLAUDE.md](../CLAUDE.md) — directives projet impératives
