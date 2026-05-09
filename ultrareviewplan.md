# UltraReviewPlan — FibGo (FibCalc)

**Plan d'exécution parallèle du refactoring identifié dans [`ultrareview.md`](ultrareview.md)**

Document opérationnel : décompose les 37 recommandations (R1.1 → R4.12) en **lots exécutables en parallèle par des équipes d'agents**, avec tableau de suivi mis à jour au fur et à mesure de l'avancement.

| Champ | Valeur |
|---|---|
| Source | [`ultrareview.md`](ultrareview.md) |
| Auteur du plan | Claude Opus 4.7 |
| Date | 2026-05-09 |
| Workflow recommandé | 4 vagues, 11 lots, ~37 PRs distinctes |
| Effort total | ~5,5 semaines (mais ~2,5 semaines en exécution parallèle agressive) |

---

## ⚠️ AVANT DE COMMENCER

**Toute exécution doit être précédée par :**
1. ✅ Bench baseline (`make benchmark > docs/audits/baseline-pre-refactor.txt`).
2. ✅ Coverage baseline (`make coverage` → snapshot).
3. ✅ Branche de travail dédiée (`git checkout -b refactor/ultrareview`).
4. ✅ Workflow CI (R4.1) installé en PRIORITÉ ABSOLUE pour gater chaque PR.

**À chaque PR mergée :** `make test -race && make lint && make benchmark` doivent rester verts. Régression > 5 % en latency = blocage.

---

## 1. Tableau de suivi

**Légende des statuts :**
- ⬜ `Pending` — non démarré
- 🟡 `InProgress` — agent dispatché
- 🔵 `InReview` — code écrit, en attente de validation
- ✅ `Done` — mergé sur `main`
- ⚠ `Blocked` — bloqué (raison à indiquer)
- ⛔ `Skipped` — décidé de ne pas faire (raison à indiquer)

**Format de mise à jour :** éditer la cellule `Statut` directement. Ajouter en commentaire `Owner` (agent ou humain) et `Commit` (SHA court) à la fin de la ligne du tableau.

### Vague 1 — Fixes critiques (½ jour, 1 lot parallèle)

| ID | Tâche | Sévérité | Files | Effort | Statut | Owner | Commit |
|----|-------|----------|-------|--------|--------|-------|--------|
| R1.1 | Fix `clearStateAliases` toujours appelé | Critique | `internal/fibonacci/fastdoubling.go` | 30 min | ✅ | agent (parallel) | (Vague1) |
| R1.2 | `defer gc.End()` panic-safe | Haute | `internal/fibonacci/memory/gc_control.go` | 1 h | ✅ | agent (parallel) | (Vague1) |
| R1.3 | Invalidation profil via `IsStale` | Haute | `internal/calibration/calibration.go` | 30 min | ✅ | agent (parallel) | (Vague1) |
| R1.4 | Fix `releaseWordSlice` | Critique | `internal/bigfft/pool.go` | 30 min | ✅ | agent (parallel) | (Vague1) |
| R1.5 | Fix `putByKey` eager alloc | Critique | `internal/bigfft/fft_cache.go` | 2 h | ✅ | agent (parallel) | (Vague1) |

### Vague 2 — Suppressions et fusions à fort levier (1 semaine, 2 lots parallèles)

| ID | Tâche | Lot | Files | Effort | Statut | Owner | Commit |
|----|-------|-----|-------|--------|--------|-------|--------|
| R2.1 | Supprimer `progress_aliases.go` + migrer imports | 2A | `internal/fibonacci/progress_aliases.go` (delete) + dépendants | 30 min | ✅ | agent | (Vague2A) |
| R2.2 | Supprimer/consolider `internal/parallel/` | 2A | `internal/parallel/` (delete) | 1 h | ⚠ | agent | BLOCKED: ErrorCollector utilisé en prod par fibonacci/common.go (P2-03 escape opt). Re-trier en refactor errgroup avec validation perf. |
| R2.4 | Supprimer `BumpAllocatorAdapter` | 2A | `internal/bigfft/allocator.go` | 30 min | ✅ | agent | (Vague2A) |
| R2.6 | Fusionner `cli/{presenter,ui_display,output}` | 2A | `internal/cli/{presenter,ui_display,output}.go` | 4 h | ✅ | agent | (Vague2A) |
| R2.7 | Fusionner `ui/{colors,themes}` + `tui/styles` | 2A | `internal/ui/`, `internal/tui/styles.go` | 6 h | ✅ | agent | (Vague2A) |
| R2.3 | Supprimer code mort `bigfft` (scan, CPU, …) | 2B | `internal/bigfft/{scan.go,cpu_amd64.go,arith_amd64.go}` + `pool.go` | 2 h | ⚠ | agent | BLOCKED: tests existants (scan_test, pool_test, cpu_amd64_extended_test, arith_amd64_test) référencent ces symboles. Suppression nécessite autorisation conjointe sur les tests. |
| R2.5 | Fusionner triple couche progression | 2B | `internal/orchestration/progress.go`, `internal/format/progress_eta.go` | 1 j | ✅ | agent | (Vague2B) split format/eta.go + progress.go ; orchestration/progress.go intégré |
| R2.8 | Fusionner `bigfft fft.go ↔ fft_poly.go` | 2B | `internal/bigfft/fft.go`, `internal/bigfft/fft_poly.go` | 1 j | ⚠ | agent | BLOCKED: fusion déjà appliquée historiquement (fftmulTo délègue à Poly.MulCachedWithBump). Reste duplication dans fft_cache.go (Mul/MulCached/MulCachedWithBump) — interdite par R2.10 en parallèle. |
| R2.9 | Fusionner `executeFFTTransforms{P,S}` | 2B | `internal/fibonacci/fft.go` | 1 h | ✅ | agent | (Vague2B) -49 LOC, helper fftInvTransformInto extrait |
| R2.10 | Consolider hash FNV | 2B | `internal/bigfft/fft_cache.go` | 1 h | ✅ | agent | (Vague2B) cacheKeyBuilder consolide les 3 versions |

### Vague 3 — Décompositions structurelles (3 semaines, 4 lots)

| ID | Tâche | Lot | Files | Effort | Statut | Owner | Commit |
|----|-------|-----|-------|--------|--------|-------|--------|
| R3.1 | Décomposer `threshold/manager.go` (3 types) | 3A | `internal/fibonacci/threshold/` | 4 h | ✅ | agent | (Vague3A) MetricsBuffer + ThresholdAnalyzer + Manager ; 97.4% coverage |
| R3.3 | Stratégie `Calibrate` adaptative | 3A | `internal/calibration/calibration.go` | 3 h | ✅ | agent | (Vague3A) FastStrategy + CompleteStrategy + escalation orchestrator |
| R3.4 | Décomposer `tui/model.go` en composants | 3A | `internal/tui/model.go` + nouveau `internal/tui/component/` | 1,5 j | ⚠ | agent | PARTIAL: model.go 425→177 LOC via routing/handlers/commands/layout ; physical relocation des 5 composants vers component/ deferred (white-box tests bloqueraient) |
| R3.5 | Éclater `cli/completion.go` par shell | 3A | `internal/cli/completion.go` → `internal/cli/completion/` | 4 h | ✅ | agent | (Vague3A) sous-package créé, parité bit-pour-bit confirmée |
| R3.10 | Standardiser noms `*Calculator` | 3A | `internal/fibonacci/{fastdoubling,matrix,fft_based}.go` | 1 h | ✅ | agent | (Vague3A) OptimizedFastDoubling→FastDoublingCalculator ; MatrixExponentiation→MatrixExponentiationCalculator |
| R3.6 | Scinder `runCalculate` + extraire `lastdigits` | 3B | `internal/app/calculate.go`, `internal/orchestration/lastdigits.go` (new) | 4 h | ✅ | agent | (Vague3B) runCalculate 215→29 LOC ; ComputeLastDigits + ValidateMemoryBudget extraits |
| R3.8 | Errors au lieu de panics dans Fermat | 3B | `internal/bigfft/fermat.go` + propagation | 4 h | ✅ | agent | (Vague3B) Stratégie hybride : panics internes préservés + wrappers Safe (MulSafe/SqrSafe/...) |
| R3.9 | Cache fast-path lock-free | 3B | `internal/bigfft/fft_cache.go` | 4 h | ✅ | agent | (Vague3B) -70% latency parallèle (3.3x speedup) en cache hit chaude |
| R3.7 | `bigfft.FFTContext` injectable | 3C | `internal/bigfft/` (refactor large) | 1 j | ⚠ | agent | PARTIAL: FFTContext + MulWithContext/SqrWithContext exposés, isolation 2 contextes vérifiée. Pools globaux conservés (scope conservateur, bench dans gate -5%). |
| R3.2 | Découpler cache FFT de `DoublingFramework` | 3D | `internal/fibonacci/doubling_framework.go` + `internal/bigfft/` | 1 j | ✅ | agent | (Vague3D) ExecuteDoublingLoop 137→91 LOC ; CacheStrategy injectable ; import bigfft retiré |

### Vague 4 — Polissage & outillage (1 semaine, 3 lots)

| ID | Tâche | Lot | Files | Effort | Statut | Owner | Commit |
|----|-------|-----|-------|--------|--------|-------|--------|
| R4.1 | **GitHub Actions CI/CD** (priorité ABSOLUE) | 4A | `.github/workflows/ci.yml` (new) | ½ j | ⬜ | — | — |
| R4.2 | Centraliser magic numbers `threshold_tuning.go` | 4A | `internal/config/threshold_tuning.go` (new) | 2 h | ⬜ | — | — |
| R4.5 | Fusionner `sysmon/` dans `metrics/` | 4A | `internal/sysmon/` → `internal/metrics/system/` | 1 h | ⬜ | — | — |
| R4.6 | `ExitAction` enum typé | 4A | `cmd/fibcalc/main.go`, `internal/app/` | 30 min | ⬜ | — | — |
| R4.11 | Fusionner property test ↔ unit test fibonacci | 4A | `internal/fibonacci/fibonacci_test.go`, `fibonacci_property_test.go` | 30 min | ⬜ | — | — |
| R4.3 | `envOverrides` via reflection | 4B | `internal/config/env.go` | 4 h | ⬜ | — | — |
| R4.4 | Centraliser erreurs via `HandleCalculationError` | 4B | `internal/app/calculate.go`, `internal/errors/handler.go` | 2 h | ⬜ | — | — |
| R4.7 | Pré-flight memory check dans `Calculator` | 4B | `internal/fibonacci/calculator.go` | 1,5 h | ⬜ | — | — |
| R4.8 | RingBuffer pour `tui/logs.go` | 4B | `internal/tui/logs.go` | 2 h | ⬜ | — | — |
| R4.12 | Étendre `t.Parallel()` aux tests file-based | 4B | divers `*_test.go` | 1 h | ⬜ | — | — |
| R4.9 | Bridge `Send` retourne `error` | 4C | `internal/tui/bridge.go` | 1 h | ⬜ | — | — |
| R4.10 | Layout TUI adaptatif <80 cols | 4C | `internal/tui/model.go` | 1 h | ⬜ | — | — |

### Synthèse de progression (à mettre à jour)

| Vague | Total tâches | ⬜ Pending | 🟡 InProgress | 🔵 InReview | ✅ Done | ⚠ Blocked | ⛔ Skipped |
|-------|--------------|-----------|--------------|-------------|---------|-----------|-----------|
| 1 | 5 | 0 | 0 | 0 | **5** | 0 | 0 |
| 2 | 10 | 10 | 0 | 0 | 0 | 0 | 0 |
| 3 | 10 | 10 | 0 | 0 | 0 | 0 | 0 |
| 4 | 12 | 12 | 0 | 0 | 0 | 0 | 0 |
| **Total** | **37** | **32** | **0** | **0** | **5** | **0** | **0** |

---

## 2. Méthodologie d'exécution parallèle

### 2.1 Principes

1. **Un lot = un envoi de message avec N appels d'`Agent` parallèles.** Le framework Claude permet jusqu'à ~10 agents simultanés.
2. **Conflits de fichiers évités par batching.** Chaque lot est conçu pour qu'aucun agent ne touche le même fichier qu'un autre du même lot.
3. **Une PR par tâche.** Évite les rebases massifs ; permet un revert fin si régression.
4. **Validation systématique** après chaque lot (`make test -race && make lint && make benchmark`).
5. **Tableau mis à jour à chaque transition de statut.**

### 2.2 Matrice de conflits (par fichier sensible)

Cette matrice identifie les fichiers touchés par plusieurs tâches — sources potentielles de conflits si lancés en parallèle.

| Fichier | Tâches | Stratégie |
|---|---|---|
| `internal/bigfft/pool.go` | R1.4, R2.3 | Sequential : R1.4 (vague 1) → R2.3 (vague 2). |
| `internal/bigfft/fft_cache.go` | R1.5, R2.10, R3.9 | Sequential : R1.5 (V1) → R2.10 (V2) → R3.9 (V3). |
| `internal/bigfft/allocator.go` | R2.4, R3.7 | Sequential : R2.4 (V2) → R3.7 (V3). |
| `internal/bigfft/fft.go`, `fft_poly.go`, `fft_recursion.go`, `fft_core.go` | R2.8, R3.7 | R2.8 d'abord (fusion), puis R3.7 (FFTContext). |
| `internal/fibonacci/fastdoubling.go` | R1.1, R3.10 | R1.1 d'abord (fix critique), puis R3.10 (rename). |
| `internal/fibonacci/doubling_framework.go` | R3.2 | Isolé — lot 3D. |
| `internal/cli/completion.go` | R3.5 | Isolé. |
| `internal/tui/model.go` | R3.4, R4.10 | R3.4 d'abord (décomposition), puis R4.10 (layout). |
| `internal/app/calculate.go` | R3.6, R4.4 | R3.6 d'abord (split), puis R4.4 (handler centralisation). |
| `internal/config/env.go` | R4.3 | Isolé. |

### 2.3 Cadre de dispatch d'un lot

```pseudo
PRE :
  - Vérifier que tous les blocking tasks de la vague précédente sont ✅
  - Snapshot baseline (test, lint, bench) si transition de vague

DISPATCH :
  - Émettre un message unique avec N appels Agent (général ou sub-agent)
  - Chaque agent reçoit : (a) prompt isolé, (b) liste de fichiers exclusivement à toucher, (c) critères de validation

POST :
  - Marquer chaque tâche 🔵 InReview après retour de l'agent
  - Lancer make test -race && make lint
  - Si vert : merge → marquer ✅ Done, commit SHA dans le tableau
  - Si rouge : marquer ⚠ Blocked + raison, investiguer
```

### 2.4 Procédure de mise à jour du tableau

À chaque transition de statut :

1. Modifier la cellule `Statut` de la ligne concernée (ex : `⬜` → `🟡`).
2. Renseigner `Owner` (nom de l'agent dispatché ou humain).
3. À la complétion, renseigner `Commit` (SHA court, ex : `a1b2c3d`).
4. Mettre à jour le **tableau de synthèse** en bas de la section 1.
5. Commit du document : `docs(plan): R1.x marked ✅ Done`.

---

## 3. Vague 1 — Fixes critiques (1 lot, 5 agents en parallèle)

**Objectif :** corriger 5 bugs latents avant tout autre refactoring. Tous indépendants → un seul lot.

### Lot 1A — 5 agents en parallèle

#### R1.1 — Fix `clearStateAliases` toujours appelé

```yaml
agent: general-purpose
prompt: |
  Dans internal/fibonacci/fastdoubling.go (autour des lignes 215-329), la fonction
  ReleaseStateWithResult/ReleaseState appelle clearStateAliases (lignes 322-328) UNIQUEMENT
  dans le chemin nominal. Lorsque overLimit=true, les slots aliasants ne sont pas nettoyés
  → use-after-free latent.

  TÂCHE :
  1. Lire intégralement internal/fibonacci/fastdoubling.go.
  2. Identifier toutes les branches de retour dans ReleaseState/ReleaseStateWithResult.
  3. Garantir que clearStateAliases est TOUJOURS appelée avant que la state ne retourne au pool,
     quelle que soit la branche (overLimit, error, nominal).
  4. Pattern recommandé : defer clearStateAliases(s) en début de fonction.
  5. Ajouter un test ciblé : forcer overLimit=true, libérer la state, ré-acquérir,
     vérifier que les aliases ne pointent pas vers l'ancienne arena.

  CONTRAINTES :
  - Ne pas toucher à d'autres fichiers que fastdoubling.go (+ test).
  - Ne pas modifier la signature publique.
  - Bench Fib(10M) avant/après doit rester dans -2 % de la baseline.

  VALIDATION :
  - go test -race -count=10 ./internal/fibonacci/
  - go test -bench=BenchmarkFastDoubling -benchmem ./internal/fibonacci/
  - golden tests verts.
files: internal/fibonacci/fastdoubling.go (+ nouveau _test.go)
```

#### R1.2 — `defer gc.End()` panic-safe

```yaml
agent: general-purpose
prompt: |
  Dans internal/fibonacci/memory/gc_control.go (lignes 63-97), GCController.Begin()
  désactive le GC mais si un panic survient avant End(), le GC reste off pour la vie
  du processus.

  TÂCHE :
  1. Lire gc_control.go intégralement.
  2. Wrapper le pattern Begin/End en une méthode WithGC(fn func() error) error qui :
     - Begin() au début.
     - defer End() pour garantir la restauration même en cas de panic.
     - retourne l'erreur de fn ou panic recover.
  3. Garder Begin/End publiques pour rétrocompatibilité, mais documenter Deprecated:.
  4. Migrer le seul caller (probablement internal/fibonacci/calculator.go) vers WithGC.
  5. Test unitaire : panic dans fn() doit restaurer le GC.

  CONTRAINTES :
  - Ne pas casser l'API publique de gc_control.go.
  - Vérifier debug.SetGCPercent retourné != 0 avant restauration.

  VALIDATION :
  - go test -race ./internal/fibonacci/memory/
  - go test ./internal/fibonacci/ (calculator_test).
  - Test avec panic injecté dans le calcul.
files: internal/fibonacci/memory/gc_control.go (+ test), internal/fibonacci/calculator.go
```

#### R1.3 — Invalidation profil via `IsStale`

```yaml
agent: general-purpose
prompt: |
  Dans internal/calibration/calibration.go, la fonction AutoCalibrateWithProfile
  (autour des lignes 251-321) charge un profil de calibration sans vérifier sa
  fraîcheur. profile.IsStale(maxAge) existe (profile.go:167-172) mais n'est
  jamais appelée.

  TÂCHE :
  1. Lire calibration.go (391 L) et profile.go (206 L).
  2. Ajouter une constante ProfileMaxAge = 7 * 24 * time.Hour (à valider).
  3. Modifier AutoCalibrateWithProfile pour invalider et recalibrer si IsStale.
  4. Logger explicitement « Profile stale, re-calibrating » via le runner.
  5. Test : créer un profil daté de 2 semaines, vérifier que la re-calibration s'enclenche.

  CONTRAINTES :
  - Ne pas changer la structure du fichier profile JSON.
  - ProfileMaxAge doit être surchargeable via env (FIBCALC_PROFILE_MAX_AGE).

  VALIDATION :
  - go test ./internal/calibration/
  - Test E2E avec FIBCALC_PROFILE_MAX_AGE=1ms force la recalibration.
files: internal/calibration/calibration.go (+ test), internal/calibration/profile.go
```

#### R1.4 — Fix `releaseWordSlice` (bigfft)

```yaml
agent: general-purpose
prompt: |
  Dans internal/bigfft/pool.go (lignes 111-123), releaseWordSlice() peut perdre
  silencieusement un slice si sa capacité ne matche plus exactement wordSliceSizes[idx].
  Cela neutralise le pooling sous charge.

  TÂCHE :
  1. Lire pool.go (467 L) intégralement.
  2. Modifier releaseWordSlice :
     - Si idx >= 0 : restaurer la pleine capacité du slice (slice = slice[:cap])
       avant Put dans le pool de l'index correct.
     - Si la capacité ne correspond à AUCUN bucket : laisser le GC traiter
       (pas de Put), et incrémenter une métrique « pool_miss ».
  3. Ajouter un test : forcer un slice resizé, vérifier qu'il est bien renvoyé
     dans le bucket correspondant à sa capacité réelle.

  CONTRAINTES :
  - Ne pas modifier les pools eux-mêmes ni leurs tailles.
  - Pas de panique sur cap=0.

  VALIDATION :
  - go test -race -count=100 ./internal/bigfft/
  - go test -bench=BenchmarkPool ./internal/bigfft/
  - Vérifier que pool_miss reste à 0 sur le bench standard.
files: internal/bigfft/pool.go (+ test)
```

#### R1.5 — Fix `putByKey` eager alloc (bigfft)

```yaml
agent: general-purpose
prompt: |
  Dans internal/bigfft/fft_cache.go (lignes 262-303), putByKey alloue un nouveau
  buffer contigu MÊME quand le cache est plein et qu'on doit évincer. Le buffer
  évincé (souvent de la bonne taille) est jeté au GC.

  TÂCHE :
  1. Lire fft_cache.go (534 L) intégralement.
  2. Modifier putByKey pour :
     - Avant d'allouer : récupérer le buffer évincé (s'il existe).
     - Si len/cap suffisants → réutiliser ; sinon allouer.
     - L'eviction passe le buffer libéré pour réutilisation.
  3. Test : remplir la cache, insérer un nouvel élément, vérifier que
     pour une taille équivalente, runtime.NumGCMallocs n'augmente pas.

  CONTRAINTES :
  - Ne pas changer la sémantique LRU.
  - Ne pas casser la thread-safety (RWMutex préservé).

  VALIDATION :
  - go test -race ./internal/bigfft/
  - Bench BenchmarkTransformCache_Put → -10 à -20 % allocations.
files: internal/bigfft/fft_cache.go (+ test)
```

### Validation Lot 1A (gate vers Vague 2)

```bash
make test -race        # Tous tests verts
make lint              # 0 nouvelle violation
make benchmark         # Ne pas régresser > 5 %
make coverage          # Couverture maintenue ou +
```

Mettre à jour les 5 lignes du tableau (Vague 1) en ✅ Done avec leur SHA respectif.

---

## 4. Vague 2 — Suppressions et fusions (2 lots, 10 agents au total)

### Lot 2A — Indépendants par package (5 agents en parallèle)

| ID | Package(s) touché(s) | Justification du parallélisme |
|---|---|---|
| R2.1 | `fibonacci/` | Suppression d'un fichier d'alias + migration imports. |
| R2.2 | `parallel/` | Suppression d'un package mort. |
| R2.4 | `bigfft/allocator.go` | Suppression d'un wrapper isolé. |
| R2.6 | `cli/` | Refactor interne CLI. |
| R2.7 | `ui/` + `tui/styles.go` | Fusion thématique. |

#### Prompts d'agent (extraits — formats identiques)

**R2.1** — `Supprimer internal/fibonacci/progress_aliases.go ; remplacer chaque usage par un import direct de internal/progress dans calculator.go et autres dépendants. Vérifier que go build et go test ./... passent.`

**R2.2** — `Vérifier qu'aucun fichier .go (hors *_test.go dans internal/parallel/) n'importe internal/parallel. Si confirmé, supprimer entièrement le package internal/parallel/. Si un caller existe, mettre la tâche en ⚠ Blocked et reporter.`

**R2.4** — `Dans internal/bigfft/allocator.go, faire que BumpAllocator implémente directement TempAllocator (méthode AllocFermatTemp). Supprimer BumpAllocatorAdapter et NewBumpAllocatorAdapter. Migrer les callers.`

**R2.6** — `Fusionner internal/cli/ui_display.go et internal/cli/output.go en un unique internal/cli/display.go. Conserver internal/cli/presenter.go intact. Vérifier qu'aucune fonction publique n'est perdue.`

**R2.7** — `Fusionner internal/ui/colors.go, internal/ui/themes.go et internal/tui/styles.go en un unique package internal/ui/style/ avec une struct Style hybride (CLI ANSI + TUI lipgloss). Migrer tous les callers. Préserver le support NO_COLOR.`

### Validation Lot 2A — gate vers Lot 2B

`make test -race && make lint && make build && make build-pgo`

### Lot 2B — Réfactos non conflictuels (5 agents en parallèle, après 2A)

| ID | Package(s) | Justification |
|---|---|---|
| R2.3 | `bigfft/{scan,cpu_amd64,arith_amd64,pool}.go` | Suppressions disjointes des autres bigfft. |
| R2.5 | `orchestration/`, `format/` | Fusion progress orthogonale à 2A. |
| R2.8 | `bigfft/{fft,fft_poly}.go` | Fusion FFT mul (gros). |
| R2.9 | `fibonacci/fft.go` | Fusion `executeFFTTransforms{P,S}`. |
| R2.10 | `bigfft/fft_cache.go` | Hash FNV consolidation (n'overlap pas avec R3.9 qui viendra plus tard). |

⚠ R2.3 et R2.10 touchent le même package `bigfft/` mais des fichiers disjoints. **Vérifier l'absence de conflit avant merge** (rebase préalable au merge).

#### Prompts d'agent (extraits)

**R2.3** — `Supprimer internal/bigfft/scan.go (88 L, sans caller). Supprimer la fonction getWordSlicePoolIndexLinear de internal/bigfft/pool.go (jamais appelée). Évaluer si internal/bigfft/cpu_amd64.go peut être supprimé (HasAVX2/HasAVX512 non utilisés). Si oui, supprimer aussi les checks triviaux len(z)==0 dans internal/bigfft/arith_amd64.go. Documenter chaque suppression dans le commit.`

**R2.5** — `Fusionner internal/format/progress_eta.go::ProgressWithETA dans internal/orchestration/progress.go::ProgressAggregator. Supprimer le wrapper redondant. Garder format/numbers.go, format/duration.go intacts. Migrer les callers de ProgressWithETA.`

**R2.8** — `Fusionner internal/bigfft/fft.go::fftmulTo et internal/bigfft/fft_poly.go::Poly.mul en un unique chemin. Préserver le mode cache + bump allocator. -100 à -150 LOC attendus. ATTENTION : bench obligatoire avant/après, gate à -5 % latency.`

**R2.9** — `Fusionner internal/fibonacci/fft.go::executeFFTTransformsParallel et executeFFTTransformsSequential en un unique executeFFTTransforms(ctx, ..., inParallel bool). Tests existants doivent rester verts.`

**R2.10** — `Dans internal/bigfft/fft_cache.go, extraire un cacheKeyBuilder type qui consolide fnvWriteUint64, computeCacheKey et computePolyKey en une seule implémentation FNV-1a. -30 LOC.`

### Validation Lot 2B (gate vers Vague 3)

```bash
make test -race
make lint
make benchmark | tee docs/audits/post-vague2.txt
diff docs/audits/baseline-pre-refactor.txt docs/audits/post-vague2.txt
# Acceptation : pas de régression > 5 % sur Fib(10M), Fib(1M), Fib(100k)
```

---

## 5. Vague 3 — Décompositions structurelles (4 lots)

Plus risquée, plus longue. Décomposée en 4 lots pour limiter les conflits.

### Lot 3A — Décompositions indépendantes (5 agents en parallèle)

| ID | Package | Effort |
|---|---|---|
| R3.1 | `fibonacci/threshold/` | 4 h |
| R3.3 | `calibration/` | 3 h |
| R3.4 | `tui/` | 1,5 j |
| R3.5 | `cli/completion.go` | 4 h |
| R3.10 | `fibonacci/` (rename) | 1 h |

**R3.1** — `Décomposer internal/fibonacci/threshold/manager.go (417 L) en 3 types dans 3 fichiers : metrics_buffer.go (ring buffer), analyzer.go (logique pure : hystérésis, speedup), manager.go (orchestration + getters thread-safe). Préserver l'API publique du Manager.`

**R3.3** — `Refactorer internal/calibration/calibration.go::AutoCalibrateWithProfile en pattern Strategy : interface CalibrationStrategy avec FastStrategy (micro-bench) et CompleteStrategy (full run). Un orchestrateur escalade Fast → Complete si confidence < 0.5. Préserver le comportement observable.`

**R3.4** — `Décomposer internal/tui/model.go (425 L). Créer internal/tui/component/ avec logs.go, chart.go, metrics.go, header.go, footer.go (chacun avec ses propres Update/View/messages). model.go devient un router (~200 L). Tests snapshot Bubble Tea à ajouter.`

**R3.5** — `Éclater internal/cli/completion.go (520 L) en internal/cli/completion/{registry,bash,zsh,fish,powershell}.go. registry.go contient flagRegistry (source unique). Chaque fichier shell expose Generate(out io.Writer, algorithms []string) error.`

**R3.10** — `Renommer OptimizedFastDoubling → FastDoublingCalculator, MatrixExponentiation → MatrixExponentiationCalculator, FFTBasedCalculator inchangé. Migrer le registry, les imports, et les tests. Aucun changement comportemental.`

### Lot 3B — Refactorings ciblés (3 agents en parallèle, après 3A)

| ID | Package | Pourquoi pas en 3A |
|---|---|---|
| R3.6 | `app/`, `orchestration/` | Trop de touches dans `app/calculate.go`. |
| R3.8 | `bigfft/fermat.go` | Préparation pour R3.7 (besoin d'errors). |
| R3.9 | `bigfft/fft_cache.go` | Réfactor ciblé indépendant. |

**R3.6** — `Scinder internal/app/calculate.go::runCalculate (215 L, 5 responsabilités). Extraire validateMemoryBudget vers internal/config/validator.go. Extraire runLastDigits vers internal/orchestration/lastdigits.go (logique pure). Scinder analyzeResultsWithOutput en selectBest+present+save.`

**R3.8** — `Convertir les panics de internal/bigfft/fermat.go (lignes 23-46, 151-206 : norm, Shift, Mul, Sqr) en errors. Propager les errors vers Poly.mul, transform et fonctions appelantes (fft.go, fft_poly.go, fft_recursion.go). Maintenir le contrat de panic UNIQUEMENT pour les invariants de programmation interne (pas de tailles invalides venant de l'API publique).`

**R3.9** — `Optimiser internal/bigfft/fft_cache.go::getByKey en fast-path lock-free pour les hits : tenter une lecture sans lock (atomic ou RLock court), ne prendre Lock QUE pour la mise à jour LRU. Bench : +10-20 % cache hit latency attendu.`

### Lot 3C — `bigfft.FFTContext` (1 agent isolé, après 3B)

**R3.7** — `Refactor MAJEUR de internal/bigfft/. Introduire un type FFTContext qui encapsule TransformCache + TempAllocator + Semaphore. Exposer NewFFTContext(opts) public. Garder une variable globale defaultContext pour rétrocompatibilité (fftmul, Mul, Sqr utilisent defaultContext si pas de contexte fourni). Ajouter MulWithContext, SqrWithContext. Tester l'isolation : 2 contextes parallèles ne se cross-contaminent pas. CRITIQUE : bench complet, gate à -5 % latency.`

### Lot 3D — Découplage final (1 agent, après 3C)

**R3.2** — `Découpler internal/fibonacci/doubling_framework.go::ExecuteDoublingLoop de bigfft. Extraire une interface CacheStrategy { CheckAndAdapt(bitLen int, hitRate float64) error } injectable dans DoublingFramework. L'implémentation par défaut wrappe bigfft.GetTransformCache. La boucle critique passe de 137 L à ~80 L.`

### Validation Vague 3

```bash
make test -race
make lint
make benchmark | tee docs/audits/post-vague3.txt
# Acceptation : amélioration ou stabilité sur tous les benchs
```

---

## 6. Vague 4 — Polissage & outillage (3 lots)

### Lot 4A — Indépendants (5 agents en parallèle, **R4.1 PRIORITÉ ABSOLUE**)

**R4.1 doit être lancée en PREMIER, idéalement avant TOUTE vague.** Sans CI, on ne détecte pas les régressions des autres tâches. Ce plan présume R4.1 livrée AVANT le début de la Vague 1 si possible, sinon en premier de la Vague 4.

**R4.1** — `Créer .github/workflows/ci.yml :
- Triggers : push, pull_request.
- Matrix : go 1.25.x × os [ubuntu, windows, macos].
- Steps : actions/checkout, actions/setup-go, make test (race), make lint (golangci-lint), make build, optionnel make benchmark sur ubuntu uniquement.
- Upload artifact coverage.html.
- Badge à ajouter au README.`

**R4.2** — `Créer internal/config/threshold_tuning.go avec une struct ThresholdTuningProfile documentant : FFTSpeedupThreshold (1.2), ParallelSpeedupThreshold (1.1), HysteresisMargin (0.15), MinFFTThreshold (100000), MinParallelThreshold (1024), MemoryLimitMultiplier (3), MicroBenchTimeout (150ms). Migrer les usages depuis threshold/manager.go, memory/gc_control.go, calibration/microbench.go.`

**R4.5** — `Fusionner internal/sysmon/ dans internal/metrics/. Créer un sous-package internal/metrics/system/ pour conserver la frontière sémantique (Fibonacci-metrics vs OS-metrics). Migrer les callers, mettre à jour doc.go.`

**R4.6** — `Remplacer le sentinel exitVersion = -1 dans cmd/fibcalc/main.go par un type ExitAction enum { ActionSuccess, ActionError, ActionVersionHandled }. internal/app/Run() retourne ExitAction.`

**R4.11** — `Fusionner internal/fibonacci/fibonacci_property_test.go dans internal/fibonacci/fibonacci_test.go (ou inversement, selon ce qui est le plus naturel). Supprimer la duplication de cas de tests qui couvrent les mêmes propriétés.`

### Lot 4B — Indépendants (5 agents en parallèle, après 4A)

**R4.3** — `Refactorer internal/config/env.go::envOverrides (lignes 111-180) en utilisant reflection sur les struct tags de internal/config/AppConfig. Une seule source de vérité = AppConfig avec tags `config:"NAME" parser:"type"`. Garder les tests verts.`

**R4.4** — `Centraliser tous les fmt.Fprintf direct dans internal/app/calculate.go vers internal/errors/handler.HandleCalculationError. Vérifier la couverture des codes de sortie ExitErrorMismatch, ExitErrorTimeout.`

**R4.7** — `Dans internal/fibonacci/calculator.go, ajouter une méthode publique CanCalculate(n uint64, memLimit uint64) bool qui appelle internal/fibonacci/memory.EstimateMemoryUsage et compare. Invoquer ce check au début de Calculate() si --memory-limit est défini. Abort gracieux au lieu de OOM.`

**R4.8** — `Remplacer le slice trim de internal/tui/logs.go (maxLogEntries=10000) par un RingBuffer (réutiliser le pattern de internal/tui/sparkline.go ou en extraire un type partagé internal/tui/ringbuffer.go).`

**R4.12** — `Ajouter t.Parallel() à tous les *_test.go file-based qui n'en ont pas (golden, CLI output, etc.). Exclure les tests qui modifient des globals ou l'environnement.`

### Lot 4C — TUI ciblé (2 agents en parallèle, après 4B)

**R4.9** — `Dans internal/tui/bridge.go::programRef.Send (lignes 31-39), retourner error au lieu de drop silencieux. Logger via log.Printf si program==nil. Vérifier les callers et propager.`

**R4.10** — `Dans internal/tui/model.go (LayoutManager), adapter le layout pour width<80 (single column) et height<20 (panneaux compacts). Préserver les tests Bubble Tea.`

### Validation finale

```bash
make all                                    # clean + build + test
make benchmark | tee docs/audits/final.txt
diff docs/audits/baseline-pre-refactor.txt docs/audits/final.txt
make coverage                               # ≥ baseline
```

Mettre à jour le tableau de synthèse en bas de la section 1 → idéalement 37/37 ✅.

---

## 7. Procédure complète de gestion d'une tâche

Pour chaque tâche, suivre ce cycle :

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. PRE-CHECK                                                    │
│    - Vérifier dépendances (autres tâches blockers : ✅)         │
│    - Marquer la tâche 🟡 InProgress dans le tableau             │
│    - Créer une branche feature : git checkout -b refactor/Rx.y  │
├─────────────────────────────────────────────────────────────────┤
│ 2. DISPATCH                                                     │
│    - Envoyer le prompt à un agent (general-purpose)             │
│    - L'agent lit, modifie, ajoute des tests, documente          │
├─────────────────────────────────────────────────────────────────┤
│ 3. REVIEW                                                       │
│    - Marquer 🔵 InReview                                        │
│    - Examiner le diff (trust but verify)                        │
│    - Vérifier conformité avec les contraintes du prompt         │
├─────────────────────────────────────────────────────────────────┤
│ 4. VALIDATE                                                     │
│    - make test -race                                            │
│    - make lint                                                  │
│    - make benchmark (si tâche perf-sensitive)                   │
│    - Comparer aux baselines                                     │
├─────────────────────────────────────────────────────────────────┤
│ 5. MERGE                                                        │
│    - PR sur main, review optionnelle d'un humain                │
│    - Squash-merge avec message « refactor(Rx.y): description »  │
│    - Marquer ✅ Done + commit SHA dans le tableau               │
│    - Mettre à jour le tableau de synthèse                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. Prompt template réutilisable (boilerplate)

Pour ajouter une tâche similaire ou réessayer une tâche bloquée :

```yaml
agent: general-purpose
prompt: |
  Tu es un Go senior chargé d'une tâche de refactoring isolée dans le repo FibGo.

  CONTEXTE :
  Le repo est documenté dans Claude.md. Le plan de refactoring complet est dans
  ultrareviewplan.md. L'audit source est dans ultrareview.md.

  TÂCHE [Rx.y] :
  <description précise de la tâche>

  FICHIERS AUTORISÉS À MODIFIER :
  - <liste exhaustive>

  FICHIERS À NE PAS TOUCHER :
  - Tous les autres (notamment les fichiers d'autres lots du même batch).

  CONTRAINTES :
  - Ne pas introduire de dépendance externe (go.mod inchangé sauf accord explicite).
  - Préserver l'API publique sauf indication contraire.
  - Ajouter des tests unitaires couvrant le changement.
  - Pas d'emoji dans le code.
  - Commenter UNIQUEMENT lorsque le « pourquoi » n'est pas évident.

  VALIDATION OBLIGATOIRE :
  - go test -race ./<package>/
  - go vet ./<package>/
  - golangci-lint run ./<package>/

  LIVRABLE ATTENDU :
  - Résumé du diff (fichiers modifiés, ajoutés, supprimés).
  - Confirmation que tests + lint passent.
  - Métrique perf si applicable (avant/après).
```

---

## 9. Risques opérationnels & mitigation

| Risque | Mitigation |
|---|---|
| Conflits de merge entre lots parallèles | Matrice de conflits (§ 2.2), rebase préalable au merge. |
| Régression silencieuse sans CI | R4.1 en priorité absolue. |
| Bench inconsistant entre runs | Lancer `make benchmark` 3× et prendre la médiane. |
| Agent corrompt le golden test | Sentinel : `internal/fibonacci/testdata/fibonacci_golden.json` est immuable sans approbation explicite. Tâche → ⚠ Blocked si modifié. |
| Refactor casse PGO profile | `make build-pgo` au moins une fois par vague. |
| Burnout : tableau jamais à jour | Mettre à jour à chaque transition (process discipliné). |

---

## 10. Annexes

### 10.1 Commandes utiles

```bash
# Snapshot baseline
make benchmark > docs/audits/baseline-$(date +%Y%m%d).txt
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

# Vérifier l'absence de cycles
go list -f '{{.ImportPath}} -> {{.Imports}}' ./... | grep -E "fibcalc.*fibcalc"

# Comparer benchmarks (nécessite benchstat)
benchstat docs/audits/baseline.txt docs/audits/post-vague.txt

# Vérifier qu'aucun internal/ ne fuit vers cmd/
go list -deps ./cmd/fibcalc | grep -v 'github.com/agbru/fibcalc/internal/app' | grep 'fibcalc/internal/'
```

### 10.2 Référence rapide vers `ultrareview.md`

| Section ultrareview.md | Vague(s) impactée(s) |
|---|---|
| § 4.1 fibonacci core | R1.1, R2.1, R2.9, R3.10, R3.2 |
| § 4.2 bigfft | R1.4, R1.5, R2.3, R2.4, R2.8, R2.10, R3.7, R3.8, R3.9 |
| § 4.3 memory/threshold/calibration | R1.2, R1.3, R3.1, R3.3, R4.2 |
| § 4.4 orchestration/progress/parallel | R2.2, R2.5 |
| § 4.5 cli/tui/ui/format | R2.6, R2.7, R3.4, R3.5, R4.8, R4.9, R4.10 |
| § 4.6 cmd/app/config/errors | R3.6, R4.3, R4.4, R4.5, R4.6 |
| § 5.4 CI/CD | R4.1 |
| § 5.5 dette tests | R4.11, R4.12 |

### 10.3 Estimation effort exécution parallèle

Avec dispatch parallèle agressif (5 agents simultanés par lot) :

| Vague | Effort séquentiel | Effort parallèle (5 agents) | Wall-time |
|---|---|---|---|
| 1 | ½ jour | ~2 h (max(R1.5)=2h) | ½ jour |
| 2 | 1 semaine | ~2 jours (max(R2.5,R2.8)=1j × 2 lots) | 2-3 jours |
| 3 | 3 semaines | ~5 jours (lot 3A=1,5j + 3B=4h + 3C=1j + 3D=1j) | ~1 semaine |
| 4 | 1 semaine | ~1,5 jour (3 lots de ½ j chacun) | ~2 jours |
| **Total** | **~5,5 sem.** | **~2 sem.** | **2-2,5 sem.** |

Soit un **gain de ~3 semaines** par rapport à l'exécution séquentielle.

---

**Fin du plan d'exécution.**

Mettre à jour le tableau de la section 1 à chaque transition de statut, et signaler tout blocage dans le commit message du document (`docs(plan): R3.7 ⚠ Blocked, FFTContext breaks BenchmarkFib10M -8%`).
