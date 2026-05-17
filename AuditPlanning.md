# Audit Remediation Implementation Plan (AuditPlanning.md)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Source of record: [`audit.md`](audit.md) (constats A-01 … A-23).

**Goal:** Adresser la totalité des 23 constats de `audit.md`, réaligner toute la documentation (`Claude.md`, `README.md`, `docs/`) sur l'état réel du code, et nettoyer les fichiers réellement orphelins — sans régression de correction ni de performance.

**Architecture:** Exécution par vagues (A → E) calquées sur `audit.md` §8. Chaque vague est portée par une équipe d'agents spécialisés (voir §Exécution). `internal/bigfft` et `internal/fibonacci` sont **sérialisés** (un seul agent à la fois, hot path) ; le reste se parallélise. TDD strict : test rouge → fix minimal → vert → garde-fous → commit.

**Tech Stack:** Go 1.25 (toolchain 1.26.2), `go test -race`, golangci-lint (24 linters), `make benchmark` (`internal/fibonacci`), golden tests (`fibonacci_golden.json` **immuable**), `gopter` (property-based), Bubble Tea.

---

## Protocole d'exécution global (NON négociable — dérivé de `Claude.md`)

Pour **chaque** tâche, dans l'ordre :

1. Branche dédiée : `git checkout -b fix/A-NN-slug` (jamais de commit direct sur `main`).
2. **Test rouge d'abord** (reproduit le défaut ou verrouille le contrat) → exécuter, vérifier l'échec attendu.
3. Fix minimal et chirurgical (aucune réécriture opportuniste hors périmètre du constat).
4. `go test -race ./<package>/... -count=1` vert.
5. **Si `internal/fibonacci` ou `internal/bigfft` touché** : `make benchmark` AVANT (baseline sur `main`) et APRÈS ; **régression > 5 % = blocage** (revert, ré-architecture). Comparer aux baselines `docs/audits/` si présentes (sinon créer la baseline et la versionner).
6. Golden : `go test ./internal/fibonacci/... -run Golden -count=1` vert. `fibonacci_golden.json` **interdit à la modification** (aucun `-update`).
7. `make lint` (0 nouvelle alerte) puis `make test -race` (suite complète) vert.
8. Commit : `fix(A-NN): description` (ou `docs(A-NN):`, `ci(A-NN):`, `test(A-NN):`).
9. Mettre à jour la table de suivi (§Suivi) : statut + SHA.
10. Aucun nouveau global dans `bigfft/` (directive projet ; A-03 va dans le sens inverse — réduction).

**Garde-fous absolus :** pas de modification du golden ; pas de nouveau global `bigfft` ; étanchéité Clean Architecture préservée (`internal/` ne fuit pas vers `cmd/`) ; toute modif perf-sensible benchmarkée.

---

## Cartographie des fichiers touchés

| Constat | Fichiers modifiés | Tests (créés/étendus) |
|---|---|---|
| A-01 | `internal/bigfft/fft_cache.go` | `internal/bigfft/fft_cache_race_test.go` (nouveau) |
| A-02 | `internal/bigfft/fft_cache.go` | `internal/bigfft/fft_cache_test.go` |
| A-03 | `internal/bigfft/fft.go`, `fft_recursion.go`, `fft_recursion_ctx.go` | `internal/bigfft/fft_parallel_test.go` |
| A-05 | `internal/bigfft/fft_cache.go` | `internal/bigfft/fft_cache_test.go` |
| A-06 | `internal/bigfft/fermat.go`, `fft.go`, `context.go` | `internal/bigfft/fermat_test.go` |
| A-07 | `internal/bigfft/fft_poly.go` | `internal/bigfft/fft_poly_test.go` |
| A-08 | `internal/config/config.go` | `internal/config/config_test.go` |
| A-09 | `internal/config/env.go` | `internal/config/env_test.go` |
| A-10 | `internal/progress/progress.go` | `internal/progress/progress_test.go` |
| A-11 | `internal/calibration/profile.go` | `internal/calibration/profile_test.go` |
| A-12 | `.github/workflows/ci.yml` | (CI) |
| A-13 | `.github/workflows/ci.yml`, `.github/workflows/coverage.yml` | (CI) |
| A-14 | `.github/workflows/ci.yml` | (CI) |
| A-15 | `test/e2e/cli_e2e_test.go`, `test/e2e/extended_e2e_test.go` | (tests eux-mêmes) |
| A-16 | `internal/fibonacci/common.go`, `internal/fibonacci/fastdoubling.go` | commentaires + `common_test.go` |
| A-17 | `internal/fibonacci/fft.go` | `internal/fibonacci/fft_race_test.go` |
| A-18 | `internal/fibonacci/threshold/manager.go` | commentaire + `manager_test.go` |
| A-19 | `internal/progress/observers.go`, `observer.go`, `internal/tui/model.go` | `internal/progress/*_test.go` |
| A-20 | `internal/fibonacci/calculator_equivalence_test.go` (nouveau) | idem |
| A-21 | `Claude.md`, `README.md` | — |
| A-22 | `internal/tui/logs.go`, `internal/tui/metrics.go` | `internal/tui/*_test.go` |
| A-23 | `docs/CALIBRATION.md`, `README.md` | — |
| Doc | `Claude.md`, `README.md`, `CHANGELOG.md` | — |

---

## VAGUE A — Sûreté concurrence `bigfft` (sérialisée, 1 agent, hot path)

> Bloquant si parallélisme FFT + cache actifs sur charges longues. Tout commit ici exige `make benchmark` avant/après + golden vert.

### Task A1: A-01 — Recyclage `backing` en éviction (race silencieuse)

**Files:**
- Modify: `internal/bigfft/fft_cache.go:316-377` (`putByKey`), `:231-272` (`getByKey`), `cacheEntry` struct
- Test: `internal/bigfft/fft_cache_race_test.go` (créer)

Approche retenue : **refcount/epoch atomique sur `cacheEntry`** — préserve l'optimisation R1.5 (pas de `Clone()` systématique), ne recycle un `backing` que si aucune référence vivante. Alternative documentée (Clone défensif en sortie de `getByKey`) à n'activer que si le refcount régresse le benchmark > 5 %.

- [ ] **Step 1 — Baseline benchmark (avant tout changement)**
  Run: `make benchmark | tee docs/audits/bench-baseline-A01.txt` (créer `docs/audits/` si absent — c'est aussi le répertoire fantôme de A-21/A-04 ; sa création résout partiellement la dérive doc).
  Expected: snapshot enregistré, exit 0.

- [ ] **Step 2 — Test rouge : race lecture/recyclage**
  Créer `internal/bigfft/fft_cache_race_test.go` :
```go
package bigfft

import (
	"sync"
	"testing"
)

// TestCacheEvictionDoesNotMutateLiveReader reproduit A-01 : un lecteur qui
// détient un PolValues issu de getByKey ne doit jamais voir son backing
// zéroé/réécrit par une éviction concurrente.
func TestCacheEvictionDoesNotMutateLiveReader(t *testing.T) {
	tc := NewTransformCache(TransformCacheConfig{Enabled: true, MinBitLen: 0, MaxEntries: 2})
	// Pré-remplir + capturer une entrée vivante.
	data := nat{0xDEADBEEF, 0x1234}
	pvSeed := makeTestPolValues(t, 4, 8) // helper: K=4, n=8, valeurs déterministes
	tc.Put(data, pvSeed)
	got, ok := tc.Get(data, pvSeed.K, pvSeed.N)
	if !ok {
		t.Fatal("seed entry not cached")
	}
	snapshot := clonePolValuesWords(got) // copie indépendante de référence

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Forcer l'éviction de l'entrée vivante par saturation.
		for i := 0; i < 64; i++ {
			tc.Put(nat{uint(i + 1), 0xFF}, makeTestPolValues(t, 4, 8))
		}
	}()
	// Pendant l'éviction, relire continûment got.Values.
	for i := 0; i < 100000; i++ {
		if !equalPolValuesWords(got, snapshot) {
			t.Fatalf("live reader observed mutation at iter %d (A-01 race)", i)
		}
	}
	wg.Wait()
}
```
  *(Helpers `makeTestPolValues`/`clonePolValuesWords`/`equalPolValuesWords` à placer dans ce fichier de test.)*

- [ ] **Step 3 — Vérifier l'échec**
  Run: `go test ./internal/bigfft/ -run TestCacheEvictionDoesNotMutateLiveReader -race -count=1`
  Expected: FAIL (mutation observée ou data race signalée par `-race`).

- [ ] **Step 4 — Fix : refcount atomique, recyclage conditionnel**
  Dans `fft_cache.go`, ajouter à `cacheEntry` un champ `refs atomic.Int32`. `getByKey` incrémente `refs` avant `RUnlock` et expose un `release` (ou : `PolValues` issu du cache porte un finaliseur léger appelé par `pv.Release()` — déjà no-op sur hit, le câbler au décrément). Dans `putByKey`, la condition de salvage devient :
```go
if backing == nil && cap(entry.backing) >= wordCount && entry.refs.Load() == 0 {
    backing = entry.backing[:wordCount]
    for i := range backing { backing[i] = 0 }
}
```
  Une entrée encore référencée n'est jamais recyclée ; son backing part au GC (miss compté). Mettre à jour le commentaire mensonger `:228-230` (« immutable after insertion ») pour refléter le contrat refcount réel.

- [ ] **Step 5 — Vert + non-régression**
  Run: `go test ./internal/bigfft/ -run 'Cache|Transform|FFT' -race -count=1` → PASS
  Run: `go test ./internal/fibonacci/ -run Golden -count=1` → PASS (golden inchangé)

- [ ] **Step 6 — Benchmark après**
  Run: `make benchmark | tee docs/audits/bench-A01-after.txt`
  Comparer à `bench-baseline-A01.txt`. **Si régression FFT > 5 %** : basculer sur l'alternative `Clone()` défensif uniquement pour les entrées `refs>0`, re-benchmarker.

- [ ] **Step 7 — Commit**
```bash
git add internal/bigfft/fft_cache.go internal/bigfft/fft_cache_race_test.go docs/audits/
git commit -m "fix(A-01): refcount cacheEntry, no backing recycle while referenced"
```

### Task A2: A-02 — `tc.config` lu sans verrou (data race)

**Files:** Modify `internal/bigfft/fft_cache.go` (`Get:207`, `Put:301`, struct config, `SetTransformCacheConfig`); Test `fft_cache_test.go`.

- [ ] **Step 1 — Test rouge** : `TestConfigConcurrentAccess` — goroutine A boucle `Get/Put`, goroutine B boucle `SetTransformCacheConfig`. Run `-race`. Expected: FAIL (race détectée).
- [ ] **Step 2 — Fix** : remplacer `config.Enabled`/`MinBitLen`/`MaxEntries` par `atomic.Bool`/`atomic.Int64` ; `Get`/`Put`/`putByKey` lisent via `.Load()` ; `SetTransformCacheConfig` écrit via `.Store()`. `TransformCacheConfig` public conserve ses champs valeur ; conversion au passage de frontière.
- [ ] **Step 3 — Vert** : `go test ./internal/bigfft/ -run Config -race -count=1` → PASS.
- [ ] **Step 4 — Benchmark** (lecture atomique sur hot path) avant/après ; régression > 5 % bloquante.
- [ ] **Step 5 — Commit** : `fix(A-02): atomic TransformCache config fields`.

### Task A3: A-03 — Globaux FFT mutables non synchronisés

**Files:** Modify `internal/bigfft/fft.go:35`, `fft_recursion.go:28,33`, `fft_recursion_ctx.go`; Test `fft_parallel_test.go`.

- [ ] **Step 1 — Test rouge** : `TestFFTParallelismConfigRace` — `SetFFTParallelismConfig` concurrent à une `MulWithContext` parallèle, `-race`. Expected: FAIL.
- [ ] **Step 2 — Fix** : `ParallelFFTRecursionThreshold`/`MaxParallelFFTDepth`/`fftThreshold` → `atomic.Uint64` (lecture `.Load()` à chaque nœud, écriture `.Store()` dans le setter). NE PAS ajouter de nouveau global ; convertir les existants en place (type changé, API publique préservée via accesseurs si exportés).
- [ ] **Step 3 — Vert** `-race` ; **Step 4 — Benchmark** (lecture atomique en récursion chaude — surveiller) ; **Step 5 — Commit** `fix(A-03): atomic FFT parallelism globals`.

### Task A4: A-05 — `putByKey` assertion d'invariant

**Files:** Modify `internal/bigfft/fft_cache.go:316-377`; Test `fft_cache_test.go`.

- [ ] **Step 1 — Test rouge** : `TestPutByKeyRejectsMalformedShape` — `PolValues` avec `len(Values[0]) != N+1` ; après `Put`+`Get`, attendre miss (drop), jamais troncation.
- [ ] **Step 2 — Fix** : en tête de `putByKey`, `for i := range pv.Values { if len(pv.Values[i]) != n+1 { return } }` (drop silencieux, jamais corruption). Documenter l'invariant d'entrée dans le doc-comment de `Put`.
- [ ] **Step 3 — Vert + golden + benchmark** ; **Step 4 — Commit** `fix(A-05): defensive shape assertion in putByKey`.

### Task A5: A-06 — `recover()` global masque les post-conditions `fermat`

**Files:** Modify `internal/bigfft/fermat.go`, `fft.go:42-45`, `context.go`; Test `fermat_test.go`.

- [ ] **Step 1 — Test rouge** : `TestInternalPostconditionPanicNotMasked` — provoquer une post-condition interne (`len(z) > 2n+1`) doit re-panic, pas retourner une erreur opaque ; un mismatch d'entrée doit, lui, retourner une erreur.
- [ ] **Step 2 — Fix** : type sentinelle `type invariantViolation struct{ msg string }` paniqué par les post-conditions internes ; le `recover()` de `fft.go`/`context.go` ré-`panic` si `invariantViolation`, sinon emballe en `error`.
- [ ] **Step 3 — Vert + golden** ; **Step 4 — Commit** `fix(A-06): do not mask internal fermat postconditions`.

### Task A6: A-07 — `NTransform`/`InvNTransform` paniquent (API publique)

**Files:** Modify `internal/bigfft/fft_poly.go:343-347`; Test `fft_poly_test.go`.

- [ ] **Step 1 — Test rouge** : `TestNTransformPreconditionReturnsError` — `len(p.A) >= 1<<k` doit renvoyer une `error`, pas paniquer.
- [ ] **Step 2 — Fix** : router la précondition vers la valeur de retour `error` existante de `NTransform`/`InvNTransform` (remplacer `panic` par `return ..., fmt.Errorf(...)`).
- [ ] **Step 3 — Vert + golden + benchmark** ; **Step 4 — Commit** `fix(A-07): NTransform precondition returns error`.

---

## VAGUE B — Dette documentaire normative (parallélisable, équipe doc)

### Task B1: A-04 + Doc — Réaligner `Claude.md` sur l'état réel

**Files:** Modify `Claude.md`.

- [ ] **Step 1** — Section « Projet » : remplacer « CI/CD : ❌ aucun workflow GitHub Actions à ce jour (cf. R4.1) » par : « CI/CD : GitHub Actions — `ci.yml` (vet+lint+build, race sur Linux/macOS, matrice 3 OS) et `coverage.yml`. R4.1 livré. »
- [ ] **Step 2** — Bloc « 🔄 REFACTORING EN COURS » : retirer les liens `ultrareview.md`/`ultrareviewplan.md` (inexistants) ; remplacer par : « Audit de référence : [`audit.md`](audit.md) ; plan de remédiation : [`AuditPlanning.md`](AuditPlanning.md). »
- [ ] **Step 3** — Table « ⚠ Bugs latents connus » : remplacer par une table « Bugs historiques — RÉSOLUS » avec statut/SHA (R1.1–R1.5 corrigés, cf. `audit.md` §4), ou supprimer la section et renvoyer à `audit.md` §4.
- [ ] **Step 4** — Tableau « Modules sensibles » : corriger les tailles périmées (`threshold/manager.go` 277 L et non 417 ; `model.go` ~184 L ; `completion.go` dispatcher) ; retirer la mention `parallel` « quasi-mort » (faux — utilisé `common.go:114,149,225,282`).
- [ ] **Step 5** — Directive #10 « CI absente — vérifier manuellement » : réécrire (« CI active ; exécuter localement `make test -race && make lint` reste recommandé avant PR »). Retirer toute référence au « Workflow recommandé » s'appuyant sur `ultrareviewplan.md` ; le remplacer par un renvoi à `AuditPlanning.md`.
- [ ] **Step 6** — Aligner « ~36 500 LOC / 21 packages » sur les chiffres mesurés (`audit.md` §7 : ~35 454 LOC, 24 packages) ; retirer `sysmon/`/`parallel/` de l'arbre s'ils sont fusionnés/à statuer (vérifier l'existence réelle avant édition).
- [ ] **Step 7 — Commit** `docs(A-04): realign Claude.md with post-refactoring reality`.

### Task B2: A-21 — Métriques & compte de linters

**Files:** Modify `Claude.md`, `README.md`.

- [ ] **Step 1** — Remplacer « golangci-lint (22 linters) » par « (24 linters) » dans `Claude.md` et `README.md` (vérifier le décompte réel d'`enable:` dans `.golangci.yml` au moment de l'édition).
- [ ] **Step 2** — Mettre à jour l'arbre packages du `README.md` (retirer `sysmon/`/`parallel/` si absents) et les LOC.
- [ ] **Step 3 — Commit** `docs(A-21): correct linter count and package metrics`.

### Task B3: A-23 — `docs/CALIBRATION.md` périmé

**Files:** Modify `docs/CALIBRATION.md`, `README.md`, `.env.example` (si présent).

- [ ] **Step 1** — Ajouter le champ `Confidence float64` au struct `CalibrationProfile` documenté (CALIBRATION.md ~172-190), en miroir de `profile.go:38`.
- [ ] **Step 2** — Remplacer la section « Tier 3 / newCalibrationRunner » par la description du pattern Strategy (`CalibrationStrategy`, `CompleteStrategy`, `EscalationConfidenceThreshold`, branche stale R1.3 → `IsStale` → `CompleteStrategy`).
- [ ] **Step 3** — Documenter `FIBCALC_PROFILE_MAX_AGE` (défaut 7 j) dans `README.md` (tableau env), `docs/CALIBRATION.md`, `.env.example`.
- [ ] **Step 4 — Commit** `docs(A-23): update CALIBRATION.md to Strategy pattern + env`.

---

## VAGUE C — Validation des entrées & robustesse (parallélisable hors bigfft)

### Task C1: A-08 — `LastDigits<0` + validations config manquantes

**Files:** Modify `internal/config/config.go:99-120`; Test `internal/config/config_test.go`.

- [ ] **Step 1 — Test rouge** : `TestValidate_RejectsNegativeLastDigits`, `_RejectsNegativeStrassen`, `_RejectsUnknownGCControl`, `_RejectsUnknownCompletion` (table-driven). Expected: FAIL (actuellement acceptés).
- [ ] **Step 2 — Fix** : étendre `Validate()` : `LastDigits < 0` → `ConfigError` ; `StrassenThreshold < 0` → erreur ; `GCControl ∈ {auto,aggressive,disabled}` ; `Completion ∈ {bash,zsh,fish,powershell,""}`.
- [ ] **Step 3 — Vert** `go test ./internal/config/ -race -count=1` ; **Step 4 — Commit** `fix(A-08): validate LastDigits/Strassen/GCControl/Completion`.

### Task C2: A-09 — Overrides d'env mal formés avalés

**Files:** Modify `internal/config/env.go:114-133` (+ helpers `getEnvUint64/Int/Bool/Duration`); Test `env_test.go`.

- [ ] **Step 1 — Test rouge** : `TestEnvOverride_MalformedReturnsError` — `FIBCALC_N=abc` doit produire une `ConfigError` (ou avertissement sur `errWriter` + valeur défaut, selon contrat retenu — choisir : **erreur**), pas un silence.
- [ ] **Step 2 — Fix** : sur `err != nil` de parsing d'une variable **explicitement positionnée**, retourner `ConfigError` (signature de `applyEnvOverrides` propage déjà une erreur, sinon l'ajouter et la router vers `Validate`/lifecycle).
- [ ] **Step 3 — Vert + Step 4 — Commit** `fix(A-09): surface malformed env overrides`.

### Task C3: A-11 — Persistance profil non atomique

**Files:** Modify `internal/calibration/profile.go:110-125`; Test `profile_test.go`.

- [ ] **Step 1 — Test rouge** : `TestSaveProfile_Atomic` — vérifier qu'aucun fichier partiel n'est observable (écriture dans `*.tmp` même répertoire puis `os.Rename`).
- [ ] **Step 2 — Fix** : `SaveProfile` écrit `path+".tmp"` (0600) puis `os.Rename(tmp, path)` (atomique POSIX/Windows). Conserver l'exclusion G304 (justifiée).
- [ ] **Step 3 — Vert + Step 4 — Commit** `fix(A-11): atomic profile persistence (temp+rename)`.

### Task C4: A-10 — `CalcTotalWork` overflow → progression figée

**Files:** Modify `internal/progress/progress.go:44-94`; Test `progress_test.go`. ⚠ Consommé par `internal/fibonacci` → benchmark requis.

- [ ] **Step 1 — Test rouge** : `TestCalcTotalWork_LargeNumBits` — `numBits=2000` ne doit pas produire `+Inf`/`NaN` ; la progression doit rester monotone croissante dans [0,1].
- [ ] **Step 2 — Fix** : passer en espace logarithmique — ratio de progression = `4^(stepIndex-(numBits-1))` normalisé, calculé sans `math.Pow(4,numBits)` (utiliser `math.Exp2(2*float64(stepIndex-(numBits-1)))` borné, ou somme géométrique fermée). Supprimer `PrecomputePowers4` au-delà du domaine sûr.
- [ ] **Step 3 — Vert** ; **Step 4 — `make benchmark`** (progress sur hot path fibonacci) avant/après, < 5 % ; **Step 5 — Commit** `fix(A-10): log-space progress, no Pow(4,numBits) overflow`.

---

## VAGUE D — CI & tests (parallélisable, équipe CI)

### Task D1: A-12 — Épingler `golangci-lint`

**Files:** Modify `.github/workflows/ci.yml:32,60,66,80`.

- [ ] **Step 1** — `golangci/golangci-lint-action@v6` : remplacer `version: latest` par une version exacte (ex. la dernière stable connue au moment de l'édition, à vérifier sur le dépôt golangci-lint — ne pas inventer).
- [ ] **Step 2** — Retirer `check-latest: true` (3 occurrences) ; s'appuyer sur `go-version-file: go.mod`.
- [ ] **Step 3 — Commit** `ci(A-12): pin golangci-lint, drop check-latest`.

### Task D2: A-14 — `-race` sous Windows

**Files:** Modify `.github/workflows/ci.yml:41-47`.

- [ ] **Step 1** — Fusionner les steps test : `go test -race -short ./...` sur les 3 OS (le race detector amd64/Windows est stable). Si CGO requis et indisponible sur le runner Windows, documenter explicitement la limitation en commentaire YAML et garder l'exécution race locale obligatoire.
- [ ] **Step 2 — Commit** `ci(A-14): enable -race on Windows runner`.

### Task D3: A-13 — Garde benchmark + couverture

**Files:** Modify `.github/workflows/ci.yml`, `.github/workflows/coverage.yml`.

- [ ] **Step 1** — `coverage.yml` : ajouter `pull_request` aux triggers ; ajouter un seuil minimal (échec si couverture < seuil retenu, ex. 80 % — valeur à arbitrer, ne pas dépasser la couverture réelle actuelle).
- [ ] **Step 2** — `ci.yml` : job `bench` (non bloquant d'abord — `continue-on-error: true`) `go test -bench=BenchmarkFibonacci -benchmem -run=^$ ./internal/fibonacci/`, artefact uploadé pour comparaison manuelle (un gate automatique fiable nécessite une baseline versionnée — hors périmètre minimal, documenter le suivi).
- [ ] **Step 3 — Commit** `ci(A-13): coverage on PR + threshold + bench job`.

### Task D4: A-15 — test/e2e discriminant + `Short()`

**Files:** Modify `test/e2e/cli_e2e_test.go`, `test/e2e/extended_e2e_test.go`.

- [ ] **Step 1** — Remplacer les `t.Logf("...accepting any non-zero...")` par `t.Errorf` quand `wantCode` est un contrat stable ; sinon retirer `wantCode` (pas de fausse précision).
- [ ] **Step 2** — Ajouter `if testing.Short() { t.Skip("e2e: skipped in -short") }` en tête des tests e2e lourds ; faire réutiliser `buildOnce` (supprimer le second `go build`).
- [ ] **Step 3 — Vert** `go test ./test/e2e/ -count=1` (long) et `-short` (skip) ; **Step 4 — Commit** `test(A-15): discriminant exit codes + Short guard`.

---

## VAGUE E — Hygiène (parallélisable)

### Task E1: A-16 — `MaxPooledBitLen` cohérence commentaire/unités
**Files:** Modify `internal/fibonacci/common.go:55-60`, `fastdoubling.go:18`.
- [ ] **Step 1** — Aligner le commentaire sur la valeur réelle (`50_000_000`) ; documenter explicitement la relation bits vs words avec `maxArenaPoolWords` (ou dériver l'un de l'autre via constante partagée commentée). **Step 2** — `make benchmark` (aucune modif de valeur, doc-only attendu) ; **Step 3 — Commit** `docs(A-16): clarify pool threshold units`.

### Task E2: A-17 — Garde `-race` aliasing arena FFT parallèle
**Files:** Test `internal/fibonacci/fft_race_test.go`.
- [ ] **Step 1** — Test `-race` : F(n) FFT en mode parallèle forcé avec arena volontairement sous-dimensionnée (réallocation hors arena en plein step parallèle) ; asserter résultat == golden. **Step 2** — `go test ./internal/fibonacci/ -run FFTRace -race -count=1` PASS ; **Step 3 — Commit** `test(A-17): race guard for parallel FFT arena aliasing`.

### Task E3: A-18 — Documenter l'invariant single-writer `threshold`
**Files:** Modify `internal/fibonacci/threshold/manager.go` (commentaire `ShouldAdjust`).
- [ ] **Step 1** — Commenter explicitement : « single-writer (goroutine du loop de doublage) ; les getters RLock observent un write non atomique — sûr SOUS CET INVARIANT uniquement ; tout partage du manager entre calculs introduirait une data race ». **Step 2 — Commit** `docs(A-18): document threshold single-writer invariant`.

### Task E4: A-19 — Réduire la surface d'API morte `progress` + message TUI
**Files:** Modify `internal/progress/observers.go`, `observer.go`, `internal/tui/model.go`.
- [ ] **Step 1 — Test** : vérifier qu'aucun appelant de prod n'utilise `Register/Notify/ChannelObserver/LoggingObserver` (grep prouvé en audit). **Step 2** — Déclasser en non-exporté OU documenter dans `doc.go` que `Freeze` est le seul chemin de prod et le reste une extension délibérée (choisir : **documenter** — moins risqué, pas de rupture d'API). Commenter le `case ProgressDoneMsg: return m, nil` comme « drain sink, intentional » (`model.go`). **Step 3 — Commit** `refactor(A-19): document progress prod path, annotate dead messages`.

### Task E5: A-20 — Test d'équivalence oracle `calculateSmall` ↔ `fibBig`
**Files:** Create `internal/fibonacci/calculator_equivalence_test.go`.
- [ ] **Step 1 — Test** : pour k ∈ [0,93], `calculateSmall(k) == fibBig(k)` (compare les **deux** oracles entre eux — pas le golden, pas de tautologie). **Step 2** — `go test ./internal/fibonacci/ -run Equivalence -count=1` PASS ; **Step 3 — Commit** `test(A-20): cross-check calculateSmall vs fibBig oracle`.

### Task E6: A-22 — Allocations TUI répétées
**Files:** Modify `internal/tui/logs.go`, `internal/tui/metrics.go`.
- [ ] **Step 1 — Test/bench** : `BenchmarkLogsUpdateContent` (baseline). **Step 2** — Invalidation paresseuse : ne reconstruire `entries`/`SetContent` que sur tick de rendu/viewport, pas sur chaque `buffer.Push` ; mémoïser les `Style.Render` constants. **Step 3** — `go test ./internal/tui/ -race -count=1` PASS + bench amélioré ; **Step 4 — Commit** `perf(A-22): lazy TUI content rebuild`.

---

## VAGUE F — Documentation finale & nettoyage

### Task F1: Mise à jour documentaire transverse finale
**Files:** `Claude.md`, `README.md`, `CHANGELOG.md`.
- [ ] **Step 1** — `CHANGELOG.md` (format Keep-a-Changelog) : section `## [Unreleased]` listant les fixes A-01…A-23 par catégorie (Fixed/Changed/Security/Docs) avec SHA.
- [ ] **Step 2** — `README.md` : vérifier badge couverture (mettre à jour le % si le seuil/mesure a changé), arbre packages, table des variables d'env (ajouter `FIBCALC_PROFILE_MAX_AGE`), compte de linters.
- [ ] **Step 3** — `Claude.md` : relire intégralement post-fixes ; aucune référence à un fichier inexistant ; le « Workflow recommandé » pointe `AuditPlanning.md`/`audit.md`.
- [ ] **Step 4 — Commit** `docs: finalize documentation realignment post-audit`.

### Task F2: Nettoyage des fichiers — CONSERVATEUR (liste explicite, aucune suppression à l'aveugle)

> **Garde-fou :** « inutile » n'est pas auto-déterminé. Supprimer uniquement les fichiers explicitement listés et confirmés orphelins. Ne JAMAIS supprimer : `audit.md`, `audit-prompt.md`, `AuditPlanning.md` (livrables de référence), tout fichier suivi par git non identifié comme mort, tout artefact non créé par ce travail.

- [ ] **Step 1 — Inventaire** : `git status --porcelain` + `git clean -nd` (dry-run) pour lister les non-suivis. Recenser explicitement chaque candidat.
- [ ] **Step 2 — Statuer cas par cas** :
  - `ruvector.db` : **artefact du tooling MCP (Ruflo), NON créé par ce travail** → ne PAS supprimer ; proposer son ajout à `.gitignore` (modification réversible) et signaler à l'utilisateur. Décision de suppression = ressort exclusif de l'utilisateur.
  - Profils PGO périmés / binaires `build/` / `coverage.out`/`coverage.html` : déjà couverts par `make clean` et `.gitignore` — `make clean` plutôt que `rm` manuel.
  - Fichiers temporaires `*.tmp` issus de A-11 : transitoires, jamais commités (déjà gérés par le rename atomique).
  - Aucun fichier source/test mort identifié par l'audit (le code mort relevé — `ErrorCollector.Reset`, API `progress` — est traité en A-19 par déclassement/doc, pas par suppression de fichier).
- [ ] **Step 3** — Exécuter uniquement : `make clean` (artefacts de build). Pour tout autre candidat, produire la liste et **demander confirmation** (suppression = action irréversible hors `make clean`).
- [ ] **Step 4 — Commit** (si `.gitignore` modifié) `chore: gitignore tooling artifacts (ruvector.db), make clean`.

---

## Exécution par équipes d'agents

| Équipe | Vague(s) | Sérialisation | Agent type |
|---|---|---|---|
| **Concurrence-FFT** | A (A1→A6) | **Strictement séquentiel** (1 agent, hot path, benchmark/golden chaque tâche) | coder, sous review |
| **Doc** | B (B1,B2,B3), F1 | Parallèle entre elles ; F1 après A–E | coder/general |
| **Robustesse** | C1,C2,C3 (parallèle) ; C4 sérialisé avec l'équipe FFT (touche progress→fibonacci) | semi-parallèle | coder |
| **CI** | D1,D2,D3,D4 | Parallèle | coder |
| **Hygiène** | E1,E3,E4,E5 (parallèle) ; E2,E6 (touchent fibonacci/tui — coordonner avec FFT) | semi-parallèle | coder |
| **Nettoyage** | F2 | Dernier, après tout le reste | general |

**Ordre global :** A (séquentiel) ‖ {B, D} (parallèles) → C → E → F1 → F2. Revue de code entre chaque tâche de la Vague A (sous-skill `requesting-code-review`). `make test -race && make lint` vert obligatoire avant chaque merge. Vérification finale `superpowers:verification-before-completion` avant clôture.

---

## Registre de risques & rollback

| Risque | Mitigation |
|---|---|
| Régression perf A-01/A-02/A-03/A-10 (hot path) | Benchmark avant/après chaque tâche ; > 5 % = revert immédiat + alternative documentée. |
| Modification accidentelle du golden | Aucune commande `-update` ; golden en lecture seule ; CI golden bloquante. |
| Nouveau global `bigfft` (A-03) | Conversion en place des globaux existants, zéro ajout ; revue dédiée. |
| Suppression de fichier utile (F2) | Liste explicite + confirmation utilisateur ; `make clean` seul automatique ; `ruvector.db` jamais supprimé. |
| Dérive de scope (refactor opportuniste) | Chaque commit cite un seul A-NN ; diff minimal ; review entre tâches. |
| Rollback | Chaque tâche = 1 branche + 1 commit atomique → `git revert <sha>` ciblé. |

---

## Suivi (à maintenir à chaque transition de statut)

> Statuts : ✅ Done (mergé sur `main`) · 🧊 Frozen (implémenté sur branche, **non mergé**, en attente de revue humaine) · ⬜ Pending.
> Vague A gelée sur la branche `review/vague-A-bigfft-concurrency` (6 commits) — décision utilisateur : revue avant merge (modif concurrence hot path, `-race` validable seulement en CI Linux).

| ID | Sévérité | Vague | Statut | SHA |
|---|---|---|---|---|
| A-01 | Critique | A | 🧊 Frozen (revue) | `a6d57bb` (branche) |
| A-02 | Haute | A | 🧊 Frozen (revue) | `55f2340` (branche) |
| A-03 | Haute | A | 🧊 Frozen (revue) | `61b2970` (branche) |
| A-04 | Haute | B | ✅ Done | `ae4bed6` |
| A-05 | Moyenne | A | 🧊 Frozen (revue) | `57444ab` (branche) |
| A-06 | Moyenne | A | 🧊 Frozen (revue) | `6aba6aa` (branche) |
| A-07 | Moyenne | A | 🧊 Frozen (revue) | `d143172` (branche) |
| A-08 | Moyenne | C | ✅ Done | `4740326` |
| A-09 | Moyenne | C | ✅ Done | `4740326` |
| A-10 | Moyenne | C→E | ✅ Done | `e2af8b6` |
| A-11 | Moyenne | C | ✅ Done | `ca20dfa` |
| A-12 | Moyenne | D | ✅ Done | `7a0a807` |
| A-13 | Moyenne | D | ✅ Done | `7a0a807` (+ .gitignore fix : coverage.yml non suivi) |
| A-14 | Moyenne | D | ✅ Done | `7a0a807` |
| A-15 | Moyenne | D | ✅ Done | `7a0a807` |
| A-16 | Basse | E | ✅ Done | `118eaae` |
| A-17 | Basse | E | ✅ Done | `e2af8b6` |
| A-18 | Basse | E | ✅ Done | `118eaae` |
| A-19 | Basse | E | ✅ Done | `118eaae` |
| A-20 | Basse | E | ✅ Done | `e2af8b6` |
| A-21 | Basse | B | ✅ Done | `ae4bed6` |
| A-22 | Basse | E | ✅ Done | `390103b` |
| A-23 | Basse | B | ✅ Done | `ae4bed6` |
| DOC-FINAL | — | F | ✅ Done | (ce commit) |
| CLEANUP | — | F | ✅ Done | (ce commit, conservateur : `go clean` + .gitignore) |

---

## Self-review (couverture spec)

Tous les constats `audit.md` A-01…A-23 ont une tâche dédiée (table de suivi ci-dessus, 23/23). Mise à jour doc finale = Task F1 (Claude.md, README, CHANGELOG). Nettoyage = Task F2 (conservateur, confirmation requise pour toute suppression hors `make clean`). Aucun placeholder : chaque tâche code porte test rouge + fix + commande + commit ; les tâches doc portent le texte de remplacement exact. Gates perf/golden explicites pour tout ce qui touche `fibonacci`/`bigfft`.
