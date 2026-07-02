# Audit de code exhaustif — FibGo

> **Date** : 2026-07-02
> **Périmètre** : totalité du dépôt (`cmd/`, `internal/` — 15 packages, `test/`, `scripts/`, `Makefile`, `Dockerfile`, `.golangci.yml`, `go.mod`, `docs/`)
> **Base auditée** : branche `main`, commit `b87a342` (arbre propre)
> **Verdict global** : base **saine et durcie** par les audits précédents (2026-06). Aucun finding critique. **11 findings majeurs** (dont 3 prouvés par exécution), ~30 mineurs, ~20 informatifs. Les invariants documentés dans CLAUDE.md sont **tous vérifiés en place**, les 12 tests gardiens existent, la suite complète est verte.

---

## 1. Méthodologie

L'audit a été mené par 8 analyses parallèles indépendantes, chacune couvrant une dimension :

| Dimension | Préfixe IDs | Périmètre |
|---|---|---|
| Cœur algorithmique | FIB | `internal/fibonacci/` (+`threshold/`, `memory/`), `internal/calibration/` |
| FFT multi-précision | FFT | `internal/bigfft/` |
| Sûreté concurrente | CONC | tout le code de production |
| Sécurité & robustesse | SEC | frontières d'entrée, completion shell, Dockerfile, scripts, dépendances |
| Couches supérieures | APP | `cmd/`, `app/`, `orchestration/`, `tui/`, `ui/`, `cli/`, `config/`, `errors/`, `format/`, `metrics/`, `parallel/`, `progress/` |
| Qualité des tests | TEST | ~29,6 k LOC de tests, gardiens, couverture |
| Dérive documentaire | DOC | tous les `.md`, `doc.go`, diagrammes vs code réel |
| Sur-ingénierie & outillage | OVR / TOOL | code mort, abstractions inutiles, Makefile, golangci, Dockerfile, go.mod |

Règles appliquées : chaque soupçon vérifié en lisant le code appelant/appelé avant d'être rapporté ; les décisions assumées (ADR-0001 à 0008, table des invariants CLAUDE.md) ne sont **pas** re-signalées ; trois findings ont été **prouvés par exécution** (sondes temporaires, supprimées ensuite) ; les outils `deadcode` (x/tools) et `golangci-lint` v1.64.8 ont été exécutés réellement.

### État de santé constaté avant audit

- `go build ./... && go vet ./... && go test -count=1 ./...` : **vert** (24 packages, exit 0).
- Couverture réelle : tous les packages entre 82,5 % (`cmd/generate-golden`, dérogation documentée A5-09) et 100 % — plancher de 80 % respecté partout.
- Les **12 tests gardiens** listés dans CLAUDE.md existent tous (vérifié par grep, emplacements en §7.1).
- Les 3 arrows interdits d'`arch_test.go` tiennent. `go.mod` est propre (10 requires directs, tous importés).

---

## 2. Synthèse des findings

### 2.1 Findings majeurs (11, après déduplication)

| # | ID(s) | Localisation | Résumé | Effort |
|---|---|---|---|---|
| M1 | APP-01 | `app/calculate.go:172-181` | Pointeur `best` invalidé par un tri en place → **panic nil-deref** possible en `-algo all -o` (prouvé) | S |
| M2 | FIB-01 = APP-02 | `app/calculate.go:79-83`, `tui/commands.go:49-53` | Flag `--gc-control` parsé, validé, documenté… **jamais câblé** : `aggressive`/`disabled` inertes (prouvé) | S |
| M3 | APP-03 | `cli/completion/fish.go:27-33` | Complétions fish : **6 flags livrés absents** — exactement ceux que le test de sync prétend couvrir (prouvé) | S |
| M4 | CONC-01 | `cli/ui.go:64-66` | **Data race réelle** sur `spinner.Suffix` (lib tierce) — invisible sous `-race` car uniquement en terminal réel | S |
| M5 | FIB-02 | `calibration/adaptive.go:25` | Candidat « Sequential » de la calibration = **doublon du défaut** (normalizeOptions remplace 0 avant mesure) | M |
| M6 | FIB-03 | `calibration/microbench.go` | Crossover parallèle mesuré sur deux charges **identiques** + confiance gonflée à 0.9 sans mesure valide → escalade `CompleteStrategy` inatteignable | M |
| M7 | FFT-01 | `bigfft/fft_poly.go:358` | `NTransform` lit de la **mémoire de pool non initialisée** → transformée fausse si pool sale (API test-only, régression de la migration `unsafe`) | S |
| M8 | FFT-02 | `bigfft/fft_recursion.go`, `fft_poly.go` (4 sites) | Panic côté appelant dans les régions parallèles → `wg.Wait()` sauté, **buffers rendus aux pools pendant que des workers les lisent** | M |
| M9 | DOC-01 = TOOL-01 | `docs/audits/` (vide) | Baseline benchmark **fantôme** : le gate perf Directive #1 (« régression > 5 % = blocage ») est inapplicable — 6 documents la référencent | S/M |
| M10 | DOC-02 | `CHANGELOG.md:16` | Lien vers `AuditPlan.md` **supprimé** (commit a4bd064) — unique lien markdown cassé du dépôt | S |
| M11 | DOC-03 | `ARCH.md:138`, `BUILD.md:95`, `PORTABILITY.md:38-45` | `cpu_amd64.go` décrit dans 3 docs alors que **supprimé** (94a5cfa) — PORTABILITY documente un mécanisme SIMD inexistant | S |

### 2.2 Répartition complète

| Sévérité | Nombre (dédupliqué) | Distribution |
|---|---|---|
| Critique | 0 | — |
| Majeur | 11 | 8 code, 3 documentation/processus |
| Mineur | ~30 | correctifs S pour l'essentiel |
| Info | ~20 | hygiène, doc-comments, code mort à décision |

### 2.3 Recoupements entre dimensions (déduplication)

| Doublon | Conservé | Note |
|---|---|---|
| APP-02 ≡ FIB-01 | M2 | même défaut vu des deux couches |
| SEC-02 ≡ FIB-04 | FIB-04 | preuve FIB plus forte : wrap-to-zero **désactive** la garde mémoire |
| CONC-02 ≡ FFT-04 | FFT-04 | race latente `FFTContext.resolved()` (chemin inutilisé en prod) |
| TOOL-01 ≡ DOC-01 | M9 | même cause : purge 7ab9098 sans suivi des références |
| FFT-05 ⊂ OVR-10 | FFT-05 | machinerie `fftState` morte ; OVR-10 élargit au cluster oracle |
| FFT-06 ⊂ OVR-10 | FFT-06 | `AddVV/SubVV/AddMulVVW` test-only |
| FFT-07 ⊂ OVR-01 | OVR-01 | si `scan.go` est supprimé (recommandé), FFT-07 devient sans objet |
| APP-12 ∩ OVR-02/03/07/09/12 | OVR-* | inventaire code mort consolidé en §6 |
| FIB-10 ∩ OVR-03/05/12 | OVR-* | idem |

---

## 3. Findings majeurs — détail

### M1 / APP-01 — Pointeur `best` invalidé par un tri en place → panic possible

- **Sévérité** : Majeur — **prouvé par exécution** (sonde : `PANIC during analyzeResultsWithOutput: runtime error: invalid memory address or nil pointer dereference`)
- **Fichiers** : `internal/app/calculate.go:172-181`, `internal/orchestration/orchestrator.go:118-123`
- **Description** : `analyzeResultsWithOutput` capture `bestResult` comme **pointeur dans le slice** (`findBestResult` retourne `&results[i]`), puis `present` → `orchestration.AnalyzeComparisonResults` **trie ce même slice en place** (`sort.Slice`). Après tri, le pointeur désigne un autre élément.
- **Conséquences** : (a) mode comparaison + `-o fichier` avec échec partiel (un algo annulé après qu'un autre a fini) → le slot pointé contient un résultat en échec avec `Result == nil` → **panic nil-deref** dans `WriteResultToFile` (`result.BitLen()`, `display.go:300`) ; (b) tous succès → fichier sauvé avec **Name/Duration du mauvais algorithme**. Le mode quiet n'est pas affecté.
- **Correctif** : copier la valeur (`best := *findBestResult(results)`) avant `present`, ou sélectionner le best **après** `AnalyzeComparisonResults` ; en profondeur : faire trier une copie dans `AnalyzeComparisonResults` (il mute son paramètre d'entrée).
- **Effort** : S

### M2 / FIB-01 = APP-02 — `--gc-control` jamais câblé : `aggressive` et `disabled` sont des no-ops

- **Sévérité** : Majeur — prouvé par grep exhaustif (`Options.GCMode` jamais assigné hors tests ; `Config.GCControl` jamais consommé hors parsing/validation)
- **Fichiers** : `internal/app/calculate.go:79-83`, `internal/tui/commands.go:49-53` (défaut consommé dans `internal/fibonacci/calculator.go:214-218`)
- **Description** : `AppConfig.GCControl` est parsé (`config.go:173`), validé (`config.go:116-121`), complété dans les 4 shells et documenté comme fonctionnel (README.md:174, docs/ARCH.md:573/765), mais aucun chemin de production ne recopie `cfg.GCControl` dans `fibonacci.Options.GCMode`. Les deux sites construisant `Options` ne renseignent que les 3 thresholds → `CalculateWithObservers` retombe toujours sur `"auto"`.
- **Note connexe (Info)** : `Options.MemoryLimitBytes` n'est pas câblé non plus depuis le CLI — sans impact utilisateur (`validateMemoryBudget` couvre au niveau app), mais le garde-fou défense-en-profondeur du calculateur (`calculator.go:203-211`) est mort sur ce chemin.
- **Correctif** : ajouter `GCMode: a.Config.GCControl` (app) et `GCMode: cfg.GCControl` (tui) + test spy vérifiant la propagation (patron `orchestration_spy_test.go`).
- **Effort** : S

### M3 / APP-03 — Complétions fish : 6 flags livrés absents

- **Sévérité** : Majeur — **prouvé par exécution** (sonde : `MISSING from fish script: "-l verbose" / "-l calculate" / "-l tui" / "-l last-digits" / "-l memory-limit" / "-l gc-control" / "-s v" / "-s c"`)
- **Fichier** : `internal/cli/completion/fish.go:27-33, 57-78`
- **Description** : le générateur fish filtre le registre via une liste de sections **maintenue à la main** ; six flags livrés en sont absents : `verbose`/`-v` (l'id `"v_short"` ne matche jamais car l'entrée registre a `Long:"verbose"` alors que le filtre exige `Long==""`), `calculate`/`-c`, `tui`, `last-digits`, `memory-limit`, `gc-control`. Ce sont exactement les six flags que `registry_sync_test.go:18-20` affirme avoir été corrigés — le garde-fou ne couvre que config↔registry, pas registry↔fish. Bash, zsh et powershell itèrent le registre complet et sont corrects.
- **Correctif** : itérer tout `flagRegistry` (avec une section « Autres »), ou compléter `sections` + remplacer `"v_short"` par `"verbose"` ; ajouter une assertion registry ⊆ fish dans le test de sync.
- **Effort** : S

### M4 / CONC-01 — Data race sur `spinner.Suffix`

- **Sévérité** : Majeur (data race réelle en production interactive)
- **Fichiers** : `internal/cli/ui.go:64-66` (`realSpinner.UpdateSuffix`), déclenchée par `internal/cli/display.go:114` (boucle ticker de `DisplayProgress`)
- **Description** : `UpdateSuffix` écrit directement le champ exporté `rs.s.Suffix` pendant que la goroutine de rendu du spinner (démarrée par `Start()`) lit ce même champ **sous le mutex interne de la bibliothèque** (spinner v1.23.2, spinner.go:340, 356-363). Un mutex pris d'un seul côté ne synchronise rien → data race au sens du memory model Go.
- **Pourquoi invisible** : `Start()` ne lance pas de goroutine si `!isRunningInTerminal(s)` — les tests `-race` (writer non-TTY ou mock) ne la voient jamais. C'est pourtant le chemin UX principal en terminal réel.
- **Correctif** : dans `realSpinner.UpdateSuffix`, encadrer l'écriture par `rs.s.Stop()` / écriture / `rs.s.Start()` (après `Stop()`, la goroutine de rendu sort sans lire `Suffix` ; l'écriture est happens-before le `go` de `Start()`). Alternative plus propre : figer le suffixe à la création et rendre la ligne de progression manuellement dans la boucle ticker existante (déjà cadencée à 200 ms).
- **Effort** : S

### M5 / FIB-02 — Calibration : le candidat « Sequential » mesure la configuration par défaut

- **Sévérité** : Majeur (les mesures ne mesurent pas ce qu'elles prétendent)
- **Fichiers** : `internal/calibration/adaptive.go:25, 54-69, 75`, `runner.go:57/84`, `calibration.go:211`, `io.go:22-24`
- **Description** : les générateurs incluent `0` comme « Sequential (no parallelism) », mais `normalizeOptions` (`options.go:71-83`) remplace `ParallelThreshold==0` par 4096 et `FFTThreshold==0` par 500 000 **avant** toute décision. Le candidat 0 est donc un doublon exact du candidat par défaut : la baseline séquentielle n'est jamais mesurée, la table de résultats l'étiquette faussement « Sequential », et la recommandation `--threshold 0` signifie en fait « auto ». Seul un seuil **négatif** désactive réellement le parallélisme (`useParallel : ParallelThreshold > 0`) ou le FFT.
- **Correctif** : utiliser `-1` comme candidat baseline (réellement séquentiel / sans FFT — compatibilité vérifiée avec `useParallel` et `smartMultiply`), adapter l'étiquette de `printCalibrationResults` et la recommandation. Nota : correctif de **justesse**, distinct de la simplification rejetée par ADR-0004 §B2.
- **Effort** : M

### M6 / FIB-03 — Microbench : crossover parallèle factice + confiance gonflée qui tue l'escalade

- **Sévérité** : Majeur
- **Fichiers** : `internal/calibration/microbench.go:332-375` (`findParallelCrossover`), `:246-288` (`analyzeResults`), `:291-329` (`findFFTCrossover`), `:101-116` (`RunQuick`)
- **Description** — trois défauts liés :
  1. `runSingleTest` ignore le flag `parallel` (documenté P1-07), mais `findParallelCrossover` compare quand même « seq vs par » — deux jeux de mesures **identiques** ; le crossover détecté est du bruit ou le défaut 4096.
  2. Les fallbacks de `findFFTCrossover` (1 000 000) et `findParallelCrossover` (4096) sont `> 0`, donc `analyzeResults` ajoute +0.2 de confiance **même sans aucune mesure valide** : avec `results` non vide mais 100 % en erreur (`bySize` vide), la confiance vaut 0.9.
  3. `RunQuick` retourne toujours `nil` en erreur.
- **Conséquence** : `conf < EscalationConfidenceThreshold (0.5)` est quasi impossible → `CompleteStrategy` est pratiquement inatteignable via l'escalade standard (seul le chemin « profil stale » l'exécute), et des profils aux valeurs par défaut sont persistés avec `Confidence=0.9`.
- **Correctif** : faire retourner `0` aux deux `find*Crossover` en absence de données mesurées (défauts déplacés dans `analyzeResults` sans bonus de confiance) ; supprimer le bonus parallèle tant que `runSingleTest` ne branche pas réellement sur `parallel`.
- **Effort** : M

### M7 / FFT-01 — `NTransform` lit de la mémoire de pool non initialisée

- **Sévérité** : Majeur (correction arithmétique — API exportée, sans appelant de production, impact borné)
- **Fichier** : `internal/bigfft/fft_poly.go:358` (et boucle 368-377)
- **Description** : `tbits := acquireWordSliceUnsafe(wordCount)` retourne un buffer de pool **non nettoyé**. La boucle n'écrit `twisted[i]` (via `Shift`, qui écrase entièrement) que pour `i < len(p.A)` ; or `len(p.A) < 1<<k` est garanti → **au moins un** `twisted[i]` contient des résidus du pool et est lu tel quel par `fourier(...)` → transformée fausse dès que le pool est sale. Le chemin jumeau `transform()` est correct (`AllocFermatSlice` zéroïsé). Régression introduite en migrant `make()` → acquisition *unsafe* ; le contrat documenté (« Use this only when the caller will immediately overwrite all elements », `pool.go:92-93`) est violé. Non détecté car `TestNTransform_InvNTransform` n'asserte pas de valeurs exactes.
- **Correctif** : remplacer par `acquireWordSlice(wordCount)` (zéroïsé — `NTransform` n'est pas hot) ou `clear(twisted[i])` dans la branche `else`. Ajouter un test qui salit d'abord le bucket concerné puis vérifie l'aller-retour NTransform/InvNTransform contre une référence `make()`.
- **Effort** : S

### M8 / FFT-02 — Panic côté appelant : `wg.Wait()` sauté, buffers recyclés sous les workers

- **Sévérité** : Majeur (concurrence, trou symétrique du contrat de propagation documenté)
- **Fichiers** : `fft_recursion.go:157-171` (`fourierRecursiveUnified`), `fft_recursion_ctx.go:80-94`, `fft_recursion.go:263-269` (`executeReconstruction`, chunk 0), `fft_poly.go:502-514` (`runPointwise`, chunk 0)
- **Description** : l'invariant documenté couvre la direction worker→appelant (`panicCh` + re-panic après `wg.Wait()`). La direction inverse ne l'est pas : si la **moitié synchrone** (ou le chunk 0) panique — p. ex. une panic d'arité `"len(z) != len(x) in Shift"`, non-sentinel donc convertie en `error` à l'entry point (le process continue) — l'unwind saute `wg.Wait()`. Les defers de l'unwind rendent alors aux pools des buffers encore lus par les workers orphelins : `ReleaseBumpAllocator(ba)` (`fft_core.go:59`) recycle le buffer bump servant de `src` aux transforms, et `rv.Release()`/`pv.Release()` (`fft_cache.go:574-586`) recyclent les `valbits`. Un acquéreur suivant écrit dans ces buffers pendant que l'orphelin les lit → **data race réelle**, résultats poubelle confinés à des buffers fuités, contrat « recover→error propre » violé.
- **Correctif** : aux 4 sites, exécuter la part synchrone dans une closure qui capture sa panic, faire `wg.Wait()` inconditionnellement, puis re-`panic` :
  `var rSync any; func(){ defer func(){ rSync = recover() }(); errSync = ... }(); wg.Wait(); if rSync != nil { panic(rSync) }`.
  Chemin froid, mais gate benchstat obligatoire (hot path, Directive #1). Ajouter un test « panic sync pendant qu'un worker tourne ».
- **Effort** : M

### M9 / DOC-01 = TOOL-01 — Baseline benchmark fantôme : le gate perf est inapplicable

- **Sévérité** : Majeur (seul finding qui casse réellement un processus documenté)
- **Fichiers** : `docs/audits/` (vide) vs `docs/PERFORMANCE.md:68,75,86-87,91`, `docs/PORTABILITY.md:136`, `docs/TESTING.md:245`, `docs/BUILD.md:247`, `CLAUDE.md:72,87,112,129`, `Makefile:217-234`
- **Description** : `docs/audits/bench-baseline.txt` n'existe pas — créé au commit 55f1d97, **supprimé au commit 7ab9098** (« purge des artefacts d'audit »), jamais régénéré. Or c'est la référence du gate perf n°1 du projet (« régression > 5 % = blocage », `benchstat` contre ce fichier versionné) ; PERFORMANCE.md documente même le `git add … && git commit` de la baseline. PERFORMANCE.md:91 renvoie aussi à `bench-dtm-{on,off}.txt`, également purgés. Le CHANGELOG (2026-06-10) affirme que les références « ont été redirigées vers ce CHANGELOG » — au moins 6 références actives le contredisent.
- **Correctif** : trancher la politique — (a) **recommandé** : recommitter une baseline via `make bench-baseline` sur machine quiescente ; ou (b) reformuler toutes les références en « baseline régénérée à la demande, non versionnée », retirer le bloc `git add … && git commit`, corriger PORTABILITY.md:136 au conditionnel, remplacer `bench-dtm-{on,off}.txt` par un renvoi CHANGELOG/ADR-0001.
- **Effort** : S (option a) / M (option b)

### M10 / DOC-02 — CHANGELOG pointe vers `AuditPlan.md`, supprimé

- **Sévérité** : Majeur (la trace d'audit promise n'existe plus ; unique lien markdown cassé du dépôt — scan exhaustif)
- **Fichier** : `CHANGELOG.md:16`
- **Description** : « Détail complet et déviations assumées : [`AuditPlan.md`](AuditPlan.md) » — fichier supprimé au commit a4bd064, postérieur à la rédaction de l'entrée.
- **Correctif** : remplacer le lien par un résumé inline ou la référence du commit d'audit.
- **Effort** : S

### M11 / DOC-03 — `cpu_amd64.go` décrit dans 3 documents alors qu'il est supprimé

- **Sévérité** : Majeur (PORTABILITY décrit un mécanisme de détection SIMD inexistant, avec accesseurs nommés)
- **Fichiers** : `docs/ARCH.md:138`, `docs/BUILD.md:95`, `docs/PORTABILITY.md:38-45`
- **Description** : ARCH.md liste `cpu_amd64.go # Runtime CPU feature detection (AVX2, etc.)` dans le tree ; BUILD.md le référence en table ; PORTABILITY.md §2.2 entière décrit `HasAVX2()` etc. Réalité : fichier supprimé au commit 94a5cfa (le CHANGELOG le liste lui-même sous « Removed (code mort) » ; `BIGFFT.md` a été nettoyé mais ces trois docs ont échappé au sweep). Aucun `HasAVX2` dans le code ; la seule détection CPU restante vit dans `internal/config/hardware.go`.
- **Correctif** : supprimer la ligne du tree ARCH.md et l'entrée BUILD.md ; réécrire PORTABILITY §2.2 pour pointer `internal/config/hardware.go`.
- **Effort** : S

---

## 4. Findings mineurs — détail par domaine

### 4.1 Sécurité & robustesse

**SEC-01 — Les seuils d'un profil de calibration chargé contournent `config.Validate()`** — Mineur, S
`internal/calibration/calibration.go:346-358, 406-417, 390-401` ; appelé depuis `app/app.go:93-94`. `ParseConfig` valide (rejette les seuils négatifs), **puis** `LoadCachedCalibration` écrase `cfg.Threshold`/`FFTThreshold`/`StrassenThreshold` avec des valeurs JSON non fiables (`--calibration-profile` ou `FIBCALC_CALIBRATION_PROFILE`). `IsValid()` (`profile.go:205-234`) ne vérifie que la compatibilité matérielle, jamais la plage des seuils. Impact contenu : ces seuils ne pilotent que des comparaisons d'algorithme, jamais des allocations. **Correctif** : rappeler `cfg.Validate(availableAlgos)` après application du profil ; en cas d'échec, ignorer le profil au profit de `config.ApplyAdaptiveThresholds(cfg)` (fallback existant).

**FIB-04 (= SEC-02) — `ParseMemoryLimit` : débordement silencieux qui peut désactiver la garde mémoire** — Mineur, S
`internal/fibonacci/memory/budget.go:71` : `return val * multiplier` wrap en uint64. Exemple : `"18014398509481984K"` (2⁵⁴ × 1024 = 2⁶⁴) → limite `0`, qui **désactive** le check (`CanCalculate` : `memLimitBytes == 0` = pas de limite). `satMul` existe déjà dans le même fichier et est utilisé partout ailleurs précisément pour éviter ce wrap. **Correctif** : `return satMul(val, multiplier), nil` (ou erreur explicite) + cas de test.

### 4.2 Cœur fibonacci / calibration

**FIB-05 — Arène dimensionnée ×15 mais 5 slots consommés ; estimateur mémoire incohérent** — Mineur (bonification), M
`memory/arena.go:53` (commentaire « 15 temporaries … up to 12 » périmé), `fastdoubling.go:326-338` (`acquireSizingForN` ×15), `budget.go:30` (`StateBytes` ×5). Seul consommateur prod : `prepareStateForN` → `PreSizeFromArena` sur 5 slots (FK, FK1, T1-T3). `AllocBigInt` : aucun appelant hors tests. Le scratch FFT vient du `BumpAllocator` depuis P1-01/F-012. L'arène alloue ~3× l'utilisé (F(10M) : ~13 Mo vs ~4,3 Mo) ; `EstimateMemoryUsage` (5×) sous-estime l'empreinte réelle (15×). **Correctif** : multiplicateur 15 → 5-6 dans `arenaTotalWords` **et** `acquireSizingForN`, revalider les caps anti-bloat (`maxArenaPoolWords`, `maxCachedArenaWords`), gate benchstat + gardiens `TestCalculatorStateCache_*`, `TestStateBump_*`.

**FIB-06 — Strassen : fallback atomic 256 mort en prod + godoc fausse** — Mineur, S
`matrix_ops.go:14-35`, `constants.go:36` (`DefaultStrassenThreshold = 3072`), `matrix.go:40-44` (doc « default 256 bits »). `ExecuteMatrixLoop` normalise toujours → `multiplyMatrices` ne reçoit jamais 0 en prod ; `Set/GetDefaultStrassenThreshold` + `init()` = tests only. **Correctif minimal** : corriger la doc de `matrix.go` (3072) + documenter le mécanisme atomic comme filet test-only.

**FIB-07 — `decideCacheTuning` peut dépasser la borne documentée de 20 %** — Mineur, S
`cache_strategy_bigfft.go:82-84` : garde `< 8192` évaluée **avant** ×1.2, sans clamp après → `MaxEntries` jusqu'à 9829 vs borne 8192. **Correctif** : clamp après multiplication.

**FIB-08 — GOMAXPROCS vs NumCPU incohérents + commentaire factuellement faux** — Mineur, S
`matrix_framework.go:59` (`NumCPU() > 1`) vs `fastdoubling.go:138` (`GOMAXPROCS(0) > 1`) : avec `GOMAXPROCS=1` sur machine multi-cœur, le chemin matriciel spawne des goroutines qui se sérialisent. `common.go:37-47` : le commentaire de `getTaskSemaphore` (« tests that mutate runtime.NumCPU via GOMAXPROCS ») est faux — `NumCPU()` est fixé au démarrage. **Correctif** : aligner sur `GOMAXPROCS(0)` (gate + taille du sémaphore), corriger le commentaire.

**FIB-09 — Calibration : chemin de profil affiché faux avec chemin custom** — Mineur, S
`calibration.go:156-157, 258-259` : charge/écrit au `profilePath` fourni mais affiche systématiquement `GetDefaultProfilePath()`. **Correctif** : afficher le chemin effectif (résoudre une fois, réutiliser).

### 4.3 bigfft

**FFT-03 — Isolation `FFTContext` incomplète** — Mineur, S (doc) / M (plumbing)
`fft_recursion.go:227`, `fft_poly.go:462` vs promesse `context.go:44-46, 68-70` (« cannot cross-contaminate ») : seule l'admission de récursion passe par `ctx.Semaphore` ; butterflies et pointwise prennent leurs jetons sur le sémaphore global. Pas d'interblocage (non bloquant). Vu le statut opt-in/test-only : adoucir la doc suffit.

**FFT-04 (= CONC-02) — `FFTContext.resolved()` mute sans synchronisation** — Mineur, S
`context.go:113-121` : doc « safe for concurrent use once constructed » mais un `&FFTContext{}` manuel partagé entre 2 goroutines produit une data race sur `ctx.Cache`/`Semaphore`. Latent (aucun appelant prod). **Correctif** : documenter « seul `NewFFTContext` produit un contexte concurrent-safe », ou rendre `resolved()` non mutant (valeurs locales sans write-back).

**FFT-05 — Machinerie `fftState` morte en production** — Mineur, M (suppression + docs) / S (marquage)
`pool.go:429-517` (`acquireFFTState`/`releaseFFTState`/`fftStatePool`/`maxPooledFFTTmpCap`), branche `state != nil` de `fourierWithState` (`fft_core.go:12-23`). Grep : tests only ; la prod appelle `fourier()` avec `state=nil`. La garde anti-bloat A2-05 protège un chemin jamais emprunté. Défaut secondaire : l'agrandissement de `state.tmp` abandonne l'ancien buffer sans `releaseFermat` (sans conséquence, code mort). **Correctif** : supprimer le bloc + tests, simplifier `fourierWithState`→`fourier`, **synchroniser CLAUDE.md** (la ligne `bigfft/pool.go` cite `fftStatePool` et A2-05). Alternative conservatrice : marquer « test-only ».

**FFT-06 — `arith_amd64.go` / `arith_generic.go` : corps identiques, build tags sans objet** — Mineur, S
Les deux fichiers délèguent identiquement aux mêmes `go:linkname` — ni « AVX2 dispatch » (annoncé `arith_generic.go:4-5`) ni « pure-Go fallback slower » (`doc.go:19-21`) n'existent. `AddVV/SubVV/AddMulVVW` : appelants = `arith_amd64_test.go` uniquement (gaté amd64). **Correctif** : fusionner en un fichier sans build tag, corriger `doc.go`, retirer le gate amd64 du test.

**FFT-07 — `FromDecimalString` accepte un signe → résultat silencieusement faux** — Mineur, S
`scan.go:12, 61-92` : doc « natural (non-negative) » mais rien ne rejette `-`/`+`. Chemin long (> 1232 chiffres) : le `-` atterrit dans la moitié gauche, `Add` d'une moitié droite positive → **valeur fausse**. Sans appelant de production. **Correctif** : rejeter `s[0]` hors `'0'-'9'` — ou **sans objet si OVR-01 (suppression de `scan.go`) est retenu**.

**FFT-08 — Buffers de pool non relâchés sur les chemins d'erreur de `fourier`** — Mineur, S
`fft_poly.go:255-263` (`transform`), `:305-313` (`invTransform`), `:385-387` (`NTransform`), `context.go:350-352, 377-379` : sur erreur de validation, les acquisitions ne sont pas rendues au pool (pool churn, pas de fuite — GC récupère). **Correctif** : `releaseWordSlice`/`releaseFermatSlice` avant chaque `return PolValues{}, err`.

**FFT-09 — Paramètre `alloc TempAllocator` mort dans les récursions** — Mineur, S
`fft_recursion.go:95`, `fft_recursion_ctx.go:27` : `alloc` n'est que passé récursivement, jamais lu (goroutines utilisent `GetPoolAllocator()`, le séquentiel réutilise `tmp`/`tmp2`). **Correctif** : retirer le paramètre des deux récursions et des 3 sites d'appel. Gate benchstat (hot path).

### 4.4 Couches supérieures

**APP-04 — La TUI sort avec le code 0 sur timeout/SIGINT** — Mineur, S
`tui/handlers.go:88-96` : `handleContextCancelled` fait `done=true` + `tea.Quit` sans renseigner `exitCode` → `tui.Run` retourne 0 là où le CLI retourne 2 (timeout) / 130 (SIGINT) — contrat `exit_action.go`. **Correctif** : mapper `msg.Err` (`DeadlineExceeded`→`ExitErrorTimeout`, `Canceled`→`ExitErrorCanceled`) vers `m.exitCode` avant `Quit`.

**APP-05 — Le restart TUI hérite du timeout absolu de session** — Mineur, M
`app/app.go:163-171` + `tui/handlers.go:130-160` : le restart (`r`) recrée le contexte depuis `m.parentCtx` qui porte le `WithTimeout` absolu posé une fois dans `runTUI`. Restart à T-10s du deadline → 10 s de budget ; après le deadline → restart instantanément annulé (et via APP-04, exit 0). **Correctif** : `WithTimeout` par génération (`handleReset`/`Init`), `signal.NotifyContext` reste sur le parent.

**APP-06 — `IndicatorsMsg` sans champ `Generation` → indicateurs périmés après Reset** — Mineur, S
`tui/messages.go:71-74`, `model.go:121-123` : seul message de calcul sans filtre de génération ; un `computeIndicatorsCmd` de la génération précédente livre après Reset et pollue le panneau du nouveau run. **Correctif** : ajouter `Generation` (depuis `FinalResultMsg.Generation`) et filtrer dans `Update`.

**APP-07 — Mode `--tui` : `--last-digits`, `--memory-limit`, `--output` silencieusement ignorés** — Mineur, M
`app/app.go:116-135, 163-171` : le dispatch part vers `runTUI` avant les branchements correspondants ; aucune erreur ni avertissement. **Correctif** : rejeter les combinaisons en `Validate()` (`ConfigError`, exit 4), ou câbler la validation mémoire dans `runTUI` et documenter le reste.

**SEC-03 (connexe) — Mode `--last-digits` (CLI) : `--memory-limit` ignoré, K non borné** — Info, S
`app/calculate.go:26-29` : dispatch vers `runLastDigits` **avant** `validateMemoryBudget` ; `k` validé seulement `> 0`, `Exp(10, k, nil)` alloue ~k·3,32 bits. Auto-infligé, borné par le timeout. **Correctif** : exécuter `validateMemoryBudget` (ou estimation O(K) dédiée) sur ce chemin, ou borner `k` dans `Validate`.

**APP-08 — `envOverrides` omet `LAST_DIGITS` et `GC_CONTROL`** — Mineur, S
`config/env.go:58-154` : 17 flags couverts, `FIBCALC_LAST_DIGITS`/`FIBCALC_GC_CONTROL` n'existent pas alors que tous leurs pairs ont un équivalent env. `validateEnvOverrides` ne vérifie que table→flags. **Correctif** : ajouter les deux entrées + allowlist « sans-env » vérifiée par test.

**APP-09 — `parseBoolEnv` silencieux contredit le contrat « loud »** — Mineur, S
`config/env.go:167-175` vs `:49-53` : `FIBCALC_QUIET=oui` ignoré sans erreur alors que les overrides numériques échouent en `ConfigError`. **Correctif** : erreur sur booléen non reconnu (helper `malformedEnvError`).

**APP-10 — `orchestration` importe `format` (flèche business→présentation)** — Mineur, M
`orchestration/progress.go:6, 20, 38` : le seul consommateur de `format.ProgressState` dans tout le dépôt est `orchestration.ProgressAggregator` — le type vit dans la mauvaise couche, flèche non couverte par `arch_test.go`. **Correctif** : déplacer `ProgressState` dans `orchestration` (ou `progress`) ; `format` redevient purement string-in/string-out.

**APP-11 — Complétion bash : valeurs erronées pour `--fft-threshold`/`--strassen-threshold`** — Mineur, S
`completion/bash.go:11-13, 84-106` : les trois flags threshold partagent `bashGroupValues` (1024…16384) ; `--fft-threshold` devrait proposer 100000/500000/1000000 (valeurs registre). Zsh/fish/PS corrects. **Correctif** : supprimer `BashGroup`/`bashGroupValues`, émettre un case par flag depuis `f.Values`.

**APP-13 — Incohérences doc/code (4 doc-comments)** — Mineur, S
(1) `config/doc.go:11-12` : « Validate() rejects … memory cap » — faux, le budget est vérifié à l'exécution ; (2) `tui/bridge.go:19-21` : race window décrite n'existe plus (`SetProgram` avant `p.Run()`) ; (3) `metrics/system/doc.go:7-8` : « consumed by metrics and TUI » — seul `tui` consomme ; (4) `cli/doc.go:9-10` : « throughput » non affiché.

### 4.5 Tests

**TEST-01 — `TestColors` tautologique** — Mineur, S
`cli/ui_test.go:117-131` : 9 appels avec retours jetés, aucune assertion ; redondant avec `ui/themes_test.go:166`. **Correctif** : supprimer (ou asserter le contrat).

**TEST-02 — `time.Sleep(10ms)` inutile dans `TestDisplayProgress`** — Mineur, S
`cli/ui_test.go:156` : le canal non bufferisé fournit déjà le rendez-vous ; le fichier frère documente « no time.Sleep needed ». **Correctif** : supprimer la ligne — ou le test entier (sous-ensemble strict de `TestDisplayProgress_LoopCoverage`).

**TEST-04 — Gardiens ADR-0002 dégradables en skip silencieux** — Mineur, S — *le plus important du lot tests*
`bigfft/recursion_extra_test.go:245, 271` (aussi 158, 197) : `if GetParallelFFTRecursionThreshold() > 4 || GetMaxParallelFFTDepth() == 0 { t.Skip }`. Les défauts actuels passent la garde, mais un tuning futur > 4 désactiverait ces gardiens **sans aucun signal** — exactement le mode de régression que CLAUDE.md redoute. **Correctif** : épingler les knobs via `SetFFTParallelismConfig` + `t.Cleanup` (politique sérielle déjà en place dans le fichier), rendant le skip impossible.

### 4.6 Documentation

**DOC-04 — ARCH.md liste encore `tui/component`, supprimé** — Mineur, S
`ARCH.md:156` : package supprimé en b87a342 (postérieur au Docs Sweep). Le répertoire vide subsiste sur disque (non tracké). **Correctif** : retirer la ligne + supprimer le répertoire vide local.

**DOC-05 — CLAUDE.md : commande de régénération du dashboard inexistante** — Mineur, S
`CLAUDE.md:140` : `pnpm --filter @understand-anything/dashboard build:demo` n'existe nulle part ; la procédure réelle est dans BUILD.md:375+. **Correctif** : remplacer par un simple renvoi à BUILD.md.

**DOC-06 — CHANGELOG muet sur la suppression de `tui/component`** — Mineur, S
b87a342 absent du CHANGELOG (tous les autres commits récents couverts). **Correctif** : puce sous « Removed (code mort) ».

**DOC-07 — Date de dernière passe `-race` périmée** — Mineur, S
`TESTING.md:365` : « 2026-06-10 » vs README + CLAUDE.md : verte au **2026-06-21** (dé-flake ec986e0). **Correctif** : corriger la date.

### 4.7 Outillage

**TOOL-02 — `.golangci.yml` : exclusions périmées** — Mineur, S
`:155-166, :136` : (a) `internal/cli/output.go` n'existe plus ; (b) gosec exécuté seul : **aucun G304 émis nulle part** — les deux blocs d'exclusion G304 sont morts ; (c) `dupl` figure dans les exclude-rules alors qu'il n'est pas activé. **Correctif** : supprimer les deux blocs G304 + retirer `dupl`.

**TOOL-03 — Dockerfile : builder installe CGO/GMP sans les utiliser** — Mineur, S
`Dockerfile:11-37` : `build-essential` + `libgmp-dev` + `CGO_ENABLED=1` avec commentaires « compile with the gmp build tag » / « run go test -race » — mais `go build` sans `-tags gmp` et aucun test. Image de build alourdie, binaire lié dynamiquement contredisant « static-friendly ». **Correctif** : soit ajouter `-tags gmp` (l'intention affichée), soit retirer apt/CGO et passer `CGO_ENABLED=0`.

**TOOL-04 — `make check` diverge du gate canonique** — Mineur, S
`Makefile:292` : lint **bloquant** (vs advisory documenté), `format` qui mutile l'arbre (`gofmt -s -w .`) avant de checker, pas de plancher de couverture. **Correctif** : faire pointer `check` sur `scripts/check.sh`, ou supprimer la cible.

---

## 5. Findings informatifs

| ID | Localisation | Résumé | Correctif |
|---|---|---|---|
| SEC-04 | `Dockerfile:11-13, 41` | Images de base non épinglées par digest (`@sha256`) | Épingler + Renovate/Dependabot |
| SEC-05 | `go.mod` | Scan CVE non confirmé (govulncheck : mismatch toolchain go1.25/go1.26) ; deps directes mainstream et à jour | Recompiler govulncheck sous go1.26, exécuter + `go list -m -u all` ; intégrer au gate |
| CONC-03 | `parallel/errors.go:43-51` | `Err()` annoncé « thread-safe » mais lu sans synchro ; appelants réels tous après `wg.Wait()` — pas de race effective | Corriger le commentaire |
| TEST-03 | `calibration_advanced_test.go:99-103` | sleep+cancel de forme (pas de flakiness réelle) | Synchroniser sur l'entrée du mock |
| TEST-05 | `fibonacci_golden_test.go:51` | `SetString` golden sans check `ok` → diagnostic trompeur si entrée corrompue | `t.Fatalf` sur entrée malformée |
| TEST-06 | `test/e2e/cli_e2e_test.go`, `metrics/system/system_test.go` | Absences de `t.Parallel()` non documentées (10 autres cas justifiés) | Ajouter ou commenter |
| FFT-10 | `bigfft/doc.go:16-23` | 4 puces d'invariants périmées (Mutex vs RWMutex, BumpAlloc/Reset, « plan cache », amd64) | Réécrire |
| FFT-11 | `fft_cache.go:67-69, 102-108` | `SetCacheLogger` : « wiring layer » inexistant — logger prod = Nop, stats cache jamais émises | Câbler le logger applicatif ou ajuster le commentaire |
| FFT-12 | `fft_cache.go`, `fft_poly.go` | Exports sans consommateur prod hors ADR (`MulCached`, `SqrCached`, `TransformCached`, `Poly.Mul`, `PolValues.Clone`, `computeKey`) | Cf. OVR-10 ; `computeKey` seul strictement gratuit |
| FFT-13 | `fft_core.go:104`, `context.go:295`, `scan.go:28`, `fft_poly.go:371-373` | Micro-incohérences (`rp.M = m` redondant, message de panic `<` vs `<=`, boucle → `clear()`) | Nettoyage opportuniste |
| APP-14 | `config/config.go:233` | Erreur de `Validate` remplacée par `errors.New("invalid configuration")` — perd le `ConfigError` typé | Retourner `err` |
| APP-15 | `display.go:322-324`, `chart.go:47-50` | Paramètres morts (`FormatQuietResult` ignore `n`, `duration` ; `AddDataPoint` ignore `progress`) | Retirer |
| APP-16 | `app/calculate.go:250` | `saveResultIfNeeded` écrit sur `os.Stderr` au lieu de `a.ErrWriter` ; double check `OutputFile == ""` | Corriger |
| APP-17 | `cli/calculate.go:19-41` vs `tui/logs.go:68-89` ; `orchestration/progress.go:95-101` vs `tui/metrics.go:58-74` | Duplications (bloc « Execution Configuration », EMA 70/30, `NullProgressReporter` ≡ `DrainChannel`+`wg.Done`) | Candidats si l'un des côtés évolue |
| APP-18 | `metrics/indicators.go:139` | Zero-padding : `x.BitLen() > n*4` rate les nombres de 20-24 chiffres (10²⁰ ≈ 67 bits) → « Last 20 digits » peut afficher `0` au lieu de `0…0` | Seuil ~n·3,33 ou test sur le nombre de chiffres |
| DOC-08 | `CONTRIBUTING.md:33-34` | URL de clone du fork `fibcalc.git` vs repo `FibGo` | Corriger |
| DOC-09 | `docs/dashboard/knowledge-graph.json` | Graphe antérieur à la suppression de `tui/component` (meta f4d3a7f) | Régénérer selon BUILD.md (artefact) |
| OVR-08 | `bigfft/fft.go:51` | `SetFFTThreshold` : commentaire « Intended for calibration » faux (calibration n'appelle que `bigfft.Mul`) ; justification ADR-0004 B1 caduque côté calibration | Corriger le commentaire + annoter l'ADR |
| TOOL-05 | `scripts/check.sh:47+68`, `check.ps1:59+82`, `Makefile:182` | Les gates exécutent la suite **deux fois** (test puis re-run `-coverprofile`) | Fusionner en une étape `-coverprofile` |
| TOOL-06 | `Makefile` | `stats`/`coverage-check` POSIX-only sans le marqueur ; aucune cible cassée ; go.mod propre | Ajouter le marqueur |

---

## 6. Code mort & sur-ingénierie (inventaire consolidé)

Vérifié par `deadcode` (x/tools) depuis les mains de production + greps systématiques prod vs tests. Total estimé : **~400-500 LOC supprimables sans risque hors hot path**. Les rétentions déjà actées par ADR-0008 (R4/R5/R6…) sont exclues.

| ID | Localisation | Contenu mort | Effort |
|---|---|---|---|
| OVR-01 | `bigfft/scan.go` (98 LOC, fichier entier) | `FromDecimalString` + type `scanner` : zéro consommateur prod (API héritée upstream jamais branchée). Supprimer avec `scan_test.go` (192 LOC) → rend FFT-07 sans objet | S |
| OVR-02 | `ui/themes.go:52-95, 187-215` | `LightTheme`, `OrangeTheme`, `SetTheme` inatteignables (seul `InitTheme` → dark/none en prod) ; `SetCurrentTheme` requis par les tests → documenter test-only | S |
| OVR-03 | `fibonacci/common.go:77` | `preSizeBigInt` dupliqué à l'identique de `memory/arena.go:101` (la copie est morte) | S |
| OVR-04 | `threshold/manager.go:138, 291, 313, 334, 340` | Résidus du refactor 1f394e0 : `analyzeFFTThreshold`, `analyzeParallelThreshold`, `avgTimePerBit`, `significantChange` (supplantés par `*From(metrics)`) + constructeur jumeau `NewDynamicThresholdManager` (prod = `FromConfig`) | S |
| OVR-05 | `fibonacci/` | `AcquireState` (prod = `AcquireStateForN`), `ShouldParallelizeMultiplication` (wrapper), `SetDefaultStrassenThreshold`, `MustNewCalculator` (zéro prod mais **vanté par doc.go**), `setOrReturn` (mort) | S |
| OVR-06 | `fibonacci/registry.go:249, 259` | `GlobalFactory()` + `RegisterCalculator` : registre à plugins **sans plugins** (app.New fait `NewDefaultFactory()` direct) | S |
| OVR-07 | `errors/errors.go:180-212, 244, 259` | `TimeoutError`, `ValidationError` jamais construites ; `WrapError`, `IsContextError` test-only | S |
| OVR-09 | `tui/sparkline.go:33-113` | `RenderBrailleChart` + `brailleDots` + `plotBrailleValue` (~65-80 LOC) ; `renderBrailleSection` (chart.go:149) = nom menteur (rend des sparklines) | S |
| OVR-11 | `progress/observer.go:18` | `RecoveredObserverCount` : **zéro référence, tests compris** — compteur write-only | S |
| FFT-05 | `bigfft/pool.go:429-517` | Machinerie `fftState` complète (cf. §4.3) — décision : suppression + sync CLAUDE.md, ou marquage test-only | M/S |
| OVR-10 | `bigfft/` (cluster) | API legacy non-bump servant d'oracle de tests (`Poly.*`, `TransformCache.Get/Put/Clear`, `*Cached` non-bump, `computeKey`, `AddVV/SubVV/AddMulVVW`) — **ne pas tailler sans ADR** ; documenter « oracle de test » | M |
| OVR-12 | multiples | Lot d'exports test-only résiduels (`GetVersionInfo`, `FlagNames`, `DisplayResultWithConfig`, `DisplayMemoryStats`, `GenerateQuickFFTThresholds`, `FormatProgressBarWithETA`, `NumCalculators`, `ErrorCollector.Reset`, `GCController.SetLogger/Stats`, accessors `threshold`, `MetricsBuffer.Written`, `SetCacheLogger`). À GARDER : `app.WithFactory` (seam légitime), `config.validateEnvOverrides` (self-check documenté) | M total |
| APP-12 reliquats | `cli/ui.go:16-18`, `format/eta.go:75-89`, `metrics/indicators.go:29` | `HexDisplayEdges` (zéro usage), `FormatProgressBarWithETA`, champ `Indicators.Live` écrit jamais lu | S |
| FIB-10 reliquats | `calibration/microbench.go:25`, `memory/arena.go:62/122` | `MicroBenchPerTestTimeout` (mort strict — zéro référence), `AllocBigInt`/`UsedWords` (test-only, lié à FIB-05) | S |

---

## 7. Vérifications positives (catégories saines)

### 7.1 Tests gardiens — les 12 existent

| Gardien | Emplacement |
|---|---|
| `TestReleaseState_OverLimit_AliasesCleared` | `fibonacci/state_pool_arena_test.go:154` |
| `TestCalculatorStateCache_OverLimitNotCached` | `fibonacci/state_cache_test.go:73` |
| `TestStateBump_*` (3) | `fibonacci/state_cache_test.go:125,157,175` |
| `TestWireThresholdTuning` | `app/app_tuning_test.go:18` |
| `TestConcurrentAccess` | `threshold/manager_test.go:529` |
| `TestFermatPostConditionPanicClassifier` | `bigfft/fft_recover_policy_test.go:11` |
| `TestFourierRecursiveAsyncPanicPropagates` (+Ctx) | `bigfft/recursion_extra_test.go:240,265` |
| `TestExecuteReconstructionPanicPropagates` | `bigfft/recursion_extra_test.go:126` |
| `TestPointwiseWorkerPanicPropagates` | `bigfft/fft_pointwise_parallel_test.go:77` |
| `TestArchitectureLayering` | `internal/arch_test.go:65` |
| `TestReleaseWordSliceAllExactBuckets` | `bigfft/pool_release_test.go:119` |

### 7.2 Autres vérifications concluantes

- **Injection shell (completion)** : étanche — `escape.go` centralise un échappement correct par dialecte (bash double-quoted, zsh single + argspec distinct, fish, powershell) ; chaque générateur route par le bon helper. Vecteur maintenu fermé.
- **Concurrence** : aucune autre data race que M4 ; toutes les goroutines de production sont appariées `wg` + jonction (y compris `runPassSequence` sur tous ses chemins de sortie) ; aucun deadlock possible (sémaphore fibonacci jamais imbriqué — corps de tâches = feuilles ; sémaphore bigfft non bloquant) ; discipline `sync.Pool` conforme ; unique ticker avec `defer Stop` ; `signal.NotifyContext` toujours accompagné de `defer stop`.
- **Arithmétique bigfft (chemin principal)** : fermat `norm`/`Shift`/`Mul`/`Sqr`, reconstruction butterfly, `IntTo`, tailles de buffers `8n+1`/`n+1`, index de buckets — revérifiés borne par borne, corrects.
- **Cœur fibonacci** : `finalizeStateReleaseTo` respecte son contrat d'ordre ; slot GC-immune borné et jamais alimenté hors chemin ; identités mathématiques de `FastDoublingMod`, Strassen-Winograd et `modular.go` vérifiées ligne à ligne.
- **Entrées** : env vars malformées rejetées bruyamment ; `satMul`/`satAdd` dans `EstimateMemoryUsage` ; écriture de profil atomique (temp → chmod 0600 → rename + backoff Windows).
- **Docs** : les 30+ symboles cités dans la table d'invariants CLAUDE.md existent tous ; golden = 26 entrées avec F(50k/100k/200k) ; `.golangci.yml` = 24 linters conformes ; diagrammes C4 et `dependency-graph.mermaid` = imports réels ; versions Go cohérentes partout (1.26).
- **Outillage** : aucune cible Makefile cassée ; `go.mod` propre (rien à tidy) ; `test/e2e` branché ; Dockerfile multi-stage distroless nonroot correct (hors TOOL-03/SEC-04).
- **Tests** : aucun test mort, aucune fixture orpheline, aucune flakiness avérée ; couverture 82,5-100 % partout ; contrainte Windows sans `-race` respectée (passes via WSL).

---

## 8. Récapitulatif et priorisation recommandée

1. **Immédiat (S, code)** : M1 APP-01 (panic), M2 FIB-01 (--gc-control), M3 APP-03 (fish), M4 CONC-01 (race spinner), M7 FFT-01 (pool sale), FIB-04 (satMul).
2. **Court terme (M, code)** : M5 FIB-02 + M6 FIB-03 (justesse calibration), M8 FFT-02 (panic sync / wg.Wait).
3. **Processus** : M9 baseline benchstat (une commande + un commit — débloquer le gate perf avant tout correctif perf-sensitive).
4. **Docs (S)** : M10, M11, DOC-04..08, doc-comments (APP-13, FFT-10, FIB-06, CONC-03, OVR-08).
5. **Hygiène** : mineurs restants, code mort (§6), outillage (TOOL-02..05), tests (TEST-01..06).
6. **Décisions à acter** (ADR court ou choix explicite) : FIB-05 (arène ×15→×5), FFT-05/OVR-10 (purge vs marquage oracle bigfft), APP-07 (flags en TUI), TOOL-03 (gmp vs CGO=0), politique baseline (option a vs b).

> Le plan d'exécution détaillé, ordonné et vérifiable de la totalité de ces correctifs et bonifications est dans [`auditPlan.md`](auditPlan.md).
