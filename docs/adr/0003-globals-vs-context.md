# ADR-0003: Globaux `bigfft` mutables → `atomic.Int64`

- **Status**: Accepted
- **Date**: 2026-05-21
- **Context source**: hardening sprint mai 2026 (commit `c0cc530`).

## Context

Trois variables globales sont exportées et mutables sans synchronisation :

- `fftThreshold` (`internal/bigfft/fft.go:35`)
- `ParallelFFTRecursionThreshold` (`internal/bigfft/fft_recursion.go:28`)
- `MaxParallelFFTDepth` (`internal/bigfft/fft_recursion.go:33`)

Toutes lues sur le hot path FFT et modifiables runtime par n'importe quel consommateur (`SetFFTParallelismConfig`, `calibration`). Contredit directement la directive 4 du `CLAUDE.md` (« Pas de nouveaux globals dans `bigfft/` »).

Une trajectoire alternative existe via `FFTContext` (`internal/bigfft/context.go`), opt-in, qui encapsule la configuration. Mais les globaux cohabitent encore et restent le chemin par défaut.

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

- Positions à HEAD (2026-06-10) : `fftThreshold` → `fft.go:38` ;
  `parallelFFTRecursionThreshold` → `fft_recursion.go:32` ;
  `maxParallelFFTDepth` → `fft_recursion.go:37` (décalage de 3-4 lignes par
  rapport aux positions citées en Context).
- Types effectifs : `fftThreshold` est bien en `atomic.Int64` ; les deux
  variables de récursion parallèle sont en `atomic.Uint64`.
- Accesseurs effectifs : `getFFTThreshold()` est privé ;
  `GetParallelFFTRecursionThreshold()` et `GetMaxParallelFFTDepth()` sont
  **exportés** (`fft_recursion.go:46,49`). La section References ci-dessus est
  correcte ; la section Decision les citait avec une casse privée.
