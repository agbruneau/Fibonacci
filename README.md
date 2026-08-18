# FibCalc — Calculateur Fibonacci haute performance

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/tag/agbruneau/FibGo?style=for-the-badge&label=Release&color=2ea44f)](CHANGELOG.md)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge&logo=apache)](LICENSE)
![Status](https://img.shields.io/badge/Status-Prototype_acad%C3%A9mique-orange?style=for-the-badge)

Ce dépôt est un **laboratoire d'expérimentation algorithmique et d'ingénierie logicielle**
(*computational sandbox*) : un bac à sable où l'on pousse un problème volontairement simple — calculer F(n) — jusqu'à ses limites, pour y expérimenter des techniques réelles et **mesurer** ce qu'elles valent.
Le nombre de Fibonacci n'est pas la finalité; c'est le banc d'essai. Il a l'avantage d'avoir une réponse exactement vérifiable, de se calculer par plusieurs algorithmes comparables entre eux, et de devenir arbitrairement coûteux quand n grandit — tout écart de conception se voit donc au chronomètre et à la mémoire, sans place pour l'opinion.

Ce qu'on y expérimente : algorithmique (Fast Doubling, exponentiation matricielle Strassen-Winograd, multiplication FFT Schönhage-Strassen), ingénierie de performance (pooling, allocateur bump, contrôle du GC, parallélisme adaptatif, PGO, auto-calibration), et méthode logicielle (Clean Architecture, tests golden et property-based, ADR, gate de qualité local). Toute affirmation chiffrée doit venir d'un artefact de mesure du dépôt; ce qui n'a pas été réexécuté est signalé comme tel. **FibCalc**, le binaire qui en sort, calcule des nombres de Fibonacci arbitrairement grands à très haute vitesse. Écrit en Go; gère des indices de plusieurs centaines de millions.

### Historique des audits et jalons

| Date | Portée | Résultats clés |
|---|---|---|
| **2026-06** | Audit complet, refactorisation et optimisation — [Claude Fable 5](https://www.anthropic.com/news/claude-fable-5-mythos-5) (Anthropic), effort Max | Temps de calcul geomean **−12 %** (FastDoubling/10M −15,3 %), allocations **~−70 %** B/op à F(10M), couverture 88,9 % → **95,0 %**, une data race réelle corrigée — [`CHANGELOG.md`](CHANGELOG.md) |
| **2026-06-24** | Revue Go exhaustive — Claude Opus 4.8, orchestration multi-agents, vérification adversariale | Trois défauts de correctness corrigés (panic de la récursion FFT parallèle re-propagée au lieu de crasher — ADR-0002 ; `--algo all --quiet` ne masque plus une divergence — exit 3 ; messages TUI obsolètes ignorés après *Restart*), durcissements (`GOMEMLIMIT`, troncature UTF-8, complétion shell, codes de sortie), purge de code mort. Chemin critique validé sans régression (`benchstat`) — [`CHANGELOG.md`](CHANGELOG.md) |
| **2026-07** | Audit exhaustif multi-agents (8 dimensions) — Claude Opus 4.8 pilote, exécuteurs Sonnet, vérification adversariale — **exécuté** (6 phases, ~30 commits) | ~40 findings corrigés (dont panic pointeur/tri, `--gc-control` inerte, complétions shell, data race spinner, correctifs calibration + re-validation profil forgé SEC-01, `bigfft` alloc pool non initialisée + ordonnancement panic FFT-02), ~500 LOC de code mort retiré, couverture 95,0 % → **95,2 %**, build `gmp` réparé, `benchstat` global **sans régression réelle**. FIB-05 (réduction du multiplicateur d'arène) initialement **rejetée sur preuve Ryzen** (+18 à +34 % à F(10M)) → [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) / [`CHANGELOG.md`](CHANGELOG.md) |
| **2026-07-07** | Suivi post-audit — exécution des 5 tâches à plus fort levier + **release v4.0.0** | Tag `v4.0.0` (première coupe CHANGELOG depuis 1.0.0) ; backend **GMP branché au gate local** (`check.sh` étape 3b, libgmp dans WSL) ; profil **PGO régénéré** ; **balayage complet du multiplicateur d'arène** (protocole ADR-0009 R4) → **×15 → ×10 adopté** : mémoire FFT 10M **−16 % B/op** à coût CPU nul, confirmé en ordre inversé (addendum [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md)) |
| **2026-08-07** | Audit qualité et documentation — Claude Opus 5, boucle bâtisseur/critique sur cinq tours (source Go entière, 30 documents markdown, 11 diagrammes Mermaid) | Outillage **mesuré, non affirmé** : `golangci-lint` **152 → 0**, `gosec` **19 → 0**, `gofmt -l` **1 → 0**, build/vet/test verts de bout en bout — **sans desserrer les outils** (`.golangci.yml` inchangé hors commentaires, `//nolint` stable à 4, `#nosec` **22 → 13** par retrait des sites, non par annotation). Tests **gagnés** : 877 → **879** fonctions, 408 → **410** sous-tests, 2 291 → **2 305** assertions (⚠ la même ligne affirmait « zéro `t.Skip` » : relevé du 2026-08-10, l'arbre en porte **17 dans 12 fichiers**, tous des gardes `-short` ou d'architecture — aucun test désactivé sans condition). Documentation confrontée à la source, une commande par affirmation ; **121 → 3** renvois `fichier:ligne` vers du Go, convertis en ancres de symbole. **Deux comportements documentés et délibérément inchangés** : un profil de calibration valide écrase les trois seuils explicites (⚠ tous les documents affirmaient l'inverse — désormais épinglé par `TestNewCachedProfileOverridesExplicitFlags`) et `GetDefaultProfilePath` retombe sur un nom relatif si `os.UserHomeDir` échoue. Chiffre de performance sans artefact : **retiré**, non reformulé — [`CHANGELOG.md`](CHANGELOG.md) |

⚠ **Deux limites déclarées par la passe du 2026-08-07, et non corrigées depuis** : `-race` n'a pas pu
tourner sur l'hôte, de sorte que ses conclusions de concurrence sont **statiques, non mesurées** ;
et l'arbre **ne compile pas pour une cible 32 bits** (`maxReasonableWords` déborde un `int` 32 bits).
`TestStateBump_PinnedAcrossCachedCalls` est *flaky* au même taux sur le commit parent (un test
antérieur laisse une arène surdimensionnée dans le pool d'état global).

☑ **Gate rejoué le 2026-08-10 sur l'arbre courant** (Windows 11, `go1.26.5`, `golangci-lint v1.64.8`,
`gosec dev`), après correction du `behaviour` → `behavior` de `internal/ui/themes.go:186` qui faisait
sortir `golangci-lint` à 1 lors du relevé précédent du même jour — **quatre sur quatre à zéro** :
`gofmt -l .` sans sortie, `go vet ./...` sortie 0, `golangci-lint run ./...` sortie 0,
`gosec ./...` 0 issue sur 122 fichiers.
*Un décompte publié sans avoir été réexécuté n'est pas une mesure.*

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

Prérequis : **Go 1.26.0+**. Toutes les commandes ci-dessous ont été exécutées telles quelles le 2026-07-07
(hôte Windows 11 + WSL2). Sous Windows natif, `-o fibcalc` produit un fichier **sans extension** que le
shell refuse d'exécuter : écrire `go build -o fibcalc.exe ./cmd/fibcalc` puis `.\fibcalc.exe`.

```bash
git clone https://github.com/agbruneau/FibGo.git
cd FibGo
go build -o fibcalc ./cmd/fibcalc
./fibcalc -n 1000000 -algo fast        # F(1 000 000) : 5 ms, 694 241 bits
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
| **GMP** (tag de build `gmp`) | — | Backend GNU MP (CGO + libgmp) ; `scripts/check.sh` étape 3b le compile et le teste **si** les en-têtes libgmp sont présentes sur l'hôte, sinon l'étape est sautée (`check.ps1` n'a pas d'équivalent) |

Détails mathématiques : [`docs/algorithms/`](docs/algorithms/) — [FAST_DOUBLING](docs/algorithms/FAST_DOUBLING.md),
[MATRIX](docs/algorithms/MATRIX.md), [FFT](docs/algorithms/FFT.md), [GMP](docs/algorithms/GMP.md),
[COMPARISON](docs/algorithms/COMPARISON.md) ; internes d'implémentation :
[BIGFFT](docs/algorithms/BIGFFT.md) (`internal/bigfft`) et
[PROGRESS_BAR_ALGORITHM](docs/algorithms/PROGRESS_BAR_ALGORITHM.md) (progression des boucles O(log n)).

### Ingénierie de performance

- **Pooling agressif** : `sync.Pool` pour `big.Int` ; `CalculationState` possède son arène **et** son scratch FFT
  (bump allocator acquis une fois par calcul). Un **slot GC-immune par calculateur** conserve l'état entre les
  appels (le GC forcé post-calcul purge les `sync.Pool`) — source des gains −12 à −15 % mesurés sur F(10M).
- **Arène dimensionnée ×10** : sur-dimensionnement mesuré par balayage complet (ADR-0009 R4, addendum
  2026-07-07) — mémoire FFT 10M −16 % B/op vs l'ancien ×15, à coût CPU nul ; la valeur optimale est
  microarchitecture-dépendante et gardée par le protocole de re-balayage documenté.
- **Allocateur bump** O(1) sans fragmentation pour les tampons FFT.
- **GC désactivé** pendant les grands calculs (N ≥ 1M), panic-safe (`WithGC`), refcount concurrent (ADR-0005).
- **Parallélisme adaptatif** : produits pointwise et butterflies FFT répartis sur les cœurs (sémaphore global,
  acquisition non bloquante) — **−14 % à −35 %** sur F(10M) selon l'algorithme (2026-06-09, chiffres
  consignés dans [`CHANGELOG.md`](CHANGELOG.md) ; le rapport de mesure a été purgé, pas de sortie archivée).
- **Seuils dynamiques** avec hystérésis (parallèle/FFT/Strassen) ajustés sur métriques observées (ADR-0001).
- **Cache LRU de transformées FFT** — bénéficie aux chemins qui le consultent (`bigfft.Mul/Sqr` directs,
  stratégie `fft`) ; le mode Fast Doubling par défaut ne le consulte pas (mesure 2026-06-10 : zéro hit).
- **Auto-calibration** (`-calibrate`) avec profil persistant et clé matérielle d'invalidation
  ([`docs/CALIBRATION.md`](docs/CALIBRATION.md)).
- **PGO** : `make build-pgo` (profil régénéré le 2026-07-07).
- **Mode `-last-digits K`** : derniers K chiffres décimaux en mémoire O(K), pour des N arbitrairement grands.

### Interfaces

- **CLI moderne** : spinners, ETA, thèmes couleur, support `NO_COLOR`, sortie `-machine` pour scripts.
- **TUI interactif** (`-tui`) : dashboard type btop (Bubble Tea) — graphe de progression, sparklines, métriques
  mémoire ([`docs/TUI_GUIDE.md`](docs/TUI_GUIDE.md)).
- **Complétion shell** : bash, zsh, fish, PowerShell (`fibcalc -completion <shell>`), générateurs avec échappement
  systématique (vecteur d'injection fermé, audit F-014).

---

## Architecture

Clean Architecture — `cmd → app → orchestration → fibonacci → bigfft`, `internal/config` étant un *frère* de
`orchestration` et non une couche sous `fibonacci` (commentaire de paquet de `internal/arch_test.go`) ; `internal/bigfft` est le noyau
et n'importe aucun package interne. Étanchéité gardée par `internal/arch_test.go`
(cinq règles d'import montant interdit — six arêtes, la dernière en couvrant deux).
Vue d'ensemble : [`docs/ARCH.md`](docs/ARCH.md) ; référence détaillée :
[`docs/architecture/`](docs/architecture/) (diagrammes C4,
[graphe de dépendances](docs/architecture/dependency-graph.mermaid)).

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
| `internal/{errors,format,metrics,ui,testutil}` | Packages de support (feuilles) |
| `test/e2e` | Tests bout-en-bout du binaire CLI (hors `internal/`) |

---

## Performance

Médianes recalculées à partir du **seul artefact de mesure du dépôt**,
[`docs/audits/bench-baseline.txt`](docs/audits/bench-baseline.txt) (linux/amd64, 24 threads,
`-count=5 -benchtime=1x`, estampille `baseline-2026-07-07`, arène ×10) :

| N | Fast Doubling | Matrix Exp. | FFT-Based | Chiffres décimaux |
|---|---|---|---|---|
| 1 000 000 | **3,15 ms** / 1,32 Mo par op | 6,03 ms / 6,33 Mo | 5,13 ms / 5,38 Mo | 208 988 |
| 10 000 000 | **23,87 ms** / 17,38 Mo par op | 30,84 ms / 92,25 Mo | 29,08 ms / 30,88 Mo | 2 089 877 |

`-benchtime=1x` : une itération par échantillon, rodage compris. Aucune autre valeur de N n'est
mesurée dans le dépôt. Pour F(100 000 000), le seul chiffre traçable est le **0,204 s** de calcul seul
(sans conversion décimale) consigné dans [`CHANGELOG.md`](CHANGELOG.md) au 2026-06-09 ; il n'a pas
d'artefact de sortie archivé.

Côté mémoire, l'adoption du multiplicateur d'arène ×10 (2026-07-07) réduit les B/op FFT à F(10M) de **−16 %**
vs ×15, allocations inchangées — gain confirmé en ordre d'exécution inversé (addendum
[ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md)).

**Choix d'algorithme** : `fast` pour l'usage général (le plus régulier) ; `matrix` pour la pédagogie et la
validation croisée ; `fft` est plus lent que `fast` aux deux seules tailles mesurées (F(1M) et F(10M)) —
l'idée qu'il devienne compétitif au-delà est une hypothèse que le dépôt ne teste pas. Méthodologie, tuning et suivi de
non-régression : [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md) ; baseline du gate perf :
`docs/audits/bench-baseline.txt` (régénérée le 2026-07-07).

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
| `-version` | `-V` | | Informations de version |

Exemples :

```bash
./fibcalc -n 10000000 -algo all -d                  # compare les trois algorithmes
./fibcalc -n 100000000 -last-digits 10 -q -machine  # → 7760546875
./fibcalc -n 1000000000 -memory-limit 8G            # validation mémoire préalable
./fibcalc -calibrate                                # calibre les seuils pour cet hôte
./fibcalc -completion bash > fibcalc.bash           # complétion shell
```

---

## Configuration

Une variable `FIBCALC_*` n'est lue que si le flag correspondant est absent de la ligne de commande
(`internal/config/env.go:applyEnvOverrides`). Priorité générale :
**flags CLI > variables d'environnement > défauts statiques**.

> **Exception — les trois seuils.** Un profil de calibration en cache **valide** écrase `--threshold`,
> `--fft-threshold` et `--strassen-threshold`, ainsi que leurs variables d'environnement.
> `app.New` appelle `calibration.LoadCachedCalibration` *après* `ParseConfig`
> (`internal/app/app.go:New`) et celle-ci réécrit les trois champs sans consulter ni le flag ni la variable
> (`internal/calibration/calibration.go:LoadCachedCalibration`). Le profil est lu à `--calibration-profile`, ou à
> `~/.fibcalc_calibration.json` par défaut ; il n'est retenu que si `IsValid()` passe (version de profil,
> nombre de CPU, `GOARCH`, taille de mot, clé heuristique SIMD) et si la config résultante valide encore.
> Sans profil valide, `ApplyAdaptiveThresholds` ne remplit que les seuils laissés à 0 (estimation adaptative,
> `internal/config/thresholds.go`), les autres gardent la valeur du flag ou de la variable. Pour qu'un seuil
> explicite soit respecté, supprimer le profil ou pointer `--calibration-profile` sur un chemin inexistant.

Liste complète : [`.env.example`](.env.example). Principales : `FIBCALC_N`, `FIBCALC_ALGO`, `FIBCALC_TIMEOUT`,
`FIBCALC_THRESHOLD`, `FIBCALC_FFT_THRESHOLD`, `FIBCALC_STRASSEN_THRESHOLD`, `FIBCALC_LAST_DIGITS`, `FIBCALC_TUI`, `FIBCALC_TUI_THEME`,
`FIBCALC_CALIBRATION_PROFILE`, `FIBCALC_PROFILE_MAX_AGE` (168h), `FIBCALC_MEMORY_LIMIT`, `FIBCALC_GC_CONTROL`, et
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
  `-race` complètes se font via **WSL** (`wsl go test -race ./...`). Les scripts shell sont épinglés en LF
  (`.gitattributes`) pour rester exécutables côté WSL.
- **Backend GMP sous gate** : depuis 2026-07, `scripts/check.sh` compile et teste `-tags gmp -race`
  (étape 3b, **dure** quand les headers libgmp sont présents, SKIP sinon) — le tag ne peut plus casser
  silencieusement. Validation manuelle : `wsl go test -tags gmp -race ./internal/fibonacci/`.
- Environnement reproductible : [`.devcontainer/`](.devcontainer/devcontainer.json) (Go + CGO + libgmp +
  benchstat) ou [`Dockerfile`](Dockerfile) multi-étages.
- Décisions architecturales : [`docs/adr/`](docs/adr/) (0001–0009, plus `0000-template.md`).
  Audit de 2026-07 : exécuté puis purgé — voir [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) et [`CHANGELOG.md`](CHANGELOG.md).
  ⚠ **Dernier audit : 2026-08-07** (qualité et documentation, `golangci-lint`/`gosec`/`gofmt` à zéro) —
  **il n'a pas d'ADR** : il ne tranche aucune décision d'architecture, et son journal de boucle
  (`gauntlet-log.md`) a été **retiré de l'arbre le 2026-08-08** ; il se relit à l'historique git.
  Le tableau « Historique des audits et jalons » en tête de ce fichier en porte le détail.

Commandes principales (équivalents `go` pour Windows sans GNU make) :

```bash
make all             # clean + build + test     (équiv. : go build ./... && go test ./...)
make test            # go test -v -race -cover ./...   (CGO requis ; Linux/macOS/WSL)
make test-win        # go test -v -cover ./...         (Windows sans gcc, sans -race)
make lint            # golangci-lint run ./...  (24 linters)
make coverage        # rapport HTML            (équiv. : go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html)
make benchmark       # benchmarks fibonacci    (équiv. : go test -bench=. -benchmem ./internal/fibonacci/)
make bench-baseline  # rafraîchit la baseline de non-régression docs/audits/
make build-pgo       # build avec PGO
make build-all       # cross-compilation linux/windows/darwin (amd64 + arm64)
make stats           # décompte canonique packages/LOC
```

Stratégie de test (table-driven, `t.Parallel()`, doubles de test, fuzzing, golden, property-based) :
[`docs/TESTING.md`](docs/TESTING.md). Portabilité (matrice OS/arch, fallbacks) :
[`docs/PORTABILITY.md`](docs/PORTABILITY.md). Build avancé (PGO, cross-compilation, Docker) :
[`docs/BUILD.md`](docs/BUILD.md).

---

## Dépannage

| Symptôme | Cause / remède |
|---|---|
| `-race` échoue : « cgo: C compiler not found » | Le race detector exige gcc/clang. Sous Windows : WSL (`wsl go test -race ./...`) ou `make test-win` (sans race). |
| `go test -bench=.` ne lance rien sous PowerShell | Quirk de parsing PowerShell : utiliser `-bench=BenchmarkFibonacci` (préfixe explicite). |
| Build tag `gmp` : « gmp.h: No such file » | Installer les en-têtes : `sudo apt-get install libgmp-dev` (Linux/WSL). Sans eux, l'étape 3b de `check.sh` est proprement sautée (SKIP). |
| `bash scripts/check.sh` : « syntax error near `$'{\r'` » | Fins de ligne CRLF (checkout antérieur au pin `*.sh eol=lf`) : `git checkout -- scripts/check.sh` ou `sed -i 's/\r$//' scripts/check.sh`. |
| Le TUI ne se lance pas | `-tui` exige un terminal interactif (TTY) ; indisponible dans les pipes/CI. |
| Calcul interrompu à 5 minutes | Défaut `-timeout 5m` — augmenter, p. ex. `-timeout 30m`. |

---

## Contribution et licence

- Changements notables : [`CHANGELOG.md`](CHANGELOG.md) (format Keep-a-Changelog, SemVer — release courante : `v4.0.0`).
- Workflow de contribution : [`CONTRIBUTING.md`](CONTRIBUTING.md) — test rouge → fix → vert,
  validation locale complète avant chaque commit.
- Licence : **Apache 2.0** — voir [`LICENSE`](LICENSE).

### Remerciements

Architecture et algorithmique inspirées de la littérature classique (Schönhage-Strassen, Strassen-Winograd,
fast doubling) ; outillage : Go, Bubble Tea, benchstat, golangci-lint, gosec. Audits, refactorisation et
optimisation 2026 réalisés avec [Claude Fable 5](https://www.anthropic.com/news/claude-fable-5-mythos-5),
Claude Opus 4.8 et Claude Opus 5 (Anthropic) : audit exhaustif 2026-07 (~40 findings, orchestration
multi-agents — Claude Opus 4.8 pilote, exécuteurs Claude Sonnet), suivi 2026-07-07 (release v4.0.0, gate GMP,
balayage arène ×10 — Claude Fable 5), puis audit qualité et documentation 2026-08-07 (boucle
bâtisseur/critique, lint et gosec à zéro — Claude Opus 5).
