# Préparation Phase 3 — vérification documentaire (workflow multi-agents, 2026-06-10)

Synthèse produite par l'agent de consolidation à partir de :

- la **revue adversariale C2** : 3 lentilles indépendantes sur le commit `fa13bfd` (`perf(fibonacci): cache per-calculator state+arena across calls`) ;
- **16 rapports de vérification de claims documentaires** couvrant la documentation du dépôt au HEAD du 2026-06-10.

Compteurs globaux : **1 019 claims passés en revue**, **924 sans problème**, **95 issues** (77 périmés, 15 douteux, 2 liens cassés, 1 non vérifiable statiquement).

---

## 1. Revue adversariale du commit fa13bfd — verdicts et findings (par sévérité décroissante)

### Verdicts des 3 lentilles

| Lentille | Verdict |
|---|---|
| Aliasing & cycle de vie (arène per-calculator slot) | **approve** |
| Concurrence — protocole `Swap(nil)`/`CompareAndSwap(nil,s)` du slot `cachedState` (exclusivité d'ownership, double Put, happens-before, rétention par goroutines internes) | **approve** |
| Conformité-repo (invariants Claude.md, tests gardiens, conventions, politique anti-bloat) | **concerns** |

**Bilan : 0 critical, 0 major, 7 minor, 7 info.** Le commit est sain par construction (protocole d'ownership prouvé correct par la lentille Concurrence) ; les réserves portent sur (a) l'absence de run `-race` tracé, (b) la robustesse de 2 tests gardiens, (c) la dérive des invariants documentés dans Claude.md.

### Findings minor (7, dont 2 redondants inter-lentilles sur `-race`)

1. **[Conformité + Concurrence] Validation `-race` du protocole Swap/CAS différée, non enregistrée avant commit** — doublon inter-lentilles.
   - Localisation : `internal/fibonacci/state_cache_test.go:111-129` (`TestCalculatorStateCache_ConcurrentCalls`), `state_pool_arena_test.go:48` (`TestArenaStateConcurrent`), `fft_race_test.go:194` (`TestFFTRaceArenaAliasing_ConcurrentCalculateCore`).
   - Détail : le commit introduit un protocole lock-free neuf (`cachedState.Swap(nil)` à l'acquisition `fastdoubling.go:375`, `CompareAndSwap(nil,s)` au release `:416`) ; le message de commit indique « -race délégué WSL/Linux (gcc absent sur cet hôte) ». Sans `-race`, les trois gardiens concurrents ne vérifient que valeurs/absence de panic, pas les data races du modèle mémoire. Non bloquant (protocole correct par construction, cf. findings info), mais **exigé avant merge** par CLAUDE.md (Directive 8 + section Commandes : validation `-race` via WSL). Trace d'exécution à archiver — voir §4, commande 1.

2. **[Aliasing] Le test gardien du deep-copy ne réfute pas réellement l'écrasement d'un résultat vivant.**
   - Localisation : `internal/fibonacci/state_cache_test.go:81-109` (et `fastdoubling.go:398-408`).
   - Détail : `TestCalculatorStateCache_SequentialResultsIndependent` utilise la séquence 100k → 120k → 100k, qui ne réutilise jamais l'arène adossant le snapshot vivant : totalWords(120k)=19 530 > arenaCapWords(100k)=16 275, donc le 2e appel **remplace** l'arène (`prepareStateForN`, `fastdoubling.go:348-350`) au lieu de la réutiliser ; r1 resterait intact même si le deep-copy de `fd.releaseStateWithResult` (`:403-405`) était supprimé. Risque aggravé par la duplication du code de copie (`fd.releaseStateWithResult` `:398-408` vs `ReleaseStateWithResult` `:466-481`).
   - Correctif Phase 4 : choisir n2 ≤ n1 (ex. 100k → 90k) pour que le 2e appel écrase réellement le backing du snapshot, et/ou factoriser la copie dans un helper unique.

3. **[Aliasing] `TestCalculatorStateCache_ReusesArena` dépend de l'état du `statePool` partagé : flake possible sous `t.Parallel`.**
   - Localisation : `internal/fibonacci/state_cache_test.go:14-37`.
   - Détail : le premier acquire peut hériter d'un state portant une arène > `maxCachedArenaWords` mise au pool par un autre test parallèle (typiquement `OverCapGoesToPool`, n=30M ≈ 4,88M mots) ; `fd.releaseStateWithResult` la route alors vers le pool (`:416`), le slot reste nil, et s2 dépend du hasard du `sync.Pool` (test vert sans exercer le slot, ou échec parasite ligne 32). Fenêtre étroite, robustesse du gardien uniquement.
   - Correctif Phase 4 : pré-peupler `fd.cachedState` avec un state connu, ou précondition `s1.arenaCapWords <= maxCachedArenaWords` + `t.Skip` sinon.

4. **[Conformité] Claude.md:69 (« Patterns de performance critiques ») désormais inexacte : triple drift.**
   - Détail : (1) le chemin unique de teardown est `finalizeStateReleaseTo` (`fastdoubling.go:518`) ; `finalizeStateRelease` (`:504-506`) n'est qu'un wrapper liant le sink par défaut, contourné par les releases du calculateur (`:391`, `:406`) ; (2) l'ordre se termine par un sink paramétrique (`statePool.Put` **ou** `cachedState.CompareAndSwap`), pas littéralement `Put` ; (3) la réutilisation inter-appels du chemin de production par défaut passe par `fd.acquireStateForN`/`fd.releaseStateWithResult` (cache GC-immune) — la paire publique ne reste le mécanisme que pour `FFTOnlyCalculator` (`fft_based.go:53-65`).
   - Amendement Phase 4 : nommer `finalizeStateReleaseTo` comme chemin unique, ordre `checkLimit → clearStateAliases → sink`, sinks = `statePool.Put` | `cachedState` (slot GC-immune borné par `maxCachedArenaWords`), gardiens `TestReleaseState_OverLimit_AliasesCleared` + les 5 `TestCalculatorStateCache_*`.

5. **[Conformité] Claude.md:84 (tableau « Invariants à préserver », ligne fastdoubling.go) : vrai transitivement mais ne nomme plus le vrai chemin de teardown ni les nouveaux gardiens.**
   - Détail : « `finalizeStateRelease` appelle `clearStateAliases` inconditionnellement » reste vrai par délégation (`:504` → `:518` → `:528`), mais ne couvre plus la surface complète de release : `fd.releaseState` (`:387-392`) et `fd.releaseStateWithResult` (`:398-408`) appellent `finalizeStateReleaseTo` directement avec le sink `fd.cacheOrPool`.
   - Amendement Phase 4 : reformuler sur `finalizeStateReleaseTo` (clearStateAliases inconditionnel avant tout sink ; overLimit jamais vers le sink) et ajouter `TestCalculatorStateCache_OverLimitNotCached` (`state_cache_test.go:66`) comme gardien du sink calculateur.

6. **[Conformité] Claude.md:103 (tableau « Modules sensibles », ligne fastdoubling.go) : sous-spécifie le nouveau sink GC-immune.**
   - Détail : « tout chemin de release doit détacher les aliases avant `statePool.Put` » est littéralement encore vrai, mais un chemin de release peut désormais se terminer dans `fd.cachedState.CompareAndSwap` (`:416`) **sans jamais atteindre** `statePool.Put` — et le détachement y est encore plus critique (slot GC-immune réutilisé tel quel au prochain appel). Un contributeur ajoutant un fast-path CAS sans passer par `finalizeStateReleaseTo` serait littéralement conforme au tableau tout en violant l'invariant.
   - Amendement Phase 4 : « ... doit détacher les aliases avant tout sink (`statePool.Put` ou `cachedState`) via `finalizeStateReleaseTo` ».

7. *(Compté dans le finding 1 — la lentille Concurrence porte le même constat `-race` que la lentille Conformité ; localisations identiques.)*

### Findings info (7)

1. **[Aliasing] Dérive documentaire** : CLAUDE.md décrit encore le cycle de vie pool-only (recoupe les minors 4-6 ci-dessus ; le slot GC-immune `fd.cachedState` est un second point de publication soumis au même invariant). Localisation : Claude.md, tableaux Invariants + Modules sensibles.
2. **[Concurrence] Double détention impossible par construction** : `Swap(nil)` est un RMW atomique (un seul appelant reçoit un pointeur non nil) ; `cacheOrPool` (`fastdoubling.go:415-420`) fait CAS réussi XOR exactement un `statePool.Put` ; `CalculateCore` libère exactement une fois (`:167` XOR `:170`) ; `finalizeStateReleaseTo` invoque le sink au plus une fois, branche overLimit retournant avant (`:538-543`, gardée par `TestCalculatorStateCache_OverLimitNotCached`).
3. **[Concurrence] Happens-before établi** : toutes les écritures non atomiques du libérateur (`clearStateAliases`, `arena.Reset`, `arena=nil`/`arenaCapWords=0`) sont séquencées avant le `CompareAndSwap` publiant ; l'acquéreur dont le `Swap(nil)` observe le pointeur obtient l'arête synchronized-before du modèle mémoire Go. Pas de risque ABA. Localisation : `fastdoubling.go:375-377, 416-417, 518-545` ; `memory/arena.go:117-119`.
4. **[Concurrence] Aucune goroutine interne ne retient FK/T1-T3 après release** : `executeParallel3` (`common.go:161`), `executeTasks` (`:250`), `executeMixedTasks` (`:324`) et les chemins bigfft (`fft_poly.go:514`, `fft_recursion.go:148, 253`, `fft_recursion_ctx.go:71`) joignent tous avant retour. Nuance : le slot rend la réutilisation déterministe, ce qui amplifierait un futur bug de goroutine non jointe — inexistant aujourd'hui, couvert par les gardiens à oracle.
5. **[Concurrence] Copie par valeur du calculateur (seul vecteur de double ownership) structurellement fermée** : `atomic.Pointer` embarque noCopy → `go vet` (copylocks) signale toute copie ; tous les sites de construction utilisent `&FastDoublingCalculator{}` (`registry.go:64, 125-157`). Chemin panic sain (fuite vers GC, pool intact).
6. **[Conformité] Asymétrie de couverture gardienne** : aucune assertion directe que le state **dans** le slot cache est alias-free (pas de `fd.cachedState.Load()` post-release nominal assertant BitLen==0 + pointeurs changés sur les 5 slots). Garanti structurellement (code partagé `:528` avant dispatch `:544`) ; ajout trivial en Phase 4. Localisation : `fastdoubling.go:410-420`.
7. **[Conformité] Conformité vérifiée** : gardien historique intact (dernier commit le touchant : 8304379) ; 5 nouveaux tests tous `t.Parallel()` ; aucun nouveau global mutable ; doc de `finalizeStateReleaseTo` vérifiée ligne à ligne ; anti-bloat cohérent (gate cache 4M mots strictement plus serré que le drop pool 50M, esprit A2-05). Nit : le commentaire de `finalizeStateRelease` (`:483`) se décrit encore comme « the single shared teardown path ». Aucun fichier bigfft ni golden touché.

---

## 2. Synthèse par document

| Document | Claims vérifiés | Issues | Liens cassés |
|---|---:|---|---:|
| docs/ARCH.md | 87 | 10 (9 périmés, 1 lien cassé) | 1 |
| docs/BUILD.md | 113 | 2 (2 périmés) | 0 |
| docs/CALIBRATION.md | 83 | 5 (5 périmés) | 0 |
| docs/PERFORMANCE.md | 62 | 8 (6 périmés, 2 douteux) | 0 |
| docs/PORTABILITY.md | 27 | 2 (2 périmés) | 0 |
| docs/TESTING.md | 85 | 6 (5 périmés, 1 douteux) | 0 |
| docs/TUI_GUIDE.md | 83 | 6 (5 périmés, 1 douteux) | 0 |
| docs/algorithms/FAST_DOUBLING.md | 34 | 5 (5 périmés) | 0 |
| docs/algorithms/MATRIX.md | 48 | 7 (6 périmés, 1 non vérifiable statiquement) | 0 |
| docs/algorithms/FFT.md | 31 | 3 (2 périmés, 1 douteux) | 0 |
| docs/algorithms/BIGFFT.md | 55 | 7 (6 périmés, 1 douteux) | 0 |
| docs/algorithms/GMP.md | 26 | 3 (1 périmé, 2 douteux) | 0 |
| docs/algorithms/COMPARISON.md + PROGRESS_BAR_ALGORITHM.md | 68 | 6 (4 périmés, 2 douteux) | 0 |
| docs/architecture/README.md + diagrammes Mermaid + patterns/design-patterns.md | 62 | 11 (9 périmés, 1 douteux, 1 lien cassé) | 1 |
| CHANGELOG.md + CONTRIBUTING.md | 87 | 6 (3 périmés, 3 douteux) | 0 |
| docs/adr/0001-0008 + docs/external-reviews/2026-02-08-jules-self-evaluation.md | 68 | 8 (7 périmés, 1 douteux) | 0 |
| **Total** | **1 019** | **95** (77 périmés, 15 douteux, 2 liens cassés, 1 non vérifiable) | **2** |

Notes :

- « Claims vérifiés » = nombre de claims passés en revue par l'agent vérificateur (issues incluses). Claims sans problème : **924**.
- Les URLs du footer de CHANGELOG.md (dépôt `agbru/fibcalc` et tags `v1.0.0`/`v0.1.0` inexistants) sont classées « périmé » par l'agent vérificateur mais constituent de facto des liens externes cassés supplémentaires.

---

## 3. Détail des issues par document

### 3.1 docs/ARCH.md (87 claims, 10 issues)

1. **périmé** — `docs/ARCH.md:192` — *Claim* : §4 interface nommée `coreCalculator` (minuscule, interne). *Évidence* : `internal/fibonacci/calculator.go:64` définit `type CoreCalculator interface` (exportée). *Fix* : remplacer par `CoreCalculator` en §4 et à la ligne 263 (tableau des patterns, Decorator).
2. **périmé** — `docs/ARCH.md:198-199,422,431,485-486` — *Claim* : types `OptimizedFastDoubling` et `MatrixExponentiation`. *Évidence* : vrais types `FastDoublingCalculator` (`fastdoubling.go:86`) et `MatrixExponentiationCalculator` (`matrix.go:45`) ; les noms documentés n'existent nulle part. *Fix* : substituer dans toutes les occurrences de §4, §7 et §8.
3. **périmé** — `docs/ARCH.md:183,237,515` — *Claim* : `CLIProgressReporter` implémente `ProgressReporter` dans `internal/cli/presenter.go`. *Évidence* : aucun résultat grep ; le progrès CLI = fonction `cli.DisplayProgress` (`display.go:326`) enveloppée par `orchestration.ProgressReporterFunc` (`interfaces.go:93`) ; `presenter.go` ne contient que `CLIResultPresenter`. *Fix* : supprimer `CLIProgressReporter` ; documenter `orchestration.ProgressReporterFunc(cli.DisplayProgress)`.
4. **périmé** — `docs/ARCH.md:397` — *Claim* : sémaphore `NumCPU*2` goroutines max. *Évidence* : `common.go:50` `globalSem = make(chan struct{}, runtime.NumCPU())` ; le commentaire `:28-29` dit explicitement que NumCPU*2 est l'ancienne valeur. *Fix* : « Semaphore: NumCPU concurrent goroutines max ».
5. **périmé** — `docs/ARCH.md:342,348` — *Claim* : flux étape 7 `GCController.Begin()`/`End()`. *Évidence* : `calculator.go:260` utilise `gcCtrl.WithGC(func() error {...})` ; `Begin`/`End` sont `Deprecated` (`memory/gc_control.go:111`). *Fix* : remplacer les deux lignes par `gcCtrl.WithGC(fn)` (panic-safe, unique point d'entrée production).
6. **périmé** — `docs/ARCH.md:353` — *Claim* : étape 8 `AcquireState() from sync.Pool`. *Évidence* : depuis fa13bfd, `fastdoubling.go:133` = `s := fd.acquireStateForN(n)` (`:374`), qui préfère le slot `fd.cachedState` (atomic.Pointer, GC-immune) avant le pool. *Fix* : montrer `fd.acquireStateForN(n)` (slot GC-immune → pool de secours) et documenter `cachedState`.
7. **périmé** — `docs/ARCH.md:731` — *Claim* : registre formel ADR `0001`..`0007`. *Évidence* : `docs/adr/` contient 0001 à 0008 (`0008-audit-2026-06-rejected-candidates.md`). *Fix* : « `0001`..`0008` ».
8. **périmé** — `docs/ARCH.md:672` — *Claim* : targets Makefile `install-mockgen` et `generate-mocks`. *Évidence* : absentes du Makefile ; seule `install-tools` existe (`Makefile:279`). *Fix* : supprimer ou remplacer par l'équivalent réel.
9. **périmé** — `docs/ARCH.md:169` — *Claim* : `AppConfig` a 23 champs. *Évidence* : `config.go:36-87` = 20 champs. *Fix* : « 20 fields », ou retirer le décompte statique sujet à dérive.
10. **lien cassé** — `docs/ARCH.md:5` — *Claim* : source du knowledge graph `.understand-anything/knowledge-graph.json`. *Évidence* : répertoire inexistant dans le dépôt ; JSON réel = `docs/dashboard/knowledge-graph.json`. *Fix* : corriger le chemin source.

### 3.2 docs/BUILD.md (113 claims, 2 issues)

1. **périmé** — `docs/BUILD.md:203` — *Claim* : `build-pgo-all` = « Build all platforms with PGO » (parité implicite avec `build-all`). *Évidence* : `Makefile:95` = linux/amd64 + windows/amd64 + darwin (amd64+arm64) ; pas de PGO pour linux/arm64 ni windows/arm64, contrairement à `build-all` (`Makefile:112`). *Fix* : décrire les plateformes réelles, ou ajouter `build-pgo-linux-arm64`/`build-pgo-windows-arm64` au Makefile.
2. **périmé** — `docs/BUILD.md:189-248` — *Claim* : la table Utility Targets est complète. *Évidence* : `stats` (`Makefile:194-214`, référencée par CLAUDE.md), `bench-baseline` et `bench-versioned` sont absentes (seules `version` et `help` listées). *Fix* : ajouter les trois lignes ou une section « Diagnostic / Baseline Targets ».

### 3.3 docs/CALIBRATION.md (83 claims, 5 issues)

1. **périmé** — `docs/CALIBRATION.md:298-306` — *Claim* : table heuristique `EstimateOptimalParallelThreshold()`. *Évidence* : les lignes par nombre de cœurs correspondent au code (`thresholds.go:55-70`), mais la couche d'ajustement SIMD (`estimateParallelThresholdForHeuristic`, `thresholds.go:40-53` : réduction de 256/512 bits si AVX2/AVX-512 et 8+ cœurs) est totalement absente. *Fix* : ajouter une note sous la table.
2. **périmé** — `docs/CALIBRATION.md:308` — *Claim* : FFT « 500 000 bits en 64-bit, 250 000 en 32-bit ». *Évidence* : `thresholds.go:78-95` retourne 460 000 (AVX-512), 480 000 (AVX2), 500 000 (sans SIMD) ; le cas 32-bit est correct. *Fix* : « 500k (générique 64-bit) ; 480k (AVX2) ; 460k (AVX-512) ; 250k (32-bit) ».
3. **périmé** — `docs/CALIBRATION.md:310-311` — *Claim* : Strassen « 256 bits si 4+ cœurs, 3 072 sinon ». *Évidence* : `thresholds.go:103-117` : 224 (AVX-512), 240 (AVX2), 256 (générique) pour 4+ cœurs ; 3 072 correct. *Fix* : détailler les trois valeurs.
4. **périmé** — `docs/CALIBRATION.md:339-346` — *Claim* : structure du package = 7 fichiers. *Évidence* : `strategy.go`, `strategy_fast.go`, `strategy_complete.go` présents sur disque mais absents de la table (l'abstraction est décrite en prose lignes 113-114). *Fix* : ajouter les 3 lignes (interface `CalibrationStrategy`, `FastStrategy`, `CompleteStrategy`).
5. **périmé** — `docs/CALIBRATION.md:370` — *Claim* : « FIBCALC_PROFILE_MAX_AGE : voir internal/config/env.go pour la liste complète ». *Évidence* : la variable est définie et consommée par `internal/calibration/calibration.go:41,48` (`ProfileMaxAgeEnv`/`profileMaxAgeFromEnv`), absente de `env.go`. *Fix* : corriger la note de bas de table.

### 3.4 docs/PERFORMANCE.md (62 claims, 8 issues)

1. **périmé** — `docs/PERFORMANCE.md:55,58,302,306,310` — *Claim* : `-bench=BenchmarkFastDoubling` (5 occurrences). *Évidence* : aucune fonction de ce nom ; seul `BenchmarkFibonacci` (`fibonacci_test.go:190`) avec sous-tests `FastDoubling/1M` etc. La commande matche 0 benchmark, silencieusement. *Fix* : `-bench='BenchmarkFibonacci/FastDoubling'` (ou `BenchmarkFibonacci`).
2. **périmé** — `docs/PERFORMANCE.md:94` — *Claim* : flags de `make bench-versioned`. *Évidence* : `Makefile:250-252` = `-bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -count=3 -benchtime=2s ./internal/fibonacci/`. *Fix* : aligner la description.
3. **périmé** — `docs/PERFORMANCE.md:261,289` — *Claim* : `FFTCacheMaxEntries` défaut = 128. *Évidence* : `fft_cache.go:37` `MaxEntries: 256` (asserté par `fft_cache_test.go:21`). *Fix* : 256 dans la table et l'exemple `opts`.
4. **périmé** — `docs/PERFORMANCE.md:123-139` — *Claim* : snippet `sync.Pool.New` retourne `interface{}` + « 20-30% performance improvement ». *Évidence* : code réel `any` (`fastdoubling.go:271`) ; chiffre non sourcé/daté ; fa13bfd ajoute le slot GC-immune non décrit (gains récents mesurés : −27,6 % via 1da5a6d puis −12,3 % via fa13bfd sur FastDoubling/10M). *Fix* : `any` + sourcer le chiffre (docs/audits/) + note sur `cachedState` (cap `maxCachedArenaWords` = 4M mots).
5. **périmé** — `docs/PERFORMANCE.md:141` — *Claim* : arène gérée pool-only (`AcquireStateForN`/`ReleaseStateWithResult`, drop > `maxArenaPoolWords` ~50M mots ≈ 400 Mo). *Évidence* : fa13bfd → slot `cachedState` par calculateur (cap 4M mots ≈ 32 Mo, `fastdoubling.go:31`), release via `finalizeStateReleaseTo` → `cacheOrPool()` ; le drop pool 400 Mo reste exact. *Fix* : ajouter une phrase sur le slot GC-immune (survit au GC de `GCController.End()` ; arènes > 32 Mo = pool-only).
6. **douteux** — `docs/PERFORMANCE.md:28` — *Claim* : table N=10M « Fast Doubling 2.1s / FFT-Based 2.3s » (Ryzen). *Évidence* : la note de provenance (Go 1.25.0) ne signale pas l'ampleur des gains post-baseline : 1da5a6d (−27,6 % FastDoubling/10M) + fa13bfd (−12,3 %). *Fix* : étendre la note de provenance avec les deux commits et la référence `docs/audits/bench-parallel-pointwise-2026-06.md`.
7. **douteux** — `docs/PERFORMANCE.md:263` — *Claim* : « 15-30% speedup » du cache FFT. *Évidence* : non sourcé (aucune mesure datée dans docs/audits/), et inapplicable au Fast Doubling par défaut (zéro hit/miss — la note adjacente le dit). *Fix* : caveat « chemins `FFTOnlyStrategy` et `bigfft.Mul/Sqr` directs ; aucune mesure datée sur fichier » ou citer un audit (mesure proposée en §4).
8. **périmé** — `docs/PERFORMANCE.md:67-70` — *Claim* : exemple de comparaison `make benchmark > /tmp/new.txt` + benchstat vs `docs/audits/bench-baseline.txt`. *Évidence* : `make benchmark` (`-bench=.`, sans `-count`) n'est pas benchstat-comparable à la baseline (`-count=5 -benchtime=1x`, sous-tests ciblés). *Fix* : `make bench-baseline > /tmp/new.txt` puis `benchstat docs/audits/bench-baseline.txt /tmp/new.txt`, ou noter l'exigence de flags identiques.

### 3.5 docs/PORTABILITY.md (27 claims, 2 issues)

1. **périmé** — `docs/PORTABILITY.md` §2.1 (lignes 23-25) — *Claim* : `arith_amd64.go` = implémentation assembleur de `addVV, subVV, addVW, subVW, shlVU`. *Évidence* : le fichier n'exporte que `AddVV`, `SubVV`, `AddMulVVW` (`:9-32`) ; `addVW`/`subVW`/`shlVU` sont dans `arith_decl.go:43-55` (`go:linkname`, commun aux deux architectures) ; aucun fichier `.s` dans `internal/bigfft/` — délégation à `math/big` via `go:linkname`. *Fix* : corriger la liste (`AddVV`, `SubVV`, `AddMulVVW`) et préciser la délégation (pas d'assembleur original dans ce dépôt).
2. **périmé** — `docs/PORTABILITY.md` §2.2 (lignes 35-38) — *Claim* : détection runtime AVX2 « pour activer un dispatch optimisé dans les routines FFT ». *Évidence* : `HasAVX2()` n'est appelé nulle part hors tests (`arith_amd64_test.go:47,51,55`, `cpu_amd64_extended_test.go:43-53`) ; aucun appel dans `fft.go`/`fft_recursion.go`/`fft_poly.go`/`fft_cache.go`. *Fix* : « détection SIMD présente (AVX2, AVX512, BMI2, ADX) ; dispatch FFT non implémenté ; usage actuel limité aux tests ».

### 3.6 docs/TESTING.md (85 claims, 6 issues)

1. **périmé** — `docs/TESTING.md:360` — *Claim* : `internal/metrics/memory_test.go` comme fichier de test clé. *Évidence* : fichier inexistant ; réel = `indicators_test.go` (+ `system/system_test.go`). *Fix* : remplacer et décrire le contenu réel (indicateurs de performance, throughput).
2. **périmé** — `docs/TESTING.md:405` — *Claim* : `TestProgressReporter` valide la monotonie inter-goroutines. *Évidence* : fonction inexistante ; réels = `FuzzProgressMonotonicity` (`fibonacci_fuzz_test.go:324`) et `TestProgress_MonotonicLargeN` (`progress_test.go:88`). *Fix* : remplacer le bullet par les deux noms réels.
3. **périmé** — `docs/TESTING.md:55-58` — *Claim* : snippet `NewCalculator(&OptimizedFastDoubling{})` / `&MatrixExponentiation{}`. *Évidence* : types inexistants ; vrais types `FastDoublingCalculator`/`MatrixExponentiationCalculator` et constructeur `MustNewCalculator` (cf. `fibonacci_test.go:40-42`). *Fix* : corriger les trois identifiants du snippet.
4. **périmé** — `docs/TESTING.md:351` — *Claim* : table des fichiers de test `internal/fibonacci` (10 listés). *Évidence* : manquent `testmain_test.go` (commit 4e34b82) et `state_cache_test.go` (commit fa13bfd, 5 gardiens du cache d'arène) ; 35 fichiers `_test.go` réels au total. *Fix* : ajouter au minimum ces deux fichiers + noter que la table est non exhaustive.
5. **douteux** — `docs/TESTING.md:188-213` — *Claim* : property-based = Cassini uniquement, « 300 total ». *Évidence* : 4 fonctions réelles (`TestCassinisIdentity_`, `TestRecurrenceRelation_`, `TestDoublingIdentity_`, `TestGCDIdentity_PropertyBased` — la dernière sur FastDoubling seul). *Fix* : lister les 4 propriétés ; préciser que 300 = par propriété à 3 calculateurs.
6. **périmé** — `docs/TESTING.md:362` — *Claim* : `internal/app` = `app_test.go`, `version_test.go`. *Évidence* : `app_tuning_test.go` existe (`TestWireThresholdTuning`, contrat A2-04). *Fix* : ajouter le fichier + « threshold-tuning wiring (A2-04) ».

### 3.7 docs/TUI_GUIDE.md (83 claims, 6 issues)

1. **périmé** — `docs/TUI_GUIDE.md:227` — *Claim* : ETA calculée via `cli.NewProgressWithETA()`. *Évidence* : fonction inexistante ; `bridge.go:80` utilise `orchestration.NewProgressAggregator(numCalculators)` (`orchestration/progress.go:60`). *Fix* : corriger la description du flux.
2. **périmé** — `docs/TUI_GUIDE.md:290` — *Claim* : `FormatDuration()` délègue à `cli.FormatExecutionDuration()`. *Évidence* : `bridge.go:128` appelle `format.FormatExecutionDuration(d)` ; le package `cli` n'est pas importé par bridge.go. *Fix* : `format.FormatExecutionDuration()` (package `internal/format`).
3. **périmé** — `docs/TUI_GUIDE.md:437` — *Claim* : ajouter un `case` dans `handleKey()` « dans model.go ». *Évidence* : `handleKey()` est défini dans `internal/tui/handlers.go:93` (appelé depuis `model.go:92`). *Fix* : « dans `handlers.go` ».
4. **douteux** — `docs/TUI_GUIDE.md:302` — *Claim* : touche `r` = « Restart calculation ». *Évidence* : incohérence interne au code — `keymap.go:29` = « Reset », `footer.go:47` = « Restart ». Le doc matche footer.go mais contredit keymap.go. *Fix* : aligner keymap.go et footer.go sur un libellé unique (« Restart » cohérent avec §9 et footer.go), puis le doc.
5. **périmé** — `docs/TUI_GUIDE.md:136` — *Claim* : diagramme « Memory: 1.2 GB | GC Runs: 12 ». *Évidence* : `metrics.go:85-92` rend « Heap: X / Y | GC: N (Xms) » (le §4 du même doc est correct). *Fix* : « Heap: 1.2 GB / 4.0 GB | GC: 12 (3.2ms) ».
6. **périmé** — `docs/TUI_GUIDE.md:444` — *Claim* : référence croisée « ADR-004 (Observer and interface-based decoupling patterns) ». *Évidence* : ADR-0004 = backlog (FFTContext B1, SA6002 B2, golden B5...), aucune mention d'Observer. *Fix* : retirer l'attribution ADR-004 ou pointer vers la couche présentation (Observer via `orchestration.ProgressReporter`/`ResultPresenter`).

### 3.8 docs/algorithms/FAST_DOUBLING.md (34 claims, 5 issues)

1. **périmé** — `FAST_DOUBLING.md:159-162` — *Claim* : `DoublingFramework` = 2 champs (strategy, dynamicThreshold). *Évidence* : `doubling_framework.go:26-30` = 3 champs (+ `CacheStrategy CacheStrategy` exporté) ; type qualifié `*threshold.DynamicThresholdManager`. *Fix* : compléter le snippet.
2. **périmé** — `FAST_DOUBLING.md:178-180` — *Claim* : `CalculationState` = FK, FK1, T1, T2, T3 uniquement. *Évidence* : `fastdoubling.go:251-255` ajoute `arena *memory.CalculationArena` et `arenaCapWords int` (intégration state+arena P1-04). *Fix* : montrer les 7 champs ou marquer le snippet comme simplifié.
3. **périmé** — `FAST_DOUBLING.md:191-196` — *Claim* : `ReleaseStateWithResult` deep-copie et retourne l'état au pool. *Évidence* : depuis fa13bfd, `fd.releaseStateWithResult` (`:398-408`) appelle `finalizeStateReleaseTo(s, fd.cacheOrPool)` → slot `cachedState` d'abord (si arène ≤ `maxCachedArenaWords` = 4M mots), pool sinon ; le pool-only ne vaut que pour le chemin public et les grosses arènes. *Fix* : paragraphe sur le slot GC-immune + gain ~−12 % F(10M) (benchstat fa13bfd).
4. **périmé** — `FAST_DOUBLING.md:171` — *Claim* : calculateur « OptimizedFastDoubling ». *Évidence* : type inexistant ; réel `FastDoublingCalculator` (`fastdoubling.go:86`). *Fix* : renommer.
5. **périmé** — `FAST_DOUBLING.md:297` — *Claim* : `go test -bench=BenchmarkFastDoubling ...` (+ pattern `Benchmark(FastDoubling|Matrix|FFT)`). *Évidence* : aucune fonction ne matche ; benchmarks réels = `BenchmarkFibonacci`, `BenchmarkCacheImpact`, `BenchmarkFibonacciDTM`. *Fix* : corriger les deux commandes.

### 3.9 docs/algorithms/MATRIX.md (48 claims, 7 issues)

1. **périmé** — `MATRIX.md:121` — *Claim* : `type MatrixExponentiation struct{}`. *Évidence* : `matrix.go:45` = `MatrixExponentiationCalculator`. *Fix* : renommer le type et les récepteurs (lignes 123, 127).
2. **périmé** — `MATRIX.md:127` — *Claim* : `reporter ProgressCallback` (non qualifié). *Évidence* : `matrix.go:73` = `reporter progress.ProgressCallback` (package `internal/progress`). *Fix* : qualifier.
3. **périmé** — `MATRIX.md:159-172` — *Claim* : pseudocode Strassen classique (18 additions, P1 = A00*(B01−B11), C00 = P5+P4−P2+P6...). *Évidence* : `matrix_ops.go:65-113` implémente la **variante Strassen-Winograd** (7 multiplications, 15 add/sub — commentaire `:80`) ; mappings P1-P7 entièrement différents (p1=s2*s6, p2=m1.a*m2.a...). *Fix* : réécrire le bloc en Winograd (S1-S8, 7 multiplications, assemblage T1/T2) ; « 7 multiplications + 15 additions/subtractions ».
4. **périmé** — `MATRIX.md:178-227` — *Claim* : diagramme Mermaid de décomposition = Strassen classique. *Évidence* : `assembleStrassenResult` (`matrix_ops.go:144-166`) : C11=P2+P3, C12=T1+P5+P6 (T1=P1+P2), C21=T2−P7 (T2=T1+P4), C22=T2+P5. *Fix* : mettre à jour le diagramme en Winograd, ou le marquer explicitement « conceptuel (Strassen classique) ».
5. **périmé** — `MATRIX.md:299` — *Claim* : table de complexité, Strassen = 18 additions. *Évidence* : 15 (Winograd, `matrix_ops.go:80`). *Fix* : 15 + note de bas de table sur la variante.
6. **périmé** — `MATRIX.md:282` — *Claim* : commentaire `matrixState` « p1-p7, s1-s10 ». *Évidence* : `matrix_types.go:79-81` = s1 à s8 seulement. *Fix* : « s1-s8 ».
7. **non vérifiable statiquement** — `MATRIX.md:3` — *Claim* : dashboard « 797 nodes / 8 layers / 13-step tour ». *Évidence* : claim sur l'app web déployée (GitHub Pages) ; non vérifiable sans exécution. *Fix* : ancrer le chiffre à une date/un commit de génération ; vérification réelle en Phase 3 (§4, commande 2).

### 3.10 docs/algorithms/FFT.md (31 claims, 3 issues)

1. **périmé** — `FFT.md:142-148` — *Claim* : snippet `s := AcquireState()` + `defer ReleaseState(s)` + retour direct de `ExecuteDoublingLoop`. *Évidence* : `fft_based.go:53-65` = `AcquireStateForN(n)` (arène pré-dimensionnée), puis `ReleaseStateWithResult(s, raw)` (succès) / `ReleaseState(s)` (erreur). *Fix* : remplacer le snippet par l'implémentation réelle (`fft_based.go:49-65`).
2. **périmé** — `FFT.md:203` — *Claim* : `go test -bench=BenchmarkFFT ...`. *Évidence* : aucune fonction `BenchmarkFFT` ; chemin réel = `BenchmarkFibonacci/FFTBased/...` (`fibonacci_test.go:190-212`). *Fix* : `go test -bench='BenchmarkFibonacci/FFTBased' -benchmem -run='^$' ./internal/fibonacci/`.
3. **douteux** — `FFT.md:177` — *Claim* : crossover Karatsuba/FFT « ~1M bits » dans le diagramme. *Évidence* : contredit l'intro (`:10`, ~500k) et `constants.go:28` (`DefaultFFTThreshold = 500_000`) ; aucun sourcing du 1M. *Fix* : « ~500k bits », ou citer une mesure empirique si le crossover réel diffère.

### 3.11 docs/algorithms/BIGFFT.md (55 claims, 7 issues)

1. **périmé** — `BIGFFT.md:389-393,632` — *Claim* : deux implémentations d'allocateur (`PoolAllocator`, `BumpAllocatorAdapter`). *Évidence* : `allocator.go:81` — le wrapper `BumpAllocatorAdapter` a été supprimé ; `*BumpAllocator` implémente `TempAllocator` directement. *Fix* : retirer des deux tables.
2. **périmé** — `BIGFFT.md:623` — *Claim* : `fft_recursion.go` ~186 lignes. *Évidence* : 266 lignes (ajout `executeReconstruction`, butterflies parallèles, gate `1<<16`). *Fix* : ~266 + mentionner `executeReconstruction` dans la description.
3. **périmé** — `BIGFFT.md:626` — *Claim* : `fft_poly.go` ~523 lignes. *Évidence* : 619 lignes (ajout `runPointwise`, `:446-519`). *Fix* : ~619 + « parallel pointwise dispatch (runPointwise) ».
4. **périmé** — `BIGFFT.md:15` — *Claim* : ~19 fichiers, ~4 100 lignes. *Évidence* : 4 322 lignes au total. *Fix* : « approximately 4,300 lines ».
5. **périmé** — `BIGFFT.md:409-413` — *Claim* : formule de capacité `* 1.2 // 20% safety margin`. *Évidence* : `bump.go:255-257` = `total := (2*transformTemp + multiplyTemp) * 11 / 10` (marge réduite à 10 % après profiling). *Fix* : « * 1.1 // 10% safety margin (reduced from 20% based on profiling) ».
6. **douteux** — `BIGFFT.md:559` — *Claim* : expression AVX-512 `cpu.X86.HasAVX512F && HasAVX512DQ`. *Évidence* : `cpu_amd64.go:77` = `cpu.X86.HasAVX512F && cpu.X86.HasAVX512DQ` (préfixe manquant sur le second opérande dans le doc). *Fix* : ajouter `cpu.X86.`.
7. **périmé** — `BIGFFT.md:634` — *Claim* : `scan.go` ~88 lignes. *Évidence* : 98 lignes (`scanWithTemp` + garde F-016 chaîne vide). *Fix* : ~98.

### 3.12 docs/algorithms/GMP.md (26 claims, 3 issues)

1. **périmé** — `GMP.md:86` — *Claim* : `-bench='Benchmark(FastDoubling|GMP)'`. *Évidence* : `BenchmarkFastDoubling` inexistant ; réels = `BenchmarkGMPCalculator` (`calculator_gmp_test.go:118`) et `BenchmarkFibonacci` ; la branche FastDoubling matche 0 résultat silencieusement. *Fix* : `go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchmem ./internal/fibonacci/`.
2. **douteux** — `GMP.md:91-98` — *Claim* : overhead CGO 50-100 ns/appel ; crossover GMP ≈ N=1 000 000 ; avantage net N > 100M. *Évidence* : aucun fichier docs/audits/ ne mentionne GMP ; chiffres sans source/date/hardware ; fa13bfd (−12,3 % F(10M)) peut déplacer le crossover. *Fix* : bench daté → `docs/audits/bench-gmp-crossover-YYYY-MM.txt` cité par le doc, ou qualifier explicitement d'approximation non vérifiée (mesure proposée en §4).
3. **douteux** — `GMP.md:68-74` — *Claim* : exemple Go `calc.Calculate(ctx, progressChan, 0, 100_000_000, fibonacci.Options{})` compilable tel quel. *Évidence* : `progressChan` doit être `chan<- progress.ProgressUpdate` (import `internal/progress` requis, non montré) ; nil accepté (`calculator.go:147`). *Fix* : snippet complet avec import, ou passer `nil` comme dans les tests (`fibonacci_test.go:173`).

### 3.13 docs/algorithms/COMPARISON.md + PROGRESS_BAR_ALGORITHM.md (68 claims, 6 issues)

1. **périmé** — `COMPARISON.md:207` — *Claim* : `go test -bench='Benchmark(FastDoubling|Matrix|FFT)' ...` exécute les trois benchmarks. *Évidence* : aucune fonction de ces noms ; sous-tests réels `BenchmarkFibonacci/{FastDoubling,MatrixExp,FFTBased}` (`fibonacci_test.go:190-211`) ; la commande matche 0 benchmark. *Fix* : `go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem ./internal/fibonacci/` (identique à `make bench-versioned`).
2. **périmé** — `PROGRESS_BAR_ALGORITHM.md:295-298` — *Claim* : `go test -v -run TestCalcTotalWork ./internal/fibonacci/` et idem `TestReportStepProgress`. *Évidence* : ces tests vivent dans `internal/progress` (`progress_test.go:13,198`) ; la commande sortirait « no tests to run ». *Fix* : `go test -v -run 'TestCalcTotalWork|TestReportStepProgress' ./internal/progress/` (la commande `TestProgress` vers ./internal/fibonacci/ est correcte).
3. **périmé** — `PROGRESS_BAR_ALGORITHM.md:271` — *Claim* : « numBits = 10 → TotalWork ≈ 1 398 101 ». *Évidence* : (4^10−1)/3 = 349 525 ; 1 398 101 = (4^11−1)/3, soit numBits=11 — off-by-one ; les pourcentages dérivés sont cohérents avec 1 398 101 mais tous faux pour numBits=10. *Fix* : passer l'exemple à « numBits = 11 (e.g., n ~ 2,000,000) » (option la plus lisible) ou recalculer pour 10.
4. **douteux** — `COMPARISON.md:75` — *Claim* : Matrix Exp. = « 3 matrices + ~22 big.Int ». *Évidence* : `matrixState` (`matrix_types.go:76-83`) = 12 big.Int de matrices + p1-p7 (7) + s1-s8 (8) + t1-t5 (5) = 20 directs / 32 au total ; aucune lecture ne donne 22. *Fix* : « 3 matrices (res, p, tempMatrix) + 20 big.Int » ou « 32 au total (dont 12 de matrices) ».
5. **périmé** — `COMPARISON.md:73-74` — *Claim* : Fast Doubling = « 5 big.Int | CalculationState » (description complète du stockage). *Évidence* : fa13bfd ajoute `FastDoublingCalculator.cachedState` (slot GC-immune par instance, cap 4M mots ≈ 32 Mo) en plus du sync.Pool ; chemin préférentiel pour appels répétés (−12,3 % F(10M)). *Fix* : note « sync.Pool + per-instance GC-immune cache slot (~32 MB) » ; Matrix et FFTBased non affectés (FFTBased utilise `AcquireStateForN` globale).
6. **douteux** — `COMPARISON.md:84-88` — *Claim* : configuration de benchmark « Go: 1.25.0 ». *Évidence* : mesures historiques non traçables (pas de fichier docs/audits/ source) ; la note de provenance existe mais sans date ni lien. *Fix* : référencer un fichier d'audit brut, ou marquer « (source perdue, indicative seulement) ».

### 3.14 docs/architecture/ — README.md, component-diagram.mermaid, patterns/interface-hierarchy.mermaid, patterns/design-patterns.md (62 claims, 11 issues)

1. **périmé** — `patterns/interface-hierarchy.mermaid:9` — *Claim* : `+Calculate(ctx, n, opts)`. *Évidence* : `calculator.go:41` = 5 paramètres `(ctx, progressChan, calcIndex, n, opts)`. *Fix* : mettre à jour la signature.
2. **périmé** — `patterns/interface-hierarchy.mermaid:16` — *Claim* : `+CalculateCore(ctx, n, opts, subject)`. *Évidence* : `calculator.go:65` = `(ctx, reporter, n, opts)` (ordre différent, `reporter` pas `subject`). *Fix* : corriger.
3. **périmé** — `patterns/interface-hierarchy.mermaid:21-24` — *Claim* : `CalculatorFactory` = `Create` + `Available()`. *Évidence* : `registry.go:14-31` = 5 méthodes (`Create`, `Get`, `List`, `Register`, `GetAll`) ; `Available()` n'existe pas (réel : `List()`). *Fix* : remplacer par l'interface complète.
4. **périmé** — `patterns/interface-hierarchy.mermaid:116` — *Claim* : `CLIProgressReporter` à `presenter.go:18`. *Évidence* : type inexistant (3 hits grep, tous dans docs) ; `presenter.go:17` = `CLIResultPresenter` (implémente `ResultPresenter`) ; `DisplayProgress` = fonction libre (`display.go:61`). *Fix* : retirer du diagramme ; noter le wrapping `ProgressReporterFunc`.
5. **périmé** — `component-diagram.mermaid:71-75` + `interface-hierarchy.mermaid:161-163` — *Claim* : `KaratsubaStrategy` comme implémentation de `DoublingStepExecutor`. *Évidence* : aucune occurrence dans `internal/` ; implémentations réelles = `AdaptiveStrategy` (`strategy.go:91`) et `FFTOnlyStrategy` (`:122`) ; Karatsuba n'apparaît qu'en commentaire (algorithme interne de math/big). *Fix* : supprimer la classe des deux diagrammes.
6. **périmé** — `component-diagram.mermaid:160-168` — *Claim* : `DynamicThresholdManager` a `metrics []IterationMetric` et `ringPos int`. *Évidence* : `threshold/manager.go:108-128` = `buffer MetricsBuffer`, `analyzer ThresholdAnalyzer`, `iterationCount atomic.Int64`, `lastAdjustment atomic.Pointer[time.Time]`. *Fix* : mettre à jour les champs de la classe.
7. **douteux** — `README.md:11` vs `patterns/design-patterns.md:3` — *Claim* : 797 nœuds/3 533 arêtes vs 744 nœuds. *Évidence* : contradiction interne au même répertoire ; source JSON absente du dépôt, ni l'un ni l'autre vérifiable statiquement. *Fix* : synchroniser sur 797 (valeur la plus récente, cohérente avec ARCH.md/BUILD.md) + date de régénération ; trancher via §4, commande 2.
8. **lien cassé** — `README.md:16` — *Claim* : source à `.understand-anything/knowledge-graph.json`. *Évidence* : `git ls-tree HEAD` ne montre aucun répertoire `.understand-anything/`. *Fix* : régénérer (cf. docs/BUILD.md#dashboard-statique-github-pages) ou repointer vers l'emplacement réel.
9. **périmé** — `README.md:53-62` — *Claim* : table ADR 0000-0007. *Évidence* : `docs/adr/0008-audit-2026-06-rejected-candidates.md` existe. *Fix* : ajouter la ligne 0008.
10. **périmé** — `patterns/interface-hierarchy.mermaid:44-49` — *Claim* : interface `SequenceGenerator` à `generator.go:30`. *Évidence* : ni le fichier ni le type n'existent. *Fix* : supprimer l'entrée du diagramme.
11. **périmé** — `patterns/design-patterns.md:32-33` — *Claim* : « chaque Get de sync.Pool est apparié à un Put différé dans le même scope ». *Évidence* : faux depuis fa13bfd — `cachedState` retient des états entre appels sans retour au pool. *Fix* : reformuler (paire Acquire/Release + slot per-calculator pouvant retenir l'état, commit fa13bfd).

### 3.15 CHANGELOG.md + CONTRIBUTING.md (87 claims, 6 issues)

1. **périmé** — `CHANGELOG.md:404-406` — *Claim* : liens footer vers `github.com/agbru/fibcalc` avec tags v1.0.0/v0.1.0. *Évidence* : remote réel `https://github.com/agbruneau/FibGo.git` (go.mod:1 idem) ; tags existants : `v3.0.0`, `baseline-pre-refactor`, `archive/vague-A-bigfft-concurrency` — pas de v1.0.0/v0.1.0. *Fix* : `https://github.com/agbruneau/FibGo/compare/v3.0.0...HEAD` + `releases/tag/v3.0.0` (ou supprimer les anciens liens).
2. **douteux** — `CHANGELOG.md:24` — *Claim* : pointwise parallèle « −23 % à −35 % sur F(10M) selon l'algorithme ». *Évidence* : la source `docs/audits/bench-parallel-pointwise-2026-06.md` donne FastDoubling −27,6 %, FFTBased −22,9 %, MatrixExp −14,0 %, DTM/Off −34,8 %, DTM/On −24,6 % → plage réelle −14 % à −35 % ; chiffres en outre pre-fa13bfd. *Fix* : « −14 % à −35 % » ou détail par algorithme + mention pre-fa13bfd.
3. **périmé** — `CHANGELOG.md:8` ([Unreleased]) — *Claim* : section à jour. *Évidence* : commit fa13bfd absent (`cachedState`, `maxCachedArenaWords`, `finalizeStateReleaseTo`, `state_cache_test.go` ; benchstat : FastDoubling/10M −12,3 %, FFTBased/10M −10,2 %, MatrixExp/10M −25,3 %, geomean −7,96 %). *Fix* : ajouter l'entrée Performance correspondante.
4. **périmé** — `CHANGELOG.md:8` ([Unreleased]) — *Claim* : section à jour. *Évidence* : commit 4e34b82 absent (`testmain_test.go`, `TestMain` alignant zerolog sur InfoLevel, sortie bench parseable par benchstat — 45+ erreurs de parse éliminées). *Fix* : ajouter l'entrée Test/Infra.
5. **douteux** — `CONTRIBUTING.md:56-58` — *Claim* : `go test ./...` présenté comme équivalent de `make test`. *Évidence* : `make test` (`Makefile:162`) = `go test -v -race -cover ./...` ; l'alternative omet `-race`/`-v`/`-cover` (Directive 8 exige `-race`). *Fix* : `go test -v -cover ./...` + note « -race requiert CGO/gcc ; sous Windows sans gcc, utiliser make test-win ou WSL ».
6. **douteux** — `CONTRIBUTING.md:5` + `CHANGELOG.md:271` — *Claim* : knowledge-graph = 744 nœuds. *Évidence* : `docs/BUILD.md:3` dit 797 ; source JSON absente du dépôt ; trois documents incohérents entre eux. *Fix* : aligner sur la valeur régénérée (§4, commande 2) et documenter que l'artefact JSON n'est pas tracké.

### 3.16 docs/adr/0001-0008 + docs/external-reviews/2026-02-08-jules-self-evaluation.md (68 claims, 8 issues)

Convention de fix pour les ADR : ne pas réécrire le corps historique ; ajouter une **note de statut datée**.

1. **périmé** — `0001-dtm-decision.md:12` — *Claim* : `threshold/manager.go` ~283 L. *Évidence* : 333 lignes (CLAUDE.md déjà à jour). *Fix* : note « (333 L après audit 2026-06) ».
2. **douteux** — `0001-dtm-decision.md:59` — *Claim* : `profile.go:196-224` = justification chiffrée de la calibration. *Évidence* : ce range couvre la fin de `renameAtomic` et le prologue de `IsValid` (qui commence `:205`). *Fix* : corriger en `profile.go:205-245` ou référence symbolique `profile.go:IsValid`.
3. **périmé** — `0002-recover-strategy.md:9` — *Claim* : les quatre `recover()` dans `fft.go:41-101`. *Évidence* : `Mul` commence `:63` (defer/recover `:64`) ; `Mul`/`MulTo`/`Sqr`/`SqrTo` = `:63-144`. *Fix* : note « range actuel fft.go:63-144 ».
4. **périmé** — `0003-globals-vs-context.md:21` — *Claim* : globaux convertis en `atomic.Int64` + accesseurs privés `getParallelFFTRecursionThreshold()`/`getMaxParallelFFTDepth()`. *Évidence* : types réels `atomic.Uint64` (`fft_recursion.go:32,37`) ; accesseurs **exportés** `GetParallelFFTRecursionThreshold()`/`GetMaxParallelFFTDepth()` (`:46,49`). *Fix* : note de statut (la section Références ligne 51 est correcte).
5. **périmé** — `0003-globals-vs-context.md:11-13` — *Claim* : `fftThreshold` à fft.go:35, etc. *Évidence* : positions réelles fft.go:38, fft_recursion.go:32, fft_recursion.go:37 (décalage 3-4 lignes). *Fix* : note de statut datée.
6. **périmé** — `0004-backlog-decisions.md:91-92` — *Claim* : B5 = entrées F(100k), F(500k), F(1M) ajoutées via `cmd/generate-golden`. *Évidence* : le golden réel contient F(50000), F(100000), F(200000) — ni 500k ni 1M. *Fix* : note de statut « corpus réellement ajouté : F(50k/100k/200k) » ou amendement ADR.
7. **périmé** — `0007-pool-pointer-vs-value.md:10` — *Claim* : sites SA6002 `pool_warming.go:70,79,88,97`. *Évidence* : `.Put(` réels aux lignes 71, 80, 89, 98 (+1 partout). *Fix* : note de statut datée.
8. **périmé** — `2026-02-08-jules-self-evaluation.md:8` — *Claim* : « voir les ADRs (notamment ADR-0001 à ADR-0004) ». *Évidence* : 0005 à 0008 existent désormais (hardening mai 2026 + audit juin 2026). *Fix* : note d'en-tête « ADR également disponibles : 0005-0008 ».

---

## 4. Commandes à exécuter réellement en Phase 3

Liste consolidée et dédupliquée des vérifications impossibles statiquement (issues `douteux` non tranchables par lecture de code, claim `non vérifiable statiquement`, et exigence `-race` de la revue C2). Toute sortie servant de source documentaire doit être archivée dans `docs/audits/` avec date, commit et hardware.

1. **Race detector sur le protocole Swap/CAS de fa13bfd** — *bloquant avant merge* (CLAUDE.md Directive 8 ; exigé par les lentilles Concurrence et Conformité). Requiert CGO/gcc → WSL ou poste Linux/macOS :

   ```bash
   go test -race -count=1 ./internal/fibonacci/...
   # cibles minimales si run complet trop long :
   go test -race -count=1 -run 'TestCalculatorStateCache_ConcurrentCalls|TestArenaStateConcurrent|TestFFTRaceArenaAliasing_ConcurrentCalculateCore' ./internal/fibonacci/
   ```

   Archiver la trace (exigence explicite : « trace d'un passage -race WSL/Linux du package internal/fibonacci »).

2. **Régénération du knowledge graph + vérification du dashboard** — tranche le conflit 744 vs 797 nœuds (`docs/architecture/README.md:11`, `patterns/design-patterns.md:3`, `CONTRIBUTING.md:5`, `CHANGELOG.md:271`, `docs/BUILD.md:3`, `docs/algorithms/MATRIX.md:3`) et fixe la cible des 2 liens cassés `.understand-anything/knowledge-graph.json` (`docs/ARCH.md:5`, `docs/architecture/README.md:16`) :

   ```bash
   pnpm --filter @understand-anything/dashboard build:demo   # cf. docs/BUILD.md#dashboard-statique-github-pages
   jq '.nodes | length, .edges | length' docs/dashboard/knowledge-graph.json
   ```

   Pour le claim « 797 nodes / 8 layers / 13-step tour » du déployé : ouvrir <https://agbruneau.github.io/FibGo/dashboard/> (seule vérification possible de l'app live).

3. **Benchmarks frais sur HEAD** — réactualise `docs/PERFORMANCE.md:28` (2.1s/2.3s), `docs/algorithms/COMPARISON.md:84-88` (provenance Go 1.25.0, source perdue) et quantifie l'effet cumulé 1da5a6d + fa13bfd. Quirk hôte Windows : ne pas utiliser `-bench=.` sous PowerShell (mal parsé) ; préférer `-bench=Benchmark...` ou exécuter sous WSL :

   ```bash
   make bench-baseline > /tmp/new.txt
   benchstat docs/audits/bench-baseline.txt /tmp/new.txt
   # équivalent sans make :
   go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -count=5 -benchtime=1x -run='^$' ./internal/fibonacci/
   ```

   Optionnel : profiter du run pour vérifier empiriquement le crossover FFT (~500k bits selon `constants.go:28` vs « ~1M » du diagramme `FFT.md:177`).

4. **Mesure datée du cache FFT** — source (ou réfute) le « 15-30 % speedup » non sourcé (`docs/PERFORMANCE.md:263`, repris par CLAUDE.md), sur les chemins qui consultent réellement le cache (`FFTOnlyStrategy`, `bigfft.Mul/Sqr` directs) :

   ```bash
   go test -bench='BenchmarkCacheImpact' -benchmem -run='^$' ./internal/fibonacci/
   ```

   Sauvegarder dans `docs/audits/` (date + hardware) et citer depuis PERFORMANCE.md.

5. **Crossover GMP** — source les chiffres de `docs/algorithms/GMP.md:91-98` (overhead CGO 50-100 ns, crossover N≈1M, avantage N>100M ; aucun audit existant ne mentionne GMP). Requiert libgmp + CGO → WSL/Linux :

   ```bash
   go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchmem -run='^$' ./internal/fibonacci/
   ```

   Sauvegarder `docs/audits/bench-gmp-crossover-2026-06.txt` et le référencer depuis GMP.md ; sinon, marquer les chiffres « approximation non vérifiée ».

6. **Dry-run des commandes corrigées (validation des fixes Phase 4)** — vérifier que chaque commande de remplacement matche au moins un benchmark/test avant de l'écrire dans la doc :

   ```bash
   go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchtime=1x -run='^$' ./internal/fibonacci/
   go test -v -run 'TestCalcTotalWork|TestReportStepProgress' ./internal/progress/
   go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchtime=1x -run='^$' ./internal/fibonacci/   # si libgmp disponible
   ```

---

## Limites

- **Documents non couverts par l'inventaire de claims** : `README.md` (racine du dépôt) ; `Claude.md`/CLAUDE.md lui-même (sa dérive est traitée par la lentille Conformité de la revue C2, mais sans inventaire claim par claim) ; `promptAudit.md` (commit b2d448a) ; `docs/audits/*` (utilisés comme sources de vérité, non audités eux-mêmes) ; `docs/dashboard/*` (artefact généré, exclu par design) ; les commentaires godoc/`doc.go` des packages ; les fichiers de `docs/architecture/` autres que `README.md`, `component-diagram.mermaid`, `patterns/interface-hierarchy.mermaid` et `patterns/design-patterns.md`.
- **Vérifications impossibles statiquement** (reportées en §4) : contenu de l'app dashboard déployée sur GitHub Pages ; nombre réel de nœuds/arêtes du knowledge graph (source JSON absente du dépôt) ; chiffres de performance historiques (Go 1.25.0/Ryzen — données brutes perdues, seule une re-mesure est possible) ; chiffres GMP (overhead CGO, crossover) ; « 15-30 % » du cache FFT ; absence de data races du protocole Swap/CAS (exige `-race` sous CGO, indisponible sur cet hôte Windows sans gcc — délégation WSL documentée dans CLAUDE.md).
- **Granularité des compteurs** : les « claims vérifiés » par document proviennent d'agents vérificateurs distincts ; la définition d'un « claim » n'est pas strictement uniformisée entre rapports — les totaux sont indicatifs de couverture, pas une métrique normalisée.
- **Périmètre de la revue adversariale** : la revue C2 porte sur le seul commit `fa13bfd` ; les commits antérieurs de l'audit (ex. `1da5a6d`, `4e34b82`) n'ont pas été re-revus par ces trois lentilles.
- **Numéros de ligne** : les localisations citées (doc et code) sont celles du HEAD au 2026-06-10 (`fa13bfd`) ; tout commit ultérieur peut les décaler — les fixes de Phase 4 doivent re-vérifier les ancres avant édition.
