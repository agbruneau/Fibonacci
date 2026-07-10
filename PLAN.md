# PLAN.md — Réécriture intégrale des modules Go (FibGo)

- **Statut** : Proposé — révision 2, durcie par analyse multi-agents (25 scénarios de défaillance dont 19 silencieux).
- **Date** : 2026-07-10
- **Révise** : la révision 1 du 2026-07-10, rédigée dans un bac à sable distinct. Sept de ses affirmations sur l'environnement se sont révélées fausses sur la machine cible (§0).
- **Portée mesurée** : ~15 948 LOC de production / ~26 052 LOC de test, 24 packages (`go list ./...`), dont `internal` (gardien d'architecture) et `test/e2e` (oracle boîte-noire) que la révision 1 ignorait. `make stats` reste la source canonique (`CLAUDE.md` § Projet) mais `make` est absent de la machine Windows : le décompte ci-dessus vient d'un comptage direct et doit être considéré comme indicatif, non comme un chiffre à recopier.
- **Décisions arrêtées** (par le mainteneur, avant analyse) :
  - **(A) Oracle en deux phases** — phase A : la suite de tests existante est gelée et sert d'oracle, la production est réécrite jusqu'au vert ; phase B : production gelée, tests réécrits, nouvel oracle validé.
  - **(B) Périmètre** — comportement observable figé ; structure libre (fusion, suppression de packages et changement de signatures internes autorisés).
  - **(C) Exécution** — pipeline multi-agents sur les packages feuilles, strictement séquentiel sur `fibonacci/` et `bigfft/`.
- **Finalité** : réécrire intégralement chaque module sous discipline `/ponytail`, sans perdre un seul invariant.
- **Document vivant** : toute déviation se justifie dans le journal (§12), jamais en silence.

---

## 1. L'objection, une fois

La discipline ponytail commence par « ce code doit-il exister ? ». Appliquée au plan lui-même, elle donne une réponse inconfortable qu'il faut inscrire ici plutôt que la taire.

L'audit ponytail du dépôt a été mené (agent dédié, arbre entier, importateurs comptés à la main). **Le gain net mesuré est d'environ 316 lignes de production supprimables, soit 2,0 % des 15 948 LOC**, et **aucune dépendance `go.mod` n'est retirable**. Le cœur de performance (`bigfft/`, `fibonacci/`) est serré : les interfaces qu'on soupçonnerait mono-implémentation (`TempAllocator`, `ResultPresenter`, `ErrorHandler`, `CalibrationStrategy`) en ont toutes deux ou plus, réellement instanciées. La sur-ingénierie est concentrée dans les feuilles, et elle est modeste (détail chiffré en §8).

Autrement dit : **la réécriture intégrale ne se justifie pas par la suppression de code.** Si son objectif est de réduire la base, une passe `/ponytail-audit` suivie de six coupes ciblées atteint 100 % du gain pour ~1 % du risque. Si son objectif est autre — reprise de compréhension, modernisation Go 1.26, refonte de la qualité des tests (26 052 LOC de test pour 15 948 de prod, ratio 1,63) — alors la réécriture se défend, mais **il faut le dire, parce que ce n'est pas le même plan** : les critères de succès changent.

La décision est prise, ce plan la sert intégralement. Cette section existe pour qu'aucun lecteur futur ne croie que la réécriture a été engagée pour couper 316 lignes.

---

## 2. Ce que la révision 1 affirmait de faux

Chaque ligne ci-dessous a été vérifiée sur la machine cible. La révision 1 décrivait un environnement fantôme ; sa section « Prérequis » est intégralement remplacée par le §4.

| Réf. rév. 1 | Affirmation | Réalité vérifiée |
|---|---|---|
| §0.1 | « `gcc` présent, `-race` utilisable sans WSL » | `gcc` **absent**, `make` **absent**, `CGO_ENABLED=0`. `go test -race` ne compile pas nativement. WSL est le **seul** environnement `-race`. Cela invalide la boucle §3.6 de la rév. 1 (`go test ./<pkg>/... -race`) **et** le gate `check.sh`, qui ne tournent qu'en WSL. |
| §0.2, §0.3, §6.3 | « installer `golangci-lint` et `benchstat` » | Déjà installés côté Windows (`C:\Users\agbru\go\bin\`). **Absents de WSL**, où tournent pourtant `-race` et la baseline perf. Le partage des outils est exactement inversé par rapport au plan. |
| §0.4 | « ~24 fichiers déjà modifiés, bruit CRLF à trier » | Arbre **propre**. `git status --porcelain` ne rend que `?? PLAN.md`. Il n'y a aucun bruit à trier. |
| §0.5, §2.5 | « `TestGenerateParallelThresholds` échoue en baseline, dépendance probable au nombre de CPU » | Le test **passe** ici (24 CPU, branche `default` de [adaptive_test.go:60](internal/calibration/adaptive_test.go:60)). Le défaut est réel mais **dormant** : une attente périmée `expected := []int{0, 512, …}` à [adaptive_test.go:39](internal/calibration/adaptive_test.go:39), alors que [adaptive.go:28](internal/calibration/adaptive.go:28) émet `-1` comme baseline séquentielle depuis FIB-02. Rouge uniquement à 2–4 cœurs (conteneur bridé, portable). |
| §5 | « le skill `ponytail` n'est pas dans le catalogue, discipline appliquée manuellement » | Le skill **est** disponible ici, avec ses quatre commandes : `/ponytail` (mode), `/ponytail-review` (diff), `/ponytail-audit` (dépôt entier), `/ponytail-debt` (registre des `ponytail:` différés). Elles s'intègrent au workflow (§5.4). |
| §5 (ligne de portée) | « 17 861 LOC prod / 29 828 LOC test » | Mesuré : **15 948 / 26 052**. Surestimation de 12 % / 14 %. `CLAUDE.md` interdit explicitement de coder en dur un décompte daté. |
| §4, §6 | « scénario A séquentiel retenu ; B parallèle écarté » | Contredit la décision (C), qui impose le pipeline multi-agents sur les feuilles. La rév. 1 ne décrivait aucune mécanique de concurrence. Traité au §9.3. |

---

## 3. Le conflit entre les décisions (A) et (B), et sa résolution

C'est le point dur du plan. Il n'était pas connu quand les décisions ont été prises.

### 3.1 Le fait

**23 des 24 répertoires de test sont boîte-blanche.** Un seul fichier `_test.go` de tout le dépôt est en package externe : [internal/arch_test.go](internal/arch_test.go) (`package internal_test`). Tous les autres sont `package foo` et sont compilés **avec** la production. Dix-sept d'entre eux nomment directement des symboles privés : `acquireSizingForN` ([fastdoubling_test.go:98](internal/fibonacci/fastdoubling_test.go:98)), `maxCachedArenaWords` ([state_cache_test.go:61](internal/fibonacci/state_cache_test.go:61)), `wordSlicePools` ([misc_extra_test.go:340](internal/bigfft/misc_extra_test.go:340)), `isFermatPostConditionPanic` ([fft_recover_policy_test.go:31](internal/bigfft/fft_recover_policy_test.go:31)), `releaseWordSlice`, `putByKey`, `clearStateAliases`, `getFFTThreshold`, et ainsi de suite.

### 3.2 La conséquence

La décision (A) exige de geler ces tests comme oracle. La décision (B) autorise à renommer les symboles internes. **Renommer un seul de ces identifiants ne fait pas rougir une assertion : il casse la compilation du binaire de test.** Il n'y a alors plus d'oracle du tout — on n'atteint jamais l'exécution. La phase A, telle qu'énoncée, est littéralement inapplicable à `bigfft/`, `fibonacci/`, `fibonacci/memory/`, `fibonacci/threshold/`, `calibration/`, `config/`, `app/`, `tui/`, `cli/` et `cli/completion/`.

Symétriquement, geler la suite d'un package que la décision (B) autorise à **supprimer** n'a pas de sens : le test n'a plus de cible.

### 3.3 La résolution — trois mécanismes, appliqués avant toute réécriture

**a) Une API interne contractuelle, explicitement gelée.** Pour les deux packages du cœur, la liste suivante de symboles privés devient contractuelle : elle ne peut être ni renommée ni supprimée pendant la phase A. C'est le prix à payer pour garder un oracle exécutable sur le code le plus dangereux du dépôt.

| Package | Symboles privés gelés (phase A) |
|---|---|
| `internal/fibonacci` | `acquireSizingForN`, `clearStateAliases`, `finalizeStateReleaseTo`, `finalizeStateRelease`, `cachedState`, `statePool`, `bump`, `arenaCapWords`, `prepareStateForN`, `checkLimit`, `maxCachedArenaWords`, `maxArenaPoolWords`, `maxReasonableWords`, `fibonacciGrowthFactor` |
| `internal/fibonacci/memory` | `arenaTotalWords` |
| `internal/bigfft` | `acquireWordSliceUnsafe`, `releaseWordSlice`, `acquireFermat`, `releaseFermat`, `acquireFermatSlice`, `releaseFermatSlice`, `wordSlicePools`, `fermatPools`, `natSlicePools`, `fermatSlicePools`, `wordSliceSizes`, `getFFTThreshold`, `isFermatPostConditionPanic`, `putByKey`, `setCacheLogger`, `fourierRecursiveUnified`, `fourierRecursiveCtx`, `executeReconstruction`, `pointwiseMinParallelWords`, `newFermatVec`, `pinFFTParallelismConfig` |
| `internal/fibonacci/threshold` | `newTestDynamicThresholdManager` |
| `internal/app` | `wireThresholdTuning` |
| `internal/tui` | `handleReset`, champs `generation`, `ctx`, `cancel`, `done` |
| `internal/cli` | forme de `realSpinner` : les trois seams injectables `stop`, `start`, `setSuffix` |
| `internal/cli/completion` | `flagRegistry` et la forme de ses entrées |

Ces symboles restent réécrivables **à l'intérieur** — corps, algorithme, découpage en fichiers. Seul le nom et la signature sont gelés jusqu'à la phase B du package concerné.

**b) Un commit préparatoire de migration vers `package foo_test`.** Pour les packages dont les tests ne touchent en réalité que l'API exportée — `errors`, `format`, `metrics`, `metrics/system`, `progress`, `ui`, `parallel`, le test golden de `fibonacci`, les tests d'équivalence — un commit mécanique, **sans le moindre changement de production**, bascule l'en-tête en `package foo_test`. Ces suites deviennent alors de vrais oracles boîte-noire, immunisés contre le renommage des internes. Coût : un commit par package, aucun risque.

**c) Le repli sur les oracles à surface stable.** Pour ce que ni (a) ni (b) ne couvrent, l'oracle de phase A n'est plus la suite unitaire mais le comportement observable : golden JSON, directives `// Output:` des `Example`, corpus de fuzz, `test/e2e`, `arch_test`. Cela vaut en particulier pour la partie de `bigfft/` et `fibonacci/` que la réécriture voudra restructurer au-delà de la liste gelée.

### 3.4 La validation d'oracle de phase B — ce qui est possible et ce qui ne l'est pas

La décision (A) prévoit de valider le nouvel oracle en le passant contre l'ancienne production (doit être vert) puis contre une mutation volontaire (doit être rouge).

**La première moitié est impossible dans le cas général, et il faut le reconnaître.** Si la phase A a renommé `acquireSizingForN` en `sizeArenaFor`, le nouveau test de phase B appelle `sizeArenaFor`, qui n'existe pas dans le worktree de l'ancienne production : `go test` échoue à la compilation, pas au comportement. Le run croisé n'a de sens **que** pour les tests dont la surface consommée n'a pas bougé.

Substitut retenu :

```
git worktree add ../fibgo-pre <commit-avant-phase-A>
# n'y exécuter QUE les tests de phase B qui ne consomment que l'API exportée inchangée.
# Pour tout le reste, l'anti-régression comportementale passe par golden + e2e + Example, pas par un run croisé.
```

**La seconde moitié — la mutation — doit être spécifiée, sinon elle valide un oracle faible.** Changer un message d'erreur fait rougir un test d'égalité de chaîne et ne prouve rien. Protocole retenu, sans nouvelle dépendance (`go-mutesting` est externe : interdit) : un catalogue de 8 à 12 mutations scriptées par module sensible, appliquées en patch réversible, réparties en trois familles.

| Famille | Exemple de mutation | Test qui **doit** rougir |
|---|---|---|
| Correction algorithmique | inverser l'identité F(2k) dans `fastdoubling.go` ; décaler un index de butterfly dans `fft.go` | golden + fuzz |
| Frontière / clamp | retirer un clamp de `acquireSizingForN` | `TestAcquireStateForN_HugeN_NoPanic` |
| Invariant de concurrence | supprimer un `clearStateAliases` avant un sink | `TestReleaseState_OverLimit_AliasesCleared` |

Les mutations **purement perf** (`×10` → `×8` sur le multiplicateur d'arène) sont **exclues** du critère « doit rougir » : aucun test comportemental ne les attrape, et c'est correct. Le catalogue doit le dire, faute de quoi on conclura à tort que l'oracle est troué. Preuve exigée : chaque mutation du catalogue produit au moins un test rouge **nommé**, journalisé au §12.

---

## 4. Étape 0 — prérequis réels (une seule fois)

### 4.1 Environnement, tranché

**WSL devient l'autorité unique de tous les gates lourds.** C'est la seule façon de sortir du « deux OS, deux vérités » : la baseline perf committée est déjà en `goos: linux`, `-race` et `-tags gmp` n'existent que là, et le `Makefile` est un Makefile POSIX (`date -u`, `mkdir -p`, `grep`) inutilisable depuis PowerShell.

| Rôle | Environnement | Commande |
|---|---|---|
| Build, vet, itération rapide | Windows | `go build ./... && go vet ./...` |
| Tests avec assertions d'allocation (`//go:build !race`) | Windows | `go test ./...` |
| Tests + détecteur de courses (**autorité**) | WSL | `wsl bash -lc "cd /mnt/c/.../FibGo && go test -race ./..."` |
| Backend GMP | WSL | `wsl bash -lc "... go build -tags gmp ./... && go test -tags gmp -race ./internal/fibonacci/"` |
| Benchmarks + `benchstat` | WSL | voir §7 |
| Lint | Windows ou WSL | `golangci-lint run ./...` (advisory) |
| Gate complet | WSL | `scripts/check.sh` |

`scripts/check.ps1` **n'est pas** un substitut de `check.sh` : il perd `-race` ([check.ps1:62](scripts/check.ps1:62)) **et** l'étape 3b GMP. La révision 1 les présentait comme équivalents (« `check.sh` ou `check.ps1` ») ; ils ne le sont pas. `check.ps1` est un filet dégradé, jamais un gate de merge.

- **0.1** — Installer les outils manquants dans WSL : `wsl go install golang.org/x/perf/cmd/benchstat@latest` et `golangci-lint`. Vérifier : `wsl bash -lc "benchstat -h"`.
- **0.2** — Vérifier `gcc` et `libgmp-dev` dans WSL (déjà présents : `/usr/include/x86_64-linux-gnu/gmp.h`).
- **0.3** — Aucun tri de bruit git à faire. L'arbre est propre.

### 4.2 Baselines à capturer avant la moindre ligne réécrite

Une baseline est un artefact daté, produit sur la machine et l'OS qui feront tourner le gate. Trois sont nécessaires ; **aucune ne doit être régénérée pour masquer une régression** (parallèle à la Directive #1).

1. **Baseline de comportement** — `wsl go test -race ./... ` et `go test ./...` (Windows) : capturer le vert intégral, à un test près (§4.3).
2. **Baseline de performance locale** — `docs/audits/bench-local-pre.txt`, capturée en WSL, protocole du §7.2. **Ne pas comparer à `docs/audits/bench-baseline.txt`** : voir §7.1, ce fichier est inexploitable comme référence de gate.
3. **Baseline de couverture par package** — `docs/audits/coverage-baseline.txt`. Le plancher actuel est un total module ([check.sh:89](scripts/check.sh:89), [check.ps1:104](scripts/check.ps1:104)) : il est aveugle. Réécrire `calibration` (~10 % du module) en laissant une branche d'erreur non testée fait tomber ce package à 60 % pendant que le total reste au-dessus de 80 %. Symétriquement, supprimer du code défensif fait **monter** le pourcentage sans un seul test ajouté — « la couverture a monté » ne prouve rien. Capturer donc, en un seul mode (**sans `-race`**, sinon le tag `!race` de [fft_poly_pool_leak_test.go:1](internal/bigfft/fft_poly_pool_leak_test.go:1) produit deux totaux différents pour le même dépôt) :

```
go test -cover ./... | Select-String "coverage:" > docs/audits/coverage-baseline.txt
```

Gate par package : `couverture_réécrite >= baseline_package - 1,0 pp` **et** `>= 80 %` absolu. Le plancher global 80 % subsiste, comme filet secondaire.

### 4.3 Le correctif préalable (Directive #7 — bug fix avant refactor)

`GenerateParallelThresholds` renvoie `-1` comme baseline séquentielle depuis FIB-02 ([adaptive.go:24-28](internal/calibration/adaptive.go:24)). L'attente de la branche `numCPU <= 4` n'a jamais suivi : [adaptive_test.go:39](internal/calibration/adaptive_test.go:39) cherche encore `0`. Le test est vert à 24 CPU, rouge à 2–4.

Le correctif d'une ligne (`0` → `-1`) fait passer le test **sans le rendre vérifiable** : `runtime.NumCPU()` n'obéit pas à `GOMAXPROCS`, on ne peut donc pas exercer la branche sur cette machine. Correctif retenu, minimal et testable :

```go
// adaptive.go
func GenerateParallelThresholds() []int { return generateParallelThresholds(runtime.NumCPU()) }
func generateParallelThresholds(numCPU int) []int { /* corps actuel, numCPU injecté */ }
```

Le test devient table-driven sur `{1, 2, 4, 8, 16, 24}` et couvre les cinq branches — dont celle qui est aujourd'hui morte sur toute machine de développement. Commit isolé : `fix(calibration): cover CPU-gated threshold branches, align ≤4-CPU expectation with FIB-02 baseline`.

### 4.4 Le commit préparatoire de durcissement d'oracle — le cœur du plan

La passe adversariale a produit **25 scénarios de défaillance, dont 19 que le gate actuel ne détecterait pas**. Dans quatre cas, le test gardien nommé par `CLAUDE.md` a été lu et **passe au vert sur la régression**. Ces trous ne sont pas des risques de la réécriture : ils existent aujourd'hui, dans le code stabilisé. Les combler **avant** de réécriture est la seule action de ce plan dont le rapport valeur/risque soit franchement positif.

Neuf gardiens à ajouter ou corriger, chacun en commit isolé, **aucun changement de comportement de production** (sauf le n°3, signalé) :

| # | Trou prouvé | Gardien à poser |
|---|---|---|
| 1 | Le multiplicateur d'arène `×10` vit en **deux littéraux indépendants** ([fastdoubling.go:308](internal/fibonacci/fastdoubling.go:308), [memory/arena.go:32](internal/fibonacci/memory/arena.go:32)) ; chaque test se compare à **sa propre** copie naïve. Aucun test ne compare les deux fonctions **entre elles**. Divergence `×8`/`×10` → deux suites vertes, arène désynchronisée. | Test croisé : `acquireSizingForN(n).totalWords == memory.arenaTotalWords(n)` sur un échantillon de `n`. C'est le seul gardien réel de l'invariant « les deux littéraux restent synchronisés » de `CLAUDE.md`. |
| 2 | `TestTransformCacheEvictionRecyclesBacking` n'assert **jamais** que le nouveau backing a une adresse **distincte** de l'évincé. Réintroduire le recyclage (la fenêtre use-after-free d'Audit-PRD E1-R4) rend le test **plus** vert : moins d'allocations. | Assertion d'adresse distincte, ou test UAF direct : garder une `PolValues` issue d'un `Get()`, forcer son éviction, vérifier que ses mots sont intacts. |
| 3 | Les trois chaînes-sentinelles de post-condition sont **dupliquées** entre les `panic()` de [fermat.go:201,226,281](internal/bigfft/fermat.go:201) et la map de [fft.go:267-271](internal/bigfft/fft.go:267). Reformuler un message sans toucher la map convertit une violation de post-condition en `error` — le bug réel est masqué (violation ADR-0002), les trois tests restent verts car ils comparent la map à elle-même. | Extraire les trois messages en constantes partagées, consommées par les deux sites. Ajouter un test qui **déclenche vraiment** un panic via `fermat.Mul`/`Sqr` et vérifie la re-propagation. *(Seule entrée du tableau qui touche la production : commit `fix(bigfft):` séparé.)* |
| 4 | `fourierRecursiveUnified` enveloppe la moitié synchrone dans un `recover()` puis `wg.Wait()` **avant** de re-panic ([fft_recursion.go:157-171](internal/bigfft/fft_recursion.go:157)) : sans cela, un panic sync déroule pendant qu'un worker async écrit encore les tampons poolés. Le gardien ne plante que la moitié **async**. Inliner l'appel → panic toujours propagé, test vert, course réintroduite. | Test qui plante la moitié **sync** pendant qu'un worker async tourne, exécuté sous `-race` en WSL. |
| 5 | SEC-01 : le sous-test « profil forgé » ne forge en négatif que `OptimalParallelThreshold` ([calibration_test.go:263](internal/calibration/calibration_test.go:263)). Une re-validation partielle *parallel-only* passe au vert pendant qu'un seuil FFT ou Strassen forgé fuite dans la config. | Trois sous-tests, un par seuil forgé négatif. |
| 6 | APP-05 : `TestModel_HandleReset_FreshTimeoutBudget` ne vérifie que le **nouveau** contexte. Omettre `m.cancel()` ([handlers.go:138-149](internal/tui/handlers.go:138)) fuit l'ancien timer et laisse le calcul de la génération précédente tourner : test vert. | Capturer l'ancien `ctx` avant `handleReset`, vérifier qu'il est `Done` après. |
| 7 | `doubling_framework.go` — la boucle critique — n'a **aucun** test d'allocation. `grep AllocsPerRun` sur `internal/fibonacci` ne rend rien. Une allocation ajoutée par itération laisse le golden vert et n'est visible que par un `benchstat` lui-même noyé dans le bruit (§7.1). | `testing.AllocsPerRun` bornant un pas de doublement pour un `N` moyen, hors `-race`. |
| 8 | `MetricsBuffer` n'est pas goroutine-safe ; son gardien nommé (`TestConcurrentAccess`) n'exerce que l'API publique du `Manager` et ne vaut **que** sous `-race` — absent du gate Windows. | Test concurrent **direct** sur `MetricsBuffer` (`Record` en parallèle de `RecentMetrics`), exécuté en WSL. |
| 9 | Aucun gate ne vérifie que `CLAUDE.md` ne ment pas. Après réécriture, il nommera des fichiers et des tests disparus — et un futur agent, ne trouvant plus `finalizeStateReleaseTo`, conclura que l'invariant a disparu et réintroduira le double teardown. C'est exactement la régression que `CLAUDE.md` existe pour empêcher. | Test qui extrait les 24 noms de gardiens cités dans `CLAUDE.md` et assert l'existence de chaque `func Test<nom>`. Échoue net dès que la doc dérive. |

Les 24 tests gardiens nommés par `CLAUDE.md` ont tous été localisés : **aucun gardien fantôme**. La dérive de documentation est nulle aujourd'hui. Le test n°9 la maintient nulle demain.

---

## 5. La boucle par package

Pour chaque package, dans l'ordre du §6. Aucune étape n'est facultative.

### 5.1 Lecture et recoupement

1. Lire l'intégralité du package : production, tests, `doc.go`. Extraire le **contrat réel** (comportement observable), pas les signatures.
2. Recouper avec `CLAUDE.md` § Invariants. Si le package y figure : écrire noir sur blanc, avant la première ligne, quels invariants et quels tests gardiens sont en jeu.
3. Recouper avec la liste des symboles gelés (§3.3a) et le registre des défaillances silencieuses (§11).

### 5.2 Phase A — production réécrite, oracle gelé

4. La suite de tests existante **ne bouge pas d'une ligne**. Elle doit compiler et passer.
5. Réécrire la production sous discipline ponytail (§5.4), en respectant les symboles gelés.
6. Gate de phase A, dans l'ordre :
   - `go build ./... && go vet ./...` (Windows) — l'arbre **entier**, pas seulement le package : un `-ldflags -X` ou un `go:build` cassé ne se voit pas autrement.
   - `go test ./<pkg>/... -count=1` (Windows, sans `-race`) — couvre les assertions `AllocsPerRun` sous `//go:build !race`.
   - `wsl go test -race ./<pkg>/... -count=1` — autorité.
   - `golangci-lint run ./<pkg>/...` — advisory, mais toute nouvelle alerte est examinée.
   - Couverture du package `>= baseline - 1,0 pp` et `>= 80 %`.
   - Si `fibonacci/` ou `bigfft/` : gate perf rapide (§7.3).
7. Commit `refactor(<pkg>): rewrite production, tests unchanged`. La justification exigée par la Directive #5 (> 50 LOC sur > 2 fichiers) figure dans le corps du message : raison, compromis, alternative écartée.

### 5.3 Phase B — tests réécrits, production gelée

8. La production **ne bouge pas d'une ligne**.
9. Réécrire la suite : table-driven où pertinent, `t.Parallel()` systématique (le dépôt est déjà à ~100 %, sauf `internal/ui` à 0 % — dette à solder). Couverture fonctionnelle au moins égale, cas limites compris.
10. **Interdits absolus en phase B** :
    - `fibonacci/testdata/fibonacci_golden.json` — immuable sans ADR, aucun `-update` n'existe et aucun ne doit apparaître.
    - Les cinq directives `// Output:` des `Example` ([example_test.go:21,49,81,96](internal/fibonacci/example_test.go:21), [fibonacci_test.go:228](internal/fibonacci/fibonacci_test.go:228)). Elles figent des chaînes de production (`Name()` des calculateurs) et **l'ordre** de `factory.List()`. Ce sont des oracles au même titre que le golden ; les changer exige un ADR.
    - Le corpus `testdata/fuzz/*` — déplacé par `git mv`, jamais régénéré. Ce sont des entrées de non-régression déjà trouvées.
11. Valider le nouvel oracle : protocole de mutation du §3.4. Journaliser au §12 chaque mutation et le test rouge qu'elle produit.
12. Commit `test(<pkg>): rewrite suite, production unchanged` + le journal de mutation.

### 5.4 La discipline ponytail, mécanisée

Le skill est disponible dans la session. Il ne remplace jamais un invariant : **en cas de conflit, l'invariant gagne, sans discussion.**

| Moment | Commande | Usage |
|---|---|---|
| Avant la vague | `/ponytail-audit` | Liste classée des coupes du périmètre de la vague. Déjà exécutée une fois : §8. |
| Après la phase A d'un package | `/ponytail-review` | Revue du diff, sur-ingénierie uniquement. Une ligne par constat. |
| Fin de vague | `/ponytail-debt` | Registre des commentaires `ponytail:` posés. Tout marqueur sans déclencheur de reprise est signalé — ceux-là pourrissent en silence. |

L'échelle, appliquée dans l'ordre, arrêt au premier barreau qui tient : (1) ce code doit-il exister ; (2) existe-t-il déjà dans le dépôt ; (3) la stdlib le fait ; (4) une fonctionnalité native le fait ; (5) une dépendance déjà déclarée le fait ; (6) une ligne suffit ; (7) le minimum nécessaire. **Jamais simplifiés** : la validation aux frontières de confiance, la gestion d'erreurs qui prévient une perte de données, la sécurité, la concurrence contrôlée, et tout invariant documenté.

Tout raccourci délibéré laisse un commentaire `// ponytail:` nommant son plafond et son chemin de reprise.

---

## 6. Ordre d'exécution — réordonné

La révision 1 plaçait `fibonacci/` et `bigfft/` en vague 3, après leurs consommateurs. Le graphe de dépendances réel rend cet ordre incohérent sur deux points :

- `config` (vague 1) importe `fibonacci/memory` (vague 3) ;
- `orchestration`, `calibration` et `app` (vague 2) importent `fibonacci` et `bigfft` (vague 3), en production **et** dans leurs tests.

Réécrire un consommateur avant son fournisseur oblige à le re-toucher quand le fournisseur bouge — et pire, un oracle de vague 2 « validé » redevient rouge après la vague 3, sans qu'aucun critère ne l'anticipe.

**Correction : le cœur passe en premier.** Ce n'est pas une entorse à la décision (C) — elle impose que le cœur soit traité *séquentiellement*, pas *tardivement*. `bigfft` est d'ailleurs une **feuille pure** (zéro import interne) ; son report était perf-motivé, jamais topologique. Front-charger le risque est aussi la bonne discipline : si le cœur échoue au gate perf, on abandonne avant d'avoir dépensé dix sessions sur la périphérie.

| Vague | Packages, dans l'ordre | LOC prod | Mode | Invariants en jeu | Sortie de vague |
|---|---|---|---|---|---|
| **P** | correctif `calibration` (§4.3) ; 9 gardiens (§4.4) ; migration `foo_test` (§3.3b) ; 3 baselines (§4.2) | ~0 | séquentiel | tous, en lecture | arbre vert, baselines figées |
| **1** | `bigfft` → `fibonacci/memory` → `fibonacci/threshold` → `fibonacci` (+ suppression de `internal/parallel`, + inline de `fibonaccitest`) | 7 758 | **séquentiel, Opus** | quasi tous | `benchstat` complet ; golden vert ; `wsl go test -race ./...` **entier** ; `-tags gmp` ; profil PGO régénéré |
| **2** | `errors`, `progress`, `ui` (les trois hubs — API exportée gelée dès la fin de vague), puis `format`, `metrics`, `metrics/system`, `testutil`, `cmd/generate-golden` | 1 250 | **pipeline multi-agents** | `errors` n'importe pas `format` | `arch_test` vert ; `test/e2e` vert |
| **3** | `config`, `cli/completion` → `orchestration` → `calibration` → `cli`, `tui` → `app` | 4 865 | pipeline puis séquentiel sur `app` | SEC-01, CONC-01, APP-03/05/11, `tui ↛ fibonacci`, `orchestration ↛ format`, `threshold ↛ config` | `arch_test` + `test/e2e` verts ; `check.sh` complet |
| **4** | `cmd/fibcalc` | 39 | séquentiel | aucun dédié | `check.sh` ; `--version` non-`dev` (§11) |

`internal` (`arch_test.go`) et `test/e2e` ne sont **pas** des cibles de réécriture : ce sont des gardiens transversaux, exécutés verts à chaque gate. La révision 1 les ignorait.

**Critère de sortie inter-vague, ajouté** : aucun oracle n'est définitif tant qu'un package dont sa suite dépend transitivement n'est pas gelé. Concrètement, après **chaque** vague : `wsl go test -race ./...` sur l'arbre **entier**, jamais seulement sur le package courant.

---

## 7. Le gate de performance — la Directive #1 est inopérante en l'état

C'est le second point dur, et il est prouvé.

### 7.1 Le constat

`docs/audits/bench-baseline.txt` est produit par [Makefile:232-233](Makefile:232) avec `-count=5 -benchtime=1x` : **cinq échantillons d'une seule itération chacun**. Ses propres données le condamnent :

```
BenchmarkFibonacci/FastDoubling/1M-24    1   4501888 ns/op   10574536 B/op   192 allocs/op
BenchmarkFibonacci/FastDoubling/1M-24    1   3069782 ns/op    1313552 B/op   109 allocs/op
```

Sur `FastDoubling/1M`, l'écart entre échantillons de la baseline atteint **+46 %** en temps et **×8** en octets alloués (le premier échantillon capture la mise en chauffe des pools). Sur `/10M`, **+52 %**. **Un seuil de blocage à 5 % est très en dessous de l'écart-type de sa propre référence.** `benchstat` rendra `~` (p élevé) sur une vraie régression de 5 %, ou criera au faux positif, indifféremment.

S'y ajoutent deux défauts :

- L'en-tête dit `goos: linux` (la baseline a été produite sous WSL) et `cpu: baseline-2026-07-07` — un **littéral daté**, pas un modèle de processeur. Le fichier ne permet même pas de vérifier qu'il vient de la même machine.
- **Les benchmarks ne bénéficient pas du PGO.** `cmd/fibcalc/default.pgo` est le seul `.pgo` du dépôt ; Go ne l'applique qu'au package `main` de son répertoire, et `go test` synthétise son `main` ailleurs. Le gate mesure donc du code **sans PGO** pendant que `make build` expédie un binaire **avec**. Il faut l'écrire, pour que personne ne croie que le gate reflète la production.

### 7.2 Le protocole retenu

Baseline **locale**, en WSL (l'OS de la baseline committée et le seul avec `-race`), machine au calme, avant toute réécriture :

```bash
go test -run='^$' -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' \
        -benchmem -count=10 -benchtime=3x ./internal/fibonacci/ > docs/audits/bench-local-pre.txt
```

Après réécriture, même commande vers `bench-local-post.txt`, puis `benchstat bench-local-pre.txt bench-local-post.txt`.

**Double critère de blocage** : régression `> 5 %` **et** `p < 0,05`. Un delta sans significativité statistique n'est pas une régression ; une significativité sans delta n'est pas un blocage.

Garde-fou anti-baseline-vide : `-bench` encode en dur `BenchmarkFibonacci` et ses sous-tests `FastDoubling|MatrixExp|FFTBased`, ainsi que le chemin `./internal/fibonacci/`. Renommer l'un d'eux fait matcher **zéro** benchmark ; `go test` sort en 0 et le fichier de baseline est **vide, sans erreur**. Après toute régénération : vérifier que le fichier contient au moins six lignes `^Benchmark`.

### 7.3 Gate à deux vitesses

Un `benchstat` complet à chaque fichier sensible (la révision 1 en listait onze) n'est pas tenable. Deux régimes :

- **Rapide, par fichier** : `FastDoubling/1M` seul, `-count=6 -benchtime=3x`. Quelques secondes. Détecte l'accident grossier.
- **Complet, par vague** : les six sous-tests, `-count=10 -benchtime=3x`, machine au calme. C'est ce gate-là qui bloque.

Le profil PGO est régénéré (`make pgo-profile`, en WSL, machine au calme, **jamais** `-tags gmp`) **après** stabilisation du cœur, avant de reconstruire la production. Contrôle de fraîcheur : `go tool pprof -top cmd/fibcalc/default.pgo` — si le top-10 des symboles ne résout plus vers des fonctions existantes, le profil est périmé et le linker l'ignore **en silence**.

---

## 8. Ce que la réécriture rapporte réellement

Audit ponytail du dépôt entier, importateurs comptés, invariants respectés. Classé du plus gros gain au plus petit.

| Tag | Coupe | Remplacement | Lignes | Vague |
|---|---|---|---|---|
| `native` | `internal/parallel` en entier (`ErrorCollector` = `sync.Once` + `error`) et son plumbing manuel dans les trois helpers de `fibonacci/common.go`. Un seul importateur. | `golang.org/x/sync/errgroup`, **déjà** dépendance directe et **déjà** utilisé en production dans `orchestration/`. Conserver `globalSem` (P2-02) : `errgroup.SetLimit` est par-groupe et ne le remplace pas. | 95 | 1 |
| `yagni` | Le pattern Observer/Subject de `progress` (`Register`/`Notify`/`ObserverCount`) : la production n'enregistre **qu'un** observateur. | Passer le canal de progression directement. | *(95)* | — |
| `delete` | `LoggingObserver` + `NoOpObserver` ([observers.go:67](internal/progress/observers.go:67)) : **zéro appelant**, prod ou test, hors du package. | Rien. Suppression pure. | 73 | 2 |
| `yagni` | Le package `internal/fibonacci/fibonaccitest` : un seul consommateur, `orchestration/contract_test.go`. | Déclarer `CoreStub` dans son unique test appelant. Un package de production entier disparaît. | 69 | 1 |
| `yagni` | Le package `internal/metrics/system` : un seul appelant, `tui/commands.go:99`. | Inliner `Sample()`. `gopsutil` reste, la feature est légitime. | 34 | 2 |
| `yagni` | L'interface `CacheStrategy` ([cache_strategy_bigfft.go:21](internal/fibonacci/cache_strategy_bigfft.go:21)) : une seule implémentation de production. | Appeler `decideCacheTuning` (fonction pure) directement. | 25 | 1 |
| `shrink` | Le `doc.go` de `internal/testutil` (25 lignes) documente une fonction de 21 lignes. | Trois lignes. `StripAnsiCodes` reste un package importable : trois fichiers de test l'utilisent, ne pas dupliquer le regexp. | 20 | 2 |

**Total net : ~316 lignes de production, soit 2,0 % du dépôt. Zéro dépendance supprimable.**

La ligne *Observer/Subject* (95 lignes de plus) est **exclue du net** : la Directive #6 impose une consultation avant toute modification structurelle du chemin `progress`, et elle touche `fibonacci/calculator.go`. À proposer, pas à couper d'office.

Deux constats hors périmètre, à signaler sans corriger (Directive : « problème hors sujet repéré → signale-le ») :

- `internal/config` est **sur-testé**, pas sur-construit : 1 756 LOC de test pour 905 de production, tirées par deux fichiers exhaustifs qui se recouvrent (`config_test.go` 508 lignes + `config_exhaustive_test.go` 569).
- `internal/fibonacci` réinvente `errgroup` via `parallel.ErrorCollector` alors que `errgroup` est déjà en production dans `orchestration`. C'est le constat ponytail le plus net du dépôt.

---

## 9. Mécanique d'exécution

### 9.1 Le contrat intangible — checklist de fin de vague

Dix contrats, vérifiés dans le code, verrouillés par les tests. Ce que la décision (B) fige.

1. **25 flags** enregistrés dans le `FlagSet` ([config.go:150-179](internal/config/config.go:150)), plus `version`/`-V` et `help`/`-h` traités hors bande. `FlagNames()` alimente `TestFlagRegistryInSyncWithConfig` : retirer un alias casse la complétion.
2. **6 codes de sortie** : `0` succès, `1` générique, `2` timeout, `3` mismatch d'algorithmes, `4` config, `130` annulation. `--version` ne fait **pas** `os.Exit`. Seuls `0` et `2` sont assertés strictement en e2e ; les autres sont figés par les constantes.
3. **Format stdout** : mode normal `F(n) = 190,392,490,709,135` (groupé par virgules) ; `--quiet` décimal **brut** ; `--last-digits K` zéro-padding à exactement K ; `--output` écrit un fichier à en-tête `#` ; `--version` imprime `fibcalc <v>`, `Commit:`, `Built:`, `Go version:`, `OS/Arch:`.
4. **14 overrides `FIBCALC_*`**, priorité stricte CLI > env > défaut, **rejet bruyant** d'un env malformé (jamais de repli silencieux).
5. **4 shells de complétion** avec leurs marqueurs (`complete`, `compdef`, `complete -c fibcalc`, `Register-ArgumentCompleter`), et le contrat dur : chaque flag du registre apparaît **littéralement**, forme courte **et** longue, dans les quatre scripts.
6. **Golden JSON** : 27 entrées `{n, result}`, immuable, aucun `-update`. La duplication délibérée `calculateSmall` / `fibBig` entre `internal/fibonacci` et `cmd/generate-golden` (P2-04) **ne doit pas être dédupliquée** — sans elle, le golden devient une tautologie.
7. **Les 4 arrows interdites** d'`arch_test` : `threshold ↛ config`, `errors ↛ format`, `tui ↛ fibonacci`, `orchestration ↛ format`. Vérification par imports **directs** seulement.
8. **Les directives `// Output:`** des `Example` (chaînes `Name()`, ordre de `factory.List()`).
9. **Le corpus `testdata/fuzz/*`**.
10. **Valeurs d'`--algo`** : `all` (défaut), `fast`, `matrix`, `fft`, plus `gmp` sous build tag. `DefaultN=100_000_000`, `DefaultTimeout=5m`, `EnvPrefix=FIBCALC_`.

### 9.2 Commits, tags, repli

**`git revert` n'est pas la primitive de repli.** Dès qu'un commit change la topologie des packages — ce que (B) autorise — le revert d'un maillon d'une chaîne dépendante réussit sans conflit git et laisse un arbre qui ne compile pas.

Règles :

- Une fusion ou suppression de package est **un seul commit** contenant le package fusionné **et** la mise à jour de tous les chemins d'import. Jamais fractionnée.
- Après chaque package dont le gate passe et dont l'arbre **entier** compile : `git tag rewrite/v<vague>/<pkg>-green`.
- La primitive de repli est `git reset --hard <tag-vert>`, pas `git revert`.
- Commits conventionnels : `refactor(<pkg>):` (phase A), `test(<pkg>):` (phase B), `fix(<pkg>):` (correctif isolé, Directive #7).
- Push direct sur `main` après chaque package vert (trunk-based, mainteneur solo).

**`CLAUDE.md` se met à jour dans le même commit** que la réécriture qui renomme ou supprime un fichier ou un symbole qu'il nomme. Entre deux commits, la doc mentirait et un point de reprise « vert » serait faux. Chaque invariant est ré-ancré en deux moitiés : (a) la description du mode de défaillance, indépendante des noms, préservée **verbatim** ; (b) l'ancre courante `fichier:symbole` + test gardien, mise à jour en lockstep. **ADR-0010 obligatoire** (Directive #5, précédent ADR-0009) : table de correspondance ancien → nouveau, pour tracer la lignée des invariants à travers des dizaines de commits.

### 9.3 Multi-agents — la mécanique que la révision 1 n'avait pas

La décision (C) impose le pipeline sur les feuilles. La révision 1 retenait le séquentiel intégral et ne décrivait aucune concurrence. Contradiction levée ici.

Deux agents réécrivant `errors` et `config` dans le **même** arbre de travail produisent trois défaillances, dont deux silencieuses :

- l'agent A a supprimé un symbole d'`errors` avant d'en réécrire les appelants ; le gate de l'agent B (`config` importe `errors`) rougit et l'échec lui est imputé à tort ;
- les deux gates écrivent le **même** `coverage.out` à la racine ([check.ps1:62](scripts/check.ps1:62)) : le run de B écrase celui de A et le plancher de couverture lit un profil issu du mauvais run ;
- `arch_test` shelle `go list` et fait `t.Fatalf` si un package voisin est momentanément non compilable ([arch_test.go:104](internal/arch_test.go:104)) : faux rouge d'architecture.

Règles :

- **Un `git worktree` par agent** (`git worktree add ../fibgo-agentX`) : index et arbre de travail distincts, store `.git` et `GOCACHE` partagés. Coût : une première compilation par worktree.
- **L'authoring parallélise ; le gate reste série.** Aucun gate whole-module (`go build ./...`, `arch_test`, `coverage.out`, `check.sh`) ne tourne dans le worktree d'un agent — le total de couverture y est faux par construction, un seul package y étant réécrit. Le gate d'intégration tourne **une fois par vague**, en série, dans l'arbre fusionné.
- `-race` n'existe que dans l'unique environnement WSL : les gates `-race` concurrents sont de toute façon infaisables.

### 9.4 Volume et points de reprise

Décompte réel : environ 26 unités de commit minimum (4 en vague P + 4 unités séquentielles en vague 1 + 8 feuilles + 6 en vague 3 + 1 en vague 4), doublées par le modèle en deux phases. **Estimation : 30 à 50 commits, 10 à 17 sessions.** La vague 1 (cœur, séquentiel, `benchstat`, WSL) domine : compter une session par unité sensible.

Règle dure : **une session ne se termine jamais sur un package à moitié réécrit.** Soit les deux phases sont finies et le package est vert (tag `-green`), soit le package est intact. Un état « nouveaux tests rouges contre ancienne production » est valide en cours de session, jamais à sa frontière.

Jalons réellement déployables : fin de vague P, fin de vague 1 (après re-run `-race` de l'arbre entier), fin de vague 2, fin de vague 3, fin de vague 4.

---

## 10. Critères de succès et d'arrêt

### 10.1 Par package

- `go build ./...` et `go vet ./...` propres sur l'**arbre entier**.
- Suite verte sous `-race` (WSL) **et** sans `-race` (Windows, pour les gardiens `//go:build !race`).
- Couverture du package `>= baseline - 1,0 pp` **et** `>= 80 %`.
- `golangci-lint` sans nouvelle alerte bloquante.
- Invariants documentés **vérifiés explicitement**, jamais supposés : le test gardien nommé est exécuté et son nom figure au rapport.
- Diff revu par `/ponytail-review`.

### 10.2 Par vague

- `arch_test` vert.
- `test/e2e` vert (il ne skippe que sous `-short`, et `check.sh` ne passe pas `-short`).
- `wsl go test -race ./...` sur l'arbre **entier** — c'est le seul filet qui attrape la re-rougissure d'un oracle de vague antérieure.
- Vague 1 spécifiquement : `benchstat` complet sous le double critère (§7.2), golden inchangé, `wsl go build -tags gmp ./...` et `wsl go test -tags gmp -race ./internal/fibonacci/`, profil PGO régénéré et contrôlé.

### 10.3 Global

- `scripts/check.sh` vert de bout en bout, en WSL. `check.ps1` ne suffit pas.
- `CLAUDE.md` à jour, ADR-0010 rédigé, `CHANGELOG.md` mis à jour.
- `build/fibcalc --version` reflète `git describe`, **pas** `dev` : le linker ignore **silencieusement** un `-X` vers un symbole inexistant ([Makefile:22-25](Makefile:22)), et l'e2e n'exige pas une version non-défaut.

### 10.4 Arrêt (suspension, pas abandon silencieux)

Une vague est suspendue si :

- un invariant documenté ne peut être préservé sans compromis non trivial ;
- une régression `benchstat` dépasse 5 % avec `p < 0,05` sans cause identifiable rapidement ;
- un golden, une directive `// Output:` ou une graine de fuzz devrait être modifié → **ADR requis avant de continuer**, jamais de contournement ;
- une mutation du catalogue (§3.4) ne produit **aucun** test rouge : l'oracle de phase B est insuffisant, il se corrige avant d'avancer.

---

## 11. Registre des défaillances silencieuses

Dix-neuf scénarios sur vingt-cinq ne seraient détectés par **aucun** gate existant. Les huit ci-dessous ont été vérifiés dans le corps du test gardien : le gardien passe **vert** sur la régression. Chacun a son garde-fou au §4.4, sauf mention contraire.

| Régression | Gardien nommé | Ce que le gardien vérifie en réalité |
|---|---|---|
| Recyclage du backing à l'éviction (UAF) | `TestTransformCacheEvictionRecyclesBacking` | allocs `<= 16`, capacités homogènes, `evictions > 0`. **Aucune** assertion d'adresse. Le recyclage rend le test *plus* vert. |
| Message de post-condition `fermat` reformulé | `TestFermatPostConditionPanicClassifier` | compare la map à elle-même. Ne déclenche jamais un vrai panic. |
| `wg.Wait()` retiré avant le re-panic sync | `TestFourierRecursiveAsyncPanicPropagates` | plante la moitié **async** ; la moitié sync-qui-panique n'est jamais exercée. |
| Re-validation SEC-01 réduite à un seul seuil | `TestAutoCalibrateWithProfile` | ne forge en négatif que le seuil *parallel*. |
| `m.cancel()` omis dans `handleReset` | `TestModel_HandleReset_FreshTimeoutBudget` | ne teste que le **nouveau** contexte. |
| `×10` désynchronisé entre les deux littéraux | `TestArenaTotalWords_ClampNoUB` + miroir | chacun se compare à **sa propre** formule. |
| Lecture `MetricsBuffer` hors `mu` | `TestConcurrentAccess` | n'exerce que l'API du `Manager`, et seulement sous `-race` (absent du gate Windows). |
| Seams `realSpinner` inlinés (CONC-01) | `TestUpdateSuffix_StopWriteStartOrder` | casse à la **compilation** en phase A (bien), mais la phase B autorise à réécrire le test : l'invariant peut s'évaporer sans échec. **Garde-fou : CONC-01 est non réécrivable en phase B** sans seam équivalent + test d'ordre, et relecture manuelle. |

Couplages par chemin de fichier, qui échouent en silence après un renommage :

- [.golangci.yml:161](.golangci.yml:161) — l'exclusion `SA6002` est ancrée sur le regex `internal/bigfft/(pool|pool_warming)\.go`. Fusionner ces fichiers réactive le finding sur une exception documentée (ADR-0007). C'est la **seule** exclusion ancrée par nom de fichier ; les quatre autres sont ancrées par texte et survivent.
- [Makefile:232](Makefile:232) — baseline vide après renommage du benchmark (§7.2).
- [Makefile:22-25](Makefile:22) — `-ldflags -X` vers un symbole inexistant : binaire figé sur `dev`, sans avertissement.
- `cmd/fibcalc/default.pgo` — ~82 symboles référencés par chemin complet ; PGO ignore silencieusement ceux qui ne résolvent plus.
- [check.sh:57-59](scripts/check.sh:57) — l'étape GMP est limitée à `./internal/fibonacci/`. Déplacer `calculator_gmp.go` hors de ce package rend l'étape **vert en ne testant plus rien**.
- `testdata/` doit rester adjacent au package de test `fibonacci` ; `cmd/generate-golden/main.go:31` code en dur `internal/fibonacci/testdata`.

---

## 12. Journal des déviations et des mutations

*(vide — à remplir au fil de l'exécution)*

Format attendu, une entrée par déviation :

```
[AAAA-MM-JJ] vague N, package P
Écart au plan : ...
Raison : ...
Alternative écartée : ...
```

Format attendu, une entrée par mutation de phase B :

```
[AAAA-MM-JJ] package P, mutation M/12 — famille : correction | frontière | concurrence
Patch : ...
Test rouge produit : func TestXxx (fichier:ligne)
```
