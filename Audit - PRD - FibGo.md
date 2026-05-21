# PRD — FibCalc Hardening Initiative

**Source de vérité amont** : `Audit - Global - FibGo.md`
**Version** : 1.0
**Date** : 2026-05-21
**Auteur** : agbruneau
**Statut** : Draft for execution

---

## 1. Contexte et Problème

FibCalc (`github.com/agbru/fibcalc`) est un calculateur Fibonacci haute performance Go (~35 500 LOC, 23 packages) noté **81/100** par audit consolidé (Claude + Gemini). Le verdict unifié est sans ambiguïté : *prototype académique abouti, dette concurrente et architecturale non close*. Six familles de défauts P0 bloquent un éventuel passage à un usage *library-style* ou à une release v1.0 industrielle :

1. **Data races** confirmées sur trois call-sites (`DynamicThresholdManager`, `TransformCache.config`, globaux `bigfft`).
2. **`recover()` global** dans `bigfft/fft.go` qui masque les violations de post-conditions algorithmiques.
3. **Triangle d'imports remontants** (`threshold → config → memory`) violant la hiérarchie Clean Architecture revendiquée.
4. **Gate de régression performance non bloquant** (`continue-on-error: true`).
5. **`internal/cli/completion/` sans aucun test** alors que le risque sécurité (injection shell) est explicitement flaggé.
6. **Absence de conteneurisation** alors que CGO/MinGW/GMP sont sur le chemin critique.

Sept autres familles P1/P2 prolongent cette dette (sur-ingénierie de la calibration, drift documentaire C4, couplages `errors → format` et `tui → fibonacci`, risque résiduel cache FFT, golden/fuzz sous-dimensionné en régime FFT).

## 2. Vision Produit Post-Bonification

À l'issue de l'initiative, FibCalc sera :

- **Concurrence-safe** : `go test -race -count=100 ./...` propre sous charge concurrente hostile, ouvrant l'usage *library multi-tenant*.
- **Reproductible** : `make all` reproductible dans `.devcontainer/` et image Docker multi-stage ; CI consomme l'image publiée.
- **Architecturalement étanche** : hiérarchie `cmd → app → orchestration → fibonacci/bigfft → config/errors` *enforced* par `go list` et un test d'architecture (`internal/arch_test.go`).
- **Gated en performance** : régression > 5 % bloque la CI ; baselines versionnées dans `docs/audits/`.
- **Documenté de manière non drift** : diagrammes C4 auto-générés ou synchronisés ; `CLAUDE.md` et `CHANGELOG.md` non contradictoires.
- **Lean** : `DynamicThresholdManager`/sur-couches de calibration soit prouvées nécessaires (benchstat), soit supprimées.

## 3. Personas et Cas d'Usage Cibles

| Persona | Besoin | Cas d'usage critique |
|---|---|---|
| **Mainteneur académique** | Garder le projet auditable, démontrable, performant. | `make benchmark` reproductible ; baselines historisées ; `make lint` strict. |
| **Contributeur externe** | Onboarding < 10 min, build reproductible, tests qui passent localement comme en CI. | Ouvrir le projet dans VS Code via `.devcontainer/`, lancer `make all`, voir un build vert. |
| **Consommateur *library*** (futur) | Importer `internal/fibonacci` (après extraction) sans dépiler de couplages remontants ; concurrence-safe. | Plusieurs goroutines appelant `Mul()` concurremment avec des `FFTContext` distincts, sans data race. |
| **Auditeur sécurité** | Surface CLI non-injectable, complétion shell échappée, secret zero. | Identifiants avec `$(...)`, backticks, `;`, espaces sont neutralisés dans tous les shells supportés. |

## 4. Périmètre et Hors-Périmètre

### 4.1 Dans le périmètre

Tous les items P0/P1 de `Audit - Global - FibGo.md §5`, soit 15 actions distinctes. Les P2 (8 actions de polish) sont inclus comme *backlog* exécutable au fil de l'eau.

### 4.2 Hors-périmètre

- Réécriture de l'algorithme Fast Doubling ou de `bigfft` (ce sont des forces avérées).
- Suppression de la TUI (D4 arbitré en faveur de Claude : découpler, pas supprimer).
- Suppression du backend GMP (CGO opt-in via build tag, ne pénalise pas la portabilité par défaut).
- Migration vers `GOEXPERIMENT=arenas` (suggestion Gemini retenue comme exploration P3 hors PRD).

## 5. Epics, Requirements et Critères d'Acceptation

### EPIC E1 — Concurrence Safe (P0)

**Objectif** : éliminer les data races confirmées et fermer la trajectoire `FFTContext`.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E1-R1 | Convertir `DynamicThresholdManager.currentFFTThreshold`, `currentParallelThreshold`, `lastAdjustment` en `atomic.Int64` / `atomic.Pointer[time.Time]`. | `go test -race -count=100 ./internal/fibonacci/threshold/...` propre. Test concurrent multi-writers ajouté. |
| E1-R2 | Protéger les 4 lectures `TransformCache.config` (`internal/bigfft/fft_cache.go:207, 301, 444`, `internal/bigfft/context.go:299`) sous `RLock` **ou** dupliquer `Enabled` dans `atomic.Bool`. | `go test -race -count=100 ./internal/bigfft/...` propre. |
| E1-R3 | Synchroniser `fftThreshold`, `ParallelFFTRecursionThreshold`, `MaxParallelFFTDepth` — `atomic.Int64` **ou** portage dans `FFTContext` exclusif. | Les globaux mutables exportés disparaissent de `bigfft/fft.go:35` et `bigfft/fft_recursion.go:28, 33`. |
| E1-R4 | Fermer le risque résiduel cache FFT (refcount sur `cacheEntry.backing` ou deep-copy à `Get`). | Test concurrent éviction + lecture vivante ajouté ; pas d'aliasing observable. |

### EPIC E2 — Restauration des Invariants Algorithmiques (P0)

**Objectif** : ne plus masquer les violations de post-conditions FFT.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E2-R1 | Supprimer ou restreindre le `recover()` global de `internal/bigfft/fft.go:41-101`. | Les panics `fermat.go:201, 226, 262, 281` propagent (test `TestFermatPostConditionPanic` dédié). |
| E2-R2 | Exposer publiquement les wrappers `*Safe` (`MulSafe`, `SqrSafe`, etc.) **ou** les supprimer. | Décision documentée dans un ADR ; 48 LOC orphelines absentes. |
| E2-R3 | Logger ou compter les `recover()` muets (`internal/progress/observer.go:142-150`). | Compteur `recoveredObservers` exposé via `metrics.Snapshot()`. |

### EPIC E3 — Étanchéité Architecturale (P0/P1)

**Objectif** : briser les couplages remontants et fermer la trajectoire Clean Architecture.

| ID | Requirement | Priorité | Critère d'acceptation |
|---|---|---|---|
| E3-R1 | Briser l'import `internal/fibonacci/threshold → internal/config` via injection de `ThresholdTuningProfile`. | P0 | `go list -deps ./internal/fibonacci/threshold` n'inclut plus `internal/config`. |
| E3-R2 | Sortir `format` de `internal/errors` (struct sérialisable, formatage délégué). | P1 | `internal/errors` n'importe plus `internal/format`. |
| E3-R3 | Découpler `internal/tui` de `internal/fibonacci` ; passer par `internal/orchestration`. | P1 | `go list -deps ./internal/tui` n'inclut plus `internal/fibonacci`. |
| E3-R4 | Ajouter un test d'architecture `internal/arch_test.go` enforçant la hiérarchie. | P1 | `go test ./internal/...` échoue si un futur import remontant est introduit. |

### EPIC E4 — Gate de Régression Performance (P0)

**Objectif** : rendre la directive 1 du `CLAUDE.md` non-déclarative.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E4-R1 | Activer `benchstat ≥5%` bloquant dans `.github/workflows/ci.yml` (retirer `continue-on-error: true`). | Le job `bench` fait échouer une PR introduisant > 5 % de régression. |
| E4-R2 | Versionner une baseline `docs/audits/bench-baseline.txt` régénérée à chaque release. | Workflow `make bench-baseline` documenté ; fichier sous Git. |
| E4-R3 | Exécuter le gate sur le sous-ensemble `BenchmarkFibonacci_FastDoubling`, `BenchmarkFibonacci_FFT`, `BenchmarkFibonacci_Strassen` (chemins critiques). | Liste de bench sous gate documentée dans `docs/PERFORMANCE.md`. |

### EPIC E5 — Sécurité Surface CLI Completion (P0)

**Objectif** : combler le trou de tests sur `internal/cli/completion/`.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E5-R1 | Tests adversariaux : identifiants contenant `$(...)`, backticks, `;`, espaces, guillemets, `\`. | Golden output par shell (bash/zsh/fish/powershell). |
| E5-R2 | Fonction d'échappement centralisée par shell, vérifiée par fuzz. | Couverture ≥ 80 % sur `internal/cli/completion/`. |
| E5-R3 | Documenter le contrat d'échappement dans `internal/cli/completion/doc.go`. | Section « Security contract » présente. |

### EPIC E6 — Conteneurisation et Reproductibilité (P0)

**Objectif** : éliminer le syndrome *it works on my machine*.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E6-R1 | `Dockerfile` multi-stage : build + test + race + GMP + PGO. | `docker build .` produit une image < 500 Mo ; `docker run … make all` vert. |
| E6-R2 | `.devcontainer/devcontainer.json` + image associée. | Ouverture VS Code → environnement complet (Go, gcc/MinGW, libgmp-dev). |
| E6-R3 | CI consomme l'image (au moins le job Linux). | `.github/workflows/ci.yml` référence l'image publiée. |
| E6-R4 | Résoudre la contradiction `CLAUDE.md:125` vs `CHANGELOG.md:44` sur le race detector Windows. | Section dédiée dans `CLAUDE.md` ; CI Windows documente sa toolchain MinGW. |

### EPIC E7 — Lean Architecture : Décider du Sort de la Calibration Adaptative (P1)

**Objectif** : trancher entre `DynamicThresholdManager` et `calibration/`.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E7-R1 | Produire `docs/audits/bench-dtm-{on,off}.txt` sur les 5 tailles `BenchmarkFibonacci_*`. | Données chiffrées présentes. |
| E7-R2 | ADR `docs/adr/0001-dtm-decision.md` : conserver si gain > 5 %, supprimer sinon. | ADR mergé, décision implémentée. |
| E7-R3 | Si suppression : ~283 LOC retirées de `internal/fibonacci/threshold/manager.go` + invariant A-18 archivé. | `grep DynamicThresholdManager` ne retourne que des références historiques. |

### EPIC E8 — Couverture de Test en Régime FFT (P1)

**Objectif** : exercer le régime ≥ 500k bits où vit la complexité.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E8-R1 | ≥ 5 nouvelles entrées golden au-delà de F(50 000), idéalement F(100k), F(500k), F(1M), F(5M), F(10M). | Fichier `internal/fibonacci/testdata/fibonacci_golden.json` étendu. |
| E8-R2 | Cibles fuzz dédiées : `FuzzFermatMul`, `FuzzFermatShift`, `FuzzPolyTransform` avec corpus seed dérivé de `fftSizeThreshold[]`. | Couverture fuzz > 0 sur `internal/bigfft/fermat.go` et `internal/bigfft/fft.go`. |
| E8-R3 | Bornes fuzz Fibonacci relevées au-delà de 50 000 (≥ 200 000). | `internal/fibonacci/fibonacci_fuzz_test.go:30` modifié. |

### EPIC E9 — Synchronisation Documentaire (P1/P2)

**Objectif** : éliminer le drift documentaire transverse.

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E9-R1 | Diagrammes C4 corrigés (`sysmon → internal/metrics/system`) dans `dependency-graph.mermaid` et `container-diagram.mermaid`. | Diagrammes à jour. |
| E9-R2 | `EVALUATION.md` retiré, déplacé vers `docs/external-reviews/` ou enrichi d'un en-tête de transparence. | Décision matérialisée. |
| E9-R3 | Décompte de packages auto-généré (`go list ./... \| wc -l`) référencé depuis `CLAUDE.md` et `ARCH.md`. | Plus de divergence 21 vs 23 vs 24. |
| E9-R4 | Badge couverture statique remplacé par badge dynamique alimenté par `coverage.yml`. | `README.md:6` mis à jour. |
| E9-R5 | `ruvector.db` ajouté à `.gitignore`. | `git check-ignore ruvector.db` non vide. |

### EPIC E10 — Polish, Hygiène et Conformité (P2)

| ID | Requirement | Critère d'acceptation |
|---|---|---|
| E10-R1 | Tests de panic ciblés pour les 13 sites non-test (priorité `fermat.go`). | Couverture des panics > 80 %. |
| E10-R2 | Renommer `cap := cap(...)` → `c := cap(...)` à `internal/bigfft/pool.go:242, 330, 418`. | Shadowing résolu. |
| E10-R3 | Activer `govet shadow` (mode warning) dans `.golangci.yml`. | Linter étendu. |
| E10-R4 | Étoffer les `doc.go` purement formels (`cli`, `config`, `orchestration`, `fibonaccitest`). | Chaque `doc.go` ≥ 20 lignes documentaires. |
| E10-R5 | Valider la matrice de portabilité (cross-compile `linux/arm64`, `darwin/arm64`) et documenter les fallbacks de `arith_amd64.go`. | CI cross-compile vert, section dédiée dans `docs/PORTABILITY.md`. |

---

## 6. Exigences Non Fonctionnelles (NFR)

| ID | Catégorie | Exigence | Mesure |
|---|---|---|---|
| NFR-P1 | Performance | Aucune régression > 5 % sur les bench chemins critiques. | `benchstat` gate CI. |
| NFR-P2 | Performance | Gain global ≥ +2 % attendu après élimination du `recover()` global (post-condition handling). | Comparaison baseline pré/post P0. |
| NFR-S1 | Sécurité | Pas d'injection shell possible via `internal/cli/completion/` quelle que soit l'entrée registry. | Tests adversariaux + fuzz. |
| NFR-S2 | Sécurité | Tous les secrets et bases de données externes (`ruvector.db`) gitignorés. | Audit `.gitignore`. |
| NFR-Q1 | Qualité | Couverture globale ≥ 80 % maintenue (existant). | `coverage.yml`. |
| NFR-Q2 | Qualité | `go test -race -count=100 ./...` propre sur les 3 OS de la matrice CI. | CI verte sous stress concurrent. |
| NFR-D1 | DevEx | Onboarding `git clone → make all vert` ≤ 10 min sur machine fraîche via `.devcontainer/`. | Test manuel mainteneur. |
| NFR-D2 | DevEx | Diagrammes C4 et `CLAUDE.md`/`ARCH.md` non contradictoires. | Linter custom ou inspection trimestrielle. |

## 7. Métriques de Succès

| Métrique | Baseline | Cible | Mesure |
|---|---:|---:|---|
| Note audit consolidée | 81 / 100 | **≥ 92 / 100** | Re-audit post-P0+P1 selon la même grille. |
| Data races détectées sous `-race -count=100` | ≥ 3 sites | 0 | CI race job. |
| Imports remontants détectés | 3 | 0 | Test `arch_test.go`. |
| LOC `internal/calibration/` (si suppression E7) | 1 686 | ≤ 1 200 ou suppression DTM | `cloc`. |
| Couverture `internal/cli/completion/` | 0 % | ≥ 80 % | `coverage.yml`. |
| Drift C4 documenté | ≥ 4 occurrences | 0 | Audit manuel. |
| Temps onboarding (devcontainer) | non mesurable | ≤ 10 min | Mesure mainteneur. |

## 8. Risques et Mitigations

| Risque | Impact | Mitigation |
|---|---|---|
| Régression performance suite au retrait du `recover()` global. | Moyen — propagation panic peut interrompre des workflows existants. | Wrappers `*Safe` exposés publiquement comme API d'opt-in ; tests panic ciblés ; baseline benchstat. |
| Conversion atomique fragilise le code (mémorisation oubliée d'un champ). | Moyen | Tests race count=100 + revue de code par agent reviewer. |
| Suppression du `DynamicThresholdManager` casse un cas d'usage non testé. | Faible | ADR formel, période d'observation 1 release avant suppression définitive. |
| Image Docker volumineuse (libgmp + MinGW + Go toolchain). | Faible | Multi-stage build agressif, base `alpine` ou `debian-slim` pour runtime. |
| Tests adversariaux completion révèlent une faille effective. | Élevé — sécurité | Traiter comme issue P0 dès découverte, hotfix release. |

## 9. Dépendances et Pré-requis

- Toolchain Go 1.25.0+ (toolchain 1.26.2) — déjà en place.
- `benchstat` (`go install golang.org/x/perf/cmd/benchstat@latest`) — à automatiser dans `Makefile`.
- `cloc` (mesure LOC) — `make stats` à ajouter.
- Image Docker base : `golang:1.26-bookworm` (CGO + GMP disponibles via apt).
- Accès écriture à `docs/audits/`, `docs/adr/`, `.devcontainer/`.

## 10. Hors-Périmètre Explicite (rappel)

- Réécriture algorithmique (Fast Doubling, FFT, Strassen) — forces avérées.
- Suppression de la TUI ou du backend GMP.
- Migration `GOEXPERIMENT=arenas` — exploration P3 hors PRD.
- Publication de `internal/fibonacci/` comme module séparé sous `github.com/agbru/fibcalc-fibonacci` — sortie potentielle post-P0+P1.

---

## 11. Critère Global de Sortie

Le PRD est **acquitté** lorsque :
1. Tous les items P0 (E1, E2 R1+R3, E3-R1, E4, E5, E6) sont mergés en `main` avec CI verte.
2. Un re-audit indépendant sur la même grille rend une note ≥ 92 / 100.
3. `go test -race -count=100 ./...` est propre sur les 3 OS.
4. Tous les items P1 sont soit mergés soit explicitement reportés via ADR.

Le PRD est **clos** lorsque les items P2 ont été triés (gardés ou refermés comme *won't fix* via ADR).
