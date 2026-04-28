# 99 — Synthèse exécutive de l'audit FibGo

Audit complet du dépôt `github.com/agbru/fibcalc` (v3.0.0, ~37 000 LOC Go,
22 packages, Go 1.25/toolchain 1.26.2). Branche `main`, HEAD `4c8f0c1`,
date 2026-04-28. Agrégation des 22 rapports `01-*.md` à `72-*.md`.

## TL;DR

FibGo est un prototype académique **techniquement très mûr** : Clean
Architecture sans cycle (cf. `10-architecture.md`), couverture tests
**88,9 %** (cf. `50-tests-coverage.md`), 86 benchmarks et 5 cibles fuzz,
zéro `TODO`/`FIXME`, dépendances 100 % whitelistées (cf. `31-licenses.md`).
**Top 3 forces** : pooling/bump allocator zéro-alloc dans le hot path FFT,
discipline Conventional Commits/audits traçables, doc package-level dans
20/21 packages. **Top 3 risques** : aucune CI versionnée (`.github/`
absent — cf. `70-makefile.md`), `golangci-lint` v1.64 incompatible avec la
toolchain Go 1.26.2 (cf. `71-lint.md`), race detector et `govulncheck`
non vérifiables sur la machine d'audit Windows (cf. `51-race-detector.md`).

## Tableau de bord (score / 10)

| # | Phase | Score | Verdict |
|---|---|---:|---|
| 1 | Reconnaissance (inventaire, deps, git) | 8 | OK |
| 2 | Architecture (Clean, layers, ISP) | 9 | OK |
| 3 | Qualité de code (cyclo, dead code, conv.) | 8 | OK |
| 4 | Sécurité & licences | 8 | À surveiller |
| 5 | Performance (fibonacci, bigfft, conc., bench) | 9 | OK |
| 6 | Tests & race detector | 7 | À améliorer |
| 7 | Documentation (user, tech, doc.go) | 8 | OK |
| 8 | Outillage (Makefile, lint, cross-build) | 6 | À améliorer |
| **Moyenne pondérée** | | **7,9** | **Sain, à industrialiser** |

## Forces majeures

- **Architecture sans cycle** : 21 packages topologiquement triables ;
  aucune fuite `internal/ → cmd/`, domaine (`fibonacci`, `bigfft`) isolé
  des couches présentation (cf. `10-` et `11-layering.md`).
- **Performance disciplinée** : 24 benchmarks à 0 alloc/op confirment
  l'objectif « réduction GC > 95 % » du CLAUDE.md ; pools `statePool`,
  `matrixStatePool`, `BumpAllocator` correctement appariés Get/Put
  (cf. `40-` et `41-bigfft-arena.md`).
- **Concurrence maîtrisée** : 9/9 sites de spawn bornés par
  `WaitGroup`/`errgroup`, 3 sémaphores adaptatifs `NumCPU`, zéro
  goroutine orpheline (cf. `42-concurrency.md`).
- **Couverture & robustesse** : 88,9 % global, 4 packages à 100 %, 23
  golden values cross-algos avec oracle indépendant volontaire, 5 cibles
  fuzz avec corpus checké-in (cf. `50-tests-coverage.md`).
- **Supply chain saine** : 36 modules (10 directs + 26 indirects) tous
  sous MIT/BSD-3/Apache-2.0, aucun `replace`/`exclude`, `go.sum`
  intègre, aucun secret historique (cf. `30-` et `31-licenses.md`).

## Risques majeurs (par sévérité)

1. **CI absente** (`.github/` non versionné — cf. `70-makefile.md`,
   `50-tests-coverage.md`) : aucune garantie automatisée que `make test`,
   les golden tests, le fuzz, le race detector ou la cross-compilation
   passent à chaque PR. **Sévérité : élevée**.
2. **`golangci-lint` cassé localement** : binaire v1.64.8 (Go 1.25)
   refuse la cible Go 1.26.2 ; `make lint` ne tourne pas sur la machine
   d'audit (cf. `71-lint.md`). **Sévérité : élevée** tant que la CI ne
   compense pas.
3. **DoS local via `-n` non borné** : `config.go:147` accepte uint64 sans
   borne haute ; `--memory-limit` est opt-in (cf. `30-security.md` M-1).
   **Sévérité : modérée** (CLI mono-utilisateur).
4. **GC controller process-global non sérialisé**
   (`memory/gc_control.go:68-74`) : deux calculs concurrents avec GC
   contrôlé corrompent `originalGCPercent` (cf. `40-fibonacci-perf.md`
   F1). **Sévérité : modérée — correctness, pas perf** ; sans impact en
   CLI mono-process, bloquant si embarqué en service.
5. **Race detector et `govulncheck` non exécutés** : CGO indisponible
   sur Windows hôte (cf. `51-race-detector.md`) ; aucun appel réseau
   pendant l'audit (cf. `30-security.md`). Confiance race-free repose
   uniquement sur l'historique CI Linux (non vérifié).

## Maturité : code vs outillage

Le **code** est extrêmement abouti : refactor 4c8f0c1 a éliminé les god
packages, les seuils `.golangci.yml` (cyclo 15 / cogni 30 / funlen
100/50) ne sont violés qu'**une seule fois** (`ExecuteDoublingLoop`,
136 L, justifiée — cf. `20-code-quality.md`), 0 `TODO`/`FIXME` dans tout
le repo, wrapping `%w` systématique. **L'outillage** est sous-développé :
pas de CI, pas de pre-commit hooks, Makefile inutilisable nativement
sous Windows (`pgo-profile`, `bench-versioned`, `clean` sont POSIX-only),
`coverage.out` obsolète référence un fichier supprimé,
`docs/architecture/validation-report.md` figé au 2026-02-08 malgré le
refactor récent. Le **CHANGELOG** s'arrête à v1.0.0 alors que le tag
réel est v3.0.0 (cf. `60-doc-user.md`). Cet écart code-mûr/outillage-
faible est le **seul vrai risque structurel**.

## Recommandations à 30 jours (priorisées)

1. **(P0)** Créer `.github/workflows/ci.yml` minimal : matrix `{ubuntu,
   windows, macos} × Go 1.25/1.26` avec `lint + test -race + build-all`.
2. **(P0)** Mettre à jour `golangci-lint` :
   `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
   (≥ Go 1.26) et pinner via `tools.go` ou directive `tool` Go 1.24+.
3. **(P1)** Ajouter `govulncheck ./...` et `go-licenses check` à la CI
   (étape `security`) ; documenter le risque libgmp/LGPL pour les builds
   `-tags gmp` distribués statiquement (cf. `31-licenses.md`).
4. **(P1)** Borner `-n` côté config (ex. `MaxN = 1e10`) et rendre
   `--memory-limit` obligatoire au-delà d'un seuil ; assainir `-output`
   et `-calibration-profile` via `filepath.Clean` (cf. `30-security.md`).
5. **(P1)** Sérialiser `GCController.Begin/End` via un mutex
   package-level dans `internal/fibonacci/memory/` ou refuser
   explicitement les calculs concurrents à GC contrôlé (cf. `40-` F1).
6. **(P2)** Synchroniser la documentation : entrées CHANGELOG `[2.0.0]`,
   `[3.0.0]`, flag `--gc-control` dans README/`.env.example`, 4 écarts
   `dependency-graph.mermaid` (cf. `10-`, `60-`, `61-doc-tech.md`).
7. **(P2)** Introduire **Mage** (`magefile.go`) en parallèle du
   Makefile pour rétablir l'ergonomie sur Windows ; à défaut, fournir
   un `make.ps1` minimal `{build, test, lint}` (cf. `70-makefile.md`).

## Hors-scope de cet audit

- **`go test -race ./...`** : non exécuté faute de toolchain C sur la
  machine d'audit Windows. Validation effective déléguée à une future CI
  Linux (cf. `51-race-detector.md`).
- **`govulncheck ./...`** : non exécuté (audit hors-réseau). Aucune
  vulnérabilité connue détectée par heuristique manuelle, mais la
  vérification CVE temps-réel reste à faire (cf. `30-security.md`).
- **`golangci-lint run` complet** : impossible (incompatibilité Go
  1.25 vs 1.26.2). Conclusions qualité de `20-code-quality.md` reposent
  sur inspection manuelle + `go vet` (qui passe à 0 finding).
- **Benchmarks statistiquement robustes** : `43-bench-baseline.md` a
  tourné en `-benchtime=1x` ; ré-exécution `-benchtime=3s -count=5` +
  `benchstat` reste à faire pour une baseline de référence.
- **Refactoring** : explicitement exclu par PLAN.md §5 et la directive
  « Codebase mature » du CLAUDE.md. Tous les findings sont consignés
  dans `99-backlog.md`, non implémentés.
