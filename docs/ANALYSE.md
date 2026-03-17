# FibGo / FibCalc — Analyse Technique

> **Date :** 2026-03-15  
> **Portée :** analyse complète du dépôt `github.com/agbru/fibcalc` (Go 1.25.0)  
> **Méthode :** revue statique du code source (105 fichiers Go, 89 fichiers de test, 16 packages internes)

---

## 1) Diagnostic des Lacunes et Dettes Techniques

### 1.1 Architecture et Conception

| # | Lacune | Sévérité | Localisation | Détail |
|---|--------|----------|-------------|--------|
| D-01 | **`coreCalculator` non exporté bloque la testabilité externe** | 🟡 Moyenne | `internal/fibonacci/registry.go` | L'interface `coreCalculator` est unexportée, ce qui empêche l'utilisation de `mockgen` et force la création de mocks manuels. Le commentaire en tête du fichier le reconnaît explicitement. |
| D-02 | **Variable globale mutable `globalFactory`** | 🟡 Moyenne | `internal/fibonacci/registry.go:235` | Singleton mutable partagé via `GlobalFactory()`. En cas d'utilisation concurrente dans des tests parallèles, cela peut provoquer des effets de bord (data race sur l'état du registre). |
| D-03 | **Variable globale mutable `squareSymmetricMatrixFunc`** | 🟡 Moyenne | `internal/fibonacci/matrix_framework.go:19` | Variable de type fonction utilisée pour le mocking de tests, mais accessible globalement — risque de data race en tests parallèles. |
| D-04 | **`ResultPresenter` trop large** | 🟠 Faible | `internal/orchestration/interfaces.go` | L'interface `ResultPresenter` combine la présentation de tableau et de résultat individuel. `CLIResultPresenter` implémente aussi `ErrorHandler`, créant un couplage implicite entre présentation et gestion d'erreurs. |
| D-05 | **Absence de contexte dans la signature `Register`** | 🟠 Faible | `internal/fibonacci/registry.go` | `Register()` retourne `error` mais n'échoue jamais (toujours `nil`). Cette signature suggère une extensibilité future non réalisée. |

### 1.2 Concurrence et Sécurité

| # | Lacune | Sévérité | Localisation | Détail |
|---|--------|----------|-------------|--------|
| D-06 | **Sémaphore de tâche initialisée paresseusement via `sync.Once`** | 🟠 Faible | `internal/fibonacci/common.go:22-35` | `taskSemaphore` est une variable globale initialisée paresseusement. Le sizing `NumCPU*2` est fixe et ne s'adapte pas à la charge réelle ni aux contraintes mémoire. |
| D-07 | **`errgroup` absorbe les erreurs dans le multi-calculateur** | 🟡 Moyenne | `internal/orchestration/orchestrator.go:59-66` | Chaque goroutine dans l'`errgroup` retourne `nil` intentionnellement (l'erreur est stockée dans `results[idx]`), ce qui rend le `errgroup.Wait()` toujours réussi. L'erreur cancellation du contexte n'est donc pas propagée via le groupe. |
| D-08 | **`executeParallel3` ne respecte pas le sémaphore** | 🟡 Moyenne | `internal/fibonacci/common.go:96-114` | Les 3 goroutines de `executeParallel3` s'exécutent sans acquérir de token du sémaphore, contrairement à `executeTasks` et `executeMixedTasks`. Cela peut entraîner un dépassement temporaire de la limite de concurrence. |

### 1.3 Gestion des Erreurs

| # | Lacune | Sévérité | Localisation | Détail |
|---|--------|----------|-------------|--------|
| D-09 | **Utilisation de `panic` dans `NewCalculator`** | 🟡 Moyenne | `internal/fibonacci/calculator.go:76` | `NewCalculator(nil)` provoque un `panic` plutôt qu'une erreur explicite. En Go idiomatique, les constructeurs devraient retourner `(T, error)`. |
| D-10 | **Écriture directe sur `os.Stderr` dans `saveResultIfNeeded`** | 🟠 Faible | `internal/app/calculate.go:200` | `fmt.Fprintf(os.Stderr, ...)` contourne le `ErrWriter` injecté dans `Application`, brisant le pattern d'injection de dépendances. |
| D-11 | **`ErrorHandler` interface avec un seul implémenteur** | 🟠 Faible | `internal/orchestration/interfaces.go:94-96` | L'interface `ErrorHandler` n'a qu'un seul implémenteur (`CLIResultPresenter`). L'abstraction est prématurée et ajoute de l'indirection sans bénéfice actuel. |

### 1.4 Configuration

| # | Lacune | Sévérité | Localisation | Détail |
|---|--------|----------|-------------|--------|
| D-12 | **Flags dupliqués sans alias propre** | 🟠 Faible | `internal/config/config.go:145-164` | Les alias courts (`-v`/`-verbose`, `-d`/`-details`, etc.) sont implémentés via des appels séparés à `fs.BoolVar` partageant la même variable, ce qui est fonctionnel mais génère des entrées dupliquées dans `--help`. |
| D-13 | **Valeur par défaut de N très élevée** | 🟠 Info | `internal/config/config.go:25` | `DefaultN = 100_000_000` est un défaut agressif qui peut surprendre les nouveaux utilisateurs (calcul de F(100M) prend plusieurs secondes). |

### 1.5 Tests et Qualité

| # | Lacune | Sévérité | Localisation | Détail |
|---|--------|----------|-------------|--------|
| D-14 | **Absence de tests d'intégration pour la boucle complète app → orchestration → fibonacci** | 🟡 Moyenne | `test/e2e/` | Les tests e2e existent mais couvrent principalement les codes de sortie. Pas de vérification de la cohérence des résultats pour des N moyens via le binaire compilé. |
| D-15 | **Pas de `golangci-lint` en CI visible** | 🟠 Faible | Racine du projet | `.golangci.yml` existe mais aucun fichier CI (`.github/workflows/`, `.gitlab-ci.yml`, etc.) n'est visible dans le dépôt. Le linting et les tests ne sont pas automatisés. |

### 1.6 Documentation

| # | Lacune | Sévérité | Localisation | Détail |
|---|--------|----------|-------------|--------|
| D-16 | **Redondance entre `docs/ARCH.md` et `docs/architecture/README.md`** | 🟡 Moyenne | `docs/` | Les deux fichiers couvrent l'architecture avec un overlap significatif (packages, interfaces, ADR, flux de données). Cela crée un risque de divergence. |
| D-17 | **Commentaires `doc.go` insuffisants dans certains packages** | 🟠 Faible | `internal/sysmon/`, `internal/format/`, `internal/testutil/` | Ces packages n'ont pas de `doc.go` ou leur documentation est minimale. |

---

## 2) Recommandations d'Amélioration Priorisées

### Priorité Haute (Impact élevé, effort modéré)

#### ✅ R-01 : Harmoniser la documentation architecturale — *Réalisé*

**Justification :** `docs/ARCH.md` et `docs/architecture/README.md` couvrent le même sujet avec des informations partiellement redondantes et partiellement complémentaires.

**Actions :**
1. ~~Faire de `docs/ARCH.md` le document « vue d'ensemble rapide » (entrée principale)~~ ✅
2. ~~Faire de `docs/architecture/README.md` la référence détaillée (diagrammes C4, ADR complets, flow mermaid)~~ ✅
3. ~~Ajouter des liens croisés explicites entre les deux~~ ✅

**Effort :** ~1h | **Impact :** 🟡 Maintenabilité documentaire

---

#### ✅ R-02 : Corriger `executeParallel3` pour respecter le sémaphore (D-08) — *Réalisé*

**Justification :** `executeTasks` et `executeMixedTasks` acquièrent des tokens avant exécution, mais `executeParallel3` ne le fait pas, créant une incohérence dans le contrôle de concurrence.

**Actions :**
1. ~~Modifier `executeParallel3` pour acquérir un token du sémaphore avant chaque opération~~ ✅
2. Ajouter un test unitaire vérifiant le respect de la limite de concurrence (couvert par les tests existants)

**Effort :** ~30min | **Impact :** 🟡 Stabilité concurrente

---

### Priorité Moyenne (Impact modéré, effort variable)

#### ✅ R-03 : Exporter `coreCalculator` ou fournir une interface de test (D-01) — *Réalisé*

**Justification :** Les utilisateurs et les tests externes ne peuvent pas implémenter de calculateurs personnalisés ni mocker proprement la couche interne.

**Actions :**
1. Option A : exporter l'interface comme `CoreCalculator`
2. Option B : fournir une `TestFactory` enrichie dans `internal/fibonacci/testing.go`
3. Documenter le workflow d'extension dans CONTRIBUTING.md

**Effort :** ~1h | **Impact :** 🟡 Extensibilité et testabilité

---

#### ✅ R-04 : Remplacer les variables globales mutables par de l'injection (D-02, D-03) — *Réalisé*

**Justification :** `globalFactory` et `squareSymmetricMatrixFunc` sont des singletons mutables exposés globalement, risquant des data races dans les tests parallèles.

**Actions :**
1. Passer `globalFactory` comme dépendance injectée (déjà partiellement fait via `WithFactory`)
2. Transformer `squareSymmetricMatrixFunc` en champ de `MatrixFramework` injectable
3. Utiliser `t.Parallel()` dans les tests pour valider l'absence de data races

**Effort :** ~2h | **Impact :** 🟡 Robustesse des tests

---

#### ✅ R-05 : Remplacer `panic` par `(T, error)` dans `NewCalculator` (D-09) — *Réalisé*

**Justification :** `panic` dans un constructeur est non-idiomatique en Go et rend le code appelant fragile.

**Actions :**
1. Modifier la signature en `NewCalculator(core coreCalculator) (Calculator, error)`
2. Actualiser tous les sites d'appel (dans `DefaultFactory` et tests)
3. Optionnel : conserver un `MustNewCalculator` pour les cas `init()`

**Effort :** ~1h | **Impact :** 🟡 Idiomaticité et robustesse

---

#### ✅ R-06 : Enrichir les tests e2e (D-14) — *Réalisé*

**Justification :** Les tests e2e actuels vérifient surtout les codes de sortie. Il manque des vérifications de correction des résultats via le binaire.

**Actions :**
1. Ajouter des cas e2e comparant la sortie à des golden values pour N modérés (1000, 10000)
2. Tester les modes `--last-digits`, `--quiet`, `--output`
3. Tester les cas d'erreur : timeout, N invalide, mémoire insuffisante

**Effort :** ~3h | **Impact :** 🟡 Couverture de non-régression

---

### Priorité Basse (Amélioration continue)

#### R-07 : Simplifier `ResultPresenter` / `ErrorHandler` (D-04, D-11)

**Actions :** Fusionner `ErrorHandler` dans `ResultPresenter` ou le supprimer si un seul implémenteur existe. Diviser `ResultPresenter` en interfaces plus fines si de nouveaux formats de sortie (JSON, CSV) sont envisagés.

**Effort :** ~1h | **Impact :** 🟠 Clarté du design

---

#### R-08 : Corriger l'écriture directe sur `os.Stderr` (D-10)

**Actions :** Remplacer `fmt.Fprintf(os.Stderr, ...)` par `fmt.Fprintf(a.ErrWriter, ...)` dans `saveResultIfNeeded`.

**Effort :** ~10min | **Impact :** 🟠 Cohérence DI

---

#### R-09 : Ajouter `doc.go` aux packages manquants (D-17)

**Actions :** Créer `doc.go` pour `internal/sysmon`, `internal/format`, `internal/testutil` avec une description concise du rôle du package.

**Effort :** ~20min | **Impact :** 🟠 Documentation

---

## 3) Pistes d'Optimisation Concrètes

### 3.1 Performance

#### ✅ O-01 : Optimiser le seuil FFT par profilage systématique — *Réalisé*

**Situation actuelle :** Le `DefaultFFTThreshold` (500K bits) est une estimation conservatrice. La calibration automatique (`AutoCalibrate`) le raffine désormais de manière granulaire.

**Actions :**
1. ~~Implémenter un benchmark dédié balayant la plage 200K–1M bits par pas de 50K~~ ✅
2. ~~Générer une courbe crossover FFT/Karatsuba par architecture (amd64/arm64)~~ ✅
3. ~~Intégrer les résultats dans les profils de calibration avec un score de confiance~~ ✅

**Gain estimé :** 5-15% sur les cas proches du seuil de crossover

---

#### ✅ O-02 : Exploiter le transform caching de manière plus agressive — *Réalisé*

**Situation actuelle :** Le cache FFT (`fft_cache.go`) est implémenté et réduit les calculs redondants lors des itérations de la boucle de doubling.

**Actions :**
1. ~~Dimensionner le cache en fonction de N (log₂(N) entrées maximum)~~ ✅
2. ~~Explorer le partage de transforms entre `Sqr(FK)` et `Sqr(FK1)` quand les tailles sont proches~~ ✅
3. ~~Mesurer le hit-rate du cache et ajuster dynamiquement~~ ✅

**Gain estimé :** 10-20% pour les calculs FFT-dominés

---

#### O-03 : Implémenter le pipelining des opérations MSB→LSB

**Situation actuelle :** Chaque itération de la boucle doubling attend la fin des 3 multiplications avant de passer à l'itération suivante.

**Optimisation proposée :**
1. Pour les dernières itérations (operands les plus gros), pipeliner la multiplication FK×FK1 (qui n'a pas de dépendance data avec les squaring) avec l'étape suivante
2. Utiliser un double-buffering des `CalculationState` pour permettre le chevauchement

**Gain estimé :** 5-10% sur les très grands N (>50M)

**Complexité :** 🔴 Élevée — nécessite une refonte partielle du framework de doubling

---

#### O-04 : Pré-calcul des plans FFT

**Situation actuelle :** Les plans FFT sont recalculés à chaque appel. Pour les calculs itératifs où les tailles des opérandes sont prévisibles (doublement à chaque itération), les plans pourraient être pré-calculés.

**Optimisation proposée :**
1. Créer un `FFTPlanCache` basé sur la taille en mots
2. Pré-calculer les plans pour les tailles attendues (séquence géométrique)
3. Partager les plans entre `Mul` et `Sqr` de même taille

**Gain estimé :** 3-8% sur les calculs FFT-dominés

---

### 3.2 Maintenabilité

#### O-05 : Extraire un package `internal/lifecycle` de `internal/app`

**Situation actuelle :** `internal/app` mélange la construction d'application (DI, options) avec la logique d'exécution (context lifecycle, signal handling, mode dispatch).

**Optimisation proposée :**
1. Extraire `SetupContext()` / `SetupSignalHandler()` dans un package `internal/lifecycle`
2. Réduire `app.go` à la construction et au dispatch pur
3. Permettre la réutilisation du lifecycle dans des variantes du binaire (bibliothèque)

**Bénéfice :** Meilleure séparation des responsabilités, testabilité accrue

---

#### O-06 : Typer les modes d'exécution

**Situation actuelle :** Le mode d'exécution est déterminé par cascade de conditions booléennes dans `Application.Run()` (completion → calibration → TUI → CLI).

**Optimisation proposée :**
1. Introduire un type `ExecutionMode` enum avec `ModeCompletion`, `ModeCalibration`, `ModeTUI`, `ModeCLI`, `ModeLastDigits`
2. Résoudre le mode dans `New()` ou `Run()` en un seul point
3. Utiliser un switch/dispatch table au lieu de cascades if/else

**Bénéfice :** Code plus lisible, extensibilité facilitée pour de nouveaux modes

---

#### O-07 : Consolider la gestion des loggers package-level

**Situation actuelle :** Plusieurs packages ont leur propre logger `zerolog.Nop()` avec des setters dédiés (`SetTaskLogger`, `SetRegistryLogger`). Ce pattern se répète sans abstraction commune.

**Optimisation proposée :**
1. Créer un pattern `internal/log` centralisant la configuration des loggers
2. Utiliser un `zerolog.Logger` partagé avec des sous-loggers par package
3. Configurer le niveau de log une seule fois au démarrage

**Bénéfice :** Configuration centralisée, cohérence, réduction du code boilerplate

---

### 3.3 Scalabilité

#### O-08 : Support d'un mode serveur / API HTTP

**Situation actuelle :** FibCalc est exclusivement un outil CLI/TUI. Il n'y a pas de moyen d'exposer le calcul via une API réseau.

**Optimisation proposée :**
1. Créer un `cmd/fibcalc-server` exposant un endpoint REST/gRPC
2. Réutiliser `internal/orchestration` pour l'exécution
3. Exposer les résultats en JSON/Protobuf
4. Ajouter des métriques Prometheus pour le monitoring

**Bénéfice :** Permet l'intégration dans des architectures distribuées, utilisation en tant que microservice

**Complexité :** 🟡 Moyenne — l'architecture Clean facilite cette extension

---

#### O-09 : Cache de résultats pour les N fréquemment calculés

**Situation actuelle :** Chaque appel recalcule F(N) intégralement, même si le même N a été calculé récemment.

**Optimisation proposée :**
1. Implémenter un cache LRU en mémoire pour les résultats (clé : N + algo)
2. Option : cache disque pour les résultats volumineux
3. Configurable via `--cache-size` et `FIBCALC_CACHE_SIZE`

**Bénéfice :** Latence quasi-nulle pour les requêtes répétées

---

#### O-10 : Distribution du calcul sur plusieurs machines

**Situation actuelle :** Le calcul est limité à une seule machine. Pour des N extrêmement grands (>10⁹), le temps de calcul et la mémoire deviennent prohibitifs.

**Optimisation proposée :**
1. Implémenter un mode de calcul distribué basé sur la décomposition récursive de la matrice Q
2. Utiliser gRPC pour la communication inter-nœuds
3. Chaque nœud calcule un sous-problème F(N/2^k) et les résultats sont combinés

**Gain estimé :** Scalabilité quasi-linéaire pour les très grands N

**Complexité :** 🔴 Élevée — nécessite une refonte significative de l'orchestration

---

### 3.4 Récapitulatif des Priorités

| # | Catégorie | Titre | Impact | Effort | Priorité |
|---|-----------|-------|--------|--------|----------|
| ✅ R-01 | Documentation | Harmoniser ARCH.md / README.md | 🟡 Moyen | ~1h | Réalisé |
| ✅ R-02 | Concurrence   | Fix `executeParallel3` sémaphore | 🟡 Moyen | ~30min | Réalisé |
| ✅ R-03 | Architecture  | Exporter `coreCalculator`        | 🟡 Moyen | ~1h | Réalisé |
| ✅ R-04 | Architecture  | Remplacer globales mutables      | 🟡 Moyen | ~2h | Réalisé |
| ✅ R-05 | Code          | Remplacer `panic` par erreur     | 🟡 Moyen | ~1h | Réalisé |
| ✅ R-06 | Tests         | Enrichir tests e2e               | 🟡 Moyen | ~3h | Réalisé |
| ✅ O-01 | Performance   | Profilage seuil FFT              | 🟡 Moyen | ~4h | Réalisé |
| ✅ O-02 | Performance   | Transform caching agressif       | 🟡 Moyen | ~6h | Réalisé |
| O-05 | Maintenabilité | Extraire lifecycle | 🟠 Faible | ~2h | Basse |
| O-06 | Maintenabilité | Typer les modes | 🟠 Faible | ~1h | Basse |
| O-07 | Maintenabilité | Consolider loggers | 🟠 Faible | ~2h | Basse |
| O-08 | Scalabilité | Mode serveur HTTP | 🟡 Moyen | ~8h | Basse |
| O-09 | Scalabilité | Cache de résultats | 🟠 Faible | ~4h | Basse |
| O-10 | Scalabilité | Calcul distribué | 🟡 Moyen | ~40h | Basse |

---

## Annexe : Points Forts du Projet

Il est essentiel de noter que le codebase FibCalc est globalement **très bien conçu**. Les points forts majeurs :

- ✅ **Clean Architecture rigoureuse** avec des frontières de dépendances bien définies
- ✅ **14 design patterns** correctement implémentés et justifiés par les besoins
- ✅ **Couverture de tests impressionnante** : unit, golden, fuzz, property-based, benchmark, e2e
- ✅ **Performance state-of-the-art** : arenas, pools, GC control, FFT, SIMD dispatch
- ✅ **Documentation très riche** : 20+ fichiers de documentation, ADR, diagrammes Mermaid/C4
- ✅ **Extensibilité validée** : le pattern Factory+Strategy a permis l'ajout de 4 algorithmes et 2 UI (CLI+TUI)
- ✅ **API ergonomique** : `--help` détaillé, complétion shell, env vars, calibration automatique
