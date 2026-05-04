# INTERVENTION_PLAN — Finalisation audit 2026-04

**Date** : 2026-05-04
**Branche** : `claude/audit-finalization-2026-05`
**Scope** : findings résiduels de `AUDIT_REPORT.md` après PR #17 (mergée 2026-04-28)
**Mode** : subagent-driven-development (un sous-agent dédié par tâche, double review)

---

## 1. Contexte

L'audit consolidé [`AUDIT_REPORT.md`](AUDIT_REPORT.md) (60 findings) a été
exécuté sous `claude/audit-execution-20260418` (~50 commits, mergés
via PR #17 le 2026-04-28). État résiduel constaté le 2026-05-04 :

| Finding | Vérif code | Statut |
|---|---|---|
| P0-01/P0-09 (FFT pool leak) | `Release()` exposé `fft_poly.go:34,204` + `defer` aux call-sites | ✅ FAIT (`307873f`) |
| P0-02 (Go 1.25 + toolchain) | `go.mod:3` `go 1.25.0`, ligne 5 `toolchain go1.26.2` | ✅ FAIT (`fd13561`) |
| **P0-03 (bubbles v1.0)** | `go.mod:11` toujours `bubbles v0.21.1` | ❌ **À FAIRE** |
| P0-04…07 (liens doc cassés) | `152df86` couvre les 4 | ✅ FAIT |
| P0-08 (`g.Wait()` orchestrator) | `_ = g.Wait()` documenté `orchestrator.go:78` | ✅ FAIT (`65b0aea`) |
| **P1-04 (arena pool)** | Tentative SKIPPED pour race ; design A+B disponible | ❌ **À FAIRE** |
| Tous autres P1/P2/P3 | commits dédiés (`92f0163`, `e360ed0`, …) | ✅ FAIT |

## 2. Tâches restantes

### Tâche A — P0-03 : Migration `bubbles v0.21.1 → v1.0.0`

**Risque** : moyen (rupture API potentielle TUI)
**Surface** : 4 fichiers (`internal/tui/{keymap.go, logs.go, model.go, keymap_test.go}`)
**Sous-packages utilisés** : `key` (3 fichiers), `viewport` (1 fichier)

Symboles sensibles aux ruptures :
- `viewport.Model.Update(tea.Msg) (Model, tea.Cmd)` — vérifier signature retournée
- `viewport.Model{ Width, Height int }` — accès direct champs publics
- `key.NewBinding(key.WithKeys(...), key.WithHelp(...))` — variadic options
- `key.Matches(tea.KeyMsg, key.Binding)` — signature
- `binding.Enabled()`, `binding.Keys()` — méthodes test

**Validation** :
- `go mod tidy && go build ./...`
- `go test -count=1 ./internal/tui/...`
- Smoke test : `go run ./cmd/fibcalc -tui --n=1000` (lancement, navigation, quit)

### Tâche B — P1-04 : Refonte pooling arena (state+arena unifiés)

**Risque** : élevé (refonte cycle de vie pooled, race documentée)
**Approche retenue** : **A + B combinées** (cf. `bench/perf-results/P1-04-SKIPPED.md`)

1. `CalculationState` owne désormais `*memory.CalculationArena` (`arenaSizedFor uint64`).
2. `AcquireState(n uint64)` réutilise/redimensionne l'arena interne.
3. Nouvelle fonction `ReleaseStateWithResult(s, src) *big.Int` :
   - deep-copie `src.Bits()` dans un `*big.Int` neuf hors-arena
   - reset les 5 slots big.Int (`s.FK/FK1/T1/T2/T3 = new(big.Int)`)
   - `s.arena.Reset()` (offset=0)
   - `statePool.Put(s)`
4. `ExecuteDoublingLoop` cesse de "voler" `s.FK` ; le détachement passe par `ReleaseStateWithResult`.

**Surface** : 4 fichiers `internal/fibonacci/{fastdoubling.go, common.go, fft_based.go, doubling_framework.go}` ; `memory/arena.go` inchangé.

**Tests** :
- Existant : `TestConcurrentCalculations` doit passer en `-count=50` (race documentée se reproduit en 5 sous l'ancienne tentative).
- Nouveau : `TestArenaPoolingNoAliasing` — vérifier que `result.Bits()` n'aliase pas l'arena après Release.
- Nouveau : `TestStateReuseAcrossSizes` — Acquire(1000) puis Acquire(100000) redimensionne.
- Bench : `go test -bench=BenchmarkFastDoubling -benchmem ./internal/fibonacci/` doit montrer baisse `B/op` sur N>10000 sans régression `ns/op`.

**Limitation locale** : race detector indisponible sur Windows (CGO requis). Mitigation : `-count=50` + tests dédiés à l'aliasing post-release.

### Tâche C — Validation

- `make test` (suite complète, sans -race local)
- `make lint`
- `go test -bench=. -benchmem -benchtime=1s ./internal/fibonacci/ ./internal/bigfft/ > docs/audits/2026-04/bench/perf-results/post-finalization.txt`
- Comparaison `benchstat` vs baseline `bench/exec-baseline/benchmark.txt`
- Golden tests : `go test -count=1 -run TestGolden ./internal/fibonacci/`

### Tâche D — Documentation

Mises à jour :
- `README.md` : statut courant, versions deps, matrice build, lien INTERVENTION_PLAN.md
- `docs/PERFORMANCE.md` : section arena pooling (P1-04 nouveau pattern)
- `docs/TUI_GUIDE.md` : si UX TUI change avec bubbles v1.0
- `docs/architecture/*` : référencer pooling unifié state+arena
- `CHANGELOG.md` `[Unreleased]` : entrées `P0-03` (deps bubbles v1.0) et `P1-04` (arena pooling refonte)
- `CLAUDE.md` : statistiques post-exécution si évolution notable (LOC/packages/deps)

## 3. Workflow d'exécution

Sequencing strict via `superpowers:subagent-driven-development` :
1. Tâche A (impl) → spec review → code quality review → commit
2. Tâche B (impl) → spec review → code quality review → commit
3. Tâche C (Bash direct, pas de subagent)
4. Tâche D (impl docs) → spec review → commit
5. Final code review (entire branch)
6. Merge `main` ou PR selon `superpowers:finishing-a-development-branch`

## 4. Critères de succès

- [ ] `go.mod` ligne 11 = `bubbles v1.0.0` (ou ≥)
- [ ] `internal/fibonacci/fastdoubling.go` expose `ReleaseStateWithResult`
- [ ] `TestConcurrentCalculations` passe en `-count=50`
- [ ] Bench `BenchmarkFastDoubling` : `B/op` baisse sur N≥10000, `ns/op` non régressif
- [ ] `make test && make lint` verts
- [ ] README + CHANGELOG + docs cohérents avec état courant
- [ ] Branche prête à merger sur `main`

## 5. Hors scope

- Audit additionnel (la base 2026-04 sert de référence)
- Modifications fonctionnelles ou nouveaux algorithmes
- Refactoring non demandé hors P1-04
