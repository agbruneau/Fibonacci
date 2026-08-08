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

## Addendum (2026-07-07) — R4 revisité : balayage complet, adoption de ×10

Le protocole ouvert par R4 (« À revoir si : un balayage complet… ») a été
**exécuté**. Contrairement à la prédiction de R4 (fondée sur une mesure Ryzen où
×5 régressait de +18 à +34 %), le balayage sur la **machine de référence
actuelle** ne montre **aucune régression CPU** en réduisant le multiplicateur.

**Protocole** : paliers m ∈ {12, 10, 8, 6} vs base ×15, A/B même session,
`-benchmem -benchtime=1s -count=10`, `BenchmarkFibonacci/(FastDoubling|FFTBased)`
à 1M et 10M. Machine : **Intel Core Ultra 9 275HX** (goarch amd64, 24 threads).
Gardiens + golden verts à chaque palier.

| Multiplicateur | geomean sec/op | FFTBased/10M B/op | geomean B/op | allocs/op |
|---|---|---|---|---|
| ×15 (base) | — | — | — | — |
| ×12 | −0,66 % | −9,92 % | −2,36 % | ~ |
| **×10** | **−2,92 %** | **−17,34 %** | −5,46 % | ~ |
| ×8 | −0,37 % | −22,83 % | −7,70 % | ~ |
| ×6 | −3,19 % | −31,24 % | −12,14 % | ~ |

Le CPU reste dans le bruit à tous les paliers (−0,4 % à −3,2 % geomean, aucun
> +5 %) ; la mémoire baisse de façon monotone ; les allocs ne bougent pas (pas
de fallback d'arène mesurable, bien que les étapes de doublement FFT puissent
consommer jusqu'à 12 temporaires — le scratch au-delà de l'arène passe par le
bump allocator).

**Confirmation anti-thermique (ordre inversé)** pour le palier retenu ×10
(palier mesuré en premier, base ×15 en second) : geomean sec/op **−0,57 %**
(vs −2,92 % en ordre direct), FFTBased/10M B/op **−15,74 %** (vs −17,34 %),
allocs plates. Le gain CPU est donc **dans le bruit** (aucune régression, ordre
direct partiellement thermique) ; le gain **mémoire est robuste et
indépendant de l'ordre** (−16 à −17 % de B/op FFT 10M).

**Décision : ×15 → ×10 adopté.** Justification : gain mémoire net et
order-stable (≈ −16 % B/op FFT 10M, allocs inchangées) à **coût CPU nul**
(≤ +2 % geomean dans les deux ordres) — les critères Cas A du protocole R4 sont
satisfaits. La conclusion R4 « ×15 = charge utile intentionnelle » reste **vraie
sur le Ryzen** de la mesure d'origine ; le multiplicateur optimal est
**dépendant de la microarchitecture** (coût du `memclr` d'arène / pression
cache). `internal/fibonacci/fastdoubling.go` (`acquireSizingForN`) et
`internal/fibonacci/memory/arena.go` (`arenaTotalWords`) sont mis à jour en
miroir ; `docs/audits/bench-baseline.txt` et le profil PGO `cmd/fibcalc/default.pgo`
sont régénérés post-adoption.

À revoir si : un mainteneur travaille sur une microarchitecture où le
sur-dimensionnement d'arène redevient favorable (re-balayer alors selon ce même
protocole avant de changer la valeur).

## Status note (2026-08-07)

- `CLAUDE.md` (cité en R2 et en Consequences comme porteur de l'invariant
  `bigfft/pool.go` et de la justification du multiplicateur d'arène) a été retiré
  du dépôt le 2026-07-31 (commit `869bd6a`). Ces renvois sont des citations
  historiques ; les invariants correspondants vivent désormais dans les
  doc-comments de `internal/bigfft/pool.go` et
  `internal/fibonacci/fastdoubling.go`.
