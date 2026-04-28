# 61 — Documentation technique

## docs/ARCH.md (785 L)
- Structure : 14 sections + appendix (Project Overview, Clean Arch, Directory Structure, Core Packages, Design Patterns, Data Flow, Algorithms, Integration, Config/Env, Errors, Testing, Build, Deps, ADR-001..010).
- Cohérence vs code : très bonne. ADR-009 (heuristique CPU) et ADR-010 (no-go FLINT) ajoutés. Constantes (DefaultParallelThreshold=4096, DefaultFFTThreshold=500_000, MaxPooledBitLen=50M, MaxFibUint64=93) cohérentes. Pointeurs vers fichiers Go corrects (`internal/fibonacci/calculator.go`, `fastdoubling.go`, `matrix_framework.go`).
- Écarts (cf. audit 2.1) : 4 écarts `dependency-graph.mermaid` non répercutés ici (ARCH.md décrit en texte les couches, peu sensible aux divergences d'arêtes). Pas de section "version doc" ni date — recommandé.

## docs/architecture/
| Fichier | Type | A jour ? |
|---|---|---|
| README.md (39 L) | Index | OK, succinct |
| system-context.mermaid | C4 niv.1 | OK |
| container-diagram.mermaid | C4 niv.2 | Corrigé B1 (5 fixes) |
| component-diagram.mermaid | C4 niv.3 | Corrigé B1 (18+ fixes) |
| dependency-graph.mermaid | Graphe deps | Audit 2.1 : 4 écarts résiduels |
| flows/ (6 .mermaid) | Sequences | Présents : cli, tui, config, fastdoubling, matrix, fft-pipeline |
| patterns/design-patterns.md (39 L) | Inventaire | 12 patterns listés (ARCH.md en cite 14 — divergence) |
| patterns/interface-hierarchy.mermaid | UML | OK |
| validation/validation-report.md (261 L) | Rapport | Daté 2026-02-08, 53 corrections, 0 FAIL/WARN — obsolète vs audit 2.1 (4 écarts apparus depuis) |

C4 : niv.1, 2, 3 présents ; niv.4 (code) absent (acceptable, couvert par godoc).

## docs/algorithms/
| Fichier | Sujet | Profondeur | Code refs |
|---|---|---|---|
| FAST_DOUBLING.md (305 L) | Identités, dérivation Q-matrix | Bonne, math correcte | Aucune (file:line) |
| MATRIX.md (343 L) | Exp. binaire + Strassen | Bonne | Aucune |
| FFT.md (221 L) | Théorie Schönhage-Strassen | Très bonne (audit 5.2) | Aucune |
| BIGFFT.md (701 L) | Internals package bigfft | Excellente (audit 5.2) | 22 refs `internal/bigfft/*.go` |
| GMP.md (106 L) | Backend GMP, build tag | Adéquate | 1 ref |
| COMPARISON.md (235 L) | Comparatif 4 algos + tableau | Bonne | Aucune |
| PROGRESS_BAR_ALGORITHM.md (373 L) | Modèle géométrique 4^k | Détaillée | 1 ref |

Couverture complète. Faiblesse : seul BIGFFT.md a vraiment des renvois `file:line` ; les autres décrivent l'algo sans pointer le code.

## docs/audits/2026-04/
- Structure : AUDIT_REPORT.md (228 L, 60 findings), EXECUTION_PLAN.md (694 L, statut exécuté), PRD.md (213 L), PDRTask.md (83 L), bench/ (baseline + perf-results P0-01..P0-09 before/after, P1-04-SKIPPED).
- Pertinence : référence directe pour audit actuel — 60 findings priorisés P0/P1/P2, 59/60 traités, gains FFT mesurés (-34 à -43 % temps, -83 à -94 % allocs). Sert de baseline méthodologique.

## Maintenance & process
- Pas de `MAINTAINERS.md` ni de section "qui met à jour". `CONTRIBUTING.md` à vérifier.
- README mentionne `docs/algorithms/`, `docs/architecture/`, `docs/audits/2026-04/` (lignes 60, 89, 121, 307) — liens en place.
- Validation report figé au 2026-02-08, jamais re-daté malgré le refactor 4c8f0c1.
- Pas de CI vérifiant la fraîcheur des diagrammes (recommandation §3 du validation-report jamais implémentée).

## Synthèse
- Score doc tech : **8/10** (couverture exemplaire, math solide, ADR riches ; pénalisée par validation-report obsolète, divergence patterns 12 vs 14, manque de file:line dans 6/7 algos).
- Top 3 actions :
  1. Re-dater `validation/validation-report.md` post-refactor 4c8f0c1 et corriger les 4 écarts `dependency-graph.mermaid` signalés en audit 2.1.
  2. Aligner `patterns/design-patterns.md` (12 patterns) avec ARCH.md §5 (14 patterns) — ajouter Bump Allocator, Arena, GC Controller, Dynamic Threshold, Zero-Copy, Generics manquants ou retirer doublons.
  3. Ajouter renvois `file:line` (style BIGFFT.md) dans FAST_DOUBLING.md, MATRIX.md, FFT.md, GMP.md, COMPARISON.md ; documenter process de maintenance (qui/quand) dans CONTRIBUTING.md ou nouveau `docs/MAINTAINING.md`.
