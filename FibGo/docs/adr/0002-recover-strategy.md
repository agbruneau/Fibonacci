# ADR-0002: Stratégie `recover()` dans `bigfft`

- **Status**: Accepted
- **Date**: 2026-05-21
- **Context source**: hardening sprint mai 2026 (commit `202a02c`).

## Context

`internal/bigfft/fft.go` installe quatre `recover()` indistincts dans les fonctions publiques `Mul`, `MulTo`, `Sqr`, `SqrTo`. Ces `recover` capturent *toute* panic, y compris les violations de post-conditions algorithmiques internes émises par `internal/bigfft/fermat.go` (`panic("len(z) > 2n+1")`, `panic("unexpected carry after normalization")`).

Conséquence : un bug authentique dans la réduction modulaire de Fermat ressort comme un `error` opaque indistinguable d'une erreur d'arité, et passe silencieusement tout test qui n'inspecte que `err != nil`.

L'intention initiale (visible dans les wrappers `*Safe` de `bigfft/fermat.go` — `MulSafe`, `SqrSafe`, `ShiftSafe`, `AddSafe`, `SubSafe`) distinguait théoriquement :
- **pré-conditions externes** (arité, taille) → à transformer en `error` retourné.
- **post-conditions internes** (invariants algorithmiques) → à laisser propager (panic).

Mais aucun consommateur de production n'utilise les wrappers `*Safe`.

## Decision

**Restreindre le `recover()` global à un set d'erreurs sentinelles** plutôt que de capturer toute panic.

1. Introduire la *map* sentinelle `fermatPostConditionPanics` listant les panics post-condition (`"len(z) > 2n+1"`, `"fermat.Mul: unexpected carry after normalization"`, `"fermat.Sqr: unexpected carry after normalization"`).
2. Le helper `isFermatPostConditionPanic(r any) bool` classifie la valeur récupérée.
3. Le `recover()` global des quatre entry-points (`Mul`, `MulTo`, `Sqr`, `SqrTo`) **re-propage** par `panic(r)` lorsque la sentinelle correspond ; sinon il convertit en `error` comme avant (préserve l'API publique sur les pré-conditions d'arité).
4. Les `panic` internes de `fermat.go` (post-conditions) sont conservés tels quels.
5. Les wrappers `*Safe` (`MulSafe`, `SqrSafe`, `ShiftSafe`, `AddSafe`, `SubSafe`) sont **conservés** : contrairement à la lecture initiale de l'audit, ils sont **testés** par `fermat_test.go` (la suite documente le contrat de pré-condition) ; les supprimer perdrait cette couverture explicite. Marquer ADR à reviser si une décision ultérieure de publication d'API est prise.

## Consequences

### Positive

- Régressions algorithmiques visibles immédiatement (test stack trace au lieu de `err opaque`).
- Couverture explicite des pré-conditions conservée via les wrappers `*Safe` **testés** (`fermat_test.go`, `misc_extra_test.go`) — cohérent avec la Décision §5 qui les conserve.
- Contrat d'API public inchangé pour les pré-conditions attendues.

### Negative / Trade-offs

- Tout consommateur qui dépendait du `recover()` global pour catcher les panics post-condition verra son programme crasher. Mais c'est précisément le comportement attendu — un bug doit interrompre la production, pas s'effacer en `error`.

### Risks and Mitigations

- **Risque** : régression performance par le check supplémentaire `isFermatPostConditionPanic` (lookup dans `fermatPostConditionPanics`). **Mitigation** : la branche `recover` est froide (seulement en cas de panic), donc impact ~0. Vérification par `benchstat`.

## Alternatives Considered

- **Supprimer totalement le `recover()`** : rejeté, les pré-conditions d'arité doivent rester recouvrables par les appelants.
- **Exposer les `*Safe` comme API publique** : rejeté, double API à maintenir, surface élargie sans demande utilisateur.

## References

- Implementation : `internal/bigfft/fft.go` (sentinel `isFermatPostConditionPanic`)
- Tests : `internal/bigfft/fft_recover_policy_test.go`, `internal/bigfft/fermat_panic_test.go`

## Status note (2026-06-10)

Les deux `file:line` cités en Context ont dérivé deux fois (2026-06-10, puis
2026-08-07) sans que le code change. Ils sont remplacés par des ancres de
symbole, et cette note ne conserve que ce qui est réellement un fait :

- Les quatre entry-points `Mul`/`MulTo`/`Sqr`/`SqrTo` de
  `internal/bigfft/fft.go` portent chacun leur `defer`/`recover()` ; le bloc
  va de `func Mul` à l'accolade fermante de `SqrTo`.
- La politique sentinelle décrite ici reste en place
  (`internal/bigfft/fft.go:isFermatPostConditionPanic`, appelée depuis
  `internal/bigfft/fft.go:fermatPanicToError`), gardée par
  `TestFermatPostConditionPanicClassifier`.
- Le décompte « 48 LOC » de wrappers `*Safe` était faux : de la première ligne
  de commentaire de `MulSafe` à l'accolade fermante de `SubSafe`,
  `internal/bigfft/fermat.go` en contient **53 lignes dont 37 de code**
  (recompté le 2026-08-07). Le constat de fond est inchangé : aucun
  consommateur de production ne les appelle.
