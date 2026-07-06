# FibCalc — Calculateur Fibonacci haute performance

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge&logo=apache)](LICENSE)
![Status](https://img.shields.io/badge/Status-Prototype_acad%C3%A9mique-orange?style=for-the-badge)
[![Dashboard](https://img.shields.io/badge/Knowledge_Graph-Live-9b59b6?style=for-the-badge)](https://agbruneau.github.io/FibGo/dashboard/)

**FibCalc** est un prototype académique qui calcule des nombres de Fibonacci arbitrairement grands à très haute
vitesse. Il démontre une Clean Architecture, des stratégies zéro-allocation, du parallélisme adaptatif et des
algorithmes optimisés (Fast Doubling, exponentiation matricielle Strassen-Winograd, multiplication FFT
Schönhage-Strassen). Écrit en Go ; gère des indices de plusieurs centaines de millions.

> **[Dashboard knowledge-graph interactif →](https://agbruneau.github.io/FibGo/dashboard/)** — l'architecture
> complète navigable : **1 128 nœuds**, **4 782 arêtes**, **9 couches**, visite guidée en **12 étapes**.
> Graphe **régénéré le 2026-07-06** (post-audit 2026-07, contenu en français), avec mise à jour automatique
> activée (`autoUpdate`) ; procédure de reconstruction : [`docs/BUILD.md`](docs/BUILD.md).

### Historique des audits

| Date | Portée | Résultats clés |
|---|---|---|
| **2026-06** | Audit complet, refactorisation et optimisation — [Claude Fable 5](https://www.anthropic.com/news/claude-fable-5-mythos-5) (Anthropic), effort Max | Temps de calcul geomean **−12 %** (FastDoubling/10M −15,3 %), allocations **~−70 %** B/op à F(10M), couverture 88,9 % → **95,0 %**, une data race réelle corrigée, 1 019 affirmations documentaires vérifiées — [`CHANGELOG.md`](CHANGELOG.md) |
| **2026-06-24** | Revue Go exhaustive — Claude Opus 4.8, orchestration multi-agents, vérification adversariale | Trois défauts de correctness corrigés (panic de la récursion FFT parallèle re-propagée au lieu de crasher — ADR-0002 ; `--algo all --quiet` ne masque plus une divergence — exit 3 ; messages TUI obsolètes ignorés après *Restart*), durcissements (`GOMEMLIMIT`, troncature UTF-8, complétion shell, codes de sortie), purge de code mort. Chemin critique validé sans régression (`benchstat`) — [`CHANGELOG.md`](CHANGELOG.md) |
| **2026-07** | Audit exhaustif multi-agents (8 dimensions) — Claude Opus 4.8 pilote, exécuteurs Sonnet, vérification adversariale — **exécuté** (6 phases, ~30 commits) | ~40 findings corrigés (dont panic pointeur/tri, `--gc-control` inerte, complétions shell, data race spinner, correctifs calibration + re-validation profil forgé SEC-01, `bigfft` alloc pool non initialisée + ordonnancement panic FFT-02), ~500 LOC de code mort retiré, couverture 95,0 % → **95,2 %**, build `gmp` réparé, `benchstat` global **sans régression réelle** (chemin critique sous le seuil de 5 %) ; **FIB-05** (réduction ×15 de l'arène) **rejetée sur preuve** (+18 à +34 % à F(10M)) → [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md). Gate final vert — [`audit.md`](audit.md) / [`auditPlan.md`](auditPlan.md) / [`CHANGELOG.md`](CHANGELOG.md) |

---

## Table des matières

1. [Démarrage rapide](#démarrage-rapide)
2. [Fonctionnalités](#fonctionnalités)
3. [Architecture](#architecture)
4. [Performance](#performance)
5. [Guide d'utilisation](#guide-dutilisation)
6. [Configuration](#configuration)
7. [Développement et tests](#développement-et-tests)
8. [Dépannage](#dépannage)
9. [Contribution et licence](#contribution-et-licence)

---

## Démarrage rapide

Prérequis : **Go 1.26.0+**. Toutes les commandes ci-dessous ont été exécutées telles quelles le 2026-06-10
(hôte Windows 11 ; sous Windows, le binaire produit est `fibcalc.exe`).

```bash
git clone https://github.com/agbruneau/FibGo.git
cd FibGo
go build -o fibcalc ./cmd/fibcalc
./fibcalc -n 1000000 -algo fast        # F(1 000 000) : 4 ms, 694 241 bits
./fibcalc -n 100 -c                    # petit indice, valeur affichée
./fibcalc -tui -n 5000000 -algo all    # dashboard TUI interactif (terminal requis)
```

Avec GNU make (Linux/macOS/WSL — absent par défaut sous Windows, voir les équivalents `go` plus bas) :

```bash
make build    # ./build/fibcalc (utilise le profil PGO s'il est présent)
make all      # clean + build + test
```

---

## Fonctionnalités

### Algorithmes

| Algorithme | Complexité | Notes |
|---|---|---|
| **Fast Doubling** (défaut) | O(log n) × M(n) | Identité F(2k) = F(k)·(2F(k+1) − F(k)) ; pooling état+arène+scratch FFT |
| **Exponentiation matricielle** | O(log n) × M(n) | Variante **Strassen-Winograd** (7 multiplications, 15 add/sub) pour les grandes matrices |
| **FFT (Schönhage-Strassen)** | O(n log n) | Bascule automatique au-delà de ~500 000 bits (seuil adaptatif) |
| **GMP** (tag de build `gmp`) | — | Backend GNU MP, nécessite CGO + libgmp |

Détails mathématiques : [`docs/algorithms/`](docs/algorithms/) — [FAST_DOUBLING](docs/algorithms/FAST_DOUBLING.md),
[MATRIX](docs/algorithms/MATRIX.md), [FFT](docs/algorithms/FFT.md), [GMP](docs/algorithms/GMP.md),
[COMPARISON](docs/algorithms/COMPARISON.md).

### Ingénierie de performance

- **Pooling agressif** : `sync.Pool` pour `big.Int` ; `CalculationState` possède son arène **et** son scratch FFT
  (bump allocator acquis une fois par calcul). Un **slot GC-immune par calculateur** conserve l'état entre les
  appels (le GC forcé post-calcul purge les `sync.Pool`) — source des gains −12 à −15 % mesurés sur F(10M).
- **Allocateur bump** O(1) sans fragmentation pour les tampons FFT.
- **GC désactivé** pendant les grands calculs (N ≥ 1M), panic-safe (`WithGC`), refcount concurrent (ADR-0005).
- **Parallélisme adaptatif** : produits pointwise et butterflies FFT répartis sur les cœurs (sémaphore global,
  acquisition non bloquante) — mesuré −23 à −35 % sur F(10M) en 2026-06.
- **Seuils dynamiques** avec hystérésis (parallèle/FFT/Strassen) ajustés sur métriques observées (ADR-0001).
- **Cache LRU de transformées FFT** — bénéficie aux chemins qui le consultent (`bigfft.Mul/Sqr` directs,
  stratégie `fft`) ; le mode Fast Doubling par défaut ne le consulte pas (mesure 2026-06-10 : zéro hit).
- **Auto-calibration** (`-calibrate`) avec profil persistant et clé matérielle d'invalidation.
- **PGO** : `make build-pgo`.
- **Mode `-last-digits K`** : derniers K chiffres décimaux en mémoire O(K), pour des N arbitrairement grands.

### Interfaces

- **CLI moderne** : spinners, ETA, thèmes couleur, support `NO_COLOR`, sortie `-machine` pour scripts.
- **TUI interactif** (`-tui`) : dashboard type btop (Bubble Tea) — graphe de progression, sparklines, métriques
  mémoire ([`docs/TUI_GUIDE.md`](docs/TUI_GUIDE.md)).
- **Complétion shell** : bash, zsh, fish, PowerShell (`fibcalc -completion <shell>`), générateurs avec échappement
  systématique (vecteur d'injection fermé, audit F-014).

---

## Architecture

Clean Architecture en quatre couches — `cmd → app → orchestration → fibonacci/bigfft → config/errors`,
étanchéité gardée par `internal/arch_test.go`. Source de vérité : [`docs/architecture/`](docs/architecture/)
(diagrammes C4, [graphe de dépendances](docs/architecture/dependency-graph.mermaid)) et le
[dashboard interactif](https://agbruneau.github.io/FibGo/dashboard/) (généré, servi par GitHub Pages depuis
[`docs/dashboard/`](docs/dashboard/)).

| Package | Responsabilité |
|---|---|
| `cmd/fibcalc` | Point d'entrée CLI |
| `cmd/generate-golden` | Générateur du golden (oracle indépendant : `math/big` itératif, zéro import interne — ne valide pas la lib par elle-même) |
| `internal/app` | Cycle de vie, dispatch, version |
| `internal/fibonacci` | Algorithmes, frameworks, stratégies ; `memory/` (arène, GC, budget), `threshold/` (seuils dynamiques) |
| `internal/bigfft` | Schönhage-Strassen sur anneaux de Fermat, bump allocator, cache LRU |
| `internal/orchestration` | Exécution concurrente (`errgroup`), agrégation, sélection des calculateurs |
| `internal/calibration` | Calibration adaptative au matériel, micro-benchmarks, profils |
| `internal/cli` / `internal/tui` | Couches de présentation (`ProgressReporter` / `ResultPresenter` partagés) ; sous-package `cli/completion` (génération complétion shell) |
| `internal/config` | Parsing flags + variables d'environnement, estimation des seuils |
| `internal/progress` | Pattern observer (chemin de production : `Freeze`) |
| `internal/{errors,format,metrics,parallel,ui,testutil}` | Packages de support (feuilles) |

---

## Performance

Mesures du **2026-06-10** (Intel Core Ultra 9 275HX, 24 threads, Windows 11, Go 1.26.4 ; `benchstat` n=6,
détail dans [`CHANGELOG.md`](CHANGELOG.md)) :

| N | Fast Doubling | Matrix Exp. | FFT-Based | Chiffres décimaux |
|---|---|---|---|---|
| 1 000 000 | 3,4 ms | 5,7 ms | 4,7 ms | 208 988 |
| 10 000 000 | **28,2 ms** | 27,9 ms | 30,8 ms | 2 089 877 |
| 100 000 000 | ~0,2 s (calcul seul, 2026-06-09¹) | — | — | 20 898 764 |

¹ binaire `-algo fast` sans conversion décimale, médiane de 4 runs.

**Choix d'algorithme** : `fast` pour l'usage général (le plus régulier) ; `matrix` pour la pédagogie et la
validation croisée ; `fft` devient compétitif sur les très grands N. Méthodologie, tuning et suivi de
non-régression : [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md). L'audit exhaustif 2026-07 a confirmé
l'absence de régression réelle sur le chemin critique (`benchstat` global sous le seuil de 5 % —
Directive #1) ; les chiffres ci-dessus restent la mesure de référence.

---

## Guide d'utilisation

```text
fibcalc [flags]
```

| Flag | Raccourci | Défaut | Description |
|---|---|---|---|
| `-n` | | 100 000 000 | Indice Fibonacci |
| `-algo` | | `all` | `fast`, `matrix`, `fft` ou `all` (comparaison) |
| `-calculate` | `-c` | `false` | Affiche la valeur calculée |
| `-verbose` | `-v` | `false` | Affiche la valeur complète |
| `-details` | `-d` | `false` | Détails de performance et métadonnées |
| `-output` | `-o` | | Écrit le résultat dans un fichier |
| `-quiet` | `-q` | `false` | Sortie minimale (scripts) |
| `-machine` | | `false` | Sortie machine (sans ANSI) |
| `-tui` | | `false` | Dashboard TUI interactif |
| `-last-digits` | | `0` | Derniers K chiffres décimaux (mémoire O(K)) |
| `-memory-limit` | | | Budget mémoire (ex. `8G`), estimation préalable |
| `-gc-control` | | `auto` | GC pendant le calcul : `auto`, `aggressive`, `disabled` |
| `-timeout` | | `5m` | Durée maximale du calcul |
| `-threshold` / `-fft-threshold` / `-strassen-threshold` | | `0` (auto) | Seuils en bits (0 = estimation adaptative) |
| `-calibrate` / `-auto-calibrate` | | `false` | Calibration des seuils pour cet hôte |
| `-calibration-profile` | | | Chemin du profil de calibration |
| `-completion` | | | Script de complétion (`bash`, `zsh`, `fish`, `powershell`) |
| `-version` | | | Informations de version |

Exemples (exécutés le 2026-06-10) :

```bash
./fibcalc -n 10000000 -algo all -d                  # compare les trois algorithmes
./fibcalc -n 100000000 -last-digits 10 -q -machine  # → 7760546875
./fibcalc -n 1000000000 -memory-limit 8G            # validation mémoire préalable
./fibcalc -calibrate                                # calibre les seuils pour cet hôte
./fibcalc -completion bash > fibcalc.bash           # complétion shell
```

---

## Configuration

Les variables d'environnement surchargent les défauts (les flags CLI gagnent toujours).
Priorité : **flags CLI > variables d'environnement > estimation adaptative > défauts statiques**.
Liste complète : [`.env.example`](.env.example). Principales : `FIBCALC_N`, `FIBCALC_ALGO`, `FIBCALC_TIMEOUT`,
`FIBCALC_THRESHOLD`, `FIBCALC_FFT_THRESHOLD`, `FIBCALC_STRASSEN_THRESHOLD`, `FIBCALC_TUI`, `FIBCALC_TUI_THEME`,
`FIBCALC_CALIBRATION_PROFILE`, `FIBCALC_PROFILE_MAX_AGE` (168h), `FIBCALC_MEMORY_LIMIT`, et
[`NO_COLOR`](https://no-color.org/).

---

## Développement et tests

- **Pas de CI distante — décision assumée** : la rigueur repose sur la discipline locale outillée
  (gate `scripts/check.ps1` / `scripts/check.sh`, plancher de couverture, baselines de benchmark).
- **Couverture** : plancher garanti **80 %** via `make coverage-check` ; la couverture réelle le
  dépasse confortablement (mesure ponctuelle avec `go test ./... -cover`, non figée — directive A5-04).
- **Golden tests immuables** : `internal/fibonacci/testdata/fibonacci_golden.json` est l'oracle de
  non-régression (étendu à F(50k/100k/200k) sous ADR-0004 §B5) — aucune mise à jour sans ADR.
- **Race detector** : exige CGO/gcc — indisponible sous Windows sans gcc ; sur cet environnement les passes
  `-race` complètes se font via **WSL** (`wsl go test -race ./...`) — suite intégralement verte au 2026-06-21
  après dé-flake d'un test de pool `bigfft` qui échouait par intermittence sous `-race` (assertion d'identité
  `sync.Pool` non contractuelle retirée ; cf. table des invariants de [`CLAUDE.md`](CLAUDE.md)).
- Environnement reproductible : [`.devcontainer/`](.devcontainer/devcontainer.json) (Go + CGO + libgmp +
  benchstat) ou [`Dockerfile`](Dockerfile) multi-étages.
- Décisions architecturales : [`docs/adr/`](docs/adr/) (0001–0009). Guide agents IA : [`CLAUDE.md`](CLAUDE.md).
  Dernier audit : [`audit.md`](audit.md) / [`auditPlan.md`](auditPlan.md).

Commandes principales (équivalents `go` pour Windows sans GNU make) :

```bash
make all             # clean + build + test     (équiv. : go build ./... && go test ./...)
make test            # go test -v -race -cover ./...   (CGO requis ; Linux/macOS/WSL)
make test-win        # go test -v -cover ./...         (Windows sans gcc, sans -race)
make lint            # golangci-lint run ./...  (24 linters)
make coverage        # rapport HTML            (équiv. : go test ./... -coverprofile=coverage.out)
make benchmark       # benchmarks fibonacci    (équiv. : go test -bench=BenchmarkFibonacci -benchmem -run=^$ ./internal/fibonacci/)
make bench-baseline  # rafraîchit la baseline de non-régression docs/audits/
make build-pgo       # build avec PGO
make build-all       # cross-compilation linux/windows/darwin (amd64 + arm64)
make stats           # décompte canonique packages/LOC
```

Stratégie de test (table-driven, `t.Parallel()`, doubles `fibonaccitest`, fuzzing, golden, property-based) :
[`docs/TESTING.md`](docs/TESTING.md). Portabilité (matrice OS/arch, fallbacks) :
[`docs/PORTABILITY.md`](docs/PORTABILITY.md). Build avancé (PGO, cross-compilation, Docker) :
[`docs/BUILD.md`](docs/BUILD.md).

---

## Dépannage

| Symptôme | Cause / remède |
|---|---|
| `-race` échoue : « cgo: C compiler not found » | Le race detector exige gcc/clang. Sous Windows : WSL (`wsl go test -race ./...`) ou `make test-win` (sans race). |
| `go test -bench=.` ne lance rien sous PowerShell | Quirk de parsing PowerShell : utiliser `-bench=BenchmarkFibonacci` (préfixe explicite). |
| Build tag `gmp` : « gmp.h: No such file » | Installer les en-têtes : `sudo apt-get install libgmp-dev` (Linux/WSL). |
| Le TUI ne se lance pas | `-tui` exige un terminal interactif (TTY) ; indisponible dans les pipes/CI. |
| Calcul interrompu à 5 minutes | Défaut `-timeout 5m` — augmenter, p. ex. `-timeout 30m`. |

---

## Contribution et licence

- Changements notables : [`CHANGELOG.md`](CHANGELOG.md) (format Keep-a-Changelog, SemVer).
- Workflow de contribution : [`CONTRIBUTING.md`](CONTRIBUTING.md) — test rouge → fix → vert,
  validation locale complète avant chaque commit (directive 8 de [`CLAUDE.md`](CLAUDE.md)).
- Licence : **Apache 2.0** — voir [`LICENSE`](LICENSE).

### Remerciements

Architecture et algorithmique inspirées de la littérature classique (Schönhage-Strassen, Strassen-Winograd,
fast doubling) ; outillage : Go, Bubble Tea, benchstat, golangci-lint. Audits, refactorisation et optimisation
2026 réalisés avec [Claude Fable 5](https://www.anthropic.com/news/claude-fable-5-mythos-5) et Claude Opus 4.8
(Anthropic) ; l'audit exhaustif 2026-07 (~40 findings corrigés, exécution complète) a été mené en
orchestration multi-agents — Claude Opus 4.8 en pilote, exécuteurs Claude Sonnet.
