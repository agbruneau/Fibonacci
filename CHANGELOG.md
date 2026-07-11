# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Multiplicateur d'arène ×15 → ×10** (`acquireSizingForN`, `arenaTotalWords`) :
  adopté après balayage complet {12, 10, 8, 6} sur la machine de référence Intel
  (≈ −16 % B/op FFT 10M order-stable, CPU dans le bruit) — voir
  [addendum ADR-0009 R4](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md).
  Ceci remplace la conclusion « ×15 = charge utile intentionnelle » de la section
  Rejected de 4.0.0, vraie sur le Ryzen d'origine seulement (valeur
  microarchitecture-dépendante). Profil PGO (`cmd/fibcalc/default.pgo`) et
  baseline benchstat (`docs/audits/bench-baseline.txt`) régénérés (2026-07-07).
- `AnalyzeComparisonResults` écrit désormais les statuts d'échec et diagnostics
  sur un writer d'erreur dédié (stderr côté CLI) ; la table de comparaison reste
  sur stdout. Codes de sortie inchangés (audit Fable5 ERR-02).

### Added

- Gate GMP dans `scripts/check.sh` (étape 3b) : build + vet + test
  `-tags gmp -race`, dur quand les headers libgmp sont présents, SKIP sinon
  (2026-07-07).
- `FastDoublingMod` prend un `context.Context` et le re-vérifie à chaque
  itération de doublement : `--last-digits` honore désormais `-timeout` et
  Ctrl-C avec un dépassement borné à une itération (audit Fable5 ERR-03).

### Removed

- **API opt-in `FFTContext`** (`internal/bigfft/context.go`,
  `fft_recursion_ctx.go` : `NewFFTContext`, `Mul/MulTo/Sqr/SqrToWithContext`,
  `fourierRecursiveCtx` et leur plomberie, ~572 LOC prod + ~530 LOC tests) :
  zéro appelant de production, construite pour la migration classée WONT-FIX
  par ADR-0004 §B1. Décision mainteneur (audit Fable5 DEAD-01, addendum
  ADR-0004 §B1) ; récupérable de l'historique git si la migration renaît.

### Fixed

- `.gitattributes` épingle `*.sh` en LF (checkout CRLF cassait `check.sh` sous
  WSL avec `core.autocrlf=true`) (2026-07-07).
- Deux use-after-release dans les tests de pool de `internal/fibonacci`
  (`TestStateBump_FollowsArenaDrop`, sous-cas nominal de
  `TestReleaseState_OverLimit_AliasesCleared`) : l'état était publié dans le
  `statePool` partagé puis relu — re-correction des fixes de vague annulés par
  la restauration `6da3f3b` (audit Fable5 CONC-01/02).
- `WriteResultToFile` remonte désormais les erreurs d'écriture et de `Close` :
  sur disque plein, le CLI n'affiche plus « Result saved » avec exit 0 (audit
  Fable5 ERR-01).
- Erreurs de calcul et de budget mémoire écrites sur stdout au lieu de stderr
  (audit Fable5 ERR-02).

### Docs

- Exemples migrés de l'API supprimée `GlobalFactory`/`RegisterCalculator` vers
  `NewDefaultFactory` (7 documents) ; `BIGFFT.md` purgé des artefacts supprimés
  (`scan.go`, `fftState`, split arith) ; `dependency-graph.mermaid` resynchronisé
  avec le graphe d'imports réel (audit Fable5 DOCS-03/04, ARCH-01).

## [4.0.0] - 2026-07-07

> Note de versionnage : les tags `v2.x`/`v3.0.0` (2026-04-18) n'ont jamais reçu
> d'entrée de version dédiée dans ce fichier ; cette entrée 4.0.0 regroupe donc
> tout le travail livré depuis 1.0.0 (vagues d'audit 2026-04 → 2026-07). Bump
> majeur : codes de sortie CLI modifiés (timeout 2, SIGINT 130, divergence 3),
> nouveaux rejets de combinaisons de flags, exigence Go 1.26.

### Audit exhaustif 2026-07

Audit exhaustif de toute la base (rapport `audit.md`, purgé post-exécution au
commit `d10299b`) exécuté via son plan `auditPlan.md` (idem) en orchestration
multi-agents (modèle exécuteur
Sonnet, vérification adversariale par panels réfutateurs + gate manuel avant
chaque commit). Gate : `go build`/`go vet`/`go test ./...` verts, WSL `-race`
propre, couverture 95,2 % (plancher 80 %), golden intact, `benchstat` A/B sans
régression réelle sur le chemin critique (les écarts 1M mono-shot sont du bruit
thermique, prouvé par inversion d'ordre). Une recommandation (FIB-05) a été
**rejetée sur preuve de performance** — voir [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md).

#### Fixed

- **app — panic pointeur invalidé par le tri** : en mode comparaison avec `-o`,
  le meilleur résultat est copié avant `present()` (qui trie le slice en place),
  évitant un nil-deref dans `WriteResultToFile` (APP-01/M1).
- **app,tui — `--gc-control` inerte** : `GCMode` est câblé dans `Options` sur les
  deux chemins CLI et TUI (FIB-01/APP-02/M2).
- **cli/completion — flags fish manquants + valeurs bash erronées** : fish itère
  tout le registre (section « Other » catch-all) ; bash émet les valeurs propres
  de chaque flag ; garde `registre ⊆ script` étendue aux 4 shells (APP-03/APP-11/M3).
- **cli — data race sur `spinner.Suffix`** : `UpdateSuffix` fait Stop→write→Start
  (CONC-01/M4).
- **calibration — baseline non séquentielle + confiance gonflée** : candidat `-1`
  (réellement séquentiel via le garde `> 0`) ; confiance 0.0 sans mesure valide ;
  chemin de profil effectif affiché ; re-validation d'un profil forgé sur les
  deux chemins de confiance (SEC-01) (FIB-02/03/09/M5/M6).
- **bigfft — mémoire pool non initialisée + workers non attendus au panic** :
  `NTransform` utilise un slice zéroïsé ; les 4 sites parallèles attendent
  `wg.Wait()` inconditionnellement avant de re-propager un panic de la moitié
  synchrone (FFT-01/FFT-02/M7/M8).
- **memory — `ParseMemoryLimit` débordait** : multiplication saturante
  (FIB-04/SEC-02).
- **fibonacci — clamp cache + gate matriciel aligné sur `GOMAXPROCS(0)`**
  (FIB-07/FIB-08).
- **config — env `LAST_DIGITS`/`GC_CONTROL`, bool malformé bruyant, erreur typée**
  (APP-08/09/14).
- **tui — codes de sortie timeout/SIGINT (2/130), génération sur `IndicatorsMsg`,
  timeout par génération, rejet `--tui`+`--last-digits`/`--output`**
  (APP-04/06/05/07).
- **bigfft — release des buffers sur chemins d'erreur** (FFT-08/13).
- **app — erreurs de sauvegarde vers `ErrWriter`, borne `--last-digits`** (APP-16/SEC-03).
- **metrics — zero-padding des derniers chiffres corrigé** (APP-18).
- **gmp — build `-tags=gmp` réparé** : `globalFactory` déplacé dans
  `calculator_gmp.go` après la suppression de la fabrique globale.

#### Removed (code mort / sur-ingénierie, ~500 LOC)

- **ui** : `LightTheme`, `OrangeTheme`, `SetTheme` (inatteignables) (OVR-02).
- **tui** : chart braille mort (`RenderBrailleChart`/`brailleDots`/`plotBrailleValue`),
  paramètre `progress` ignoré d'`AddDataPoint` (OVR-09/APP-15).
- **errors** : `TimeoutError`, `ValidationError`, `WrapError`, `IsContextError` (OVR-07).
- **fibonacci/threshold** : `preSizeBigInt` dupliqué, wrappers DTM legacy +
  constructeur jumeau, `AcquireState`, `ShouldParallelizeMultiplication`,
  `setOrReturn`, `GlobalFactory`/`RegisterCalculator`, `MicroBenchPerTestTimeout`
  (OVR-03/04/05/06/FIB-10).
- **bigfft** : `scan.go` (OVR-01), machinerie `fftState`/`fftStatePool` (FFT-05),
  fusion `arith_amd64.go`+`arith_generic.go`→`arith.go` (FFT-06), alias
  `computeKey` (FFT-12) — cluster oracle **conservé** et documenté (OVR-10, ADR-0009).
- Exports test-only dé-exportés (OVR-12), `HexDisplayEdges`, champ `Indicators.Live`.

#### Changed

- **format→orchestration** : `ProgressState` déplacé ; 4ᵉ arrow interdit
  `orchestration → format` dans `TestArchitectureLayering` (APP-10).
- **Dockerfile** : `CGO_ENABLED=0` sans apt (toolchain CGO/libgmp mort retiré) (TOOL-03).
- **gate** : `make check` délègue à `scripts/check.sh` ; test+couverture fusionnés
  en une passe (TOOL-04/05). `docs/audits/bench-baseline.txt` régénérée et
  committée (M9/DOC-01/TOOL-01).
- **.golangci.yml** : blocs G304 morts + `dupl` retirés (TOOL-02).

#### Rejected

- **FIB-05 — réduction du multiplicateur d'arène ×15** : tentée puis abandonnée,
  régression benchstat mesurée (+18 % à +34 % sur F(10M), alloc-neutre). Le ×15
  est charge utile intentionnelle. Détail : [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md).

#### Docs

- Purge `cpu_amd64.go` (→ `internal/config/hardware.go`) et `tui/component` des
  docs ; doc-comments bigfft/config/parallel corrigés ; CLAUDE.md synchronisé
  (invariants Phases 1-5, gardiens, décision FIB-05). `govulncheck` (recompilé
  go1.26) : 0 vulnérabilité atteignable.

### Audit Go exhaustif (2026-06-24)

Revue Go exhaustive et vérifiée (Claude Opus 4.8, orchestration multi-agents, vérification
adversariale de chaque trouvaille moyenne/haute). Gate : `go build` / `go vet` / `go test ./...`
verts, `gofmt` propre, couverture 95,0 % (plancher 80 %), aucune nouvelle alerte `golangci-lint`
introduite. Chemin critique T1.1 validé sans régression (`benchstat` A/B, cas 1M-mots p=0,72).
Détail complet et déviations assumées : voir les entrées ci-dessous (le
document de plan `AuditPlan.md` associé à cette passe a depuis été supprimé
du dépôt).

#### Fixed

- **bigfft — panic de récursion FFT parallèle** (`fft_recursion.go`, `fft_recursion_ctx.go`) : les
  goroutines async capturent désormais les panics worker (`panicCh` + re-panic après `wg.Wait()`)
  au lieu de crasher le process sur une goroutine nue (ADR-0002). Gardé par
  `TestFourierRecursiveAsyncPanicPropagates` / `...CtxAsyncPanicPropagates`.
- **app — `--algo all --quiet` masquait une divergence** : le mode quiet vérifie la cohérence
  inter-algorithmes et renvoie `ExitErrorMismatch` (3) au lieu d'un résultat faux avec exit 0
  (helper partagé `orchestration.HasResultMismatch`).
- **tui — corruption d'état après *Restart*** : `ProgressMsg`/`ComparisonResultsMsg`/`FinalResultMsg`/
  `ErrorMsg` portent une `Generation` ; les messages d'une génération obsolète sont ignorés.
- **memory — restauration de `GOMEMLIMIT`** : `GCController.End()` restaure la limite mémoire
  d'origine au lieu de la remettre à « illimité ».
- **errors — troncature UTF-8** : `sanitizeConfigExcerpt` coupe sur une frontière de rune.
- **cli/completion** : 6 flags manquants ajoutés au registre (test de sync contre `config.FlagNames`) ;
  escaper dédié zsh `_arguments` neutralisant `:` `[` `]`.
- **orchestration** : un ensemble de résultats vide renvoie un code d'échec (plus d'exit 0 silencieux).
- **cmd** : erreurs de configuration → `ExitErrorConfig` (4) ; le générateur golden remonte l'erreur
  de `Close()` et échoue vite hors de la racine du module.
- **modular** : suppression d'une branche de correction de modulo négatif inatteignable (Mod euclidien).

#### Removed (code mort)

- Sous-système `internal/bigfft/cpu_amd64.go` (détection CPU sans consommateur de production, doublon
  de la détection de `internal/config`) et ses tests dédiés.
- `internal/tui/component` (commit `b87a342`) : package ne contenant qu'un `doc.go` avec une
  interface `Component` jamais implémentée ni importée (placeholder de refactor jamais réalisé) ;
  zéro importateur confirmé par `go list`.
- 5 helpers `getEnv*` (`config/env.go`), `NewMatrixFrameworkWithSquareFunc`, setter `SetTaskLogger`,
  déclaration `go:linkname mulAddVWW` — tous sans appelant de production.

#### Changed

- `findHighestBit` → `bits.Len64` (backend `gmp`).
- `threshold.ShouldAdjust` : un seul snapshot du buffer de métriques par ajustement (un lock + une
  copie en moins).
- `calibration` : annulation de contexte honorée avant le warm-up multiply.
- Renommage `Claude.md` → `CLAUDE.md` (découvrabilité Claude Code sous système de fichiers sensible à la casse).

#### Docs

- README : bannière d'audit 2026-06-24 ; plancher de couverture délié (directive A5-04).
- `bigfft/doc.go` (claims SIMD/CPU obsolètes), `config/doc.go` (dépendance `internal/ui`),
  `BIGFFT.md` (recomptage, retrait de `cpu_amd64.go`) ; hash de provenance du dashboard réconcilié
  sur l'artefact généré (`f4d3a7f`).

### Audit loop (2026-06-10)

Performance and coverage audit loop (multi-agent workflow, branch
`refactor/audit-loop-2026-06`). Benchmark figures below: Windows 11,
Intel Core Ultra 9 275HX (24 threads), Go 1.26.4, `benchstat` n=6 —
cumulative results and rejected candidates are summarised below. Full
`-race` run validated under WSL (go1.26.0 linux/amd64). The non-regression
baseline is regenerated on demand via `make bench-baseline`.

#### Housekeeping

- **Purge des artefacts d'audit** — suppression de `promptAudit.md`, du
  répertoire `docs/audits/` (rapports et baselines datées, régénérables via
  `make bench-baseline` / `BenchmarkFibonacciDTM`) et de
  `docs/external-reviews/`, ainsi que des snapshots de travail non suivis à la
  racine (logs, profils, `*.out`, `*.test`, `ruvector.db`). Les références
  documentaires correspondantes ont été redirigées vers ce CHANGELOG.

#### Performance

- **Per-calculator state+arena cache** (`fa13bfd`) — the GC disable/re-enable
  pattern forces a collection after every calculation ≥ 1M, which purges
  `sync.Pool`: `statePool` never retained the arena between calls (~46 % of
  allocations at F(10M) were arena recreation; mem profile 3.89 GB / ~155 ops).
  A GC-immune single-state slot per `FastDoublingCalculator` instance
  (`cachedState`, `atomic.Pointer`) with `sync.Pool` fallback now retains it,
  bounded by `maxCachedArenaWords` (4M words, ~32 MB); the single teardown
  path is preserved (`finalizeStateReleaseTo`, order checkLimit →
  clearStateAliases → sink, over-limit states never reach the sink).
  benchstat (commit message, n=6, p=0.002): FastDoubling/10M
  33.30 ms → 29.22 ms (−12.3 %, B/op −45.4 %), MatrixExp/10M −25.3 %,
  FFTBased/10M −10.2 %, DTM Off/On 10M −16.8 %/−16.3 %, geomean sec/op
  −7.96 %. 5 guard tests added (`state_cache_test.go`).
- **FFT bump scratch acquired once per calculation** (`7999c39`, implements
  F-012 from the 2026-05-29 audit) — the forward-transform `BumpAllocator`
  was acquired/released at every doubling step and regrown almost every
  iteration (~25 % of the residual allocations after the arena fix). It is
  now sized for the final step (`s.fftBumpCapWords`), carried by the
  `CalculationState`, and `Reset()` between steps; retention follows the
  arena anti-bloat policy. benchstat (commit message, n=6): geomean sec/op
  −4.42 %, DTM Off/On 10M −6.9 %/−6.7 % (p=0.002), fast-doubling B/op −46 %
  (13.57 → 7.25 Mi), no regression ≥ 5 %. +3 bump guard tests.
- **Cumulative effect** (2026-06-10, the two commits above vs the same-day
  `make bench-baseline`) — geomean sec/op **−12.0 %** vs that baseline;
  FastDoubling/10M 33.30 ms → 28.20 ms (−15.3 %); B/op ~**−70 %** cumulative
  at 10M (−61 % at 1M). No significant sec/op regression.

#### Fixed

- **threshold — data race in the dynamic threshold manager** (`a2e4eee`) —
  the first full `-race` pass of the module (WSL) caught a real race:
  `MetricsBuffer.Record` (write, via `RecordIteration`) vs
  `MetricsBuffer.Count` (read, via `GetStats`). All `MetricsBuffer` accesses
  now go through the manager's existing mutex (`snapshotMetrics` copies under
  lock, analysis outside); thresholds/counters stay atomic and the A2-04
  package-level knobs are untouched. Targeted 10M benchmarks neutral
  (p>0.05).
- **TUI keymap** (`7b97d48`) — the help label of the `r` key said "Reset"
  while `footer.go` and `docs/TUI_GUIDE.md` say "Restart"; label aligned on
  "Restart".

#### Added

- **Benchmark output hygiene** (`4e34b82`) — `internal/fibonacci` tests gain
  a `TestMain` (`testmain_test.go`) aligning the global zerolog level on
  `InfoLevel` (the production level set by `app.New`); trace-level JSON log
  lines no longer interleave with benchmark result lines, which had made the
  bench output unparseable by `benchstat` (45+ parse errors eliminated).
- **Test repair** (`9cad06e`) — `TestSetTransformCacheConfig` built malformed
  `PolValues` that the A-05 shape guard silently rejected; the test only
  passed through global-cache pollution from neighbouring parallel tests and
  failed deterministically in isolation. Fixed with a conformant mock
  (coefficients of N+1 words) and `t.Parallel` removed from the two subtests
  mutating the global singleton.
- **Coverage wave (Phase 2)** — total coverage 88.9 % → **95.0 %** across
  9 commits (`c306344`..`d549da4`; Phase 2 close `83e3404`): bigfft
  85.3 → 95.9 %, tui 87.8 → 97.3 %, memory 87.9 → 99.1 %, orchestration
  88.0 → 99.1 %, app 88.6 → 94.9 %, config 88.6 → 99.6 %, errors
  89.2 → 100 %, cmd/fibcalc 75.0 → 91.7 %, cmd/generate-golden
  28.6 → 88.6 %. Unreachable paths are documented as exceptions; the golden
  corpus is untouched.

#### Docs

- **Rejected optimization candidates** (`cfb4ff2`, Phase 1 close) — documented
  with evidence: `fermat.Shift` `shlVU` guard refuted by reading `math/big`;
  forward-transform parallelization set aside. The dated benchmark baseline
  that backed them (`docs/audits/bench-audit-loop-2026-06.md`) was purged with
  the rest of `docs/audits/` (see Housekeeping above); regenerate via
  `make bench-baseline`.
- **`internal/cli/completion`** (`ed6c334`) — dedicated `doc.go` per project
  convention (package comment moved out of `registry.go`, enriched with the
  shell-escaping note).

### Refactoring audit (2026-06-09)

Audit de refactorisation exhaustif (exploration multi-agents, vérification
par grep/tests avant chaque modification, branche
`refactor/audit-deepening-2026-06`). Les candidats rejetés après
vérification sont matérialisés dans ADR-0008.

#### Performance

- **bigfft** — Les K produits de coefficients fermat des phases pointwise
  (`PolValues.Mul`/`Sqr`) et les butterflies de `executeReconstruction`
  sont désormais répartis sur les cœurs pour les grandes transformées
  (gate 64k mots ; sémaphore FFT global, acquisition non bloquante,
  scratch pool par worker — le bump allocator reste mono-goroutine).
  Mesuré (médianes appariées, hôte 24 threads, 2026-06-09 — chiffres
  antérieurs aux optimisations `fa13bfd`/`7999c39` du 2026-06-10 ci-dessus) :
  **−14 % à −35 %** sur F(10M) selon l'algorithme (FastDoubling −27,6 %,
  FFTBased −22,9 %, MatrixExp −14,0 %, DTM Off/On −34,8 %/−24,6 %),
  **−46 %** sur le calcul seul de F(100M)
  (0,379 s → 0,204 s). Aucune régression à F(1M) ; chemin séquentiel
  strictement préservé sous le gate. Détails :
  `docs/audits/bench-parallel-pointwise-2026-06.md`. Les panics de
  workers sont re-propagées dans la goroutine appelante (politique
  ADR-0002 inchangée) — gardé par `TestPointwiseWorkerPanicPropagates`
  et `TestPointwiseParallelMatchesSequential`.

#### Fixed

- **A-05** — `bigfft.TransformCache.putByKey` rejette désormais les
  `PolValues` dont les coefficients ne font pas exactement `n+1` mots
  (une entrée malformée était silencieusement tronquée par `copy`,
  servant ensuite une transformée corrompue). Fix porté depuis le tag
  `archive/vague-A-bigfft-concurrency` ; garde
  `TestPutByKeyRejectsMalformedShape`.
- **ADR-0002** — Les 4 variantes `*WithContext` de `bigfft` convertissaient
  TOUTES les panics en erreur, y compris les sentinelles post-condition de
  `fermat.go` qui doivent se propager. La politique de panic est extraite
  dans un helper unique (`fermatPanicToError`) partagé par les 8 points
  d'entrée publics.
- **A2-04** — Le câblage documenté `config.DefaultThresholdTuning →
  threshold.SetTuning` n'était exécuté nulle part (zéro appelant de
  production). Réalisé dans `app.New` derrière un `sync.Once` ;
  comportement inchangé aujourd'hui (valeurs identiques), canal de dérive
  fermé.

#### Changed

- **API interne** — Suppression de la surface morte vérifiée :
  `metrics.MemoryCollector`/`MemorySnapshot` (module entier sans
  consommateur), stubs pass-through `calibration.EstimateOptimal*`,
  dé-export de `config.estimate*ThresholdForHeuristic` (×3).
  `CalibrationN` déménage de `fibonacci/constants.go` vers
  `internal/calibration` (son seul domaine consommateur).
- **fibonacci** — Les heuristiques grow/shrink du cache FFT sont extraites
  en fonction pure `decideCacheTuning`, testée unitairement ; le seam
  `CacheStrategy` est conservé (deux adapters — cf. ADR-0008 R3).

#### Added

- **Couverture** — `TestMatrixExponentiation_FFTPathIsolation` (le chemin
  matrix × FFT n'était exercé par aucun test aux seuils par défaut),
  `TestDecideCacheTuning`, `TestFormatBytes`, bornes e2e `--last-digits`
  et contrat e2e des overrides d'environnement `FIBCALC_*`,
  `TestWireThresholdTuning`.

#### Docs

- **ADR-0008** — Sept candidats de refactorisation rejetés après
  vérification (ColorProvider, executeTasks, seam CacheStrategy,
  observers progress, exports bigfft, TestFactory, knobs threshold),
  avec preuves, pour que les audits futurs ne les re-suggèrent pas.

### Audit exactness pass (2026-05-29)

Audit de suivi (23 constats `F-NN`, rapport archivé
`docs/audits/audit-2026-05-29.md`) ciblant correctness fine, durcissement
sécurité et angles morts de couverture. 12 constats corrigés ; le reste différé
ou report-only.

#### Security

- **F-014** — Les 4 générateurs de complétion shell (bash/zsh/fish/powershell)
  échappent désormais `f.Help` et `f.Values` vers le shell (durcissement
  injection ; helpers `escape*` dans `internal/cli/completion/escape.go`).

#### Fixed

- **F-016** — `bigfft.FromDecimalString` rejette désormais une entrée décimale
  malformée (`SetString` tronquait silencieusement) ; le contrat `""` → 0 est
  préservé.
- **F-022** — `internal/errors` : la constante locale `max` (shadowing du builtin)
  renommée `maxLen`.

#### Added

- **Couverture** — Nouveaux tests de garde : F-001 (correction DTM-on de
  `FastDoubling`), F-006 (branche oversize de `releaseMatrixState`), F-007
  (seam `threshold.SetTuning`), F-008 (fallback d'épuisement d'arène
  `preSizeBigInt`).

#### Docs

- **F-015** — Commentaire d'identité erroné corrigé dans `executeDoublingStepFFT`.
- Corrections d'exactitude `CLAUDE.md` / baseline (versions, comptes, renvois)
  alignées sur le code réel.

### Audit remediation (May 2026)

Remédiation des 45 constats de l'audit multi-agents (rapports d'audit archivés
en historique git ; plan et décisions consolidés dans les ADR 0005-0007 + ce
changelog). Chaque constat est soit corrigé, soit
acté comme décision documentée. Nouveaux ADR : 0005 (GC concurrent), 0006
(annulation FFT), 0007 (SA6002 pool).

#### Fixed

- **A2-01 (CRITIQUE)** — Contrôle GC *concurrency-safe* : refcount package-level
  (`gc_control.go`). En mode `--algo all`, seul le 1ᵉʳ `Begin` actif désactive le
  GC / mémorise le vrai `GOGC`, seul le dernier `End` restaure et lève la limite
  mémoire. Invariant `WithGC` panic-safe préservé (ADR-0005).
- **A2-02 (MAJEUR)** — `TransformCache.logger` → `atomic.Pointer[zerolog.Logger]`
  (écriture `SetCacheLogger` / lecture hot path `logPeriodicStats`).
- **A5-01 (MAJEUR)** — Flaky `TestSaveProfile` sous Windows : `renameAtomic`
  backoff exponentiel borné (10→40 tentatives) + tolérance de l'erreur de partage
  côté écrivain dans le test.
- **A1-04 / A1-05 (MINEUR)** — Arithmétique saturante (`EstimateMemoryUsage`) et
  clamp float→int + ×15 (`AcquireStateForN` / `NewCalculationArena`) contre les
  débordements à `n` non borné.
- **A2-05 (MINEUR)** — `releaseFFTState` relâche les buffers `tmp/tmp2` au-delà de
  `maxPooledFFTTmpCap` (anti-rétention de pic).
- **A3-03 (MINEUR)** — `FFTOnlyStrategy.Multiply/Square` écrivent dans `z` via
  `bigfft.MulTo/SqrTo` (suppression alloc neuve + copie O(n)).
- **Correctness/tests** — A1-01 (cross-val régime FFT F(1M) + oracle
  `FastDoublingMod`), A1-02 (cross-val GMP, build tag `gmp`), A1-03 (oracle dans
  `FuzzFastDoublingMod`), A1-08 (assertions T1/T2/T3), A1-06 (métrique `usedFFT`
  alignée sur FK1).
- **Idiomatique** — A4-02 (dead code `formatAlgoList`), A4-03 (`RenderBrailleChart`
  cyclo 17→9), A4-04 (dead store `fermat.norm`), A4-05 (récepteur `fermat`),
  A4-06 (`run() error` generate-golden), A4-08 (octal `0o`, `if`→`switch`,
  `paramTypeCombine` ; `bash.go` `%q` *non* appliqué — annotation), A4-13
  (`#nosec G304`).
- **Structure/doc** — A5-03 (versions Go 1.26.0/toolchain 1.26.3), A5-06 (path
  module `github.com/agbruneau/FibGo`), A5-08/A5-09 (angles morts de couverture
  documentés).

#### Changed

- **A5-02 (MAJEUR) — décision assumée** : pas de CI distante réintroduite.
  Garde-fous **locaux** ajoutés à la place : cibles `make test-win` /
  `make coverage-check` (plancher 80 %) et `scripts/check.ps1` / `scripts/check.sh`.
- **A5-05 / A5-10** — `-race` documenté comme requérant CGO/WSL ; plancher de
  couverture ré-outillé localement.
- **A4-07 / A4-09 / A4-10 / A5-07** — `.golangci.yml` : exclusion *stutter* revive
  ciblée ; `misspell.locale` maintenu **US** (la mesure a infirmé la prémisse
  « britannique cohérent » : ~339 US vs ~50 UK) avec normalisation des ~50
  graphies UK résiduelles ; `.gitattributes` `*.go text eol=lf` (faux positifs
  gofmt CRLF) ; verrou schéma golangci-lint v1 documenté.
- **A3-01 / A3-04 / A3-06** — Documentation perf corrigée : le cache de
  transformées FFT ne bénéficie qu'aux chemins `bigfft.Mul/Sqr` / `FFTOnly` (pas
  au calculateur Fast Doubling par défaut) ; prudence sur l'ordre des algos à
  très grand N ; domination Karatsuba sous le seuil FFT.

#### Decisions (no-fix documenté)

- **A2-03 (MAJEUR)** — Annulation fine *intra*-multiplication FFT **reportée** au
  token par-appel (`FFTContext`, ADR-0004 §B1) ; le drapeau atomic global est
  **rejeté** (clear-race sous FFT concurrentes, ADR-0006). L'annulation grossière
  existante (entre pas + entre les 3 produits) reste fonctionnelle.
- **A4-01 (MAJEUR)** — SA6002 : micro-benchmark (ADR-0007) prouve que le fix
  préservant la signature est **alloc-neutre** ; le vrai zéro-alloc exige le
  refactor `FFTContext`. Exclusion SA6002 ciblée + ADR.
- **A2-04** (knobs `threshold` single-writer-before-use), **A2-06** (LRU correct
  par construction), **A2-07** (deux sémaphores `NumCPU`), **A3-02** (clé cache
  O(n) gatée par `MinBitLen`), **A3-05** (sous-goroutines FFT sur pool — compromis
  sécurité concurrence), **A3-07** (baseline conservatrice), **A4-11**
  (annotations mathématiques conservées), **A4-12** (shadow `err` hot path vérifié
  bénin) — commentés ou documentés sans changement fonctionnel.

#### Deferred verification (indisponible sur l'hôte Windows `CGO_ENABLED=0`)

- `-race` : A2-01/A2-02/A2-06/A4-04/A4-12 — rejouer `CGO_ENABLED=1 go test -race`
  sous Linux/WSL.
- `-tags gmp` : A1-02 — `CGO_ENABLED=1 go test -tags gmp ./internal/fibonacci/`
  avec `libgmp-dev`.
- `ns/op` fiables : A3-04/A3-07 — machine de référence Ryzen/Linux idle.

### Hardening sprint (May 2026)

Architecture / concurrency / security hardening sweep documented in
`docs/adr/0001`..`0004`. Net effect : closed every data race the dual-
audit (Claude + Gemini) had flagged, broke three upward import arrows
gated by `internal/arch_test.go`, restored panic propagation for FFT
post-condition violations, lifted `internal/cli/completion/` coverage
from 0 % to 95.7 %, and added the multi-stage `Dockerfile` /
`.devcontainer/` for reproducible builds. Working audit documents have
been purged ; the ADR series is the surviving source of truth.

### Added (hardening)

- **Concurrence safe** : `DynamicThresholdManager` fields converted to
  `atomic.Int64` / `atomic.Pointer[time.Time]` ; `bigfft` globals
  (`fftThreshold`, `parallelFFTRecursionThreshold`,
  `maxParallelFFTDepth`) routed through atomic accessors (ADR-0003) ;
  `TransformCache.cacheGate()` snapshot under RLock at 5 hot-path sites.
- **Cache FFT** : `putByKey` allocates fresh backing on eviction — no
  recycle — eliminating the use-after-free aliasing window for
  concurrent `Get()` holders.
- **FFT panic policy** : sentinel `isFermatPostConditionPanic` re-
  propagates internal invariant violations instead of masking them as
  opaque errors (ADR-0002).
- **CLI completion security** : per-shell escape helpers
  (`escapeBashDoubleQuoted`, `escapeFishSingleQuoted`,
  `escapeZshSingleQuoted`, `escapePowerShellSingleQuoted`) plus
  9 adversarial tests per shell covering `$(...)`, backticks, `;`,
  spaces, quotes, backslashes, newlines.
- **Reproducible build** : multi-stage `Dockerfile`
  (`golang:1.26-bookworm` builder + distroless runtime) and
  `.devcontainer/devcontainer.json` (VS Code, CGO + libgmp +
  benchstat pre-installed).
- **Architecture gate** : `internal/arch_test.go` blocks PRs that
  reintroduce `threshold → config`, `errors → format`, or
  `tui → fibonacci`.
- **Perf regression gate** : `.github/workflows/ci.yml` `bench` job is
  blocking ; `benchstat` + `.github/scripts/bench_gate.py` compare each
  PR to `docs/audits/bench-baseline.txt` at a 5 % threshold.
  `make bench-baseline` refreshes the file.
- **Cross-compile job** : CI builds `linux/arm64`, `darwin/arm64`,
  `darwin/amd64` on every PR.
- **Fuzz coverage extended** : new `bigfft.FuzzMul` / `FuzzSqr` targets
  validate against `math/big` ; Fibonacci fuzz bounds raised
  50 000 → 200 000 to exercise the FFT regime.
- **Golden corpus extended** : `internal/fibonacci/testdata/fibonacci_golden.json`
  gained F(50 000), F(100 000), F(200 000) under ADR-0004 §B5.
- **Observer counter** : `progress.RecoveredObserverCount()` exposes
  swallowed panics from frozen `ProgressCallback`s.
- **Documentation** : `docs/PORTABILITY.md` (matrix + fallbacks),
  `docs/adr/0001..0004`, enriched `doc.go` for `cli`, `config`,
  `orchestration`, `fibonaccitest`, and a `make stats` target as the
  canonical source for package & LOC counts.

### Removed (hardening)

- Working audit drafts at repo root (`Audit - Claude - FibGo.md`,
  `Audit - Gemini - FibGo.md`, `Audit - Global - FibGo.md`,
  `Audit - Global - FibGo - v2.md`, `Audit - PRD - FibGo.md`,
  `Audit - PRDPLan - FibGo.md`) ; the durable record now lives in
  `docs/adr/` and this CHANGELOG.
- Pre-hardening bench baseline `docs/audits/bench-baseline-pre-hardening.txt`
  (superseded by `docs/audits/bench-baseline.txt`) and the legacy
  `bench-A10-*.txt` artifacts from the previous audit cycle.

### Changed

- **Post-audit remediation period closed.** The audit-driven remediation workflow (A-NN finding tracking, "Vague A" review freeze, post-audit branch gating) is wound down. Project documentation is purged of audit scaffolding: contributor-facing docs no longer route through `audit.md` / `AuditPlanning.md` / `audit-prompt.md`, and the standard `CONTRIBUTING.md` workflow is the single source of process truth going forward. Historical CHANGELOG entries below are retained verbatim as the accurate record of what was done during remediation.
- **Project status repositioned** from `Production-Ready` to `Academic Prototype` (README status badge). Aligns the marketing claim with the actual self-description in the README intro and removes the implicit promise of automated quality gates that the CI removal (below) makes untenable.

### Removed (CI/CD retirement)

- **GitHub Actions workflows deleted** : `.github/workflows/ci.yml` (vet + golangci-lint + 3-OS race matrix + cross-compile + bench gate) and `.github/workflows/coverage.yml` (PR/push coverage with `MIN_COVERAGE=80%` floor). The accompanying regression-gate engine `.github/scripts/bench_gate.py` and the now-empty `.github/` tree are gone as well.
- **Consequence for contributors** : every gate that used to run on push/PR is now the contributor's local responsibility — `make test` (race, requires CGO/gcc), `make lint`, `make coverage`, `make build-all` for cross-compile, and `benchstat` against `docs/audits/bench-baseline.txt` for perf-sensitive changes. CLAUDE.md directive #8 is the canonical checklist.
- **Documentation refresh** : README/CLAUDE/PORTABILITY/PERFORMANCE/TESTING/BUILD/architecture docs reworded to drop references to `ci.yml`/`coverage.yml` and to reframe gates as local-discipline conventions (no behavioural change to the Go code).

### Added

- **Interactive knowledge-graph dashboard** published on GitHub Pages: <https://agbruneau.github.io/FibGo/dashboard/>. 797 nodes / 3 533 edges / 8 architectural layers / 13-step guided tour (figures from the 2026-06 regeneration, verified against the tracked `docs/dashboard/knowledge-graph.json`), generated from `.understand-anything/knowledge-graph.json` via the `understand-anything` plugin and bundled into `docs/dashboard/` as a static Vite build — the `.understand-anything/` source JSON is not tracked in git; only the `docs/dashboard/` bundle is. Republish steps documented in [docs/BUILD.md — Dashboard statique (GitHub Pages)](docs/BUILD.md#dashboard-statique-github-pages).
- **Interactive TUI mode**: btop-style dashboard built with Bubble Tea (Elm architecture), featuring real-time progress charts, algorithm comparison, and keyboard navigation
- Portable arithmetic fallback for non-amd64 architectures (`arith_generic.go`)
- Godoc example functions for `Calculator`, `DefaultFactory`, and `CalculateWithObservers`
- `doc.go` for every internal package; enriched `internal/bigfft`, `internal/calibration`, `internal/app`, `internal/parallel`, `internal/testutil` with role/invariants/example comments (audit P2-13, P2-18)
- Cross-compilation targets for Linux and Windows **arm64** in the Makefile
- `.env.example` coverage for `FIBCALC_TUI_THEME`, `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT` (audit P1-13, P1-14)
- `docs/architecture/patterns/design-patterns.md` inventory of concrete design patterns in use (audit P2-21)
- **Arena pooling**: `AcquireStateForN(n)` / `ReleaseStateWithResult(s, src)` API in `internal/fibonacci/` — `CalculationState` now owns its `CalculationArena`, the two pools share a single lifecycle, and all big.Int slot aliases are detached before `sync.Pool.Put` to keep the arena race-free for reuse (audit P1-04)
- New regression tests: `TestArenaPoolingNoAliasing`, `TestArenaStateConcurrent` (16 goroutines × 8 iters × 3 sizes), `TestStateReuseAcrossSizes` in `internal/fibonacci/state_pool_arena_test.go`

### Changed

- **Go toolchain**: bumped `go.mod` to Go 1.25 (toolchain go1.26.2) — audit P0-02
- **Dependencies**: minor/patch upgrades for `golang.org/x/sync`, `x/sys`, `x/term`, `x/text`, `github.com/rs/zerolog`, and `gopsutil` (audit P1-24)
- **Dependencies (major)**: bumped `github.com/charmbracelet/bubbles` from `v0.21.1` to `v1.0.0`. The bubbles v1.0 release preserved the `key` and `viewport` sub-package surfaces actually used by the TUI; zero source changes were required (audit P0-03)
- **Package restructuring**: Extracted `internal/progress/` package from `internal/fibonacci/` (observer pattern, progress types); backward-compatible type aliases in `progress_aliases.go`
- **Package restructuring**: Extracted `internal/fibonacci/memory/` sub-package (arena, GC control, memory budget)
- **Package restructuring**: Extracted `internal/fibonacci/threshold/` sub-package (dynamic threshold manager)
- Extracted `internal/app/calculate.go` — calculation dispatch logic from `app.go`
- Extracted `internal/config/thresholds.go` — adaptive threshold estimation (canonical implementation)
- Added `internal/orchestration/progress.go` — `ProgressAggregator` for multi-calculator progress
- Dependency injection: `app.New()` accepts `WithFactory()` option for custom `CalculatorFactory`
- Removed `MultiplicationStrategy` deprecated type alias from `strategy.go`
- Removed server, REPL, and observability layers to simplify the codebase
- Documentation restructure: badge coverage updated to 87.5%; README condensed (~38 KB → ≤ 22 KB) with deep-links to `docs/PERFORMANCE.md`, `docs/BUILD.md`, `docs/TUI_GUIDE.md`
- Benchmark reporting unified on AMD Ryzen 9 5900X reference; Intel Core Ultra 9 numbers retained as comparison annex (audit P2-19)
- Dependency graph (`docs/architecture/dependency-graph.mermaid`) now includes `progress`, `memory`, `threshold` nodes (audit P2-22)
- Makefile hygiene: POSIX-only targets, `go mod tidy`, removed dead cross-compile targets (audit P2-23 to P2-26)
- **CI hardening (audit A-12/A-13/A-14)**: pinned `golangci-lint` to `v1.64.8` (was `latest`, non-reproducible); dropped `check-latest: true`; `-race` now runs on Windows too; `coverage.yml` runs on PRs with a minimum-coverage floor; added an informational benchmark job
- **test/e2e (A-15)**: heavy build+subprocess e2e tests are skipped under `-short`; timeout exit-code assertions are now discriminant (`ExitErrorTimeout=2`) instead of accepting any non-zero; `TestCLI_E2E` reuses the shared one-shot build
- **docs (A-16/A-18/A-19)**: clarified `MaxPooledBitLen` (bits) vs `maxArenaPoolWords` (words); documented the `threshold.ShouldAdjust` single-writer invariant; documented that `progress` production path is `Freeze` and annotated the TUI `ProgressDoneMsg` drain sink

### Fixed

- **orchestration**: propagate `errgroup` error instead of silently ignoring (audit P0-08)
- **fibonacci**: check `ctx.Err()` after semaphore acquisition (audit P1-19)
- **bigfft / io**: handle `Flush()` and `fourier()` errors explicitly (audit P2-11, P2-12)
- Documentation links: fixed 20+ broken cross-references across `README.md`, `docs/ARCH.md`, `docs/TESTING.md`, architecture/ hub (audit P0-04 through P0-07, P1-16)
- `docs/TESTING.md` Test Organization table: paths updated for extracted sub-packages (audit P1-15)
- `docs/PERFORMANCE.md`: removed phantom `FIBCALC_GC_CONTROL` reference; GC control is automatic (audit P1-12)
- `CONTRIBUTING.md` vs `docs/TESTING.md` mockgen divergence: `TESTING.md` is now the single source of truth (audit P2-17)
- Formatting: applied `gofmt -s` and `goimports` across the tree (audit P1-09)
- **Documentation realignment (audit A-04/A-21/A-23)**: `Claude.md` rewritten to reflect post-refactoring reality (CI exists; R1.1–R1.5 resolved; removed dead `ultrareview.md`/`ultrareviewplan.md` references; `parallel` is alive; pointers now to `audit.md`/`AuditPlanning.md`). `README.md`: linter count 22→24, `sysmon`→`metrics/system`, added `FIBCALC_PROFILE_MAX_AGE`. `docs/CALIBRATION.md`: added `Confidence` field, Strategy-pattern note, `MicroBenchTimeout` var correction, env var documented. `.env.example`: added `FIBCALC_PROFILE_MAX_AGE`
- **config (A-08)**: `Validate()` now rejects `LastDigits < 0` (was silently ignored, falling back to O(memory) full computation), negative `StrassenThreshold`, out-of-set `GCControl`/`Completion`
- **config (A-09)**: malformed explicitly-set env overrides (e.g. `FIBCALC_N=abc`) now return a structured `ConfigError` instead of being silently swallowed (phantom default)
- **calibration (A-11)**: `SaveProfile` is now atomic (temp file + `os.Rename`) — no truncated/corrupt profile under concurrent writers or crash
- **progress (A-10)**: `CalcTotalWork` no longer overflows to `+Inf` (`math.Pow(4,numBits)` past ~512 bits) which froze the progress bar on exactly the large-N calculations; reworked in log-space / closed form
- **build (A-13)**: `.gitignore` `coverage.*` glob was unanchored and also hid `.github/workflows/coverage.yml` — the coverage workflow was never tracked and never ran on GitHub; now kept under version control
- Audit reference: `audit.md` (23 findings), remediation plan `AuditPlanning.md`. Concurrency hardening of `internal/bigfft` (A-01 Critical use-after-free, A-02/A-03 data races, A-05/A-06/A-07) is implemented on the frozen branch `review/vague-A-bigfft-concurrency`, **pending human review before merge** (hot-path concurrency; `-race` validated only in CI)

### Security

- **gosec G115**: explicit whitelist with justification in `.golangci.yml` (audit P1-23, P2-09)
- **gosec G304**: documented file-path inclusion exceptions (audit P2-10)

### Performance

- **FFT pool leak**: release `PolValues` / `Poly` buffers in `internal/bigfft` (audit P0-01, P0-09)
- **Arena reuse**: `CalculationArena` is now retained across calls inside the pooled `CalculationState` (sized once, `Reset()` between uses; rebuilt only when `n` outgrows the previous tenancy). `B/op` improves ~7 % on `BenchmarkFibonacci/FastDoubling/1M`; `ns/op` is unchanged within run-to-run variance. The "steal `s.FK`" zero-copy trick was removed in `ExecuteDoublingLoop` (incompatible with arena reuse) and replaced by a single deep-copy in `ReleaseStateWithResult` (~850 KB memcpy for F(10M), <0.01 % of runtime). Audit P1-04 — previously skipped because `s.FK` aliasing pooled memory raced with the next tenant's `Reset()`; now resolved by detaching the result before pool return.

- **TUI (A-22)**: lazy log-pane content flush — `updateContent()` no longer rebuilds the whole snapshot on every push (O(N²) over a long session). `BenchmarkLogsUpdateContent` ~8× faster, −76 % allocations; observable TUI output unchanged
- Added regression guards: `TestCalculateSmall_OracleEquivalence` (A-20, cross-checks `calculateSmall` vs an independent oracle, no golden tautology) and parallel-FFT arena-aliasing race tests (A-17)

### Removed

- **Audit scaffolding documents**: deleted `audit.md`, `AuditPlanning.md`, and `audit-prompt.md` from the repository. These tracked the now-closed post-audit remediation effort and are intentionally not restored; outstanding follow-ups are carried as normal issues, not audit findings.
- Phantom `FIBCALC_GC_CONTROL` environment variable reference from `docs/PERFORMANCE.md` (never implemented)
- Tracked development log artifacts and stale files (audit P1-20)

---

## [1.0.0] - 2025-12-22

### Added

#### Core Features

- **Fast Doubling Algorithm**: O(log n) Fibonacci calculation with parallel multiplication
- **Matrix Exponentiation**: O(log n) with Strassen's algorithm for large matrices
- **FFT-Based Calculator**: Optimized for extremely large numbers using FFT multiplication
- **GMP Support**: Optional GNU Multiple Precision library integration via build tag

#### Performance Optimizations

- Zero-allocation strategy using `sync.Pool` for 95%+ reduction in GC pressure
- Adaptive parallelism based on input size and hardware capabilities
- Smart multiplication switching (Karatsuba vs FFT) based on operand size
- Symmetric matrix squaring optimization (50% reduction in multiplications)
- Auto-calibration system for hardware-specific threshold optimization

#### User Interface

- Modern CLI with progress spinners, ETA calculation, and colour themes
- Shell autocompletion generation (bash, zsh, fish, PowerShell)
- JSON output format support
- Hexadecimal result display option

#### Documentation

- Comprehensive README with production deployment guide
- Architecture documentation with ADRs
- Performance tuning guide
- Security policy with vulnerability disclosure process
- Algorithm-specific documentation (Fast Doubling, Matrix, FFT, GMP)

#### Development

- Comprehensive test suite with 80%+ coverage
- Benchmark suite for performance validation
- Mock generation with mockgen
- golangci-lint configuration

### Security

- Input validation for all parameters
- Maximum N value limit (1 billion) to prevent resource exhaustion
- Configurable request timeouts
- Rate limiting protection against DoS

---

## [0.1.0] - 2025-11-01

### Added

- Initial project structure
- Basic Fast Doubling implementation
- Command-line interface
- Unit tests for core algorithms

---

<!-- v4.0.0 and v3.0.0 are the release tags; no v1.0.0/v0.1.0 tags exist,
so the 1.0.0 and 0.1.0 sections above are intentionally unlinked. -->

[Unreleased]: https://github.com/agbruneau/FibGo/compare/v4.0.0...HEAD
[4.0.0]: https://github.com/agbruneau/FibGo/compare/v3.0.0...v4.0.0
