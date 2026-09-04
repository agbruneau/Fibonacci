# Design Patterns — FibGo

**Inventaire faisant foi.** C'est ici — et nulle part ailleurs — qu'est tenue la liste
des patterns en usage dans FibGo, avec pour chacun sa raison d'être et son site
d'implémentation. [`docs/ARCH.md` §5](../../ARCH.md#5-design-patterns) y renvoie et n'en
garde pas de copie : jusqu'au 2026-09-04 les deux fichiers tenaient chacun leur table
(14 entrées d'un côté, 11 de l'autre, ensembles différents, aucune mention réciproque) ;
la table ci-dessous est leur union réconciliée contre la source.

Compagnon : la figure [`interface-hierarchy.md`](./interface-hierarchy.md) pour les
interfaces mises en jeu. Registre canonique des décisions : [`docs/adr/`](../../adr/)
(`docs/ARCH.md` §14 est un journal narratif distinct, avec sa propre numérotation à
trois chiffres — il ne l'indexe pas, il y renvoie).

Les sites sont donnés par fichier et par symbole, sans numéro de ligne : ceux-ci dérivent
à chaque édition, alors qu'un `grep` du symbole reste vrai.

## Inventaire — 17 patterns

| Pattern | Pourquoi il existe | Site d'implémentation |
|---|---|---|
| **Decorator** | Ajoute le transversal — chemin rapide N ≤ 93, adaptation des observateurs, contrôle GC, configuration du cache FFT, préchauffage des pools — sans toucher aux cœurs d'algorithme | `FibCalculator` enveloppant `CoreCalculator`, `internal/fibonacci/calculator.go` |
| **Strategy** | Permet d'échanger la politique de multiplication selon la charge ou l'intention de mesure | `Multiplier` / `DoublingStepExecutor` avec `AdaptiveStrategy` et `FFTOnlyStrategy`, `internal/fibonacci/strategy.go` ; calculateurs dans `fastdoubling.go`, `matrix.go`, `fft_based.go` |
| **Interface Segregation (ISP)** | Un consommateur qui n'a besoin que de multiplier/élever au carré dépend de l'interface étroite ; seuls les cadres de boucle prennent la large | `Multiplier` (étroite : `Multiply`/`Square`) vs `DoublingStepExecutor` (large : `+ExecuteStep`), `internal/fibonacci/strategy.go` |
| **Factory + Registry** | Enregistrement, recherche et mise en cache centralisés des calculateurs, avec création paresseuse et double vérification sous verrou | `DefaultFactory` implémentant `CalculatorFactory`, `internal/fibonacci/registry.go` |
| **Observer** | Découple la production de progression de ses consommateurs (UI, journal) et en admet plusieurs à la fois | `progress.ProgressSubject` + implémentations de `ProgressObserver`, `internal/progress/observer.go`, `observers.go` |
| **Framework (Template Method)** | Le cadre possède la boucle (itération des bits, rapport de progression, vérifications de contexte) et délègue l'opération à une stratégie enfichable | `DoublingFramework` (`internal/fibonacci/doubling_framework.go`), `MatrixFramework` (`matrix_framework.go`) |
| **Facade** | `app.Application` masque l'analyse des drapeaux, le dispatch de mode et la traduction erreur → code de sortie | `internal/app/app.go`, type `Application` et méthode `Run` |
| **Adapter** | Deux usages : une fonction promue en `ProgressReporter` sans type dédié ; les sondes hôte `gopsutil` converties en message Bubble Tea | `orchestration.ProgressReporterFunc` (`internal/orchestration/interfaces.go`) enveloppant `cli.DisplayProgress` ; `sampleSysStatsCmd` (`internal/tui/commands.go`, replié depuis l'ancien `internal/metrics/system`) |
| **Object Pool** | Réduit les allocations et la pression GC sur les chemins chauds | `sync.Pool` d'état Fibonacci (`internal/fibonacci/fastdoubling.go`, `AcquireStateForN`/`ReleaseStateWithResult`) et pools par classe de taille de `internal/bigfft/pool.go`. Plafonné par `MaxPooledBitLen` = 50 000 000 bits (`internal/fibonacci/common.go`) : au-delà, l'objet est jeté au lieu d'être recyclé |
| **Arena Allocator** | Pré-dimensionne un bloc contigu pour les tableaux de support des `big.Int` d'état, ce qui réduit la fragmentation et le suivi GC par tampon | `memory.CalculationArena`, `internal/fibonacci/memory/arena.go` (`PreSizeFromArena`) |
| **Bump Allocator** | Allocations temporaires par lot avec remise à zéro en O(1) pour les internes FFT | `bigfft.BumpAllocator`, `internal/bigfft/bump.go` |
| **Cache (LRU) — transformées FFT** | Réutilise une transformée directe (`PolValues`) d'une opération à l'autre. Borné en entrées **et** en octets (audit M-08) | `internal/bigfft/fft_cache.go`, type `TransformCache`. ⚠ Consulté uniquement depuis `Mul`/`MulTo`/`Sqr`/`SqrTo` — **aucune boucle de doublement ne le lit** (`executeDoublingStepFFT` appelle `TransformWithBump`) ; voir la note `TransformCache` du [component-diagram](../component-diagram.md) |
| **Circuit Breaker (léger)** | Sort avant l'OOM plutôt que pendant : le besoin est estimé et comparé au budget avant tout calcul | Estimation `memory.EstimateMemoryUsage` (`internal/fibonacci/memory/budget.go`) ; la sortie effective est `Application.validateMemoryBudget` (`internal/app/calculate.go`), qui rend `ExitErrorConfig` |
| **Dynamic Threshold Adjustment** | Enregistre le temps par itération et corrige les seuils FFT/parallélisme en cours de calcul | `threshold.DynamicThresholdManager`, `internal/fibonacci/threshold/manager.go`. **Optionnel et inactif par défaut** : `--dynamic-thresholds` (câblé par l'audit 2026-09, M-04 ; mesuré neutre sur CPU — voir [ADR-0001](../../adr/0001-dtm-decision.md)) |
| **Zero-Copy Result Return** | « Vole » `res.a` de l'état matriciel au lieu de le copier | `MatrixFramework.ExecuteMatrixLoop` **seulement**, `internal/fibonacci/matrix_framework.go`. Délibérément **pas** fait dans `DoublingFramework.ExecuteDoublingLoop` (P1-04) : son état alias l'arène, donc le chemin de succès copie en profondeur via `ReleaseStateWithResult` |
| **Generics with Pointer Constraints** | Une seule exécution de tâches pour les multiplications et les mises au carré, sans duplication | `executeTasks[T any, PT interface{*T; task}]`, `internal/fibonacci/common.go` |
| **GC Controller** | Coupe le GC pendant les gros calculs (N ≥ 1M en mode `auto`) et le restaure ensuite ; `debug.SetMemoryLimit` sert de filet OOM | `memory.GCController`, `internal/fibonacci/memory/gc_control.go` (`WithGC`, `Begin`, `End`) |

## Mécanismes d'ingénierie — 5

Pas des patterns de conception au sens catalogue, mais des choix récurrents qu'un lecteur
rencontre partout dans le code.

| Mécanisme | Détail | Site |
|---|---|---|
| **Heuristiques de seuils configurables à l'exécution** | Estimation adaptative dérivée du nombre de cœurs, de l'architecture et de la classe SIMD | `internal/config/thresholds.go`, `hardware.go` |
| **Agrégation de progression par canal bufferisé** | Tampon = `numCalcs × ProgressBufferMultiplier`, avec `ProgressBufferMultiplier = 5` | `internal/orchestration/orchestrator.go` |
| **Instantané d'observateurs sans verrou** | `Freeze()` copie la tranche d'observateurs une fois et rend une fermeture `ProgressCallback` : plus aucune prise de verrou dans la boucle chaude | `ProgressSubject.Freeze`, `internal/progress/observer.go` |
| **Limitation de concurrence par sémaphore** | **Deux** sémaphores distincts, dimensionnés différemment : celui des tâches Fibonacci vaut `runtime.GOMAXPROCS(0)`, celui de la récursion FFT vaut `runtime.NumCPU()` | `getTaskSemaphore` (`internal/fibonacci/common.go`), `getSemaphore` (`internal/bigfft/fft_recursion.go`) |
| **Options fonctionnelles** | Construction de l'`Application` par options, dont l'injection d'une fabrique de calculateurs pour les tests | `AppOption`, `WithFactory`, `internal/app/app.go` |

## Contrats transversaux

- **Concurrence** : les goroutines ont toujours un cycle de vie borné ; la propagation
  d'erreur passe par `errgroup`, ou par la structure `parallel3Result` sans allocation sur
  le chemin chaud de fastdoubling (`internal/fibonacci/common.go`).
- **Propriété des ressources** : toute acquisition d'état est appariée à une libération
  dans la même portée (`AcquireStateForN`/`ReleaseStateWithResult`, ou les variantes par
  calculateur `acquireStateForN`/`releaseStateWithResult`). Depuis le commit `fa13bfd`, le
  puits de libération est soit le `sync.Pool` partagé, soit un créneau de cache immunisé
  au GC propre au calculateur (`FastDoublingCalculator.cachedState`, borné par
  `maxCachedArenaWords`) qui retient l'état d'un appel à l'autre. Une arène bump doit
  appeler `Reset` avant réemploi. Quand un état mis en cache ou en pool possède une arène,
  chaque emplacement `*big.Int` doit être détaché (`s.FK = new(big.Int)` et suivants) avant
  que l'état n'atteigne l'un ou l'autre puits — sinon l'arène aliaserait des données que le
  prochain occupant écrase (`clearStateAliases`, appelé sans condition par
  `finalizeStateReleaseTo` dans `internal/fibonacci/fastdoubling.go`).

## Notes de maintenance

- Ce fichier documente le code *actuel* ; ce n'est pas un catalogue d'intentions. Un
  pattern ajouté se déclare ici, avec son site.
- Un pattern retiré du code se retire d'ici. Il n'existe pas de seconde table à
  synchroniser : [`ARCH.md` §5](../../ARCH.md#5-design-patterns) ne cite que les cinq
  patterns dont sa propre narration dépend, et pointe vers ce fichier pour le reste.
- Pour les relations entre paquets, voir [`dependency-graph.md`](../dependency-graph.md) ;
  pour les interfaces, [`interface-hierarchy.md`](./interface-hierarchy.md).

---
[← Retour au hub architecture](../README.md) · Légende narrative : [`ARCH.md` §5](../../ARCH.md#5-design-patterns)
