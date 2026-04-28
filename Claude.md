# CLAUDE.md — FibGo (FibCalc)

Calculateur Fibonacci haute performance en Go. Prototype académique démontrant les patterns d'ingénierie logicielle avancés : Clean Architecture, pooling mémoire, parallélisme adaptatif et optimisation PGO.

## Projet

- **Module** : `github.com/agbru/fibcalc`
- **Go** : 1.25.0+
- **Licence** : Apache 2.0
- **Taille** : ~31 900 lignes Go, 108 fichiers source, 95 fichiers de test, 22 packages Go

## Architecture (Clean Architecture, 4 couches)

```
cmd/
  fibcalc/           # Point d'entrée CLI principal
  generate-golden/   # Générateur de données de test
internal/
  app/               # Cycle de vie applicatif, dispatch, version
  bigfft/            # Multiplication FFT (Schönhage-Strassen), allocateur bump
  calibration/       # Auto-calibration adaptative, micro-benchmarks
  cli/               # Interface CLI, formatage, complétion shell
  config/            # Parsing config, flags, variables d'environnement
  errors/            # Types d'erreurs structurées (ConfigError, CalcError)
  fibonacci/         # CŒUR : Fast Doubling, Matrix Exp., FFT, Strassen, GMP
    memory/          # Arena, GCController, budget mémoire
    threshold/       # Gestionnaire dynamique de seuils (FFT/parallèle)
    fibonaccitest/   # Doubles de test pour CoreCalculator
  format/            # Formatage durées, nombres, ETA
  metrics/           # Indicateurs de performance, monitoring mémoire
  orchestration/     # Exécution concurrente, agrégation résultats
  parallel/          # Utilitaires d'exécution parallèle
  progress/          # Rapports de progression (pattern observer)
  sysmon/            # Monitoring mémoire système
  testutil/          # Utilitaires de test partagés
  tui/               # Dashboard TUI interactif (Bubble Tea)
  ui/                # Thèmes couleur et styling
docs/
  architecture/      # Diagrammes C4 (Mermaid)
  algorithms/        # Documentation mathématique des algorithmes
```

## Algorithmes implémentés

1. **Fast Doubling** (défaut) — O(log n), identité F(2k) = F(k)(2F(k+1) - F(k))
2. **Matrix Exponentiation** — O(log n), algorithme de Strassen pour grandes matrices
3. **FFT (Schönhage-Strassen)** — Seuil adaptatif (~500k bits par défaut)
4. **GMP** (optionnel, build tag) — Backend GNU Multiple Precision

## Patterns de performance

- **sync.Pool** pour recyclage `big.Int` (réduction GC >95%)
- **Allocateur bump** pour FFT (O(1), zéro fragmentation)
- **GC désactivé** pendant calculs N ≥ 1M
- **Parallélisme adaptatif** via sémaphore (`NumCPU()*2`)
- **Cache FFT** (LRU thread-safe, 15-30% speedup)
- **PGO** (Profile-Guided Optimization) supporté

## Commandes essentielles

```bash
make all             # clean + build + test
make test            # Tests avec race detector
make test-short      # Tests rapides
make coverage        # Rapport couverture HTML
make benchmark       # Benchmarks
make lint            # golangci-lint (22 linters)
make build-pgo       # Build avec PGO
make build-all       # Cross-compilation (linux, windows, macOS)
```

## Conventions de code

- **Packages par responsabilité** (pas par feature)
- **Interfaces étroites** (ISP) : `Multiplier`, `DoublingStepExecutor`
- **Erreurs structurées** : `fmt.Errorf("%w", err)`
- **Tests parallèles** : `t.Parallel()` systématique
- **Race detector** activé par défaut dans CI
- **Complexité cyclomatique** max 15, cognitive max 30
- **Longueur fonction** max 100 lignes / 50 statements

## Lignes directrices comportementales

Principes généraux qui complètent les directives projet ci-dessous. Compromis : favorise la prudence sur la vitesse — pour les tâches triviales, utiliser le jugement.

### 1. Réfléchir avant de coder

**Ne pas supposer. Ne pas masquer la confusion. Exposer les compromis.**

Avant d'implémenter :
- Énoncer les hypothèses explicitement. En cas de doute, demander.
- Si plusieurs interprétations existent, les présenter — ne pas choisir silencieusement.
- Si une approche plus simple existe, le dire. Pousser en arrière si justifié.
- Si quelque chose n'est pas clair, s'arrêter. Nommer la confusion. Demander.

### 2. Simplicité d'abord

**Code minimum qui résout le problème. Rien de spéculatif.**

- Pas de fonctionnalités au-delà de ce qui est demandé.
- Pas d'abstractions pour du code à usage unique.
- Pas de « flexibilité » ou « configurabilité » non demandée.
- Pas de gestion d'erreurs pour des scénarios impossibles.
- Si 200 lignes peuvent en faire 50, réécrire.

Question test : « Un ingénieur senior dirait-il que c'est sur-compliqué ? » Si oui, simplifier.

### 3. Modifications chirurgicales

**Toucher uniquement ce qui est nécessaire. Nettoyer uniquement son propre désordre.**

Lors d'édition de code existant :
- Ne pas « améliorer » le code adjacent, les commentaires ou le formatage.
- Ne pas refactorer ce qui n'est pas cassé.
- Respecter le style existant, même si vous feriez différemment.
- Si vous remarquez du code mort non lié, le mentionner — ne pas le supprimer.

Lorsque vos changements créent des orphelins :
- Supprimer imports/variables/fonctions devenus inutilisés *par* vos changements.
- Ne pas supprimer du code mort préexistant sans demande.

Test : chaque ligne modifiée doit tracer directement à la demande utilisateur.

### 4. Exécution dirigée par l'objectif

**Définir des critères de succès. Itérer jusqu'à vérification.**

Transformer les tâches en objectifs vérifiables :
- « Ajouter une validation » → « Écrire les tests pour entrées invalides, puis les faire passer »
- « Corriger le bug » → « Écrire un test qui le reproduit, puis le faire passer »
- « Refactorer X » → « S'assurer que les tests passent avant et après »

Pour les tâches multi-étapes, énoncer un plan bref :

```
1. [Étape] → vérifier : [contrôle]
2. [Étape] → vérifier : [contrôle]
3. [Étape] → vérifier : [contrôle]
```

Des critères de succès forts permettent d'itérer de manière autonome. Des critères faibles (« faire que ça marche ») exigent des clarifications constantes.

---

**Ces lignes directrices fonctionnent si :** moins de changements inutiles dans les diffs, moins de réécritures pour cause de sur-complication, et les questions de clarification arrivent avant l'implémentation plutôt qu'après les erreurs.

## Directives pour Claude

Directives spécifiques à ce projet (FibGo). À lire en complément des lignes directrices comportementales ci-dessus.

1. **Performance critique** : Ce projet optimise au niveau mémoire/GC. Ne pas introduire d'allocations inutiles.
2. **Tests obligatoires** : Tout changement algorithmique doit passer les golden tests (`testdata/fibonacci_golden.json`).
3. **Cohérence architecturale** : Respecter la séparation en couches. Les packages `internal/` ne doivent pas fuiter vers `cmd/` directement.
4. **Linting** : `make lint` doit passer. Respecter les seuils de complexité configurés dans `.golangci.yml`.
5. **Documentation** : Chaque package a un `doc.go`. Maintenir les commentaires de package.
6. **Concurrence** : Utiliser `sync.Pool`, `errgroup`, sémaphores bornés. Pas de goroutines sans contrôle de cycle de vie.
7. **Modifications chirurgicales** : Ce codebase est mûr — ne pas refactorer sans demande explicite.
