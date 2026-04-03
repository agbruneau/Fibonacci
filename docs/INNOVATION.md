# FibCalc — Pistes d’innovation et d’amélioration

Ce document complète [ARCH.md](ARCH.md) : il propose des évolutions **actionnables**, classées par axe, sans remplacer les guides opérationnels ([CONTRIBUTING.md](../CONTRIBUTING.md), [PERFORMANCE.md](PERFORMANCE.md), [CALIBRATION.md](CALIBRATION.md)). Le plan d’exécution priorisé est dans [INNOVEPLAN.md](INNOVEPLAN.md).

**Légende — complexité :** faible · moyenne · élevée  
**Légende — risque :** impact possible sur perfs, stabilité API, ou maintenance.

Les priorités **P1–P4**, les livrables, critères de fin et le **tableau de suivi des tâches** (état du dépôt) sont dans **[INNOVEPLAN.md](INNOVEPLAN.md#suivi-des-tâches)**. Le suivi y indique l’état courant (y compris P3 heuristique CPU et la décision documentée P4 sur les backends de recherche).

---

## 1) Fiabilité et contexte

| Piste | Intérêt | Complexité | Risque |
|------|---------|------------|--------|
| **Audit systématique `context`** sur les chemins longs (calibration, [internal/calibration](../internal/calibration), boucles FFT / doubling). | Annulation propre, deadlines cohérentes avec [internal/app](../internal/app). | moyenne | Points de rendez-vous supplémentaires peuvent légèrement impacter les perfs si mal placés. |
| **Erreurs enrichies** : rattacher seuils effectifs, estimation mémoire, extrait de config (sans secrets) aux `CalculationError` / [internal/errors](../internal/errors). | Debug utilisateur et support sans relancer en verbose. | faible | Messages plus verbeux ; attention i18n si un jour multilingue. |

---

## 2) Tests et qualité

| Piste | Intérêt | Complexité | Risque |
|------|---------|------------|--------|
| **Réduire la friction des mocks** : interface de test exportée ou façade pour le cœur non exporté (`coreCalculator`, déjà noté dans [ARCH.md](ARCH.md) §11). | Plus de tests unitaires ciblés sans hacks. | moyenne | Légère dilution des frontières `internal`. |
| **Tests de contrat** entre [internal/orchestration](../internal/orchestration) et implémentations `Calculator` (invariants : pas de panic, respect `context`, canal progrès fermé). | Régressions détectées plus tôt lors d’évolution des algos. | moyenne | Coût CI si trop de scénarios lourds — garder des `N` modestes. |
| **CI multi-`GOOS` / multi-arch** pour [internal/bigfft](../internal/bigfft) (dont `arith_amd64` vs générique). | Confiance sur Windows/Linux/macOS, ARM. | faible à moyenne | Temps de pipeline ; échecs flakey à isoler. |

---

## 3) Performance et portabilité

| Piste | Intérêt | Complexité | Risque |
|------|---------|------------|--------|
| **Bench comparatifs versionnés** (régressions dans CHANGELOG ou artefact CI) en s’appuyant sur le [Makefile](../Makefile) / PGO existants. | Historique objectif des gains. | faible | Nécessite runner dédié pour stabilité. |
| **Backends arithmétiques** : extension au-delà de GMP (`-tags gmp`, [docs/algorithms/GMP.md](algorithms/GMP.md)) — évaluation flint, etc. | Point de comparaison recherche. | élevée | Charge de build et licences. |
| **Affinage heuristique** : seuils pilotés par type de CPU (cache L3, AVX-512) au lieu de généralisations par `runtime.NumCPU` uniquement, en s’appuyant sur [internal/config/thresholds.go](../internal/config/thresholds.go) et calibration. | Gains sur machines hétérogènes. | moyenne | Complexité de calibration et de validation. |

---

## 4) Expérience utilisateur

| Piste | Intérêt | Complexité | Risque |
|------|---------|------------|--------|
| **Mode « quiet machine »** unifié (pas de spinner ANSI, sortie stdout propre) aligné sur [internal/cli](../internal/cli) et usages scripts. | Pipelines shell prévisibles. | faible | — |
| **TUI** ([internal/tui](../internal/tui), [TUI_GUIDE.md](TUI_GUIDE.md)) : thèmes contrastés, réduction de dépendance aux couleurs seules pour l’état. | Accessibilité terminal. | moyenne | Cohérence visuelle avec [internal/ui](../internal/ui). |
