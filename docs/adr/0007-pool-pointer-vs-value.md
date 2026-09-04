# ADR-0007: SA6002 (`sync.Pool.Put` de slice) — décision mesurée

- **Status**: Accepted
- **Date**: 2026-05-28
- **Context source**: audit mai 2026, constat `A4-01` (MAJEUR), axe 4 Idiomatique (rapport archivé en historique git).

## Context

`staticcheck SA6002` signale 8 sites (`internal/bigfft/pool.go` ; `pool_warming.go` — positions relevées à la rédaction, voir la *Status note* pour les lignes courantes) où un `sync.Pool.Put` reçoit une **valeur de slice** (`[]big.Word`, `fermat`, `[]nat`, `[]fermat`). Mettre une valeur non *pointer-like* dans une interface boxe l'en-tête (3 mots), provoquant une allocation à chaque `Put` — contre-productif dans la couche de pooling censée éliminer les allocations.

La « correction » naturelle (pooler des `*[]T`) n'est bénéfique que si le **box pointeur est réutilisé** d'un `Get` à l'autre. Or l'API actuelle `acquire*()` retourne une slice **valeur** et `release*(slice)` reçoit une slice **valeur** : un `release` qui ferait `Put(&s)` sur son paramètre **fait s'échapper** `&s` vers le tas → une allocation par appel, **identique** au boxing.

## Decision

**Ne pas modifier le code des pools.** Documenter la décision (cet ADR) et **exclure SA6002** pour `pool.go`/`pool_warming.go` dans `.golangci.yml`, avec renvoi à cet ADR.

### Preuve mesurée (micro-benchmark, `go test -bench`, 1e6 itérations)

| Style de `Put` | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `Put(sliceValue)` — **code actuel** | 16.88 | 24 | **1** |
| `Put(&s)` — préservant la signature `release([]T)` | 16.53 | 24 | **1** |
| `Get`/`Put` d'un **même `*[]T` réutilisé** | 4.30 | 0 | **0** |

Conclusion : le correctif préservant l'API est **strictement neutre** en allocation. Le seul gain réel (0 alloc) exige de **threader `*[]T`** à travers tout l'API d'allocation FFT (`acquire*`/`release*` + leurs appelants) — c'est la **migration `FFTContext` exclusive**, déjà tracée et reportée (ADR-0004 §B1, won't-fix release courante).

Modifier 4 pools globaux d'un module sous gel perf pour un gain nul violerait « Surgical Changes » et la directive #4. L'audit lui-même propose explicitement l'annotation/exclusion comme alternative légitime à coût nul.

## Consequences

### Positive

- Aucun risque introduit sur un module sensible/gelé pour zéro gain.
- Le warning SA6002 est neutralisé de façon **documentée et tracée** (plus de bruit lint masquant un vrai défaut).
- L'invariant `releaseWordSlice` route-sur-`cap` reste intact.

### Negative / Trade-offs

- Le boxing (24 B / `Put`) subsiste. C'est le coût accepté jusqu'à la migration `FFTContext`. Sur le hot path il est largement amorti par le travail arithmétique (FFT de plusieurs Mo).

### Risks and Mitigations

- *Masquage d'un futur vrai SA6002 dans ces fichiers* : l'exclusion est ciblée par path (`pool.go`/`pool_warming.go`) ; toute nouvelle classe de pool ailleurs reste contrôlée. À ré-examiner lors de la migration `FFTContext`.

## Alternatives Considered

- **Pooler des `*[]T` en préservant la signature `release([]T)` (`Put(&s)`)** — rejetée : mesurée neutre (l'échappement de `&s` remplace le boxing).
- **Refactor complet `acquire/release` vers `*[]T` (zéro alloc)** — reporté : touche tout le hot path FFT + ses appelants ; relève de la migration `FFTContext` (ADR-0004 §B1).
- **8 annotations `//nolint` inline** — rejetée au profit d'une exclusion centralisée (cohérent avec les exclusions SA1019/G304 existantes), moins de bruit sur le hot path.

## References

- Code concerné : `internal/bigfft/pool.go`, `internal/bigfft/pool_warming.go`.
- Config : `.golangci.yml` (exclude-rule SA6002 ciblée).
- Related ADR(s) : ADR-0004 §B1 (migration `FFTContext`, backlog).
- Audit : axe 4 Idiomatique, constat `A4-01` (rapport archivé en historique git).

## Status note (2026-09-04 — remplace les notes du 2026-06-10 et du 2026-08-07)

**Plus aucun numéro de ligne n'est cité ici.** Les quatre sites de `pool.go` ont
dérivé trois fois sans qu'un seul `Put` change : 148/245/333/421 à la rédaction,
147/243/330/417 le 2026-08-07, **141/227/304/381** le 2026-09-04
(`grep -n '\.Put(' internal/bigfft/pool.go`). C'était la référence qui était
fragile, pas le code. Les huit sites SA6002 sont désormais nommés par la
fonction qui les contient, ce qui ne peut plus dériver :

- `internal/bigfft/pool.go` : `releaseWordSlice`, `releaseFermat`,
  `releaseNatSlice`, `releaseFermatSlice` — un `Put` de valeur chacun ;
- `internal/bigfft/pool_warming.go` : les quatre `Put` de la boucle de
  pré-chauffage de `PreWarmPools` (dont les positions, elles, n'ont pas bougé
  depuis 2026-08-07).

La décision et l'exclusion ciblée par chemin dans `.golangci.yml` restent
inchangées : elles portent sur les fichiers, pas sur les numéros de ligne.

La table « Preuve mesurée » du §Decision n'a **pas** d'artefact dans
`docs/audits/` : c'est un micro-benchmark jetable de mai 2026, non réexécuté
depuis, et la décision repose sur son ordre de grandeur (correctif alloc-neutre)
plutôt que sur ses trois chiffres.

Le type `fftState` cité à l'origine par le §Context et par les Alternatives
**n'existe plus** dans l'arbre (grep insensible à la casse sur `*.go` : zéro
occurrence). Il a été supprimé avec la machinerie FFT-05 — voir
[ADR-0009 §R2](0009-audit-2026-07-cleanup-and-rejected-fib05.md). Les mentions
correspondantes ont été retirées le 2026-08-07 ; le commentaire d'exclusion
SA6002 dans `.golangci.yml` a été corrigé au même titre.

### La cible « migration `FFTContext` » n'existe plus

Le §Decision, les Consequences et deux Alternatives renvoient le seul gain réel
(zéro allocation, en threadant des `*[]T`) à « la migration `FFTContext`
exclusive, déjà tracée et reportée ». Ce renvoi est aujourd'hui sans objet :
l'API `FFTContext` a été **supprimée de l'arbre** le 2026-07-11 (commit
`23ab593`, addendum [ADR-0004 §B1](0004-backlog-decisions.md)) et la migration
reste classée WONT-FIX.

Ce que cela change et ne change pas :

- **La décision tient telle quelle.** Elle repose sur une mesure — le correctif
  préservant la signature est alloc-neutre — et non sur l'existence d'un
  chantier à venir.
- **Le boxing de 24 B par `Put` n'est plus « le coût accepté jusqu'à la
  migration `FFTContext` »** : c'est le coût accepté, sans échéance. Le lever
  demanderait de threader `*[]T` à travers `acquire*`/`release*` et tous leurs
  appelants, pour un gain jamais mesuré sur le hot path complet.
- La clause « à ré-examiner lors de la migration `FFTContext` » des Risks se lit
  désormais : à ré-examiner si quelqu'un mesure ce que coûte réellement le
  boxing sur un calcul complet, plutôt qu'au `Put` isolé.
