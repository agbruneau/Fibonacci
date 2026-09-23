# FibCalc — Calculateur Fibonacci haute performance

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/tag/agbruneau/FibGo?style=for-the-badge&label=Release&color=2ea44f)](CHANGELOG.md)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge&logo=apache)](LICENSE)
![Status](https://img.shields.io/badge/Status-Prototype_acad%C3%A9mique-orange?style=for-the-badge)

Ce dépôt est un **laboratoire d'expérimentation algorithmique et d'ingénierie logicielle**
(*computational sandbox*) : un bac à sable où l'on pousse un problème volontairement simple — calculer F(n) — jusqu'à ses limites, pour y expérimenter des techniques réelles et **mesurer** ce qu'elles valent.
Le nombre de Fibonacci n'est pas la finalité; c'est le banc d'essai. Il a l'avantage d'avoir une réponse exactement vérifiable, de se calculer par plusieurs algorithmes comparables entre eux, et de devenir arbitrairement coûteux quand n grandit — tout écart de conception se voit donc au chronomètre et à la mémoire, sans place pour l'opinion.

Ce qu'on y expérimente : algorithmique (Fast Doubling, exponentiation matricielle Strassen-Winograd, multiplication FFT Schönhage-Strassen), ingénierie de performance (pooling, allocateur bump, contrôle du GC, parallélisme adaptatif, PGO, auto-calibration), et méthode logicielle (Clean Architecture, tests golden et property-based, ADR, gate de qualité local). Toute affirmation chiffrée doit venir d'un artefact de mesure du dépôt; ce qui n'a pas été réexécuté est signalé comme tel. **FibCalc**, le binaire qui en sort, calcule des nombres de Fibonacci arbitrairement grands à très haute vitesse. Écrit en Go ; le plus grand indice chronométré dans le dépôt est n = 100 000 000 ([`bench-scale-2026-09.txt`](docs/audits/bench-scale-2026-09.txt)).

### Historique

Neuf campagnes d'audit entre mai 2026 et septembre 2026, avec pour chacune ce
qui a été mesuré, ce qui a été rejeté et pourquoi :
[`docs/audits/HISTORY.md`](docs/audits/HISTORY.md). Le détail commit par commit
est dans [`CHANGELOG.md`](CHANGELOG.md), les décisions dans
[`docs/adr/`](docs/adr/).

**État vérifié le 2026-09-23** sur `ed56592` (Windows 11, `go1.27.0`, `golangci-lint v2.13.2`) :
`scripts/check.ps1` vert de bout en bout — build, vet, `go test -race -shuffle=on
-count=1` sur les 21 paquets, lint à **0 finding**, couverture **96,0 %** (plancher
90 %), `govulncheck` sans vulnérabilité. La même séquence tourne en CI sur Ubuntu
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

Dépannage : [`docs/BUILD.md` § Troubleshooting](docs/BUILD.md#troubleshooting). Historique des
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
| `fast` (défaut) — **Fast Doubling** | Θ(M(n)) — O(log n) multiplications | Identité F(2k) = F(k)·(2F(k+1) − F(k)) ; `AdaptiveStrategy` choisit M pas par pas ; pooling état+arène+scratch FFT |
| `matrix` — **Exponentiation matricielle** | Θ(M(n)) — O(log n) multiplications | Variante **Strassen-Winograd** (7 multiplications, 15 add/sub) pour les grandes matrices ; choisit M lui aussi, mais à un autre point du graphe d'appel |
| `fft` — **FFT-Based Doubling** | Θ(M(n)), M **toujours** FFT | **Pas un troisième algorithme** : `FFTBasedCalculator.CalculateCore` relance la boucle de `fast` — le même `ExecuteDoublingLoop` — en échangeant `AdaptiveStrategy` contre `FFTOnlyStrategy`, qui ne consulte plus aucun seuil. C'est un banc d'essai du chemin FFT isolé, et il n'est **plus rapide que `fast`** à aucune des quatre tailles mesurées ([Performance](#performance)) |
| `gmp` (tag de build `gmp`) — **GMP** | Θ(M(n)), M de GMP | Même boucle de doublement, sur les entiers `mpz` ; backend GNU MP (CGO + libgmp) ; `scripts/check.sh` étape 3b le compile et le teste **si** les en-têtes libgmp sont présentes sur l'hôte, sinon l'étape est sautée (`check.ps1` n'a pas d'équivalent) |

M(n) est le coût d'**une** multiplication de deux nombres de n bits, pas celui de F(n) : `math/big`
(Karatsuba [[3]](docs/REFERENCES.md#ref-3), M(n) = Θ(n^1,585) [[13]](docs/REFERENCES.md#ref-13)) sous le seuil FFT, `internal/bigfft` au-dessus. Ce
dernier est une FFT de Schönhage-Strassen [[1]](docs/REFERENCES.md#ref-1) **à un seul niveau** : ses produits point à point
retournent à `math/big` et sa longueur de transformée plafonne à 2^16, donc sa classe reste
Θ(n^1,585), avec une constante plus petite. La borne O(n log n log log n) [[1]](docs/REFERENCES.md#ref-1) suppose la récursion,
que ce code ne fait pas ; O(n log n) est le résultat de Harvey et van der Hoeven [[2]](docs/REFERENCES.md#ref-2), non
implémenté. Les trois calculateurs font O(log n) multiplications sur des opérandes qui doublent à
chaque étape, d'où un coût total Θ(M(n)), et non O(log n) × M(n). Ils ne diffèrent que par la
routine qu'ils y branchent et par l'endroit où ils la choisissent : **aucune ligne du tableau ne
domine donc les autres asymptotiquement**. Ce qui les sépare tient aux constantes et se lit au
chronomètre ([Performance](#performance)), pas dans la colonne « Coût ». `fast` n'est d'ailleurs pas
la formulation la plus économe : GMP et Takahashi [[7]](docs/REFERENCES.md#ref-7)[[11]](docs/REFERENCES.md#ref-11) doublent avec deux carrés par bit,
contre un produit et deux carrés ici
([`FAST_DOUBLING.md`](docs/algorithms/FAST_DOUBLING.md#against-the-two-squaring-formulations)). Qui
prend quel chemin, à quel moment et avec quel seuil :
[`docs/algorithms/FFT.md` § FFT Routing](docs/algorithms/FFT.md#fft-routing), description canonique
du routage. Bibliographie : [`docs/REFERENCES.md`](docs/REFERENCES.md).

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
(six règles d'import montant interdit — huit arêtes, les deux dernières en couvrant deux chacune).
Vue d'ensemble : [`docs/ARCH.md`](docs/ARCH.md) ; référence détaillée :
[`docs/architecture/`](docs/architecture/) (diagrammes C4,
[graphe de dépendances](docs/architecture/dependency-graph.md)).

| Package | Responsabilité |
|---|---|
| `cmd/fibcalc` | Point d'entrée CLI |
| `cmd/generate-golden` | Générateur du golden (oracle indépendant : `math/big` itératif, zéro import interne — ne valide pas la lib par elle-même) |
| `internal/app` | Cycle de vie, dispatch, version |
| `internal/fibonacci` | Algorithmes, frameworks, stratégies ; `memory/` (arène, GC, budget), `fibmath/` (taille de F(n)) |
| `internal/bigfft` | Schönhage-Strassen sur anneaux de Fermat, bump allocator, cache LRU |
| `internal/orchestration` | Exécution concurrente (`errgroup`), agrégation, sélection des calculateurs |
| `internal/calibration` | Calibration adaptative au matériel, micro-benchmarks, profils |
| `internal/cli` / `internal/tui` | Couches de présentation (`ProgressReporter` / `ResultPresenter` partagés) ; sous-package `cli/completion` (génération complétion shell) |
| `internal/config` | Parsing flags + variables d'environnement, estimation des seuils |
| `internal/progress` | Pattern observer (chemin de production : `Freeze`) |
| `internal/{apperrors,format,metrics,ui,testutil}` | Packages de support (feuilles) |
| `test/e2e` | Tests bout-en-bout du binaire CLI (hors `internal/`) |

## Performance

Médianes de 10 échantillons calculées par `benchstat` sur
[`docs/audits/bench-scale-2026-09.txt`](docs/audits/bench-scale-2026-09.txt) — la courbe d'échelle
(EVAL-09) : Windows 11, `go1.27.0`, Intel Core Ultra 9 275HX, 24 fils, commit `751c1cc`,
`-count=10 -benchtime=1x`. Intervalle de confiance à 95 % et B/op médian, dans les unités binaires
de `benchstat` :

| N | Fast Doubling | Matrix Exp. | FFT-Based | Chiffres décimaux |
|---|---|---|---|---|
| 100 000 | **143,2 µs ± 46 %** / 54,12 Kio | 342,8 µs ± 24 % / 382,7 Kio | 876,7 µs ± 25 % / 523,7 Kio | 20 899 |
| 1 000 000 | **3,972 ms ± 27 %** / 1,270 Mio | 8,341 ms ± 8 % / 6,150 Mio | 5,853 ms ± 17 % / 5,157 Mio | 208 988 |
| 10 000 000 | **32,28 ms ± 13 %** / 16,63 Mio | 35,23 ms ± 5 % / 84,95 Mio | 36,53 ms ± 13 % / 29,52 Mio | 2 089 877 |
| 100 000 000 | **193,2 ms ± 11 %** / 561,5 Mio | 351,4 ms ± 19 % / 1,059 Gio | 272,0 ms ± 3 % / 347,1 Mio | 20 898 764 |

Pour rejouer :
`go run golang.org/x/perf/cmd/benchstat@v0.0.0-20260825160852-19be9d8e6c70 docs/audits/bench-scale-2026-09.txt`
(préfixer `MSYS_NO_PATHCONV=1` sous Git Bash).

Ce que ces chiffres mesurent : un appel à `Calculate`, résultat rendu en `*big.Int` sans conversion
décimale, en temps écoulé sur 24 fils — aucun temps CPU n'est relevé. `-benchtime=1x` : un appel par
échantillon, sans itération de rodage écartée ; le premier échantillon est le plus lent de son groupe
dans 6 groupes sur 12. La médiane et l'intervalle y résistent, mais à 100 000 les autres échantillons
se dispersent aussi (± 46 %) : ligne à lire comme un ordre de grandeur. Un seul
hôte, non reproduit ailleurs. `fast` a la médiane la plus basse aux quatre tailles ; au seuil de Bonferroni
(0,05 / 12 comparaisons ≈ 0,004), l'écart est significatif à 100K, 1M et 100M, pas à 10M, où les trois
calculateurs ne se séparent pas. À 100M, `fft` alloue toutefois moins que `fast`.
Comparaisons par paires, pente entre tailles et confrontation à la borne Θ(n^1,585) de
[`FFT.md`](docs/algorithms/FFT.md#complexity-analysis) :
[`docs/PERFORMANCE.md` § Scale curve](docs/PERFORMANCE.md#scale-curve-four-sizes-one-host). En bref, la
pente ne confirme ni ne réfute la borne : celle-ci est asymptotique, son régime commence vers
n ≈ 9·10⁸, et un temps écoulé sur 24 fils mêle le travail et l'efficacité du parallélisme.

F(100 000 000) est donc désormais chronométré : 193,2 ms en médiane pour `fast`, calcul seul. Ce chiffre
remplace le 0,204 s consigné sans artefact dans [`CHANGELOG.md`](CHANGELOG.md) au 2026-06-09.

La **baseline du gate de non-régression** reste
[`docs/audits/bench-baseline.txt`](docs/audits/bench-baseline.txt) (linux/amd64, 24 fils, `-count=5
-benchtime=1x`, estampille `baseline-2026-07-07`, arène ×10) : 3,15 ms et 23,87 ms pour `fast` à F(1M) et
F(10M). Autre hôte, autre OS, autre version de Go : elle ne se compare pas à la courbe ci-dessus, et
`benchstat` refuse d'ailleurs d'aligner les deux fichiers. Les autres fichiers de
[`docs/audits/`](docs/audits/) sont des A/B ciblés (seuils dynamiques et leur retrait, cache FFT,
memclr des pools, stabilité du micro-benchmark), un relevé mémoire, et la mesure GMP ci-dessous.

**Positionnement.** [`docs/audits/bench-gmp-2026-09.txt`](docs/audits/bench-gmp-2026-09.txt) (EVAL-07 :
WSL2 Ubuntu sur le même processeur, `go1.26.1`, libgmp 6.3.0, 5 échantillons) oppose `fast` à
`GMPCalculator` — la **même boucle de doublement à trois produits**, exécutée sur les entiers `mpz` de
GMP. Ce n'est **pas** `mpz_fib_ui`, la fonction Fibonacci de GMP, qui double avec deux carrés par bit
[[11]](docs/REFERENCES.md#ref-11) ; ni elle ni le `fibonacci()` de PARI/GP ne sont mesurés ici
([`COMPARISON.md` § Related work](docs/algorithms/COMPARISON.md#related-work-not-measured)). Rapport
`fast` ÷ `gmp` : **1,21** à F(1M) (4,037 ms contre 3,324 ms), écart **non significatif** (p = 0,151,
n = 5 — un premier échantillon GMP lent emporte le verdict) ; **0,77** à F(10M) (31,49 ms contre
41,06 ms, p = 0,008). `fast` peut paralléliser, `GMPCalculator` est séquentiel : c'est un rapport de
temps écoulé sur 24 fils, pas une comparaison d'arithmétique à cœur égal, et le dépôt n'établit donc
pas que `math/big` multiplie plus vite que GMP. Les B/op de GMP ne se comparent pas non plus (libgmp
alloue hors du tas Go). Détail et limites : [`GMP.md` § Performance](docs/algorithms/GMP.md#performance).

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
(cf. [Algorithmes](#algorithmes)) — n'est plus rapide que `fast` à aucune des quatre tailles mesurées
(à égalité statistique à F(10M), 41 % plus lent à F(100M)) ; qu'il devienne compétitif au-delà de F(100M)
reste une hypothèse que le dépôt ne teste pas. Méthodologie, tuning et suivi de
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
| `-algo` | | `all` | `fast`, `matrix`, `fft` ou `all` (comparaison) ; `gmp` en plus sous `-tags gmp` |
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
`FIBCALC_CALIBRATION_PROFILE`, `FIBCALC_PROFILE_MAX_AGE` (168h), `FIBCALC_MEMORY_LIMIT`, `FIBCALC_GC_CONTROL` et
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
  **96,0 %** des instructions (2026-09-23, `go1.27.0 windows/amd64`, 21 paquets). Le chiffre est
  daté, pas figé ; le plancher, relevé de 80 à 90 % le 2026-09-23, laisse 4 points sous la mesure
  de la CI Ubuntu (94,0 % le 2026-09-21). Détail, commande de re-datation et angles morts :
  [`docs/TESTING.md` § Coverage](docs/TESTING.md#coverage) (directive A5-04, amendée le 2026-09-04).
- **Golden tests immuables** : `internal/fibonacci/testdata/fibonacci_golden.json` est l'oracle de
  non-régression (étendu à F(50k/100k/200k) sous ADR-0004 §B5) — aucune mise à jour sans ADR.
- **Race detector** : exige CGO et un compilateur C. `scripts/check.ps1` sonde les deux et active `-race`
  quand ils sont présents — relevé du 2026-09-23 : 21 paquets verts sur cet hôte Windows. Sans compilateur
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
- Décisions architecturales : [`docs/adr/`](docs/adr/) (0001–0013, plus `0000-template.md`).
  ⚠ **Dernière campagne : l'évaluation académique du 2026-09-15**, exécutée le 2026-09-23
  ([ADR-0013](docs/adr/0013-evaluation-2026-09-decisions.md)), après l'audit « livre » du 2026-09-07
  ([ADR-0012](docs/adr/0012-audit-2026-09-livre-decisions.md)). Avant eux, l'audit du 2026-09-03,
  en deux passes — l'audit exhaustif (23 constats,
  [ADR-0010](docs/adr/0010-audit-2026-09-decisions.md)) puis la passe de sur-ingénierie
  ([ADR-0011](docs/adr/0011-audit-2026-09-ponytail.md)). Les deux ADR consignent les décisions retenues
  **et** les candidats rejetés, avec la mesure ou l'ADR qui les rejette, pour qu'un audit futur ne les
  re-propose pas sans élément nouveau. Le plan de travail du premier (`audit.md`) a été **retiré de
  l'arbre** une fois exécuté et se relit à l'historique git.
  Les audits 2026-07 ([ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md)) et 2026-08-07
  ont suivi la même règle ; celui de 2026-08-07 **n'a pas d'ADR** — il ne tranchait aucune décision
  d'architecture, et son journal de boucle (`gauntlet-log.md`) a été retiré le 2026-08-08.
  Le tableau « Historique des audits et jalons » de
  [`docs/audits/HISTORY.md`](docs/audits/HISTORY.md) en porte le détail.

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

- Changements notables : [`CHANGELOG.md`](CHANGELOG.md) (format Keep-a-Changelog, SemVer — release courante : `v5.0.0`).
- Workflow de contribution : [`CONTRIBUTING.md`](CONTRIBUTING.md) — test rouge → fix → vert,
  validation locale complète avant chaque commit.
- Langue des documents : narratif (README, CHANGELOG, ADR, `docs/audits/`) en français, référence
  technique (`docs/*.md`, `docs/algorithms/`, `docs/architecture/`) et code en anglais —
  [`CONTRIBUTING.md`](CONTRIBUTING.md), point 4 (règle amendée le 2026-09-23, ADR-0013 D2).
- Licence : **Apache 2.0** — voir [`LICENSE`](LICENSE) ; le code dérivé de `bigfft` reste sous
  BSD-3-Clause, voir [`NOTICE`](NOTICE).

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
sur-ingénierie 2026-09-03 (~25 suppressions ou replis, build `gmp` réparé — Claude Opus 5), audit « livre »
2026-09-07 (CI, outils épinglés, `v4.1.0` — Claude Fable 5.1), évaluation académique du 2026-09-15 (grille
universitaire C1–C10 — Claude Fable 5.1) et exécution de son plan le 2026-09-23 (`v5.0.0`, boucle
bâtisseur/critique à l'aveugle — Claude Opus 5.5).
