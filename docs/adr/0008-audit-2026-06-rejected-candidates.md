# ADR-0008: Audit de refactorisation 2026-06 — candidats rejetés après vérification

- **Status**: Accepted
- **Date**: 2026-06-09
- **Context source**: audit de refactorisation exhaustif (branche
  `refactor/audit-deepening-2026-06`), exploration multi-agents +
  vérification par grep/tests avant chaque modification.

## Context

L'audit a identifié une liste de « deepening opportunities » (modules
shallow, exports morts, seams à adapter unique). Plusieurs candidats,
plausibles sur papier, ont été **invalidés par la vérification** : les
supprimer aurait cassé des consommateurs réels ou détruit des contrats de
test. Cet ADR les matérialise pour que les audits futurs ne les
re-suggèrent pas sans nouvel élément.

## Decision

### R1 — `errors.ColorProvider` : interface conservée

L'interface 2 méthodes (`Yellow()`, `Reset()`) de
`internal/errors/handler.go` est shallow au sens strict (profondeur
quasi nulle, deux adapters pass-through). Elle est néanmoins
**conservée** : son rôle n'est pas la profondeur mais la **rupture du
cycle d'import** cli ↔ errors, gardée par `TestArchitectureLayering`
(`errors ↛ format`, et cli dépend d'errors pour les codes de sortie).
L'alternative évaluée — état package-level mutable
(`errors.SetColors(...)`) — remplace une interface explicite par un
global écrit au démarrage et lu partout : perte d'injectabilité dans les
tests, ordre d'initialisation implicite, et le même nombre de lignes.
Rejet motivé : l'interface est le moindre mal et le coût de maintenance
observé est nul.

À revoir si : la couche erreurs doit exposer plus de 2-3 capacités de
style (l'interface grossirait alors au lieu de rester un seam minimal).

### R2 — `executeTasks` / `executeMixedTasks` : conservés

Le candidat « branche parallèle quasi morte, inliner dans
`executeParallel3` » est **factuellement faux** : les deux helpers sont
le moteur de dispatch des chemins matriciels (`matrix_ops.go` —
produit 2×2 complet, carré symétrique mixte, Strassen), branches
séquentielle ET parallèle. Le deletion test échoue : la suppression
forcerait trois réimplémentations. Le générique `[T, PT]` est le prix de
deux types de tâches partageant un seul chemin d'exécution sous le même
sémaphore global (`globalSem`, P2-02).

### R3 — Seam `CacheStrategy` : conservé (et approfondi)

Le candidat « interface à adapter unique, pur pass-through » ignorait le
second adapter : `mockCacheStrategy` (doubling_framework_test.go) qui
épingle la discipline d'appel de la boucle (cadence par itération,
gating DTM, propagation d'erreur). Deux adapters = seam réel. Le manque
réel était la testabilité des heuristiques grow/shrink, résolu par
extraction de la fonction pure `decideCacheTuning` (commit dédié) sans
toucher au seam.

### R4 — Observers `progress` (`LoggingObserver`, `NoOpObserver`) : conservés

Leur rétention comme « extension surface used by tests and embedders »
est une décision déjà documentée (A-19, `internal/progress/doc.go`).
Aucune friction nouvelle ne justifie de la rouvrir.

### R5 — Exports `bigfft` (`GetFFTParams`, `PolyFromInt`, `ValueSize`) : conservés

Signalés « tests-only » par l'exploration ; le grep prouve le contraire :
`internal/fibonacci/fft.go` les consomme en production (construction du
pipeline FFT du fast doubling). Ce sont des points du seam
fibonacci → bigfft, pas des exports morts.

### R6 — `fibonacci.TestFactory` / `MockCalculator` : conservés en l'état

Consommés hors package par `internal/app/app_test.go`. Le déménagement
vers `fibonaccitest` créerait un cycle d'import pour les tests
in-package de fibonacci (`registry_test.go`, `testing_test.go`) — Go
interdit `package fibonacci` (test) → `fibonaccitest` → `fibonacci`.
Le gain (surface de prod plus propre) ne paie pas la conversion des
tests in-package en package externe.

### R7 — Knobs de tuning `threshold` : pas de migration atomic ni de garde `Once`

L'invariant A2-04 (commentaire « INVARIANT (A2-04) » au-dessus du bloc `var` de tuning dans `manager.go`) acte explicitement « no atomic
migration this pass ». Une garde `sync.Once` dans `SetTuning` casserait
`TestSetTuning` (F-007) qui appelle légitimement la fonction plusieurs
fois en séquentiel. La réponse retenue est ailleurs : **exécuter le
câblage documenté** (appel unique de `SetTuning` depuis `app.New`,
derrière un `sync.Once` côté app) — commit `fix(app)` dédié.

## Consequences

- Les sept candidats ci-dessus sont fermés avec leur preuve ; un futur
  audit doit apporter un élément nouveau (consommateur disparu, friction
  mesurée) pour les rouvrir.
- Le reste de l'audit (A-05, ADR-0002 contexte, hygiène d'exports,
  extraction `decideCacheTuning`, câblage A2-04, tests) est livré dans
  les commits de la branche `refactor/audit-deepening-2026-06`.

## References

- ADR-0004 (backlog), ADR-0002 (recover), A-19 (`progress/doc.go`),
  A2-04 (commentaire « INVARIANT (A2-04) » de `threshold/manager.go`).
- Commits de la branche `refactor/audit-deepening-2026-06`.
