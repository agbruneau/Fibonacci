# ADR-0001: Sort de `DynamicThresholdManager` vs `internal/calibration/`

- **Status**: Accepted (decision: KEEP, with reduced maintenance cost via atomic conversion).
- **Date**: 2026-05-21 (mesures complétées)
- **Context source**: hardening sprint mai 2026 (commits `c0cc530` → `3d8b977`).

## Context

Le projet maintient **deux** mécanismes adaptatifs pour les seuils FFT/parallèle :

1. `internal/calibration/` (~1 686 LOC) produit `OptimalFFTThreshold`/`OptimalParallelThreshold` *au démarrage* via micro-benchmarks (`internal/calibration/profile.go:30-32`).
2. `internal/fibonacci/threshold/manager.go` (~283 L) ajuste ces seuils *en cours de calcul* via une fenêtre glissante de 20 métriques avec hystérésis.

L'audit a relevé que cette redondance n'est appuyée par aucun benchmark publié dans `docs/audits/`, et que l'invariant single-writer du DTM (A-18, lignes 167-174) est fragile sans `atomic`.

## Decision

Deux étapes :

1. **Court terme** ✅ : convertir les trois champs mutés hors verrou (`currentFFTThreshold`, `currentParallelThreshold`, `lastAdjustment`) en `atomic.Int64`/`atomic.Pointer[time.Time]`. L'invariant A-18 devient inutile. Réalisé dans le commit `c0cc530`.
2. **Moyen terme** ✅ : `docs/audits/bench-dtm-{on,off}.txt` produits via `BenchmarkFibonacciDTM` (`internal/fibonacci/dtm_bench_test.go`).

### Résultats benchmark (single-sample, `-count=5 -benchtime=1x`, Windows amd64)

| Taille | DTM Off (ns/op) | DTM On (ns/op) | Δ |
|---|---:|---:|---:|
| F(1M)  | 11 085 900 | 10 539 700 | **−4.93 %** (sous le seuil) |
| F(10M) | 46 947 600 | 44 047 200 | **−6.18 %** (au-dessus du seuil) |

**Interprétation** : DTM On est marginalement plus rapide. Au seuil ≥ 5 % défini par la directive 1 du `CLAUDE.md`, le verdict est **borderline** (sous le seuil à 1M, au-dessus à 10M). La mesure est *single-sample* (`-benchtime=1x`) donc bruitée ; une régression statistique rigoureuse demanderait `-count=20 -benchtime=3s` sur machine quiescente.

### Décision finale : **KEEP**

Trois raisons :

1. **Coût de maintenance fortement réduit** depuis l'atomic conversion (E1-R1) : l'invariant single-writer A-18 a disparu, donc le coût conceptuel de DTM est essentiellement nul désormais.
2. **Gain plausible (5-6 %) au régime ≥ 10M**, qui est précisément le régime *production* pour ce projet (le régime ≤ 1M est dominé par d'autres frais).
3. **Principe de prudence** : la suppression est irréversible alors que la conservation peut être révisée si une mesure ultérieure plus rigoureuse infirme le gain.

## Consequences

### Positive

- Réduction LOC ≥ 283 si suppression.
- Élimination d'une source de complexité accidentelle non profilée.
- Atomic conversion ferme la race théorique en attendant la décision finale.

### Negative / Trade-offs

- Si suppression, perte d'un *failsafe* dynamique pour les machines très atypiques (charge externe pendant le calcul). Mitigation : la calibration au démarrage reste.

### Risks and Mitigations

- **Risque** : la suppression casse un cas d'usage non testé. **Mitigation** : période d'observation 1 release avec `Deprecated` annotation avant suppression définitive.

## Alternatives Considered

- **Conserver les deux sans benchmark** : rejeté, contredit la directive 1 du `CLAUDE.md`.
- **Supprimer la calibration au lieu du DTM** : rejeté, la calibration au démarrage est bien plus testée et a une justification chiffrée (`internal/calibration/profile.go:196-224`).

## References

- Benchmark artifacts : `docs/audits/bench-dtm-{on,off}.txt`
- Implementation : `internal/fibonacci/dtm_bench_test.go`, `internal/fibonacci/threshold/manager.go`

## Status note (2026-06-10)

Précisions factuelles issues de l'audit documentaire 2026-06 ; le corps
historique ci-dessus est conservé tel quel et la décision KEEP reste inchangée.

- `internal/fibonacci/threshold/manager.go` compte 353 lignes à HEAD
  (2026-06-10), contre ~283 à la rédaction : extensions de l'audit 2026-06,
  dont le mutex qui sérialise désormais `Reset` et tout accès au
  `MetricsBuffer` (fix de data race, commit `a2e4eee`).
- La référence « justification chiffrée (`internal/calibration/profile.go:196-224`) »
  a dérivé : ce range couvre la fin de `renameAtomic`. La cible visée est la
  validation de profil `CalibrationProfile.IsValid` (référence symbolique
  `profile.go:IsValid` ; lignes 205-234 à HEAD).
- Les artefacts `docs/audits/bench-dtm-{on,off}.txt` (Decision §2, References)
  ont été purgés avec le reste de `docs/audits/` (CHANGELOG, Housekeeping
  2026-06-10) ; ils restent régénérables à la demande via `BenchmarkFibonacciDTM`
  (`internal/fibonacci/dtm_bench_test.go`). Les résultats chiffrés restent inline
  dans la table ci-dessus, donc la décision KEEP n'en dépend pas.
