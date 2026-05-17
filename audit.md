# Audit exhaustif — FibGo (`github.com/agbru/fibcalc`)

> Audit **lecture seule**. Aucun fichier de code, test, build ou configuration n'a été modifié. Seul fichier produit : `audit.md` (+ `audit-prompt.md`, le prompt l'ayant généré).
> Date : 2026-05-17 · Périmètre : 239 fichiers `.go` / ~35 500 LOC / 24 packages · Révision : `main@5577529`.

---

## 1. Résumé exécutif

**Verdict global : codebase mature et saine sur le fond ; les 5 bugs latents historiques (R1.1–R1.5 / L1–L5) sont TOUS corrigés ou infirmés, avec gardes de régression.** `go vet ./...` est propre (exit 0). L'étanchéité Clean Architecture est respectée, l'orchestration concurrente est correcte, le pooling critique est protégé par tests. Le risque résiduel n'est **pas** dans les zones signalées par `Claude.md` (toutes traitées par les vagues de refactoring déjà mergées) mais dans **trois angles morts non documentés** : (a) une race silencieuse dans le cache FFT sous pression+concurrence, (b) des globaux mutables lus sur le hot path FFT sans synchronisation, (c) une **dérive documentaire critique du fichier d'instructions normatif `Claude.md`** qui décrit un état pré-refactoring.

**Top 5 risques** :

1. **A-01 [Critique]** — Cache FFT : le recyclage du buffer `backing` en éviction (`putByKey`) zéroie puis réécrit un tableau encore aliasé par un `PolValues` retourné par `getByKey` → résultat FFT silencieusement corrompu sous concurrence + pression cache. Le commentaire « immutable after insertion » est faux sur le chemin d'éviction.
2. **A-04 [Haute]** — `Claude.md` (instructions projet impératives) décrit un état antérieur au refactoring : « CI absente » alors que `ci.yml`+`coverage.yml` existent ; 5 bugs « non corrigés » tous corrigés ; références `ultrareview.md`/`ultrareviewplan.md`/`docs/audits/` **inexistantes** ; package `parallel` dit « quasi-mort » alors qu'il est sur le hot path. Un agent autonome le suivant à la lettre **défait des correctifs livrés**.
3. **A-02 [Haute]** — Data race sur `tc.config` (lu sans verrou dans `Get`/`Put` sur le hot path, muté sous `Lock` par `SetTransformCacheConfig`).
4. **A-03 [Haute]** — Globaux mutables `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` lus à chaque nœud de récursion FFT sans synchronisation, mutés sans verrou par les setters de calibration.
5. **A-08 [Moyenne]** — `--last-digits <0` ignoré silencieusement : le mode partiel O(K) demandé bascule en calcul complet O(mémoire) sans message → risque OOM contraire à l'intention utilisateur.

**Posture mise en production** : **acceptable pour un prototype académique mono-utilisateur**, sous réserve de A-01 (corruption silencieuse possible — bloquant si le parallélisme FFT + cache sont activés sur charges longues) et A-02/A-03 (faux si recalibration à chaud concurrente). Sans recalibration concurrente ni multi-tenant, les races A-02/A-03 ne se déclenchent pas (le projet reste mono-calcul par invocation). Aucun défaut de correction *actif* en usage nominal séquentiel n'a été trouvé.

**Dette agrégée** : 23 constats — 1 Critique, 3 Haute, 11 Moyenne, 8 Basse. Effort cumulé estimé ≈ 18–25 j·dev, dominé par le durcissement concurrence du package `bigfft` et la remise à niveau documentaire.

---

## 2. Méthodologie & couverture

- **Approche** : passe package par package via 5 agents d'audit lecture seule (clusters : cœur `fibonacci`, `bigfft`, `app/orchestration/config/errors/parallel`, `cli/tui/ui/progress/format`, `calibration/metrics/CI`), puis vérification personnelle des constats Critique et des bugs latents (lecture directe `fastdoubling.go`, `gc_control.go`, `fft_cache.go`, `pool.go`).
- **Outils non mutants** : `go vet ./...` (exit 0, aucun avertissement) ; lecture/grep ; `git ls-files`. `golangci-lint`/`gosec` non exécutés isolément (couverts en CI via `golangci-lint-action`, gosec inclus) — résultats inférés de `.golangci.yml` et corrélés au code, non recopiés bruts.
- **Limites** : les fichiers `*_test.go` ont été échantillonnés (grep + lecture ciblée), pas lus ligne à ligne ; les évaluations D8 sont indicatives. Les fichiers assembleur (`arith_amd64.go`, SIMD) n'ont pas été audités sur le fond. Les emplacements ligne de `Claude.md` ne correspondant plus au code, les références y sont traitées comme historiques.
- **Source de vérité** : le code. `Claude.md`/`README`/`docs/` sont confrontés au code ; les écarts sont des constats à part entière (§6.4).

---

## 3. Matrice de risques

| ID | Titre | Dim. | Sévérité | Effort | Package |
|----|-------|------|----------|--------|---------|
| A-01 | Recyclage `backing` en éviction zéroie un buffer aliasé | D2/D3 | **Critique** | M | bigfft |
| A-02 | `tc.config` lu sans verrou sur hot path | D2 | Haute | S | bigfft |
| A-03 | Globaux FFT mutables non synchronisés (hot path) | D2/D5 | Haute | M | bigfft |
| A-04 | Dérive documentaire normative `Claude.md` | D5/D9 | Haute | M | (doc) |
| A-05 | `putByKey` sans assertion d'invariant + clé FNV partielle | D1 | Moyenne | S | bigfft |
| A-06 | `recover()` global masque les post-conditions `fermat` | D7 | Moyenne | M | bigfft |
| A-07 | `NTransform`/`InvNTransform` paniquent (API publique) | D7 | Moyenne | S | bigfft |
| A-08 | `LastDigits<0` ignoré → bascule O(mémoire) silencieuse | D1/D9 | Moyenne | S | config/orchestration |
| A-09 | Overrides d'env mal formés avalés silencieusement | D7/D9 | Moyenne | S | config |
| A-10 | `CalcTotalWork` overflow → progression figée grands N | D1 | Moyenne | M | progress |
| A-11 | Persistance profil non atomique (corruption inter-process) | D2 | Moyenne | M | calibration |
| A-12 | CI `golangci-lint version: latest` non reproductible | D9 | Moyenne | S | CI |
| A-13 | CI sans garde benchmark / seuil couverture / PR coverage | D9 | Moyenne | M | CI |
| A-14 | `-race` absent sous Windows (plateforme primaire) | D2/D9 | Moyenne | S | CI |
| A-15 | test/e2e : codes de sortie permissifs, pas de `Short()` | D8 | Moyenne | M | test/e2e |
| A-16 | `MaxPooledBitLen` : commentaire ≠ valeur, bits vs words | D3/D5 | Basse | S | fibonacci |
| A-17 | Aliasing arena chemin FFT parallèle sans garde `-race` | D2/D3 | Basse | M | fibonacci |
| A-18 | `threshold.ShouldAdjust` write non synchronisé (single-writer) | D2 | Basse | S | threshold |
| A-19 | Surface d'API morte `progress` + messages TUI inexploités | D5 | Basse | S | progress/tui |
| A-20 | Oracle `calculateSmall`↔`fibBig` sans test d'équivalence | D8 | Basse | M | fibonacci |
| A-21 | Métriques/arbre packages obsolètes (24 linters ≠ « 22 ») | D5 | Basse | S | (doc) |
| A-22 | Allocations ANSI répétées + `updateContent()` quadratique | D3 | Basse | M | tui |
| A-23 | `docs/CALIBRATION.md` périmé (struct, Strategy, env) | D5 | Basse | S | (doc) |

---

## 4. Vérification des bugs latents connus (L1–L5 / R1.1–R1.5)

**Conclusion : les 5 sont neutralisés. `Claude.md:80-90` (« Ces bugs n'ont pas encore été corrigés ») est factuellement faux pour le code actuel.**

| Réf. | Statut | Preuve (vérifiée) |
|------|--------|-------------------|
| **L1** `clearStateAliases` contournable (`overLimit`) | **CORRIGÉ** *(confirmé, lecture directe)* | `fastdoubling.go:401-428` — `finalizeStateRelease` est le chemin unique ; `clearStateAliases(s)` appelé **inconditionnellement** l.411, **avant** le `return` `overLimit` (l.425) et avant `statePool.Put` (l.427). Ordre commenté + garde `TestReleaseState_OverLimit_AliasesCleared`. Use-after-free **infirmé**. |
| **L2** GC désactivé persistant si panic | **CORRIGÉ** *(confirmé, lecture directe)* | `gc_control.go:84-88` — `WithGC` fait `gc.Begin(); defer gc.End(); return fn()` : restauration garantie même sur panic. `Begin`/`End` directs marqués `Deprecated` (l.90-93) avec avertissement de fuite. |
| **L3** `IsStale` jamais invoqué | **CORRIGÉ** *(confirmé, agent + file:line)* | `calibration.go:302` : `profileStale := profileFresh && profile.IsStale(maxAge)` ; branche stale `:314-322` re-calibre via `CompleteStrategy` ; tests `profile_test.go:158-175`, `calibration_test.go:423-478`. |
| **L4** `releaseWordSlice` perd les buffers resizés | **CORRIGÉ** *(confirmé, lecture directe)* | `pool.go:136-155` — routage sur `cap` (non `len`), restauration de la capacité pleine avant `Put`, drop GC + `wordSlicePoolMissCount` pour observabilité ; design intentionnel documenté ; test `TestReleaseWordSliceResizedReturnsToBucket`. |
| **L5** `putByKey` alloue eager même en éviction | **INFIRMÉ** *(confirmé, lecture directe)* | `fft_cache.go:331-359` — `backing` reste `nil` ; salvage du premier backing évincé `cap>=wordCount` ; `make` l.358 **uniquement si `backing==nil`** après salvage. Pas d'allocation eager superflue. |

---

## 5. Constats par package (par couche)

### 5.1 `cmd/` (fibcalc, generate-golden) — sain

Étanchéité parfaite : `cmd/fibcalc` n'importe que `internal/app` ; `cmd/generate-golden` n'importe **aucun** `internal/` (oracle indépendant, dualité `fibBig`/`calculateSmall` intentionnelle P2-04). Aucun constat propre. Voir **A-20** (couplage par convention de l'oracle).

### 5.2 `internal/app`, `orchestration`, `errors`, `parallel` — sain, dette doc

Orchestration concurrente **correcte** : `errgroup.WithContext` + `WaitGroup` + `close(progressChan)` ordonné après `g.Wait()` + sémaphore borné `NumCPU` + vérification `ctx.Err()` post-acquisition. Aucune goroutine sans contrôle de cycle de vie. Wrapping `%w` cohérent, types d'erreurs structurés, `Unwrap()` correct, **aucun `panic(`** dans le périmètre. Le cycle potentiel `errors↔cli` est cassé proprement via l'interface `ColorProvider` (ISP). **Package `parallel` VIVANT** : `parallel.ErrorCollector` consommé à 4 sites de `internal/fibonacci/common.go` (l.114,149,225,282) — unique mécanisme first-error des exécutions parallèles ; seul `ErrorCollector.Reset()` est mort (aucun appelant non-test). → contredit `Claude.md` (A-04). Constats : **A-08**, **A-09**.

### 5.3 `internal/config` — validations incomplètes

`AppConfig.Validate()` (`config.go:99-120`) couvre `Timeout/Threshold/FFTThreshold/Algo` mais **ni** `LastDigits<0` (**A-08**), **ni** `StrassenThreshold<0`, **ni** le domaine fermé de `GCControl` (`auto|aggressive|disabled`), **ni** `Completion` (validé tardivement au dispatch). Overrides d'env : échec de parsing **avalé silencieusement** (`env.go:114-133`), `FIBCALC_N=abc` → défaut sans avertissement (**A-09**). Pas de borne haute sur `N` sans `--memory-limit` (acceptable pour un prototype, mais non documenté comme intentionnel).

### 5.4 `internal/fibonacci` (+ memory, threshold) — cœur sain post-refactoring

L1/L2 corrigés (§4). `threshold/manager.go` = **277 L** (≠ « 417 L » de `Claude.md`), responsabilités déléguées (`MetricsBuffer`, `ThresholdAnalyzer`) — pas un God-object. Réserve **A-18** : `ShouldAdjust` mute `currentFFTThreshold`/`lastAdjustment` sans `Lock` (invariant single-writer correct mais non documenté ; lu sous `RLock` par les getters). **A-16** : `MaxPooledBitLen=50_000_000` mais commentaire « 100M bits », et confusion d'unités avec `maxArenaPoolWords` (words) — footgun de réglage perf. **A-17** : sûreté de l'aliasing arena sur le chemin FFT parallèle (réallocation hors arena) non couverte par un test `-race` dédié. **A-20** : équivalence `calculateSmall`↔`fibBig` protégée par commentaire seul.

### 5.5 `internal/bigfft` — concentration du risque résiduel

Package le plus dense (~7900 LOC, ~25 globaux). L4 corrigé / L5 infirmé (§4). Constats : **A-01 (Critique)**, **A-02**, **A-03**, **A-05**, **A-06**, **A-07**.

- **A-01 [Critique] — `fft_cache.go:345-365` vs `:244-271`** *(confirmé, lecture directe)*. `getByKey` retourne `pv := PolValues{Values: entry.values}` **partageant le backing** du cache (commentaire l.217 explicite ; RUnlock l.253 ; le consommateur lit `pv` hors verrou ensuite). Concurremment, `putByKey` (sous `Lock`) peut **évincer cette entrée**, salvager son `entry.backing`, le **zéroïer** (l.350-352) puis le **réécrire** (`copy` l.364). → race lecture (FFT en cours sur `pv`) / écriture (recyclage) ⇒ **résultat FFT corrompu silencieusement**. Le commentaire l.228-230 « Cache entries are immutable after insertion » est **faux en chemin d'éviction**. *Impact* : faux résultat Fibonacci non détecté. *Condition* : cache à `MaxEntries` + hit concurrent sur l'entrée précisément évincée pendant que son `pv` est encore consommé (parallélisme FFT, défaut sur grands N — cible du projet). *Reco (non appliquée)* : refcount/epoch atomique sur `cacheEntry`, recyclage du backing seulement si plus aucune référence ; ou copie défensive (`Clone()`) en sortie de `getByKey` au prix d'une alloc (annule une partie du gain R1.5) ; ou interdire le recyclage. Trade-off central perf-cache vs sûreté mémoire ; alternative : refcount (préserve l'optim). Condition de renversement : si le parallélisme FFT est désactivé par défaut sur cette charge, le risque tombe à Basse.
- **A-02 [Haute] — `fft_cache.go:207,301`** *(confirmé)*. `Get`/`Put` lisent `tc.config.Enabled/MinBitLen/MaxEntries` **sans verrou** ; `SetTransformCacheConfig` mute `tc.config` sous `Lock`. Data race si recalibration de cache concurrente à des opérations FFT. *Reco* : `atomic.Bool`/`atomic.Int64` pour `Enabled`/`MinBitLen`/`MaxEntries`, ou contrat dur « config figée avant usage » asserté en debug.
- **A-03 [Haute] — `fft.go:35`, `fft_recursion.go:28,33`** *(confirmé, agent)*. `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` lus à chaque nœud de récursion par N goroutines, mutés par `SetFFTParallelismConfig` sans verrou/atomique. *Reco* : `atomic.Uint64` ou injection via `FFTContext` (trajectoire R3.7, conforme à la directive « pas de nouveaux globals dans bigfft »).
- **A-05 [Moyenne] — `fft_cache.go:316-377`** *(confirmé, latent)*. `putByKey` ne vérifie pas `len(pv.Values[i]) == n+1` ; `copy` tronque silencieusement une entrée mal formée. Clé FNV-1a 64 bits n'encodant pas la forme → collision (proba faible) renverrait un transform pour d'autres données. Non déclenché (producteurs internes respectent l'invariant). *Reco* : assertion défensive en tête (drop du cache, jamais corruption).
- **A-06 [Moyenne] — `fft.go:42-45` / `fermat.go`** *(confirmé)*. Le `recover()` de frontière capture **toutes** les panics, y compris les **post-conditions internes** de `fermat.go` (`panic("len(z) > 2n+1")`) — masquées en erreur opaque, exactement ce que l'intention documentée R3.8 prétend éviter. *Reco* : distinguer panics « mismatch input » (→ erreur) des post-conditions internes (→ re-panic) via sentinelle.
- **A-07 [Moyenne] — `fft_poly.go:343-347`** *(confirmé)*. `NTransform`/`InvNTransform` (API publique exportée) paniquent sur précondition, hors chemin `recover`, sans variante `*Safe`. *Reco* : router la précondition vers la valeur de retour `error` existante, ou documenter le contrat dur.

### 5.6 `internal/calibration`, `metrics` — sain, persistance fragile

L3 corrigé (§4). **A-11** : `SaveProfile` (`profile.go:110-125`) fait `os.WriteFile` direct (pas de temp+rename atomique) → corruption si calibrations concurrentes inter-process / crash en écriture ; `LoadOrCreateProfile` masque l'échec (re-calibration coûteuse silencieuse). La concurrence *intra*-process est correcte (sémaphore, goroutine progress jointe). G304 exclu volontairement (CLI mono-utilisateur, P2-10) — à conserver.

### 5.7 `internal/cli` (+ completion), `tui`, `ui`, `progress`, `format` — mature, déjà refactoré

Les « God functions » de `Claude.md` (model.go 425 L, completion.go 520 L, Update 16 messages) **n'existent plus** (model.go ~184 L routeur pur, completion = dispatcher 30 L, 4 shells dérivés d'un `flagRegistry` unique). Pas de duplication `ui`↔`tui/styles` (dérivation, source unique `ui/themes.go`). Thread-safety `progress` **saine sur le chemin de production** (`Freeze` snapshot+`recover` par observateur, `programRef` sous `RWMutex`). Constats : **A-10** (overflow `CalcTotalWork` → progression figée grands N), **A-19** (API morte `Register/Notify/ChannelObserver` — seul `Freeze` vivant ; `ProgressDoneMsg` routé puis ignoré), **A-22** (allocations ANSI répétées + `updateContent()` rebâti à chaque ajout — quadratique sur longue session TUI), et **injection latente non exploitable** dans les scripts de complétion (texte `Help`/`Values` non échappé ; `flagRegistry` statique → non déclenchable aujourd'hui, devient vulnérable si noms d'algos dynamiques/i18n).

### 5.8 `test/e2e` — peu discriminant

**A-15** : plusieurs cas déclarent `wantCode` puis acceptent tout code non nul via `t.Logf` (`cli_e2e_test.go:154-160,271-273`) → régression de mapping de codes de sortie non détectée. Aucun `testing.Short()` guard : les e2e (rebuild + exécutions) tournent même en `-short` (job Windows CI), allongeant et fragilisant le pipeline.

---

## 6. Constats transverses

### 6.1 Concurrence
Le cœur orchestration est correct. Le risque concurrent est **concentré dans `bigfft`** : état global mutable lu sur le hot path FFT sans synchronisation (A-02, A-03) et aliasing cache/backing (A-01). Tous **latents** : ne se déclenchent que sous recalibration concurrente, multi-tenant `FFTContext`, ou pression cache élevée — absents du flux mono-calcul nominal, mais sur la trajectoire d'évolution annoncée (R3.7).

### 6.2 Mémoire & pooling
Pooling robuste et testé (L1/L4 corrigés, gardes de régression). Dette : incohérence de documentation des seuils (A-16), ~25 globaux dans `bigfft` empêchant l'isolation des tests (commentaires « No t.Parallel(): global pool state » confirmés).

### 6.3 Gestion d'erreurs
`%w` cohérent hors `bigfft`. Dans `bigfft`, le modèle panic/recover est *globalement* défendable (wrappers `*Safe` + recover de frontière) mais le `recover()` indiscriminé masque les régressions internes (A-06) et `NTransform` panique hors filet (A-07). Un cas `orchestration/lastdigits.go:38` utilise `fmt.Errorf` brut au lieu d'un type structuré (incohérence mineure).

### 6.4 Écarts documentation ↔ code (risque process majeur — **A-04 / A-21 / A-23**)
`Claude.md` est l'instruction projet **impérative** et décrit un état **pré-refactoring** :

1. **« CI/CD : aucun workflow GitHub Actions » (`Claude.md:19,134,165`)** — **FAUX** : `.github/workflows/ci.yml` (test/lint/build, 3 OS) **et** `coverage.yml` existent ; badge CI actif `README.md:3`. R4.1 est livré (`a2595b7`).
2. **`ultrareview.md` / `ultrareviewplan.md` / `docs/audits/`** cités ~12× comme sources normatives obligatoires (« Toute modification doit citer la section de `ultrareview.md` ») — **inexistants** (`git ls-files` : aucun match `ultrareview|audit`). Tout le « Workflow recommandé » et la convention de message de commit sont inapplicables.
3. **Table « bugs latents non corrigés » (`Claude.md:80-90`)** — les 5 sont corrigés/infirmés (§4) ; lignes citées périmées.
4. **Package `parallel` « quasi-mort »** — faux, sur le hot path concurrent (§5.2).
5. **« 22 linters »** (`Claude.md:121,135`, README:239) — **24** réellement activés dans `.golangci.yml`. `sysmon/`/`parallel/` listés malgré R4.5/R2.2 ; LOC/packages périmés (A-21).
6. **`docs/CALIBRATION.md`** : struct `CalibrationProfile` sans `Confidence` ; « Tier 3 / newCalibrationRunner » obsolète vs pattern Strategy ; `FIBCALC_PROFILE_MAX_AGE` non documenté (A-23).

*Impact* : un agent autonome instruit de suivre `Claude.md` (directives marquées impératives, prioritaires) **re-implémente du travail livré, cherche des fichiers fantômes, ou défait des correctifs gardés en régression**. C'est, en pratique, le risque de régression le plus probable de ce dépôt — d'où la sévérité **Haute** (au-delà du « Moyenne » nominal pour un écart doc/code, car le document dérivant est le contrat d'instruction normatif). *Note* : `audit-prompt.md` (périmètre §1) hérite lui-même de cette dérive (« parallel quasi-mort ») — limite assumée de cet audit, le code restant la source de vérité.

### 6.5 Sécurité, build & CI
`go vet` propre ; CI avec `permissions: contents: read` (moindre privilège), matrice 3 OS, cache, `fail-fast:false`, `concurrency` cancel-in-progress — bonnes pratiques. Lacunes : **A-12** (`golangci-lint version: latest` + `check-latest:true` → verdicts non reproductibles), **A-13** (pas de garde benchmark malgré la politique « régression >5 % = blocage », pas de seuil de couverture, `coverage.yml` pas sur PR, branche PGO de `make build` non exercée), **A-14** (`-race` absent sous Windows, plateforme primaire du mainteneur). Exclusions gosec G115/G304 documentées et justifiées (CLI mono-utilisateur) — à conserver.

---

## 7. Métriques

| Indicateur | Valeur (mesurée / observée) |
|---|---|
| Fichiers `.go` / LOC | 239 / ~35 454 |
| Packages | 24 (22 internes + 2 `cmd/`) |
| `go vet ./...` | **exit 0, aucun avertissement** |
| Packages les plus denses | `bigfft` 7936 · `fibonacci` 6802 · `tui` 4460 · `calibration` 2629 · `config` 2137 |
| Linters golangci-lint actifs | **24** (doc annonce 22) |
| Seuils complexité configurés | gocyclo 15 · gocognit 30 · funlen 100 L/50 stmts |
| Globaux `internal/bigfft` | ~25 (recensés ; dont 4 mutables non synchronisés sur hot path) |
| Bugs latents R1.1–R1.5 | **5/5 corrigés ou infirmés**, avec gardes de régression |
| `panic(` hors `bigfft/fermat`+invariants | aucun détecté dans app/orchestration/config/errors |
| Couverture (badge README) | 87,5 % (déclaré ; non re-mesuré ici) |
| Fichiers obsolètes cités par doc | `ultrareview.md`, `ultrareviewplan.md`, `docs/audits/` (inexistants) |
| `t.Parallel()` | adoption élevée (échantillonné, non chiffré fonction par fonction) |

---

## 8. Feuille de route priorisée (proposée — ne réordonne aucun plan existant)

**Vague A — Sûreté concurrence `bigfft` (bloquant si parallélisme FFT + cache actifs sur charges longues)**
- A-01 (M) — refcount/epoch sur `cacheEntry`, recyclage `backing` conditionné à l'absence de référence.
- A-02 (S) — atomiser `tc.config.{Enabled,MinBitLen,MaxEntries}`.
- A-03 (M) — atomiser/injecter `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` (aligner R3.7 `FFTContext`).
- A-05 (S) — assertion défensive d'invariant dans `putByKey`.
> Toute modif `bigfft`/`fibonacci` : `make benchmark` avant/après (politique régression >5 %).

**Vague B — Dette documentaire normative (risque de régression process)**
- A-04 (M) — réaligner `Claude.md` sur l'état réel (CI existante, bugs résolus, purger références `ultrareview*`/`docs/audits`, corriger statut `parallel`).
- A-21 (S) / A-23 (S) — métriques, arbre packages, `docs/CALIBRATION.md`.

**Vague C — Validation des entrées & robustesse**
- A-08 (S), A-09 (S) — validations config + non-avalement des erreurs d'env.
- A-06 (M), A-07 (S) — discrimination panic input vs post-condition ; `NTransform` sans panic.
- A-11 (M) — écriture atomique du profil (temp+rename).

**Vague D — CI & tests**
- A-12 (S), A-14 (S) — épingler `golangci-lint` ; activer `-race` Windows.
- A-13 (M) — job benchmark comparatif sur PR, seuil de couverture, `coverage.yml` sur PR.
- A-15 (M) — `testing.Short()` guard e2e ; `t.Errorf` sur codes de sortie contractuels.

**Vague E — Hygiène (Basse)**
- A-10 (M, espace log), A-16 (S), A-17 (M, garde `-race`), A-18 (S, documenter invariant), A-19 (S, réduire surface API), A-20 (M, test d'équivalence oracle), A-22 (M, invalidation paresseuse TUI).

Effort cumulé indicatif : Vague A ≈ 4–6 j · B ≈ 2–3 j · C ≈ 4–5 j · D ≈ 4–6 j · E ≈ 4–5 j.

---

## 9. Checklist de couverture

| Package | Statut |
|---|---|
| `cmd/fibcalc`, `cmd/generate-golden` | Audité intégralement (sources) |
| `internal/app` | Audité intégralement (sources) |
| `internal/orchestration` | Audité intégralement (sources) |
| `internal/config` | Audité intégralement (sources) |
| `internal/errors` | Audité intégralement (sources) |
| `internal/parallel` | Audité intégralement |
| `internal/fibonacci` | Sources clés lues intégralement ; ~22 `*_test.go` échantillonnés |
| `internal/fibonacci/memory` | Audité intégralement (`gc_control.go`/`arena.go`/`budget.go` lus directement) |
| `internal/fibonacci/threshold` | Audité intégralement (sources) |
| `internal/fibonacci/fibonaccitest` | Audité (stub) |
| `internal/bigfft` | 26/26 sources non-test lues ; tests échantillonnés ; asm non audité sur le fond |
| `internal/calibration` | Audité intégralement (sources) ; tests échantillonnés |
| `internal/metrics`, `metrics/system` | Audité (sources) |
| `internal/cli`, `cli/completion` | Audité intégralement (sources) |
| `internal/tui`, `tui/component` | Sources clés lues ; `*_test.go` échantillonnés |
| `internal/ui` | Audité (sources) |
| `internal/progress` | Audité intégralement (sources) |
| `internal/format` | Audité (sources principales ; `duration.go` échantillonné) |
| `internal/testutil` | Audité |
| `test/e2e` | Audité intégralement |
| Build/CI/config (`Makefile`, `.golangci.yml`, `.github/workflows/`, `go.mod`) | Audité intégralement |
| Cohérence `docs/`, `README.md`, `Claude.md` ↔ code | Audité (écarts en §6.4) |

Aucun package laissé « non audité ». Limite assumée : couverture des `*_test.go` par échantillonnage (les évaluations D8 sont indicatives).

---

## 10. Annexe — commandes (lecture seule)

- `git ls-files "*.go"` — inventaire (239 fichiers, 24 dirs).
- LOC par package — `Get-Content | Measure-Object -Line` (≈ 35 454 LOC).
- `go vet ./...` — **exit 0** (CGO_ENABLED=0).
- Lectures directes vérifiées : `gc_control.go`, `fastdoubling.go:388-428`, `fft_cache.go:200-389`, `pool.go:124-160`, `.github/workflows/ci.yml`, `.golangci.yml`, `Makefile`, `go.mod`, `README.md`.
- 5 agents d'audit lecture seule (clusters §2). Aucune commande mutante exécutée. `golangci-lint`/`gosec` non lancés isolément (couverts en CI).
