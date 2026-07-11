# auditFable5 — Audit exhaustif FibGo (2026-07-11)

> **Statut : FINAL** — 8 agents thématiques + passe réfutative adversariale (workflow, 13 agents : 12 majeurs + 1 piste, **13/13 confirmés, 0 réfuté**). Gate `scripts/check.ps1` vert sur l'arbre audité.

- **Auditeur** : Claude Fable 5 (orchestration multi-agents : 8 agents thématiques parallèles + passe réfutative sur les findings majeurs).
- **Exécution prévue** : plan compagnon [`auditPlanFable5.md`](auditPlanFable5.md), exécuté par Opus 4.8 en mode ultramode (ultracode).
- **État du dépôt audité** : `main` @ `6da3f3b` (« Restore beforme planning exécution », 2026-07-10), arbre propre. 24 packages, ~17,9k LOC prod / ~29,8k LOC test (mesure directe, `make stats` indisponible sous Git-Bash Windows).

---

## 1. Contexte et périmètre

FibGo est un calculateur Fibonacci haute performance en Go 1.26 (Clean Architecture `cmd → app → orchestration → fibonacci/bigfft → config/errors`), **stabilisé** et déjà audité deux fois (2026-06, 2026-07 — décisions dans ADR-0008/0009). Trois événements récents cadrent cet audit :

1. **Restauration pré-réécriture (2026-07-10, `6da3f3b`)** — un plan de réécriture des suites de tests (vagues ponytail 1–4) a été exécuté puis **volontairement annulé** par le mainteneur : l'arbre est revenu à l'état d'avant le plan (PLAN.md et ADR-0010 supprimés). Conséquences auditables : (a) deux correctifs de **races réelles** trouvés pendant les vagues ont potentiellement été annulés avec le reste ; (b) docs et artefacts (PGO, baseline bench) peuvent être désynchronisés de l'arbre restauré.
2. **18 findings ponytail non traités** — le re-run final `/ponytail-audit` (5 agents, grep-vérifié, journalisé dans PLAN.md §14 avant sa purge — récupéré depuis `git show 03a3168:PLAN.md`) a laissé 18 constats ouverts, dont deux majeurs : l'API opt-in **FFTContext** (~572 LOC, zéro appelant prod) et le **pipeline oracle poolé** (~230 LOC, en tension avec ADR-0009 R3 qui le conserve).
3. **Décisions fermées** — une liste substantielle de candidats a déjà été tranchée (§3) ; cet audit ne les rouvre pas sans élément nouveau.

## 2. Thématiques retenues (meilleures pratiques d'ingénierie logicielle)

| # | Thème | Préfixe | Justification |
|---|---|---|---|
| 1 | Correction & concurrence | CONC | Hot path pooling/arène/GC + invariants CLAUDE.md ; risque de régression silencieuse post-restauration |
| 2 | Sécurité & frontières de confiance | SEC | Injection shell (complétion), profils calibration, validation d'entrée, deps |
| 3 | Code mort / YAGNI / sur-ingénierie | DEAD | 18 findings ponytail ouverts, jamais traités |
| 4 | Qualité des tests & couverture | TEST | Plancher 80 %, gardiens CLAUDE.md, fragilité timing |
| 5 | Architecture & hygiène d'API | ARCH | Étanchéité des couches, dérive CLAUDE.md ↔ code |
| 6 | Documentation & cohérence | DOCS | Références pendantes post-purge PLAN.md/ADR-0010, drift docs ↔ code |
| 7 | Build, outillage & hygiène perf | BUILD | Parité check.sh/ps1, gitattributes CRLF, fraîcheur PGO/baseline |
| 8 | Gestion d'erreurs & UX d'échec | ERR | Discipline %w, codes de sortie, chemins d'annulation, restauration terminal |

Thèmes exclus délibérément : CI/CD (décision A5-02 : aucune CI distante), observabilité production (outil CLI local), i18n (hors périmètre prototype académique).

## 3. Exclusions — décisions déjà tranchées (ne pas rouvrir sans élément nouveau)

- **ADR-0008 R1–R7** : `errors.ColorProvider` ; `executeTasks`/`executeMixedTasks` ; seam `CacheStrategy` ; observers `progress` ; exports bigfft `GetFFTParams`/`PolyFromInt`/`ValueSize` ; emplacement `TestFactory`/`MockCalculator` ; knobs threshold package-level (A2-04).
- **ADR-0009 R1–R5 + addendum** : `scan.go` et machinerie `fftState` déjà supprimés ; oracles bigfft conservés (R3, clause de révision explicite) ; **multiplicateur d'arène ×10 adopté** après balayage complet — ne pas re-balayer sans nouvelle microarchitecture ; migration FFTContext = WONT-FIX release courante (ADR-0004 §B1).
- **A5-02** : validation locale uniquement (`scripts/check.{sh,ps1}`), pas de GitHub Actions.
- **`fibonacci_golden.json`** : immuable sans ADR.
- **Directive #6** : pas de nouveaux fichiers `progress*`.
- **Restauration `6da3f3b` volontaire** : pas de re-proposition en bloc des réécritures de tests des vagues 1–3.

## 4. Méthodologie

1. **Fan-out** : 8 agents thématiques parallèles, lecture seule, chaque finding exigé grep/lecture-vérifié avec citation `fichier:ligne`, sections « zones saines » et « incertitudes » obligatoires.
2. **Passe réfutative** (exécutée) : les 12 findings majeurs + 1 piste soumis à 13 agents réfutateurs indépendants (workflow parallèle) avec mandat explicite de les casser — **13/13 confirmés**, nuances intégrées en place (marqueur « Nuance du réfutateur »).
3. **Synthèse** : dédoublonnage inter-thèmes (ARCH-03=CONC-03, DOCS-02=ARCH-01), calibrage des sévérités, renvoi vers le plan d'exécution.

Contraintes d'environnement : Windows sans gcc (pas de `-race` direct ; WSL ciblé uniquement), `make` absent de Git-Bash (cibles lues, pas lancées).

## 5. Findings

> _Section en cours de remplissage — collecte des 8 agents en cours._

### 5.1 CONC — Correction & concurrence

#### [CONC-01] Use-after-release dans `TestStateBump_FollowsArenaDrop` — état lu après `statePool.Put`
- **Sévérité** : majeur · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/fibonacci/state_cache_test.go:185-191` ; chemin prod impliqué : `internal/fibonacci/fastdoubling.go:538-548`
- **Preuve** : le test force le drop anti-bloat (`s.arenaCapWords = maxArenaPoolWords + 1`), appelle `ReleaseState(s)`, puis lit `s.arena` (l.186) et `s.bump` (l.189). L'état n'étant pas overLimit, `finalizeStateReleaseTo` atteint `statePool.Put(s)` (fastdoubling.go:548) : l'état est publié dans le pool partagé et tout test parallèle passant par `AcquireStateForN` peut le récupérer et le muter (`s.Reset()`, `prepareStateForN`) pendant les lectures. Race réelle sous `-race` avec `t.Parallel()` + flake fonctionnel possible. **C'est l'un des deux correctifs de la vague 2 annulée, défait par la restauration `6da3f3b`.**
- **Nuance du réfutateur** : le drop anti-bloat annule arena/bump mais n'empêche **pas** le `put(s)` (vérifié fastdoubling.go:524-548) ; écrivains concurrents réels identifiés (`state_pool_test.go:44-64` : 100 goroutines ; `state_cache_test.go:197` : 8×25). Portée test-only, fenêtre de course étroite (localité per-P de `sync.Pool`) — flake improbable mais contrat `sync.Pool` violé. Corroboré par l'historique (`103d614`/`86828d1`, correctifs annulés par la restauration).
- **Recommandation** : fix minimal test-only — remplacer `ReleaseState(s)` par `finalizeStateReleaseTo(s, func(*CalculationState) {})` (sink no-op : la logique testée est exercée, la propriété exclusive de `s` est conservée). Aucun code prod touché.
- **Vérification post-correctif** : `wsl go test -race ./internal/fibonacci/ -run 'TestStateBump' -count=10` vert ; gate `scripts/check.sh`. Diff test-only : benchstat sans objet (à noter au commit).

#### [CONC-02] Use-after-release dans `TestReleaseState_OverLimit_AliasesCleared` (sous-cas nominal)
- **Sévérité** : majeur · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/fibonacci/state_pool_arena_test.go:219-233`
- **Preuve** : le sous-cas `ReleaseState_nominal_alsoCleared` appelle `ReleaseState(s)` sur un état non-overLimit (publié via `statePool.Put`, fastdoubling.go:548) puis `assertCleared(t, s, captured)` déréférence les cinq slots `big.Int` (`BitLen()`, l.159-175) — data race avec un `Get` + `Reset()` concurrent, assertion non déterministe. Les deux sous-cas overLimit (l.177-217) sont sains (état droppé, jamais publié). **Second correctif de vague 2 défait par la restauration.**
- **Nuance du réfutateur** : écrivain concurrent réel = `AcquireStateForN` → `Get()` + `Reset()` (`fastdoubling.go:320-321,267-269` : `SetInt64` sur FK/FK1) depuis les tests `t.Parallel()` du package ; sous-cas overLimit confirmés sains. Même portée test-only que CONC-01.
- **Recommandation** : même correctif que CONC-01 — `finalizeStateReleaseTo(s, func(*CalculationState) {})` dans le sous-cas nominal uniquement.
- **Vérification post-correctif** : `wsl go test -race ./internal/fibonacci/ -run 'TestReleaseState' -count=10` vert ; gate `scripts/check.sh`.

#### [CONC-03] Commentaire périmé « ×15 » dans `memory/arena.go`
- **Sévérité** : info · **Effort** : S
- **Fichiers** : `internal/fibonacci/memory/arena.go:12-13`
- **Preuve** : le commentaire d'en-tête dit « the ×15 multiplication below » alors que le code multiplie par 10 (l.32) depuis l'addendum ADR-0009 R4 ; le commentaire jumeau de `fastdoubling.go:288` dit correctement ×10.
- **Recommandation** : corriger « ×15 » → « ×10 ».
- **Vérification post-correctif** : relecture ; aucun test impacté.

**Zones saines vérifiées (CONC)** : chemin unique `finalizeStateReleaseTo` (aucun sink hors chemin) ; `clearStateAliases` inconditionnel avant tout sink ; overLimit jamais publié, bump relâché sur drop et overLimit ; multiplicateur ×10 synchronisé entre `acquireSizingForN` et `arenaTotalWords` + miroirs de test (formules bit-identiques) ; `WithGC` panic-safe + refcount GOGC package-level, `Begin`/`End` bien `Deprecated` ; `MetricsBuffer` : 5 accès tous sous mutex ; seuils bigfft en atomic privés ; `panicCh` bufferisé + re-panic après `wg.Wait()` sur les 4 chemins parallèles, classifier sentinel intact ; APP-05 (génération + timeout frais, 5 gardes de génération + 2 dans le routeur) ; spinner CONC-01 (ordre stop→write→start) ; `putByKey` backing frais ; routage `releaseWordSlice` sur `cap`. Balayage frais : aucune goroutine sans cycle de vie, aucun contexte fuité, unique `time.Sleep` prod légitime (backoff rename Windows, `calibration/profile.go:192`), canaux bufferisés/non bloquants, aucun mélange atomic/non-atomic, annulation vérifiée dans la boucle de doublement.

**Incertitudes (CONC)** : passe `-race` ciblée WSL verte sur CONC-01/02 (fenêtre de course en ns — la preuve est statique : violation du contrat `sync.Pool`) ; chemins `-tags gmp` non inspectés ; audit de fond calibration délégué à l'agent SEC.

### 5.2 SEC — Sécurité

#### [SEC-01] Bash : l'échappement de complétion ne couvre pas la ré-expansion `compgen -W`
- **Sévérité** : info (défense en profondeur ; aucun exploit actuel) · **Effort** : S (commentaire de contrat)
- **Fichiers** : `internal/cli/completion/escape.go:25-43` (`escapeBashDoubleQuoted`), `internal/cli/completion/bash.go:124,130`
- **Preuve** : le script généré émet `algorithms="…"` puis `COMPREPLY=( $(compgen -W "${algorithms}" -- "${cur}") )`. L'échappement neutralise `\ $ " \`` au niveau de l'affectation, mais `compgen -W` **ré-expanse** chaque mot (substitution de commande incluse) : un nom contenant `$(...)` stocké littéralement serait exécuté à la seconde expansion. Le gardien `TestGenerateBash_AdversarialAlgo` ne teste que l'équilibrage des guillemets, pas cette ré-expansion. zsh/fish/PowerShell non concernés (éléments traités littéralement). **Aucun vecteur aujourd'hui** : les noms d'algorithmes sont 100 % statiques (`a.Factory.List()`, registre compilé, `app.go:139`) — mais le contrat E5 d'`escape.go` présente l'échappement comme barrière suffisante pour une future source dynamique, ce qui est faux pour bash.
- **Recommandation** : aucune action de code tant que la source reste statique ; ajouter un commentaire de contrat dans `bash.go` (« ne jamais brancher de source dynamique sans repenser la couche compgen ; à défaut, filtrer à `[A-Za-z0-9_-]` à la source ») et amender le contrat E5.
- **Vérification post-correctif** : assertion dans le test bash que la doc de contrat est en place ; si source dynamique un jour : test d'exécution réelle (nom `evil$(touch témoin)`, complétion déclenchée, témoin absent).

#### [SEC-02] Vulnérabilité non appelée dans `golang.org/x/sys` (GO-2026-5024)
- **Sévérité** : info · **Effort** : S
- **Fichiers** : `go.mod:16` (`golang.org/x/sys v0.43.0`)
- **Preuve** : `govulncheck -show verbose ./...` → GO-2026-5024 (overflow `NewNTUnicodeString`, `x/sys/windows`) classée « imported but not called » ; verdict global « affected by 0 vulnerabilities ».
- **Recommandation** : bump `golang.org/x/sys` au prochain cycle de deps, sans urgence.
- **Vérification post-correctif** : `govulncheck ./...` → 0 vuln y compris au niveau import.

**Zones saines vérifiées (SEC)** : re-validation SEC-01 des seuils forgés présente et testée (`calibration.go:342-344`, gardien PASS) ; second chemin `LoadCachedCalibration` gardé par `config.Validate` (rejet des seuils négatifs) ; parsing N en `uint64` + clamps saturants (`arena.go:21-33`, `budget.go`) ; écritures fichiers en 0600 + rename atomique ; `ConfigExcerpt` jamais peuplé en prod ; gardiens de complétion + batterie adversariale d'échappement PASS ; govulncheck 0 vuln appelée (33 modules + stdlib go1.26.5).

**Incertitudes (SEC)** : profil JSON lu sans borne de taille (`profile.go:95`) — non retenu (fichier dans le home, modèle de menace local) ; pas de passe `-race` (Windows) ; SEC-01 non matérialisé dans un bash réel (sémantique `compgen -W` documentée, lecture de code).

### 5.3 DEAD — Code mort / YAGNI

#### [DEAD-01] FFTContext — API opt-in entière sans aucun appelant de production ⚠️ DÉCISION MAINTENEUR
- **Sévérité** : majeur (volume) · **Effort** : M · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/bigfft/context.go:1-453` (17 fonctions), `internal/bigfft/fft_recursion_ctx.go:1-119`
- **Preuve** : grep hors `_test.go` et hors les 2 fichiers propres → **1 seul hit, un commentaire** (`fft.go:52`). 0 appelant prod. Tests dédiés ~401 LOC + ~130 mêlés (dont le gardien nommé CLAUDE.md `TestFourierRecursiveCtxAsyncPanicPropagates`). **LOC : prod 572 / tests ~530.**
- **Options (ne pas trancher unilatéralement)** :
  - **A — Couper** : ADR-0004 §B1 classe la migration FFTContext won't-fix release courante — l'abstraction a été codée pour une migration qui n'aura pas lieu (YAGNI). Collatéraux à traiter dans le même lot : gardien CLAUDE.md à retirer, commentaires `SetFFTThreshold` (fft.go:49-52) et invariant « panics worker » à réécrire, ADR-0004 §B1 à amender.
  - **B — Conserver** : la migration FFTContext reste la solution tracée pour SA6002 (ADR-0007) et la dé-globalisation (ADR-0003) ; couper = re-payer écriture + re-validation si la migration est un jour engagée ; zéro coût runtime (opt-in), code testé.
- **Nuance du réfutateur** : LOC confirmés (453+119, 17 fonctions, zéro appelant externe y compris tags gmp) ; tests dédiés = 401 LOC exactement (le ~530 inclut des portions de fichiers mêlés). `defaultContext` porte un `//nolint:unused` citant ADR-0004 §B1 — **rétention délibérée et documentée** : la suppression relève d'une décision ADR, pas d'un nettoyage de code mort ordinaire.
- **Vérification post-correctif (si A)** : build + vet, suite bigfft `-race` (WSL), golden, benchstat vs baseline (gate Directive #1 ; attendu : aucun impact, code jamais appelé).

#### [DEAD-02] Pipeline FFT « oracle de test » dupliqué — tension explicite avec ADR-0009 R3 ⚠️ DÉCISION MAINTENEUR
- **Sévérité** : majeur (volume) · **Effort** : M · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/bigfft/fft_poly.go` (`Poly.Mul`, `Transform`, `NTransform`, `InvNTransform`, `PolValues.Clone`), `fft_cache.go` (`TransformCache.Get/Put/Clear`, `TransformCached`, `MulCached`, `SqrCached`), `arith.go:1-36` (`AddVV`/`SubVV`/`AddMulVVW`)
- **Preuve** : les 12 sites portent le commentaire « Test oracle … (audit OVR-10) » ; 0 appelant prod pour chacun (la prod route par `MulCachedWithBump`/`SqrCachedWithBump` → `getByKey`/`putByKey` ; `fermat.go` utilise les `addVV`/… privés). **LOC prod ~240**, tests imbriqués dans les suites de cross-validation (non isolables).
- **Options (ne pas trancher unilatéralement)** :
  - **Pour couper** : double API à maintenir sur le package le plus sensible ; chaque évolution du pipeline bump doit être répliquée côté pool pour garder l'oracle signifiant.
  - **Pour conserver (position ADR-0009 R3, toujours en vigueur)** : chemins de référence non-bump contre lesquels fuzz/précision valident les chemins bump ; la clause de révision R3 (« si la stratégie de test migre vers une autre validation ») n'est **pas** déclenchée — aucun substitut n'existe. Couper aujourd'hui = perdre la cross-validation sans remplacement.
- **Nuance du réfutateur** : zéro appelant prod confirmé symbole par symbole avec désambiguïsation des homonymes (`fibonacci/fft.go:203` = `PolValues.Mul`, chemin prod distinct ; `Transform`→`TransformCached`→`MulCached`/`SqrCached` = chaîne transitivement morte) ; `fermat.go` utilise les primitives privées via `go:linkname` (`arith_decl.go`). Rétention délibérée OVR-10/R3 — pas du code mort oublié.
- **Vérification (si coupe un jour décidée)** : ADR amendant R3 **d'abord** ; puis build + fuzz + golden + benchstat.

#### [DEAD-03] `getVersionInfo`/`VersionData` — code mort revenu avec la restauration
- **Sévérité** : mineur · **Effort** : S — `internal/app/version.go:56-83`, `version_test.go:71-93` : 0 appelant prod, seul `TestGetVersionInfo` teste la fonction morte. Coupé par `760064a` puis réintroduit par la restauration. **Re-couper** (coupe déjà validée le 2026-07-10). Vérif : build + `go test ./internal/app/`.

#### [DEAD-04] `internal/parallel` — micro-package à consommateur unique, réinvention d'errgroup
- **Sévérité** : mineur · **Effort** : S–M — 64+38 LOC, **1 importateur** (`fibonacci/common.go`), errgroup déjà dep directe utilisée dans orchestration. Ligne #1 de la cut-list du re-run. ⚠️ `fibonacci/` = perf-sensible : la vague annulée avait mesuré **+15–19 % allocs/op avec errgroup naïf** (closures) et corrigé via une struct `parallel3Result` — reprendre cette approche, benchstat obligatoire. Vérif : build + `-race` WSL + golden + benchstat.

#### [DEAD-05] `internal/metrics/system` — package de 28 LOC pour un appelant unique
- **Sévérité** : mineur · **Effort** : S — 1 importateur (`tui/commands.go:14`). Inline déjà exécuté (`58eb09c`) puis annulé par la restauration. Re-inliner `Sample()` dans `tui`. Vérif : build + `go test ./internal/tui/`.

#### [DEAD-06] `fibonacci/fibonaccitest` — package de prod pour un seul fichier de test
- **Sévérité** : mineur · **Effort** : S — 69 LOC prod, consommateur unique `orchestration/contract_test.go` ; le doc.go prétend servir app/tui — faux (ils utilisent `MockCalculator`/`TestFactory`, ADR-0008 R6). Déplacer `CoreStub` dans son unique test appelant, supprimer le package. Vérif : build + `go test ./internal/orchestration/`.

#### [DEAD-07] Machinerie Observer : `AsProgressCallback` mort ; pattern complet sous Directive #6
- **Sévérité** : mineur · **Effort** : M — `progress/observer.go:131-136` (`AsProgressCallback` : 0 prod, 1 test) et `RecoveredObserverCount` (accesseur test-only). La prod n'enregistre qu'un observateur puis `Freeze`. A minima couper `AsProgressCallback` ; le pattern entier reste soumis à **consultation mainteneur** (D-02/Directive #6). `LoggingObserver`/`NoOpObserver` exclus (A-19). Vérif : `go test ./internal/progress/ ./internal/fibonacci/`.

#### [DEAD-08] `threshold.DynamicThresholdManager` — 6 méthodes exportées test-only
- **Sévérité** : mineur · **Effort** : S — `SetLogger`/`GetThresholds`/`GetFFTThreshold`/`GetParallelThreshold`/`GetStats`/`Reset` + type `ThresholdStats` : 0 appelant prod chacun (surface prod réelle = 4 symboles). ⚠️ `Reset` est cité par l'invariant A2-04 de CLAUDE.md — sync doc requise. Dé-exporter ou couper. Vérif : `wsl go test -race ./internal/fibonacci/threshold/`.

#### [DEAD-09] `bigfft` — knobs test-only (`SetFFTThreshold`, trio `FFTParallelismConfig`)
- **Sévérité** : mineur · **Effort** : S — `SetFFTThreshold` (0 prod, déjà auto-documenté « Test-only in practice » mais renvoie vers FFTContext → lié à DEAD-01) ; trio `FFTParallelismConfig` (0 prod ; le hot path lit les atomics privés). Documenter test-only ou dé-exporter ; coordonner avec la décision DEAD-01. Vérif : build + suite bigfft `-race` ; benchstat non requis (symboles jamais appelés).

#### [DEAD-10] `BumpAllocator.AllocUnsafe` — export test-only
- **Sévérité** : mineur · **Effort** : S — `bigfft/bump.go:127-135` : 0 prod, 3 tests. Couper ou dé-exporter. Vérif : build + `go test ./internal/bigfft/`.

#### [DEAD-11] `fibonacci/memory` — primitives d'arène et stats GC sans appelant prod
- **Sévérité** : mineur · **Effort** : S — `AllocBigInt` (0 prod, 15 tests), `UsedWords` (0/9), `GCStats`+`stats()` (0/2). ⚠️ `d89eb5c` (audit 2026-07) avait **explicitement conservé** `AllocBigInt`/`UsedWords` (« arena primitives, unit-tested ») — aucun élément nouveau : décision mainteneur pour ces deux-là ; `GCStats`/`stats()` (aucune conservation documentée) coupables sans dette. Vérif : `-race` WSL memory + gardien `TestArenaTotalWords_ClampNoUB`.

#### [DEAD-12] `calibration` — options à valeur constante et knobs figés
- **Sévérité** : mineur · **Effort** : S — `RunCalibration` passe toujours `{SaveProfile: true, LoadProfile: false}` → `tryUseCachedCalibrationProfile` inatteignable en prod ; `MicroBenchIterations`/`MicroBenchTestSizes` exportés consommés in-package. Accepter ou aplatir l'indirection. **Piste routée en vérification** : le mode `-calibrate` ignore `--calibration-profile` (`app.go:149` ne transmet pas `cfg.CalibrationProfile` alors que le flux AutoCalibrate le transmet) — possible bug de câblage, vérification en cours (→ §5.9).

#### [DEAD-13] `ui.SetCurrentTheme` — setter test-only
- **Sévérité** : info · **Effort** : S — `ui/themes.go:155-162` : 0 prod, 7 tests (in-package). Dé-exporter ou couper.

#### [DEAD-14] `app.AppOption`/`WithFactory` — functional options pour une seule option
- **Sévérité** : info · **Effort** : S — seam nécessaire aux tests (injection de `TestFactory`, ADR-0008 R6), déjà **gardé délibérément** par `d89eb5c`. Sans élément nouveau : laisser ; signalé pour complétude.

#### [DEAD-15] Exports cosmétiques — visibilité excédentaire sans appelant externe (lot groupé)
- **Sévérité** : info · **Effort** : S — 10 symboles (détail : `FFTSafetyMarginWords`, `GCAutoThreshold`, `ProgressBufferMultiplier`, `FormatQuietResult`, `ExecutionState`, `FibCalculator`, `matrix.SetIdentity`/`SetBaseQ`, `DefaultProfileMaxAge`/`ProfileMaxAgeEnv`, trio analyzer threshold, `ErrNoUsefulResults`). Lot de dé-exports opportuniste, à ne porter que par un autre chantier. Nota : `matrix.Set` (0 appelant, relayé par TEST) appartient à cette famille.

#### [DEAD-16] `internal/testutil/doc.go` — 25 lignes de doc pour 21 lignes de code
- **Sévérité** : info · **Effort** : S — réduire doc.go à ~3 lignes.

**Zones saines vérifiées (DEAD)** : les 20 champs d'`AppConfig` tous lus en prod ; toutes les interfaces ≥ 2 implémentations réelles ou seam justifié (comptes frais conformes aux exclusions ADR-0008) ; chaînes prod complètes vérifiées (`EnsurePoolsWarmed`, `QuickCalibrate`, `ComputeLastDigits`, `ValidateMemoryBudget`, `ProgressAggregator`, `MatrixFramework`, GMP `init()`, `SIMDKind`, `Get/SetTuning`) ; `WordSlicePoolMissCount` = canal de validation documenté (ne pas toucher) ; wrappers `fermat.*Safe` conservés par ADR-0002 §5 (non re-proposés) ; `format`, `metrics` racine, `progress.go`, `cmd/*`, `errors`, `completion` : balayage exports complet sans autre trouvaille ; duplication `GoldenData` volontaire (8 lignes).

**Incertitudes (DEAD)** : comptes par grep avec désambiguïsation manuelle des homonymes (`Poly.Mul` vs `PolValues.Mul` — ce dernier est prod) — risque résiduel faible, confirmer par `go vet`/staticcheck avant coupe ; LOC tests des oracles non isolables ; liste originale des 18 constats reconstruite depuis `git show 03a3168:PLAN.md` §8 + commits de vague restaurés (`760064a`, `58eb09c`, `d89eb5c`).

### 5.4 TEST — Qualité des tests

**Couverture** : `go test ./... -cover -count=1` — 24/24 packages verts, **95,2 % global** (`go tool cover -func` mergé), plancher 80 % largement tenu. Points bas relatifs : `cmd/generate-golden` 82,5 % (`main()` 0 %, corps dans `run()` testé), `calibration.GenerateParallelThresholds` 55,6 %, chemins TTY (`app.runTUI` 33 %, `tui.Run` 0 %) structurellement non testables sans terminal. Nota : le profil per-package ne crédite pas la couverture cross-package (`config.FlagNames` 0 % localement mais exercé par `completion`).

#### [TEST-01] L'invariant E1-R4 (pas de recyclage de backing à l'éviction) n'est réfutable par aucun test — et les commentaires affirment l'inverse
- **Sévérité** : majeur · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/bigfft/fft_cache_test.go:284-298,301-321,329-331` ; commentaire prod périmé `internal/bigfft/fft_cache.go:47`
- **Preuve** : la prod est correcte (`putByKey` alloue toujours frais, `fft_cache.go:397`, contrat l.340-352). Mais le test dit le contraire : « the backing buffer must NOT be reallocated because the recycled evicted buffer fits » (l.284-286), message d'échec `"(R1.5 backing recycling regressed?)"` (l.298), et le « direct check » (l.303) ne vérifie que l'homogénéité des capacités — trivialement vraie avec des allocations fraîches. `fft_cache.go:47` : `// recycled on eviction` contredit le contrat de la même struct 300 lignes plus bas. Conséquence : réintroduire le recyclage (le use-after-free E1-R4, Audit-PRD) ne ferait rougir **aucun** test.
- **Nuance du réfutateur** : point (d) vérifié par grep exhaustif des `*_test.go` d'`internal/` — aucun test ni fuzz ne retient un `PolValues` à travers une éviction ; un recyclage fits-checked passerait même le plafond de 16 allocs. L'en-tête du test (l.226-231) documente correctement E1-R4 — seuls les commentaires du corps sont périmés.
- **Recommandation** : (1) corriger les commentaires périmés (test + `fft_cache.go:47`) ; (2) ajouter l'assertion d'épinglage : capturer `&entry.backing[0]` de l'entrée LRU-tail avant un `Put` à capacité, asserter que le backing de la nouvelle entrée est un pointeur différent (~10 lignes, pas de nouveau fichier).
- **Vérification post-correctif** : mutation locale réintroduisant le salvage → test rouge ; code réel → vert. `go test ./internal/bigfft/ -run TestTransformCacheEviction -count=1`.

#### [TEST-02] Sous-tests calibration à assertion conditionnelle (chemin timeout vert silencieux)
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `internal/calibration/calibration_test.go:305-317,337-344`
- **Preuve** : `updated, ok := AutoCalibrateWithProfile(...)` puis `if ok { … }` — si `ok == false` le sous-test se termine sans aucune assertion. Assertion de sortie lâche par ailleurs (tout mot « calibration » suffit).
- **Recommandation** : sur le chemin `!ok`, asserter la config inchangée (pattern déjà utilisé par le sous-test « No fast calculator », l.367-377).
- **Vérification post-correctif** : forcer `ok=false` (contexte annulé) → le sous-test exerce encore une assertion.

#### [TEST-03] `TestModel_HandleReset_FreshTimeoutBudget` : fenêtre ~10 ms sous charge (fragilité connue, confirmée)
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `internal/tui/model_test.go:404-440`
- **Preuve** : `Timeout: 20ms` (l.410), assertion `remaining < cfg.Timeout/2` (l.429) — un stall d'ordonnanceur > 10 ms fait échouer à tort (faux positifs déjà observés sous charge, journal 2026-07-10). Seule vraie sensibilité au timing du dépôt ; aucun `time.Sleep` dans les tests.
- **Recommandation** : asserter sans dépendre du temps écoulé : capturer `before := time.Now()` avant `handleReset` et vérifier `deadline.Sub(before) >= cfg.Timeout` (prouve le budget frais, insensible au stall).
- **Vérification post-correctif** : `go test ./internal/tui/ -run TestModel_HandleReset -count=20` sous charge CPU → 0 échec.

#### [TEST-04] Oracle golden non épinglé côté test : une troncation silencieuse passerait au vert
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `internal/fibonacci/fibonacci_golden_test.go:28-31`
- **Preuve** : le test itère `for _, tc := range cases` sans asserter taille ni présence des entrées ADR-0004 §B5 — un JSON réduit à une entrée donnerait une suite verte. Corpus actuel vérifié : 26 entrées 0→200 000, F(50k/100k/200k) présents.
- **Recommandation** : après décodage, asserter `len(cases) >= 26` + présence de N=50 000/100 000/200 000. Ne touche pas l'oracle (immuable).
- **Vérification post-correctif** : copie tronquée du JSON dans un répertoire temporaire → rouge ; fichier réel → vert.

#### [TEST-05] Adoption `t.Parallel()` : 90,6 % top-level, écart majoritairement justifié
- **Sévérité** : info · **Effort** : S
- **Preuve** : 839 tests top-level, 79 sans `t.Parallel()` dont 43 avec commentaire « Not parallel: … » ; les 36 restants sont presque tous structurels (`t.Setenv` incompatible, état global bigfft/Strassen, écriture cwd). Aucun oubli clair prouvé.
- **Recommandation** : au fil de l'eau, ajouter le commentaire « Not parallel: … » manquant sur les cas à état global.

#### [TEST-06] `TestTickCmd_DeliversTickMsg` paie 500 ms de mur réel
- **Sévérité** : info · **Effort** : S
- **Fichiers** : `internal/tui/model_test.go:641-657`
- **Preuve** : le test exécute `cmd()` qui attend réellement le tick bubbletea de 500 ms. Non flaky (sens déterministe) ; seul délai réel fixe du dépôt.
- **Recommandation** : acceptable en l'état ; ne vaut un changement que si la durée de suite devient un enjeu.

**Zones saines vérifiées (TEST)** : les 22 gardiens CLAUDE.md existent et assertent réellement leur contrat (détail vérifié un par un : slots + pointeurs pour AliasesCleared, miroirs ×10 cohérents post-addendum R4, compteur de miss uniquement pour ResizedReturnsToBucket — aucune assertion d'identité `sync.Pool` réintroduite, 4 arrows exacts dans arch_test.go, séquence spinner assertée, registry↔config bidirectionnel + APP-11) ; oracle golden présent/consommé, **aucun flag `-update` dans le dépôt**, `cmd/generate-golden` à noyau itératif indépendant (P2-04) ; aucun `time.Sleep` de test, deadlines courtes toutes dans le sens déterministe, état global restauré via `t.Cleanup`.

**Incertitudes (TEST)** : `matrix.Set` (`matrix_types.go:38`) à 0 % et **sans appelant trouvé** — transmis au thème DEAD ; `TestConcurrentAccess` non exécuté sous `-race` (mandat) ; pourcentages per-package = planchers (couverture cross-package non créditée).

### 5.5 ARCH — Architecture & API

#### [ARCH-01] `dependency-graph.mermaid` désynchronisé du graphe réel — dont une arête **interdite** par arch_test
- **Sévérité** : majeur (doc « source de vérité » contredisant un invariant gardé) · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `docs/architecture/dependency-graph.mermaid:58` (arête périmée) ; l.45, 54, 68 (contexte des arêtes manquantes)
- **Preuve** : l.58 dessine `orch --> format`, or `go list` montre qu'`orchestration` n'importe **pas** `format` — c'est précisément l'un des 4 arrows interdits par `internal/arch_test.go:69-73` (APP-10). Arêtes réelles absentes : `config → fibonacci/memory` (`validator.go:7`, `threshold_tuning.go:6`, `config.go:11`), `config → ui` (`usage.go:8`), `config → errors`, `fibonacci → errors`.
- **Recommandation** : corriger le mermaid — retirer `orch --> format`, ajouter les 4 arêtes réelles.
- **Vérification post-correctif** : diff arête par arête contre `go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./...` ; `TestArchitectureLayering` vert.

#### [ARCH-02] Arrows latérales `config → fibonacci/memory` et `config → ui` tolérées mais non gardées
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `internal/config/validator.go:7`, `threshold_tuning.go:6`, `config.go:11`, `usage.go:8` ; `internal/arch_test.go:38-39` (commentaire seulement)
- **Preuve** : la hiérarchie déclarée place config en feuille ; en réalité config importe `fibonacci/memory` (assumé, documenté `config/doc.go:33`) et `ui`. Aucun garde n'empêche config d'importer demain `fibonacci` racine ou `bigfft` (fermerait un cycle).
- **Recommandation** : ajouter une règle `forbiddenImport` config ↛ {fibonacci racine, bigfft} — fige la tolérance actuelle à `fibonacci/memory` seul. Optionnel : une phrase dans CLAUDE.md.
- **Vérification post-correctif** : `go test ./internal/ -run TestArchitectureLayering`.

#### [ARCH-03] Commentaire « ×15 » périmé dans `arena.go` — _même constat que CONC-03 (dédoublonné en synthèse)_

#### [ARCH-04] Aliases `orchestration.Default*Threshold` sans aucun consommateur
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `internal/orchestration/interfaces.go:30-32` ; `doc.go:24-26`
- **Preuve** : `grep -rn 'orchestration\.Default' cmd internal test` → 0 résultat. CLAUDE.md (§tui) les présente pourtant comme chemin d'accès prévu. Non couverts par ADR-0008/0009 (vérifié).
- **Recommandation** : supprimer les 3 aliases (+ mention doc.go) **ou** retirer `Default*Threshold` de la phrase CLAUDE.md — au choix du mainteneur, cohérence dans les deux cas. `Calculator`/`Options` restent (réellement consommés, `tui/commands.go:20,49`).
- **Vérification post-correctif** : `go build ./...` + `TestArchitectureLayering` verts ; grep vide.

#### [ARCH-05] 4 fonctions prod au-dessus du plafond cyclomatique 15
- **Sévérité** : mineur (advisory, mais convention chiffrée violée) · **Effort** : M
- **Fichiers** : `internal/tui/model.go:94` (Update : 16), `internal/bigfft/fft_recursion_ctx.go:27` (17), `internal/bigfft/fft_recursion.go:95` (17), `internal/config/config.go:99` (Validate : 16)
- **Preuve** : `golangci-lint run ./...` (v1.64.8). Total : 155 issues, quasi toutes en `_test.go` ; errcheck prod : 0 ; gocognit/funlen prod : 0.
- **Recommandation** : ne pas toucher bigfft pour du lint (coût benchstat > bénéfice) ; `AppConfig.Validate` et `Model.Update` refactorables à faible risque si un passage y est déjà prévu ; sinon assumer en exclusion `.golangci.yml` documentée.
- **Vérification post-correctif** : `golangci-lint run` sans gocyclo prod ; benchstat ±5 % si bigfft touché.

#### [ARCH-06] Surface exportée large à usage strictement intra-package
- **Sévérité** : info · **Effort** : L (en bloc — non recommandé)
- **Preuve** : balayage systématique — `SetFFTThreshold`, `EstimateMemoryNeeds`, `PreWarmPools`, `Get/SetFFTParallelismConfig`, `QuickCalibrate`, `Generate*Thresholds`, `AcquireStateForN`, etc. : aucun usage qualifié hors package en prod (tous vivants in-package). `calibration` n'est consommé par `app` que via 5 symboles ; `completion` n'expose utilement que `Generate`.
- **Recommandation** : aucune action en bloc (`internal/` borne déjà la visibilité) ; dé-exports opportunistes lors d'un passage déjà justifié.

#### [ARCH-07] Fichiers prod > 500 lignes (constat, pas d'action)
- **Sévérité** : info
- **Preuve** : `fft_cache.go` (661), `fft_poly.go` (651), `fastdoubling.go` (549), `calibration.go` (503) — les trois premiers sont des modules sensibles CLAUDE.md.
- **Recommandation** : aucune (le découpage casserait la localité des invariants documentés).

**Table de dérive CLAUDE.md ↔ code (ARCH)** — 15 affirmations vérifiées une à une : **13 exactes** (maxCachedArenaWords=4M ; ×10 synchronisé + miroirs ; chemin unique de teardown ; A2-04 aux lignes annoncées ; threshold sans import config + câblage Once dans app.New ; atomics threshold/bigfft/logger ; récepteur fermat z ×13 méthodes ; errors sans format ; routeur Update pur ; spinner stop→write→start ; WithGC panic-safe, tag Deprecated porté par Begin seul ; putByKey backing frais ; 4 arrows exacts dans arch_test), **1 décalée** (aliases `Default*Threshold` cités mais non consommés → ARCH-04), **1 fausse en l'état** (« source de vérité docs/architecture/ » → ARCH-01).

**Zones saines vérifiées (ARCH)** : graphe d'imports complet 24 packages — aucun import remontant hors exceptions config documentées ; feuilles propres ; `cmd/fibcalc` → `internal/app` uniquement ; tui atteint fibonacci uniquement via orchestration ; doc.go présent dans les 20 packages Go ; interfaces toutes étroites (comptage réel : Multiplier 3, Calculator 4, ProgressReporter 1…) ; `metrics` vs `metrics/system` et `format` vs `ui` : responsabilités distinctes ; packages de support test consommés uniquement par des `_test.go`.

**Incertitudes (ARCH)** : usages de types par inférence non détectés par grep (chaque candidat notable contre-vérifié) ; version golangci-lint non épinglée formellement ; symboles `-tags gmp` hors périmètre (non compilables ici).

### 5.6 DOCS — Documentation

#### [DOCS-01] CHANGELOG `[Unreleased]` vide alors que l'arbre a divergé de v4.0.0
- **Sévérité** : majeur · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `CHANGELOG.md:8-10,92-96`
- **Preuve** : `## [Unreleased]` → « _Rien pour l'instant._ », or `git diff v4.0.0 HEAD --stat` = 13 fichiers (+176/−87) avec des changements de comportement jamais changelogués : `e7caa83` (arène ×15→×10), `9804463` (gate GMP 3b dans check.sh), `ce53ef2` (épinglage LF `*.sh`), régénérations PGO/baseline. Pire : la dernière parole du CHANGELOG sur le multiplicateur est la section « Rejected » de 4.0.0 (« ×15 = charge utile intentionnelle », l.96) — contredite par le code actuel (`arena.go:32` : `* 10`). Trou antérieur à la restauration.
- **Recommandation** : ajouter sous `[Unreleased]` : adoption ×10 (renvoi addendum ADR-0009 R4), gate GMP 3b, épinglage LF, régénération PGO/baseline.
- **Vérification post-correctif** : chaque fichier de `git diff v4.0.0 HEAD --stat` rattachable à une ligne du CHANGELOG ; grep `×10` ≥ 1 hit.

#### [DOCS-02] Arête fantôme `orch --> format` dans le mermaid — _même constat qu'ARCH-01 (corroboration indépendante ; dédoublonné en synthèse)_
- Complément de preuve propre à DOCS : l'arête a été **réintroduite par la restauration** `6da3f3b` (diff `+ orch --> format` ; la réécriture l'avait retirée en vague 3, `4b85f7e`, mais elle était déjà périmée avant le plan). Arête manquante supplémentaire : `config --> errors`.

#### [DOCS-03] Exemples documentés sur l'API supprimée `GlobalFactory()`/`RegisterCalculator`
- **Sévérité** : majeur · **Effort** : M · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `docs/algorithms/COMPARISON.md:166,179,191` ; `FAST_DOUBLING.md:299` ; `FFT.md:203` ; `MATRIX.md:373` ; `GMP.md:60,68,115` ; `CONTRIBUTING.md:189` ; `docs/ARCH.md:781`
- **Preuve** : `factory := fibonacci.GlobalFactory()` (7 sites) et `fibonacci.RegisterCalculator(...)` — zéro occurrence dans le code ; `calculator_gmp.go:34` : « OVR-06 removed the exported GlobalFactory/RegisterCalculator ». Remplacement existant : `NewDefaultFactory()` (`calculator.go:62`). Ces exemples ne compilent plus.
- **Nuance du réfutateur** : 8 des 11 sites sont des blocs de code non compilables ; GMP.md:60/115 et ARCH.md:781 sont des mentions en prose. Le remplacement `NewDefaultFactory` est défini à `registry.go:57` (le `calculator.go:62` cité est un doc-comment). Vérifié aussi sous build tag gmp : seule une var privée `globalFactory` subsiste.
- **Recommandation** : réécrire les exemples avec `fibonacci.NewDefaultFactory()` ; pour CONTRIBUTING/ARCH/GMP, décrire le point d'extension réel (fabrique par défaut + build tag, plus de registre global).
- **Vérification post-correctif** : `grep -rn "GlobalFactory()\|RegisterCalculator" docs/ CONTRIBUTING.md` → zéro hit hors CHANGELOG/ADR historiques.

#### [DOCS-04] BIGFFT.md documente `scan.go`, `fftState` et le split arith comme actuels
- **Sévérité** : majeur · **Effort** : M · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `docs/algorithms/BIGFFT.md:236,457,559,586-619`
- **Preuve** : sections entières sur `FromDecimalString`/`scan.go` (l.586-595), `type fftState struct` (l.457), `fourierWithState` (l.236), table des fichiers listant `scan.go`, `arith_amd64.go`, `arith_generic.go` — tous supprimés (OVR-01, FFT-05, FFT-06 ; ADR-0009 R1/R2). Zéro hit code. La passe doc de l'audit 2026-07 a raté ce fichier.
- **Recommandation** : supprimer les sections mortes ; remplacer les lignes arith par `arith.go` ; recompter la table (LOC codés en dur tous périmés — envisager de retirer la colonne Lines).
- **Vérification post-correctif** : chaque fichier de la table existe dans `internal/bigfft/` ; grep `fftState|FromDecimalString|arith_amd64` dans BIGFFT.md → zéro.

#### [DOCS-05] BUILD.md et PORTABILITY.md décrivent encore le split `arith_amd64.go`/`arith_generic.go`
- **Sévérité** : mineur · **Effort** : S — `docs/BUILD.md:93-94,129-134` ; `docs/PORTABILITY.md:23-29` ; réalité : un seul `arith.go` sans build-tag (ARCH.md:137 est correct). Correctif : remplacer par `arith.go`. Vérif : grep → zéro hors historiques.

#### [DOCS-06] ARCH.md liste `scan.go`, `TimeoutError`, `ValidationError` — supprimés
- **Sévérité** : mineur · **Effort** : S — `docs/ARCH.md:130,247,617,619` ; `internal/errors/errors.go` ne définit que `ConfigError`, `CalculationError`, `MemoryError` (OVR-07). Correctif : retirer scan.go de l'arbre §3, corriger les deux listes. Vérif : chaque type/fichier nommé existe.

#### [DOCS-07] Gate d'architecture documenté à « trois règles » — il y en a quatre
- **Sévérité** : mineur · **Effort** : S — `docs/TESTING.md:163-170`, `docs/architecture/README.md:53-73`, `docs/ARCH.md:731` : `orchestration → format` (APP-10) manque partout (CLAUDE.md et README racine sont corrects) ; la table ADR s'arrête à 0008 alors que 0009 existe. Correctif : 4 règles + ligne 0009 + « 0001..0009 ». Vérif : nombre documenté = `len(architectureRules)`.

#### [DOCS-08] Pied de CHANGELOG : lien `[Unreleased]` sur v3.0.0, `[4.0.0]` sans lien, commentaire tags faux
- **Sévérité** : mineur · **Effort** : S — `CHANGELOG.md:651-656` : ancre compare `v3.0.0...HEAD` (devrait être v4.0.0), pas de définition `[4.0.0]:`, commentaire « v3.0.0 only release tag » périmé (`git tag -l` : v4.0.0 existe). Vérif : chaque `## [x.y.z]` a sa définition ; ancre sur le dernier tag.

#### [DOCS-09] Référence pendante `docs/INNOVEPLAN.md` dans un commentaire de test
- **Sévérité** : mineur · **Effort** : S — `internal/orchestration/contract_test.go:16` : renvoi vers un fichier supprimé (commit `1755c3d`) + mention « fast CI » (pas de CI, A5-02). Unique référence restante. Vérif : grep INNOVEPLAN → zéro.

#### [DOCS-10] validation-report.md affirme la cohérence des 4 diagrammes — faux tant qu'ARCH-01/DOCS-02 est ouvert
- **Sévérité** : mineur · **Effort** : S — `docs/architecture/validation/validation-report.md:36-41`. Correctif : corriger le mermaid puis re-signer la phrase. Les invariants ligne à ligne du rapport restent vrais.

#### [DOCS-11] Convention de numérotation ADR après la suppression de 0010
- **Sévérité** : info · **Effort** : S — ADR-0010 a existé puis a été supprimé par la restauration ; aucune référence pendante, mais le numéro est « brûlé » dans l'historique git. Recommandation : acter (template ou CLAUDE.md) que le prochain ADR = **0011**. Vérif : note présente.

#### [DOCS-12] Procédure dashboard (BUILD.md) : `config.json` jamais copié vers `docs/dashboard/`
- **Sévérité** : mineur · **Effort** : S — `docs/BUILD.md:388-400` : le bundle bake `VITE_CONFIG_URL="./config.json"` et l'artefact déployé contient `config.json`, mais la procédure `rm -rf` + copies ne le copie pas. Incertitude : `dist/` le contient peut-être déjà (build Vite non exécuté). Correctif : ajouter le `cp` ou documenter. Vérif : dérouler la procédure → `config.json` présent.

**Zones saines vérifiées (DOCS)** : 100 % des liens relatifs de tous les .md résolvent (script exhaustif, zéro lien cassé) ; les mentions `internal/parallel`/`metrics/system`/`fibonaccitest` sont redevenues exactes post-restauration ; aucune référence pendante à PLAN.md/ADR-0010/bench-local-*/coverage-baseline ; les 17 gardiens nommés dans CLAUDE.md existent (grep `func <Nom>(`) ; constantes exactes (FFT 500 000 = `constants.go:28`, 24 entrées linters, plancher 80.0 dans les deux scripts, go 1.26.0, tag v4.0.0 daté cohérent) ; PERFORMANCE.md étiquette explicitement ses chiffres comme historiques ; doc.go conformes ; le reste du mermaid exact (35+ arêtes vérifiées une à une) ; README table des packages complète ; TESTING.md §GMP à jour.

**Incertitudes (DOCS)** : contenu de `dist/` Vite non vérifiable sans build (DOCS-12) ; état des tags sur le remote non vérifié (finding DOCS-08 fondé sur le tag local, certain) ; corroboration indépendante de CONC-01/02 (commits `fix(fibonacci)` `103d614`/`86828d1` annulés par la restauration, qualifiés « bogues préexistants » par le journal) — relayée au thème CONC.

### 5.7 BUILD — Build & outillage

#### [BUILD-01] Baseline benchstat statistiquement faible : `-benchtime=1x` inclut l'outlier de warm-up
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `Makefile:232-233` ; `docs/audits/bench-baseline.txt:5-8`
- **Preuve** : `bench-baseline` = `-count=5 -benchtime=1x`. Dans la baseline commitée, `FastDoubling/1M` : 4 501 888 → 3 069 782 ns/op (écart intra-échantillons ~46 %) ; B/op du 1er échantillon 10 574 536 vs ~1 320 000 ensuite (warm-up pools/arène inclus). Le gate Directive #1 (seuil 5 %) perd en sensibilité : IC larges, « no significant difference » peut masquer une vraie régression.
- **Recommandation** : à la **prochaine régénération justifiée seulement** (CLAUDE.md : jamais sans cause) : `-benchtime=3x` ou `-count=6` en écartant le 1er échantillon. Ne rien régénérer maintenant.
- **Vérification post-correctif** : écart intra-échantillons < 10 % sur ns/op ; benchstat baseline-vs-baseline (2 runs) → deltas ~0, IC serrés.

#### [BUILD-02] `.gitattributes` incomplet — mécanisme de divergence WSL/Windows latent (correctif S prouvé sans mega-diff)
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `.gitattributes:5,11`
- **Preuve** : seuls `*.go`/`*.sh` épinglés ; `core.autocrlf=true` (gitconfig système). Aujourd'hui `git ls-files --eol` → 355 fichiers `i/lf w/lf` (index ET worktree 100 % LF) et `wsl git status` → 0 entrée — mais tout checkout futur (switch de branche, stash pop) smudgera en CRLF les fichiers non épinglés (`*.md`, `*.yml`, `*.json`, `Makefile`, `*.ps1`…), que git WSL verra « modifiés » (les ~45 faux modifiés du 2026-07-10).
- **Recommandation** : ajouter `* text=auto eol=lf` en tête + binaires explicites (`*.pgo binary`, `*.ico binary`). **Sans danger prouvé** : index déjà 100 % LF → `git add --renormalize .` serait un no-op, aucun mega-diff possible. Précaution : `docs/dashboard/index.html` auto-détecté binaire, `text=auto` préserve cette détection.
- **Vérification post-correctif** : `git add --renormalize . && git status --porcelain` → vide ; `wsl git status --porcelain` → vide après checkout frais.

#### [BUILD-03] Plancher de couverture 80 % codé en 3 endroits
- **Sévérité** : mineur · **Effort** : S
- **Fichiers** : `Makefile:184-188`, `scripts/check.sh:23`, `scripts/check.ps1:29`
- **Preuve** : trois implémentations indépendantes, sémantique identique aujourd'hui (`<` strict, 80.0 passe) — dérive possible à la prochaine modification.
- **Recommandation** : option minimale — faire déléguer `coverage-check` (Makefile) à `check.sh`. Ne pas factoriser davantage.
- **Vérification post-correctif** : une seule source par plateforme.

#### [BUILD-04] `install-tools` installe golangci-lint `@latest` pour une config verrouillée v1.64.8
- **Sévérité** : info · **Effort** : S
- **Fichiers** : `Makefile:284` ; `.golangci.yml:4`
- **Preuve** : chemin de module sans `/v2` → résout de facto le dernier v1 (= v1.64.8), cohérent aujourd'hui mais le verrou n'est pas explicite.
- **Recommandation** : épingler `@v1.64.8`.
- **Vérification post-correctif** : `make install-tools` puis `golangci-lint version` → v1.64.8.

#### [BUILD-05] « 24 linters » documenté vs 23 actifs rapportés
- **Sévérité** : info · **Effort** : S
- **Preuve** : 24 entrées `enable:` dans `.golangci.yml` ; `golangci-lint linters` en rapporte 23 (`typecheck` = frontend compilateur, non togglable, non listé). Écart cosmétique.
- **Recommandation** : aucune action code ; préciser « 24 entrées (23 linters + typecheck implicite) » à la prochaine retouche doc.

#### [BUILD-06] Marqueurs « POSIX-only » du Makefile incomplets
- **Sévérité** : info · **Effort** : S
- **Fichiers** : `Makefile:37-42,45,181,195,218`
- **Preuve** : toutes les recettes exigent un shell POSIX (`[ -f ]`, `mkdir -p`), pas seulement les 4 cibles marquées ; `make` absent de Git Bash local, chemin réel = WSL. Non-problème en pratique.
- **Recommandation** : une ligne en tête « Makefile = POSIX/WSL uniquement » à la prochaine retouche.

**Zones saines vérifiées (BUILD)** : **chronologie PGO/baseline COHÉRENTE — suspicion infirmée** (`git diff 3f9e7be HEAD -- internal/fibonacci internal/bigfft` → vide ; le restore `6da3f3b` reprend byte-à-byte l'état du 2026-07-07, PGO 354 854 o inclus ; le changement d'octets du PGO observé au restore était la restauration du profil documenté ; la baseline mesure exactement le code prod de HEAD, gate Directive #1 valide) ; 10 cibles Makefile conformes à CLAUDE.md, `make check` délègue à `check.sh`, `pgo-profile` sans `-tags gmp` ; parité check.sh/check.ps1 (seules divergences = documentées ; `set -euo pipefail` vs `Set-StrictMode` + `$LASTEXITCODE` par étape ; parse InvariantCulture côté ps1) ; `go mod verify` OK, `go mod tidy -diff` sans dérive, 10 deps directes toutes utilisées ; `go vet ./...` propre ; golangci-lint v1.64.8 = verrou config ; exclusion SA6002 toujours justifiée (8 sites vivants) ; smoke build + bench PASS ; arbre propre des deux côtés Windows/WSL aujourd'hui.

**Incertitudes (BUILD)** : cause exacte des ~45 faux modifiés du 2026-07-10 non déterminable a posteriori (BUILD-02 ferme le mécanisme quoi qu'il en soit) ; `golangci-lint run` complet non exécuté (advisory) ; baseline en `goos: linux` — toute comparaison benchstat doit se faire sous WSL (implicite dans le workflow, non écrit).

### 5.8 ERR — Gestion d'erreurs

#### [ERR-01] `WriteResultToFile` ignore les erreurs d'écriture et de Close — perte de données silencieuse
- **Sévérité** : majeur · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/cli/display.go:288-307` ; appelant `internal/app/calculate.go:237-246`
- **Preuve** : `defer file.Close()` non vérifié + 9 `fmt.Fprintf(file, …)` aux retours ignorés + `return nil` inconditionnel. Sur disque plein/quota : échec silencieux, « ✓ Result saved to: … » imprimé, exit 0. Contraste : `cmd/generate-golden/main.go:59-63` corrige exactement ce patron (Close vérifié via named return).
- **Recommandation** : vérifier l'erreur de Close via named return (patron generate-golden) ; optionnellement `bufio.Writer` + `Flush` vérifié. La remontée existe déjà (`saveResultIfNeeded` → exit 1).
- **Vérification post-correctif** : test avec writer qui échoue → erreur non-nil ; errcheck vert.

#### [ERR-02] Erreurs de calcul écrites sur stdout au lieu de stderr — casse le scripting
- **Sévérité** : majeur · **Effort** : M · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/app/calculate.go:51,109-125` ; `internal/orchestration/orchestrator.go:150,161` ; `internal/errors/handler.go:52-62`
- **Preuve** : observé — `fibcalc -n 2000000 -timeout 1ms -algo fast 1>/dev/null 2>err.txt` → exit 2, **stderr vide** (statut d'échec, table et diagnostic sur stdout) ; idem `--memory-limit banana` → exit 4, stderr vide ; même en `-q`. À l'inverse, flags/env, `--last-digits` et la sauvegarde écrivent bien sur stderr. Un script `result=$(fibcalc -q …)` capture le texte d'erreur comme résultat.
- **Recommandation** : router les messages d'échec (`validateMemoryBudget`, `HandleCalculationError` du chemin comparaison) vers `a.ErrWriter` ; la table de comparaison peut rester sur stdout, le « Status: … » final et le diagnostic vont sur stderr.
- **Vérification post-correctif** : les trois invocations → messages d'échec sur stderr, stdout vide (ou table seule), codes inchangés (2/4/130) ; goldens CLI ajustés.

#### [ERR-03] `--last-digits` : timeout et Ctrl-C non honorés une fois le calcul lancé
- **Sévérité** : majeur · **Effort** : M · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11)
- **Fichiers** : `internal/orchestration/lastdigits.go:42-50` ; `internal/fibonacci/modular.go:17` ; borne K : `internal/app/calculate.go:133`
- **Preuve** : `ComputeLastDigits` ne vérifie le contexte qu'une fois **avant** le calcul ; `FastDoublingMod(n, m)` ne prend pas de contexte. Observé : `fibcalc -n 99999999 --last-digits 200000 -timeout 50ms -q` → **exit 0 en 272 ms** (deadline dépassée, calcul complet). Avec `maxLastDigits = 10_000_000` (modules ~33 Mbits), un calcul peut durer des minutes sans que `-timeout` ni SIGINT ne l'interrompent.
- **Recommandation** : passer `ctx` à `FastDoublingMod` et tester `ctx.Err()` en tête de chaque itération de doublement (≤ 64 itérations, coût nul) — overshoot borné à une itération.
- **Nuance du réfutateur** : reproduit indépendamment — même commande : exit 0 en 418 ms (vs 272 ms, variance machine) ; `--last-digits 2000000 -timeout 50ms` → exit 0 en **4,1 s**. L'extrapolation « minutes » à k=10M est plausible (~10×) mais non mesurée.
- **Vérification post-correctif** : test deadline mi-boucle → erreur wrappée `context.DeadlineExceeded`, exit 2 ; la commande observée retourne exit 2.

#### [ERR-04] `tui.Run` avale l'erreur de `p.Run()` — exit 1 muet
- **Sévérité** : mineur · **Effort** : S — `internal/tui/commands.go:31-34` : l'erreur est jetée sans aucun message (ex. échec TTY → exit 1 inexpliqué). Correctif : écrire `err` sur stderr ou remonter à `app.runTUI` (qui possède `ErrWriter`). Vérif : `p.Run` en échec injecté → message stderr + exit 1.

#### [ERR-05] `--auto-calibrate` échoue en silence totale
- **Sévérité** : mineur · **Effort** : S — `internal/calibration/calibration.go:396-416`, `internal/app/app.go:153-160` : `runStrategy` retourne `(BaseConfig, false)` sans écrire ; l'utilisateur qui demande explicitement la calibration n'a aucun signal d'échec. Contraste : le chemin profil-forgé imprime son warning. Correctif : `Warning: auto-calibration failed (%v), using default thresholds`. Vérif : strategy en échec → warning sur `out`.

#### [ERR-06] `--version`/`-V` absent de l'aide
- **Sévérité** : mineur · **Effort** : S — `internal/config/usage.go:27-42` : `--help` (26 flags) ne liste ni `-version`/`-V` ni `-h` (gérés hors FlagSet, reconnu par le commentaire de `FlagNames`, config.go:184). Le reste de l'usage est auto-synchronisé (`fs.VisitAll`). Correctif : ligne statique dans le footer de `setCustomUsage` ; suggestion « did you mean » optionnelle (coût/bénéfice faible). Vérif : `--help` liste `-version` ; `usage_test.go` étendu.

#### [ERR-07] Mode comparaison : `firstError` choisi par tri de durée, pas par cause racine
- **Sévérité** : info · **Effort** : S — `internal/orchestration/orchestrator.go:64-78,122-157` : tous les calculateurs en échec → `firstError` = échec le plus court ; un frère `context.Canceled` classé premier mapperait exit 130 au lieu de 1 (non déterministe entre classes, quasi jamais en pratique). Correctif : première erreur non-`Canceled`, repli sur la première. Vérif : test `[erreur réelle lente, Canceled rapide]` → exit 1, « Failure ».

#### [ERR-08] Commentaires français résiduels dans le code de prod
- **Sévérité** : info · **Effort** : S — `bigfft/fft_recursion.go:94`, `fft_recursion_ctx.go:26`, `cli/display.go:43,310`, `orchestration/interfaces.go:17`, `cmd/generate-golden/main.go:55`. Tous les messages utilisateur sont en anglais (vérifié dynamiquement) ; seuls des commentaires dérogent. Correctif : traduire au prochain passage sur ces fichiers.

**Zones saines vérifiées (ERR)** : 69 `fmt.Errorf` prod tous en `%w` (zéro `%v` cassant `errors.Is/As`) ; aucune sentinelle comparée par string ; errcheck prod : zéro ; cartographie des codes de sortie vérifiée dynamiquement (0/1/2/3/4/130, defaults sûrs, diagnostic riche) ; tous les `_ =` prod justifiés par commentaire ; annulation standard (`signal.NotifyContext` + timeout, `ctx.Err()` à chaque bit de la boucle) ; spinner avec defer filet + affichage final honnête ; TUI : recovery bubbletea restaure le terminal sur panic, altscreen relâché sur q/timeout/SIGINT, générations périmées ignorées ; pools libérés sur early-return d'erreur (FastDoubling, FFT-based, matrix en defer) ; `WithGC` unique chemin prod, restauration GOGC même sur panic ; les 9 recover prod conformes ADR-0002 (le seul absorbant est documenté et compté) ; `SaveProfile` exemplaire ; messages actionnables (valeurs valides listées, suggestion `--last-digits`).

**Incertitudes (ERR)** : chemin d'erreur `p.Run()` non observé dynamiquement (`--tui </dev/null` reste suspendu — ERR-04 prouvé statiquement) ; SIGINT non testé dynamiquement sous Windows (chemin 130 vérifié statiquement + tests du dépôt) ; panic non recoverée en mode comparaison CLI = issue assumée ADR-0002 (errgroup ne propage pas les panics), non comptée. Signalements routés : commentaire ×15 (→ CONC-03) ; `getVersionInfo`/`VersionData` re-morts post-restauration (`app/version.go:60-82`, supprimés par `760064a` puis réintroduits par la restauration — → DEAD).

### 5.9 CAL — Finding issu de la vérification de piste (DEAD-12)

#### [CAL-01] Le mode `-calibrate` ignore `--calibration-profile` — bug de câblage confirmé, portée UX
- **Sévérité** : mineur (ajusté à la baisse par le réfutateur) · **Effort** : S · **Réfutation** : **CONFIRMÉ** (passe adversariale 2026-07-11, preuve statique)
- **Fichiers** : `internal/app/app.go:149` ; `internal/calibration/calibration.go:97-101,149,252` ; aide CLI `internal/config/config.go:165` ; `docs/CALIBRATION.md:24,29,78`
- **Preuve** : `app.go:149` appelle `RunCalibration` dont la signature n'accepte aucun chemin ; options codées en dur `{SaveProfile: true, LoadProfile: false}`, `ProfilePath: ""` → `persistCalibrationProfile` sauvegarde via `SaveProfile("")` au chemin par défaut `~/.fibcalc_calibration.json`. Le flux `--auto-calibrate` transmet bien `cfg.CalibrationProfile` (`calibration.go:305`). L'aide CLI (`config.go:165`) et ARCH.md:560 promettent un chemin générique sans restriction de mode ; **mais** `docs/CALIBRATION.md:24,29,78` documente explicitement la sauvegarde au chemin par défaut pour `-calibrate` — l'incohérence est donc entre le texte d'aide CLI et le comportement, pas entre la doc de référence et le comportement.
- **Recommandation** : deux options — **A (recommandée)** : câbler `cfg.CalibrationProfile` dans le flux `-calibrate` (honorer le flag partout, moindre surprise) et aligner CALIBRATION.md ; **B** : restreindre le texte d'aide du flag (« utilisé par --auto-calibrate »). Contournement actuel trivial (copier le profil).
- **Vérification post-correctif** : test unitaire du chemin de sauvegarde avec profil personnalisé ; l'aide CLI et CALIBRATION.md racontent la même histoire.

## 6. Zones vérifiées saines (consolidation)

Chaque thème porte sa liste détaillée en §5 (« Zones saines vérifiées »). L'image d'ensemble :

- **Le cœur tient** : les 15 invariants CLAUDE.md vérifiés exacts (1 décalage doc, 1 doc fausse — corrigés par ARCH-01/04) ; les 22 tests gardiens présents et réellement assertifs ; chemin unique de teardown, panics worker propagées, atomics en place, GOGC refcount sain.
- **Qualité mesurée** : couverture 95,2 % (24/24 packages verts) ; `go vet` propre ; errcheck prod zéro ; `go mod tidy -diff` sans dérive ; govulncheck 0 vuln appelée ; `scripts/check.ps1` **vert** sur HEAD (constaté 2026-07-11).
- **Artefacts perf cohérents** : la suspicion baseline/PGO périmés est **infirmée avec preuve** (restore byte-identique à l'état 2026-07-07) — le gate Directive #1 est valide.
- **Étanchéité** : graphe d'imports 24 packages conforme à la hiérarchie (2 exceptions config documentées, à garder sous garde — ARCH-02) ; 100 % des liens relatifs des .md résolvent.
- **Discipline d'erreurs** : 69 `fmt.Errorf` prod tous en `%w` ; codes de sortie cartographiés et cohérents (hors ERR-02) ; recover conformes ADR-0002.

## 7. Synthèse et priorisation

**59 findings uniques** (après dédoublonnage ARCH-03=CONC-03, DOCS-02=ARCH-01) : **12 majeurs** (dont 2 « décision mainteneur »), **30 mineurs**, **17 infos**. Les 12 majeurs et la piste CAL ont tous survécu à la passe réfutative adversariale (13/13 confirmés, 0 réfuté).

| Priorité | ID | Quoi | Pourquoi d'abord |
|---|---|---|---|
| P1 | CONC-01, CONC-02 | races use-after-release re-perdues par la restauration | correctifs déjà validés une fois, S chacun, test-only |
| P1 | ERR-01 | « Result saved » menteur sur échec d'écriture | perte de données silencieuse, S |
| P1 | ERR-03 | `--last-digits` ininterruptible | contrat timeout/SIGINT violé, prouvé 2× |
| P2 | ERR-02 | erreurs sur stdout | casse le scripting ; M (goldens à ajuster) |
| P2 | DOCS-01/03/04 + ARCH-01 | CHANGELOG muet, exemples morts, BIGFFT.md fantôme, mermaid interdit | la doc « source de vérité » contredit code et arch_test |
| P2 | TEST-01 | E1-R4 non réfutable + commentaires inversés | protège un invariant Règle d'or |
| P3 | mineurs (30) | vagues 2-6 du plan | voir auditPlanFable5.md |
| Décision | DEAD-01, DEAD-02 (+DEAD-07, DEAD-11, ARCH-04/05, CAL-01 A/B) | ~810 LOC de rétentions délibérées documentées | exigent arbitrage mainteneur / ADR, pas un nettoyage |

Exécution : ordre, gates et gabarits dans [`auditPlanFable5.md`](auditPlanFable5.md) (7 vagues, risque croissant, ultramode Opus 4.8).

## 8. Hypothèses et limites

- Hypothèse de lancement (tâche autonome, mainteneur non joignable) : audit exécuté via fan-out d'agents (8 thématiques) puis passe réfutative en Workflow (ultracode activé en cours de session) — le mandat `ultramode` désigne par ailleurs l'exécution future du plan par Opus 4.8.
- La liste détaillée des 16 petits findings ponytail n'ayant jamais été committée, elle a été **re-dérivée** par balayage frais — la correspondance 1:1 avec la liste originale n'est pas garantie.
- Pas de passe `-race` complète dans cette session (Windows ; seule une passe ciblée WSL sur CONC-01/02 a été exécutée, verte — la preuve de ces findings est statique) ; le plan d'exécution impose le gate WSL `-race` à chaque vague.
- Sévérités : CONC-01/02 maintenus « majeur » malgré la nuance test-only du réfutateur (contrat `sync.Pool` violé + précédent Directive #7 : ces mêmes bugs avaient été corrigés immédiatement en vague 2) ; DEAD-01/02 maintenus « majeur (volume) » avec statut décisionnel explicite — les réfutateurs confirment les faits et rappellent que la coupe exige un accord ADR.
