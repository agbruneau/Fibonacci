# Audit — Axe 2 : Concurrence & data races

## 1. Verdict d'axe

La machinerie de concurrence du hot path (sémaphores bornés, `errgroup`, `sync.Pool`, seuils `atomic`, cache LRU `RWMutex`) est globalement saine et `go vet ./...` (incluant `copylocks`, `atomic`, `loopclosure`, `lostcancel`) ne signale **aucun** diagnostic. Le défaut le plus sérieux n'est pas dans les algorithmes mais dans le `GCController` : en mode comparaison (`--algo all`), plusieurs calculateurs s'exécutent en parallèle via `errgroup` et appellent chacun `debug.SetGCPercent`/`debug.SetMemoryLimit` — un état **global au processus** — ce qui produit une course sur ces réglages et une restauration de GC erronée (`-1` capturé comme « original »). Les autres constats sont des écritures de globaux non synchronisées (`SetCacheLogger`, `SetTuning`) et l'absence de propagation du `context` dans la récursion FFT parallèle. Aucune `data race` n'a pu être confirmée dynamiquement : `go test -race` est indisponible sur l'hôte (`CGO_ENABLED=0`) — tous les constats sensibles aux courses sont marqués **[à vérifier]** avec la commande de confirmation sous Linux/WSL.

## 2. Tableau récapitulatif

| ID | Sévérité | Titre | Marqueur |
|----|----------|-------|----------|
| A2-01 | CRITIQUE | `GCController` mute un état GC global concurremment en mode comparaison | [à vérifier] |
| A2-02 | MAJEUR | `SetCacheLogger` écrit `tc.logger` hors verrou (lu par `logPeriodicStats`) | [à vérifier] |
| A2-03 | MAJEUR | Récursion FFT parallèle : `context` non propagé, annulation ignorée | [confirmé] |
| A2-04 | MINEUR | `threshold.SetTuning` mute des globaux de package sans synchronisation | [probable] |
| A2-05 | MINEUR | `acquireFFTState` ne « rétrécit » jamais `tmp/tmp2` : rétention mémoire par goroutine | [confirmé] |
| A2-06 | INFORMATIF | `getByKey` (cache LRU) : modèle snapshot-sous-RLock correct, à reconfirmer sous `-race` | [à vérifier] |
| A2-07 | INFORMATIF | Deux sémaphores `NumCPU` indépendants (fibonacci + bigfft) : sur-souscription contrôlée mais non bornée globalement | [confirmé] |

---

## 3. Détail des constats

### [A2-01] `GCController` mute un état GC global au processus, concurremment, en mode comparaison
- **Sévérité** : CRITIQUE
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/fibonacci/memory/gc_control.go:94-128` (Begin/End) ; déclenché depuis `internal/fibonacci/calculator.go:218,260` et `internal/orchestration/orchestrator.go:53-79`
- **Preuve** :

`ExecuteCalculations` lance N calculateurs concurrents quand il y en a plusieurs :

```go
// internal/orchestration/orchestrator.go
g, ctx := errgroup.WithContext(ctx)
for i, calc := range cfg.Calculators {
    idx, calculator := i, calc
    g.Go(func() error {
        res, err := calculator.Calculate(ctx, progressChan, idx, cfg.N, cfg.Opts)
        ...
    })
}
```

`GetCalculatorsToRun("all", ...)` renvoie bien plusieurs calculateurs :

```go
// internal/orchestration/calculator_selection.go:18-27
if algo == "all" {
    keys := factory.List()
    calculators := make([]fibonacci.Calculator, 0, len(keys))
    for _, k := range keys {
        if calc, err := factory.Get(k); err == nil {
            calculators = append(calculators, calc)
        }
    }
    return calculators
}
```

Chaque `Calculate` crée son propre `GCController` (mode « auto » par défaut puisque `app/calculate.go:79-83` ne renseigne pas `Opts.GCMode`) et l'active dès `N >= 1_000_000` :

```go
// internal/fibonacci/calculator.go:214-264
gcMode := opts.GCMode
if gcMode == "" { gcMode = "auto" }
gcCtrl := memory.NewGCController(gcMode, n)
...
err = gcCtrl.WithGC(func() error { ... })
```

Or `Begin`/`End` agissent sur un état **global au processus** :

```go
// internal/fibonacci/memory/gc_control.go:99-120
gc.originalGCPercent = debug.SetGCPercent(-1)
...
debug.SetGCPercent(gc.originalGCPercent)
debug.SetMemoryLimit(math.MaxInt64)
```

- **Impact** : avec deux calculateurs actifs simultanément sur `N >= 1M` :
  1. **Restauration GC erronée.** Le 1er `Begin` capture l'original (ex. 100) et pose `-1`. Le 2e `Begin`, exécuté après, capture `-1` comme « original ». Quand le 2e `End` s'exécute, il restaure `-1` (GC reste désactivé), puis le 1er `End` restaure 100. Selon l'ordonnancement, le GC peut rester **désactivé après la fin du calcul** (le 1er finit avant le 2e : 1er End restaure 100, 2e End repose -1) → fuite de réglage GC à l'échelle du processus, exactement la classe de bug que `WithGC` prétend éliminer mais seulement pour le cas mono-calcul + panic.
  2. **`debug.SetMemoryLimit` global** : le 1er `End` rétablit `math.MaxInt64`, supprimant le garde-fou OOM encore voulu par le 2e calcul toujours en cours.
  3. **Lecture/écriture concurrentes** de `gc.startStats`/`endStats` restent locales à chaque contrôleur (pas de course inter-objets), mais `runtime.ReadMemStats` impose un `stop-the-world` que deux contrôleurs déclenchent en se chevauchant.
- **Recommandation** : sérialiser l'activation GC au niveau processus. Options : (a) un `sync.Mutex` + compteur de profondeur package-level dans `memory` (refcount : seul le premier `Begin` pose `-1`, seul le dernier `End` restaure) ; ou (b) n'autoriser le contrôle GC que sur le chemin mono-calculateur (le fast path `len==1` de l'orchestrateur), et le désactiver en mode comparaison. Cela touche un comportement documenté (`WithGC` panic-safe) sans contredire l'invariant : formuler comme **proposition d'ADR** (extension de l'invariant `gc_control.go` au cas concurrent, aujourd'hui non couvert par `TestGCController_WithGC_*`).
- **Marqueur** : [à vérifier] — confirmer sous Linux/WSL : `CGO_ENABLED=1 go test -race -run TestExecuteCalculations ./internal/orchestration/` avec deux calculateurs et `N=2_000_000`, en instrumentant `debug.SetGCPercent`. La logique d'interleaving est prouvée par lecture ; seule l'observation par `-race` du double `SetGCPercent` reste à matérialiser.

---

### [A2-02] `SetCacheLogger` écrit `tc.logger` sans verrou, lu concurremment par `logPeriodicStats`
- **Sévérité** : MAJEUR
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/bigfft/fft_cache.go:98-101` (écriture) ; `internal/bigfft/fft_cache.go:286-307` (lecture)
- **Preuve** :

```go
// fft_cache.go:97-101 — écriture SANS verrou
func SetCacheLogger(l zerolog.Logger) {
    cache := GetTransformCache()
    cache.logger = l
}
```

```go
// fft_cache.go:298-307 — lecture SANS verrou (dans le hot path Get/Put)
tc.mu.RLock()
size := tc.lru.Len()
tc.mu.RUnlock()
tc.logger.Debug().
    Uint64("hits", hits).
    ...
```

Tout le reste de la structure protège ses champs : `config` via `cacheGate()` sous `RLock`, `entries`/`lru` sous `mu`, les compteurs en `atomic.Uint64`. Le champ `logger` est la seule exception — mutation directe hors section critique.

- **Impact** : si `SetCacheLogger` est appelé alors qu'un calcul FFT est en cours (les deux peuvent venir de goroutines distinctes : configuration tardive du logger pendant qu'un `--algo all` tourne), c'est une **data race** sur le champ `zerolog.Logger` (struct copiée par valeur, contient un pointeur `io.Writer` et un niveau). `zerolog.Logger` n'est pas conçu pour une réaffectation concurrente non synchronisée du champ qui le contient. En pratique `SetCacheLogger` est probablement appelé une fois au câblage avant tout calcul, ce qui rend la course improbable à l'exécution — d'où MAJEUR et non CRITIQUE — mais le contrat « `TransformCache` thread-safe » annoncé (l. 55-57) est violé pour ce champ.
- **Recommandation** : protéger l'écriture sous `tc.mu.Lock()` et la lecture sous `RLock` (cohérent avec `Config()`), ou stocker le logger dans un `atomic.Pointer[zerolog.Logger]`. Variante minimale : documenter explicitement « `SetCacheLogger` doit être appelé avant toute opération FFT » et l'asserter par convention de câblage.
- **Marqueur** : [à vérifier] — `CGO_ENABLED=1 go test -race -run TestTransformCache ./internal/bigfft/` en appelant `SetCacheLogger` depuis une goroutine pendant un `Get`/`Put` concurrent. La lecture du code confirme l'absence de verrou ; la matérialisation de la course nécessite `-race`.

---

### [A2-03] Récursion FFT parallèle : `context` non propagé, annulation/timeout ignorés
- **Sévérité** : MAJEUR
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/bigfft/fft_recursion.go:93-169` et `internal/bigfft/fft_recursion_ctx.go:25-91`
- **Preuve** :

`fourierRecursiveUnified` (et son jumeau `fourierRecursiveCtx`) ne reçoit **aucun** `context.Context`. La boucle de doublement vérifie `ctx.Err()` entre étapes (`doubling_framework.go:169`, `common.go:125`), mais une fois entré dans **une seule** multiplication FFT d'un opérande géant (F(10M+) → des centaines de Mbits), la transformée de Fourier récursive parallèle s'exécute jusqu'au bout sans jamais consulter le contexte :

```go
// fft_recursion.go:123-155
if size >= GetParallelFFTRecursionThreshold() && depth < GetMaxParallelFFTDepth() {
    select {
    case getSemaphore() <- struct{}{}:
        var wg sync.WaitGroup
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() { <-getSemaphore() }()
            ...
            errAsync = fourierRecursiveUnified(dst2, ..., size-1, depth+1, ...)
        }()
        errSync := fourierRecursiveUnified(dst1, ..., size-1, depth+1, ...)
        wg.Wait()
        ...
```

- **Impact** : un `Ctrl-C` ou un `context.WithTimeout` déclenché pendant **la** multiplication dominante d'un très grand `N` n'interrompt pas le calcul en cours ; le processus reste occupé jusqu'à la fin de la transformée (potentiellement plusieurs secondes à minutes pour les plus grands `N`). Ce n'est pas une fuite de goroutine (toutes sont jointes par `wg.Wait`) ni un deadlock (admission par `select`/`default` non bloquant, donc dégradation séquentielle propre), mais une **latence d'annulation non bornée** sur le chemin le plus coûteux. La granularité de vérification du contexte (entre étapes de doublement) est trop grossière pour les opérandes massifs.
- **Recommandation** : ne pas instrumenter le cœur récursif (coûteux, casserait l'invariant ADR-0003 de signatures hot path). Préférer une vérification de contexte au point d'admission parallèle (avant chaque `select` du sémaphore) ou un `atomic.Bool` « annulé » consulté en tête de `fourierRecursive*`, alimenté par un watcher du `ctx` côté `bigfft` entry point. À traiter comme amélioration de robustesse, pas comme bug de correction.
- **Marqueur** : [confirmé] — l'absence de paramètre `context` et de toute lecture de `ctx.Err()` dans les deux fichiers de récursion est vérifiable par lecture directe ; ne nécessite pas `-race`.

---

### [A2-04] `threshold.SetTuning` mute des globaux de package sans synchronisation
- **Sévérité** : MINEUR
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/fibonacci/threshold/manager.go:33-88` (variables + `SetTuning`) ; lectures en `analyzeFFTThreshold`/`analyzeParallelThreshold` (l. 262-287)
- **Preuve** :

```go
// manager.go:33-54 — globaux de package mutables, non atomiques
var (
    FFTSpeedupThreshold      = 1.2
    ParallelSpeedupThreshold = 1.1
    HysteresisMargin         = 0.15
)
var (
    minFFTThresholdFloor      = 100_000
    minParallelThresholdFloor = 1024
)

// manager.go:72-88 — écriture directe, sans verrou
func SetTuning(t Tuning) {
    if t.FFTSpeedupThreshold > 0 { FFTSpeedupThreshold = t.FFTSpeedupThreshold }
    ...
}
```

Ces variables sont lues dans `analyzeFFTThreshold` (`SpeedupThreshold: FFTSpeedupThreshold`, l. 265) potentiellement depuis la goroutine de doublement pendant un calcul.

- **Impact** : data race théorique si `SetTuning` était appelé pendant qu'un calcul actif lit ces globaux. Le `CLAUDE.md` et le commentaire de `SetTuning` (l. 67-71) documentent l'usage « once at startup from the wiring layer … before any manager is constructed », ce qui en fait un patron **single-writer-before-use** correct s'il est respecté. Contrairement aux champs d'instance du manager (correctement passés en `atomic.Int64`/`atomic.Pointer` — invariant A-18 obsolète bien appliqué), ces globaux de package restent en clair. Le risque est résiduel et conditionné au respect d'une convention non asservie par le compilateur.
- **Recommandation** : soit documenter l'invariant « `SetTuning` appelé avant toute construction de manager » directement au-dessus du bloc `var` (pas seulement dans `SetTuning`), soit migrer vers `atomic.Pointer[Tuning]` pour fermer définitivement la fenêtre. Comme l'invariant est volontaire, formuler une éventuelle migration comme **proposition d'ADR** (cohérence avec la migration atomic déjà faite pour les champs d'instance).
- **Marqueur** : [probable] — la course est conditionnelle (dépend d'un appel hors patron documenté) ; non déclenchable avec l'usage actuel mais structurellement présente.

---

### [A2-05] `acquireFFTState` ne rétrécit jamais `tmp/tmp2` : rétention mémoire par goroutine via `fftStatePool`
- **Sévérité** : MINEUR
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/bigfft/pool.go:455-499`
- **Preuve** :

```go
// pool.go:462-475
tmpSize := n + 1
if cap(state.tmp) < tmpSize {
    state.tmp = acquireFermat(tmpSize)
} else {
    state.tmp = state.tmp[:tmpSize]
    clear(state.tmp)
}
```

`releaseFFTState` (l. 491-499) remet l'état au pool **sans** relâcher `tmp`/`tmp2` ni les redimensionner. Un état ayant servi un très grand `n` conserve donc indéfiniment des buffers `fermat` massifs ; le prochain `acquireFFTState(petit n)` les réutilise tels quels (branche `else`) et ne libère jamais la sur-capacité.

- **Impact** : ce n'est pas une fuite au sens strict (la mémoire est joignable et réutilisée), mais une **rétention de pic** : après un F(50M), les `fftState` du pool gardent des `tmp` de plusieurs Mo même si la charge ultérieure redevient petite. Contraste avec la discipline anti-bloat explicite ailleurs (`fastdoubling.go:415` `maxArenaPoolWords`, `common.go:65` `MaxPooledBitLen`) — `fftStatePool` n'a pas d'équivalent. Pas de problème de correction ni de course (chaque goroutine détient son `fftState` le temps d'une opération).
- **Recommandation** : aligner sur le patron anti-bloat existant : dans `releaseFFTState`, si `cap(state.tmp)` dépasse un seuil, relâcher `tmp`/`tmp2` au pool fermat via `releaseFermat` et les remettre à `nil` avant `Put`. Diff minimal, cohérent avec `bigfft/pool.go`.
- **Marqueur** : [confirmé] — l'absence de redimensionnement à la baisse et de relâche des buffers est lisible directement dans les deux fonctions.

---

### [A2-06] Cache LRU `getByKey` : modèle snapshot-sous-RLock + non-recyclage du backing — correct par construction
- **Sévérité** : INFORMATIF
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/bigfft/fft_cache.go:242-283` (`getByKey`) et `335-384` (`putByKey`)
- **Preuve** :

```go
// fft_cache.go:243-264
tc.mu.RLock()
elem, found := tc.entries[key]
...
entry := elem.Value.(*cacheEntry)
pv := PolValues{ K: entry.k, N: entry.n, Values: entry.values }
atFront := tc.lru.Front() == elem
tc.mu.RUnlock()
if !atFront {
    tc.mu.Lock()
    if _, stillPresent := tc.entries[key]; stillPresent {
        tc.lru.MoveToFront(elem)
    }
    tc.mu.Unlock()
}
```

Le `PolValues` retourné aliase `entry.values` → `entry.backing`. L'invariant E1-R4 (CONVENTIONS §F) impose que `putByKey` alloue **toujours** un backing frais et laisse les `backing` évincés au GC ; c'est bien le cas (`fft_cache.go:366` `backing := make([]big.Word, wordCount)`, commentaire l. 348-363). Donc un `PolValues` obtenu par `Get()` puis évincé par un `putByKey` concurrent reste valide : aucune écriture ne survient sur l'ancien `backing` après éviction.

- **Impact** : aucun défaut identifié. Toutes les mutations de `entries`/`lru` se font sous `Lock` exclusif ; les lectures (`Front()`, indexation map) sous `RLock`. La re-vérification `stillPresent` avant `MoveToFront` ferme proprement la fenêtre RUnlock→Lock. L'invariant anti-aliasing est respecté. Ce constat documente la **vérification positive** de la zone la plus délicate du mandat.
- **Recommandation** : aucune. Préserver l'invariant `putByKey` backing-frais (ne pas réintroduire le recyclage à l'éviction).
- **Marqueur** : [à vérifier] — l'analyse statique conclut à la correction, mais une confirmation `CGO_ENABLED=1 go test -race -run 'TestTransformCache|TestFFTCache' ./internal/bigfft/` (et un stress Get/Put concurrent) reste la seule preuve dynamique possible faute de `-race` ici.

---

### [A2-07] Deux sémaphores `NumCPU` indépendants : sur-souscription bornée par paire mais non globalement
- **Sévérité** : INFORMATIF
- **Axe** : 2 Concurrence
- **Emplacement** : `internal/fibonacci/common.go:40-53` (`globalSem`, `NumCPU`) et `internal/bigfft/fft_recursion.go:12-24` (`concurrencySemaphore`, `NumCPU`)
- **Preuve** :

```go
// common.go:48-53
func getTaskSemaphore() chan struct{} {
    globalSemOnce.Do(func() { globalSem = make(chan struct{}, runtime.NumCPU()) })
    return globalSem
}
```

```go
// fft_recursion.go:19-24
func getSemaphore() chan struct{} {
    concurrencyOnce.Do(func() { concurrencySemaphore = make(chan struct{}, runtime.NumCPU()) })
    return concurrencySemaphore
}
```

Le commentaire `common.go:29-35` reconnaît explicitement que les deux sémaphores restent séparés (fusionner imposerait à `bigfft` d'importer `fibonacci` — inversion de couche).

- **Impact** : en mode parallèle, la couche fibonacci peut occuper jusqu'à `NumCPU` goroutines (3 multiplications via `executeParallel3`), chacune pouvant déclencher une FFT qui ouvre jusqu'à `NumCPU` goroutines supplémentaires côté bigfft. Plafond théorique combiné ≈ `2·NumCPU` goroutines CPU-bound. `shouldParallelizeMultiplicationCached` (`fastdoubling.go:204-205`) atténue cela en **désactivant** la parallélisation fibonacci dès que FFT entre en jeu (sauf au-delà de `ParallelFFTThreshold`), donc les deux niveaux se chevauchent rarement. Admission non bloquante (`select`/`default`) côté FFT → pas de deadlock même en saturation. Pas de défaut, mais un point de contention résiduel à connaître pour le tuning.
- **Recommandation** : aucune action requise. Si un jour la contention combinée devient mesurable, envisager un sémaphore partagé injecté (passé en paramètre plutôt qu'importé) — cela rejoint la piste `FFTContext` exclusif déjà tracée en backlog (ADR-0004 §B1).
- **Marqueur** : [confirmé] — les deux sémaphores, leur dimensionnement `NumCPU` et leur indépendance sont vérifiables par lecture directe.

---

## 4. Notes de méthode

- `go test -race` **non exécuté** : `CGO_ENABLED=0`, aucun compilateur C sur l'hôte (cf. `00_bootstrap.md §2`). Tous les constats sensibles aux courses portent **[à vérifier]** avec la commande de confirmation sous Linux/WSL.
- `go vet ./...` (avec `copylocks`, `atomic`, `loopclosure`, `lostcancel`) : **exit 0, aucun diagnostic** — vérifié sur `./...` puis re-vérifié explicitement sur `./internal/...`.
- Analyse strictement en lecture seule ; aucun fichier source modifié.
