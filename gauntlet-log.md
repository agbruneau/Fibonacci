# Gauntlet log — audit Go + documentation

Append-only. Un bloc par tour.

## Barre (choisie par l'utilisateur)

**Gate propre + doc vraie.**

1. `golangci-lint run ./...`, `gosec ./...` et `gofmt -l .` rapportent chacun **zéro**,
   sans élargir les exclusions de `.golangci.yml`, sans `//nolint`, sans supprimer de test.
   `go build ./...`, `go vet ./...`, `go test ./...` continuent de passer.
2. Chaque affirmation factuelle des 30 `.md` correspond à ce que la source dit aujourd'hui.

## État initial (mesuré, HEAD = 869bd6a)

| Contrôle | Résultat |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./...` | 21 paquets OK |
| `gofmt -l .` | 1 (`internal/fibonacci/memory/gc_control.go`) |
| `golangci-lint run ./...` | **152** |
| `gosec ./...` | **19** |

Répartition lint : gocritic 58 (commentedOutCode 21, emptyStringTest 16, paramTypeCombine 6,
unnamedResult 4, evalOrder 3, captLocal 2, octalLiteral 2, unnecessaryDefer 2, appendCombine 1,
builtinShadow 1) · whitespace 32 · errcheck 26 · revive 15 · govet 7 · prealloc 5 ·
staticcheck 3 · unparam 2 · gofmt 1 · ineffassign 1 · misspell 1 · unused 1.

Répartition par répertoire : `internal/bigfft` 48+9 · `internal/fibonacci/**` 41+5 ·
`internal/tui` 28 · reste 35+5.

Répartition gosec : G115 ×18, G304 ×1.

## Découpe

| Morceau | Périmètre possédé |
|---|---|
| A1 | `internal/bigfft/**` |
| A2 | `internal/fibonacci/**` (y compris `memory/`, `threshold/`) |
| A3 | tout le reste du code Go (`internal/tui`, `calibration`, `cli`, `config`, `app`, `progress`, `orchestration`, `metrics`, `format`, `errors`, `ui`, `testutil`, `internal/arch_test.go`, `cmd/**`, `test/**`) |
| B | les 30 `.md` + les en-têtes de commentaire de `scripts/`, `Makefile`, `.golangci.yml` |

Note d'adaptation : la moitié « gate » de la barre a un critère binaire (l'outil sort 0 ou non).
La comparaison A/B à l'aveugle n'y apporte rien — le critique **rejoue les outils** et vérifie
qu'aucune exclusion n'a été élargie. Les règles non négociables sont conservées : le bâtisseur
ne note jamais son travail, chaque tour reçoit un critique neuf, le critique juge l'artefact.

---

## Tour 1

**Bâtisseurs** — 4 en parallèle, périmètres disjoints, `.golangci.yml` interdit à tous
sauf B (commentaires uniquement).

| Morceau | Fichiers touchés | Rendu du bâtisseur |
|---|---|---|
| A1 `internal/bigfft` | 22 | gofmt/lint/gosec/test à 0 |
| A2 `internal/fibonacci/**` | 23 | gofmt/lint/gosec/test à 0 ; 5 sites G115 résolus en supprimant le cast (`n>>uint(i)` → `n>>i`) ou par une borne réelle ; `threshold.setLogger` supprimé (mort) |
| A3 reste du code | 38 | lint/gosec/gofmt à 0 ; 4 `#nosec G115` déplacés en ligne (ils étaient sur la ligne précédente, gosec ne les voyait pas) ; **1 `#nosec G304` nouveau** ; `startCalculationCmd` réordonné pour `context-as-argument` |
| B documentation | 20 | ~60 affirmations fausses corrigées, chacune avec sa source ouverte |

**Mesure de l'arbre fusionné, par moi, avant toute critique :**

```
gofmt -l .              → (vide)
go build ./...          → exit 0
go vet ./...            → exit 0
golangci-lint run ./... → exit 0        (était: 152)
gosec ./...             → exit 0        (était:  19)
go test -count=1 ./...  → 21 paquets ok
```

`git diff -- .golangci.yml` : commentaires seulement. Aucune exclusion élargie, aucun seuil
déplacé, aucune règle ajoutée ou retirée. La barre n'a donc pas été atteinte par la porte
dérobée qu'elle interdit.

**Verdict A/B** — sans objet sur la moitié « gate » : critère binaire, 152+19 → 0. Les critiques
neufs jugent l'artefact (le diff), pas l'atteinte du chiffre.

**Points remontés par les bâtisseurs eux-mêmes, non tranchés par eux :**
- le `#nosec G304` de `internal/calibration/profile.go` est **nouveau** et son argument est une
  affirmation de frontière de confiance, pas une preuve de borne ;
- deux branches `if` vides de tests ont été converties en assertions plutôt que supprimées ;
- `docs/TUI_GUIDE.md:105` montrait encore l'ancien ordre de `startCalculationCmd`.

**Critiques** — 4 agents neufs, sans connaissance des bâtisseurs.

| Morceau | Verdict | Défauts retenus |
|---|---|---|
| A1 `internal/bigfft` | PASS | `TempAllocator` exportée dont tout le jeu de méthodes est devenu non exporté ; résultats nommés incohérents entre `allocFermatTemp` et `allocFermatSlice` |
| A2 `internal/fibonacci/**` | PASS | `defer gc.End()` retiré des deux seuls tests où `End()` fait quelque chose ; `SetDefaultStrassenThreshold` avale silencieusement le hors-plage |
| A3 reste du code | PASS | justification du `#nosec G304` fausse par omission ; 4 `#nosec G115` mis en vigueur = suppression nette ; invariant de commentaire plus large que le vrai |
| B documentation | **FAIL** | 40 affirmations fausses encore debout, dont 2 corrections elles-mêmes fausses |

**Un défaut de critique réfuté par moi**, source ouverte : le critique A3 affirmait que
`.golangci.yml` ne porte aucune exclusion gosec G104/G115 et que les commentaires de
`scripts/check.sh` mentaient. Elles existent, `.golangci.yml:106-112`, sous
`linters-settings.gosec.excludes` — il n'avait regardé que `issues.exclude-rules`. Défaut écarté.

**Un désaccord bâtisseur/critique tranché par moi**, mesure à l'appui :

```
golangci-lint -c <3 règles gocyclo retirées> ./internal/config/... ./internal/tui/... ./internal/bigfft/...
internal\config\config.go:99:1: cyclomatic complexity 16 of func `(AppConfig).Validate` is high (> 15)
internal\tui\model.go:94:1:    cyclomatic complexity 16 of func `(Model).Update` is high (> 15)
internal\bigfft\fft_recursion.go:99:1: cyclomatic complexity 17 of func `fourierRecursiveUnified`
```

Le critique a raison : gocyclo **rapporte bien** `AppConfig.Validate`. Le bâtisseur B avait
remplacé une affirmation vraie par une fausse dans `.golangci.yml:150-152`.

### Écart retenu pour le tour 2

Un seul, celui du morceau qui a échoué : **chaque correction a été appliquée au seul site où elle
a été remarquée, jamais balayée sur le corpus.** `RunCalibration` corrigé dans PERFORMANCE.md mais
pas ARCH.md ; `tui.Run` dans TUI_GUIDE.md mais pas ARCH.md ;
`shouldParallelizeMultiplicationCached` dans FFT.md mais pas ARCH.md ; les dé-exports
`AllocFermat*` dans BIGFFT.md mais dans aucun des deux diagrammes Mermaid ; les paquets supprimés
`metrics/system` et `parallel` retirés de README.md et ARCH.md mais pas de
`container-diagram.mermaid` ; « 3-Tier » corrigé dans FAST_DOUBLING.md mais pas CALIBRATION.md ;
`CLAUDE.md` purgé de ARCH.md mais toujours cité comme autorité vivante dans cinq ADR.

---

## Tour 2

**Écart transmis aux bâtisseurs** — travailler *par affirmation*, pas par fichier : pour chaque
affirmation fausse, `grep` sur tout le corpus, corriger tous les sites en une passe, puis passer à
la suivante. Une édition mono-fichier est le signe qu'on a sauté le balayage.

Les trois morceaux de code reçoivent chacun les défauts de leur critique, avec instruction de
vérifier le critique avant d'agir (il s'est déjà trompé une fois) et, pour l'oracle de
`fft_race_test.go`, de **prouver** la reprise en cassant volontairement la boucle de production
pour voir le test échouer, puis en la restaurant.

**Bâtisseurs** — 4 en parallèle.

| Morceau | Rendu |
|---|---|
| A1' `internal/bigfft` | garde entrée dans `valueSize`, point de passage unique des six appelants → gosec accepte le cast **sans aucune suppression**, le `#nosec G115` est supprimé. `TempAllocator → tempAllocator`, `PoolAllocator → poolAllocator`, `GetPoolAllocator()` remplacé par une variable de paquet. Deux sous-tests ajoutés. |
| A2' `internal/fibonacci/**` | oracle de `fft_race_test.go` réécrit sur `big.Int.Bit()` — route structurellement distincte du décalage de production ; `t.Cleanup(gc.End)` dans les deux tests où `End()` agit ; commentaire de `SetDefaultStrassenThreshold` rendu vrai sur les deux largeurs de `int`. |
| A3' reste du code | `#nosec` en périmètre **7 → 3** : les quatre G115 supprimés en retirant les conversions (`Indicators.DoublingSteps` prend le type `int` que `bits.Len64` renvoie déjà), pas en les annotant. G304 restant argumenté : refus de blanchir le chemin par `filepath.Clean`, qui ferait passer la porte sans ajouter de contrôle. |
| B' documentation | 44 familles d'affirmations, chacune balayée sur tout le corpus. A rattrapé en vol les renames de A1'. Balayage étendu à `.env.example` et aux 7 `.mermaid`. 27 fichiers. |

**Preuve de reprise de l'oracle, fournie par le bâtisseur** (production décalée d'un bit) :

```
--- FAIL: TestFFTRaceArenaAliasing_ConcurrentCalculateCore (0.00s)
    fft_race_test.go:243: concurrent result mismatch for N=20000
```
puis retour à l'état correct → `ok  github.com/agbruneau/FibGo/internal/fibonacci  0.421s`

**Mesure de l'arbre fusionné, par moi :**

```
gofmt -l .              → (vide)
go build ./...          → exit 0
go vet ./...            → exit 0
golangci-lint run ./... → exit 0
gosec ./...             → exit 0
go test -count=1 ./...  → 21 paquets ok
```

Recensement des suppressions, HEAD contre arbre de travail :
`//nolint` **4 → 4**, mêmes quatre fichiers (`bigfft/fft_recursion.go`, `cli/completion/bash.go`,
`fibonacci/common.go`, `fibonacci/fft.go`). `#nosec` **19 → 13**. La porte n'a donc pas été
atteinte en muselant les outils : elle a été atteinte en retirant six sites de suppression.

**Corrigé par moi**, défaut hors-périmètre remonté par B' : `internal/fibonacci/options.go:33`
annonçait 128 entrées de cache par défaut ; `DefaultTransformCacheConfig` en pose 256
(`internal/bigfft/fft_cache.go:37`).

**Critiques** — 4 agents neufs, avec mandat de *prouver par mutation* plutôt que de raisonner.

| Morceau | Verdict | Ce qui a été prouvé, pas argumenté |
|---|---|---|
| A1' `internal/bigfft` | PASS | garde de `valueSize` retirée → le G115 réapparaît sur la même ligne (suppression méritée) ; sous-test 1 tué en retirant la garde, sous-test 2 tué en passant `>` à `>=` ; allocations mesurées 3,0 → 5,0 par instrumentation |
| A2' `internal/fibonacci/**` | PASS | 4 mutations de la boucle de production tuées par l'oracle (décalage, test inversé, deux masques), restauration confirmée par sha256 ; ordonnancement `t.Cleanup` / `t.Parallel()` mesuré par test jetable |
| A3' reste du code | PASS | `DoublingSteps` uint64→int : sondes compilées dans deux worktrees, sortie CLI et TUI **identique octet pour octet** sur tout le domaine ; index d'annulation 32 bits recalculé indépendamment (3549559750) |
| B' documentation | **FAIL** | 16 défauts ; les corrections chiffrées du tour 2 (table mémoire, Strassen-Winograd, gocyclo, 4^i) toutes confirmées exactes |

**`-race` n'a pas pu tourner** : pas de compilateur C sur cette machine
(`go: -race requires cgo` / `gcc not found`). Deux critiques l'ont signalé plutôt que de le
maquiller. Les conclusions de concurrence de ce tour sont donc statiques, pas mesurées.

**Défaut préexistant découvert, hors périmètre de l'audit** :
`GOARCH=386 go build ./internal/fibonacci/` échoue sur un débordement de constante dans
`internal/fibonacci/memory/arena.go:26,29,30` (`maxReasonableWords` déborde un `int` 32 bits).
Sortie identique octet pour octet sur HEAD non modifié. Le `Makefile` ne construit que amd64 et
arm64 ; aucune barrière du projet ne contrôle donc la cible 32 bits.

### Écart retenu pour le tour 3

Les trois morceaux de code ont convergé sur **un seul thème** : plus aucun défaut de comportement,
uniquement des *commentaires qui affirment plus que le code ne tient* — `allocator.go:74` dit
qu'un test couvre une régression qu'il ne couvre pas ; `gc_control_test.go:48` énonce une règle
fausse sur `defer` ; `fft_race_test.go:107` dit « arithmétique » pour un décalage logique ;
`profile.go:95` dit « exhaustively, exactly three routes » quand il y en a quatre.

Côté documentation, l'écart est distinct et précis : **le balayage a été mené identifiant par
identifiant, pas relation par relation.** Tout ce qui survit est une arête entre deux paquets qui
ne se parlent pas, un ordonnancement qui dessine B après A quand le code fait les deux dans le
même constructeur, une branche conditionnelle là où le code n'en a aucune, un numéro de ligne
décalé de deux, une énumération qui se dit complète et ne l'est pas. Aucun `grep` de symbole ne
voit cela.

---

## Tour 3

Découpe refondue sur les deux thèmes restants, plus trois morceaux de code séparés :

| Morceau | Écart transmis |
|---|---|
| Code | les cinq commentaires qui affirment trop. Un seul n'est pas une reformulation : bigfft n'a aucune mesure d'allocation sur `poolAllocator`, son allocateur de production — le bâtisseur doit y laisser un test, et prouver qu'il échoue avant de le déclarer utile. |
| Documentation | balayer **relation par relation** : chaque arête, chaque ordonnancement, chaque « only/always/never », chaque numéro de ligne cité, chaque énumération qui se dit complète. Les sept `.mermaid` sont les fichiers les moins lus du dépôt, ce qui explique qu'ils aient concentré les dégâts. |

**Bâtisseurs** — 2 en parallèle.

Le morceau code a livré les cinq commentaires rendus vrais **plus un test** : bigfft n'avait aucune
mesure d'allocation sur `poolAllocator`. `TestPoolAllocatorSliceAllocsSteadyState` mesure 3,0,
passe à 5,0 sous mutation et échoue — pendant que `TestTransformReleasesBuffersOnError`, celui que
le commentaire prétendait suffisant, reste vert.

Le morceau documentation a corrigé **45+ relations**, chacune tranchée par une commande. Trouvailles
que trois passes précédentes avaient manquées : `flows/matrix.mermaid` dessinait un test de symétrie
inexistant, `flows/fft-pipeline.mermaid` une branche AVX2 avec repli pur-Go alors que les six
`go:linkname` sont inconditionnels, `flows/fastdoubling.mermaid` acquérait le sémaphore avant de
lancer les goroutines quand le code fait l'inverse.

**Corrigé par moi**, sur signalement du bâtisseur documentation qui n'avait pas mandat sur le `.go` :
- `internal/bigfft/fft_recursion.go:17` annonçait le sémaphore Fibonacci en `NumCPU` ;
  `internal/fibonacci/common.go:49` l'initialise à `runtime.GOMAXPROCS(0)`.
- `internal/arch_test.go` : l'en-tête citait `CLAUDE.md` (supprimé), une chaîne de dépendances
  fausse (`internal/fibonacci` n'importe pas `internal/config`, `internal/bigfft` n'importe aucun
  paquet interne) et quatre flèches gardées là où `architectureRules` en porte cinq.

**Critiques**

| Morceau | Verdict | Défaut retenu |
|---|---|---|
| Code | **FAIL** | `gc_control_test.go:51` affirmait que `t.Cleanup` est « the form this suite uses **everywhere** it restores a process global » — le même fichier restaure par `defer` à quatre autres endroits |
| Documentation | **FAIL** | 54 défauts, dont 13 corrections du tour 3 elles-mêmes fausses |

Le critique code a validé le nouveau test par mutation (3,0 → 5,0, échec, restauration, succès) et
jugé défendables les trois défauts laissés — `arena.go` en 32 bits (documenté au bon endroit),
la route 4 de `profile.go` (documentée, non contrainte), et l'enregistrement `gmp` mort dans le
binaire livré (documenté dans `GMP.md`, rejeté proprement par la CLI).

**L'écart le plus lucide du tour** : *la classe « commentaires qui affirment plus que le code ne
porte » se reproduit à taux constant précisément parce que chaque tour la corrige à la main, un
site à la fois, en écrivant à chaque fois un nouveau quantificateur absolu qu'aucun mécanisme du
dépôt ne peut invalider.* Le correctif du tour 2 avait produit le défaut du tour 3.

**Corrigé par moi** : le quantificateur faux de `gc_control_test.go`, et la marge tacite du plafond
d'allocation (4 au lieu de 3 mesuré) désormais énoncée avec sa raison et sa limite.

### Défaut de comportement découvert au tour 3, non corrigé

`LoadCachedCalibration` (`internal/calibration/calibration.go:455-458`) écrase `Threshold`,
`FFTThreshold` et `StrassenThreshold` **sans vérifier si l'utilisateur a posé le drapeau**, et
`app.New` (`internal/app/app.go:93-97`) l'applique après `ParseConfig`. Vérifié à la source et
empiriquement au binaire par le critique : un profil valide fait ignorer silencieusement
`--threshold`. Les six documents qui énoncent la précédence disent l'inverse.

Arbitrage : la documentation est corrigée pour dire le vrai ; la sémantique de la CLI n'est **pas**
changée sans mandat. Remonté à l'utilisateur.

---

## Tour 4 — documentation seule

Le morceau code est clos : le tour 3 n'y a produit qu'un seul défaut, et c'était le correctif du
tour 2 qui l'avait créé. Rendements marginaux atteints.

**Écart transmis** : *rouvrir le fichier qu'on décrit.* Chacune des affirmations neuves du tour 3
portant sur le texte courant d'un autre fichier était fausse dans le même sens — elle décrivait
l'arbre tel qu'il était avant d'être édité. Les numéros de ligne cités sont la catégorie la plus
dense en défauts du corpus : les seuls ADR en portent neuf.

**Bâtisseur** — les 55 défauts traités, plus quatre trouvés en propre (table de performance du
README attribuée à un ADR qui ne la contient pas, « −23 à −35 % » quand le CHANGELOG dit
« −14 % à −35 % », « 1 019 affirmations documentaires vérifiées » sans trace nulle part, un
répertoire `docs/external-reviews/` qui n'a jamais existé).

Il a aussi **corrigé le critique** là où celui-ci surestimait : `CacheStrategy.Sample` n'est pas
appelé « à chaque itération », il est conditionné à `dtm != nil` et étranglé. Un bâtisseur qui
vérifie son propre mandat plutôt que de l'appliquer.

Règle CHANGELOG qu'il a énoncée avant de l'appliquer : ne corriger une entrée historique que
lorsqu'elle contredit une entrée sœur *de la même section de version*, jamais pour la faire
coïncider avec l'arbre d'aujourd'hui. Les trois corrections du 4.0.0 tombent toutes sous cette
règle.

**Erreur de ma part, corrigée** : au tour 2 j'avais changé `options.go:33` de « 128 » à « 256 ».
Les deux sont faux. Quand `FFTCacheMaxEntries` vaut 0 et `n > 0`, `configureFFTCache` calcule
`clamp(2*bits.Len64(n), 64, 4096)`, soit **64 à 128** puisque `bits.Len64` plafonne à 64 ; les 256
de `DefaultTransformCacheConfig` ne s'appliquent que si `n == 0`.

---

## Tour 5 — commentaires Go, dernière famille

Sept défauts de source Go remontés par le bâtisseur documentation, tous de la même famille :
**des chiffres affirmés sans artefact dans le dépôt** (« 15-30 % speedup », « <0,01 % du temps
total », « 2-5 % », « ~4,55 % mesuré »), plus le commentaire de `internal/config/thresholds.go:3-8`
qui énonce à l'envers la précédence des seuils et qui est l'origine du mensonge propagé dans six
documents.

Mandat : `docs/audits/bench-baseline.txt` est le seul artefact de mesure du dépôt. Ce qui peut y
être rattaché est cité ; le reste est supprimé. Interdiction de lancer des benchmarks pour
fabriquer la preuve.

**Rendu** : 24 commentaires, bien au-delà des 6 demandés. Aucun des trois chiffres visés n'a pu
être adossé (`bench-baseline.txt` ne contient que du temps mural et des allocations pour six
combinaisons, pas de décomposition memcpy, pas d'A/B sur le cache de `BitLen`, pas de taux de
succès) — les trois ont donc été retirés. Le bâtisseur a aussi **pinné** la précédence surprenante
plutôt que de seulement la documenter : `TestNewCachedProfileOverridesExplicitFlags`.

---

## Sortie de boucle

**Critique final, mandat unique** : est-ce livrable ? Verdict **DO NOT SHIP** — trois bloqueurs,
tous des commentaires, ~10 lignes. Exactement le mode d'échec annoncé : *le retrait d'un chiffre
non adossé remplacé par une affirmation fausse.* Les deux plus nets pointaient
`BenchmarkCacheImpact` comme preuve d'un effet de cache alors que ce banc pilote un
`FastDoublingCalculator`, qui ne consulte jamais le cache (taux de succès 0 %), et attribuaient à
`CompleteStrategy` un `CalibrationTime` qu'elle n'écrit nulle part.

Un agent de vérification en profondeur a ensuite trouvé 13 défauts de plus et nommé leur mécanisme
commun : **la doc cite `fichier:ligne` pour des fichiers Go que le même lot d'édition décale.**
Correction appliquée en règle, pas en renumérotage : ancrage sur le symbole.

```
Références fichier:ligne vers du .go   121 → 3
```

Les trois survivantes sont délibérées : deux localisations d'erreur compilateur citées mot pour mot
dans `PORTABILITY.md`, et une citation d'un défaut retiré dans ADR-0001. Les 92 ancres symboliques
qui les remplacent ont toutes été vérifiées comme résolvant vers une déclaration réelle. Preuve que
la convention s'imposait : les numéros de ligne de mon propre brief étaient **déjà périmés** quand
le bâtisseur les a lus, sans qu'aucun fichier n'ait bougé entretemps.

### État final, mesuré

```
gofmt -l .              → (vide)      était: 1 fichier
golangci-lint run ./... → exit 0      était: 152
gosec ./...             → exit 0      était:  19
go build ./...          → exit 0
go vet ./...            → exit 0
go test -count=1 ./...  → 21 paquets ok
```

| Recensement | HEAD | Arbre final |
|---|---|---|
| `//nolint` | 4 | **4** (mêmes fichiers) |
| `#nosec` | 22 | **13** |
| `func Test\|Benchmark\|Fuzz` | 877 | **879** |
| `t.Run(` | 408 | **410** |
| assertions | 2291 | **2305** |
| refs `fichier:ligne` vers du `.go` | 121 | **3** |

`git diff -- .golangci.yml` : commentaires seulement, sur les six passes. La porte n'a jamais été
atteinte en muselant un outil.

### Trois tests ajoutés, chacun tué par mutation

- bornes de `bigfft.valueSize` — un sous-test par côté de la limite ;
- `poolAllocator.allocFermatSlice`, régime permanent à 3 allocations — le paquet n'avait aucune
  mesure sur son allocateur de production ;
- `TestNewCachedProfileOverridesExplicitFlags` — épingle le comportement surprenant.

### Laissé sciemment, avec sa raison

1. **`LoadCachedCalibration` écrase les drapeaux explicites.** Défaut de comportement réel, vérifié
   au binaire. La doc dit désormais le vrai et un test l'épingle ; changer la sémantique de la CLI
   dépasse le mandat d'un audit.
2. **`GetDefaultProfilePath` retombe sur un nom relatif** quand `os.UserHomeDir()` échoue — lecture
   et écriture atterrissent dans le répertoire courant. Documenté, non contraint.
3. **`TestStateBump_PinnedAcrossCachedCalls` est instable**, 1 fois sur 24, **à l'identique sur le
   commit parent** : `TestContextCancellation` calcule F(100M) et laisse dans le pool global une
   arène de 10,8 M mots, au-dessus de `maxCachedArenaWords` (4 M), donc `cacheOrPool` refuse de la
   cacher. Préexistant ; le corriger touche un pool partagé par des tests parallèles.
4. **L'arbre ne construit pas en 32 bits** (`maxReasonableWords = 1<<60` déborde un `int` 32 bits,
   `arena.go:26,29,30`). Le `Makefile` ne cible que amd64 et arm64. `PORTABILITY.md` le dit
   maintenant, `386` **et** `arm`.
5. **`-race` n'a pu tourner nulle part** : pas de compilateur C sur cette machine. Les conclusions
   de concurrence de cet audit sont statiques, pas mesurées. Même limite sur le commit parent.

### Ce que la boucle a réellement produit

Six passes, ~25 agents. Les défauts ont changé de nature à chaque tour, ce qui est le signe que la
boucle mordait : outils au tour 1, comportement au tour 2, commentaires faux au tour 3, relations
de diagrammes au tour 4, chiffres non adossés au tour 5, numéros de ligne au tour 6. Trois fois un
critique s'est trompé et a été réfuté à la source ; deux fois un correctif a engendré le défaut du
tour suivant.


