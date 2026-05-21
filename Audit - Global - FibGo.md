# Audit Global — FibCalc (`github.com/agbru/fibcalc`)

**Dépôt audité** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo`
**Commit de référence** : `6b9de67` (branche `main`)
**Date de consolidation** : 2026-05-21
**Sources unifiées** :
- `Audit - Claude - FibGo.md` — 82/100, inspection solo + 5 agents parallèles, citations `file:line` exhaustives.
- `Audit - Gemini - FibGo.md` — 82/100, revue narrative axée principes architecturaux et idiomatisme Go.

---

## 1. Synthèse Globale et Verdict Unifié

**Note consolidée : 82 / 100** (convergence stricte des deux évaluations indépendantes).

Les deux audits convergent sur la note globale tout en empruntant des chemins de raisonnement complémentaires. Claude établit ses pénalités à partir d'évidences ponctuelles citées (`file:line`), surfaçant trois familles de défauts concrets : data races hors verrou, `recover()` global masquant des invariants algorithmiques, et un triangle d'imports remontants. Gemini privilégie une critique systémique, dénonçant une hypertrophie architecturale (orchestrateur, TUI, calibration), une subversion du modèle mémoire managé (`arena`, `gc_control`), et un verrouillage matériel (`arith_amd64.go`, CGO/GMP).

La triangulation fait émerger un **diagnostic unifié** : *prototype académique abouti, mais inadapté à un usage `library-style` ou à une release v1.0 industrielle*. Les deux audits s'accordent sur la priorité absolue de la conteneurisation (Docker/devcontainer), sur l'existence d'une sur-ingénierie auto-reconnue, et sur la nécessité de refermer la trajectoire Clean Architecture. Ils divergent sur la lecture du `gc_control.go` (« garde-fou réel » pour Claude, « hérésie managée » pour Gemini), sur la valeur des tests golden (force vs. *brittle tests*), et sur la pertinence de l'assembleur amd64 et de GMP.

---

## 2. Méta-Analyse : Convergences, Divergences, Arbitrages

### 2.1 Convergences à Haute Confiance

Constats partagés par les deux audits, donc fortement validés :

| # | Constat partagé | Claude `file:line` | Gemini | Confiance |
|---|---|---|---|---|
| C1 | **Absence de containerisation/devcontainer** = priorité haute | §4 item 11 | §4 Priorité Haute | ★★★ |
| C2 | **Sur-ingénierie de la calibration** (~1 686 LOC vs 4 277 cœur) | §2.5 (−2) | §2.1 (−5) | ★★★ |
| C3 | **TUI questionnable** pour calcul scalaire séquentiel | §2.1 couplage `tui→fibonacci` | §2.5 « hypertrophie absurde » | ★★ |
| C4 | **Architecture orchestrateur surdimensionnée** | §3.3 (d) `globalFactory` vs `DefaultFactory` | §2.1 *God Object* | ★★ |
| C5 | **`DynamicThresholdManager` redondant avec calibration** | §2.5 item 2, §4 item 6 | §2.5 (impliqué) | ★★ |
| C6 | **Excellence algorithmique réelle** (FFT, Fast Doubling, PGO) | §2.5 forces | §2.5 forces | ★★★ |
| C7 | **Documentation C4/algorithmes/Makefile substantielle** | §2.4 forces | §2.4 « remarquable » | ★★★ |

### 2.2 Divergences et Arbitrage

| # | Sujet | Position Claude | Position Gemini | Arbitrage retenu |
|---|---|---|---|---|
| D1 | **`gc_control.go` (désactivation GC)** | Innovation justifiée, panic-safe via `WithGC(fn)` | « Hérésie », fuites OOM, gêne *escape analysis* | **Claude** — l'invariant est gardé par `defer End()` ; trace une frontière acceptable pour N ≥ 1M ; à conserver mais documenter le périmètre. |
| D2 | **Tests golden** | Force (oracle indépendant via `cmd/generate-golden/`) | Faiblesse (*brittle tests*) | **Claude** — golden + property-based sont **complémentaires** ; le golden détecte les régressions byte-exactes que les propriétés mathématiques manquent. |
| D3 | **Assembleur amd64 + GMP CGO** | Non pénalisé | Verrouillage matériel grave | **Mixte** — `arith_amd64.go` est légitime (perf critique), GMP est un *opt-in* (`build tag gmp`). Pas un défaut de portabilité tant que les *fallbacks* Go purs existent ; à vérifier explicitement. |
| D4 | **TUI complète** | Couplage à fixer, pas suppression | Marquer pour dépréciation | **Claude** — découpler `tui → orchestration` plutôt que supprimer ; la TUI a une valeur DevEx réelle. |
| D5 | **`recover()` global FFT** | Défaut majeur (3.2) | Non identifié | **Claude** — confirmé par lecture directe : `bigfft/fft.go:41-101`. À corriger. |
| D6 | **Data races concrètes** | 3 sites identifiés et cités | Non identifié | **Claude** — à corriger en priorité haute. |
| D7 | **Imports remontants** | `threshold → config`, `errors → format`, `tui → fibonacci` cités | Non identifié | **Claude** — confirmé, à briser. |

### 2.3 Constats Uniques

**Spécifiques à Claude** (issus de la grille à 100 points exécutée en parallèle) :
- Drift documentaire transverse (C4 `sysmon`, badge couverture statique, `EVALUATION.md` orphelin, race detector Windows contradictoire entre `CLAUDE.md` et `CHANGELOG.md`).
- Gate `>5%` perf déclaré `continue-on-error: true` (`.github/workflows/ci.yml:84-90`).
- `internal/cli/completion/` (~555 LOC) **sans aucun test**, alors que le risque sécurité (échappement shell) est explicitement flaggé.
- Corpus fuzz sous-dimensionné : bornes `n > 50000` qui n'exercent **pas** le régime FFT.
- Wrappers `*Safe` orphelins (48 LOC sans consommateur production).
- `ruvector.db` (1.5 Mo) non gitignoré présent à la racine.

**Spécifiques à Gemini** :
- Surcharge événementielle hot path (`progress/observer` ↔ `tui/handlers`) jugée pathologique.
- Compilation croisée Go potentiellement compromise par CGO/asm (à valider).
- Recommandation `GOEXPERIMENT=arenas` comme alternative idiomatique au `arena.go` manuel.

---

## 3. Verdict Détaillé Unifié par Critère

| Critère | Claude | Gemini | **Consolidé** | Justification synthétique |
|---|---:|---:|---:|---|
| Architecture et Conception Système | 19 / 25 | 20 / 25 | **19 / 25** | Couplages d'imports remontants (Claude) + orchestrateur surdimensionné (Gemini) tirent vers le bas. |
| Qualité du Code et Robustesse | 21 / 25 | 20 / 25 | **20 / 25** | `recover()` global (Claude) + `recover()` muet `progress` + shadowing `cap` ; GMP/asm acceptables. |
| Stratégie de Validation et de Test | 17 / 20 | 19 / 20 | **17 / 20** | Verdict Claude (gate perf non bloquant, fuzz hors FFT, completion sans tests) plus rigoureux. |
| Documentation et DevEx | 12 / 15 | 11 / 15 | **12 / 15** | Convergence forte : forte sur le contenu, faible sur la conteneurisation et le drift C4. |
| Complexité Technique et Innovation | 13 / 15 | 12 / 15 | **13 / 15** | Sophistication réelle (PGO, Fast Doubling matriciel, deep-copy justifiée chiffré). |
| **TOTAL** | **82** | **82** | **81** | Léger durcissement sur Qualité du Code pour intégrer les évidences Claude que Gemini n'avait pas observées. |

> **Note finale unifiée : 81 / 100** — la triangulation a marginalement durci le verdict (−1 sur Qualité du Code) sans pour autant changer la catégorie : *prototype académique abouti, dette concurrente et architecturale non close*.

---

## 4. Critiques Techniques Consolidées

### 4.1 Concurrence à granularité fine incomplète

**Sources** : Claude §3.1 (3 sites cités) ; Gemini implicite via critique mémoire.
**Description** : (a) `DynamicThresholdManager.ShouldAdjust` mute trois champs hors verrou (`internal/fibonacci/threshold/manager.go:194-202`) alors que les getters lisent sous `RLock` ; (b) `TransformCache.config.Enabled/MinBitLen` lu sur quatre call-sites hot path (`internal/bigfft/fft_cache.go:207, 301, 444`, `internal/bigfft/context.go:299`) sans `RLock` ; (c) globaux `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` exportés mutables sans `atomic`.
**Impact** : ferme la porte à un usage *library multi-tenant* ; `go test -race` flaggera sous charge concurrente hostile.

### 4.2 `recover()` global masque les invariants algorithmiques

**Sources** : Claude §3.2 ; non identifié par Gemini.
**Description** : `internal/bigfft/fft.go:41-101` installe `recover()` indistinct dans `Mul/Sqr/MulTo/SqrTo`. Toute violation de post-condition algorithmique dans `fermat.go` (`panic("len(z) > 2n+1")`, `panic("unexpected carry")`) ressort comme `error` opaque, indistinguable d'une erreur d'arité.
**Impact** : voie de régression silencieuse au cœur algorithmique ; 48 LOC de wrappers `*Safe` orphelins témoignent d'une intention non finalisée.

### 4.3 Dette architecturale transverse non close

**Sources** : Claude §3.3 ; Gemini §2.1 (orchestrateur God Object).
**Description** : (a) triangle `threshold → config → memory` ; (b) `errors → format` ; (c) `tui → fibonacci` court-circuitant `orchestration` ; (d) `globalFactory` package-level coexistant avec `DefaultFactory` injectable ; (e) globaux `bigfft` coexistant avec `FFTContext` injectable.
**Impact** : tout futur consommateur souhaitant extraire `internal/fibonacci/` comme bibliothèque indépendante doit dépiler ces couplages avant publication.

### 4.4 Reproductibilité environnementale insuffisante

**Sources** : Claude §2.4 (item 11) + Gemini §4 Priorité Haute.
**Description** : absence de `Dockerfile`, `.devcontainer/`, `docker-compose.yml`. Le projet exige CGO pour `-race` sous Windows et pour GMP. Les contributeurs sous Windows sans gcc rencontrent des friction documentées (`CLAUDE.md:125` vs `CHANGELOG.md:44` se contredisent).
**Impact** : syndrome *it works on my machine*, drift CI/local, friction d'onboarding.

### 4.5 Sur-ingénierie auto-reconnue

**Sources** : Claude §2.5 (−2) + Gemini §2.1, §2.5.
**Description** : `internal/calibration/` (1 686 LOC) vs `internal/fibonacci/` (4 277 LOC) ratio 39 % ; `DynamicThresholdManager` redondant ; orchestrateur dynamique pour un calcul déterministe ; `cmd/fibcalc` mentionne dans le README « the most over-engineered Fibonacci calculator ».
**Impact** : maintenance déraisonnable, surface d'attaque conceptuelle, complexité accidentelle.

---

## 5. Plan d'Action Consolidé

### 5.1 Priorité P0 — Bloquant pour usage *library-style* ou release v1.0

| # | Action | Sources | Effort | Critère de succès |
|---|---|---|---:|---|
| P0-01 | **Clore les data races confirmées** (`DynamicThresholdManager`, `TransformCache.config`, globaux `bigfft`) via `atomic.*` ou `FFTContext`. | Claude §4.1, §4.2 | M | `go test -race -count=100 ./...` propre sous charge concurrente. |
| P0-02 | **Restaurer la propagation des post-conditions FFT** : supprimer ou restreindre le `recover()` global de `internal/bigfft/fft.go:41-101`. | Claude §4 item 2 | S | Wrappers `*Safe` exposés publiquement ou supprimés ; tests panic ciblés sur `fermat.go`. |
| P0-03 | **Briser le triangle d'imports `threshold → config → memory`** via injection à la construction. | Claude §4 item 3 | M | `go list -json` confirme la hiérarchie `cmd → app → orchestration → fibonacci/bigfft → config/errors` du `CLAUDE.md`. |
| P0-04 | **Activer le gate `benchstat ≥5%` bloquant** en CI (`continue-on-error: false`) avec baseline versionnée. | Claude §4 item 4 | S | Workflow CI fait échouer une PR introduisant une régression > 5 % sur les bench `BenchmarkFibonacci`. |
| P0-05 | **Tests adversariaux sur `internal/cli/completion/`** (injection shell, échappement). | Claude §4 item 5 | S | ≥ 80 % couverture sur `completion/`, golden output par shell, fuzz sur identifiants spéciaux. |
| P0-06 | **Conteneurisation : Dockerfile multi-stage + `.devcontainer/devcontainer.json`** intégrant CGO/MinGW. | Claude §4 item 11 + Gemini §4 Haute | M | `make all` reproductible dans le container ; CI consomme l'image. |

### 5.2 Priorité P1 — Dette architecturale et qualité substantielle

| # | Action | Sources | Effort | Critère de succès |
|---|---|---|---:|---|
| P1-01 | **Statuer sur `DynamicThresholdManager` vs `calibration/`** : preuve `benchstat` > 5 % de gain, ou suppression. | Claude §4 item 6 | M | Décision documentée (ADR) + benchmarks `docs/audits/bench-dtm-{on,off}.txt`. |
| P1-02 | **Fermer le risque résiduel cache FFT** (refcount sur `cacheEntry.backing` ou deep-copy à `Get`). | Claude §4 item 7 | M | Test concurrent éviction + lecture vivant : pas d'aliasing. |
| P1-03 | **Étendre golden + fuzz dans le régime FFT** (≥ 5 entrées golden au-delà de F(50 000), fuzz `FuzzFermatMul`, `FuzzShift`, `FuzzPolyTransform`). | Claude §4 item 8 | M | Corpus seed dérivé de `fftSizeThreshold[]`. |
| P1-04 | **Sortir `format` de `errors`** : struct sérialisable, formatage délégué au présentateur. | Claude §4 item 9 | S | `import "internal/format"` retiré de `internal/errors/errors.go`. |
| P1-05 | **Découpler `internal/tui` de `internal/fibonacci`** en passant par `orchestration`. | Claude §4 item 10 + Gemini §3.2 | M | `go list -json internal/tui` n'importe plus `internal/fibonacci`. |
| P1-06 | **Synchroniser les diagrammes C4** (`sysmon → internal/metrics/system`). | Claude §4 item 12 | XS | Mermaid corrigé dans `dependency-graph.mermaid` et `container-diagram.mermaid`. |
| P1-07 | **Statuer sur `EVALUATION.md`** : retirer ou déplacer vers `docs/external-reviews/` avec en-tête de transparence. | Claude §4 item 13 | XS | Fichier déplacé ou supprimé, lien depuis README si conservé. |
| P1-08 | **Logger ou compter les `recover()` muets** (`internal/progress/observer.go:142-150`). | Claude §4 item 14 | XS | Compteur `recoveredObservers` exposé via metrics. |
| P1-09 | **Valider que les *fallbacks* Go purs existent pour `arith_amd64.go`** ou documenter la matrice de portabilité. | Gemini §3.3 | S | Build cross-compile `linux/arm64`, `darwin/arm64` validé en CI. |

### 5.3 Priorité P2 — Polish et hygiène

| # | Action | Sources | Effort |
|---|---|---|---:|
| P2-01 | Tests de panic ciblés pour les 13 sites non-test (priorité `fermat.go`). | Claude §4 item 15 | S |
| P2-02 | Renommer `cap := cap(...)` → `c := cap(...)` à `internal/bigfft/pool.go:242, 330, 418`. | Claude §4 item 16 | XS |
| P2-03 | Activer `govet shadow` (mode warning) dans `.golangci.yml`. | Claude §4 item 17 | XS |
| P2-04 | Remplacer le badge couverture statique par dynamique. | Claude §4 item 18 | XS |
| P2-05 | Réconcilier le décompte de packages (`CLAUDE.md:14` vs `ARCH.md:12`) — auto-générer. | Claude §4 item 19 | XS |
| P2-06 | Ajouter `ruvector.db` à `.gitignore`. | Claude §2.4 | XS |
| P2-07 | Étoffer les `doc.go` purement formels (`cli`, `config`, `orchestration`, `fibonaccitest`). | Claude §2.4 | S |
| P2-08 | Marquer dépréciation TUI ou la consolider derrière metrics Prometheus (optionnel). | Gemini §4 Basse | M |

---

## 6. Tableau Récapitulatif Unifié

| Dimension | Claude | Gemini | Consolidé | Verdict synthétique |
|---|---:|---:|---:|---|
| Architecture | 19/25 | 20/25 | **19/25** | Hiérarchie respectée mais data races + imports remontants ouverts. |
| Qualité du code | 21/25 | 20/25 | **20/25** | Idiomatique Go, `recover()` global et wrappers orphelins à fixer. |
| Tests | 17/20 | 19/20 | **17/20** | 5 couches indépendantes, mais gate perf déclaratif et angles morts. |
| Doc / DevEx | 12/15 | 11/15 | **12/15** | C4 + Makefile riches ; drift et absence de conteneurisation. |
| Complexité / Innovation | 13/15 | 12/15 | **13/15** | Sophistication réelle ; sur-ingénierie auto-reconnue. |
| **TOTAL** | **82/100** | **82/100** | **81/100** | Prototype académique abouti, dette concurrente et architecturale non close. |

---

## 7. Notes de Méthode et Confiance Épistémique

- **Convergence indépendante sur 82/100** par deux méthodes distinctes (audit par citations vs. audit par principes) constitue un fort signal de robustesse du verdict.
- **L'audit Claude est plus actionnable** (citations `file:line` précises permettant un patch direct), tandis que **l'audit Gemini est plus structurant** (cadre conceptuel YAGNI/KISS/OCP utile pour la priorisation).
- **Désaccords arbitrés** : 7 divergences identifiées, 5 tranchées en faveur de Claude (évidence directe), 1 mixte (`arith_amd64`/GMP), 1 nuancée (`gc_control`).
- **Risque résiduel d'angle mort partagé** : ni Claude ni Gemini n'a profilé sous charge réelle production (multi-tenant + N ≥ 10M) — les data races théoriques restent à reproduire empiriquement (la sortie de `make benchmark` et `go test -race` sous stress concurrent est un prérequis P0).
