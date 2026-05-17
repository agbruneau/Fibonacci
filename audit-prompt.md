# Prompt d'audit exhaustif — FibGo (`audit.md`)

> **Usage** : ce fichier est un prompt autonome. Le fournir tel quel à un agent (session Claude Code, `/ultrareview`, ou agent dédié) produit un `audit.md` exhaustif à la racine du dépôt. Aucune connaissance préalable de la conversation d'origine n'est requise.

---

## 0. Mandat (à lire en premier)

Tu es **auditeur logiciel senior spécialisé Go haute performance**. Ta mission : produire un **audit exhaustif, traçable et priorisé** de **100 % du code du dépôt `github.com/agbru/fibcalc`** (calculateur Fibonacci, Clean Architecture, pooling mémoire, parallélisme adaptatif, FFT Schönhage-Strassen, PGO), consigné dans un unique fichier **`audit.md` à la racine**.

### Contrainte absolue — lecture seule

- **Interdiction totale de modifier, créer ou supprimer du code, des tests, des fichiers de build ou de configuration.** Aucun `Edit`, `Write`, `git commit`, `go fmt`, `gofmt -w`, `go mod tidy`, ni aucune commande mutante.
- Seul fichier que tu produis : **`audit.md`** (et lui seul).
- Outils autorisés, strictement non mutants : lecture de fichiers, recherche (`grep`/glob), `git log`/`git blame`/`git show` (lecture), `go vet ./...`, `go build ./...` (sans `-o` dans l'arbre, ou vers `/tmp`), `golangci-lint run` (sans `--fix`), `gosec ./...`, `go test -run … -count=1` **uniquement pour collecter une preuve d'observation** (jamais comme livrable, jamais avec génération de profil ou d'artefact dans l'arbre).
- Si une vérification exigerait une modification (ex. ajouter un test de reproduction), **ne pas la faire** : documenter le constat comme « hypothèse — à vérifier » avec le protocole de reproduction proposé.
- En cas de doute sur le caractère mutant d'une action : s'abstenir et le noter.

### Posture épistémique

- Conclusion d'abord, justification ensuite. Densité de pair technique senior ; pas de définitions des termes du domaine.
- **Toute affirmation factuelle doit être prouvée par `chemin/fichier.go:ligne`** (plage de lignes si pertinent). Pas de constat sans ancrage.
- Marqueurs d'incertitude obligatoires et explicites : `confirmé` (lu et vérifié dans le code), `probable` (forte présomption, raisonnement exposé), `hypothèse` (à instrumenter), `à vérifier`. Ne jamais présenter une hypothèse comme un fait.
- **Interdiction de fabrication** : aucun numéro de ligne, symbole, métrique ou citation inventé. En cas d'incertitude, le dire avec la piste de vérification.
- Langue : **français** (canadien), terminologie technique en anglais lorsque c'est l'usage établi (hot path, race, escape analysis, etc.).

---

## 1. Périmètre — couverture exhaustive obligatoire

Le dépôt comporte **~239 fichiers `.go` / ~35 500 LOC**, répartis en **24 répertoires de packages**. L'audit doit **couvrir chaque package** et justifier sa couverture. Volumétrie de référence (à revérifier, ne pas présumer figée) :

| Package | Fichiers | LOC (approx.) | Rôle |
|---|---|---|---|
| `cmd/fibcalc` | 2 | 257 | Point d'entrée CLI |
| `cmd/generate-golden` | 3 | 269 | Oracle golden tests indépendant |
| `internal/app` | 7 | 1720 | Lifecycle, dispatch, version |
| `internal/bigfft` | 36 | 7936 | Multiplication FFT, allocateur bump, pools, cache |
| `internal/calibration` | 17 | 2629 | Auto-calibration, micro-benchmarks |
| `internal/cli` | 16 | 1461 | Interface CLI, formatage |
| `internal/cli/completion` | 6 | 483 | Complétion shell (4 shells) |
| `internal/config` | 14 | 2137 | Parsing config, flags, env |
| `internal/errors` | 5 | 811 | Erreurs structurées |
| `internal/fibonacci` | 45 | 6802 | CŒUR : Fast Doubling, Matrix, FFT, Strassen, GMP |
| `internal/fibonacci/fibonaccitest` | 3 | 81 | Doubles de test |
| `internal/fibonacci/memory` | 7 | 620 | Arena, GCController, budget mémoire |
| `internal/fibonacci/threshold` | 8 | 1375 | Gestionnaire dynamique de seuils |
| `internal/format` | 6 | 436 | Formatage durées/nombres/ETA |
| `internal/metrics` | 5 | 444 | Indicateurs de performance |
| `internal/metrics/system` | 3 | 63 | Échantillonnage CPU/mém |
| `internal/orchestration` | 12 | 1192 | Exécution concurrente (errgroup) |
| `internal/parallel` | 3 | 161 | Quasi-mort (à statuer) |
| `internal/progress` | 6 | 958 | Observer + DTO progression |
| `internal/testutil` | 3 | 83 | Helpers de test partagés |
| `internal/tui` | 26 | 4460 | Dashboard TUI (Bubble Tea) |
| `internal/tui/component` | 1 | 45 | Composant TUI |
| `internal/ui` | 3 | 491 | Thèmes couleur |
| `test/e2e` | 2 | 540 | Tests bout-en-bout |

Inclure aussi dans le périmètre : `Makefile`, `.golangci.yml`, `.github/workflows/`, `go.mod`/`go.sum`, build tags (`gmp`), scripts, et la cohérence des docs (`docs/`, `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `CLAUDE.md`) **vis-à-vis du code réel** (les écarts doc/code sont des constats à part entière).

**Checklist de couverture** : `audit.md` doit se terminer par un tableau listant chaque package avec statut `Audité intégralement / Audité par échantillonnage / Non audité (justifier)`. Aucun package ne doit rester `Non audité` sans justification explicite.

---

## 2. Méthodologie

1. **Cartographie** — graphe de dépendances inter-packages ; vérifier l'étanchéité Clean Architecture (sens autorisé : `cmd → app → orchestration → fibonacci/bigfft → config/errors` ; `internal/` ne doit pas fuiter vers `cmd/`). Repérer cycles, dépendances inversées, couplages cachés.
2. **Passe package par package** — pour chaque package : responsabilité unique ? interfaces étroites (ISP) ? `doc.go` présent et exact ? cohésion ? surface publique justifiée ?
3. **Passe transverse** — concurrence, mémoire, gestion d'erreurs, sécurité, performance, tests : motifs récurrents à travers les packages.
4. **Vérification des bugs latents connus** (§4) — confirmer/infirmer chacun avec preuve `fichier:ligne`, statut actuel (corrigé / partiellement / non corrigé).
5. **Outillage de preuve** (lecture seule) — `go vet ./...`, `golangci-lint run`, `gosec ./...` : intégrer les sorties pertinentes comme preuves, mais **trier le bruit** (un dump de linter n'est pas un audit ; corréler, prioriser, expliquer l'impact).
6. **Cohérence doc/code** — confronter `CLAUDE.md`, `README.md`, `docs/` au code réel. Tout écart (ex. CI décrite absente alors qu'elle existe, ou l'inverse) est un constat de sévérité ≥ Moyenne (risque de décision sur base périmée).

---

## 3. Dimensions d'audit (9 axes — chacune obligatoirement traitée)

Pour chaque axe, ne pas se limiter à lister : **expliquer l'impact** (corruption, panique, fuite, régression perf, dette) et **les conditions de déclenchement**.

**D1 — Correction & bugs latents.** Use-after-free via aliasing de `[]big.Word`, off-by-one, invariants non garantis, conditions limites (n=0,1,2, n négatif, overflow d'index), erreurs ignorées (`errcheck`), valeurs de retour non vérifiées, `defer` mal ordonnés, comparaisons `big.Int` par pointeur.

**D2 — Concurrence & race safety.** Cycle de vie des goroutines (fuites, absence d'annulation), usage `context`, `errgroup`, sémaphores bornés (`NumCPU`), accès concurrents non protégés, fermeture de canaux, `sync.Pool` partagé entre goroutines, data races (signaler ce que `-race` couvre vs. ne couvre pas — rappel : la CI ne lance pas `-race` sous Windows), atomicité, double-checked locking, ordre de verrous.

**D3 — Mémoire, allocations & pooling.** `sync.Pool` `big.Int` (remise à zéro avant `Put` ? réutilisation correcte ?), `CalculationState`/`CalculationArena` (aliasing, `clearStateAliases`), allocateur bump FFT (réinitialisation, fragmentation, dépassement), buffers resizés perdus au retour en pool, escape analysis sur le hot path, allocations dans les boucles critiques, croissance non bornée des caches.

**D4 — Performance & hot paths.** `fastdoubling.go`, `doubling_framework.go`, multiplication FFT, exponentiation matricielle : allocations évitables, copies superflues, conversions, verrouillage sur chemin chaud, faux partage, calculs redondants, complexité algorithmique réelle vs. annoncée. Signaler tout risque de régression > 5 % (politique projet) et l'absence de garde-fou benchmark en CI.

**D5 — Architecture Clean & étanchéité des couches.** Respect du sens des dépendances, fuites de couches, packages « fourre-tout », responsabilités multiples (citer le nombre de responsabilités et le seuil), packages quasi-morts, code mort, duplication structurelle, `God object`/`God function`.

**D6 — Design d'API & interfaces.** ISP (`Multiplier`, `DoublingStepExecutor`, `Calculator`, `ProgressReporter`…), interfaces trop larges, abstractions prématurées ou manquantes, paramètres booléens drapeaux, retours d'erreur non typés, surface exportée non nécessaire, stabilité des contrats.

**D7 — Gestion d'erreurs.** `fmt.Errorf("%w", …)` systématique ? `panic`/`recover` utilisés à la place d'erreurs (en particulier `bigfft/fermat.go`) ? paniques non récupérées laissant un état global corrompu (ex. GC désactivé), erreurs avalées, sentinelles vs. types, messages exploitables.

**D8 — Tests, déterminisme & golden.** Couverture par package (chiffrer si possible), `t.Parallel()` (% réel vs. cible 100 %), tests non déterministes (temps, ordonnancement, `math/rand` non seedé), dépendance à l'horloge/au système, golden tests (`fibonacci_golden.json` **immuable**), property-based (`gopter`), tests de régression pour chaque bug latent, qualité des assertions, tests fragiles, tests désactivés/`skip`.

**D9 — Sécurité, build, CI/CD & chaîne d'appro.** Sorties `gosec` (G115/G304 exclus — valider que les exclusions restent justifiées), entrées non validées, chemins de fichiers, dépendances (`go.mod` : versions épinglées ? `golangci-lint version: latest` en CI = non reproductible ?), build tags (`gmp`), reproductibilité (`-trimpath`, ldflags), couverture CI réelle (jobs, OS, `-race` partiel, **absence de garde benchmark/coverage/gosec dans le pipeline**), `permissions` GitHub Actions, secrets, `go.sum` vérifié.

---

## 4. Spécificités FibGo à challenger explicitement

Ces points sont des **hypothèses de travail documentées** dans `CLAUDE.md` ; les **vérifier dans le code réel** (le code est la source de vérité ; `CLAUDE.md` peut être périmé) et statuer `confirmé / partiellement / infirmé / déjà corrigé` avec preuve :

| Réf. | Fichier indicatif | Hypothèse à vérifier |
|---|---|---|
| L1 | `internal/fibonacci/fastdoubling.go` | `clearStateAliases` contournable sur la branche `overLimit=true` → use-after-free latent. |
| L2 | `internal/fibonacci/memory/gc_control.go` | GC reste désactivé si panique entre `Begin()` et `End()` (pas panic-safe). |
| L3 | `internal/calibration/calibration.go` | `IsStale` jamais invoqué → profils de calibration obsolètes acceptés. |
| L4 | `internal/bigfft/pool.go` | `releaseWordSlice` perd silencieusement les buffers redimensionnés. |
| L5 | `internal/bigfft/fft_cache.go` | `putByKey` alloue de façon « eager » même en éviction. |

À challenger également : panic-safety globale des calculs N ≥ 1M (GC off), globaux dans `bigfft/` (recensement exhaustif + risque de réentrance/test isolé), thread-safety réelle du cache FFT LRU, nombre de responsabilités de `threshold/manager.go`, statut du package `parallel` (mort ? à supprimer ?), cohérence `CLAUDE.md` ↔ arbre réel (notamment l'état réel de la CI).

---

## 5. Modèle de sévérité

| Niveau | Critère |
|---|---|
| **Critique** | Corruption mémoire/données, use-after-free, race exploitable, faux résultat de calcul, ou blocage. Exploitable dans un usage nominal documenté. |
| **Haute** | État global corrompu sous condition (panique, profil périmé), fuite de ressource non bornée, régression perf > 5 % sur hot path, faille sécurité sans exploitation directe, absence de garde-fou critique (golden/CI). |
| **Moyenne** | Dette structurelle réelle (couche fuyante, God function, package quasi-mort), test non déterministe, écart doc/code induisant des décisions erronées, gestion d'erreur fragile. |
| **Basse** | Lisibilité, nommage, duplication mineure, commentaire trompeur, micro-optimisation, `nolint` non justifié. |

Chaque constat porte aussi un **effort de remédiation** : `S` (< 0,5 j), `M` (0,5–2 j), `L` (> 2 j).

---

## 6. Schéma de constat normalisé (obligatoire)

Chaque constat, identifiant unique `A-<NN>` (séquentiel, stable), présenté ainsi :

```
### A-NN — <titre court> [Sévérité] [Effort S/M/L] [Dimension Dx]
- **Emplacement** : `chemin/fichier.go:Lstart-Lend` (+ autres sites si récurrent)
- **Constat** : description factuelle, ancrée, marqueur d'incertitude.
- **Impact** : conséquence concrète + condition de déclenchement.
- **Preuve** : extrait minimal cité ou référence ligne + (le cas échéant) sortie d'outil.
- **Recommandation** : correctif visé (descriptif, NON appliqué), alternative si pertinente, condition qui renverse la reco.
```

---

## 7. Structure imposée de `audit.md`

1. **Résumé exécutif** (≤ 1 page) — verdict global, top 5 risques, posture de mise en production, dette agrégée.
2. **Méthodologie & couverture** — outils utilisés, périmètre réellement couvert, limites de l'audit.
3. **Matrice de risques** — tableau `ID | Titre | Dimension | Sévérité | Effort | Package`.
4. **Vérification des bugs latents connus (L1–L5)** — statut prouvé de chacun.
5. **Constats par package** — pour les 24 packages, dans l'ordre des couches ; chaque sous-section ouvre par une appréciation synthétique du package puis ses constats `A-NN`.
6. **Constats transverses** — concurrence, mémoire, erreurs, perf, sécurité/CI : motifs systémiques.
7. **Métriques** — LOC par package, fonctions hors seuils (cyclo > 15 / cognit > 30 / > 100 lignes / > 50 stmts), % `t.Parallel()`, packages sans `doc.go`, code mort, écarts doc/code.
8. **Feuille de route priorisée** — vagues recommandées (Critique → Basse), regroupées par dépendance, avec effort cumulé. Ne pas réordonner ni renuméroter un éventuel plan existant ; proposer un ordre.
9. **Checklist de couverture** — tableau par package (Audité intégralement / Échantillonné / Non audité + justification).
10. **Annexe** — commandes exécutées (lecture seule) et leurs versions, pour reproductibilité.

---

## 8. Garde-fous & critères de qualité de l'audit

- **Zéro modification** : si `audit.md` n'est pas le seul fichier nouveau/modifié à la fin, l'audit est invalide.
- **Traçabilité** : tout constat sans `fichier:ligne` est rejeté.
- **Signal sur bruit** : ne pas recopier les sorties de linters ; les corréler, dédupliquer, prioriser par impact réel.
- **Pas de faux positifs non qualifiés** : si un risque apparent est en réalité non exploitable (bornes contextuelles, mono-utilisateur CLI), le dire et le classer Basse/Info plutôt que de gonfler la sévérité.
- **Pas de recommandation appliquée** : décrire le correctif, ne jamais l'implémenter.
- **Exhaustivité prouvée** : la checklist §7.9 doit être complète et honnête.
- **Calibrage** : un audit utile hiérarchise ; un audit qui classe tout « Critique » n'aide pas.

## 9. Anti-objectifs (à NE PAS faire)

- Ne pas modifier, formater, « nettoyer » ou corriger quoi que ce soit dans le code.
- Ne pas exécuter de benchmark générant des artefacts (`pgo-profile`, `bench-versioned`) ni écrire dans `build/`.
- Ne pas réécrire ou « améliorer » la documentation existante (seulement signaler les écarts).
- Ne pas inventer de métriques précises non mesurées (dire « non mesuré » plutôt qu'estimer faussement).
- Ne pas produire un audit générique : chaque constat doit être spécifique à ce code.
