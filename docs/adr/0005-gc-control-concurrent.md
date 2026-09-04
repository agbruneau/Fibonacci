# ADR-0005: Contrôle GC concurrency-safe (refcount package-level)

- **Status**: Accepted
- **Date**: 2026-05-28
- **Context source**: audit mai 2026, constat `A2-01` (CRITIQUE), axe 2 Concurrence (rapport archivé en historique git).

## Context

`GCController` (`internal/fibonacci/memory/gc_control.go`) désactive le GC pendant un calcul (`debug.SetGCPercent(-1)`) et installe un garde-fou OOM (`debug.SetMemoryLimit`), puis restaure à la fin. Ces deux réglages sont **globaux au processus**.

En mode comparaison (`--algo all`, `N >= GCAutoThreshold = 1_000_000`), l'orchestrateur lance **plusieurs calculateurs en parallèle** (`internal/orchestration/orchestrator.go:ExecuteCalculations`, `errgroup.Go`). Chaque calculateur crée son propre `GCController` (`calculator.go:FibCalculator.CalculateWithObservers`) et l'exécute via `WithGC`. La sauvegarde/restauration **par contrôleur** était alors incorrecte :

1. Le 1ᵉʳ `Begin` capture l'original (ex. 100) et pose `-1`.
2. Le 2ᵉ `Begin` capture **`-1`** comme « original ».
3. Selon l'entrelacement, le GC peut **rester désactivé** après la fin du calcul (fuite de réglage à l'échelle du processus — précisément la classe de bug que `WithGC` prétend éliminer).
4. Le 1ᵉʳ `End` exécute `SetMemoryLimit(MaxInt64)`, **retirant le filet OOM** alors qu'un calculateur frère calcule encore un très grand nombre.

Vérifié par lecture de code (orchestrateur + calculateur + `gc_control.go`). L'aspect *data race* dynamique reste `[à vérifier]` sous `-race` (indisponible localement : `CGO_ENABLED=0`).

## Decision

**Sérialiser le contrôle GC par un refcount package-level** dans `internal/fibonacci/memory` (option (a) de l'audit), plutôt que de remonter le contrôle au niveau orchestration (option (b), qui disperserait la logique et casserait le contrat `WithGC`).

- Ajout de `var (gcGlobalMu sync.Mutex; gcActiveDepth int; gcSavedPercent int)`.
- `Begin()` : sous `gcGlobalMu`, **seul le premier entrant actif** (`gcActiveDepth == 0`) appelle `debug.SetGCPercent(-1)` (et mémorise le **vrai** original dans `gcSavedPercent`) puis installe la limite mémoire ; les suivants ne font qu'incrémenter `gcActiveDepth`.
- `End()` : sous `gcGlobalMu`, décrémente ; **seul le dernier sortant** (`gcActiveDepth == 0`) restaure `gcSavedPercent`, remet `SetMemoryLimit(MaxInt64)` et déclenche `runtime.GC()`.
- `gc.startStats` / `gc.endStats` restent **par instance**. Le champ `originalGCPercent` par instance est **supprimé** (la source de vérité de restauration est désormais `gcSavedPercent`).

L'invariant `WithGC` panic-safe (`defer End()`) est **préservé** : un panic entre `Begin` et `End` décrémente correctement le refcount via le defer.

## Consequences

### Positive

- En mode comparaison, le GC reste désactivé tant qu'**au moins un** calculateur tourne, et n'est restauré qu'une fois **tous** terminés. Le filet OOM (`SetMemoryLimit`) reste en place pendant toute la fenêtre concurrente.
- Le vrai `GOGC` d'origine est restauré exactement une fois (plus de capture erronée de `-1`).
- Pas de changement de contrat public ni de signature ; `WithGC` reste le chemin recommandé.

### Negative / Trade-offs

- L'invariant de `gc_control.go` est **étendu** au cas concurrent (auparavant non couvert par `TestGCController_WithGC_*`).
- La limite mémoire installée est celle calculée par le **premier** entrant (sur son `runtime.MemStats.Sys`) ; acceptable (ordre de grandeur identique entre calculateurs frères).

### Risks and Mitigations

- *Décompte déséquilibré (`End` sans `Begin`)* : garde `if gcActiveDepth > 0` avant décrément — pas d'underflow.
- *Confirmation data race* : à rejouer sous `CGO_ENABLED=1 go test -race -run 'TestGCController|TestExecuteCalculations' ./internal/fibonacci/memory/ ./internal/orchestration/` (Linux/WSL).

## Alternatives Considered

- **Option (b) — contrôle GC au niveau orchestration (une seule fois autour de toute la comparaison)** — rejetée : disperse la responsabilité hors de `memory`, casse le contrat `WithGC` par-calculateur documenté, et complique le chemin mono-calculateur.
- **`sync.Once`-like sans refcount** — rejetée : ne gère pas correctement la sortie (le dernier sortant doit restaurer, pas le premier).

## References

- Implementation : `internal/fibonacci/memory/gc_control.go` (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`, `Begin`/`End`).
- Test : `internal/fibonacci/memory/gc_control_test.go` (`TestGCController_ConcurrentBeginEnd_RestoresOriginal`).
- Related ADR(s) : ADR-0003 (atomics privés `bigfft`).
- Audit : axe 2 Concurrence, constat `A2-01` (rapport archivé en historique git).

## Status note (2026-06-10)

L'audit 2026-06 (commit `fa13bfd`) a introduit dans `internal/fibonacci` un
slot de cache state+arena « GC-immune »
(`FastDoublingCalculator.cachedState`, borné par `maxCachedArenaWords`). Sa
raison d'être est précisément l'interaction avec le présent ADR : le
`runtime.GC()` déclenché par le dernier `End()` vide les `sync.Pool`, si bien
que chaque appel répété re-allouait l'arena complète ; le slot, lui, survit à
ce GC forcé et conserve le state entre deux appels. Voir
`internal/fibonacci/fastdoubling.go` (commentaires
`cachedState`/`maxCachedArenaWords`) et
`internal/fibonacci/state_cache_test.go`.

## Status note (2026-08-07)

L'accesseur d'observabilité `GCController.Stats()` (et le type `GCStats`) a
été supprimé : sans appelant de production, c'était un stub mort. Les champs
`gc.startStats` / `gc.endStats` subsistent, désormais consommés uniquement par
les logs de `Begin()`/`End()`. La décision de sérialisation par refcount
ci-dessus est inchangée.

## Status note (2026-09-04) — la vérification `-race` différée est exécutée

Le Context (« l'aspect *data race* dynamique reste `[à vérifier]` sous `-race`
(indisponible localement : `CGO_ENABLED=0`) ») et la mitigation « *Confirmation
data race* » des Risks décrivaient une vérification jamais faite. Elle l'est.

La prémisse d'indisponibilité était fausse et non la plate-forme :
[ADR-0010 §D4](0010-audit-2026-09-decisions.md) a établi que `go test -race`
passe sur les **21 paquets** de cet hôte Windows (`CGO_ENABLED=1`, chaîne C
présente), et `scripts/check.ps1` sonde désormais les deux conditions pour
activer `-race`. La commande exacte que citait la mitigation a été rejouée le
2026-09-04 :

```
CGO_ENABLED=1 go test -race -count=1 \
  -run 'TestGCController|TestExecuteCalculations' \
  ./internal/fibonacci/memory/ ./internal/orchestration/
ok  github.com/agbruneau/FibGo/internal/fibonacci/memory   1.364s
ok  github.com/agbruneau/FibGo/internal/orchestration      1.398s
```

Aucune course rapportée. La décision reste inchangée ; seule la mention
« WSL/Linux requis » est caduque.
