# 03 — Snapshot Git

_Date du snapshot : 2026-04-28_

## État working tree

- Branche courante : `main`
- Synchronisation : à jour avec `origin/main`
- Working tree : **propre** (aucun fichier modifié, ajouté ou non suivi)

```
On branch main
Your branch is up to date with 'origin/main'.
nothing to commit, working tree clean
```

## Branches & tags

### Branches locales

| Branche | Rôle |
|---|---|
| `main` | Branche par défaut (HEAD) |
| `claude/audit-execution-20260418` | Branche d'audit antérieure (non fusionnée localement, déjà mergée via PR #17) |

### Branches distantes (`origin/*`)

- `origin/HEAD -> origin/main`
- `origin/main`
- `origin/claude/audit-execution-20260418`
- `origin/add-academic-evaluation-7670166451931156632`
- `origin/bonification-agent-mesh-kafka-10457233271422912541`
- `origin/documentation-improvement-3215497703197735245`
- `origin/perf/zero-copy-result-return`

### Tags

| Tag | Date | Commit |
|---|---|---|
| `v3.0.0` | 2026-04-18 | `6ad0c1f` |

Un seul tag de release. Pas de schéma SemVer intermédiaire (pas de `v3.0.0-rc.x`, pas de tags `v2.x`).

## Activité (6 derniers mois)

- **Commits totaux** : 318
- **Fichiers changés (cumul)** : 3 805
- **Lignes** : +1 378 962 / −260 204 (note : gonflé par historique d'archives héritées — voir Observations)

### Auteurs principaux

| Auteur | Commits |
|---|---|
| André-Guy Bruneau | 295 |
| google-labs-jules[bot] | 12 |
| agbruneau | 11 |
| Andre-Guy Bruneau | 1 |

> _Note_ : `André-Guy Bruneau`, `agbruneau` et `Andre-Guy Bruneau` désignent le même contributeur humain — incohérence d'identité Git (`user.name`/`user.email` non normalisé entre machines/sessions).

### Top 15 fichiers les plus modifiés

| # commits | Fichier |
|---|---|
| 55 | `README.md` |
| 48 | `CLAUDE.md` |
| 23 | `.gitignore` |
| 20 | `Makefile` |
| 19 | `internal/app/app.go` |
| 18 | `internal/fibonacci/common.go` |
| 17 | `internal/tui/model.go` |
| 16 | `Docs/ARCHITECTURE.md` |
| 15 | `internal/config/config.go` |
| 14 | `internal/fibonacci/doubling_framework.go` |
| 14 | `go.mod` |
| 13 | `internal/bigfft/fft_cache.go` |
| 12 | `internal/tui/model_test.go` `internal/orchestration/orchestrator.go` `internal/fibonacci/fft.go` `internal/fibonacci/calculator.go` `internal/config/env.go` `internal/cli/output.go` |
| 11 | `internal/fibonacci/fastdoubling.go` `go.sum` |

Les fichiers cœur algorithmique (`fibonacci/*`, `bigfft/*`) et orchestration (`app.go`, `orchestrator.go`) concentrent l'activité technique récente, conforme à un projet en optimisation perf active.

## 30 derniers commits

| SHA | Date | Sujet |
|---|---|---|
| bbaf7cb | 2026-04-28 | Create PLAN.md |
| 4c8f0c1 | 2026-04-28 | refactoring complet |
| c4de75f | 2026-04-28 | Create REFACTOR_PROMPT.md |
| b8af06b | 2026-04-28 | Update Claude.md |
| 64f6038 | 2026-04-27 | Update Claude.md |
| 6ad0c1f | 2026-04-18 | Merge pull request #17 from agbruneau/claude/audit-execution-20260418 |
| 370a1a8 | 2026-04-18 | chore: Phase 6 — archive audit folder to docs/audits/2026-04/ |
| 4d203b5 | 2026-04-18 | test+build: P3-01/P3-02/P3-03 — fix viewport skip, deterministic sync, broader bench-versioned |
| 8810795 | 2026-04-18 | test(fuzz): P2-08 — add persistent fuzz corpus for 5 targets |
| abbf286 | 2026-04-18 | refactor(calibration): P2-05 — split RunCalibrationWithOptions into configure/runPass helpers |
| 4fb071f | 2026-04-18 | chore(tests/makefile): P2-06/P2-07 — cover TestFactory and remove dead mock targets |
| c6fc9c9 | 2026-04-18 | docs(generate-golden): P2-04 — document intentional fibBig/calculateSmall duplication as oracle |
| 07e40e0 | 2026-04-18 | refactor(cli): P1-25 — remove unused presenter wrappers or add tests |
| 7a34140 | 2026-04-18 | test(generate-golden): P1-26 — add FFT-bound n>700k and dynamic targets |
| c40192b | 2026-04-18 | refactor: P1-07 — clean up 5 unparam findings |
| 92f0163 | 2026-04-18 | refactor(config): P1-11 — add tests for thresholds.go (NOT duplicate) |
| e360ed0 | 2026-04-18 | test(tui): P1-10 — add t.Parallel() to 122 TUI tests for faster suite |
| ab5499d | 2026-04-18 | refactor(fibonacci): P1-08 — reduce releaseMatrixState cyclomatic complexity (24 → ≤4) |
| 26081a0 | 2026-04-18 | refactor(fibonacci): P1-05/P1-06 — extract common executeParallel3 and arena pre-size helpers |
| b30d5de | 2026-04-18 | perf(fibonacci): P2-03 — keep executeParallel3 wg/ec on stack (closure path) |
| ec90227 | 2026-04-18 | perf(fibonacci): P2-02 — unify parallel semaphores into single NumCPU-sized pool |
| 82c24a5 | 2026-04-18 | perf(cache): P2-01 — deprecate WithOptimizedCache (slower than default) |
| 4ff7f83 | 2026-04-18 | perf(pgo): P1-22 — commit default.pgo profile for PGO-enabled builds |
| 2437c2b | 2026-04-18 | docs(audit): P1-04 — document why arena pooling was skipped |
| b67af0a | 2026-04-18 | docs: P1-17 + README condensed — update CHANGELOG [Unreleased] and compress README (38->13 KB) |
| 137c684 | 2026-04-18 | perf(fibonacci): P1-03 — add NextInto(dst) to avoid new big.Int in generator |
| 3645963 | 2026-04-18 | perf(fibonacci): P1-02 — throttle cache.Stats() calls (24 -> 3) |
| 6968bd3 | 2026-04-18 | perf(fibonacci): P1-01 — use BumpAllocator in executeDoublingStepFFT |
| 1a4fd94 | 2026-04-18 | perf(bigfft): P2-22 — add progress/memory/threshold nodes to dependency graph |
| 307873f | 2026-04-18 | perf(bigfft): P0-01/P0-09 — release PolValues/Poly buffers to fix pool leak |

## Fichiers volumineux / binaires dans l'historique

### État courant (HEAD)

Aucun PDF, image >1 Mo, archive ou binaire dans `git ls-tree -r HEAD`. Le tree pointé contient exclusivement du code Go, des Markdown, des Mermaid, des JSON de tests (golden, fuzz seeds) et `cmd/fibcalc/default.pgo` (profil PGO, taille raisonnable).

### Blobs > 500 Ko encore présents dans `git rev-list --all`

| Taille | Blob historique |
|---|---|
| 25.9 Mo | `bench/baseline/benchmark.txt` |
| 19.0 Mo | `Presentation/Presentation.pdf` |
| 15.2 Mo | `docs/08` |
| 12.5 Mo | `pdf-generator/output/consolidated/Entreprise_Agentique_Monographie_Complete.pdf` |
| 10.1 Mo | `Docs/PRESENTATION.pdf` |
| 7.0 Mo | `Presentation/Poster.png` (×2 versions) |
| 1.0–3.7 Mo | Volumes I–V `pdf-generator/output/volumes/Volume_*.pdf` |
| 0.6–1.4 Mo | `Volume_*_Consolide.md` |

Tracées via `git log --all` : ces fichiers proviennent de commits hérités (`5bfc50e added FibGo`, `38213ff TuneUp Landing page`, `c127921 Volume 1 à 5 consolidée`, etc.) — vraisemblablement le dépôt a été initialisé en réutilisant un repo Git existant qui contenait une monographie « Entreprise Agentique » et des supports de présentation. Un `45fc4bf Purge` a retiré ces fichiers du HEAD mais ils restent atteignables dans l'historique, gonflant la taille du `.git/`.

## Observations

1. **Working tree propre, branche à jour** — aucune dette de commit non publiée.
2. **Cadence soutenue** : 318 commits en 6 mois, dont une rafale de ~28 commits le 2026-04-18 correspondant à l'exécution de l'audit précédent (préfixes `P0`/`P1`/`P2`/`P3`). Le projet a une discipline de _feature flags_ d'audit (IDs traçables).
3. **Identité Git non normalisée** : 4 alias pour le même auteur. Recommander d'aligner `user.name`/`user.email` (`.mailmap` possible).
4. **Tag unique `v3.0.0`** : pas de tags pré-release ni d'historique de versions précédentes — pas de release antérieure visible. Politique de versionning à clarifier (CHANGELOG mentionne `[Unreleased]`).
5. **Branches distantes orphelines** : 3 branches `origin/*` sans équivalent local (`add-academic-evaluation-…`, `bonification-agent-mesh-kafka-…`, `documentation-improvement-…`, `perf/zero-copy-result-return`) suggèrent des branches d'agents IA ou expérimentations non nettoyées. Candidates à `git push origin --delete`.
6. **Bloat historique** : ~100+ Mo de PDFs/PNGs/Markdowns volumineux issus d'un projet antérieur subsistent dans `.git/`. Si le repo doit rester public/léger, envisager `git filter-repo` (action destructive — à valider).
7. **Hygiène commit** : la majorité des messages suivent une convention `type(scope): IDscope — description` (Conventional Commits étendu). Quelques exceptions récentes (`refactoring complet`, `Update Claude.md`, `Create PLAN.md`) sont moins informatives.
8. **Pas de signatures GPG visibles** dans l'historique consulté (à confirmer si exigé par la politique).
