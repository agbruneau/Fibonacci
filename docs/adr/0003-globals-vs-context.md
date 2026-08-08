# ADR-0003: Globaux `bigfft` mutables → `atomic.Int64`

- **Status**: Accepted
- **Date**: 2026-05-21
- **Context source**: hardening sprint mai 2026 (commit `c0cc530`).

## Context

Trois variables globales sont exportées et mutables sans synchronisation :

- `fftThreshold` (`internal/bigfft/fft.go:fftThreshold`)
- `ParallelFFTRecursionThreshold` (`internal/bigfft/fft_recursion.go:parallelFFTRecursionThreshold`)
- `MaxParallelFFTDepth` (`internal/bigfft/fft_recursion.go:maxParallelFFTDepth`)

Toutes lues sur le hot path FFT et modifiables runtime par n'importe quel consommateur (`SetFFTParallelismConfig`, `calibration`). Contredit directement la directive 4 du `CLAUDE.md` (« Pas de nouveaux globals dans `bigfft/` »).

Une trajectoire alternative existait alors via `FFTContext` (`internal/bigfft/context.go`), opt-in, qui encapsulait la configuration. Mais les globaux cohabitaient encore et restaient le chemin par défaut.

## Decision

**Approche minimale-invasive** : convertir les globaux en `atomic.Int64` (typés derrière des accesseurs `getFFTThreshold()` / `getParallelFFTRecursionThreshold()` / `getMaxParallelFFTDepth()`), tout en préservant l'API publique existante (`SetFFTParallelismConfig`).

- L'API publique inchangée évite de casser la calibration et les tests existants.
- Les écritures concurrentes sont sérialisées par `atomic.StoreInt64`.
- Les lectures hot path sont `atomic.LoadInt64` — coût négligeable (un load fence).
- La migration vers `FFTContext` exclusif est reportée à une release ultérieure si elle se justifie par un *use case multi-tenant* effectif.

## Consequences

### Positive

- Data race théorique éliminée sans casser l'API.
- Pas de duplication de code, pas de migration *big-bang* vers `FFTContext`.

### Negative / Trade-offs

- Les globaux restent (mais désormais thread-safe). La résorption *complète* (suppression des globaux au profit de `FFTContext`) est reportée. Tracé dans le backlog.

### Risks and Mitigations

- **Risque** : oubli d'un site de lecture/écriture. **Mitigation** : grep `fftThreshold|ParallelFFTRecursionThreshold|MaxParallelFFTDepth` doit retourner uniquement les accesseurs après PR.

## Alternatives Considered

- **Migration *big-bang* vers `FFTContext` exclusif** : rejeté, casse l'API publique et la calibration ; effort > 1 sprint.
- **`sync.RWMutex` autour des globaux** : rejeté, plus coûteux que `atomic.Int64` sur le hot path.

## References

- Implementation : `internal/bigfft/fft.go` (`fftThreshold` atomic), `internal/bigfft/fft_recursion.go` (`parallelFFTRecursionThreshold`, `maxParallelFFTDepth`)
- Accessors : `getFFTThreshold()`, `GetParallelFFTRecursionThreshold()`, `GetMaxParallelFFTDepth()`

## Status note (2026-06-10)

Précisions factuelles (audit documentaire 2026-06) ; la décision reste
inchangée.

- Les positions ont été recomptées deux fois (2026-06-10, 2026-08-07) sans
  qu'aucune de ces trois variables change : le `file:line` était le défaut, pas
  le code. Les renvois sont désormais ancrés sur le symbole
  (`fft.go:fftThreshold`, `fft_recursion.go:parallelFFTRecursionThreshold`,
  `fft_recursion.go:maxParallelFFTDepth`) et ne peuvent plus dériver.
- Types effectifs : `fftThreshold` est bien en `atomic.Int64` ; les deux
  variables de récursion parallèle sont en `atomic.Uint64`.
- Accesseurs effectifs : `getFFTThreshold()` est privé ;
  `fft_recursion.go:GetParallelFFTRecursionThreshold` et
  `fft_recursion.go:GetMaxParallelFFTDepth` sont **exportés**. La section
  References ci-dessus est correcte ; la section Decision les citait avec une
  casse privée.

## Status note (2026-08-07)

- `FFTContext` et `internal/bigfft/context.go` (cités en Context) **n'existent
  plus** : l'API opt-in a été supprimée de l'arbre le 2026-07-11 (addendum
  ADR-0004 §B1). Les globaux atomiques restent l'unique chemin.
- `CLAUDE.md` a été retiré du dépôt le 2026-07-31 (commit `869bd6a`) ; le renvoi
  à sa « directive 4 » est une citation historique, il n'existe plus de fichier
  de directives dans l'arbre.
