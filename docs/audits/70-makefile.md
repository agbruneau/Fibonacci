# 70 — Audit Makefile & CI

Date : 2026-04-28 — Source : `Makefile` (264 lignes, 9.7 KB).

## Cibles Makefile

| Cible | Description | .PHONY ? | Notes |
|---|---|---|---|
| `all` | clean + build + test (cible par défaut métier) | oui | `.DEFAULT_GOAL := help` |
| `build` | Build courant ; auto-PGO si `default.pgo` existe | oui | `-trimpath`, LDFLAGS version |
| `build-pgo` | Build avec profil PGO (dépend `pgo-check`) | oui | |
| `build-pgo-linux` / `-windows` / `-darwin` / `-all` | Cross-build PGO | oui | `darwin` couvre amd64+arm64 |
| `pgo-profile` | Génère `cpu.prof` via bench `BenchmarkFastDoubling` | oui | POSIX-only (`mv`, `[ -f ]`) |
| `pgo-check` | Vérifie présence du profil | oui | |
| `pgo-rebuild` | `pgo-profile` + `build-pgo` | oui | |
| `pgo-clean` | Supprime profil et raw | oui | |
| `build-all` | Cross-build complet | oui | linux/windows amd64+arm64, darwin amd64+arm64 |
| `build-linux` / `-linux-arm64` / `-windows` / `-windows-arm64` / `-darwin` | Cross-build par cible | oui | |
| `install` | `go install ./cmd/fibcalc` | oui | |
| `clean` | Supprime `build/`, coverage | oui | |
| `test` | `go test -v -race -cover ./...` | oui | |
| `test-short` | `go test -v -short ./...` | oui | |
| `coverage` | Coverage HTML | oui | |
| `benchmark` | Bench `internal/fibonacci` | oui | |
| `bench-versioned` | Snapshot bench horodaté avec git/Go | oui | POSIX-only (`tee`, `date`) |
| `run` / `run-fast` / `run-calibrate` | Exécutions paramétrées | oui | |
| `version` | `--version` après build | oui | |
| `lint` | `golangci-lint run ./...` | oui | |
| `security` | `gosec ./...` | oui | |
| `install-tools` | Installe golangci-lint + gosec | oui | |
| `format` | `go fmt` + `gofmt -s -w .` | oui | |
| `check` | `format` + `lint` + `test` | oui | |
| `tidy` / `deps` / `upgrade` | Gestion modules | oui | |
| `help` | Auto-listing via `awk` sur `## ` | oui | |

Total : **34 cibles**, toutes correctement déclarées dans `.PHONY` (ligne 28).

## Variables & paramètres

- `BINARY_NAME=fibcalc`, `BINARY_UNIX`, `BINARY_WIN`, `BUILD_DIR=./build`, `CMD_DIR=./cmd/fibcalc`
- `GO=go` (override possible)
- `VERSION ?=`, `COMMIT ?=`, `BUILD_DATE ?=` (overridables)
- `LDFLAGS` : `-s -w` + injection `app.Version` / `app.Commit` / `app.BuildDate`
- `GOFLAGS=$(LDFLAGS)` (écrase la sémantique standard de `GOFLAGS`)
- `-trimpath` systématique
- Cross-build via `GOOS`/`GOARCH` inline
- **Aucun build tag** (pas de cible `gmp` malgré le tag mentionné dans CLAUDE.md)

## Cohérence avec docs

| Cible CLAUDE.md | Présente | Notes |
|---|---|---|
| `all`, `test`, `test-short`, `coverage`, `benchmark`, `bench-versioned`, `lint`, `build-pgo`, `build-all` | oui (9/9) | Parfait |

| Cible BUILD.md | Présente | Notes |
|---|---|---|
| `build`, `build-pgo`, `build-all`, `pgo-profile/check/rebuild/clean`, `run*`, `tidy`, `deps`, `upgrade`, `install-tools`, `lint`, `version`, `help` | oui | |
| `generate-mocks` | **NON** | BUILD.md L194 fantôme (confirme audit 7.1) |
| `install-mockgen` | **NON** | BUILD.md L195 fantôme |

| Cible README | Présente | Notes |
|---|---|---|
| `build`, `all`, `test`, `test-short`, `lint`, `coverage`, `benchmark`, `build-pgo`, `build-all`, `help` | oui (10/10) | Cohérent |

Pas de cible `gmp` ni `build-tags`, alors que CLAUDE.md cite `build tag gmp`.

## CI / Automatisation

- **`.github/` : ABSENT** (confirmé). Aucun workflow GitHub Actions.
- **pre-commit hooks** : aucun (`.git/hooks/` ne contient que les `*.sample` par défaut). Pas de `.pre-commit-config.yaml`.
- **Recommandation CI minimale** (`.github/workflows/ci.yml`) :
  1. `lint` (golangci-lint, ubuntu-latest, Go 1.25)
  2. `test` matrix `{ubuntu, windows, macos} × Go 1.25` avec `-race -cover`
  3. `build-all` sur tag (release artifacts)
  4. `gosec` job hebdomadaire (security)
  5. Couverture vers Codecov (≥75 % cf. audit 5.0)

## Portabilité Windows

- `make` indisponible nativement sous bash MSYS limité et PowerShell → **toutes les cibles inutilisables hors WSL/Git-Bash + GNU make**.
- Cibles spécifiquement POSIX-only :
  - `pgo-profile` (`[ -f ]`, `mv`, `exit 1` shell)
  - `bench-versioned` (`tee`, `date -u`, here-doc `{ ... }`)
  - `clean` (`rm -rf`)
  - `help` (`awk` requis)
- **Alternatives** :
  1. **Mage** (`magefile.go`) — Go natif, multiplateforme, déjà dans l'écosystème Go (recommandé).
  2. **Task** (`Taskfile.yml`) — YAML, binaire unique, syntaxe lisible.
  3. **justfile** — léger, multiplateforme.
  4. **Scripts `go run ./scripts/...`** — zéro dépendance, intégration native.
- À court terme : ajouter un `make.ps1` minimal (build/test/lint) en attendant Mage.

## Synthèse

- **Score : 7/10** — Makefile mature, bien structuré, `.PHONY` complet, help auto-généré. Pénalisé par : portabilité Windows nulle, CI absente, cibles fantômes documentées (`generate-mocks`).

### Top 5 actions

1. **Créer `.github/workflows/ci.yml`** : matrix lint + test + race sur 3 OS (priorité critique).
2. **Supprimer `generate-mocks`/`install-mockgen` de `docs/BUILD.md`** ou les ajouter au Makefile (cohérence).
3. **Introduire Mage (`magefile.go`)** pour portabilité Windows native ; conserver Makefile en parallèle.
4. **Renommer `GOFLAGS` → `LDFLAGS_FLAGS`** : la variable `GOFLAGS` a une sémantique standard (env Go) qui prête à confusion.
5. **Ajouter cible `build-gmp`** (`go build -tags gmp ...`) pour exposer le backend GMP cité par CLAUDE.md.
