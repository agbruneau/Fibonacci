# ADR-0002: Stratégie `recover()` dans `bigfft`

- **Status**: Accepted
- **Date**: 2026-05-21
- **Audit source**: `Audit - Global - FibGo.md` §4.2

## Context

`internal/bigfft/fft.go:41-101` installe quatre `recover()` indistincts dans les fonctions publiques `Mul`, `MulTo`, `Sqr`, `SqrTo`. Ces `recover` capturent *toute* panic, y compris les violations de post-conditions algorithmiques internes émises par `internal/bigfft/fermat.go` (`panic("len(z) > 2n+1")`, `panic("unexpected carry after normalization")`).

Conséquence : un bug authentique dans la réduction modulaire de Fermat ressort comme un `error` opaque indistinguable d'une erreur d'arité, et passe silencieusement tout test qui n'inspecte que `err != nil`.

L'intention initiale (visible dans 48 LOC de wrappers `*Safe` à `bigfft/fermat.go:291-339`) distinguait théoriquement :
- **pré-conditions externes** (arité, taille) → à transformer en `error` retourné.
- **post-conditions internes** (invariants algorithmiques) → à laisser propager (panic).

Mais aucun consommateur de production n'utilise les wrappers `*Safe`.

## Decision

**Restreindre le `recover()` global à un set d'erreurs sentinelles** plutôt que de capturer toute panic.

1. Définir un type `bigfftPanic` interne pour les pré-conditions explicitement attendues (ex. tailles incohérentes).
2. Le `recover()` global ne capture que les `bigfftPanic`; les autres panics propagent.
3. Les `panic` internes de `fermat.go` (post-conditions) sont conservés et propagent normalement.
4. Les wrappers `*Safe` orphelins (`MulSafe`, `SqrSafe`, `ShiftSafe`, `AddSafe`, `SubSafe`) sont **supprimés** (48 LOC inutilisés).

## Consequences

### Positive

- Régressions algorithmiques visibles immédiatement (test stack trace au lieu de `err opaque`).
- Suppression du *code mort* `*Safe`.
- Contrat d'API public inchangé pour les pré-conditions attendues.

### Negative / Trade-offs

- Tout consommateur qui dépendait du `recover()` global pour catcher les panics post-condition verra son programme crasher. Mais c'est précisément le comportement attendu — un bug doit interrompre la production, pas s'effacer en `error`.

### Risks and Mitigations

- **Risque** : régression performance par le check supplémentaire `bigfftPanic`. **Mitigation** : la branche `recover` est froide (seulement en cas de panic), donc impact ~0. Vérification par `benchstat`.

## Alternatives Considered

- **Supprimer totalement le `recover()`** : rejeté, les pré-conditions d'arité doivent rester recouvrables par les appelants.
- **Exposer les `*Safe` comme API publique** : rejeté, double API à maintenir, surface élargie sans demande utilisateur.

## References

- PRD : `Audit - PRD - FibGo.md` Epic E2
- Plan : `Audit - PRDPLan - FibGo.md` S2-T1, S2-T2
