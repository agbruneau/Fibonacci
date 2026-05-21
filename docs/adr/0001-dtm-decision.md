# ADR-0001: Sort de `DynamicThresholdManager` vs `internal/calibration/`

- **Status**: Proposed (stub — décision finale après benchmarks E7-R1)
- **Date**: 2026-05-21
- **Audit source**: `Audit - Global - FibGo.md` §2.1 C5, §5.2 P1-01

## Context

Le projet maintient **deux** mécanismes adaptatifs pour les seuils FFT/parallèle :

1. `internal/calibration/` (~1 686 LOC) produit `OptimalFFTThreshold`/`OptimalParallelThreshold` *au démarrage* via micro-benchmarks (`internal/calibration/profile.go:30-32`).
2. `internal/fibonacci/threshold/manager.go` (~283 L) ajuste ces seuils *en cours de calcul* via une fenêtre glissante de 20 métriques avec hystérésis.

L'audit a relevé que cette redondance n'est appuyée par aucun benchmark publié dans `docs/audits/`, et que l'invariant single-writer du DTM (A-18, lignes 167-174) est fragile sans `atomic`.

## Decision

Deux étapes :

1. **Court terme (S1-T1)** : convertir les trois champs mutés hors verrou (`currentFFTThreshold`, `currentParallelThreshold`, `lastAdjustment`) en `atomic.Int64`/`atomic.Pointer[time.Time]`. L'invariant A-18 devient inutile.
2. **Moyen terme (S3-T1)** : produire `docs/audits/bench-dtm-{on,off}.txt` sur les 5 tailles `BenchmarkFibonacci_*`. Si le gain DTM-on est `< 5%`, supprimer le DTM (`internal/fibonacci/threshold/manager.go`, ~283 LOC).

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

- PRD : `Audit - PRD - FibGo.md` Epic E7
- Plan : `Audit - PRDPLan - FibGo.md` S3-T1, S3-T2
