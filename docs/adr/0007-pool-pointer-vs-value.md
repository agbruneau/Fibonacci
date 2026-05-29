# ADR-0007: SA6002 (`sync.Pool.Put` de slice) — décision mesurée

- **Status**: Accepted
- **Date**: 2026-05-28
- **Context source**: audit mai 2026, constat `A4-01` (MAJEUR), axe 4 Idiomatique (rapport archivé en historique git).

## Context

`staticcheck SA6002` signale 8 sites (`internal/bigfft/pool.go:148,245,333,421` ; `pool_warming.go:70,79,88,97`) où un `sync.Pool.Put` reçoit une **valeur de slice** (`[]big.Word`, `fermat`, `[]nat`, `[]fermat`). Mettre une valeur non *pointer-like* dans une interface boxe l'en-tête (3 mots), provoquant une allocation à chaque `Put` — contre-productif dans la couche de pooling censée éliminer les allocations.

La « correction » naturelle (pooler des `*[]T`) n'est bénéfique que si le **box pointeur est réutilisé** d'un `Get` à l'autre. Or l'API actuelle `acquire*()` retourne une slice **valeur** et `release*(slice)` reçoit une slice **valeur** : un `release` qui ferait `Put(&s)` sur son paramètre **fait s'échapper** `&s` vers le tas → une allocation par appel, **identique** au boxing.

## Decision

**Ne pas modifier le code des pools.** Documenter la décision (cet ADR) et **exclure SA6002** pour `pool.go`/`pool_warming.go` dans `.golangci.yml`, avec renvoi à cet ADR.

### Preuve mesurée (micro-benchmark, `go test -bench`, 1e6 itérations)

| Style de `Put` | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `Put(sliceValue)` — **code actuel** | 16.88 | 24 | **1** |
| `Put(&s)` — préservant la signature `release([]T)` | 16.53 | 24 | **1** |
| `Get`/`Put` d'un **même `*[]T` réutilisé** | 4.30 | 0 | **0** |

Conclusion : le correctif préservant l'API est **strictement neutre** en allocation. Le seul gain réel (0 alloc) exige de **threader `*[]T`** à travers tout l'API d'allocation FFT (`acquire*`/`release*` + leurs appelants + `fftState`) — c'est la **migration `FFTContext` exclusive**, déjà tracée et reportée (ADR-0004 §B1, won't-fix release courante).

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
- **Refactor complet `acquire/release` vers `*[]T` (zéro alloc)** — reporté : touche tout le hot path FFT + `fftState` + appelants ; relève de la migration `FFTContext` (ADR-0004 §B1).
- **8 annotations `//nolint` inline** — rejetée au profit d'une exclusion centralisée (cohérent avec les exclusions SA1019/G304 existantes), moins de bruit sur le hot path.

## References

- Code concerné : `internal/bigfft/pool.go`, `internal/bigfft/pool_warming.go`.
- Config : `.golangci.yml` (exclude-rule SA6002 ciblée).
- Related ADR(s) : ADR-0004 §B1 (migration `FFTContext`, backlog).
- Audit : axe 4 Idiomatique, constat `A4-01` (rapport archivé en historique git).
