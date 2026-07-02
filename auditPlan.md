# auditPlan.md — Plan d'exécution des correctifs et bonifications de l'audit

> **Source** : [`audit.md`](audit.md) (2026-07-02). Ce plan couvre la **totalité** des findings (majeurs, mineurs, informatifs, code mort).
> **Mode d'exécution** : **ultracode** — orchestration multi-agents via le tool Workflow, modèle exécuteur **Claude Sonnet 5** (`model: 'sonnet'` sur chaque `agent()`).
> **Stratégie** : 7 phases séquentielles ; à l'intérieur d'une phase, les lots sont parallélisables quand ils touchent des packages disjoints (isolation worktree), sinon séquentiels. Chaque lot = TDD (test rouge → correctif minimal → vert) + gate de vérification, puis agents réfutateurs avant clôture de phase.

---

## 0. Règles d'exécution (s'appliquent à tous les lots)

1. **TDD strict pour tout bug** : écrire d'abord le test qui reproduit le défaut (il doit échouer), puis le correctif minimal, puis vert. Pour les suppressions de code mort : grep de non-usage re-vérifié avant suppression, suite verte après.
2. **Gates durs** (échec = blocage du lot) :
   - `go build ./... && go vet ./... && go test -count=1 ./...` (Windows) — ou `wsl go test -race ./...` quand disponible ;
   - couverture ≥ 80 % (`make coverage-check` ou équivalent) ;
   - golden test `fibonacci_golden.json` intact — fichier **immuable**, aucun `-update` ;
   - **benchstat ≤ 5 %** vs `docs/audits/bench-baseline.txt` pour tout lot touchant `internal/fibonacci/` ou `internal/bigfft/` (Directive #1 — d'où la Phase 0 en premier).
3. **Commits conventionnels** par lot (`fix(scope):`, `refactor(scope):`, `docs(scope):`, `test(scope):`), diff minimal, trunk-based : commit + push `main` en fin de phase vérifiée (pas de branche/PR — décision utilisateur).
4. **Invariants CLAUDE.md** : avant toute modification dans `fibonacci/` ou `bigfft/`, l'agent lit la ligne correspondante de la table « Invariants à préserver ». Interdits absolus : publier un état hors `finalizeStateReleaseTo`, réintroduire des globaux mutables non synchronisés dans `bigfft/`, masquer les sentinels fermat en `error`, toucher au golden.
5. **Orchestration Workflow (ultracode)** :
   - un appel `Workflow` par phase ; `meta.phases` = lots de la phase ;
   - `agent(prompt, {model: 'sonnet', schema: RESULT_SCHEMA, isolation: 'worktree'})` pour les lots **mutants parallèles** (packages disjoints uniquement) ; fusion en ordre de dépendance, gate re-exécuté après fusion ;
   - lots dépendants → `pipeline()` ; vérifications indépendantes → `parallel()` ;
   - **réfutateurs** : après chaque phase de code, 2-3 agents adverses par correctif majeur (`Essaie de réfuter que ce correctif est correct et complet…`), verdict majoritaire requis ;
   - schéma de sortie par lot : `{lot, findings: [{id, status: fixed|skipped|blocked, commit, evidence}], gates: {build, tests, coverage, benchstat}}`.
6. **Escalade** : si un correctif révèle un problème plus profond que décrit dans audit.md, l'agent s'arrête, documente, et le lot est marqué `blocked` — pas d'improvisation hors périmètre.

---

## Phase 0 — Débloquer le gate perf (séquentiel, bloquant pour tout le reste)

| Étape | Findings | Action | Vérification |
|---|---|---|---|
| P0.1 | **M9 / DOC-01 / TOOL-01** | **Option (a) retenue** : régénérer la baseline sur machine quiescente — `make bench-baseline` — et committer `docs/audits/bench-baseline.txt`. | Fichier présent et tracké ; `benchstat docs/audits/bench-baseline.txt docs/audits/bench-baseline.txt` s'exécute sans erreur. |
| P0.2 | (préparation) | Passe de référence : `go build ./... && go vet ./... && go test -count=1 ./...` + si WSL disponible `wsl go test -race ./...`. | Tout vert — c'est l'état « avant » opposable aux réfutateurs. |

> Nota : la mise en cohérence **documentaire** de la baseline (références PERFORMANCE/PORTABILITY/TESTING/BUILD/CLAUDE.md) est en Phase 5 (5B) — après que la politique (a) est actée par ce commit.

---

## Phase 1 — Correctifs majeurs de code (6 lots, TDD strict)

Lots parallélisables par paires de packages disjoints : {1A, 1D}, {1B}, {1C}, {1E}, {1F}. En orchestration : `pipeline()` sur les lots avec `isolation: 'worktree'` pour 1A/1C/1D/1E (packages disjoints), 1B après 1A (même fichier `app/calculate.go`), 1F seul (bigfft, gate benchstat).

### Lot 1A — APP-01 (panic pointeur/tri) — `app/` + `orchestration/`
1. Test rouge : mode comparaison + `-o` + résultats `{échec, lent, rapide}` → doit reproduire le panic nil-deref (patron de la sonde décrite dans audit.md §3/M1).
2. Correctif : copier la valeur (`best := *findBestResult(results)`) **avant** `present` ; en complément, documenter dans la godoc d'`AnalyzeComparisonResults` qu'il mute (trie) son paramètre.
3. Test complémentaire : cas « tous succès » → le fichier de sortie porte le Name/Duration du **bon** algorithme.
- **Vérif** : nouveaux tests verts, package `app` + `orchestration` verts. Commit `fix(app): copy best result before comparison sort invalidates pointer`.

### Lot 1B — FIB-01/APP-02 (--gc-control inerte) — `app/` + `tui/`
1. Test rouge (spy, patron `orchestration_spy_test.go`) : `--gc-control disabled` → l'`Options.GCMode` reçu par le calculateur vaut `"disabled"` (échoue aujourd'hui : `""`).
2. Correctif : `GCMode: a.Config.GCControl` dans `app/calculate.go` ; `GCMode: cfg.GCControl` dans `tui/commands.go`.
3. Bonus (Info, même commit ou séparé) : câbler `Options.MemoryLimitBytes` depuis le CLI pour réactiver le garde-fou du calculateur (`calculator.go:203-211`) — ou documenter explicitement qu'il est couvert au niveau app.
- **Vérif** : spy vert dans les deux modes (CLI + TUI). Commit `fix(app,tui): wire --gc-control into fibonacci Options`.

### Lot 1C — APP-03 + APP-11 (complétions fish/bash) — `cli/completion/`
1. Tests rouges : (a) assertion **registry ⊆ fish** (chaque flag du registre a sa ligne `-l`/`-s` dans le script généré) — échoue sur les 6 flags manquants ; (b) assertion valeurs bash par flag : `--fft-threshold` propose les valeurs registre (100000/500000/1000000).
2. Correctifs : fish — itérer tout `flagRegistry` (section « Autres ») ou compléter `sections` + `"v_short"` → `"verbose"` ; bash — supprimer `BashGroup`/`bashGroupValues`, émettre un case par flag depuis `f.Values`.
3. Étendre l'assertion registry ⊆ script aux 4 shells (généralisation du garde-fou, ferme la classe de bug).
- **Vérif** : tests de sync verts pour les 4 shells. Attention invariant : l'échappement par dialecte (`escape.go`) ne doit pas être contourné par le nouveau code. Commit `fix(completion): fish misses six shipped flags; bash group values wrong for fft/strassen`.

### Lot 1D — CONC-01 (data race spinner) — `cli/`
1. Test : la race n'étant visible qu'en TTY réel, le test unitaire vérifie le **contrat** du correctif (Stop→write→Start appelés dans l'ordre via un spy/fake, ou suffixe figé selon l'option retenue). Documenter dans le commit pourquoi `-race` ne la voit pas.
2. Correctif (option recommandée par audit.md) : figer le suffixe à la création et rendre la ligne de progression manuellement dans la boucle ticker existante (200 ms) — le spinner ne porte plus de texte variable. Repli : encadrer l'écriture par `rs.s.Stop()` / write / `rs.s.Start()`.
- **Vérif** : suite `cli` verte ; inspection manuelle du rendu TUI/CLI si terminal disponible. Commit `fix(cli): eliminate data race on spinner.Suffix`.

### Lot 1E — FIB-02 + FIB-03 + FIB-09 (justesse calibration) — `calibration/`
1. Tests rouges : (a) le candidat baseline produit une exécution **réellement** séquentielle/sans FFT (observable via spy sur les options effectives après `normalizeOptions` — seuils négatifs) ; (b) `analyzeResults` sans aucune mesure valide (`bySize` vide) → confiance **< 0.5** (échoue aujourd'hui : 0.9) ; (c) chemin d'affichage : profil custom → le chemin affiché est le chemin effectif.
2. Correctifs : candidat `-1` au lieu de `0` (générateurs + étiquette `printCalibrationResults` + recommandation) ; `find*Crossover` retournent `0` sans données mesurées, défauts déplacés dans `analyzeResults` **sans** bonus de confiance ; suppression du bonus « parallèle » tant que `runSingleTest` ne branche pas sur `parallel` ; propagation d'erreur dans `RunQuick` ; chemin de profil effectif affiché (FIB-09).
3. Garde-fou : vérifier que l'escalade vers `CompleteStrategy` redevient atteignable (test sur `tryFastThenEscalate` avec mesures invalides → escalade déclenchée).
- **Vérif** : suite `calibration` verte ; `IsStale`/branche stale intacte (invariant CLAUDE.md). Commit `fix(calibration): measure a real sequential baseline and stop inflating confidence`.

### Lot 1F — FFT-01 + FFT-02 (bigfft) — `bigfft/` — gate benchstat obligatoire
1. Tests rouges : (a) FFT-01 — salir le bucket de pool concerné puis vérifier l'aller-retour `NTransform`/`InvNTransform` contre une référence construite par `make()` (échoue aujourd'hui si le pool est sale) ; (b) FFT-02 — « panic de la moitié synchrone pendant qu'un worker tourne » sur les 4 sites : le process doit récupérer proprement (pas de race détectée sous `-race` WSL, pas de buffer recyclé sous les workers).
2. Correctifs : FFT-01 — `acquireWordSlice` (zéroïsé) à la place d'`unsafe` (chemin non-hot) ; FFT-02 — aux 4 sites (`fourierRecursiveUnified`, `fourierRecursiveCtx`, `executeReconstruction`, `runPointwise`), closure + `recover()` sur la part synchrone, `wg.Wait()` inconditionnel, re-`panic` ensuite.
3. **Benchstat** : `make benchmark` avant/après vs baseline P0 ; régression > 5 % = blocage et retour à l'atelier.
- **Vérif** : gardiens existants verts (`TestFourierRecursive*PanicPropagates`, `TestExecuteReconstructionPanicPropagates`, `TestPointwiseWorkerPanicPropagates`) + nouveaux tests + benchstat. Passe `-race` WSL fortement recommandée sur ce lot. Commit `fix(bigfft): zeroed pool memory in NTransform; wait for workers when the sync half panics`.

### Clôture de phase 1
- Fusion des worktrees en ordre {1A, 1C, 1D, 1E} → 1B → 1F ; re-exécution du gate complet après fusion.
- **Réfutateurs (ultracode)** : 3 agents par lot majeur, prompt type : « Voici le diff du correctif X et le finding d'audit. Essaie de le réfuter : le bug est-il réellement corrigé ? Le correctif introduit-il une régression (perf, invariant CLAUDE.md, autre appelant) ? Réponds refuted/confirmed avec preuve. » Verdict majoritaire `confirmed` requis pour committer/pusher.

---

## Phase 2 — Correctifs mineurs de code (lots par package, parallélisables)

| Lot | Findings | Actions clés | Commit type |
|---|---|---|---|
| 2A `memory/` | FIB-04/SEC-02 | `satMul(val, multiplier)` dans `ParseMemoryLimit` + test wrap-to-zero | `fix(memory)` |
| 2B `calibration/` + `app/` | SEC-01 | Re-valider `cfg` après application d'un profil chargé ; profil invalide → ignoré, fallback `ApplyAdaptiveThresholds` + test profil forgé (seuil négatif) | `fix(calibration)` |
| 2C `fibonacci/` | FIB-07, FIB-08, FIB-06(doc) | Clamp post-×1.2 ; alignement `GOMAXPROCS(0)` (gate matriciel + taille sémaphore) + correction commentaire `getTaskSemaphore` ; godoc matrix 3072. **Benchstat** (package sensible) | `fix(fibonacci)` |
| 2D `config/` | APP-08, APP-09, APP-14 | Entrées env `LAST_DIGITS`/`GC_CONTROL` ; `parseBoolEnv` loud (`malformedEnvError`) ; retourner l'erreur typée de `Validate` | `fix(config)` |
| 2E `tui/` | APP-04, APP-06 | Mapper timeout/SIGINT → `exitCode` (2/130) dans `handleContextCancelled` + test ; champ `Generation` sur `IndicatorsMsg` + filtre dans `Update` + test | `fix(tui)` |
| 2F `bigfft/` | FFT-08, FFT-13 | Release des buffers sur chemins d'erreur de `transform`/`invTransform`/jumeaux ctx ; micro-nettoyages (`rp.M` redondant, message panic `<=`, `clear(src)`). **Benchstat** | `fix(bigfft)` |
| 2G `app/` | APP-16, SEC-03 | `a.ErrWriter` au lieu d'`os.Stderr` ; dédoublonner le check `OutputFile` ; budget mémoire (ou borne K) sur le chemin last-digits + test | `fix(app)` |
| 2H `metrics/` | APP-18 | Seuil de zero-padding corrigé (test sur un nombre de 20-24 chiffres, ex. 10²⁰) | `fix(metrics)` |
| 2I tests | TEST-01, TEST-02, TEST-04, TEST-05, TEST-03, TEST-06 | Supprimer `TestColors` ; retirer le sleep (ou `TestDisplayProgress` entier) ; **épingler les knobs** des 4 gardiens ADR-0002 (`SetFFTParallelismConfig` + `t.Cleanup`) pour rendre le skip impossible ; check `ok` de `SetString` golden ; synchronisation mock calibration ; `t.Parallel()` ou commentaire (e2e, system) | `test(...)` |

- Parallélisation : 2A-2H touchent des packages disjoints → worktrees ; 2I après fusion (touche plusieurs packages de test).
- **Vérif de phase** : gate complet + couverture ≥ 80 % + benchstat (2C, 2F) + réfutateur unique par lot (léger — findings mineurs).

---

## Phase 3 — Décisions structurantes (M) — une mini-analyse avant chaque implémentation

Chaque lot commence par un agent d'analyse qui confirme l'approche (ou documente le rejet), puis implémente.

| Lot | Findings | Décision recommandée | Vérification |
|---|---|---|---|
| 3A | APP-05 | `WithTimeout` **par génération** dans la TUI (posé dans `handleReset`/`Init`), `signal.NotifyContext` reste sur le parent | Test : restart après expiration → nouveau budget complet ; codes de sortie cohérents avec 2E |
| 3B | APP-07 | **Rejeter en `Validate()`** les combinaisons `--tui` + (`--last-digits` \| `--output`) (`ConfigError`, exit 4) ; câbler `validateMemoryBudget` dans `runTUI` pour `--memory-limit` | Tests des 3 combinaisons ; message d'erreur actionnable |
| 3C | APP-10 | Déplacer `ProgressState` de `format` vers `orchestration` (ou `progress`) ; envisager d'ajouter l'arrow `orchestration → format` à la liste interdite d'`arch_test.go` une fois le déplacement fait | `TestArchitectureLayering` étendu et vert ; `format` redevient string-in/string-out |
| 3D | FIB-05 | Réduire le multiplicateur d'arène 15 → 5-6 dans `arenaTotalWords` **et** `acquireSizingForN` ; revalider les caps anti-bloat (`maxArenaPoolWords`, `maxCachedArenaWords`) ; mettre à jour le commentaire d'arena.go et l'estimateur | **Benchstat impératif** (hot path) + gardiens `TestCalculatorStateCache_*`, `TestStateBump_*`, `TestReleaseState_OverLimit_AliasesCleared` verts ; mesure mémoire avant/après sur F(10M) documentée dans le commit |
| 3E | TOOL-03 | Dockerfile : retenir **`CGO_ENABLED=0` sans apt** (le tag gmp n'est pas l'intention du binaire par défaut ; plus simple, image plus légère, binaire statique) — sinon documenter le choix `-tags gmp` | `docker build` passe ; smoke test `--version` OK |
| 3F | TOOL-04, TOOL-05 | `make check` → délègue à `scripts/check.sh` ; fusionner test+coverage en une étape `go test -race -coverprofile=coverage.out ./...` dans les **deux** gates (sh/ps1, sans `-race` côté ps1) | Les deux gates passent en une seule exécution de la suite ; sémantique advisory du lint préservée |

---

## Phase 4 — Nettoyage code mort & sur-ingénierie

Ordre : décisions d'abord (4E), mécanique ensuite. Chaque suppression : re-grep de non-usage, adaptation des tests orphelins, suite verte.

| Lot | Findings | Contenu |
|---|---|---|
| 4A `ui/` | OVR-02 | Supprimer `LightTheme`, `OrangeTheme`, `SetTheme` ; documenter `SetCurrentTheme` test-only (les tests de restauration en dépendent) |
| 4B `tui/` | OVR-09, APP-15(chart) | Supprimer `RenderBrailleChart`/`brailleDots`/`plotBrailleValue` ; renommer `renderBrailleSection` (nom menteur) ; retirer le paramètre `progress` ignoré d'`AddDataPoint` |
| 4C `errors/` | OVR-07 | Supprimer `TimeoutError`, `ValidationError`, `WrapError`, `IsContextError` + tests. Invariant : ne pas réintroduire d'import `format` |
| 4D `fibonacci/` (+`threshold/`, `memory/`, `calibration/`) | OVR-03, OVR-04, OVR-05, OVR-06, FIB-10 | `preSizeBigInt` (copie common.go) ; 4 méthodes legacy + constructeur jumeau DTM (adapter les tests vers `FromConfig`) ; `AcquireState`, `ShouldParallelizeMultiplication`, `SetDefaultStrassenThreshold` (cohérent avec la décision FIB-06), `setOrReturn` ; `MustNewCalculator` : **décision** — garder + corriger doc.go, ou retirer partout (12 fichiers tests) ; `GlobalFactory`/`RegisterCalculator` ; `MicroBenchPerTestTimeout` (mort strict) ; `AllocBigInt`/`UsedWords` (cohérent avec 3D). **Benchstat** après coup (package sensible) |
| 4E `bigfft/` | OVR-01, FFT-05, FFT-06, OVR-10, FFT-12 | **Décisions à acter (ADR court, patron ADR-0008)** : (a) OVR-01 — supprimer `scan.go`+`scan_test.go` (rend FFT-07 sans objet) — recommandé ; (b) FFT-05 — supprimer la machinerie `fftState` + **synchroniser la ligne CLAUDE.md** (`fftStatePool`, A2-05) — recommandé ; (c) FFT-06 — fusionner `arith_amd64.go`/`arith_generic.go`, corriger doc, dé-gater le test ; (d) OVR-10 — **conserver** le cluster oracle (Poly.*, TransformCache.Get/Put, *Cached non-bump) mais le **documenter « oracle de test »** en tête de chaque fonction ; remplacer les 5 usages test de `computeKey` par `computeCacheKey` et supprimer l'alias. **Benchstat** après coup |
| 4F divers | OVR-11, OVR-12, APP-12 reliquats, APP-15, APP-17 | `RecoveredObserverCount` : ajouter le test qui l'exerce (option la plus utile) ; lot d'exports test-only (dé-export au fil de l'eau — `GetVersionInfo`, `DisplayResultWithConfig`, `DisplayMemoryStats`, `FormatProgressBarWithETA`, `NumCalculators`, `ErrorCollector.Reset`, etc. — en **conservant** `app.WithFactory` et `config.validateEnvOverrides`) ; `HexDisplayEdges` ; champ `Indicators.Live` ; params morts `FormatQuietResult` ; duplications APP-17 : **won't-fix documenté** (à ne toucher que si l'un des côtés évolue) |

- **Vérif de phase** : gate complet + couverture (les suppressions retirent aussi des tests — vérifier que le plancher tient) + benchstat (4D, 4E) + `TestArchitectureLayering`.

---

## Phase 5 — Documentation & outillage documentaire

| Lot | Findings | Actions |
|---|---|---|
| 5A ponctuels | M10/DOC-02, M11/DOC-03, DOC-04, DOC-05, DOC-06, DOC-07, DOC-08 | CHANGELOG : remplacer le lien `AuditPlan.md` + puce `tui/component` ; purger `cpu_amd64.go` de ARCH/BUILD/PORTABILITY (§2.2 → `internal/config/hardware.go`) ; retirer `tui/component` d'ARCH.md + supprimer le répertoire vide ; CLAUDE.md → renvoi BUILD.md pour le dashboard ; date `-race` 2026-06-21 ; CONTRIBUTING `FibGo` |
| 5B baseline | M9 (volet doc) | Mettre PERFORMANCE/PORTABILITY/TESTING/BUILD/CLAUDE.md en cohérence avec la politique (a) actée en P0 (baseline committée, procédure de refresh) ; remplacer la réf `bench-dtm-{on,off}.txt` par un renvoi CHANGELOG/ADR-0001 |
| 5C doc-comments code | FFT-10, FFT-11, APP-13, CONC-03, OVR-08, FIB-06(doc si pas fait en 2C) | Réécrire les 4 puces de `bigfft/doc.go` ; `SetCacheLogger` (décision : câbler le logger applicatif **ou** ajuster le commentaire — recommandé : ajuster, l'observabilité cache n'est pas demandée) ; 4 doc-comments APP-13 ; `ErrorCollector.Err()` ; `SetFFTThreshold` + annotation ADR-0004 B1 (justification calibration caduque) |
| 5D dashboard | DOC-09 | Régénérer `knowledge-graph.json` + bundle selon BUILD.md (artefact généré — ne pas éditer à la main) ; sinon documenter le décalage comme connu |
| 5E outillage | TOOL-02, TOOL-06, SEC-04, SEC-05 | `.golangci.yml` : supprimer les 2 blocs G304 morts + `dupl` ; marqueurs POSIX manquants (`stats`, `coverage-check`) ; Dockerfile : épingler les images par digest ; exécuter `govulncheck ./...` (recompilé sous go1.26) + `go list -m -u all`, intégrer au gate `scripts/check.*` si concluant |
| 5F CLAUDE.md final | (synthèse) | Répercuter dans la table des invariants toutes les décisions de Phase 4 (fftState supprimé → retirer la mention A2-05/fftStatePool ; oracle bigfft documenté ; etc.) et les nouveaux tests gardiens créés en Phases 1-3 |

- **Vérif** : zéro lien markdown cassé (re-scan), chaque claim documentaire modifié confronté au code (patron de l'agent DOC), gate complet.

---

## Phase 6 — Vérification finale, ADR et clôture

1. **Gate complet final** : `scripts/check.ps1` (Windows) + `wsl go test -race ./...` (si WSL disponible) + `make coverage-check` + golden.
2. **Benchstat final** : `make benchmark` vs baseline P0 — bilan global ≤ 5 % sur tous les benchmarks (les phases 1F/2C/2F/3D/4D/4E ont déjà leurs gates locaux ; ceci est la confirmation d'ensemble).
3. **ADR** : rédiger l'ADR court actant les décisions de Phase 4E (purge scan.go/fftState, rétention oracle documentée) + l'annotation ADR-0004 B1 — patron ADR-0008.
4. **CHANGELOG** : entrée « Audit 2026-07 » listant les correctifs par catégorie, avec renvoi à `audit.md`.
5. **Réfutation finale (ultracode)** : panel de 3 agents adverses sur l'ensemble du diff cumulé — « un invariant CLAUDE.md a-t-il été cassé ? un finding marqué fixed ne l'est-il pas ? une régression benchstat est-elle masquée ? » — verdict majoritaire requis.
6. **Livraison** : commit(s) finaux + push `main`.

---

## Matrice de traçabilité findings → plan

| Finding | Phase.Lot | | Finding | Phase.Lot | | Finding | Phase.Lot |
|---|---|---|---|---|---|---|---|
| M1/APP-01 | 1A | | FIB-07 | 2C | | OVR-01 | 4E |
| M2/FIB-01/APP-02 | 1B | | FIB-08 | 2C | | OVR-02 | 4A |
| M3/APP-03 | 1C | | FIB-09 | 1E | | OVR-03 | 4D |
| M4/CONC-01 | 1D | | FIB-10 | 4D | | OVR-04 | 4D |
| M5/FIB-02 | 1E | | FFT-03 | 5C (doc) | | OVR-05 | 4D |
| M6/FIB-03 | 1E | | FFT-04/CONC-02 | 5C (doc) | | OVR-06 | 4D |
| M7/FFT-01 | 1F | | FFT-05 | 4E | | OVR-07 | 4C |
| M8/FFT-02 | 1F | | FFT-06 | 4E | | OVR-08 | 5C |
| M9/DOC-01/TOOL-01 | 0 + 5B | | FFT-07 | 4E (via OVR-01) | | OVR-09 | 4B |
| M10/DOC-02 | 5A | | FFT-08 | 2F | | OVR-10 | 4E |
| M11/DOC-03 | 5A | | FFT-09 | 2F ou 4E | | OVR-11 | 4F |
| SEC-01 | 2B | | FFT-10 | 5C | | OVR-12 | 4F |
| FIB-04/SEC-02 | 2A | | FFT-11 | 5C | | TOOL-02 | 5E |
| SEC-03 | 2G | | FFT-12 | 4E | | TOOL-03 | 3E |
| SEC-04 | 5E | | FFT-13 | 2F | | TOOL-04 | 3F |
| SEC-05 | 5E | | APP-04 | 2E | | TOOL-05 | 3F |
| CONC-03 | 5C | | APP-05 | 3A | | TOOL-06 | 5E |
| TEST-01 | 2I | | APP-06 | 2E | | DOC-04 | 5A |
| TEST-02 | 2I | | APP-07 | 3B | | DOC-05 | 5A |
| TEST-03 | 2I | | APP-08 | 2D | | DOC-06 | 5A |
| TEST-04 | 2I | | APP-09 | 2D | | DOC-07 | 5A |
| TEST-05 | 2I | | APP-10 | 3C | | DOC-08 | 5A |
| TEST-06 | 2I | | APP-11 | 1C | | DOC-09 | 5D |
| FIB-05 | 3D | | APP-12 | 4A/4B/4C/4F | | APP-13 | 5C |
| FIB-06 | 2C + 4D | | APP-14 | 2D | | APP-16 | 2G |
| APP-15 | 4B/4F | | APP-17 | 4F (won't-fix doc) | | APP-18 | 2H |

**Décisions explicites intégrées au plan** (à confirmer ou amender à l'exécution — chacune a une recommandation) : politique baseline = committée (P0) ; candidat calibration = `-1` (1E) ; spinner = suffixe figé + rendu manuel (1D) ; `--tui` + flags incompatibles = rejet en `Validate` (3B) ; Dockerfile = `CGO_ENABLED=0` (3E) ; `scan.go` et `fftState` = suppression avec ADR (4E) ; cluster oracle bigfft = rétention documentée (4E) ; duplications APP-17 = won't-fix documenté (4F).

**Fin du plan** — le travail est terminé quand : toutes les cases de la matrice sont `fixed` (ou `won't-fix` documenté par ADR/commentaire), le gate final Phase 6 est vert, benchstat global ≤ 5 %, et le panel de réfutation finale a rendu un verdict majoritaire `confirmed`.
