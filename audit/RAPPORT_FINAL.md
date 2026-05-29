# RAPPORT FINAL — Audit Go du dépôt FibGo (FibCalc)

> Audit multi-agents (5 axes), lecture seule, commit `866b8cd` (2026-05-24), Go 1.26.3 / module `go 1.26.0`. Date : 2026-05-28. Langue : FR-CA. Marqueurs épistémiques sur chaque constat.
>
> Détail par axe : [`01_correctness.md`](01_correctness.md) · [`02_concurrence.md`](02_concurrence.md) · [`03_performance.md`](03_performance.md) · [`04_idiomatique.md`](04_idiomatique.md) · [`05_structure_tests_ci.md`](05_structure_tests_ci.md). Méthodologie : [`00_bootstrap.md`](00_bootstrap.md), [`CONVENTIONS.md`](CONVENTIONS.md).

---

## 1) Verdict global

**Le dépôt est correct, performant et bien couvert — au cœur.** Aucun écart de résultat détecté : golden tests (3 calculateurs × N jusqu'à 200 000), **7 cibles fuzz toutes vertes** (0 crasher), property tests (Cassini, récurrence, GCD) et cross-validation Fast Doubling/Matrix/FFT tiennent solidement l'exactitude. Couverture **mesurée à 88,6 %** (> seuil documenté 80 %). **Aucune régression de performance** vs baselines `docs/audits/` : le hot path Fast Doubling à N=1M alloue même *moins* qu'avant (102-110 vs 132 allocs/op). `go vet ./...` est vert.

**Le risque #1 — et seul constat CRITIQUE — est le contrôle GC non *concurrency-safe*** (`internal/fibonacci/memory/gc_control.go`), invoqué concurremment en mode comparaison (`--algo all`, N ≥ 1M) : la restauration de `GOGC` devient non déterministe et le garde-fou mémoire (`SetMemoryLimit`) est **retiré pendant qu'un calculateur frère tourne encore**. *Vérifié par lecture de code* (orchestrateur + calculateur + gc_control) ; l'aspect data race reste **[à vérifier]** sous `-race` (indisponible : `CGO_ENABLED=0`). Le multiplicateur de risque #2 est **structurel : aucune CI distante** ne capte ce genre de régression — toute la rigueur repose sur la discipline locale.

**En une phrase :** noyau algorithmique sain et rapide, mais **deux trous de garde-fou** (sûreté GC en concurrence + absence de CI) à fermer avant d'élever ce prototype au rang de référence.

---

## 2) Tableau de bord

### Constats par sévérité et par axe (45 au total)

| Axe | CRITIQUE | MAJEUR | MINEUR | INFORMATIF | Total |
|---|:--:|:--:|:--:|:--:|:--:|
| 1 — Correctness | 0 | 2 | 3 | 3 | 8 |
| 2 — Concurrence | **1** | 2 | 2 | 2 | 7 |
| 3 — Performance | 0 | 1 | 2 | 4 | 7 |
| 4 — Idiomatique | 0 | 1 | 7 | 5 | 13 |
| 5 — Structure/Tests/CI | 0 | 2 | 4 | 4 | 10 |
| **Total** | **1** | **8** | **18** | **18** | **45** |

### Indicateurs mesurés

| Indicateur | Valeur | Marqueur |
|---|---|---|
| Build (`go build ./...`) | propre, exit 0 | [confirmé] |
| `go vet ./...` | exit 0, aucun diagnostic | [confirmé] |
| Couverture totale | **88,6 %** (doc : 87,5 %) | [confirmé] |
| Golden tests | PASS (N jusqu'à 200k, 3 calculateurs) | [confirmé] |
| Fuzz (7 cibles) | PASS, 0 crasher | [confirmé] |
| Régression perf vs baselines | aucune ; allocs en amélioration | [confirmé] |
| Vulnérabilités (govulncheck) | aucune signalée | [probable] |

### Disponibilité de l'outillage durant l'audit

| Capacité | Dispo | Note |
|---|:--:|---|
| `go build` / `test` / `vet` / `bench` / `fuzz` | ✅ | toolchain go1.26.3 locale |
| `go test -race` | ❌ | `CGO_ENABLED=0`, ni gcc ni clang → constats de race **[à vérifier]** |
| `gofmt` | ⚠️ | faux positifs CRLF (`core.autocrlf=true`) ; code commité gofmt-propre |
| `staticcheck` 2026.1 | ✅ | recompilé go1.26.3 au bootstrap |
| `golangci-lint` v1.64.8 | ✅ | recompilé go1.26.3 ; config `.golangci.yml` v1 |
| `gosec`, `govulncheck` | ✅ | — |

---

## 3) Synthèse par axe (du plus critique au moins critique)

### Axe 2 — Concurrence & data races  *(1 CRITIQUE, 2 MAJEUR)*

**Verdict :** la machinerie du hot path (sémaphores bornés, errgroup, `sync.Pool`, seuils `atomic`, cache LRU sous `RWMutex`) est saine et `go vet` ne signale rien. La faille sérieuse est l'**état GC global** muté concurremment.

- **`A2-01` — CRITIQUE — Contrôle GC global muté concurremment en mode comparaison.** `[confirmé]` (logique) / `[à vérifier]` (race).
  *Preuve (vérifiée par l'orchestrateur) :* `gc_control.go:94-128` appelle `debug.SetGCPercent(-1)` + `debug.SetMemoryLimit(...)` — **état au niveau processus** — et mémorise `originalGCPercent` par contrôleur. `calculator.go:218,260` instancie un `GCController` **par calculateur** et l'exécute via `WithGC`. `orchestrator.go:55-72` lance les calculateurs **concurremment** (`errgroup.Go`) dès `len(Calculators) > 1`. GC auto s'active à N ≥ 1 000 000.
  *Impact :* deux `Begin/End` entrelacés → le 2ᵉ capte `-1` comme « valeur d'origine » et **laisse GC désactivé** ; et le 1ᵉʳ `End` exécute `SetMemoryLimit(MaxInt64)`, **retirant le filet OOM** alors qu'un calculateur frère calcule encore un très grand nombre.
  *Recommandation :* sérialiser le contrôle GC (compteur global `atomic` + `sync.Once`-like : seul le 1ᵉʳ entrant désactive, le dernier sortant restaure), **ou** déplacer le contrôle GC au niveau orchestration (une seule fois autour de toute la comparaison) plutôt que par calculateur. Confirmer d'abord sous `-race` (Linux/WSL).
- **`A2-03` — MAJEUR — Récursion FFT parallèle : `context` non propagé** (`bigfft/fft_recursion.go:93-169`). Annulation/timeout ignorés une fois dans la récursion. `[confirmé]`
- **`A2-02` — MAJEUR — `SetCacheLogger` écrit `tc.logger` hors verrou**, lu par `logPeriodicStats` (`fft_cache.go:98-101` vs `286-307`). `[à vérifier]`
- Mineurs/info : `threshold.SetTuning` mute des globaux sans synchro (`A2-04`, `[probable]`) ; rétention mémoire par goroutine dans `acquireFFTState` (`A2-05`) ; deux sémaphores `NumCPU` indépendants → sur-souscription contrôlée mais non bornée globalement (`A2-07`).

### Axe 5 — Structure, tests & CI  *(2 MAJEUR)*

**Verdict :** structure fidèle à la Clean Architecture annoncée (gate `arch_test.go` exécutable), tous les types de tests présents, `t.Parallel()` quasi systématique. Deux problèmes : un test *flaky* sous Windows et l'absence totale de CI.

- **`A5-02` — MAJEUR — Aucune CI distante.** `.github/` inexistant ; le README évoque une CI retirée. `[confirmé]`
  *Impact :* aucun garde-fou automatisé (vet/lint/race/coverage/build) — c'est précisément ce qui aurait capté `A2-01`, `A5-01`, `A4-01`.
  *Recommandation :* GitHub Actions, matrice Go **1.25/1.26**, `go test -race` sur runner Linux (CGO dispo), `golangci-lint`, **gate de couverture 80 %**, `build-all`.
- **`A5-01` — MAJEUR — Test *flaky* sous Windows** : `TestSaveProfile_NeverObservablyTruncated` échoue de façon non déterministe sous charge ; `renameAtomic` borné à 10 tentatives (`calibration/profile.go:175`). `[confirmé]` (reproduit en réexécutions).
- Mineurs : dérives documentaires de version (`A5-03` : README « Go 1.25+ / toolchain 1.26.2 » vs `go.mod 1.26.0` / réel **1.26.3**) ; couverture doc 87,5 % vs **88,6 %** réelle (`A5-04`) ; `make test`/README revendiquent `-race` indisponible sous Windows (`A5-05`) ; incohérence path module `agbru/fibcalc` vs dépôt `agbruneau/FibGo` — **aucun badge CI dans le README** (le claim de l'énoncé est *infirmé*, `A5-06`). Info : `.golangci.yml` en schéma **v1 legacy** (`A5-07`) ; angles morts de couverture e2e/gmp (`A5-08`).

### Axe 1 — Correctness & exactitude algorithmique  *(2 MAJEUR)*

**Verdict :** aucun défaut de logique. Le risque résiduel est de **couverture**, pas de calcul.

- **`A1-01` — MAJEUR — Pas d'oracle de non-régression sur le chemin FFT à l'échelle production (> 500k bits).** `[confirmé]` Le régime FFT réel (N=100M par défaut) n'est jamais recoupé contre un oracle indépendant ; golden et fuzz s'arrêtent bien en deçà.
- **`A1-02` — MAJEUR — Backend GMP non vérifiable ici (CGO désactivé) et couvert seulement jusqu'à F(100), jamais recoupé.** `[à vérifier]`
- Mineurs : `FuzzFastDoublingMod` ne recoupe pas le résultat modulaire contre un oracle (`A1-03`) ; débordement `uint64` silencieux dans `EstimateMemoryUsage` (`A1-04`, garde-fou mémoire) ; débordement `int` *probable* dans le dimensionnement d'arène `AcquireStateForN` (`A1-05`, `[probable]`). Info : asymétrie métriques DTM `FK` vs décision FFT `FK1` (`A1-06`) ; `TestExecuteDoublingStepFFT` n'assertit pas la valeur (`A1-08`).

### Axe 3 — Performance & benchmarks  *(1 MAJEUR)*

**Verdict :** aucune régression (directive > 5 % respectée) ; `sync.Pool`/bump/arena efficaces (chemins de réutilisation à 0 alloc/op) ; complexité confirmée. Les constats sont de nature **doc/structurelle**.

- **`A3-01` — MAJEUR — Le cache de transformées FFT est contourné sur le chemin de production par défaut.** `[confirmé]` (vérifié par l'orchestrateur).
  *Preuve :* l'étape de doublement (`fibonacci/fft.go:128-140`) construit les transformées directement via `TransformWithBump` + `executeFFTTransforms`, **sans passer** par `MulCachedWithBump`/`SqrCachedWithBump` (`bigfft/fft_core.go:65,100`) qui portent le cache. Le speedup « 15-30 % » documenté (CLAUDE.md, PERFORMANCE.md) ne s'applique donc pas à ce chemin.
  *Recommandation :* soit câbler le `TransformCache` dans l'étape de doublement, soit corriger la documentation pour cantonner le gain au chemin `bigfft.Mul/Sqr`. Mesurer avant/après (directive > 5 %).
- Mineurs : coût O(n) du hachage de clé de cache FFT (~5 % CPU) imposé même sans gain (`A3-02`) ; copie redondante `z.Set(result)` dans `FFTOnlyStrategy` (`A3-03`). Info : ordre Matrix/Fast Doubling à 10M contredit la doc *sur cet hôte* (`A3-04`, `[à vérifier]`) ; sous-goroutines FFT retombent sur le pool, bénéfice zéro-alloc non étendu (`A3-05`) ; `math/big.getStack` = 27 % des allocations (interne Karatsuba, `A3-06`).

### Axe 4 — Idiomatique Go & qualité  *(1 MAJEUR)*

**Verdict :** code idiomatique et propre. `go vet` vert, zéro erreur ignorée en production, **seuils de complexité respectés** (`gocognit`/`funlen` = 0 dépassement, `gocyclo` = 1 seul à 17). Le bruit `golangci-lint` (315 lignes) est dominé par des **faux positifs** (86 `gofmt` = CRLF, 50 `misspell` = orthographe britannique volontaire) et des choix documentés.

- **`A4-01` — MAJEUR — `SA6002` ×8 : `sync.Pool.Put` d'une *valeur* slice** (boxing → allocation) dans la couche de pooling (`bigfft/pool.go:148,245,333,421` ; `pool_warming.go:70,79,88,97`). `[confirmé]`
  *Impact :* allocation de boxing à chaque `Put` — contre-productif pour un pool « zéro-alloc » sur le hot path.
  *Recommandation :* stocker/`Put` un **pointeur** vers slice (`*[]T`) plutôt que la slice. Benchmark avant/après.
- Mineurs : code mort `formatAlgoList`/`defaultContext` (`A4-02`) ; `RenderBrailleChart` cyclo 17 > 15 (`A4-03`) ; affectation morte `fermat.go:61` (`A4-04`, fichier sensible) ; nommage de récepteur `fermat` incohérent z/n (`A4-05`) ; `os.Exit` court-circuitant `defer Close` (`A4-06`) ; `revive` *stutter* `format.FormatBytes` & co. (`A4-07`, en partie intentionnel). Info : `misspell`/`gofmt`/`commentedOutCode` majoritairement faux positifs (`A4-09..A4-11`) ; `govet shadow` ×5 dont hot path (`A4-12`).

---

## 4) Plan d'action priorisé (Effort / Impact)

| # | Constat | Sév. | Effort | Impact | Action |
|:--:|---|:--:|:--:|:--:|---|
| 1 | `A2-01` | CRITIQUE | Moyen | **Élevé** | Rendre le contrôle GC *concurrency-safe* (compteur global atomique : 1ᵉʳ entrant désactive / dernier sortant restaure) **ou** remonter le contrôle GC au niveau orchestration. **Confirmer d'abord sous `-race`.** |
| 2 | `A5-02` | MAJEUR | Moyen | **Élevé** | Réintroduire une CI GitHub Actions (matrice Go 1.25/1.26, `-race` Linux, lint, gate couverture 80 %, build-all). Rétablit le garde-fou qui aurait capté #1, #3, #4. |
| 3 | `A5-01` | MAJEUR | Faible | Moyen | Corriger le *flaky* `TestSaveProfile` (rendre `renameAtomic` robuste sous Windows, augmenter/supprimer la borne de 10 tentatives, attente exponentielle). |
| 4 | `A4-01` | MAJEUR | Faible | Moyen | Corriger `SA6002` ×8 : `Put`/`Get` un `*[]T`. Benchmark avant/après (directive > 5 %). |
| 5 | `A2-02`, `A2-03` | MAJEUR | Faible-Moyen | Moyen | Verrouiller `SetCacheLogger` ; propager `context` dans la récursion FFT parallèle (annulation/timeout). Confirmer `A2-02` sous `-race`. |
| 6 | `A3-01` | MAJEUR | Moyen-Élevé | Moyen | Câbler le `TransformCache` sur l'étape de doublement **ou** corriger la doc. Mesurer. |
| 7 | `A1-01`, `A1-02` | MAJEUR | Moyen | Moyen | Étendre l'oracle de non-régression au régime FFT grande échelle (générer via `cmd/generate-golden` sous accord ADR) ; recouper GMP sur runner Linux avec libgmp. |
| 8 | `A1-04`, `A1-05` | MINEUR | Faible | Moyen | Borner/valider les calculs `uint64`/`int` des garde-fous mémoire (débordement silencieux). |
| 9 | Hygiène doc | MINEUR | Faible | Faible | Aligner README/CLAUDE.md (Go 1.26.0, toolchain 1.26.3, couverture 88,6 %, retirer claim `-race` sous Windows, corriger path module/badge). |
| 10 | `A5-07`, `A4-02..A4-08` | MINEUR | Faible | Faible | Migrer `.golangci.yml` vers schéma v2 (`golangci-lint migrate`) ; retirer code mort/affectation morte ; uniformiser récepteurs. |

> **Ordre conseillé :** #1 (sûreté) → #2 (garde-fou systémique) → #3, #4, #5 (corrections rapides à fort ratio) → #6, #7 (perf/couverture, effort moyen) → #8-#10 (hygiène).

---

## 5) Annexe

### Versions & environnement
- **OS** : Windows 11 Pro (26220), `windows/amd64`. **Go** : `go1.26.3`. **Module** : `go 1.26.0`.
- **Commit** : `866b8cdcdde5256bd78db260ed5434e1837d86ec` (2026-05-24), working tree propre.
- **Outils** : staticcheck 2026.1, golangci-lint v1.64.8 (recompilés go1.26.3), gosec dev, govulncheck v1.1.4.
- **Inventaire** : 248 `.go` (135 prod / 113 test), 25 packages, 52 benchmarks, 7 fuzz, 1 property (gopter), golden 80 ko, e2e ×2.

### Écarts notables relevés (réponses aux points « à valider » de l'énoncé)
- `go.mod` = **`go 1.26.0`** (et non 1.25) ; toolchain réelle **1.26.3** (doc dit 1.26.2). `[confirmé]`
- **7 cibles fuzz** (5 `fibonacci` + 2 `bigfft`), pas 5. `[confirmé]`
- **Aucune CI** (`.github/` absent) ; **aucun badge CI dans le README** → le claim « badge pointe vers `agbru/fibcalc` » est *infirmé* (il n'y a pas de badge), mais le **path module** `github.com/agbru/fibcalc` diverge bien du dépôt `agbruneau/FibGo`. `[confirmé]`
- Couverture réelle **88,6 %** (doc 87,5 %). `[confirmé]`

### Limites du sandbox
- **`go test -race` indisponible** (`CGO_ENABLED=0`, ni gcc ni clang). Tout constat de race est **[à vérifier]**.
- **Backend GMP non bâti** (`-tags gmp` requiert CGO + libgmp). `A1-02` reste **[à vérifier]**.
- Benchmarks bornés (`-benchtime=2x/×N`) sur hôte Windows partagé → `ns/op` absolus **[probable]** ; comparaisons relatives et `allocs/op` fiables.
- `staticcheck`/`golangci-lint` préinstallés refusaient go1.26 → recompilés localement (sinon analyse via `go vet` seul).

### Éléments restés [à vérifier] et comment les lever
1. **`A2-01` (race GC)** — rejouer en mode comparaison à N ≥ 1M sous Linux/WSL : `CGO_ENABLED=1 go test -race ./internal/orchestration/... ./internal/fibonacci/...` + un test concurrent qui lance ≥ 2 calculateurs avec `GCMode=auto`.
2. **`A2-02` (logger cache hors verrou)** — `go test -race ./internal/bigfft/` avec accès concurrent `SetCacheLogger` + `logPeriodicStats`.
3. **`A1-02` (GMP)** — runner Linux avec libgmp : `CGO_ENABLED=1 go test -tags gmp ./internal/fibonacci/ -run TestGMP`, recoupé contre Fast Doubling sur grands N.
4. **`A3-04` (ordre des algos)** — rejouer `go test -bench` sur la machine de référence des baselines `docs/audits/` (l'écart constaté est probablement lié à l'hôte).
5. **`govulncheck`** — relancer avec un binaire bâti go1.26 pour une sortie complète (la version utilisée a donné une sortie partielle).
6. **Concern historique bigfft (use-after-free, archivé tag `archive/vague-A-bigfft-concurrency`)** — non confirmé/infirmé statiquement ici (la passe a jugé le cache LRU « correct par construction », `A2-06`, `[à vérifier]`) ; à recouper explicitement sous `-race` lors de la levée du point 1.

---

*Aucun fichier source, `go.mod`, `go.sum` ni `testdata/` n'a été modifié. Seuls les fichiers du dossier `audit/` ont été produits.*
