# FibCalc — Calculateur Fibonacci haute performance

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/tag/agbruneau/FibGo?style=for-the-badge&label=Release&color=2ea44f)](CHANGELOG.md)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge&logo=apache)](LICENSE)
![Status](https://img.shields.io/badge/Status-Prototype_acad%C3%A9mique-orange?style=for-the-badge)

Ce dépôt est un **laboratoire d'expérimentation algorithmique et d'ingénierie logicielle**
(*computational sandbox*) : un bac à sable où l'on pousse un problème volontairement simple — calculer F(n) — jusqu'à ses limites, pour y expérimenter des techniques réelles et **mesurer** ce qu'elles valent.
Le nombre de Fibonacci n'est pas la finalité; c'est le banc d'essai. Il a l'avantage d'avoir une réponse exactement vérifiable, de se calculer par plusieurs algorithmes comparables entre eux, et de devenir arbitrairement coûteux quand n grandit — tout écart de conception se voit donc au chronomètre et à la mémoire, sans place pour l'opinion.

Ce qu'on y expérimente : algorithmique (Fast Doubling, exponentiation matricielle Strassen-Winograd, multiplication FFT Schönhage-Strassen), ingénierie de performance (pooling, allocateur bump, contrôle du GC, parallélisme adaptatif, PGO, auto-calibration), et méthode logicielle (Clean Architecture, tests golden et property-based, ADR, gate de qualité local). Toute affirmation chiffrée doit venir d'un artefact de mesure du dépôt; ce qui n'a pas été réexécuté est signalé comme tel. **FibCalc**, le binaire qui en sort, calcule des nombres de Fibonacci arbitrairement grands à très haute vitesse. Écrit en Go; gère des indices de plusieurs centaines de millions.

### Historique

Huit campagnes d'audit entre mai 2026 et septembre 2026, avec pour chacune ce
qui a été mesuré, ce qui a été rejeté et pourquoi :
[`docs/audits/HISTORY.md`](docs/audits/HISTORY.md). Le détail commit par commit
est dans [`CHANGELOG.md`](CHANGELOG.md), les décisions dans
[`docs/adr/`](docs/adr/).

**État vérifié le 2026-09-07** (Windows 11, `go1.27.0`, `golangci-lint v2.13.2`) :
`scripts/check.ps1` vert de bout en bout — build, vet, `go test -race -shuffle=on
-count=1` sur les 22 paquets, lint à **0 finding**, couverture **96,1 %** (plancher
80 %), `govulncheck` sans vulnérabilité. La même séquence tourne en CI sur Ubuntu
et Windows à chaque poussée, avec en plus le backend `gmp`, un build 32 bits et
l'image Docker.

⚠ **Limites déclarées.** Le backend `gmp` ne se compile pas sur cet hôte (pas
d'en-têtes libgmp) ; il est couvert par le job CI `gmp`. Les cibles 32 bits
compilent depuis le 2026-09-07 mais ne sont ni testées ni distribuées
([`docs/PORTABILITY.md`](docs/PORTABILITY.md) §1).

---

## Table des matières

1. [Démarrage rapide](#démarrage-rapide)
2. [Fonctionnalités](#fonctionnalités)
3. [Architecture](#architecture)
4. [Performance](#performance)
5. [Guide d'utilisation](#guide-dutilisation)
6. [Configuration](#configuration)
7. [Développement et tests](#développement-et-tests)
8. [Contribution et licence](#contribution-et-licence)

Dépannage : [`docs/BUILD.md` § Dépannage](docs/BUILD.md#dépannage). Historique des
audits : [`docs/audits/HISTORY.md`](docs/audits/HISTORY.md). Chemin complet d'un
calcul : [`docs/ARCH.md`](docs/ARCH.md).

---

## Démarrage rapide

Prérequis : **Go 1.26.1+** (`go.mod` déclare `go 1.26.1`, sans directive `toolchain`). Sous Windows natif,
`-o fibcalc` produit un fichier **sans extension** que le shell refuse d'exécuter : écrire
`go build -o fibcalc.exe ./cmd/fibcalc` puis `.\fibcalc.exe`.

Le dépôt s'appelle **`agbruneau/Fibonacci`** ; le chemin de module est resté
`github.com/agbruneau/FibGo`, nom d'origine du dépôt. L'écart est délibéré :
changer un chemin de module casse tous les imports existants, et la redirection
GitHub couvre les deux usages — vérifié le 2026-09-07, `git ls-remote` et
`go list -m github.com/agbruneau/FibGo@latest` réussissent tous les deux.

```bash
git clone https://github.com/agbruneau/Fibonacci.git
cd Fibonacci
go build -o fibcalc ./cmd/fibcalc
./fibcalc -n 1000000 -algo fast        # 694 241 bits (la durée dépend de l'hôte, cf. Performance)
./fibcalc -n 100 -c                    # → 354224848179261915075
./fibcalc -tui -n 5000000 -algo all    # dashboard TUI interactif (terminal requis)
```

Les deux premières lignes ont été réexécutées telles quelles le 2026-09-04 (Windows 11 natif,
`go1.27.0`) et rendent les sorties annotées ci-dessus. La troisième ne l'est pas : `-tui` exige un
terminal interactif.

Avec GNU make (Linux/macOS/WSL — absent par défaut sous Windows, voir les équivalents `go` plus bas) :

```bash
make build    # ./build/fibcalc (utilise le profil PGO s'il est présent)
make all      # clean + build + test
```

---

## Fonctionnalités

### Algorithmes

| `-algo` | Coût de F(n) | Notes |
|---|---|---|
| `fast` (défaut) — **Fast Doubling** | O(log n) × M(n) | Identité F(2k) = F(k)·(2F(k+1) − F(k)) ; `AdaptiveStrategy` choisit M pas par pas ; pooling état+arène+scratch FFT |
| `matrix` — **Exponentiation matricielle** | O(log n) × M(n) | Variante **Strassen-Winograd** (7 multiplications, 15 add/sub) pour les grandes matrices ; choisit M lui aussi, mais à un autre point du graphe d'appel |
| `fft` — **FFT-Based Doubling** | O(log n) × M(n), M **toujours** FFT | **Pas un troisième algorithme** : `FFTBasedCalculator.CalculateCore` relance la boucle de `fast` — le même `ExecuteDoublingLoop` — en échangeant `AdaptiveStrategy` contre `FFTOnlyStrategy`, qui ne consulte plus aucun seuil. C'est un banc d'essai du chemin FFT isolé, et il est **plus lent que `fast`** aux deux seules tailles mesurées ([Performance](#performance)) |
| **GMP** (tag de build `gmp`) | — | Backend GNU MP (CGO + libgmp) ; `scripts/check.sh` étape 3b le compile et le teste **si** les en-têtes libgmp sont présentes sur l'hôte, sinon l'étape est sautée (`check.ps1` n'a pas d'équivalent) |

M(n) est le coût d'**une** multiplication de deux nombres de n bits, pas celui de F(n) : `math/big`
(Karatsuba, M(n) ≈ O(n^1,585)) sous le seuil FFT, Schönhage-Strassen (`internal/bigfft`,
M(n) ≈ O(n log n)) au-dessus. Les trois calculateurs font O(log n) multiplications et ne diffèrent
que par la routine qu'ils y branchent et par l'endroit où ils la choisissent — **aucune ligne du
tableau ne domine donc les autres asymptotiquement** : ce qui les sépare tient aux constantes et se
lit au chronomètre
([Performance](#performance)), pas dans la colonne « Coût ». Qui prend quel chemin, à quel moment et
avec quel seuil : [`docs/algorithms/FFT.md` § FFT Routing](docs/algorithms/FFT.md#fft-routing),
description canonique du routage.

Détails mathématiques : [`docs/algorithms/`](docs/algorithms/) — [FAST_DOUBLING](docs/algorithms/FAST_DOUBLING.md),
[MATRIX](docs/algorithms/MATRIX.md), [FFT](docs/algorithms/FFT.md), [GMP](docs/algorithms/GMP.md),
[COMPARISON](docs/algorithms/COMPARISON.md) ; internes d'implémentation :
[BIGFFT](docs/algorithms/BIGFFT.md) (`internal/bigfft`) et
[PROGRESS_BAR_ALGORITHM](docs/algorithms/PROGRESS_BAR_ALGORITHM.md) (progression des boucles O(log n)).

> **Le seuil FFT est une table de constantes, pas une mesure.** `estimateFFTThresholdForHeuristic`
> (`internal/config/thresholds.go`) le fixe au démarrage sur la seule détection du jeu d'instructions :
> 250 000 bits si le mot machine n'a pas 64 bits, sinon **460 000** (AVX-512), **480 000** (AVX2),
> **500 000** par défaut. Aucun chronomètre n'intervient et rien ne s'ajuste en cours de route : deux hôtes
> au même SIMD obtiennent la même valeur. Sur `-algo fast`, la bascule compare ensuite ce seuil à la
> taille de l'**opérande courant** (`FK1.BitLen()`, à chaque tour de la boucle de doublement), jamais
> à `n` ni à la taille du résultat — sur l'hôte de ce dépôt (amd64/AVX2, seuil **480 000 bits**, affiché à chaque exécution non
> silencieuse sous « Optimization thresholds »), `-algo fast -n 400000` plafonne à **138 848 bits**
> d'opérande et `-n 1000000`, la commande du démarrage rapide, à **347 121 bits** : ni l'un ni l'autre
> n'emprunte la FFT, bien que F(1 000 000) fasse 694 241 bits, et sur ce chemin la bascule ne commence
> que vers **n ≈ 1,38 million**. Ce chiffre ne vaut **que pour `fast`** : `matrix` lit le même seuil
> ailleurs dans le graphe d'appel et bascule plus tard (`n ≥ 1 739 980` sur cet hôte), `fft` ne le lit
> pas du tout. Le binaire n'annonce pas non plus qu'un pas est réellement parti en FFT : il
> n'imprime que le seuil. Seul `-auto-calibrate` le mesure — et seulement à défaut d'un profil frais en
> cache ; `-calibrate`, lui, ne mesure que le seuil de parallélisme et recopie la table pour les deux
> autres. Mécanisme complet, chemin par chemin :
> [`docs/algorithms/FFT.md` § FFT Routing](docs/algorithms/FFT.md#fft-routing).

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
- **Seuils dynamiques** avec hystérésis (parallèle/FFT/Strassen) ajustés sur métriques observées —
  **opt-in, désactivé par défaut** (`-dynamic-thresholds`, câblé en 2026-09 ; la mesure à `-count=8` ne
  reproduit pas le gain de 5-6 % qui avait justifié sa conservation, [ADR-0001](docs/adr/0001-dtm-decision.md)).
- **Cache LRU de transformées FFT** — bénéficie aux chemins qui le consultent (`bigfft.Mul/Sqr` directs,
  stratégie `fft`) ; le mode Fast Doubling par défaut ne le consulte pas (mesure 2026-06-10 : zéro hit).
  Borné en **octets** depuis l'audit 2026-09 (M-08) : un plafond en nombre d'entrées n'est pas une borne
  mémoire, la taille d'une entrée croissant linéairement avec n.
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
(sept règles d'import montant interdit — neuf arêtes, les deux dernières en couvrant deux chacune).
Vue d'ensemble : [`docs/ARCH.md`](docs/ARCH.md) ; référence détaillée :
[`docs/architecture/`](docs/architecture/) (diagrammes C4,
[graphe de dépendances](docs/architecture/dependency-graph.md)).

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

## Performance

Médianes recalculées à partir de [`docs/audits/bench-baseline.txt`](docs/audits/bench-baseline.txt)
(linux/amd64, 24 threads, `-count=5 -benchtime=1x`, estampille `baseline-2026-07-07`, arène ×10) —
**seul artefact de débit** du dépôt. Les cinq autres fichiers de [`docs/audits/`](docs/audits/) sont
des A/B ciblés (DTM, cache FFT, memclr des pools, stabilité du micro-benchmark) ou un relevé mémoire :
ils comparent deux variantes dans une même session, ils ne mesurent pas un débit de référence.

| N | Fast Doubling | Matrix Exp. | FFT-Based | Chiffres décimaux |
|---|---|---|---|---|
| 1 000 000 | **3,15 ms** / 1,32 Mo par op | 6,03 ms / 6,33 Mo | 5,13 ms / 5,38 Mo | 208 988 |
| 10 000 000 | **23,87 ms** / 17,38 Mo par op | 30,84 ms / 92,25 Mo | 29,08 ms / 30,88 Mo | 2 089 877 |

`-benchtime=1x` : une itération par échantillon, rodage compris. **Aucune autre valeur de N n'est
chronométrée dans le dépôt** (la mémoire, elle, l'est à sept points — voir plus bas). Pour
F(100 000 000), le seul chiffre de durée traçable est le **0,204 s** de calcul seul (sans conversion
décimale) consigné dans [`CHANGELOG.md`](CHANGELOG.md) au 2026-06-09 ; il n'a pas d'artefact de sortie
archivé.

Côté mémoire, l'adoption du multiplicateur d'arène ×10 (2026-07-07) réduit les B/op FFT à F(10M) de **−16 %**
vs ×15, allocations inchangées — gain confirmé en ordre d'exécution inversé (addendum
[ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md)).

L'**empreinte réelle du processus** est un chiffre distinct des B/op ci-dessus : elle est relevée dans
[`docs/audits/mem-baseline-2026-09.txt`](docs/audits/mem-baseline-2026-09.txt) (delta de
`runtime.MemStats.Sys`, **un processus par point** — `Sys` ne redescend jamais, plusieurs points dans un
même processus ne rapporteraient que leur maximum) :

| N | `-algo fast` | `-algo fft` | `-algo matrix` | `-algo all` (défaut, trois calculateurs de front) |
|---|---|---|---|---|
| 1 000 000 | 9 Mo | 18 Mo | 13 Mo | 23 Mo |
| 10 000 000 | 62 Mo | 67 Mo | 141 Mo | 101 Mo |
| 100 000 000 | 617 Mo | 460 Mo | — | — |

C'est cet ordre de grandeur que l'estimation de `--memory-limit` manquait d'un facteur 5 à 12 avant
l'audit 2026-09 : 12 Mo annoncés pour 141 Mo réels à F(10M). Le modèle actuel ne passe jamais sous le
réel et le majore d'au plus **2,47×** ; il reste donc une borne haute, pas une prédiction.

**Choix d'algorithme** : `fast` pour l'usage général (le plus régulier) ; `matrix` pour la pédagogie et la
validation croisée ; `fft` — la même boucle de doublement que `fast`, multiplication forcée en FFT
(cf. [Algorithmes](#algorithmes)) — est plus lent que `fast` aux deux seules tailles mesurées (F(1M) et F(10M)) —
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
| `-tui-theme` | | `dark` | Palette TUI : `dark` ou `high-contrast` |
| `-last-digits` | | `0` | Derniers K chiffres décimaux (mémoire O(K)) |
| `-memory-limit` | | | Budget mémoire (ex. `8G`) ; l'estimation préalable est une **borne haute** (re-modélisée en 2026-09 : elle sous-estimait d'un facteur 5 à 12) |
| `-gc-control` | | `auto` | GC pendant le calcul : `auto`, `aggressive`, `disabled` |
| `-dynamic-thresholds` | | `false` | Ajuste les seuils FFT/parallélisme pendant le calcul (mesuré neutre, [ADR-0001](docs/adr/0001-dtm-decision.md)) |
| `-timeout` | | `5m` | Durée maximale du calcul |
| `-log-level` | | `off` | Diagnostics sur stderr : `off`, `error`, `warn`, `info`, `debug` |
| `-threshold` / `-fft-threshold` | | `0` (auto) | Seuils en bits (0 = valeur lue dans une table selon le matériel détecté — nombre de CPU, SIMD, taille de mot —, **pas une mesure** ; `-1` = désactive) |
| `-strassen-threshold` | | `0` (auto) | Seuil en bits (0 = même table matérielle ; `-1` invalide, voir ci-dessous) |
| `-calibrate` / `-auto-calibrate` | | `false` | Calibration des seuils pour cet hôte |
| `-calibration-profile` | | | Chemin du profil de calibration |
| `-profile-max-age` | | `168h` | Âge maximal d'un profil de calibration en cache avant recalibration |
| `-cpuprofile` / `-memprofile` | | | Profils pprof (CPU pendant le calcul, tas après) écrits dans le fichier donné |
| `-completion` | | | Script de complétion (`bash`, `zsh`, `fish`, `powershell`) |
| `-version` | `-V` | | Informations de version |

Exemples :

```bash
./fibcalc -n 10000000 -algo all -d                  # compare les trois algorithmes
./fibcalc -n 100000000 -last-digits 10 -q -machine  # → 7760546875
./fibcalc -n 1000000000 -memory-limit 8G            # validation mémoire préalable
./fibcalc -calibrate                                # calibre les seuils pour cet hôte
./fibcalc -n 10000000 -threshold -1                 # force le calcul séquentiel
./fibcalc -completion bash > fibcalc.bash           # complétion shell
```

> **`-1` désactive un seuil.** `-threshold -1` supprime toute parallélisation,
> `-fft-threshold -1` supprime le recours à la FFT. C'est la valeur que la
> calibration retient sur les hôtes où le séquentiel gagne, et elle est
> désormais acceptée telle quelle : jusqu'à l'audit 2026-09 elle était rejetée
> à la validation, si bien que le profil calibré était jeté en silence à chaque
> démarrage. `-strassen-threshold` n'admet pas `-1` : son consommateur compare
> `taille <= seuil`, donc une valeur négative forcerait Strassen en permanence
> au lieu de le désactiver.

---

## Configuration

Une variable `FIBCALC_*` n'est lue que si le flag correspondant est absent de la ligne de commande
(`internal/config/env.go:applyEnvOverrides`). Priorité générale :
**flags CLI > variables d'environnement > défauts statiques**.

> **Les trois seuils.** Un profil de calibration en cache **valide** ne remplit que les seuils que vous
> n'avez pas fixés : `--threshold`, `--fft-threshold`, `--strassen-threshold` et leurs variables
> d'environnement l'emportent sur le profil. `app.New` appelle `calibration.LoadCachedCalibration` *après*
> `ParseConfig` (`internal/app/app.go:New`), et celle-ci consulte les marqueurs posés par `ParseConfig`
> pour laisser intact ce qui a été fixé explicitement (`internal/config/thresholds.go`).
> Le profil est lu à `--calibration-profile`, ou à `~/.fibcalc_calibration.json` par défaut ; il n'est
> retenu que si `IsValid()` passe (version de profil, nombre de CPU, `GOARCH`, taille de mot, clé
> heuristique SIMD) et si la config résultante valide encore. Sans profil valide,
> `ApplyAdaptiveThresholds` ne remplit que les seuils laissés à 0.
>
> Jusqu'à l'audit 2026-09, le profil écrasait les trois seuils sans condition : un `--fft-threshold`
> explicite était abandonné en silence sur toute machine ayant déjà exécuté `--calibrate`. Une passe
> fraîche de `--calibrate` / `--auto-calibrate` reste hors de cette règle : vous avez demandé une mesure,
> elle est affichée, et c'est elle qui est enregistrée et appliquée.

Liste complète : [`.env.example`](.env.example). Principales : `FIBCALC_N`, `FIBCALC_ALGO`, `FIBCALC_TIMEOUT`,
`FIBCALC_THRESHOLD`, `FIBCALC_FFT_THRESHOLD`, `FIBCALC_STRASSEN_THRESHOLD`, `FIBCALC_LAST_DIGITS`, `FIBCALC_TUI`, `FIBCALC_TUI_THEME`,
`FIBCALC_CALIBRATION_PROFILE`, `FIBCALC_PROFILE_MAX_AGE` (168h), `FIBCALC_MEMORY_LIMIT`, `FIBCALC_GC_CONTROL`,
`FIBCALC_DYNAMIC_THRESHOLDS`, et
[`NO_COLOR`](https://no-color.org/).

---

## Développement et tests

- **CI GitHub Actions** (`.github/workflows/ci.yml`, réintroduite le 2026-09-07,
  [ADR-0012](docs/adr/0012-audit-2026-09-livre-decisions.md) D1) : le gate tourne sur Ubuntu et
  Windows à chaque poussée, plus les vérifications que l'hôte de développement ne peut pas faire
  (`-tags gmp`, build 32 bits, image Docker) et un fuzzing hebdomadaire. Le gate local
  (`scripts/check.ps1` / `scripts/check.sh`) reste le chemin rapide.
- **Outils épinglés** dans [`scripts/tools.env`](scripts/tools.env) et exécutés par
  `go run <pkg>@<version>` : rien à installer, et un binaire ne peut plus périmer en silence
  contre la chaîne Go (c'est ce qui avait cassé le lint, puis `govulncheck`, `gosec` et
  `staticcheck` d'un coup).
- **Couverture** : plancher garanti **90 %** via `make coverage-check`, `check.ps1` et la CI ; dernière mesure
  **96,1 %** des instructions (2026-09-07, `go1.27.0 windows/amd64`, 22 paquets). Le chiffre est
  daté, pas figé ; le plancher, relevé de 80 à 90 % le 2026-09-23, laisse 4 points sous la mesure
  de la CI Ubuntu (94,0 % le 2026-09-21). Détail, commande de re-datation et angles morts :
  [`docs/TESTING.md` § Coverage](docs/TESTING.md#coverage) (directive A5-04, amendée le 2026-09-04).
- **Golden tests immuables** : `internal/fibonacci/testdata/fibonacci_golden.json` est l'oracle de
  non-régression (étendu à F(50k/100k/200k) sous ADR-0004 §B5) — aucune mise à jour sans ADR.
- **Race detector** : exige CGO et un compilateur C. `scripts/check.ps1` sonde les deux et active `-race`
  quand ils sont présents — relevé du 2026-09-07 : 22 paquets verts sur cet hôte Windows. Sans compilateur
  C, la passe complète se fait via **WSL** (`wsl go test -race ./...`). Les scripts shell sont épinglés en
  LF (`.gitattributes`) pour rester exécutables côté WSL.
- **Lint bloquant** : depuis l'audit 2026-09 (GATE-01), `golangci-lint` **v2** fait échouer
  `check.sh`/`check.ps1`, y compris quand le binaire est absent. Il était consultatif : les scripts
  affichaient l'échec puis écrivaient `Overall: PASS`.
- **Backend GMP sous gate** : depuis 2026-07, `scripts/check.sh` compile et teste `-tags gmp -race`
  (étape 3b, **dure** quand les headers libgmp sont présents, SKIP sinon) — le tag ne peut plus casser
  silencieusement. Validation manuelle : `wsl go test -tags gmp -race ./internal/fibonacci/`.
- Environnement reproductible : [`.devcontainer/`](.devcontainer/devcontainer.json) (Go + CGO + libgmp +
  benchstat) ou [`Dockerfile`](Dockerfile) multi-étages.
- Décisions architecturales : [`docs/adr/`](docs/adr/) (0001–0011, plus `0000-template.md`).
  ⚠ **Dernier audit : 2026-09-03**, en deux passes — l'audit exhaustif (23 constats,
  [ADR-0010](docs/adr/0010-audit-2026-09-decisions.md)) puis la passe de sur-ingénierie
  ([ADR-0011](docs/adr/0011-audit-2026-09-ponytail.md)). Les deux ADR consignent les décisions retenues
  **et** les candidats rejetés, avec la mesure ou l'ADR qui les rejette, pour qu'un audit futur ne les
  re-propose pas sans élément nouveau. Le plan de travail du premier (`audit.md`) a été **retiré de
  l'arbre** une fois exécuté et se relit à l'historique git.
  Les audits 2026-07 ([ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md)) et 2026-08-07
  ont suivi la même règle ; celui de 2026-08-07 **n'a pas d'ADR** — il ne tranchait aucune décision
  d'architecture, et son journal de boucle (`gauntlet-log.md`) a été retiré le 2026-08-08.
  Le tableau « Historique des audits et jalons » en tête de ce fichier en porte le détail.

Commandes principales (équivalents `go` pour Windows sans GNU make) :

```bash
make all             # clean + build + test     (équiv. : go build ./... && go test ./...)
make test            # go test -v -race -cover ./...   (CGO + compilateur C requis)
make test-win        # go test -v -cover ./...         (Windows sans gcc, sans -race)
make lint            # golangci-lint run ./...  (v2 : 21 linters + formateur gofmt)
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

## Contribution et licence

- Changements notables : [`CHANGELOG.md`](CHANGELOG.md) (format Keep-a-Changelog, SemVer — release courante : `v4.1.0`).
- Workflow de contribution : [`CONTRIBUTING.md`](CONTRIBUTING.md) — test rouge → fix → vert,
  validation locale complète avant chaque commit.
- Licence : **Apache 2.0** — voir [`LICENSE`](LICENSE).

### Remerciements

Le cœur de multiplication FFT de `internal/bigfft` dérive de
[bigfft](https://github.com/remyoudompheng/bigfft) de Rémy Oudompheng (BSD-3-Clause, « Copyright (c)
2012 The Go Authors ») ; la notice amont est conservée dans `internal/bigfft/LICENSE` et
[`NOTICE`](NOTICE), l'image Docker l'embarque, et chaque fichier dérivé l'indique en en-tête.

Architecture et algorithmique inspirées de la littérature classique (Schönhage-Strassen, Strassen-Winograd,
fast doubling) ; outillage : Go, Bubble Tea, benchstat, golangci-lint, gosec. Audits, refactorisation et
optimisation 2026 réalisés avec [Claude Fable 5](https://www.anthropic.com/news/claude-fable-5-mythos-5),
Claude Opus 4.8 et Claude Opus 5 (Anthropic) : audit exhaustif 2026-07 (~40 findings, orchestration
multi-agents — Claude Opus 4.8 pilote, exécuteurs Claude Sonnet), suivi 2026-07-07 (release v4.0.0, gate GMP,
balayage arène ×10 — Claude Fable 5), audit qualité et documentation 2026-08-07 (boucle
bâtisseur/critique, lint et gosec à zéro — Claude Opus 5), audit exhaustif du code Go 2026-09-03
(23 constats, trois défauts hauts corrigés, lint rendu bloquant — Claude Opus 5), puis passe de
sur-ingénierie 2026-09-03 (~25 suppressions ou replis, build `gmp` réparé — Claude Opus 5).
