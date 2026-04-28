# 71 — Audit lint

## .golangci.yml — config
- Linters activés : 22 (disable-all + enable explicite). `tests: true`, timeout 5m, `max-issues-per-linter: 0`, `max-same-issues: 0`.
- Seuils : gocyclo 15 / gocognit 30 / funlen 100 lignes / 50 statements.
- govet : `enable-all` sauf `fieldalignment` et `shadow` (désactivés).
- Exclusions :
  - `_test.go` : gocyclo, gocognit, funlen, gosec, dupl, unparam.
  - staticcheck SA1019 (deprecation) globalement supprimé.
  - gosec G304 : `internal/calibration/profile.go`, `internal/cli/output.go` (paths CLI mono-utilisateur, audit P2-10).
  - gosec G104 (doublon errcheck), G115 (overflow casts audités P1-23/P2-09) : exclus globalement.

## Linters par catégorie
- **Style** : gofmt, revive (14 règles), whitespace, misspell (US), nolintlint.
- **Bugs** : govet (enable-all), errcheck, staticcheck, gosimple, unused, ineffassign, typecheck, bodyclose, noctx, copyloopvar.
- **Complexité** : gocyclo, gocognit, funlen.
- **Performance** : prealloc (simple+range+for), gocritic (tag performance), unparam, unconvert, nakedret.
- **Sécurité** : gosec.

## go vet
- Statut : OK (log vide, aucune sortie, exit 0 implicite).
- Findings : 0.

## golangci-lint
- Version disponible : v1.64.8 (built with go1.25.5).
- Compatibilité Go : INCOMPATIBLE. Toolchain projet = Go 1.26.2 ; le binaire golangci-lint est compilé avec go1.25 → refus de chargement de la config (`the Go language version (go1.25) used to build golangci-lint is lower than the targeted Go version (1.26.2)`).
- Statut : FAIL (exécution impossible).
- Total findings : indéterminé (lint non exécuté).
- Top 3 linters fauteurs : N/A.
- Note : `make` également absent du PATH (`/usr/bin/bash: line 1: make: command not found`) — premier essai via Makefile a échoué avant le run direct.

## Synthèse
- Score : **5/10**. Config exemplaire (24 linters, exclusions documentées par référence d'audit), `go vet` propre, mais l'outil principal ne s'exécute pas → CI dégradée tant que golangci-lint n'est pas mis à jour.
- **Top 3 actions** :
  1. Réinstaller un binaire compatible : `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` (v2.x compilé avec Go ≥ 1.26) puis re-exécuter `golangci-lint run --timeout 5m ./...` pour générer un baseline réel.
  2. Pinner la version de golangci-lint dans le Makefile et la CI (`tool` directive Go 1.24+ ou `bingo`/`tools.go`) afin d'éviter les dérives toolchain ; aligner sur le `go.mod` directive `go 1.25.0` + `toolchain go1.26.2`.
  3. Installer `make` dans l'environnement de référence (ou documenter `mingw32-make`/`scoop install make` pour Windows) et router les audits via les cibles Makefile (`make lint`, `make test`) plutôt qu'invocations directes.
- Cible "zéro warning bloquant en CI" : **non vérifiable** dans cet audit. Action 1 est prérequise.
