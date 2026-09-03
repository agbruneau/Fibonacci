# ADR-0001: Sort de `DynamicThresholdManager` vs `internal/calibration/`

- **Status**: Accepted (decision: KEEP, with reduced maintenance cost via atomic conversion).
- **Date**: 2026-05-21 (mesures complétées)
- **Context source**: hardening sprint mai 2026 (commits `c0cc530` → `3d8b977`).

## Context

Le projet maintient **deux** mécanismes adaptatifs pour les seuils FFT/parallèle :

1. `internal/calibration/` (~1 686 LOC) produit `OptimalFFTThreshold`/`OptimalParallelThreshold` *au démarrage* via micro-benchmarks (champs `OptimalParallelThreshold` / `OptimalFFTThreshold` de `internal/calibration/profile.go:CalibrationProfile` ; le champ voisin `OptimalStrassenThreshold` est hors sujet ici).
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
- **Supprimer la calibration au lieu du DTM** : rejeté, la calibration au démarrage est bien plus testée et a une justification chiffrée (`internal/calibration/profile.go:CalibrationProfile.IsValid`).

## References

- Benchmark artifacts : `docs/audits/bench-dtm-{on,off}.txt`
- Implementation : `internal/fibonacci/dtm_bench_test.go`, `internal/fibonacci/threshold/manager.go`

## Status note (2026-06-10)

Précisions factuelles issues de l'audit documentaire 2026-06 ; le corps
historique ci-dessus est conservé tel quel et la décision KEEP reste inchangée.

- `internal/fibonacci/threshold/manager.go` compte **328 lignes** (recompté le
  2026-08-07 ; le « 353 » noté ici en 2026-06-10 a dérivé), contre ~283 à la
  rédaction : extensions de l'audit 2026-06, dont le mutex qui sérialise
  désormais `Reset` et tout accès au `MetricsBuffer` (fix de data race,
  commit `a2e4eee`).
- La référence « justification chiffrée » pointait un `file:line`
  (`profile.go:196-224`) qui a dérivé deux fois et désignait en réalité la fin
  de `renameAtomic`. La cible visée est la validation de profil, désormais
  ancrée sur le symbole : `profile.go:CalibrationProfile.IsValid`. Aucun
  numéro de ligne n'est conservé — c'est précisément ce qui rendait la
  référence fausse.
- Les artefacts `docs/audits/bench-dtm-{on,off}.txt` (Decision §2, References)
  ont été purgés avec le reste de `docs/audits/` (CHANGELOG, Housekeeping
  2026-06-10) ; ils restent régénérables à la demande via `BenchmarkFibonacciDTM`
  (`internal/fibonacci/dtm_bench_test.go`). Les résultats chiffrés restent inline
  dans la table ci-dessus, donc la décision KEEP n'en dépend pas.
- `CLAUDE.md` (renvois « directive 1 » ci-dessus) a été retiré du dépôt le
  2026-07-31 (commit `869bd6a`) : ces renvois sont des citations historiques, il
  n'existe plus de fichier de directives dans l'arbre.

## Status note (2026-09-03) — câblage exécuté, gain non reproduit

L'audit 2026-09 (M-04) a constaté que la décision **KEEP** ci-dessus n'avait
jamais été suivie d'un câblage : `fibonacci.Options.EnableDynamicThresholds`
n'était mis à `true` par aucun chemin de production — ni flag, ni variable
d'environnement, ni `internal/app`. Le gain de 5-6 % qui justifie le KEEP
n'était donc livré à aucun utilisateur du binaire, et `docs/ARCH.md` /
`docs/PERFORMANCE.md` présentaient le DTM comme un composant actif du pipeline.

**Câblage livré** : `--dynamic-thresholds` / `FIBCALC_DYNAMIC_THRESHOLDS`,
**désactivé par défaut**, propagé sur les deux chemins (CLI et TUI).

**Mesure** (`docs/audits/bench-dtm-2026-09.txt`) : `BenchmarkFibonacciDTM`,
`-count=8 -benchtime=1x -benchmem`, Intel Core Ultra 9 275HX, go1.27.0.

| Benchmark | Off | On | Δ |
|---|---:|---:|---|
| F(1M) sec/op | 3,144 ms ± 5 % | 3,041 ms ± 5 % | ~ (p=0,279) |
| F(10M) sec/op | 26,20 ms ± 3 % | 25,71 ms ± 5 % | ~ (p=0,382) |
| geomean sec/op | — | — | −2,59 % (non significatif) |
| F(1M) allocs/op | 117,5 ± 3 % | 138,5 ± 3 % | **+17,87 % (p=0,006)** |
| geomean B/op | — | — | +0,04 % |

**Interprétation.** Le gain de 5-6 % de la table d'origine **ne se reproduit
pas**. Les deux écarts CPU sont dans le bruit (`~`, p > 0,25), ce qui était
prévisible : la mesure d'origine était `-count=5 -benchtime=1x` *single-sample*
et l'ADR la qualifiait lui-même de « bruitée » en appelant une reprise à
`-count=20`. Cette reprise à `-count=8` avec test statistique ne montre aucun
effet CPU. Le seul écart significatif est un **surcoût** d'allocations à 1M
(+17,9 %), attribuable à l'enregistrement des métriques par itération.

**Décision : KEEP maintenu, défaut `false`.** Le sous-système reste dans
l'arbre — son coût de maintenance est nul depuis la conversion atomique (E1-R1)
et il est désormais réellement atteignable — mais rien ne justifie de l'activer
par défaut. Les deux raisons 1 et 3 du KEEP d'origine tiennent ; la raison 2
(« gain plausible 5-6 % au régime ≥ 10M ») est **caduque sur cette machine**.

À revoir si : une mesure sur une autre microarchitecture, ou sur un régime
au-delà de F(10M), montre un gain significatif ; le flag existe désormais pour
la produire sans modifier le code.
