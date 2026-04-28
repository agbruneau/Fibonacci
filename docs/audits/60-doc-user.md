# 60 — Documentation utilisateur

## README.md

- **Sections présentes** : Quick Start, Key Features, Architecture (mermaid), Performance Benchmarks, Usage Guide, Configuration, Development, Testing, Troubleshooting, Changelog, Contributing, License. TOC en tête, badges (Go, License, Coverage 87.5%, Status). Structure claire et complète.
- **Cohérence flags CLI** (vs `internal/config/config.go` — flags effectifs : `n`, `v/verbose`, `d/details`, `timeout`, `algo`, `threshold`, `fft-threshold`, `strassen-threshold`, `calibrate`, `auto-calibrate`, `calibration-profile`, `output/o`, `quiet/q`, `machine`, `completion`, `calculate/c`, `tui`, `last-digits`, `memory-limit`, `gc-control`) :
  - **Manquant dans README** : `--gc-control` (auto/aggressive/disabled) — pourtant déclaré dans `config.go:172`. Documenté dans `docs/ARCH.md` mais absent du tableau README et de Configuration.
  - **Inexact** : `--version` / `-V` listé comme un flag (l. 176) mais c'est en réalité une *pré-flag* gérée par `app/version.go` (`HasVersionFlag`), pas par `flag.FlagSet`. Mineur.
  - Tous les autres flags concordent. Defaults (N=100M, timeout=5m, algo=all) corrects.
- **Exemples** : 6 exemples dans Usage Guide, tous reproductibles. Quick Start `git clone` correct.
- **Lisibilité** : titres clairs, tableaux à jour, mermaid à jour, deep-links vers docs/* corrects.
- **Score** : 8.5/10

## CHANGELOG.md

- Format **Keep a Changelog** + SemVer respecté.
- **Incohérence majeure** : tag git réel = `v3.0.0` ; CHANGELOG s'arrête à `[1.0.0] — 2025-12-22` puis `[Unreleased]`. Aucune entrée pour `v2.0.0` ni `v3.0.0`. Footer compare `v1.0.0...HEAD` au lieu de `v3.0.0...HEAD`. (Confirme audit 1.3.)
- Catégories propres (Added/Changed/Fixed/Security/Performance/Removed) ; section Unreleased riche et traçable aux audits.
- **Score** : 5/10 (forme bonne, versioning cassé).

## CONTRIBUTING.md

- Couvre : setup dev, commandes, branch naming, Conventional Commits, PR process, coding standards, testing, mock generation (renvoi à TESTING.md), issue reporting.
- **Pas de mention CLA** — conforme audit 4.2.
- Snippet `RegisterCalculator` utilise `func() coreCalculator` minuscule (l. 183) — la signature publique est `fibonacci.CoreCalculator`. Erreur de copy/paste.
- Tableau "Useful Commands" omet `make build-pgo`, `make build-all`, `make security`.
- File Organization (l. 224) montre encore une structure `internal/fibonacci/` pré-refactor (pas de `memory/`, `threshold/`).
- **Score** : 7/10.

## docs/*.md

| Fichier | Statut | Écarts vs code |
|---|---|---|
| BUILD.md | OK | Tableau cross-compilation (l. 119-123) omet `build-linux-arm64` et `build-windows-arm64` (présents dans Makefile). Tableau Code Generation mentionne `generate-mocks`/`install-mockgen` (cibles absentes du Makefile actuel — incohérence avec CONTRIBUTING). `.golangci.yml` annoncé "22 linters" alors que CLAUDE.md indique 24. |
| PERFORMANCE.md | OK | Chiffres référencés Ryzen 5900X + annexe Intel Ultra 9. Méthodologie `bench-versioned` documentée. Utilise `make bench-versioned`. Cohérent avec CHANGELOG. |
| CALIBRATION.md | OK | Couvre les 3 modes (`--calibrate`, `--auto-calibrate`, `--calibration-profile`), seuils (Parallel/FFT/Strassen) avec defaults cohérents (4096/500k/3072). |
| TESTING.md | OK | Couvre golden tests (`fibonacci_golden.json`), race detector, fuzz (5 cibles). Quick reference inclut `go generate` mais aucun `//go:generate` n'existe (cohérent avec section Mock Generation qui le précise). |
| TUI_GUIDE.md | OK | Structure cohérente avec `internal/tui/` (model, header, logs, metrics, chart, footer, bridge, keymap, sparkline présents). Exemples flags `--tui` corrects. |

## Cohérence globale

- **Liens cassés** : README l. 307 pointe `docs/audits/2026-04/` — répertoire existe mais ne contient que `AUDIT_REPORT.md`, `EXECUTION_PLAN.md`, `PDRTask.md`, `PRD.md` (pas de notes par numéro ; les `60-*.md` etc. vivent dans `docs/audits/`). Lien valide mais sémantique trompeuse.
- **Redondances** : tableau env vars dupliqué entre README §Configuration, BUILD.md §Environment Variables et `.env.example`. Risque de drift (déjà observé : `FIBCALC_GC_CONTROL` absent de `.env.example` et README, mais flag `--gc-control` actif).
- **Build tag GMP** : documenté dans BUILD.md mais absent du tableau "Build Tags" du README (mention furtive l. 58 sans procédure).
- **Coverage badge** : 87.5% (CHANGELOG) cohérent avec README. À revérifier après refactor récent.

## Synthèse

- **Score doc utilisateur global : 7/10** (lisible, complet en surface, mais désynchronisations clés sur version et flags).

### Top 5 corrections

1. **CHANGELOG** : ajouter entrées `[2.0.0]`, `[3.0.0]` (ou consolider Unreleased→3.0.0) et corriger les liens compare `v3.0.0...HEAD`.
2. **README + .env.example** : documenter `--gc-control` / `FIBCALC_GC_CONTROL` (ou retirer le flag du code si non supporté).
3. **CONTRIBUTING.md** : corriger snippet `RegisterCalculator` (`CoreCalculator` majuscule), ajouter `make build-pgo`/`build-all`, mettre à jour File Organization avec sous-packages `memory/` et `threshold/`.
4. **BUILD.md** : ajouter cibles `build-linux-arm64` / `build-windows-arm64`, retirer ou implémenter `generate-mocks`/`install-mockgen`, aligner compteur de linters (22 vs 24).
5. **README §Build Tags** : ajouter section explicite `gmp` (procédure `go build -tags=gmp`) avec renvoi BUILD.md, plutôt que mention isolée.
