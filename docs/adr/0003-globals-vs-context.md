# ADR-0003: Globaux `bigfft` mutables → `atomic.Int64`

- **Status**: Accepted
- **Date**: 2026-05-21
- **Audit source**: `Audit - Global - FibGo.md` §4.1 item (c), §5.1 P0-01

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

- PRD : `Audit - PRD - FibGo.md` Epic E1-R3
- Plan : `Audit - PRDPLan - FibGo.md` S1-T3
