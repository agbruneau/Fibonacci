# 42 — Audit concurrence

Audit du modele de concurrence FibGo : goroutines, errgroup, semaphores,
sync.Pool, primitives mutex/atomic, channels, race detector, cycle de vie.

Perimetre : code de production uniquement (`*.go` hors `_test.go`).

## Goroutines en production

| Localisation | Goroutine | Mecanisme d'arret | Risque fuite |
|---|---|---|---|
| `internal/orchestration/orchestrator.go:43` | `cfg.ProgressReporter.DisplayProgress` (display) | `close(progressChan)` puis `displayWg.Wait()` (l.81-82) | Faible — close systematique apres `g.Wait()` |
| `internal/orchestration/orchestrator.go:57` | `g.Go(...)` calculator worker (errgroup) | `errgroup` ctx + `g.Wait()` (l.78) | Faible — ctx propage l'annulation |
| `internal/fibonacci/common.go:186-188` | `runParallel3Op` (3 ops paralleles) | `wg.Wait()` (l.190) ; semaphore borne | Faible |
| `internal/fibonacci/common.go:262` | `executeTasks[T,PT]` worker | `wg.Wait()` (l.279) ; ctx checke apres acquisition sem | Faible |
| `internal/fibonacci/common.go:321,338` | `executeMixedTasks` workers (sqr/mul) | `wg.Wait()` (l.353) ; ctx checke | Faible |
| `internal/bigfft/fft_recursion.go:113` | Branche FFT parallele | `wg.Wait()` (l.131) ; release sem en defer | Faible — token toujours libere en defer |
| `internal/calibration/calibration.go:160` | `progressDisplay` | `close(progressChan)` + `wg.Wait()` (l.165-166, 178-179, 190-191) | Faible — close present sur les 3 chemins de sortie |
| `internal/calibration/microbench.go:140` | Workers benchmark | `wg.Wait()` (l.164) ; semaphore + `<-ctx.Done()` | Faible |

Total : 9 sites de spawn en production. Tous sont bornes par WaitGroup ou
errgroup et propagent le contexte.

## errgroup

| Site | Wait() OK ? | Notes |
|---|---|---|
| `internal/orchestration/orchestrator.go:54-78` | OUI : `_ = g.Wait()` | Le retour est volontairement ignore — chaque erreur est deja stockee dans `results[idx].Err` (l.62). `Wait()` est conserve pour synchroniser avant `close(progressChan)`. Note : la propagation `return err` (l.70) declenche bien la cancellation cross-calculator via le ctx partage. **Non bloquant** — le commentaire l.73-77 documente l'intention. |

Aucun autre `errgroup` en production (tests : `fibonacci_edge_test.go` x2).

## Semaphores

| Site | Borne | Adaptatif NumCPU |
|---|---|---|
| `internal/fibonacci/common.go:51` (`globalSem`) | `runtime.NumCPU()` | OUI — lazy init via `sync.Once` apres ajustement GOMAXPROCS |
| `internal/bigfft/fft_recursion.go:20` (`concurrencySemaphore`) | `runtime.NumCPU()` | OUI — meme pattern lazy/Once |
| `internal/calibration/microbench.go:136` | `runtime.NumCPU()` | OUI |

3 semaphores distincts, tous adaptatifs. Note doc fibonacci/common.go:30-36 :
le decouplage volontaire entre `globalSem` (fibonacci) et le semaphore
bigfft evite une inversion de couches (bigfft ne doit pas importer
fibonacci). **Risque** : oversubscription possible si les deux niveaux
saturent simultanement (NumCPU + NumCPU goroutines actives).

## sync.Pool

| Pool | Type | Get/Put equilibres |
|---|---|---|
| `internal/fibonacci/fastdoubling.go:220` (`statePool`) | `*CalculationState` | OUI — `AcquireState` / `ReleaseState` (defer recommande dans la doc l.232-238) ; rejet objets oversize via `checkLimit` |
| `internal/fibonacci/matrix_types.go:98` (`matrixStatePool`) | `*matrixState` | OUI — pattern similaire |
| `internal/bigfft/bump.go:34` (`bumpAllocatorPool`) | `*BumpAllocator` | OUI — `Acquire`/`Release` ; non thread-safe par instance (instance par goroutine) |
| `internal/bigfft/pool.go:18,132,223,311,409` | shards `[]big.Word`, `fermat`, `[]nat`, `[]fermat`, `*fftState` | OUI (sharding par classe de taille) |

Risque : aucun Get/Put non equilibre detecte. La complexite de `pool.go`
(5 arrays de pools indexes par taille) merite une revue dediee dans un
audit perf, mais le contrat Get/Put est respecte.

## Mutex / RWMutex / atomic

| Site | Type primitive | Justification |
|---|---|---|
| `internal/progress/observer.go:37` | `sync.RWMutex` | Notify(read) >> Register(write) — RWMutex justifie |
| `internal/progress/observers.go:71` (`LoggingObserver`) | `sync.Mutex` | Map ecrit a chaque Update — Mutex correct |
| `internal/tui/bridge.go:20` (`programRef`) | `sync.RWMutex` | Send(read) >> SetProgram(write, 1x) — RWMutex justifie |
| `internal/fibonacci/registry.go:42` | `sync.RWMutex` | Get(read) >> Register(write) — OK |
| `internal/fibonacci/threshold/manager.go:41` | `sync.RWMutex` | Lectures frequentes des seuils — OK |
| `internal/ui/themes.go:100` | `sync.RWMutex` | Lecture theme >> mutation — OK |
| `internal/bigfft/fft_cache.go:58` | `sync.RWMutex` | LRU mute la liste a chaque acces — RWMutex potentiellement mal optimise (cf. notes ci-dessous) |
| `internal/bigfft/fft_cache.go:62-65` | `atomic.Uint64` x4 (hits/misses/evictions/accesses) | OK — compteurs lock-free |
| `internal/bigfft/pool_warming.go:104` | `atomic.Bool` (`poolsWarmed`) | OK — CAS pour init unique |
| `internal/fibonacci/matrix_ops.go:14` | `atomic.Int32` (Strassen threshold) | OK — config tuneable lock-free |

**Observation** : dans `fft_cache.go`, l'usage `RWMutex` est probablement
sub-optimal car la mise a jour LRU (`MoveToFront`) requiert un Lock
exclusif a chaque cache hit. Un `Mutex` simple ou un design lock-free
type "CLOCK" pourrait etre evalue (audit perf, hors perimetre 5.3).

## Channels

| Site | Pattern | Risque |
|---|---|---|
| `progressChan` (orchestrator.go:39, calibration.go:158) | Fan-in producteurs -> 1 consommateur (display) | Faible — buffer dimensionne (`*ProgressBufferMultiplier`), close systematique |
| `ChannelObserver.channel` (observers.go:55) | Send non bloquant (`select default`) | **Faible** — drop sur saturation, pas de fuite |
| `globalSem` / `concurrencySemaphore` | Token bucket | Faible — release en `defer` dans tous les chemins observes |

Aucun channel sans close detecte sur les chemins critiques.

## Race detector

- Active en CI : **OUI**, cf. `Makefile:162` :
  `$(GO) test -v -race -cover ./...`
- Couverture estimee : la cible `make test` couvre `./...` donc tous
  les packages. `make test-short` n'inclut pas `-race`. Tests
  specifiquement dedies aux conditions de course presents :
  `internal/fibonacci/fft_race_test.go`, `state_pool_test.go`,
  `internal/bigfft/fft_cache_test.go`, `internal/fibonacci/threshold/manager_test.go`.

## Cycle de vie des goroutines longues

- **TUI (`tea.NewProgram`)** : `model.go:331-335`. Lance via `p.Run()` qui
  bloque jusqu'a `tea.Quit`. `defer model.cancel()` (l.329) garantit
  l'annulation du ctx propage vers les workers d'orchestration. Bridge
  goroutines (DisplayProgress) terminent par close du progressChan en
  fin de calcul.
- **sysmon** : pas de goroutine longue — `Sample()` est synchrone
  (`internal/sysmon/sysmon.go`). Les snapshots sont declenches a la
  demande par le model TUI via `tea.Tick`.
- **Progress observer** : aucune goroutine de fond — Notify est
  synchrone, le `Freeze` (observer.go:135) libere le RLock avant
  iteration. Robuste aux panics observers via `recover()` (l.144).

## Synthese

- **Findings critiques : 0**
- **Findings non bloquants : 2**
  1. `RWMutex` du `TransformCache` (fft_cache.go:58) probablement
     sub-optimal vu que LRU mute a chaque hit — a evaluer en audit perf.
  2. Deux semaphores independants (`fibonacci.globalSem` +
     `bigfft.concurrencySemaphore`) peuvent oversubscriber jusqu'a
     `2*NumCPU` goroutines actives lorsque les deux niveaux saturent.
     Decouplage justifie par layering (bigfft ne doit pas importer
     fibonacci) et documente dans le code, mais l'effet cumulatif
     merite une mesure.

### Top 5 risques concurrence (ordonne par severite decroissante)

1. **Oversubscription cross-package** (fibonacci+bigfft) — 2 semaphores
   independants peuvent depasser NumCPU agrege.
2. **`_ = g.Wait()` dans orchestrator.go:78** — Audit 3.1 l'avait
   signale ; verifie ici **non bloquant** (errors stockees dans
   results[idx].Err, Wait() retenu pour synchro avant close).
3. **TransformCache LRU sous RWMutex** — write-heavy malgre RWMutex.
4. **`make test-short` sans `-race`** — bypass possible du detector
   pour les contributeurs presses ; documenter ou aligner.
5. **`ChannelObserver` drop silencieux** — sends en `select default`
   ; comportement intentionnel, mais une UI lente perd des updates.
   A documenter cote API si pas deja fait.

### Conformite directive `CLAUDE.md` §4

- `sync.Pool` : OUI (5 pools shardes + 3 pools d'objets metier).
- `errgroup` : OUI (1 site, usage correct).
- Semaphores bornes : OUI (3 sites, tous `NumCPU`-adaptatifs).
- Goroutines avec controle de cycle de vie : OUI (9/9 sites bornes
  par WaitGroup ou errgroup).

**Verdict global : conforme aux directives, aucun bug de concurrence
detecte. Findings essentiellement de tuning perf, hors perimetre 5.3.**
