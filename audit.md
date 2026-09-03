# Audit de code FibGo — 2026-09-03

Régime : production. Le code est versionné et le rapport sert de base d'action.

## 0. Périmètre, méthode, état des gates

**Périmètre.** Tout le code Go de production : `cmd/` et `internal/`, 17 900 lignes hors tests, lues en totalité. Les 58 900 lignes de tests n'ont pas été auditées ligne à ligne ; elles ont été consultées pour établir ce qui est déjà couvert. Le backend GMP (`internal/fibonacci/calculator_gmp.go`, build tag `gmp`) n'est pas compilable ici et n'a pas été exécuté.

**Méthode.** Lecture intégrale, puis vérification outillée, puis trois sondes temporaires (code en annexe A) pour confirmer les soupçons H-01, H-03 et M-01 par mesure plutôt que par déduction. Les sondes ont été retirées de l'arbre.

**État des gates sur cet hôte** (Windows 11, `go1.27.0`, sans CGO) :

| Gate | Résultat |
|---|---|
| `go build ./...` | OK |
| `go vet ./...` | OK |
| `go test -short -count=1 ./...` | OK, 21 paquets |
| `gofmt -l`, `go mod tidy -diff`, `go mod verify` | propres |
| `golangci-lint run ./...` (v1.64.8) | **inopérant** : `export data version 4 is greater than maximum supported version 2` (voir GATE-01) |
| `staticcheck ./...` (2026.1) | **inopérant**, même cause |
| `go test -race` | non exécutable (pas de CGO) |
| `-tags gmp` | non exécutable (pas de libgmp) |

**Antécédents.** Deux audits exhaustifs ont déjà été exécutés (ADR-0008, ADR-0009, CHANGELOG 4.0.0 et Unreleased). Ce rapport ne re-suggère aucun candidat qu'ils ont rejeté sur preuve (multiplicateur d'arène, oracles bigfft, `ColorProvider`, `executeTasks`, `CacheStrategy`, observers `progress`, `TestFactory`, knobs `threshold`). Chaque constat ci-dessous est nouveau ou apporte un élément de preuve nouveau.

**Note sur `audit Gemini.md`.** Ce document, présent dans l'arbre de travail, décrit un code qui n'existe pas dans ce dépôt : un cache LRU de résultats `*big.Int` indexé par `n`, un type `FibCalculator.cache`, une fonction `fastDoublingScalar`, une « version 2.0 » ciblant Go 1.22. La section 5 confronte chacune de ses six anomalies au code réel. Résultat : aucune ne s'applique telle qu'énoncée ; deux d'entre elles pointent vers des constats voisins réels, intégrés ici sous M-08 et L-11.

## 1. Synthèse

| ID | Sévérité | Composant | Constat |
|---|---|---|---|
| H-01 | Haute | tui | Le code de sortie d'un calcul terminé est écrasé par une annulation de contexte ultérieure (timeout ou `q`) |
| H-02 | Haute | calibration, config | La sentinelle `-1` (séquentiel) est persistée dans le profil, que `config.Validate` rejette : le profil est jeté en silence à chaque démarrage |
| H-03 | Haute | memory, app | `--memory-limit` sous-estime la mémoire réelle d'un facteur 3 à 6 (mesuré) |
| GATE-01 | Gate | outillage | Le lint est mort sur toolchain 1.27 et `check.ps1` le traite comme consultatif : plus aucune analyse statique au-delà de `go vet` |
| M-01 | Moyenne | calibration | Le micro-benchmark FFT compare des charges identiques sous 1800 mots et oscille d'un facteur 4 entre deux exécutions, avec une confiance fixe de 0,70 |
| M-02 | Moyenne | config, app | `--memory-limit` malformé et `--last-digits` hors borne échappent à `Validate` ; erreurs incohérentes selon le mode |
| M-03 | Moyenne | app, config | Un profil en cache écrase un `--threshold` explicite (décision documentée « ouverte » depuis trois audits) |
| M-04 | Moyenne | fibonacci/threshold, docs | Les seuils dynamiques (DTM) ne sont atteignables par aucun flag ; le gain mesuré en ADR-0001 n'est jamais livré ; la doc les présente comme une fonctionnalité |
| M-05 | Moyenne | bigfft | Les `acquire*` des pools zéroïsent toute la capacité du bucket, jusqu'à 4× la taille demandée |
| M-06 | Moyenne | app, orchestration | Divergence d'algorithmes en mode `--quiet` : exit 3 sans un mot sur stderr ; message non-quiet sans retour à la ligne |
| M-07 | Moyenne | tui | `SIGINT` reçu par signal (stdin non-TTY) → « TUI error » et exit 1 au lieu de 130 |
| M-08 | Moyenne | bigfft | `TransformCache` est borné en nombre d'entrées, pas en octets ; la rétention croît linéairement avec `n` sur le chemin `matrix` |
| L-01 | Basse | progress | `ReportStepProgress` traîne trois paramètres morts et deux helpers qui n'existent que pour les alimenter |
| L-02 | Basse | divers | Code de production sans appelant (liste en 2.12) |
| L-03 | Basse | progress, cli | Envoi non bloquant : la mise à jour finale 1.0 peut être perdue et le CLI affiche « interrupted » sur un succès |
| L-04 | Basse | app | `HasVersionFlag` ne s'arrête pas à `--` et ignore `--version=false` |
| L-05 | Basse | config | `--help` avec `--machine` ou `-q` émet des codes ANSI |
| L-06 | Basse | bigfft | `fourierRecursiveUnified` transporte un `alloc` jamais consommé : le bump allocator n'atteint jamais la récursion |
| L-07 | Basse | calibration | Le mode `--calibrate` réutilise un seul agrégateur de progression sur toutes les passes : ETA sans signification |
| L-08 | Basse | fibonacci | `decideCacheTuning` : croissance qui stagne pour `MaxEntries ≤ 4`, `MinBitLen` sans plafond |
| L-09 | Basse | dépôt | Worktree `.claude/worktrees/lucid-nightingale-3e810f` en HEAD détaché, copie complète de l'arbre |
| L-10 | Basse | format, errors, memory | Trois copies de `FormatBytes` ; couplage accepté par l'architecture en couches, aucune action |
| L-11 | Basse | cli | La conversion décimale `big.Int.String()` est recalculée jusqu'à quatre fois pour un même résultat |

## 2. Constats détaillés

### 2.1 H-01 — Code de sortie TUI écrasé après complétion

**Fichiers.** `internal/tui/handlers.go:90-104` (`handleContextCancelled`), `:76-85` (`handleCalculationComplete`), `internal/tui/commands.go:130-136` (`watchContextCmd`), `internal/tui/model.go:59` (timeout par génération).

**Preuve (sonde A.3, exécutée).**

```
after complete:            done=true exitCode=0
after deadline post-done:  exitCode=2   quit=true
after cancel post-done:    exitCode=130
```

`handleContextCancelled` ne consulte pas `m.done`. Or `watchContextCmd` reste armé après la fin du calcul, sur un contexte dont le délai (`-timeout`, 5 min par défaut) continue de courir.

**Impact.**
- Un utilisateur qui laisse le tableau de bord ouvert plus de `-timeout` après un calcul réussi voit le TUI se fermer seul avec le code 2 (« timeout »).
- Appuyer sur `q` après complétion appelle `m.cancel()` puis `tea.Quit`. Le `ContextCancelledMsg` déclenché par `cancel()` et le `QuitMsg` arrivent dans un ordre non déterministe ; si le premier gagne, le processus sort avec 130 au lieu de 0. Un script qui enchaîne sur le code de sortie est cassé de façon intermittente.

**Correctif.** Dans `handleContextCancelled`, ignorer le message quand `m.done` est vrai : `if m.done { return nil }`. Le code de sortie du calcul est préservé ; le TUI ne se ferme que sur action de l'utilisateur. `SIGINT`/`SIGTERM` restent pris en charge par bubbletea lui-même (voir M-07), pas par ce chemin.

**Vérification.** Test `TestModel_ContextCancelledAfterDone_KeepsExitCode` : `CalculationCompleteMsg{ExitCode: 0}` puis `ContextCancelledMsg{DeadlineExceeded}` → `exitCode == 0`, `cmd == nil`. Même chose avec `context.Canceled`. Les quatre tests existants `TestModel_Update_ContextCancelledMsg*` (`model_test.go:337-390`) couvrent uniquement le cas « pas encore terminé » et restent verts.

### 2.2 H-02 — Sentinelle `-1` persistée, profil rejeté en silence

**Fichiers.** `internal/calibration/adaptive.go:28,61-71,78` (candidats `-1`), `calibration.go:140-146` (le code sait que `-1` n'est pas un flag valide), `:253-261` (`persistCalibrationProfile`), `:434-444` (`finalizeStrategyResult`), `strategy_complete.go:68-69,80`, `internal/config/config.go:103-108` (`Validate` refuse `< 0`), `internal/app/app.go:93-97`.

**Chaîne de causalité.** FIB-02 (audit 2026-07) a introduit `-1` comme vrai candidat séquentiel dans les trois générateurs de seuils. Quand `-1` gagne (machines à 1-2 cœurs, ou quand la FFT ne paie pas), il est écrit tel quel dans `OptimalParallelThreshold` ou `OptimalFFTThreshold` du profil. Au démarrage suivant :
- `app.New` charge le profil, appelle `cfgWithProfile.Validate`, qui échoue sur `threshold cannot be negative`, et retombe **sans message** sur les seuils adaptatifs (parallèles) ;
- avec `--auto-calibrate`, `AutoCalibrateWithProfile` (`calibration.go:343-350`) affiche « invalid thresholds, re-calibrating », relance `FastStrategy`, qui ne mesure pas le parallélisme et écrit 4096 par défaut. Le résultat de la calibration complète est donc perdu et remplacé par un défaut.

**Impact.** Sur les machines où le séquentiel est optimal, la calibration ne sert à rien et l'utilisateur n'en est jamais informé. `--calibrate` recommande à l'écran « Sequential (no parallelism) » tout en sauvegardant un profil inapplicable.

**Correctif.** Faire de la valeur « désactivé » un citoyen de première classe plutôt qu'une sentinelle interne :
1. `config.Validate` accepte `-1` pour `Threshold` et `FFTThreshold` (« -1 disables ») ; `0` reste « auto ». `StrassenThreshold` inchangé (aucun générateur ne produit `-1`).
2. Aide des flags, `README.md` (table des flags), `.env.example`, registre de complétion (`Values`) mis à jour.
3. Vérifier que tous les consommateurs traitent `-1` comme désactivé : c'est déjà le cas (`fastdoubling.go:142` et `matrix_framework.go:59` testent `> 0` ; `fft.go:62,80` et `strategy.go:101` testent `> 0`). `cli.PrintExecutionConfig` doit afficher « disabled » plutôt que `-1`.
4. `AutoCalibrateWithProfile` : la garde `>= 0` devient `>= -1`.

**Vérification.** `TestValidate_MinusOneDisables` (config), `TestNewCachedProfileSequentialThresholdApplied` (app : profil avec `-1` → `cfg.Threshold == -1`, pas de repli adaptatif), test aller-retour `RunCalibration` → `LoadCachedCalibration` → `Validate == nil` quand `-1` gagne (forcer via `GenerateParallelThresholds` sur un runtime à 1 CPU, `runtime.GOMAXPROCS` ne suffit pas : `NumCPU` est lu ; injecter via un seam ou tester `persistCalibrationProfile` directement).

### 2.3 H-03 — Estimation mémoire sous-évaluée d'un facteur 3 à 6

**Fichiers.** `internal/fibonacci/memory/budget.go:20-43` (`EstimateMemoryUsage`), `internal/bigfft/pool_warming.go:52-101`, `internal/config/validator.go:51-77`, `internal/app/calculate.go:97-129`.

**Preuve (sonde A.1, exécutée, calculateur `fast`, `Options{}` par défaut).**

| n | Estimation `TotalBytes` | `Sys` delta mesuré | `HeapSys` après | `TotalAlloc` delta |
|---|---|---|---|---|
| 10 M | 12 MB | 67 MB | 69 MB | 73 MB |
| 100 M | 124 MB | 381 MB | 445 MB | 729 MB |

**Causes identifiées par lecture.**
- L'estimation compte 5 `big.Int` pour l'état ; l'arène réelle en alloue 10 fois la taille de F(n) (`arenaTotalWords`, ADR-0009 R4).
- Le pré-chauffage des pools (`PreWarmPools`) alloue `numBuffers` (6 pour n ≥ 10 M) tampons de la classe de taille supérieure : à F(100 M), 6 × 32 MB de `[]big.Word` plus 6 × 16 MB de `fermat`, soit 288 MB avant la première multiplication. Rien de cela n'est modélisé.
- Les `PolValues` d'un pas de doublement FFT (2 transformées avant + 3 produits + 3 inverses, vivants simultanément en mode parallèle) pèsent chacun ≈ 2 × taille de l'opérande.
- Le budget « cache » (2 × F(n)) ne correspond pas au comportement réel de `TransformCache` (voir M-08).

**Impact.** Le flag existe pour éviter un OOM ; il valide des exécutions qui consomment jusqu'à six fois le budget déclaré. Le message « Memory estimate: … (limit: …) » est faux.

**Correctif.** Re-modéliser `EstimateMemoryUsage` sur les composants réels : arène (×10 mots), pré-chauffage (fonction de `EstimateMemoryNeeds` et de `numBuffers`), jeu de travail FFT (8 `PolValues` à la taille finale), cache (borne de M-08). Calibrer contre `Sys` mesuré à n ∈ {1 M, 10 M, 50 M, 100 M} pour les trois algorithmes, `--algo all` inclus. Critère : estimation ≥ `Sys` delta mesuré à chaque point, et ≤ 2 × mesuré (une estimation trop prudente refuse des calculs faisables).

**Vérification.** Test de table `TestEstimateMemoryUsage_Envelope` avec les points mesurés en dur (bornes larges pour absorber le bruit inter-machines) ; sonde A.1 rejouée après correction ; `docs/PERFORMANCE.md` et `.env.example` (« estimation préalable ») ajustés.

### 2.4 GATE-01 — Lint inopérant, gate consultatif

**Fichiers.** `.golangci.yml` (en-tête : « verrouillé sur v1.64.8 »), `scripts/check.ps1:88-105`, `scripts/check.sh:109-118`.

**Constat.** Le binaire `golangci-lint` v1.64.8 installé a été compilé avec go1.26.3 ; la toolchain active est go1.27.0, qui émet des export data version 4 que l'importateur embarqué (x/tools) ne lit pas. Toute exécution échoue en typecheck sur `math/bits`, `sync/atomic`, etc. `staticcheck` 2026.1 échoue de même. Les deux scripts de gate déclarent le lint « soft » : ils affichent l'échec, écrivent `Overall: PASS` et sortent 0. Il n'y a pas de CI distante (ADR-0004 B3, mise à jour 2026-06-21). Conséquence : `gosec`, `gocritic`, `revive`, `errcheck`, `shadow`, `unused` ne tournent plus nulle part.

**Correctif.** Deux options, à trancher :
- **A (recommandée).** Migrer vers golangci-lint v2 (backlog A5-07, désormais forcé : la ligne v1 ne suit plus la toolchain). Convertir `.golangci.yml` (`golangci-lint migrate`), re-baseliner les exclusions documentées dans l'en-tête des scripts, puis rendre le lint **dur** dans les deux scripts.
- **B (transitoire).** Épingler la toolchain d'analyse : `GOTOOLCHAIN=go1.26.x golangci-lint run` dans les scripts. Nécessite le téléchargement de la toolchain 1.26 (aucune en cache sur cet hôte) et fige le lint sur une version de Go différente de celle qui compile.

Dans les deux cas : un lint qui ne peut pas s'exécuter doit faire échouer le gate (distinguer « absent » de « en erreur » ; aujourd'hui les deux passent).

**Vérification.** `scripts/check.ps1` sort non-zéro quand `golangci-lint` retourne une erreur de typecheck ; lint vert sur l'arbre après migration.

### 2.5 M-01 — Micro-benchmark FFT non fondé et instable

**Fichiers.** `internal/calibration/microbench.go:34-39` (tailles 500, 2000, 8000, 16000 mots), `:156` (sémaphore `NumCPU`), `:252-258` (`multiplyTest` passe par `bigfft.Mul`), `:298-301` (+0,2 de confiance), `:321-355` (`findFFTCrossover`, `size * 64` en `:344`), `internal/bigfft/fft.go:32,70-75` (seuil interne 1800 mots).

**Preuve (sonde A.2, six exécutions consécutives, machine au repos).**

```
run 0: FFTThreshold=460800 bits  conf=0.70
run 1-5: FFTThreshold=115200 bits  conf=0.70
```

**Analyse.**
- `bigfft.Mul` n'utilise la FFT qu'au-dessus de 1800 mots. À la taille 500, les configurations « FFT » et « std » exécutent exactement le même code `math/big` ; tout écart est du bruit, et le bruit suffit à déclarer un crossover à 500 × 64 × 0,9 = 28 800 bits. Cela ne s'est pas produit en six essais, mais le code le permet.
- 115 200 = 2000 × 64 × 0,9 : le crossover « mesuré » est en réalité le seuil interne de bigfft. 460 800 = 8000 × 64 × 0,9. Le résultat bascule entre les deux selon le bruit à la taille 2000, avec la même confiance 0,70 ≥ 0,50 : il est accepté et **persisté** par `tryFastThenEscalate`.
- Les 16 configurations tournent en parallèle sous un sémaphore `NumCPU`, et `bigfft.Mul` parallélise lui-même sa récursion : les timings mesurent de la contention, pas la multiplication.
- `size * 64` suppose un mot de 64 bits.

**Impact.** Deux démarrages successifs avec `--auto-calibrate` peuvent installer un seuil FFT de 115 K ou 460 K bits, contre 500 K par défaut. Le chemin de doublement FFT (`executeDoublingStepFFT`) n'est pas ce que le micro-benchmark mesure ; aucune donnée ne dit que 115 K est meilleur pour lui.

**Correctif.**
1. Exclure de la comparaison FFT les tailles ≤ `defaultFFTThresholdWords` (ou exposer une entrée bigfft qui force la FFT pour le benchmark).
2. Exécuter les configurations **séquentiellement** ; garder la concurrence uniquement si elle est mesurée comme neutre.
3. Exiger une marge (FFT plus rapide d'au moins 10 %, comme `findParallelCrossover`).
4. Dériver la confiance de la dispersion (rapport min/max sur les `Iterations`) plutôt que d'un bonus fixe.
5. `bits.UintSize` à la place de 64.

**Vérification.** Sonde A.2 : dix exécutions donnent le même seuil, ou des seuils différents avec une confiance < 0,5. Test unitaire de `findFFTCrossover` avec un jeu de résultats synthétique où la taille 500 « gagne » par bruit : aucun crossover rapporté.

### 2.6 M-02 — Validation de configuration incomplète

**Fichiers.** `internal/config/config.go:99-144`, `internal/app/calculate.go:24-33,135,150-153`, `internal/app/app.go:176`.

**Constat.** `Validate` ne parse pas `MemoryLimit` ; le parsing n'a lieu que dans `validateMemoryBudget`, appelé sur les chemins calcul complet et TUI. `fibcalc --last-digits 5 --memory-limit 4GB` (suffixe invalide) et `fibcalc --calibrate --memory-limit xyz` s'exécutent sans un mot. La borne `maxLastDigits` vit dans `runLastDigits` : l'erreur n'affiche pas l'usage, contrairement aux autres erreurs de configuration, et une valeur `FIBCALC_LAST_DIGITS` hors borne passe `Validate`.

**Correctif.** Déplacer dans `Validate` : `memory.ParseMemoryLimit` (erreur `ConfigError`, usage affiché, exit 4) et `LastDigits ≤ maxLastDigits` (constante déplacée dans `config`). `validateMemoryBudget` conserve l'estimation et la comparaison.

**Vérification.** Cas de table dans `config_test.go` ; test d'intégration `--last-digits 5 --memory-limit 4GB` → exit 4.

### 2.7 M-03 — Le profil en cache écrase les flags explicites

**Fichiers.** `internal/config/thresholds.go:3-24`, `internal/app/app.go:93-97`, `internal/calibration/calibration.go:450-460`, `README.md` (section Configuration), `.env.example` (en-tête).

**Constat.** Documenté trois fois comme « KNOWN SURPRISE », épinglé par `TestNewCachedProfileOverridesExplicitFlags`, et laissé « open behavior decision ». Un utilisateur qui tape `--fft-threshold 800000` obtient la valeur du profil sans avertissement. Chaque audit a repoussé la décision ; ce rapport la met dans le plan comme item à trancher (phase 0), avec une recommandation.

**Recommandation.** Le flag explicite gagne. `ParseConfig` enregistre `ThresholdExplicit`, `FFTThresholdExplicit`, `StrassenThresholdExplicit` (via `isFlagSetAny` et présence de la variable d'environnement) ; `LoadCachedCalibration` ne remplit que les seuils non explicites. Le test épinglant est inversé. README, `.env.example` et `thresholds.go` simplifiés (l'exception documentée disparaît).

**Alternative.** Conserver le comportement et avertir sur stderr quand le profil écrase un flag explicite. Moins de code, mais l'utilisateur reste privé de son réglage.

### 2.8 M-04 — Seuils dynamiques (DTM) inatteignables

**Fichiers.** `internal/fibonacci/options.go:46-53`, `fastdoubling.go:148-163`, `internal/config/config.go:150-179` (aucun flag), `internal/app/app.go:40-64` (`wireThresholdTuning`), `internal/config/threshold_tuning.go`, `internal/fibonacci/cache_strategy_bigfft.go`, `docs/ARCH.md:295,392`, `docs/PERFORMANCE.md:316-342`, `docs/CALIBRATION.md:7`, ADR-0001.

**Constat.** `EnableDynamicThresholds` n'est mis à `true` par aucun chemin de production : ni flag, ni variable d'environnement, ni `app`. Toute la chaîne `threshold/` (manager, analyzer, buffer), `CacheStrategy`, `decideCacheTuning`, `wireThresholdTuning` et `ThresholdTuningProfile` (sauf `MicroBenchTimeout`) n'est exécutée que par les tests et `BenchmarkFibonacciDTM`. ADR-0001 a décidé KEEP sur la foi d'un gain de 5-6 % à F(10 M) ; ce gain n'est livré à aucun utilisateur du binaire. La documentation d'architecture présente le DTM comme un composant actif du pipeline.

**Ce que ce rapport ne fait pas.** Il ne rouvre pas la décision KEEP d'ADR-0001. Il constate que la décision n'a pas été suivie d'un câblage et que la doc surestime.

**Options (phase 0).**
- **A.** Câbler un flag `--dynamic-thresholds` (et `FIBCALC_DYNAMIC_THRESHOLDS`), désactivé par défaut, puis mesurer avec `benchstat -count=10` à 1 M / 10 M / 100 M. Si le gain ≥ 5 % se confirme, passer le défaut à `true` dans une release ultérieure.
- **B.** Déclarer le DTM « library-only » : mise à jour de `docs/ARCH.md`, `docs/PERFORMANCE.md`, `docs/CALIBRATION.md` et d'ADR-0001 (status note), et suppression de `wireThresholdTuning` + `ThresholdTuningProfile` (sauf `MicroBenchTimeout`) qui ne configurent rien d'atteignable.

L'option A coûte peu (un flag, un champ, un test) et donne enfin une mesure en conditions réelles. L-08 (heuristiques de `decideCacheTuning`) se corrige dans la foulée si A est retenue, et devient sans objet sinon.

### 2.9 M-05 — Pools : zéroïsation de toute la capacité du bucket

**Fichiers.** `internal/bigfft/pool.go:80-89` (`acquireWordSlice`), `:218-226` (`acquireFermat`), `:305-313` (`acquireNatSlice`), `:392-400` (`acquireFermatSlice`).

**Constat.** Chaque `acquire*` fait `clear(slice)` sur le tampon complet du pool puis retourne `slice[:size]`. Les classes de taille sont espacées d'un facteur 4 : une demande de 65 537 mots obtient un tampon de 262 144 mots et en zéroïse 262 144. En moyenne, le `memclr` fait 2,5 × le travail utile ; au pire 4 ×. Les appelants n'utilisent que `[:size]` et `release*` restaure `[:cap]` : `clear(slice[:size])` est équivalent et suffisant.

**Correctif.** `clear(slice[:size])` dans les quatre fonctions (une ligne chacune).

**Vérification.** Protocole ADR-0009 R4 : `benchstat` A/B, `-count=10 -benchtime=1s`, `BenchmarkFibonacci/(FastDoubling|FFTBased)` à 1 M et 10 M, ordre inversé pour exclure le thermique. Attendu : neutre à positif ; régression > 5 % = rejet.

### 2.10 M-06 — Divergence en mode quiet sans message ; retour à la ligne manquant

**Fichiers.** `internal/app/calculate.go:217-226`, `internal/orchestration/orchestrator.go:170-173`.

**Constat.** `--algo all --quiet` avec des résultats divergents : `present` retourne `ExitErrorMismatch` sans rien écrire. Le script appelant voit stdout vide et code 3, sans explication. En mode normal, le message « CRITICAL ERROR! » n'a pas de `\n` final.

**Correctif.** Écrire « Global Status: CRITICAL ERROR … » sur `a.ErrWriter` dans la branche quiet ; ajouter `\n` dans `AnalyzeComparisonResults`.

**Vérification.** Test `TestRunQuietMismatchWritesStderr` avec deux `MockCalculator` divergents.

### 2.11 M-07 — `SIGINT` par signal en TUI → exit 1

**Fichiers.** `internal/tui/commands.go:38-42`, bubbletea v1.3.10 `tea.go:273-310,405-406`.

**Constat.** Quand stdin n'est pas un TTY, bubbletea reçoit `SIGINT` par `signal.Notify`, émet `InterruptMsg` et `p.Run` retourne `ErrInterrupted`. `tui.Run` traite toute erreur de `runProgram` comme fatale : « TUI error: program was interrupted » et `ExitErrorGeneric` (1). Le contrat APP-04 (130 sur SIGINT) n'est donc tenu que pour `ctrl+c` frappé dans un terminal. `SIGTERM` devient `QuitMsg` et suit le chemin normal, mais entre en course avec `ContextCancelledMsg` (cf. H-01).

**Correctif.** Dans `tui.Run` : `if errors.Is(err, tea.ErrInterrupted) { return apperrors.ExitErrorCanceled }` avant le message d'erreur générique.

**Vérification.** Test via le seam `runProgram` retournant `tea.ErrInterrupted` → 130, sans « TUI error » sur `errOut`.

### 2.12 M-08 — `TransformCache` borné en entrées, pas en octets

**Fichiers.** `internal/bigfft/fft_cache.go:18-41,358-420`, `internal/fibonacci/options.go:108-140` (`configureFFTCache` : 64 à 128 entrées), `internal/fibonacci/fft.go:62-64` et `matrix_ops.go` (chemin `smartMultiply` → `bigfft.MulTo` → cache).

**Constat (déduit par lecture, non mesuré).** Le cache n'est consulté que par `bigfft.Mul/MulTo/Sqr/SqrTo`, donc par le calculateur `matrix` (défaut `--algo all` inclus) et la calibration. Chaque entrée copie `K × (n+1)` mots, soit ≈ 2 × la taille de l'opérande. À F(100 M), une entrée de fin de calcul pèse ≈ 17 MB ; la dernière itération de l'exponentiation matricielle en insère une douzaine, l'avant-dernière moitié moins, etc. La rétention plausible en fin d'exécution est de plusieurs centaines de MB, bornée par `MaxEntries` (128) × taille d'entrée, soit un plafond théorique de plusieurs GB pour des `n` plus grands. Rien ne libère le cache entre deux calculs du même processus (TUI avec `r`, calibration). `EstimateMemoryUsage` compte 2 × F(n) pour ce poste.

Cette observation recoupe l'« anomalie 3 » d'`audit Gemini.md`, appliquée au seul cache qui existe réellement.

**Correctif.** Ajouter `MaxBytes` à `TransformCacheConfig` (défaut dérivé de `n` dans `configureFFTCache`, par exemple 4 × taille de F(n)) et faire évincer sur les deux bornes dans `putByKey` ; appeler `Clear()` en fin de `CalculateWithObservers` quand le cache a été activé pour ce calcul, ou au moins dans le TUI sur `r`. Mesurer avant : sonde A.1 adaptée au calculateur `matrix`.

**Vérification.** Sonde matrix à 10 M et 100 M avant/après ; `Stats().Size` et somme des `len(backing)` bornées ; test unitaire d'éviction par octets.

### 2.13 L-01 — `ReportStepProgress` : paramètres morts

**Fichiers.** `internal/progress/progress.go:55-72,123-164,184-211`, appelants `doubling_framework.go:153-157,230`, `matrix_framework.go:62-66,97`.

**Constat.** Depuis A-10, la progression est calculée en forme close par `stepProgress(i, numBits)`. `totalWork` ne sert qu'à un garde `> 0` toujours vrai, `workDone`/`powers` ne servent qu'à calculer une valeur de retour que les deux appelants réassignent sans jamais la lire. `CalcTotalWork`, `PrecomputePowers4` et la table `powersOf4` n'existent que pour alimenter ces paramètres.

**Correctif.** Signature `ReportStepProgress(reporter, lastReported *float64, i, numBits int)` ; supprimer les deux helpers, la table et l'`init`. Environ 80 lignes de production et leurs tests.

### 2.14 L-02 — Code de production sans appelant

Vérifié par grep sur les fichiers non-test (les références restantes sont la définition et, parfois, un doc-comment). Les oracles bigfft documentés (ADR-0009 R3) sont exclus de cette liste.

| Symbole | Fichier | Action |
|---|---|---|
| `displayResultWithConfig` | `internal/cli/display.go:365-386` | supprimer |
| `displayMemoryStats` | `internal/cli/presenter.go:107-117` | supprimer |
| `formatProgressBarWithETA` | `internal/format/eta.go:85-89` | supprimer |
| `CalculationArena.AllocBigInt`, `UsedWords` | `internal/fibonacci/memory/arena.go:65-80,125-127` | supprimer |
| `MetricsBuffer.writtenCount` | `internal/fibonacci/threshold/metrics_buffer.go:43-45` | supprimer |
| `ProgressAggregator.calculatorCount` | `internal/orchestration/progress.go:123-125` | supprimer |
| `GCController.setLogger` | `internal/fibonacci/memory/gc_control.go:86-88` | supprimer |
| alias `"ps"` | `internal/cli/completion/completion.go:25` | supprimer (`config.Validate` le refuse, `config.go:128`) |
| `threshold.{getThresholds,getFFTThreshold,getParallelThreshold,getStats,reset}` | `manager.go:181-194,304-328` | dé-exportés en 2026-07, toujours test-only ; à traiter avec M-04 |

Le linter `unused` ne les signale pas : les tests les référencent.

### 2.15 L-03 — Mise à jour finale perdue par envoi non bloquant

**Fichiers.** `internal/progress/observers.go:53-58`, `internal/cli/display.go:98-106`, `internal/orchestration/orchestrator.go:23`.

**Constat.** `ChannelObserver.Update` abandonne la mise à jour si le canal (5 × nombre de calculateurs) est plein. Si les deux envois finaux de 1.0 (`ReportStepProgress` à `i == 0`, puis `reporter(1.0)` dans `CalculateWithObservers`) sont perdus, `DisplayProgress` conclut sur « ETA: N/A (interrupted) » pour un calcul réussi. Probabilité faible (le consommateur draine sans attendre le tick), non observée.

**Correctif.** Faire l'envoi final bloquant (avec `select` sur `ctx.Done()`), ou faire dériver l'affichage final de l'issue du calcul plutôt que de la dernière valeur reçue.

### 2.16 L-04 — `HasVersionFlag`

`internal/app/version.go:32-39` scanne tous les arguments, y compris après `--` et les valeurs de flags. `fibcalc -o --version` affiche la version au lieu d'écrire dans un fichier nommé `--version` ; `--version=false` n'est pas reconnu. Correctif : s'arrêter à `--`. Faible valeur ; à grouper avec une autre modification d'`app`.

### 2.17 L-05 — `--help` colorisé malgré `--machine`/`-q`

`internal/config/usage.go:13-19` : le thème n'est initialisé qu'après le parsing. `fibcalc --machine --help` émet des séquences ANSI. Correctif : dans la fonction d'usage, `if isFlagSetAny(fs, "machine", "quiet", "q") { t = ui.NoColorTheme }`.

### 2.18 L-06 — `alloc` jamais consommé dans la récursion FFT

`internal/bigfft/fft_recursion.go:100,153-158,195-198` : le paramètre `alloc tempAllocator` n'est que transmis récursivement ; la branche parallèle utilise toujours `defaultPoolAllocator`. `fourierWithBump` (`fft_core.go:18-24`) croit passer le bump allocator à la récursion ; il ne sert qu'aux deux `tmp` initiaux. Soit une optimisation manquée (les temporaires des branches séquentielles pourraient venir du bump), soit un paramètre mort. Correctif minimal : supprimer le paramètre et le `//nolint:gocognit` gagne quelques branches. Correctif ambitieux : consommer `alloc` dans la branche séquentielle et mesurer.

### 2.19 L-07 — Progression du mode `--calibrate`

`internal/calibration/calibration.go:206-241` : un seul `DisplayProgress` et un seul agrégateur pour toutes les passes ; la progression retombe à 0 à chaque passe, l'ETA lissé n'a pas de sens, et la ligne finale « Progress: 100 % » ne s'affiche qu'une fois. Cosmétique.

### 2.20 L-08 — Heuristiques de `decideCacheTuning`

`internal/fibonacci/cache_strategy_bigfft.go:79-99` : `int(float64(MaxEntries) * 1.2)` stagne pour `MaxEntries ≤ 4` ; `MinBitLen *= 1.1` sans plafond finit par désactiver le cache sur un long processus puisque `Stats()` est cumulatif et n'est jamais remis à zéro. Sans objet tant que M-04 n'a pas de flag.

### 2.21 L-09 — Worktree résiduel

`git worktree list` montre `.claude/worktrees/lucid-nightingale-3e810f` (HEAD détaché `408a0c9`), copie complète de l'arbre, exclue de git par `.git/info/exclude`. À retirer (`git worktree remove`) quand il n'est plus utile. Non touché par cet audit.

### 2.22 L-10 — Trois `FormatBytes`

`format.FormatBytes`, `errors.formatBytesLocal`, `memory.formatBytesInternal`. Le couplage est accepté par l'architecture en couches (ADR-0008 R1, `errors` et `memory` sont des feuilles). Aucune action.

### 2.23 L-11 — Conversion décimale répétée

**Fichiers.** `internal/cli/display.go:146,165,317,321`.

**Constat.** `writeResult` appelle `result.String()` deux fois (en-tête « Digits », puis corps). `displayDetailedAnalysis` et `displayCalculatedValue` l'appellent chacune une fois. Avec `-d -c -o fichier`, la conversion base 10 de F(100 M) (≈ 21 M de chiffres, plusieurs secondes) est faite quatre fois. `-d` seul paie une conversion complète pour afficher un nombre de chiffres.

Cette observation recoupe l'« anomalie 5 » d'`audit Gemini.md`, dont la moitié « E/S non tamponnées » est fausse ici (`WriteResultToFile` passe par `bufio`).

**Correctif.** Convertir une fois par `DisplayResult` quand `details || showValue`, et une fois dans `writeResult` ; passer la chaîne aux helpers. Pour `-d` sans `-c`, le nombre de chiffres peut venir de la même conversion unique (pas de formule approchée : la valeur affichée doit rester exacte).

**Vérification.** Test qui compte les appels via un `fmt.Stringer` de substitution est impossible sur `*big.Int` ; mesurer avec `-d -c -o` à F(10 M) avant/après (`time`).

## 3. Vérifié sans anomalie

Points examinés qui n'ont pas produit de constat, listés pour délimiter la couverture :

- Invariants d'aliasing arène/état (`ReleaseStateWithResult`, `finalizeStateReleaseTo`, ordre `checkLimit` → `clearStateAliases` → sink) : conformes au contrat P1-04.
- Sémaphores `globalSem` (GOMAXPROCS) et `concurrencySemaphore` (NumCPU) : aucun détenteur n'attend un second jeton ; pas de cycle. Acquisitions non bloquantes dans la récursion, le pointwise et la reconstruction ; `wg.Wait()` inconditionnel avant re-panic (FFT-02).
- Index de pools O(1) (`bits.Len`) : vérifiés à la main aux bornes 64/65/256/257/1024/1025 et 32/33/128/129 ; la référence linéaire est testée.
- `GCController` : refcount global, restauration du `GOMEMLIMIT` d'origine, `WithGC` panic-safe.
- `fermat.Mul/Sqr` : capacité `8n` du scratch suffisante pour `2n+2` ; normalisation identique aux deux chemins.
- `TransformCache.getByKey/putByKey` : snapshot sous `RLock`, pas de salvage des buffers évincés (E1-R4), rejet des formes malformées.
- `ExecuteCalculations` : écritures `results[idx]` disjointes, `Wait` avant `close(progressChan)`.
- `FastDoublingMod` et `ExecuteDoublingLoop` : bornes n = 0, 1, 2 correctes par trace manuelle ; golden couvre 0..200 000.
- `SaveProfile` : écriture atomique via temp + rename avec retry Windows.
- Générateurs de complétion : échappement par shell, garde `registre ⊆ script` (SEC-01 documenté pour `compgen -W`).
- `.gitattributes`, `go.mod` (tidy), `Dockerfile`/devcontainer cohérents avec `go 1.26.0`.

## 4. Plan d'exécution

### 4.1 Règles communes

- Branche `audit/2026-09`, un commit par constat (préfixes existants : `fix(tui)`, `fix(calibration)`, `perf(bigfft)`, `refactor(progress)`, `docs`, `build`).
- Gate à chaque commit : `go build ./...`, `go vet ./...`, `go test -count=1 ./...`, golden intact. Sur hôte CGO : `-race`. Après GATE-01 : lint vert.
- Gate perf pour les tâches marquées **perf** : `benchstat` A/B, `-benchmem -benchtime=1s -count=10`, `BenchmarkFibonacci/(FastDoubling|FFTBased)` 1 M et 10 M, ordre inversé pour confirmer (protocole ADR-0009 R4). Régression > 5 % geomean = rejet.
- Aucun push sans accord explicite.
- Effort : S < 2 h, M ≈ ½ jour, L ≈ 1-2 jours.

### 4.2 Phase 0 — Décisions du mainteneur (bloquantes pour les phases 3 et 4)

| Décision | Options | Recommandation | Débloque |
|---|---|---|---|
| D1 — Priorité flag vs profil (M-03) | A : le flag explicite gagne ; B : le profil gagne + avertissement | A | T-08 |
| D2 — Sort du DTM (M-04) | A : flag `--dynamic-thresholds` + mesure ; B : library-only + doc | A | T-09, L-08, ligne 9 de L-02 |
| D3 — Outillage lint (GATE-01) | A : migration golangci-lint v2 ; B : `GOTOOLCHAIN` épinglé | A | T-01, lint dur |
| D4 — CI minimale | Réintroduire un job Linux build/vet/test/lint/`-race` ; ou rester local | Réintroduire : c'est le seul endroit où `-race` et le lint tourneraient de façon fiable | aucune tâche bloquée |

### 4.3 Phase 1 — Outillage (prérequis)

**T-01 (GATE-01, M).** Selon D3 : migrer `.golangci.yml` vers v2, réinstaller le linter compatible go1.27, re-baseliner les exclusions, rendre le lint dur dans `check.sh` et `check.ps1` (échec sur erreur d'exécution, pas seulement sur findings). Livrable : lint vert sur `main`, scripts sortent non-zéro sur lint en erreur. Traiter les findings révélés (probablement `gosec G304` sur `cli/display.go:289`, à annoter ou corriger) dans le même commit ou un commit `chore(lint)` séparé.

### 4.4 Phase 2 — Correctifs de comportement (indépendants, parallélisables)

| Tâche | Constat | Effort | Fichiers | Tests à ajouter |
|---|---|---|---|---|
| T-02 | H-01 | S | `tui/handlers.go` | `TestModel_ContextCancelledAfterDone_KeepsExitCode` (deadline et cancel) |
| T-03 | M-07 | S | `tui/commands.go` | `TestRun_ErrInterrupted_Exit130` via seam `runProgram` |
| T-04 | H-02 | M | `config/config.go`, `calibration/calibration.go:343`, `cli/calculate.go`, `README.md`, `.env.example`, `completion/registry.go` | `TestValidate_MinusOneDisables`, `TestNewCachedProfileSequentialThresholdApplied`, aller-retour persist/load |
| T-05 | M-06 | S | `app/calculate.go`, `orchestration/orchestrator.go` | `TestRunQuietMismatchWritesStderr` |
| T-06 | M-02 | S | `config/config.go`, `app/calculate.go` | table `config_test.go` ; e2e `--last-digits 5 --memory-limit 4GB` → 4 |
| T-07 | L-04, L-05 | S | `app/version.go`, `config/usage.go` | cas `--`, `--machine --help` sans ANSI |

Ordre conseillé : T-02, T-03 (même fichier de tests), puis T-04 (le plus long), T-05, T-06, T-07.

### 4.5 Phase 3 — Décisions appliquées (après phase 0)

**T-08 (M-03, M, dépend de D1).** Champs `*Explicit` dans `AppConfig`, renseignés par `ParseConfig` ; `LoadCachedCalibration` respecte les explicites ; inverser `TestNewCachedProfileOverridesExplicitFlags` ; simplifier `thresholds.go`, README, `.env.example`. Si D1 = B : avertissement stderr uniquement.

**T-09 (M-04, M si A / S si B, dépend de D2).**
- A : flag `--dynamic-thresholds`, env `FIBCALC_DYNAMIC_THRESHOLDS`, câblage dans `executeCalculations` et `startCalculationCmd`, registre de complétion, `FlagNames` sync test ; puis mesure `benchstat` 1 M / 10 M / 100 M et note dans ADR-0001. Corriger L-08 dans la foulée (plancher de croissance `max(x+1, x*1.2)`, plafond `MinBitLen`, remise à zéro des stats par calcul).
- B : docs `ARCH.md`, `PERFORMANCE.md`, `CALIBRATION.md`, status note ADR-0001 ; supprimer `wireThresholdTuning`, `ThresholdTuningProfile` (garder `MicroBenchTimeout` en constante dans `calibration`), et les accesseurs test-only de `threshold/manager.go`.

### 4.6 Phase 4 — Mémoire et calibration

**T-10 (H-03 + M-08, L, perf).** Dans cet ordre :
1. Rejouer la sonde A.1 pour `fast`, `fft`, `matrix` et `all` à 1 M, 10 M, 50 M, 100 M ; consigner `Sys` delta dans `docs/audits/mem-baseline.txt`.
2. Implémenter `MaxBytes` dans `TransformCache` et le `Clear()` de fin de calcul (M-08) ; re-mesurer `matrix`.
3. Re-modéliser `EstimateMemoryUsage` sur les composants réels ; ajuster jusqu'à satisfaire l'enveloppe `mesuré ≤ estimation ≤ 2 × mesuré` aux 16 points.
4. `TestEstimateMemoryUsage_Envelope` avec les points consignés ; mettre à jour `docs/PERFORMANCE.md` et `.env.example`.
Gate perf sur `matrix` (l'éviction par octets ajoute du travail dans `putByKey`).

**T-11 (M-01, M).** Les cinq points de 2.5 ; sonde A.2 sur dix exécutions comme critère ; test synthétique de `findFFTCrossover`. Après T-11, l'utilisateur doit supprimer ou re-calibrer les profils existants : ajouter une note CHANGELOG et, si le format change, incrémenter `CurrentProfileVersion` (ce qui invalide les anciens profils automatiquement).

**T-12 (L-07, S).** Un agrégateur par passe, ou une ligne de résumé par passe. Optionnel.

### 4.7 Phase 5 — Performance et nettoyage

| Tâche | Constat | Effort | Gate |
|---|---|---|---|
| T-13 | M-05 (`clear(slice[:size])` × 4) | S, **perf** | benchstat ; attendu neutre ou mieux |
| T-14 | L-11 (une conversion par résultat) | S | `time` sur `-d -c -o` à F(10 M) avant/après |
| T-15 | L-01 (signature `ReportStepProgress`) | S | tests `progress` réécrits ; golden |
| T-16 | L-02 (suppressions du tableau 2.14) | S | build + tests ; les tests des symboles supprimés partent avec eux |
| T-17 | L-06 (paramètre `alloc`) | S ou M si l'on tente de consommer le bump dans la branche séquentielle (**perf** dans ce cas) | benchstat si M |
| T-18 | L-03 (envoi final bloquant) | S | test canal saturé + succès → pas de « interrupted » |

### 4.8 Phase 6 — Clôture

**T-19 (S).** CHANGELOG `[Unreleased]` : une ligne par tâche livrée avec l'ID du constat. ADR-0010 « Audit 2026-09 » consignant D1-D4 et les rejets éventuels avec leur preuve (même forme qu'ADR-0008/0009), pour que le prochain audit ne re-suggère pas ce qui a été tranché ici. Régénérer `docs/audits/bench-baseline.txt` et le profil PGO si T-13 ou T-17 changent le chemin chaud. Retirer le worktree résiduel (L-09) si le mainteneur le confirme.

### 4.9 Dépendances et enchaînement

```
Phase 0 (D1-D4)
  ├── T-01 (lint) ─────────────────────────── requis avant de considérer le lint vert sur toute tâche
  ├── Phase 2 : T-02 T-03 T-04 T-05 T-06 T-07  (indépendantes, après T-01 de préférence)
  ├── Phase 3 : T-08 (D1)  T-09 (D2)
  ├── Phase 4 : T-10 (après T-09 si D2 = A, car le DTM touche le cache) ; T-11 ; T-12
  ├── Phase 5 : T-13 T-14 T-15 T-16 T-17 T-18  (indépendantes ; T-16 après T-09)
  └── Phase 6 : T-19
```

Effort total estimé : 6 à 8 jours-personne, dont T-10 (mémoire) et T-01 (lint) sont les deux gros postes.

## 5. Confrontation avec `audit Gemini.md`

Le document décrit un code étranger à ce dépôt. Tableau anomalie par anomalie :

| Gemini | Énoncé | Réalité dans le dépôt | Verdict |
|---|---|---|---|
| 1 (S1) | Aliasing d'un cache LRU de `*big.Int` retournés par référence | Aucun cache de résultats n'existe. Le seul cache est `bigfft.TransformCache` : les entrées sont copiées à l'insertion (`fft_cache.go:402-408`), `Get` retourne une vue en lecture seule documentée, pas de salvage à l'éviction (E1-R4). Le risque d'aliasing réel du projet, résultat vs arène, est déjà couvert par la copie profonde de `ReleaseStateWithResult` (`fastdoubling.go:465-480`). | Sans objet |
| 2 (S2) | Allocations `big.Int` à chaque itération de la boucle | Le « scratchpad » proposé existe : `CalculationState` (5 `big.Int`, `statePool`, `fastdoubling.go:246-288`), arène pré-dimensionnée ×10, slot `cachedState` immunisé au GC, `GCController`. Les benchmarks publient B/op et allocs/op (ADR-0009). | Déjà en place |
| 3 (S2) | Cache LRU borné en nombre, pas en octets → OOM | Sans objet pour un cache de résultats. Pertinent pour `TransformCache`, borné à 64-128 entrées dont la taille croît avec `n`. | Repris sous **M-08** |
| 4 (S3) | Contexte vérifié une fois par bit seulement | Vérifié par itération (`doubling_framework.go:169`), entre chaque multiplication séquentielle (`:112-125`), avant les transformées FFT (`fft.go:112`), dans chaque worker parallèle (`common.go:106`), et par itération dans `FastDoublingMod`. La non-préemptibilité d'un `Mul` de `math/big` est intrinsèque et déjà notée. | Déjà en place |
| 5 (S3) | `String()` quadratique + `os.Stdout` non tamponné | `math/big` convertit en sous-quadratique ; `WriteResultToFile` passe par `bufio` (`display.go:308`) ; les grandes valeurs partent en un seul `Fprintf`. En revanche la conversion est bien recalculée jusqu'à quatre fois. | Moitié fausse ; moitié reprise sous **L-11** |
| 6 (S4) | Pas de `-race`, pas de `ReportAllocs`, pas de cas limites | `check.sh` étape 3 exécute `-race` sur hôte CGO ; les benchmarks sont `-benchmem` ; golden couvre n = 0..200 000 ; `n` est `uint64`, les valeurs négatives n'existent pas. Ce qui manque réellement est une CI qui exécute `-race` quelque part de façon fiable. | Faux ; le point CI est repris sous **D4** |

Ses quatre sprints : les sprints 1 et 3 visent un cache inexistant ; le sprint 2 refait ce que `CalculationState` fait déjà ; du sprint 4, seuls « une conversion, pas quatre » (L-11) et « une CI » (D4) survivent. Son `--meta-only` correspond au comportement par défaut actuel (`-d` sans `-c`).

Conclusion : n'exécuter aucune tâche d'`audit Gemini.md` telle quelle. Les deux idées valables y sont intégrées sous M-08 et L-11 ; le reste contredirait des décisions prises sur preuve.

## Annexe A — Sondes reproductibles

Fichiers `_test.go` temporaires ; les placer dans le paquet indiqué, exécuter `go test -count=1 -run AuditProbe -v ./internal/<paquet>`, puis les retirer. Copies dans le scratchpad de la session.

### A.1 Estimation mémoire (paquet `fibonacci`)

```go
func TestAuditProbeMemoryEstimate(t *testing.T) {
	for _, n := range []uint64{10_000_000, 100_000_000} {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		calc := MustNewCalculator(&FastDoublingCalculator{}) // remplacer par MatrixExponentiationCalculator pour M-08
		if _, err := calc.Calculate(context.Background(), nil, 0, n, Options{}); err != nil {
			t.Fatal(err)
		}
		runtime.ReadMemStats(&after)
		est := memory.EstimateMemoryUsage(n)
		t.Logf("n=%d estimate=%d MB | Sys delta=%d MB | HeapSys=%d MB | TotalAlloc delta=%d MB",
			n, est.TotalBytes>>20, (after.Sys-before.Sys)>>20, after.HeapSys>>20, (after.TotalAlloc-before.TotalAlloc)>>20)
	}
}
```

### A.2 Stabilité du micro-benchmark (paquet `calibration`)

```go
func TestAuditProbeQuickCalibrateNoise(t *testing.T) {
	for i := 0; i < 10; i++ {
		r, err := QuickCalibrate(context.Background())
		t.Logf("run %d: FFT=%d bits parallel=%d conf=%.2f dur=%s err=%v",
			i, r.FFTThreshold, r.ParallelThreshold, r.Confidence, r.Duration, err)
	}
}
```

### A.3 Code de sortie TUI après complétion (paquet `tui`)

```go
func TestAuditProbeExitCodeAfterDone(t *testing.T) {
	cfg := config.AppConfig{N: 100, Timeout: time.Minute}
	m := NewModel(context.Background(), nil, cfg, "v")
	defer m.cancel()
	mm, _ := m.Update(CalculationCompleteMsg{ExitCode: 0, Generation: 0})
	m = mm.(Model)
	mm, cmd := m.Update(ContextCancelledMsg{Err: context.DeadlineExceeded, Generation: 0})
	m = mm.(Model)
	_, isQuit := cmd().(tea.QuitMsg)
	t.Logf("after deadline post-done: exitCode=%d quit=%v", m.exitCode, isQuit) // attendu après T-02 : 0, false
}
```

## Annexe B — Commandes de vérification utilisées

```
go build ./... && go vet ./...
go test -short -count=1 ./...
gofmt -l . ; go mod tidy -diff ; go mod verify
golangci-lint run ./...        # échec typecheck sur go1.27 (GATE-01)
staticcheck ./...              # idem
git worktree list              # L-09
```
