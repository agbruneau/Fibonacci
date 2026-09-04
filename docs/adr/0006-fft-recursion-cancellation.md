# ADR-0006: Annulation de la récursion FFT — report au token par-appel (FFTContext)

- **Status**: Accepted
- **Date**: 2026-05-28
- **Context source**: audit mai 2026, constat `A2-03` (MAJEUR), axe 2 Concurrence (rapport archivé en historique git).

> **État courant (annoté le 2026-09-04 ; le `Status` ci-dessus et le corps sont
> conservés tels qu'ils ont été écrits le 2026-05-28)** : la décision de cet ADR
> est un **report** vers « la migration `FFTContext` exclusif ». Cette cible
> **n'existe plus** : le code a été supprimé de l'arbre le 2026-07-11 (commit
> `23ab593` ; `grep -rn FFTContext --include=*.go .` ne renvoie rien au
> 2026-09-04) et la migration reste classée WONT-FIX
> ([ADR-0004 §B1](0004-backlog-decisions.md)). Partout ci-dessous, lire
> « reporté à la migration `FFTContext` » comme **« sans échéance »** : la
> latence d'annulation non bornée pendant une multiplication d'opérande géant
> est le comportement courant et le reste par défaut. Le raisonnement du
> §Decision — pourquoi un drapeau global `atomic.Bool` est faux sous FFT
> concurrentes, et pourquoi la solution correcte est un token par appel — reste,
> lui, entièrement valide. Détail : *Status note (2026-09-04)*, en fin de
> document.

## Context

`fourierRecursiveUnified` (`internal/bigfft/fft_recursion.go:fourierRecursiveUnified`) et `fourierRecursiveCtx` ne reçoivent **aucun** `context.Context` et ne consultent jamais `ctx.Err()`. La boucle de doublement vérifie déjà le contexte **entre étapes** (`doubling_framework.go`) et **entre les 3 produits** d'un pas FFT (`fibonacci/fft.go executeFFTTransforms` : `ctx.Err()` avant/entre `op1/op2/op3`). Le trou résiduel : **une seule** multiplication FFT d'un opérande géant (N ≫ régime FFT) s'exécute jusqu'au bout sans consulter le contexte → **latence d'annulation non bornée** sur le chemin le plus coûteux.

> **Correctif (annoté le 2026-09-04, le §Context est conservé tel qu'il a été
> écrit)** : `fourierRecursiveCtx` **n'existe plus** — il vivait dans
> `fft_recursion_ctx.go`, supprimé avec l'API `FFTContext` le 2026-07-11. La
> seconde variante en place à HEAD est `fft_recursion.go:fourierRecursive`, qui
> ne reçoit pas non plus de `context.Context` : le constat de fond tient, seul le
> nom a changé. Déjà relevé dans la *Status note (2026-08-07)*.

L'audit qualifie A2-03 d'**amélioration de robustesse, pas de bug de correction** (pas de fuite de goroutine : `wg.Wait` ; pas de deadlock : admission `select`/`default`).

## Decision

**Reporter la granularité fine** (annulation à l'intérieur d'une multiplication FFT) à la migration **`FFTContext` exclusif** (ADR-0004 §B1, backlog). **Aucun changement de code** dans cette passe ; on documente l'invariant et on s'appuie sur l'annulation grossière existante (entre pas et entre les 3 produits).

> **Correctif (annoté le 2026-09-04, la décision est conservée telle qu'elle a
> été écrite)** : la cible du report n'existe plus. `FFTContext` a été supprimé
> de l'arbre le 2026-07-11 (commit `23ab593`) et la migration reste WONT-FIX
> ([ADR-0004 §B1](0004-backlog-decisions.md)), sans clause de réouverture autre
> qu'un *use case multi-tenant* concret. Le report **n'a donc plus d'échéance** :
> « aucun changement de code » n'est pas une étape avant un chantier à venir,
> c'est l'état courant et le défaut durable.

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

> **Correctif (annoté le 2026-09-04, les trois sections ci-dessus sont conservées
> telles qu'elles ont été écrites)** : « Accepté jusqu'à la migration
> `FFTContext` » (Negative), « La granularité fine arrivera avec `FFTContext` »
> (Risks) et « relève de la migration `FFTContext` » (Alternatives) renvoient
> toutes les trois à un chantier **supprimé de l'arbre** le 2026-07-11 (commit
> `23ab593`) et classé WONT-FIX. Aucune de ces trois échéances ne tient : la
> latence non bornée est le comportement courant, sans date de levée. Reprendre
> le chantier veut désormais dire **re-créer** le porteur de token — le code
> supprimé se récupère par
> `git show 23ab593^:internal/bigfft/context.go` — et non attendre une migration
> en cours.

## References

- Code concerné : `internal/bigfft/fft_recursion.go`, `internal/fibonacci/fft.go` (vérifs `ctx.Err()` existantes).
- Related ADR(s) : ADR-0003 (signatures hot path), ADR-0004 §B1 (migration `FFTContext`).
- Audit : axe 2 Concurrence, constat `A2-03` (rapport archivé en historique git).

## Status note (2026-08-07)

Précisions factuelles ; la décision (report) reste inchangée.

- `fourierRecursiveUnified` vit dans `internal/bigfft/fft_recursion.go`
  à HEAD (le range `93-169` que citait alors le Context avait dérivé ; le
  `99-201` noté ici le 2026-08-07 était décalé d'une ligne — corrigé le
  2026-08-07, et le Context est depuis ancré sur le symbole).
- `fourierRecursiveCtx` **n'existe plus** : il vivait dans
  `fft_recursion_ctx.go`, supprimé avec l'API `FFTContext` le 2026-07-11
  (addendum ADR-0004 §B1). La seule autre variante en place est
  `fourierRecursive` (`fft_recursion.go:fourierRecursive`), qui ne reçoit pas non plus de
  `context.Context` — le constat de fond du Context tient donc toujours.

## Status note (2026-09-04) — la cible du report n'existe plus dans l'arbre

Cet ADR **reporte** la granularité fine d'annulation « à la migration
`FFTContext` exclusif ». Cette cible n'existe plus, à deux titres :

1. Le code sur lequel elle reposait a été **supprimé** de l'arbre le 2026-07-11
   (commit `23ab593`, ~572 LOC prod + ~530 LOC tests, zéro appelant de
   production) — addendum ADR-0004 §B1.
2. La migration elle-même reste classée **WONT-FIX** par
   [ADR-0004 §B1](0004-backlog-decisions.md), sans clause de réouverture autre
   qu'un *use case multi-tenant* concret.

Conséquence à énoncer clairement plutôt que de la laisser implicite : le
« report » n'a plus d'échéance rattachée à un chantier vivant. La limitation
décrite en Negative / Trade-offs — **latence d'annulation non bornée pendant
une seule multiplication d'opérande géant** — est le comportement courant et le
restera par défaut. Les annulations grossières restent le seul mécanisme :
entre pas de doublement (`internal/fibonacci/doubling_framework.go`) et entre
les trois produits d'un pas (`internal/fibonacci/fft.go:executeFFTTransforms`).

Ce qui n'a **pas** changé : l'analyse du §Decision reste valide. Un drapeau
global `atomic.Bool` demeure faux sous FFT concurrentes (clear-race), et la
solution correcte demeure un **token d'annulation par appel**. Reprendre le
chantier veut désormais dire re-créer ce porteur de token — le code supprimé se
récupère par `git show 23ab593^:internal/bigfft/context.go` — et non attendre
une migration en cours.
