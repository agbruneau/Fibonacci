# Rapport d'Évaluation Académique — FibCalc (`github.com/agbru/fibcalc`)

**Dépôt audité** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo`
**Commit** : `6b9de67` (branche `main`)
**Date du rapport** : 2026-05-20
**Méthode** : Inspection solo (pass 1) + dispatch de 5 agents parallèles spécialisés sur la grille à 100 points (pass 2)
**Évaluateur** : Professeur titulaire (chaire de Génie Logiciel et d'Architecture des Systèmes Complexes)

---

## 1. Synthèse de l'Évaluation et Note Globale

**Note : 82 / 100**

La revue parallélisée révise le premier verdict à la baisse de 4 points : les agents ont surfacé des couplages d'imports remontants (`fibonacci/threshold → config`, `config → fibonacci/memory`, `errors → format`), des data races confirmées (`DynamicThresholdManager` lignes 194-202 hors verrou alors que les getters tiennent `RLock`, lectures hot path de `TransformCache.config.Enabled` sur quatre call-sites), et une dette de gouvernance documentaire (`EVALUATION.md` auto-évalué à 98/100 dans le dépôt sans revue tierce, badge couverture statique, drift C4 `sysmon → metrics/system`) que la première inspection avait sous-estimés. Le projet reste un travail d'ingénierie de très haute facture pour son périmètre (cinq couches de validation, race detector tri-OS avec CGO Windows, PGO versionné réellement intégré, invariants mémoire documentés et gardés par tests nommés), mais la consolidation finale de la concurrence à granularité fine et la résorption des dettes architecturales transverses ne sont pas terminées. C'est un prototype académique abouti, pas un système prêt pour usage *library-style* multi-tenant.

---

## 2. Ventilation Détaillée par Critère

### 2.1 Architecture et Conception Système — **19 / 25**

**Forces majeures**
- Decorator + ISP rigoureux : `FibCalculator` enveloppe `CoreCalculator` (`internal/fibonacci/calculator.go:74-95`), interfaces étroites `Multiplier` puis extension `DoublingStepExecutor` (`internal/fibonacci/strategy.go:29-84`).
- Pool d'état + arena scellés par protocole panic-safe documenté, ordre `checkLimit → clearStateAliases → Put` gardé par un test de régression nommé (`internal/fibonacci/fastdoubling.go:393-398`).
- `FFTContext` opt-in expose la trajectoire de sortie des globaux (`internal/bigfft/context.go:55-94`).

**Pénalités (−6)**
- **Triangle d'imports** : `internal/fibonacci/threshold → internal/config` (`internal/fibonacci/threshold/manager.go:11`), puis `internal/config → internal/fibonacci/memory` (`internal/config/config.go:12`, `internal/config/validator.go:7`, `internal/config/threshold_tuning.go:6`). Trois packages bas-niveau s'entrelacent — violation directe de la hiérarchie revendiquée dans `Claude.md` (« cmd → app → orchestration → fibonacci/bigfft → config/errors »). Le commentaire `threshold_tuning.go:74-78` admet le cycle latent sans le résoudre.
- **`internal/errors` importe `internal/format`** (`internal/errors/errors.go:11, 143`) pour `FormatBytes`. Le package d'erreurs assume une responsabilité de présentation (violation SRP).
- **Data race confirmée sur `DynamicThresholdManager`** : `ShouldAdjust` mute trois champs hors verrou (`internal/fibonacci/threshold/manager.go:194-202`) alors que `GetThresholds`/`GetFFTThreshold`/`GetParallelThreshold`/`GetStats` les lisent sous `RLock` (`internal/fibonacci/threshold/manager.go:142-159, 261-272`). Le commentaire A-18 (ligne 167-174) reconnaît la fragilité ; rien dans le type-system n'empêche un futur partage d'instance.
- **Lectures non synchronisées de `TransformCache.config`** sur quatre sites hot path : `internal/bigfft/fft_cache.go:207, 301, 444`, `internal/bigfft/context.go:299` — alors que `SetTransformCacheConfig:113-124` écrit sous `Lock`. `Config()` (ligne 71) acquiert correctement la RLock, ce qui démontre que les autres call-sites sont défaillants.
- **Trois globaux mutables `bigfft`** : `fftThreshold` (`internal/bigfft/fft.go:35`), `ParallelFFTRecursionThreshold` et `MaxParallelFFTDepth` (`internal/bigfft/fft_recursion.go:28, 33`) sans `atomic` ni verrou, exportés et donc mutables runtime par n'importe quel consommateur. Contredit la directive 4 de `Claude.md`.
- **Couplage `tui → fibonacci` direct** : `internal/tui/model.go:11`, `internal/tui/commands.go:13` importent `internal/fibonacci` et instancient `[]fibonacci.Calculator` (`internal/tui/model.go:48`) au lieu de passer par `orchestration`. La séparation UI/calcul existe sur le papier mais est court-circuitée.

### 2.2 Qualité du Code et Robustesse — **21 / 25**

**Forces**
- Wrapping `%w` systématique (44 occurrences), aucun `fmt.Errorf` non-recover sans verbe `%w`.
- `go vet ./...` clean ; 24 linters activés ; complexité gatée (cyclo 15, cognitive 30, funlen 100/50).
- Validation env exhaustive avec erreur explicite plutôt que fallback silencieux (`internal/config/env.go:111-114, 218-222`) — justifiée contre risque OOM.
- Persistance de profil atomique cross-platform avec retry borné Windows (`internal/calibration/profile.go:117-187`).

**Pénalités (−4)**
- **`recover()` global au sommet de `bigfft.Mul/Sqr/MulTo/SqrTo`** (`internal/bigfft/fft.go:41-45, 57-61, 84-88, 97-101`) capture *toute* panic incluant les post-conditions algorithmiques légitimes (`internal/bigfft/fermat.go:201, 226, 262, 281` : `len(z) > 2n+1`, `unexpected carry after normalization`). Un bug authentique dans la réduction modulaire est dégradé en `error` opaque indistinguable d'une erreur d'arité.
- **Wrappers `*Safe` orphelins** : `MulSafe`, `SqrSafe`, `ShiftSafe`, `AddSafe`, `SubSafe` (`internal/bigfft/fermat.go:291-339`) n'ont aucun appelant en production (vérifié par grep). 48 LOC de remédiation R3.8 décorative.
- **Shadowing du built-in `cap`** à trois sites : `internal/bigfft/pool.go:242, 330, 418`. Incohérent avec `internal/bigfft/pool.go:143` qui utilise correctement `c := cap(slice)`. `govet shadow` désactivé (`.golangci.yml:52`) masque la régression.
- **`recover()` muet** dans `ProgressSubject.Freeze` (`internal/progress/observer.go:142-150`) : aucun log, aucun compteur. Un observer cassé est silencieusement masqué.
- **`gosec G304` désactivé globalement** (`.golangci.yml:136-143`) sans `filepath.Clean`/`Abs` sur `--calibration-profile` ni `--output`. Modèle CLI mono-utilisateur défendable, mais défense en profondeur abandonnée.

**Note sécurité** : aucun risque exploitable dans `internal/cli/completion/` (les valeurs viennent d'un `flagRegistry` statique, pas de donnée runtime utilisateur dans le script généré). Aucun `exec.Command`. Aucun secret injecté. Le risque est latent (si un futur contributeur ajoute des `Values` dérivés d'env var, l'apostrophe dans `internal/cli/completion/fish.go:91` et `internal/cli/completion/powershell.go:101` casserait sans fonction d'escape centralisée).

### 2.3 Stratégie de Validation et de Test — **17 / 20**

**Inventaire**
- 107 fichiers `_test.go` pour 134 sources non-test → ratio **0.80**.
- **924** invocations `t.Parallel()` sur ~693 fonctions `Test*`/`Fuzz*` (≈ 1.3 par fonction).
- Cinq layers : unitaires (~95), golden (`internal/fibonacci/fibonacci_golden_test.go:19` — 23 entrées, plafond F(10000)), property-based gopter (Cassini, récurrence, doublement, GCD identité `GCD(F(m), F(n))=F(GCD(m,n))` à `internal/fibonacci/fibonacci_property_test.go:190`), fuzz cross-algorithme (`internal/fibonacci/fibonacci_fuzz_test.go:13, 69`), E2E (`test/e2e/cli_e2e_test.go`).
- Oracle indépendant hors-module : `cmd/generate-golden/main_test.go:13, 126` (identité de récurrence pour F(800k) en régime FFT).

**Pénalités (−3)**
- **Gate `>5%` perf déclaratif mais non implémenté** : `.github/workflows/ci.yml:84-90` déclare explicitement `continue-on-error: true` et reconnaît dans son commentaire « A real >5% regression gate needs a versioned baseline ». La directive 1 du `Claude.md` n'est tenue que par procédure humaine vs baselines `docs/audits/bench-A10-*.txt`.
- **Deux packages sans tests** : `internal/cli/completion/` (6 fichiers, ~555 LOC, **zéro test** — exactement le risque sécurité latent flaggé dans `Claude.md`) et `internal/tui/component/`.
- **Corpus fuzz seed minuscule** : 3 fichiers par cible dans `testdata/fuzz/Fuzz*/`. Bornes basses (`n > 50000 { return }` à `internal/fibonacci/fibonacci_fuzz_test.go:30`) qui **n'exercent pas le régime FFT lourd** (≥ 500k bits) où vit la complexité du projet.
- **Aucun fuzz ciblant `bigfft` directement** (`fermat.Mul`, `fermat.Shift`, `Poly.Transform` testés uniquement par tables et via Fibonacci indirect).
- **Aucun test de régression sur les 13 panics non-test** — `grep "defer.*recover" internal/bigfft/*_test.go` est vide.

### 2.4 Documentation et Expérience Développeur — **12 / 15**

**Forces**
- README.md exhaustif (12 sections, badges, quickstart, tableau flags, exemples).
- 20/21 packages internes ont un `doc.go` ; trois sont substantiels (`internal/bigfft/doc.go`, `internal/fibonacci/doc.go`, `internal/calibration/doc.go`).
- Architecture C4 complète : system-context, container, component, dependency-graph + 6 diagrammes de flux.
- Makefile très complet (build/PGO/test/coverage/benchmark/lint/security/cross-compile Linux/Windows/macOS × amd64/arm64).
- Zéro `TODO/FIXME/XXX/HACK` dans le code Go.
- `Claude.md` opérationnel et non décoratif : table d'invariants gardés par test nommé, modules sensibles, 8 directives projet exécutables.

**Pénalités (−3)**
- **Aucune containerisation ni devcontainer** : pas de `Dockerfile`, pas de `.devcontainer/`, pas de compose. Pour un projet où `-race` exige CGO/gcc et où le build PGO multi-OS est central, c'est une lacune de reproductibilité majeure.
- **Drift C4** : le package a été renommé `sysmon → internal/metrics/system` (cf. `CHANGELOG.md:58`, `README.md:267`) mais le nœud `sysmon` reste dans `docs/architecture/dependency-graph.mermaid:34, 81` et `docs/architecture/container-diagram.mermaid:16`. Les diagrammes ne suivent plus le code.
- **`EVALUATION.md` orphelin et auto-évalué 98/100** (`EVALUATION.md:6`) tracké à la racine sans lien depuis README/ARCH/CHANGELOG/CONTRIBUTING. Document de gouvernance ambigu : auto-publié sans revue tierce, métriques LOC divergentes vs `Claude.md:14` (31 386 vs ~35 500).
- **Race detector Windows : contradiction trans-document**. `Claude.md:125` affirme « race indisponible sous Windows sans gcc » ; `CHANGELOG.md:44` annonce « `-race` runs on Windows too » via CGO MinGW en CI. La doc n'a pas été synchronisée avec la CI.
- **Décompte de packages incohérent** : `Claude.md:14` dit « 23 packages (21 internes + 2 sous cmd/) » ; `ARCH.md:12` dit « 24 (22 internal + cmd × 2 + test/e2e) » ; réel via `find internal -type d` = 21.
- **Badge de couverture statique faux** : `README.md:6` hardcode `87.5%`, non câblé sur `coverage.yml`.
- **`ruvector.db` (1,5 Mo)** physiquement présent à la racine, ni tracké ni gitignoré (`git check-ignore` vide).
- **Quatre `doc.go` purement formels** : `internal/cli/doc.go`, `internal/config/doc.go`, `internal/orchestration/doc.go`, `internal/fibonacci/fibonaccitest/doc.go` — 4-5 lignes chacun, sans invariant ni exemple.

### 2.5 Complexité Technique et Innovation — **13 / 15**

**Forces** (correction mathématique, innovations rares)
- Identités Fast Doubling rigoureusement implémentées avec dérivation matricielle documentée (`internal/fibonacci/fastdoubling.go:25-46`). Route `Sqr` séparée sauvant ~33% du coût FFT (`internal/bigfft/fft.go:80-94`).
- **État + arena unifiés** avec invariant non-trivial encodé : `finalizeStateRelease` détache les aliases avant Put dans tous les chemins (`internal/fibonacci/fastdoubling.go:401-428`) — innovation correcte contre les pièges classiques de `sync.Pool`.
- **Soft GOMEMLIMIT comme garde OOM** pendant GC désactivé (`internal/fibonacci/memory/gc_control.go:101-106`) — garde-fou réel, pas gadget.
- **Heuristique CPU dans la clé d'invalidation de profil** (`internal/calibration/profile.go:196-224`), branche `IsStale` distincte (`internal/calibration/calibration.go:302-322`).
- **Modular Fast Doubling** O(log m) mémoire pour F(10^18) mod m (`internal/fibonacci/modular.go:9-66`).
- **PGO versionné réellement intégré** : `cmd/fibcalc/default.pgo` checked-in, workflow `make pgo-rebuild`.
- **Deep-copy à la release** avec justification chiffrée explicite : « ~850 KB memcpy for F(10M), <0.01% of runtime » (`internal/fibonacci/doubling_framework.go:230-237`).

**Pénalité (−2) : sur-ingénierie auto-reconnue**
- **`internal/calibration/` = 1 686 LOC non-test** vs `internal/fibonacci/` (cœur) = 4 277 LOC → ratio **39%**. Pour un CLI académique, < 15% serait attendu. L'appareil `FastStrategy → CompleteStrategy → EscalationConfidenceThreshold` (`internal/calibration/strategy.go:25-51`) reproduit un mini-framework de stratégie là où un seul code path suffirait. L'admission « the most over-engineered Fibonacci calculator » (`README.md:11`) est factuelle.
- **`DynamicThresholdManager` redondant avec la calibration** : la calibration produit déjà `OptimalFFTThreshold`/`OptimalParallelThreshold` (`internal/calibration/profile.go:30-32`) ; ajouter en plus un ajustement intra-calcul avec hystérésis et fenêtre de 20 métriques pour quelques dizaines de doublings est probablement non rentable. Aucun benchmark dans `docs/audits/` ne mesure son gain.
- **`docs/audits/` ne contient que 3 fichiers txt** pour A-10 (15 lignes au total) alors que la directive 1 le décrit comme « baselines benchmark » — directive non actionnable.
- **Coexistence globaux + `FFTContext`** : la trajectoire de migration est entamée mais les deux mécanismes cohabitent (`internal/bigfft/context.go` 439 LOC + globaux maintenus), ce qui est plus risqué que l'un ou l'autre seul.

---

## 3. Critiques Techniques Ciblées

### 3.1 Concurrence à granularité fine incomplète — *impact maintenabilité × scalabilité*

Trois data races convergentes documentées par les agents : (1) `DynamicThresholdManager.ShouldAdjust` mute hors verrou (`internal/fibonacci/threshold/manager.go:194-202`) ce que les getters lisent sous RLock — invariant « single-writer » non *enforceable* par le type-system ; (2) `TransformCache.config.Enabled/MinBitLen` lu sur quatre call-sites hot path (`internal/bigfft/fft_cache.go:207, 301, 444`, `internal/bigfft/context.go:299`) sans synchroniser avec `SetTransformCacheConfig` qui prend `Lock` ; (3) globaux `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` exportés mutables sans `atomic`. **Impact** : le projet est aujourd'hui *single-threaded-process-safe* mais ferme la porte à un usage *library* où plusieurs `Mul()` concurrents s'exécuteraient avec des configurations différentes. `go test -race` finira par flagger ces accès sous charge concurrentielle hostile. La trajectoire (`FFTContext`) existe mais n'est pas terminée.

### 3.2 `recover()` global masque les violations d'invariants algorithmiques — *impact correction × régression*

`internal/bigfft/fft.go:41-101` installe un `recover()` indistinct dans Mul/Sqr/MulTo/SqrTo. Le commentaire de `fermat.go:21-28` distingue théoriquement les post-conditions internes (à laisser propager) des pré-conditions d'arité (à transformer en `error`), et 48 LOC de wrappers `*Safe` (`internal/bigfft/fermat.go:291-339`) appliquent correctement cette distinction — mais aucun call-site de production ne les utilise. La conséquence opérationnelle : un futur bug dans la réduction modulaire `fermat.Mul` ou `fermat.Sqr` (sites de `panic("len(z) > 2n+1")`, `panic("unexpected carry")`) ressort comme un `error` opaque indistingable d'une erreur d'arité, et passe silencieusement tout test qui n'inspecte que `err != nil`. C'est une voie de régression silencieuse au cœur algorithmique du projet.

### 3.3 Dette architecturale transverse non close — *impact maintenabilité × évolutivité*

Quatre symptômes du même phénomène : (a) triangle d'imports `threshold → config → memory` documenté comme « cycle latent » sans résolution ; (b) `errors → format` violation SRP ; (c) `tui → fibonacci` court-circuitant `orchestration` ; (d) `globalFactory` package-level coexistant avec la `DefaultFactory` injectable. Chacun pris isolément est mineur ; ensemble ils témoignent d'un effort Clean Architecture *incomplet* — la couche logique existe, l'injection existe, mais la fermeture finale (suppression des points d'accès parallèles, résolution des cycles latents) n'a pas été appliquée. **Impact à long terme** : tout futur consommateur qui veut extraire `internal/fibonacci/` comme bibliothèque indépendante doit dépiler ces couplages avant publication.

---

## 4. Plan de Bonification et Améliorations

### Priorité HAUTE — bloquant pour usage *library-style* ou release v1.0

1. **Clore les data races confirmées.** Convertir `currentFFTThreshold`/`currentParallelThreshold`/`lastAdjustment` en `atomic.Int64` / `atomic.Pointer[time.Time]` (`internal/fibonacci/threshold/manager.go:194-202`). Protéger les quatre lectures de `TransformCache.config` sous `RLock` (ou dupliquer `Enabled` dans un `atomic.Bool`). Synchroniser `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` — soit via `atomic`, soit en les portant dans `FFTContext` (trajectoire déjà ouverte).
2. **Restaurer la propagation des post-conditions FFT.** Supprimer le `recover()` global de `internal/bigfft/fft.go:41-101` ou le restreindre à un set d'erreurs sentinelles. Exposer les `*Safe` variants comme API publique ; sinon supprimer les 48 LOC orphelines.
3. **Briser l'import remontant `internal/fibonacci/threshold → internal/config`** (`internal/fibonacci/threshold/manager.go:11`) en injectant `ThresholdTuningProfile` à la construction plutôt qu'en l'important.
4. **Activer un gate `benchstat ≥5%` bloquant** en CI (`.github/workflows/ci.yml:84` — retirer `continue-on-error: true`). Baseline versionnée dans `docs/audits/bench-baseline.txt`.
5. **Tests adversariaux sur `internal/cli/completion/`** : identifiants contenant `$(...)`, backticks, `;`, espaces, guillemets ; golden output par shell. Risque sécurité explicitement flaggé sans aucun test.

### Priorité MOYENNE

6. **Décider explicitement** entre `DynamicThresholdManager` et `internal/calibration/` — pas les deux. Si DTM reste, exiger `docs/audits/bench-dtm-{on,off}.txt` prouvant > 5% de gain ; sinon supprimer (~283 LOC + invariant A-18 fragile).
7. **Fermer le risque résiduel cache FFT** : refcount sur `cacheEntry.backing` ou deep-copy à la sortie du `Get` (`internal/bigfft/fft_cache.go:316-377`). Le contrat « MUST NOT modify » est non *enforceable*.
8. **Étendre golden + fuzz** dans le régime FFT : ≥ 5 entrées golden au-delà de F(50 000), fuzz ciblé `FuzzFermatMul`/`FuzzShift`/`FuzzPolyTransform` avec corpus seed dérivé de `fftSizeThreshold[]`.
9. **Sortir `format` de `errors`** (`internal/errors/errors.go:11, 143`) : retourner une struct sérialisable, déléguer le formatage au présentateur.
10. **Découpler `internal/tui` de `internal/fibonacci`** (`internal/tui/model.go:11`, `internal/tui/commands.go:13`).
11. **Ajouter `Dockerfile` multi-stage + `.devcontainer/devcontainer.json`.** Résout le trou Windows-sans-gcc et formalise la matrice CGO.
12. **Synchroniser les C4 diagrams** : remplacer `sysmon → internal/metrics/system` dans `docs/architecture/dependency-graph.mermaid:34, 81` et `docs/architecture/container-diagram.mermaid:16`.
13. **Statuer sur `EVALUATION.md`** : retirer ou déplacer vers `docs/external-reviews/` avec en-tête « auto-évaluation non revue par tiers ». Une note 98/100 auto-publiée à la racine sans contre-expertise est un signal négatif de gouvernance.
14. **Logger ou compter les `recover()`** muets (`internal/progress/observer.go:142-150`).

### Priorité BASSE

15. **Tests de panic ciblés** pour les 13 sites non-test (priorité `fermat.go`).
16. **Renommer `cap := cap(...)` en `c := cap(...)`** à `internal/bigfft/pool.go:242, 330, 418`.
17. **Activer `govet shadow`** dans `.golangci.yml:52` au moins en mode warning.
18. **Remplacer le badge couverture statique** (`README.md:6`) par un badge dynamique alimenté par `coverage.yml`.
19. **Réconcilier le décompte de packages** entre `Claude.md:14` et `ARCH.md:12`, l'auto-générer via `go list ./... | wc -l`.
20. **Renommer `Claude.md` → `CLAUDE.md`** dans Git (`git mv`) et ajouter `* text=auto eol=lf` dans `.gitattributes`.

---

## 5. Notes de Méthode

L'audit a été conduit en deux passes :

1. **Pass 1 (solo)** : inspection initiale par lecture directe des fichiers critiques (`fastdoubling.go`, `fft.go`, `fft_cache.go`, `fermat.go`, `manager.go`, `gc_control.go`, `orchestrator.go`, `pool.go`, README, Claude.md, .golangci.yml, ci.yml, coverage.yml). Verdict : 86/100.

2. **Pass 2 (parallèle, 5 agents spécialisés)** : un agent par critère de la grille, instruction de citations `file:line` strictes et notation chiffrée indépendante. Verdicts agents : Architecture 19/25, Qualité 21/25, Tests 17/20, Doc/DevEx 12/15, Complexité 13/15. **Total consolidé : 82/100**.

La revue parallèle a abaissé la note de 4 points en surfaçant trois familles de constats que la première passe avait sous-estimés :
- **Data races concrètes** hors verrou détectables au race detector sous charge concurrente (DTM, TransformCache.config).
- **Drift de cohérence trans-documents** (C4 vs code, badge couverture, race detector Windows entre Claude.md et CHANGELOG, EVALUATION.md orphelin).
- **Couplages d'imports remontants** encodant une dette architecturale latente (`threshold → config`, `errors → format`, `tui → fibonacci`).

Coût : ~162 min CPU agent cumulées pour gagner ~50 citations `file:line` précises supplémentaires et abaisser la marge d'erreur sur la note.

Rapport établi sans présomption sur l'intention de l'auteur ; uniquement sur les artéfacts visibles au commit `6b9de67`.

---

## 6. Tableau Récapitulatif

| Critère | Note | Max | Pondération |
|---|---:|---:|---|
| Architecture et Conception Système | 19 | 25 | Hiérarchie respectée, mais data races et couplages d'imports remontants |
| Qualité du Code et Robustesse | 21 | 25 | Idiomatique Go, mais `recover()` global et wrappers `*Safe` orphelins |
| Stratégie de Validation et de Test | 17 | 20 | 5 layers indépendants, mais gate perf non bloquant et angles morts |
| Documentation et DevEx | 12 | 15 | C4 + Makefile complets, mais drift et absence de containerisation |
| Complexité Technique et Innovation | 13 | 15 | Sophistication réelle, mais sur-ingénierie de la calibration auto-reconnue |
| **TOTAL** | **82** | **100** | **Prototype académique abouti, dette concurrente non close** |
