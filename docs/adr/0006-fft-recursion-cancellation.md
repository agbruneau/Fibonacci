# ADR-0006: Annulation de la récursion FFT — report au token par-appel (FFTContext)

- **Status**: Accepted
- **Date**: 2026-05-28
- **Context source**: audit mai 2026, constat `A2-03` (MAJEUR), axe 2 Concurrence (rapport archivé en historique git).

## Context

`fourierRecursiveUnified` (`internal/bigfft/fft_recursion.go:93-169`) et `fourierRecursiveCtx` ne reçoivent **aucun** `context.Context` et ne consultent jamais `ctx.Err()`. La boucle de doublement vérifie déjà le contexte **entre étapes** (`doubling_framework.go`) et **entre les 3 produits** d'un pas FFT (`fibonacci/fft.go executeFFTTransforms` : `ctx.Err()` avant/entre `op1/op2/op3`). Le trou résiduel : **une seule** multiplication FFT d'un opérande géant (N ≫ régime FFT) s'exécute jusqu'au bout sans consulter le contexte → **latence d'annulation non bornée** sur le chemin le plus coûteux.

L'audit qualifie A2-03 d'**amélioration de robustesse, pas de bug de correction** (pas de fuite de goroutine : `wg.Wait` ; pas de deadlock : admission `select`/`default`).

## Decision

**Reporter la granularité fine** (annulation à l'intérieur d'une multiplication FFT) à la migration **`FFTContext` exclusif** (ADR-0004 §B1, backlog). **Aucun changement de code** dans cette passe ; on documente l'invariant et on s'appuie sur l'annulation grossière existante (entre pas et entre les 3 produits).

### Pourquoi pas un drapeau global `atomic.Bool` (approche initialement proposée) ?

Un `var fftCancelled atomic.Bool` package-level, armé par un watcher du `ctx` au point d'entrée et consulté en tête de récursion, **introduit une data race logique** sous FFT concurrentes :

- `executeParallel3` lance **3 multiplications FFT simultanées** ; `--algo all` lance plusieurs calculateurs.
- Le watcher pose le drapeau sur `ctx.Done()`, et chaque appel doit le **remettre à false** en sortie (sinon il reste armé et casse tout calcul ultérieur).
- **Clear-race** : le calculateur/op A termine, `ClearFFTCancellation()` remet `false`, **pendant** que l'op B (toujours sous un `ctx` annulé) compte sur le drapeau pour s'arrêter → B voit `false` et **ne s'annule pas**. Le watcher de B a déjà tiré (une seule fois) et ne ré-arme pas.

Le `set` est correct (errgroup.WithContext annule tous les frères ensemble), mais le `clear` par-appel ne l'est pas. Un drapeau **binaire global** ne peut pas porter un état d'annulation **scopé par appel** sous concurrence. La solution correcte est un **token d'annulation par-appel** (un `*atomic.Bool` porté par un `FFTContext` threadé), ce qui revient précisément au refactor `FFTContext`.

Changer la signature de `fourierRecursive*` pour y passer un `context.Context` contredit par ailleurs l'invariant de stabilité des signatures hot path (esprit ADR-0003) et ajoute un `ctx.Err()` (verrou) par nœud de récursion.

## Consequences

### Positive

- Pas de régression de correction ni de concurrence introduite (le drapeau global aurait été subtilement bogué).
- L'annulation **grossière** existante reste fonctionnelle : `Ctrl-C`/timeout interrompt entre les pas de doublement et entre les 3 produits FFT — borne déjà la latence à « une multiplication ».

### Negative / Trade-offs

- La latence d'annulation pendant **une** multiplication d'opérande géant reste non bornée (plusieurs secondes à minutes aux plus grands N). Accepté jusqu'à la migration `FFTContext`.

### Risks and Mitigations

- *Attente perçue à l'annulation aux très grands N* : documenté ; mitigé par les vérifications grossières existantes. La granularité fine arrivera avec `FFTContext` (token par-appel sûr).

## Alternatives Considered

- **Drapeau global `atomic.Bool` + watcher** — rejeté : clear-race sous FFT concurrentes (cf. ci-dessus).
- **`context.Context` threadé dans `fourierRecursive*`** — rejeté : casse la stabilité de signature hot path (ADR-0003) et ajoute un coût de verrou par nœud.
- **Token `*atomic.Bool` par-appel via `FFTContext`** — **retenu comme cible**, mais relève de la migration `FFTContext` (ADR-0004 §B1), hors scope de la remédiation d'audit courante.

## References

- Code concerné : `internal/bigfft/fft_recursion.go`, `internal/fibonacci/fft.go` (vérifs `ctx.Err()` existantes).
- Related ADR(s) : ADR-0003 (signatures hot path), ADR-0004 §B1 (migration `FFTContext`).
- Audit : axe 2 Concurrence, constat `A2-03` (rapport archivé en historique git).
