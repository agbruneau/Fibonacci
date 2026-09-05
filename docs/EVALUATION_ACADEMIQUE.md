# Évaluation académique — dépôt FibGo / FibCalc

| | |
|---|---|
| **Objet** | `github.com/agbruneau/FibGo`, branche `main`, commit `4149136` (2026-09-04) |
| **Date de l'évaluation** | 2026-09-05 |
| **Niveau de référence** | Projet de génie logiciel de 2e cycle (maîtrise) ou projet terminal de 1er cycle avancé, volet « laboratoire expérimental » |
| **Note globale** | **81 / 100** (A−) — détail au §4 |

Le dépôt se présente lui-même comme un *computational sandbox* : calculer F(n) n'est pas la finalité, c'est le banc d'essai d'un travail d'algorithmique, d'ingénierie de performance et de méthode logicielle. L'évaluation le juge sur ce qu'il prétend être.

---

## 1. Grille d'évaluation

Neuf critères, pondérés selon ce qu'un projet de ce type doit démontrer. Chaque critère est noté sur sa pondération; la note globale est la somme.

| # | Critère | Pond. | Ce qu'on attend au niveau visé |
|---|---|---:|---|
| C1 | Fondements algorithmiques et rigueur mathématique | 15 | Algorithmes corrects, complexités justifiées, identités démontrées, choix argumentés |
| C2 | Conception et architecture | 15 | Séparation des responsabilités, dépendances maîtrisées, complexité proportionnée au problème |
| C3 | Qualité du code et pratiques de génie logiciel | 15 | Lisibilité, outillage statique, gestion des erreurs et de la concurrence, portabilité |
| C4 | Vérification et validation | 15 | Oracles indépendants, couverture, tests de propriétés, fuzzing, détection de courses |
| C5 | Méthodologie expérimentale et mesure | 15 | Protocole reproductible, statistiques, étalons externes, distinction mesuré / affirmé |
| C6 | Documentation et communication scientifique | 10 | Lisibilité pour un tiers, cohérence doc↔source, synthèse des résultats et limites |
| C7 | Reproductibilité et outillage | 5 | Build, gate, environnement reproductible, intégration continue |
| C8 | Intégrité intellectuelle, attribution, licence | 5 | Sources citées, licences respectées, assistance IA déclarée |
| C9 | Gestion de projet et traçabilité des décisions | 5 | Historique lisible, décisions consignées, versions |

---

## 2. Méthode : ce qui a été réexécuté, ce qui a été lu

Tout ce qui suit a été fait sur l'hôte d'évaluation (Windows 11, `go1.27.0 windows/amd64`, `golangci-lint v2.13.2`, 24 threads). Les chiffres de débit du dépôt (Linux, Ryzen) n'ont **pas** été rejoués; ils sont pris tels quels et jugés sur leur protocole.

| Vérification | Résultat |
|---|---|
| `go build ./...`, `go vet ./...` | OK |
| `go test ./...` | 21 paquets OK |
| `golangci-lint run ./...` | 0 issue |
| `gofmt -l .` | vide |
| `go tool cover -func=coverage.out` | 96,6 % des instructions |
| `GOARCH=386 go build ./...` | échoue (`maxReasonableWords` déborde `int`), conforme à `docs/PORTABILITY.md` |
| Binaire : `-n 100 -c`, `-n 1000000 -algo fast`, `-n 10000000 -algo all`, `-last-digits 10` | Résultats exacts, trois calculateurs concordants, codes de sortie 0 |
| Graphe d'imports vs `docs/architecture/dependency-graph.md` | Non rejoué; le relevé de validation du dépôt le fait |
| `-race`, `-tags gmp` | Non rejoués ici; `-race` vert selon le relevé du 2026-09-03, `gmp` non compilable sur cet hôte (pas d'en-têtes libgmp) |

Décomptes sur les fichiers suivis par git (`git ls-files`) :

| Mesure | Valeur |
|---|---:|
| Go de production / de test | 16 945 / 29 948 lignes |
| Markdown | 12 129 lignes |
| Fichiers de test | 134 |
| Fonctions `Test*` / `Benchmark*` / `Fuzz*` / `Example*` | 820 / 51 / 7 / 5 |
| `t.Skip` | 21 (gardes `-short` ou d'architecture) |
| `//nolint` / `#nosec` / `TODO` | 5 / 13 / 0 |
| Commits (2026-01 → 2026-09) | 697, dont 389 au format conventionnel |
| Références à des identifiants d'audit dans les commentaires Go (`M-01`, `R4.2`, `FIB-03`…) | 172 |

---

## 3. Évaluation par critère

### C1 — Fondements algorithmiques et rigueur mathématique : 12 / 15

**Forces.**
- Les trois chemins (`fast`, `matrix`, `fft`) sont tous en O(log n · M(n)) et le README le dit franchement : aucune ligne ne domine asymptotiquement, ce qui les sépare tient aux constantes. C'est la bonne lecture, souvent absente des projets étudiants.
- La boucle de doublement (`doubling_framework.go`) implémente correctement F(2k) = 2·T3 − T2 et F(2k+1) = T1 + T2 avec rotation de pointeurs; la copie hors arène avant restitution au pool est expliquée et justifiée (aliasing).
- Strassen-Winograd à 7 multiplications et 15 additions/soustractions, avec carré symétrique à 4 multiplications; `docs/algorithms/COMPARISON.md` compte les opérations par itération.
- `FastDoublingMod` donne les K derniers chiffres en O(log m) mémoire, indépendant de n.
- `FAST_DOUBLING.md` contient une preuve par induction sur Qⁿ; `FFT.md` explique le théorème de convolution et, surtout, démontre que le seuil FFT se compare à un opérande intermédiaire, d'où les premiers n réels (1 440 422 sur `fast`, 1 768 788 sur `matrix`).
- L'honnêteté sur `fft` : ce n'est pas un troisième algorithme, c'est la même boucle avec `FFTOnlyStrategy`, et il est plus lent que `fast` aux tailles mesurées.

**Critiques.**
- La contribution algorithmique propre est nulle : Fast Doubling, Strassen-Winograd et Schönhage-Strassen sont des classiques, et le noyau FFT est hérité (voir C8). La valeur est dans l'ingénierie, pas dans l'algorithmique — ce n'est pas un défaut, mais le titre « laboratoire d'expérimentation algorithmique » promet plus.
- Deux seuils FFT à deux niveaux (500 000 bits dans `fibonacci.Options`, 1 800 mots ≈ 115 000 bits dans `bigfft`), documentés mais déroutants, et le second n'a **aucun appelant de production** (`SetFFTThreshold`) : la calibration ne pilote pas le seuil interne du multiplicateur.
- Le seuil de 500 000 bits est une table de constantes par jeu d'instructions, pas une mesure; le dépôt le reconnaît, mais un laboratoire de mesure devrait l'avoir mesuré.

### C2 — Conception et architecture : 12 / 15

**Forces.**
- Clean Architecture réelle, pas déclarative : `internal/arch_test.go` interdit cinq arêtes montantes et échoue si on les introduit; les 46 imports internes sont vérifiés contre `go list`.
- `bigfft` n'importe aucun paquet interne; `orchestration` définit les interfaces (`ProgressReporter`, `ResultPresenter`) que `cli` et `tui` implémentent. Les interfaces sont étroites (`Multiplier` à deux méthodes).
- Les ADR 0008 à 0011 consignent aussi les candidats **rejetés**, avec la preuve — une pratique de conception mature.

**Critiques.**
- Sur-ingénierie relative au problème : 21 paquets, 17 patterns inventoriés, ~17 000 lignes de production pour calculer F(n). Deux mécanismes adaptatifs redondants coexistent (`calibration`, 1 952 lignes, et `threshold/` DTM), le second désactivé par défaut faute de gain reproduit (ADR-0001, ADR-0010). Le calculateur `fft` duplique `fast`. La TUI (1 912 lignes) est périphérique au propos scientifique.
- `internal/fibonacci` concentre 4 564 lignes de production dans un seul paquet : cohésion faible, navigation coûteuse.
- État global atomique (seuils, parallélisme FFT) plutôt que porté par le contexte (ADR-0003 en fait un choix assumé, mais il reste une dette).

### C3 — Qualité du code et pratiques de génie logiciel : 12,5 / 15

**Forces.**
- 21 linters à zéro finding, `gosec` inclus, `gofmt` propre, 5 `//nolint` et 13 `#nosec` seulement, aucun `TODO` dans le Go.
- Politique de panic explicite (ADR-0002) : pré-conditions converties en erreurs, post-conditions re-propagées; annulation par contexte vérifiée entre multiplications; GC restauré même en cas de panic.
- Gestion mémoire soignée (arène, pools, `sync.Pool` avec plafonds justifiés) et commentée avec ses pièges.

**Critiques.**
- 172 références à des identifiants d'audit (`M-01`, `R4.2`, `FIB-03`, `GATE-01`…) dans les commentaires Go, alors que le plan d'audit (`audit.md`) a été retiré de l'arbre. Un lecteur externe ne peut pas les résoudre sans fouiller l'historique git; le code documente son passé plutôt que son intention.
- Commentaires parfois disproportionnés (37 lignes pour deux constantes en tête de `fastdoubling.go`).
- `go:linkname` vers les internes de `math/big` : fragile à chaque version de Go, documenté comme tel mais sans plan de repli compilé.
- Aucune cible 32 bits ne compile; un test reconnu *flaky* (`TestStateBump_PinnedAcrossCachedCalls`).

### C4 — Vérification et validation : 14 / 15

**Forces.**
- Couverture 96,6 % vérifiée, plancher de 80 % appliqué par le gate.
- Oracle indépendant : `cmd/generate-golden` produit le golden par `math/big` itératif sans aucun import interne; golden jusqu'à F(200 000), immuable sans ADR.
- Tests de propriétés (identité de Cassini, gopter, 100 cas par calculateur), équivalence croisée entre les trois calculateurs, tests de contrat de panic, test d'architecture, tests bout-en-bout du binaire, `-race` vert sur 21 paquets.
- Validation croisée en production : `-algo all` compare les résultats et sort avec le code 3 en cas de divergence.
- Le fuzzing est repositionné honnêtement : 63 seeds rejoués en régression, la mutation ne tourne sur aucun gate.

**Critiques.**
- Le chemin FFT n'est validé aux tailles de production que par équivalence entre implémentations et par des seuils abaissés en test (20 000 bits dans les tests de propriétés). Aucun oracle indépendant ne couvre n ≥ 1,44 M, là où `fast` bascule réellement en FFT.
- Pas de test de mutation; le fuzzing n'est pas planifié (même nocturne).
- Ratio test/production de 1,77 : la suite est un actif, mais aussi un coût de maintenance que rien ne mesure.

### C5 — Méthodologie expérimentale et mesure : 11,5 / 15

**Forces.**
- Protocole A/B `benchstat` en double ordre d'exécution, valeurs p rapportées, seuil de 5 %; trois recommandations rejetées sur mesure (FIB-05, cache FFT ×4, gain DTM), et le rejet est consigné.
- Discipline « mesuré / non mesuré » exemplaire : chaque chiffre pointe vers un artefact de `docs/audits/` ou est marqué sans provenance; un chiffre sans artefact a été retiré plutôt que reformulé.
- Mémoire résidente relevée par processus (un point par processus, pour éviter le plateau de `Sys`).

**Critiques.**
- **Aucun étalon externe.** Ni GMP (`mpz_fib_ui`), ni la bibliothèque `bigfft` amont, ni une autre implémentation ne servent de référence. Pour un dépôt qui se définit comme laboratoire de mesure, c'est la lacune principale : « très haute vitesse » n'est comparé qu'à soi-même. Le calculateur GMP existe et n'est pas mesuré.
- Une seule baseline de débit, deux valeurs de n (1 M et 10 M), un seul hôte, `-count=5 -benchtime=1x`. Pas de courbe de scalabilité, pas de profil `pprof` archivé, pas d'analyse de l'écart entre la baseline Linux (23,87 ms à F(10M)) et le run Windows de cette évaluation (80 ms, trois calculateurs de front).
- Tables historiques sans provenance conservées dans `PERFORMANCE.md` (écart ×88 reconnu sur la ligne 10 M). Les garder « pour l'ordre relatif » est discutable : elles polluent la lecture.
- Pas de gate de non-régression de performance automatisé; `benchstat` reste manuel.
- Le fait que `fft` soit plus lent que `fast` est constaté, pas expliqué.

### C6 — Documentation et communication scientifique : 8 / 10

**Forces.**
- Corpus de 12 129 lignes : 11 ADR, 11 figures Mermaid validées contre la source, guides build/test/portabilité/calibration, changelog Keep-a-Changelog, documentation algorithmique avec preuves.
- Trois passes de resynchronisation doc↔source (2026-08, 2026-09) avec une commande par affirmation : peu de projets étudiants atteignent ce niveau de cohérence.

**Critiques.**
- Le README ouvre sur un tableau d'historique des audits d'une trentaine de lignes **avant** le démarrage rapide. Un lecteur qui arrive doit traverser le journal du projet pour atteindre `go build`. C'est hostile au lecteur, et c'est la première impression.
- Ton auto-référentiel omniprésent (⚠, « annoté le… », « recompté le… », « revérifié le… ») : la documentation devient un journal d'audit. La rigueur est réelle, mais la forme fatigue et masque le contenu.
- Bilinguisme non maîtrisé : README en français, `docs/*.md` majoritairement en anglais, ADR mixtes, sortie du binaire en anglais.
- La section 1.0.0 du CHANGELOG porte encore des affirmations fausses et non annotées : « Security policy with vulnerability disclosure process » (aucun `SECURITY.md`), « Rate limiting protection against DoS », « Mock generation with mockgen » (absent de `go.mod`).
- Il manque ce qu'un lecteur académique cherche en premier : un résumé d'une page — problème, méthode, résultats chiffrés, limites, ce qui a été appris.

### C7 — Reproductibilité et outillage : 3,5 / 5

**Forces.**
- Gate local dur (`check.sh`, `check.ps1`) : build, vet, test `-race`, lint bloquant, plancher de couverture; un binaire de lint absent fait échouer le gate — la leçon GATE-01 (lint silencieusement éteint pendant des mois) est bien tirée.
- Devcontainer avec libgmp et benchstat, Dockerfile multi-étages distroless, PGO, cross-compilation cinq cibles.

**Critiques.**
- **Aucune intégration continue distante**, par décision assumée. Cette décision prive tout tiers d'une preuve indépendante que le gate passe : l'évaluateur a dû tout rejouer. En contexte académique, la reproductibilité par un tiers est un critère, pas une option.
- Digests Docker non épinglés (`TODO(SEC-04)`); le devcontainer installe `staticcheck` alors que le gate exige `golangci-lint v2`; `check.ps1` n'a pas d'étape GMP.

### C8 — Intégrité intellectuelle, attribution, licence : 2,5 / 5

**Forces.**
- Licence Apache 2.0 présente.
- Divulgation exemplaire de l'assistance IA : modèles, dates, rôles (pilote, exécuteurs, critique), nature des passes, jusqu'aux commits d'agents identifiés (`google-labs-jules[bot]`). C'est rare, et c'est exactement ce que l'intégrité académique demande en 2026.

**Critiques.**
- `internal/bigfft` est **très probablement dérivé** de `github.com/remyoudompheng/bigfft` (licence BSD-3-Clause) : mêmes fichiers (`fft.go`, `fermat.go`, `arith_decl.go`), mêmes identifiants (`fermat`, `norm`, `ShiftHalf`, `polyFromNat`, `PolValues`, `NTransform`, `fourier`), même constante de 1 800 mots, et un commentaire repris mot pour mot (« TestCalibrate seems to indicate a threshold of 60kbits on 32-bit arches… »). Le dépôt ne nomme l'auteur d'origine nulle part : ni NOTICE, ni copie de la licence BSD, ni mention dans `BIGFFT.md` ou les remerciements; seul l'en-tête « The Go Authors » d'`arith_decl.go` survit. Le travail d'extension (bump allocator, cache LRU, parallélisme, politique de panic) est réel et substantiel, mais il est présenté comme s'il partait de zéro. En contexte académique, c'est une lacune d'attribution sérieuse; en contexte licence, la clause 1 de BSD-3 exige la conservation de l'avis de copyright. **Ceci est une inférence forte, pas une vérification** : un `diff` contre l'amont trancherait; il n'a pas été fait ici.

### C9 — Gestion de projet et traçabilité des décisions : 4,5 / 5

**Forces.**
- 697 commits sur neuf mois, 56 % au format conventionnel (la totalité depuis l'été); tags `v3.0.0`, `v4.0.0` et tags de réécriture; changelog détaillé.
- Les ADR consignent les décisions **et** les rejets avec leur preuve, puis sont annotés au point exact quand la réalité change (ADR-0001, 0005, 0006, 0007), sans réécrire le raisonnement d'époque. C'est la bonne pratique.

**Critiques.**
- Premiers commits opaques (« Points 3 », « purge », « 2e tentative PDR »); une seule PR, tout passe par `main`; pas de tags 1.0.0/2.0.0 alors que le changelog les décrit.

---

## 4. Note globale

| Critère | Note | / |
|---|---:|---:|
| C1 Fondements algorithmiques | 12 | 15 |
| C2 Conception et architecture | 12 | 15 |
| C3 Qualité du code | 12,5 | 15 |
| C4 Vérification et validation | 14 | 15 |
| C5 Méthodologie expérimentale | 11,5 | 15 |
| C6 Documentation | 8 | 10 |
| C7 Reproductibilité et outillage | 3,5 | 5 |
| C8 Intégrité et attribution | 2,5 | 5 |
| C9 Gestion de projet | 4,5 | 5 |
| **Total** | **80,5 → 81** | **100** |

**Appréciation.** Un travail d'ingénierie logicielle d'un niveau nettement supérieur à la moyenne des projets de maîtrise : la vérification est exemplaire, la discipline de mesure (mesuré / affirmé, rejets sur preuve) est celle d'un bon laboratoire, et la traçabilité des décisions est rare. Trois choses le retiennent sous la barre du A : l'absence d'étalon externe dans un projet qui se définit par la mesure, l'attribution manquante du noyau FFT, et une documentation dont la rigueur étouffe la lisibilité. La complexité du système est disproportionnée au problème, mais le dépôt le sait et a commencé à la réduire (ADR-0011).

---

## 5. Bonifications

Listées par identifiant; la colonne « effet » donne l'ampleur estimée sur la note. Les points sont indicatifs.

| # | Bonification | Critère | Effet |
|---|---|---|---:|
| B1 | **Attribuer `bigfft`** : NOTICE, copie de la licence BSD-3, en-tête de provenance dans chaque fichier hérité, paragraphe dans `BIGFFT.md` et les remerciements distinguant l'hérité de l'ajouté | C8 | +2 |
| B2 | **Étalon externe et scalabilité** : mesurer le calculateur GMP existant, ajouter `mpz_fib_ui` et la `bigfft` amont comme références; courbe n ∈ {10⁴ … 10⁸} sur au moins deux hôtes, `-count ≥ 10`, intervalles de confiance | C5 | +3 |
| B3 | **CI distante** : le gate existant dans GitHub Actions, `benchstat` contre la baseline avec seuil, fuzzing nocturne borné | C7, C4 | +1,5 |
| B4 | **Réécrire l'entrée du dépôt** : un résumé d'une page (problème, méthode, résultats, limites), le démarrage rapide en premier, l'historique des audits déplacé dans `docs/AUDITS.md`; choisir une langue par document | C6 | +1,5 |
| B5 | **Rendre les commentaires autoportants** : remplacer les 172 identifiants d'audit par l'intention qu'ils désignent, ou restaurer un index `docs/audits/INDEX.md` qui les résout | C3 | +1 |
| B6 | **Réduire** : trancher DTM contre calibration (un seul mécanisme, mesuré), supprimer le calculateur `fft` ou le reclasser en outil de test, brancher la calibration sur le seuil interne de `bigfft` ou retirer `SetFFTThreshold` | C2, C1 | +1 |
| B7 | Oracle indépendant au-delà du seuil FFT (une valeur golden à n ≈ 2 M produite par GMP ou par `generate-golden`) | C4 | +0,5 |
| B8 | Tests de mutation (`go-mutesting` ou équivalent) sur `fibonacci` et `bigfft` | C4 | +0,5 |
| B9 | Hygiène : corriger la section 1.0.0 du CHANGELOG, épingler les digests Docker, aligner le devcontainer sur `golangci-lint v2`, rendre `maxReasonableWords` dépendant de la plateforme ou retirer toute mention 32 bits | C6, C7, C3 | +0,5 |

B1 à B4 suffisent à passer de A− à A.

---

## 6. Projets futurs suggérés

Chacun part d'un actif que le dépôt possède déjà.

1. **Étude comparative des multiplicateurs.** Schönhage-Strassen (présent) contre une NTT sur premiers 64 bits avec réduction de Montgomery, et contre `math/big` et GMP; même harnais, mêmes tailles, plusieurs architectures. Le dépôt a déjà le banc; il lui manque les concurrents. Mémoire de maîtrise plausible.
2. **Généralisation aux récurrences linéaires.** Le framework matriciel et la boucle de doublement s'étendent aux suites de Lucas, Pell, k-bonacci et à toute récurrence à polynôme caractéristique donné. Le pooling, l'arène et la stratégie adaptative se réutilisent tels quels.
3. **Vérification formelle du noyau.** Prouver les identités de doublement et l'arithmétique modulo 2ⁿ+1 (`fermat.norm`, `Shift`, `reduce`) en Lean ou Dafny; ou modéliser les invariants d'aliasing arène/pool en TLA+. Le dépôt a déjà une politique de panic et des tests de contrat qui définissent les propriétés à prouver.
4. **Harnais de benchmark reproductible multi-hôtes.** Conteneur épinglé, `benchstat`, traitement statistique, publication des artefacts selon les critères de badge « Artifact Evaluated / Reproduced » de l'ACM. Le projet a la discipline; il lui manque l'infrastructure.
5. **Étude empirique de la boucle humain-IA.** Le dépôt est déjà un cas d'étude : quatre audits, une soixantaine de constats, des recommandations rejetées sur preuve, des affirmations fausses débusquées puis corrigées. Une analyse systématique (taux de faux positifs des audits, coût par constat, dérive documentaire) est publiable en génie logiciel empirique.
6. **Contribution amont.** Proposer à `remyoudompheng/bigfft` le bump allocator, le cache de transformées et le parallélisme de récursion, avec les mesures. La revue par la communauté est la validation externe qui manque au projet — et elle règle B1 en passant.
7. **Passage à l'échelle n ≥ 10⁹.** Multiplication FFT hors-cœur ou distribuée, puis GPU; le cas n = 10⁸ à 617 Mo résidents montre où la mémoire devient la contrainte.
8. **Performance par watt.** Mesurer l'énergie (RAPL, `perf`) plutôt que le temps seul; la stratégie adaptative et le contrôle du GC ont des effets énergétiques que le chronomètre ne voit pas.
