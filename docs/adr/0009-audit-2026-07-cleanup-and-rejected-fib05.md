# ADR-0009: Audit 2026-07 — purge bigfft (Phase 4E), rétention oracle, et rejet FIB-05

- **Status**: Accepted
- **Date**: 2026-07-03
- **Context source**: audit exhaustif 2026-07 (rapport `audit.md`, exécuté via
  le plan `auditPlan.md` ; les deux fichiers ont été purgés post-exécution au
  commit `d10299b`, cf. CHANGELOG) — (orchestration
  multi-agents, vérification par grep/tests/benchstat + panels réfutateurs
  avant chaque commit).

## Context

L'audit 2026-07 a corrigé ~40 findings sur toute la base. Trois décisions
structurantes méritent d'être matérialisées en ADR : deux suppressions de
code mort dans `internal/bigfft/` (Phase 4E) qui touchent des symboles
historiquement présents mais sans consommateur prod, et un rejet mesuré
(FIB-05) d'une « optimisation » mémoire qui s'est révélée être une
régression de performance. Sans cet ADR, un audit futur re-suggérerait ces
mêmes candidats sans le nouvel élément de preuve.

## Decision

### R1 — `bigfft/scan.go` supprimé (OVR-01)

`FromDecimalString` + le type `scanner` (parsing décimal quadratique) n'avaient
**aucun consommateur prod** : le CLI parse via `strconv`/`SetString`. Refs =
`scan_test.go` + `misc_extra_test.go` uniquement (API héritée de bigfft
upstream jamais branchée). Fichier + tests supprimés. Rend FFT-07 sans objet.

À revoir si : un chemin d'entrée décimal haute performance devient requis
(rebrancher depuis l'historique git plutôt que de garder du mort).

### R2 — Machinerie `fftState`/`fftStatePool` supprimée (FFT-05)

`fftState`, `fftStatePool`, `acquireFFTState`, `releaseFFTState`,
`maxPooledFFTTmpCap`, et le wrapper `fourierWithState` : **zéro appelant prod**
(le seul appelant, `fourier()`, passait toujours `state=nil`). `fourier()`
acquiert désormais `tmp`/`tmp2` directement via `acquireFermat`/`releaseFermat`
— **alloc-neutre** (benchstat : allocs/op inchangé, cf. R4). L'invariant
CLAUDE.md `bigfft/pool.go` (mention `fftStatePool`/A2-05) est synchronisé.

### R3 — Cluster oracle bigfft : **conservé** et documenté (OVR-10)

`Poly.Mul/Transform/NTransform`, `PolValues.InvNTransform/Clone`,
`TransformCache.Get/Put/Clear`, le trio `*Cached` non-bump, et
`AddVV/SubVV/AddMulVVW` (ce dernier issu de la fusion `arith_amd64.go` +
`arith_generic.go` → `arith.go`, FFT-06) n'ont **aucun appelant prod** (la
prod route par `mulFFT`/`fftmulTo`/`sqrFFT`/`fftsqrTo` → variantes `*WithBump`).
Ils sont **conservés** : ce sont des **oracles de fuzz / cross-validation** —
chemins de référence non-bump contre lesquels les tests valident les chemins
bump optimisés. Chaque fonction porte désormais un commentaire « test oracle ».
Suppression rejetée : détruirait la validation croisée sur un package
extrêmement sensible.

À revoir si : la stratégie de test migre vers une autre forme de validation
(golden élargi, property-based), rendant les oracles redondants.

### R4 — FIB-05 : réduction du multiplicateur d'arène ×15 → ×5-6 **rejetée**

L'audit recommandait de réduire le multiplicateur d'arène de ×15 à ×5-6
(`arenaTotalWords`, `acquireSizingForN`), au motif que `prepareStateForN` ne
consomme que 5 slots. La réduction à ×5 a été **implémentée puis abandonnée
en Phase 3D** après mesure benchstat.

**Preuve** (bisect isolé, HEAD propre vs HEAD+3D-only, même session,
`benchtime=1s`) :

| Benchmark | ×15 (base) | ×5 | Δ |
|---|---|---|---|
| FastDoubling/10M | 29.4 ms | 34.7 ms | **+18 %** |
| FFTBased/10M | 32.0 ms | 42.9 ms | **+34 %** |
| geomean sec/op | — | — | **+26 %** |
| allocs/op | — | — | **~0 %** (inchangé) |

La régression **survit à l'inversion de l'ordre d'exécution** (non thermique)
et l'économie mémoire est réelle (F(10M) : ~12 Mo → ~4 Mo, FFT B/op −31 %),
mais le gain mémoire **ne compense pas** le coût CPU. Le mécanisme : le
sur-dimensionnement ×15 fournit une marge de scratch/croissance in-place qui
évite du travail (copies/localité) **sans** allocation supplémentaire — le
×15 est donc **charge utile intentionnelle**, pas un reliquat mort. Gate
Directive #1 (régression > 5 % = blocage) → **rejet**.

À revoir si : un balayage complet du multiplicateur (plusieurs paliers,
benchstat à chacun) démontre un palier intermédiaire à gain mémoire net
**sans** régression CPU > 5 %.

### R5 — ADR-0004 §B1 : justification calibration caduque (OVR-08)

L'annotation d'ADR-0004 §B1 (« le code de calibration s'appuie sur
`SetFFTThreshold` ») est **caduque** : la calibration n'appelle que
`bigfft.Mul` (`microbench.go`), jamais `SetFFTThreshold`/`Set*FFTParallelismConfig`
(uniques écrivains post-init, test-only). Annoté dans
[`0004-backlog-decisions.md`](0004-backlog-decisions.md).

## Consequences

- `internal/bigfft/` perd ~250 LOC de code mort strict (scan + fftState +
  fusion arith) sans changement de comportement ni de perf (benchstat ~,
  WSL `-race` propre, gardiens verts).
- Le multiplicateur d'arène ×15 est désormais **documenté comme intentionnel**
  dans CLAUDE.md (`fibonacci/fastdoubling.go`) — FIB-05 ne doit pas être
  re-tenté sans le protocole R4.
- Les oracles bigfft sont explicitement étiquetés, protégeant la validation
  croisée d'une suppression future naïve.
