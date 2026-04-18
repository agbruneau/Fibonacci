# TEAM F — Build, CI/CD & Outillage

> Audit read-only effectue depuis `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo` — Go 1.26.2 (env), `go.mod` declare `go 1.24.3`, CLAUDE.md cible Go 1.25.0+.
> Scope : `Makefile`, `.github/workflows/**`, `.golangci.yml`, `.env.example`, `go.mod`/`go.sum`, `.gitignore`, scripts.

## Resume executif

- Cibles Makefile recensees : **34** (PHONY declare 28 — 6 manquantes au .PHONY)
- Cibles non portables Windows : **31/34** (utilisent `mkdir -p`, `rm -rf`, `if [ -f ... ]`, `tee`, `date -u +%FT%TZ`, `sed`, `column` — exigent un shell POSIX)
- Workflows GitHub Actions : **1** fichier (`ci.yml`) couvrant uniquement `internal/bigfft/...`
- Linters golangci-lint actives : **22** (cf. CLAUDE.md). `gosec` actif (21 findings G115). `dupl` reference dans `exclude-rules` mais pas active.
- Dependances directes : **9 + 1 (`x/sync`)**. **6 obsoletes** (1 majeure `bubbles v0.21->v1.0`, 5 patch/minor)
- Dependances indirectes : **27 dont 8 obsoletes** (patch/minor uniquement)
- Total modules listes : **200** (immense graphe transitif via `gopsutil` / `bubbletea`)
- `default.pgo` **absent** dans `cmd/fibcalc/` : la cible `build` choisit toujours la branche non-PGO. Cibles `pgo-*` definies mais profil non present dans le repo.
- `-trimpath` et `-buildvcs=false` **absents** du `LDFLAGS` Makefile. Build non reproductible.
- Cross-compilation `build-all` : couvre `linux/amd64`, `windows/amd64`, `darwin/amd64`, `darwin/arm64` — **manque `linux/arm64`** et `windows/arm64`.
- Artefacts orphelins detectes a la racine : `build_err.txt`, `e2e_rich_out.txt`, `test_err.txt`, `test_out.txt` (couverts par `*.txt` ? non — `.gitignore` ignore seulement `test_results.txt`).
- Findings P0 : 3 — P1 : 7 — P2 : 6.

---

## Inventaire Makefile

| # | Cible | Description | Portabilite Win (`cmd.exe`) | Notes |
|---|-------|-------------|-----------------------------|-------|
| 1 | `all` | clean + build + test | Non | Depend de cibles non portables |
| 2 | `build` | Build avec PGO si profil present | Non | `mkdir -p`, `if [ -f ... ]` |
| 3 | `pgo-profile` | Genere CPU profile pour PGO | Non | `mv`, `if [ -f ... ]` |
| 4 | `pgo-check` | Verifie PGO profile | Non | `if [ ! -f ... ]` |
| 5 | `build-pgo` | Build PGO (depend pgo-check) | Non | `mkdir -p` |
| 6 | `build-pgo-linux` | Build PGO Linux amd64 | Non | env inline `GOOS=...` (KO sur cmd.exe) |
| 7 | `build-pgo-windows` | Build PGO Windows amd64 | Non | idem |
| 8 | `build-pgo-darwin` | Build PGO macOS amd64+arm64 | Non | idem |
| 9 | `build-pgo-all` | Aggregat | N/A | depend des 3 ci-dessus |
| 10 | `pgo-rebuild` | Profil + build PGO | Non | depend de pgo-profile |
| 11 | `pgo-clean` | Supprime PGO artifacts | Non | `rm -f` |
| 12 | `version` | Affiche version (apres build) | Non | depend de build |
| 13 | `build-all` | Build all platforms (no PGO) | Non | depend de 3 cross |
| 14 | `build-linux` | Linux amd64 | Non | env inline `GOOS=...` |
| 15 | `build-windows` | Windows amd64 | Non | idem |
| 16 | `build-darwin` | macOS amd64+arm64 | Non | idem |
| 17 | `install` | go install | Oui | natif Go |
| 18 | `clean` | Supprime build/, coverage.* | Non | `rm -rf` |
| 19 | `test` | Tests + race + cover | Oui | natif Go |
| 20 | `test-short` | Tests rapides | Oui | natif Go |
| 21 | `coverage` | HTML coverage | Oui | natif Go |
| 22 | `benchmark` | bench `internal/fibonacci` | Oui | natif Go |
| 23 | `bench-versioned` | Snapshot benchmark horodate | Non | `tee`, `mkdir -p`, `date`, here-doc shell |
| 24 | `run` | Build + run | Non | depend build |
| 25 | `run-fast` | Run avec n=1000 | Non | idem |
| 26 | `run-calibrate` | Mode calibration | Non | idem |
| 27 | `lint` | golangci-lint run | Oui | si binaire dispo |
| 28 | `security` | gosec run | Oui | si binaire dispo |
| 29 | `install-tools` | go install golangci-lint + gosec | Oui | natif Go |
| 30 | `generate-mocks` | go generate ./... | Oui | natif Go |
| 31 | `install-mockgen` | go install mockgen | Oui | natif Go |
| 32 | `format` | gofmt + go fmt | Oui | natif Go |
| 33 | `check` | format + lint + test | Oui | composition |
| 34 | `tidy` / `deps` / `upgrade` / `help` | divers | Oui (sauf `help`) | `help` utilise `sed | column` (KO sur cmd.exe) |

### Verifications PHONY

`.PHONY` declare 28 cibles. **Manque** : `version`, `build-all`, `build-linux`, `build-windows`, `build-darwin`, `security`, `install-tools`, `tidy`, `deps`, `upgrade`. Si un fichier portant ces noms apparaissait, make refuserait de re-executer la cible.

### Coherence LDFLAGS

`LDFLAGS` injecte 3 valeurs (Version, Commit, BuildDate) dans `internal/app` — **OK**. Symboles strippes (`-s -w`) — adapte pour binaire prod. **Manque** `-trimpath` et option `-buildvcs=false`. Voir Findings F-F4 / F-F8.

---

## Inventaire workflows

| Fichier | Triggers | OS matrix | Go version | Cache | Race | Notes |
|---------|----------|-----------|------------|-------|------|-------|
| `.github/workflows/ci.yml` | push (main/master) + PR | ubuntu, windows, macOS (latest) | `go-version-file: go.mod` | oui (`cache: true`) | **non** | Couvre seulement `internal/bigfft/...`. Pas de lint, pas de coverage, pas de full test, pas de PGO. |

Actions utilisees :
- `actions/checkout@v4` — actuel (v5 publiee, mais v4 reste supportee).
- `actions/setup-go@v5` — actuel.
- Pas d'utilisation de `actions/cache` (cache delegue a `setup-go`, OK).

---

## Findings

### F-F1 : `.PHONY` incomplete — risque de collisions silencieuses

- **Severite** : P2
- **Fichier:ligne** : `Makefile:28`
- **Diagnostic** : 10 cibles ne sont pas declarees `.PHONY` (`version`, `build-all`, `build-linux`, `build-windows`, `build-darwin`, `security`, `install-tools`, `tidy`, `deps`, `upgrade`). Si un fichier homonyme existe (ex: `version` cree par un outil), make considere la cible "up-to-date" et ne l'execute pas.
- **Patch propose** :
```diff
-.PHONY: all build clean test coverage benchmark bench-versioned run help install lint format check pgo-profile pgo-check pgo-clean pgo-rebuild build-pgo-linux build-pgo-windows build-pgo-darwin build-pgo-all generate-mocks install-mockgen
+.PHONY: all build clean test test-short coverage benchmark bench-versioned run run-fast run-calibrate \
+        help install lint format check pgo-profile pgo-check pgo-clean pgo-rebuild \
+        build-pgo build-pgo-linux build-pgo-windows build-pgo-darwin build-pgo-all \
+        version build-all build-linux build-windows build-darwin \
+        security install-tools tidy deps upgrade generate-mocks install-mockgen
```
- **Effort** : S

---

### F-F2 : Couverture CI quasi nulle — risque de regression non detectee

- **Severite** : **P0**
- **Fichier:ligne** : `.github/workflows/ci.yml:27-28`
- **Diagnostic** : Le seul job CI est `bigfft-matrix` qui execute `go test -count=1 -timeout=10m ./internal/bigfft/...`. Les 15 autres packages (`fibonacci`, `orchestration`, `cli`, `tui`, etc.) ne sont jamais testes en CI. **Aucun job lint** (alors que `make lint` existe et 200+ findings sont presents dans `bench/baseline/lint.txt`). **Race detector absent** (CLAUDE.md exige race detector activee par defaut en CI). Pas de coverage upload. Cela contredit explicitement CLAUDE.md ligne "Race detector active par defaut dans CI".
- **Patch propose** :
```diff
 jobs:
   bigfft-matrix:
     name: bigfft (${{ matrix.os }})
     runs-on: ${{ matrix.os }}
     strategy:
       fail-fast: false
       matrix:
         os: [ubuntu-latest, windows-latest, macos-latest]
     steps:
       - uses: actions/checkout@v4
       - uses: actions/setup-go@v5
         with:
           go-version-file: go.mod
           cache: true
       - name: Go test internal/bigfft
-        run: go test -count=1 -timeout=10m ./internal/bigfft/...
+        run: go test -race -count=1 -timeout=15m ./internal/bigfft/...
+
+  test-full:
+    name: full tests (${{ matrix.os }})
+    runs-on: ${{ matrix.os }}
+    strategy:
+      fail-fast: false
+      matrix:
+        os: [ubuntu-latest, windows-latest, macos-latest]
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v5
+        with: { go-version-file: go.mod, cache: true }
+      - run: go test -race -short -timeout=20m ./...
+
+  lint:
+    runs-on: ubuntu-latest
+    steps:
+      - uses: actions/checkout@v4
+      - uses: actions/setup-go@v5
+        with: { go-version-file: go.mod, cache: true }
+      - uses: golangci/golangci-lint-action@v6
+        with: { version: latest }
```
- **Effort** : M

---

### F-F3 : Race detector absent du seul job CI

- **Severite** : P1
- **Fichier:ligne** : `.github/workflows/ci.yml:28`
- **Diagnostic** : `go test -count=1 -timeout=10m ./internal/bigfft/...` n'inclut pas `-race`. CLAUDE.md indique explicitement "Race detector active par defaut dans CI". `bigfft` manipule lourdement des goroutines (FFT parallele), et `bench/baseline/lint.txt:4` montre des appels concurrents non checkes (`fft_parallel_test.go`).
- **Patch propose** : Ajouter `-race` (cf. patch F-F2).
- **Effort** : S

---

### F-F4 : `-trimpath` absent — builds non reproductibles

- **Severite** : P1
- **Fichier:ligne** : `Makefile:22-26`
- **Diagnostic** : Sans `-trimpath`, les binaires distribues contiennent les chemins absolus de la machine de build (`C:\Users\agbru\OneDrive\Documents\GitHub\FibGo\...`). Cela compromet la reproductibilite, fuite des donnees user, et empeche les builds bit-a-bit identiques entre dev et CI. CLAUDE.md mentionne PGO et performance mais pas explicitement reproductibilite — neanmoins une attente pour un projet "academique demontrant patterns d'ingenierie avancee".
- **Patch propose** :
```diff
-LDFLAGS=-ldflags="-s -w \
+TRIMPATH=-trimpath
+LDFLAGS=-ldflags="-s -w \
   -X github.com/agbru/fibcalc/internal/app.Version=$(VERSION) \
   -X github.com/agbru/fibcalc/internal/app.Commit=$(COMMIT) \
   -X github.com/agbru/fibcalc/internal/app.BuildDate=$(BUILD_DATE)"
-GOFLAGS=$(LDFLAGS)
+GOFLAGS=$(TRIMPATH) $(LDFLAGS)
```
- **Effort** : S

---

### F-F5 : `default.pgo` absent — la branche PGO du `build` est morte

- **Severite** : P1
- **Fichier:ligne** : `cmd/fibcalc/` (repertoire inspecte : seul `main.go` + `main_test.go`)
- **Diagnostic** : `Makefile:37` teste `if [ -f $(PGO_PROFILE) ]` ou `PGO_PROFILE=cmd/fibcalc/default.pgo`. Le fichier n'existe pas dans le repo. `glob **/*.pgo` retourne 0 fichiers. Toute mention de PGO dans CLAUDE.md (`make build-pgo`, "PGO supporte") est donc theorique — aucun developpeur ne beneficie de PGO sans avoir d'abord lance `make pgo-profile` (qui exige 5+ minutes de bench). **De plus, `.gitignore:28` ignore `*.pgo`** — donc meme si on commit `default.pgo`, il sera filtre. Go [recommande](https://go.dev/doc/pgo) de commiter `default.pgo` a cote de `main.go` pour build PGO automatique.
- **Patch propose** :
```diff
 # Go profile-guided optimization files
-*.pgo
+*.pgo
+!cmd/*/default.pgo
```
+ commit d'un `default.pgo` representatif (hors scope read-only).
- **Effort** : M

---

### F-F6 : `make help` non portable Windows

- **Severite** : P2
- **Fichier:ligne** : `Makefile:251-253`
- **Diagnostic** : `sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'`. `column` n'existe pas sous Windows/PowerShell. `.DEFAULT_GOAL := help` (ligne 255) signifie que faire `make` sans argument tombe sur cette commande KO sur Windows. Or l'utilisateur travaille sur Windows 11 (env).
- **Patch propose** : utiliser awk (souvent present dans Git Bash sur Windows) :
```diff
-help:
-	@echo "Available targets:"
-	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
+help:
+	@echo "Available targets:"
+	@awk 'BEGIN{FS=":"} /^## / {sub(/^## /,"",$$0); printf "  %-22s %s\n", $$1, substr($$0, index($$0,":")+2)}' $(MAKEFILE_LIST)
```
- **Effort** : S

---

### F-F7 : Cross-compilation incomplete — `linux/arm64` et `windows/arm64` absents

- **Severite** : P2
- **Fichier:ligne** : `Makefile:111`
- **Diagnostic** : `build-all: build-linux build-windows build-darwin` ne produit que `linux/amd64`, `windows/amd64`, `darwin/amd64+arm64`. Pas de `linux/arm64` (Graviton, Raspberry Pi 64-bit) ni `windows/arm64` (Surface Pro X, Snapdragon X). CLAUDE.md mentionne "Cross-compilation (linux, windows, macOS)" sans preciser arm64 — mais c'est en pratique attendu en 2026.
- **Patch propose** :
```diff
 build-linux:
 	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) $(CMD_DIR)
+	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux_arm64 $(CMD_DIR)

 build-windows:
 	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_WIN) $(CMD_DIR)
+	GOOS=windows GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_windows_arm64.exe $(CMD_DIR)
```
- **Effort** : S

---

### F-F8 : `go.mod` declare Go 1.24.3 alors que CLAUDE.md exige 1.25.0+

- **Severite** : **P0**
- **Fichier:ligne** : `go.mod:3`
- **Diagnostic** : `go 1.24.3` dans `go.mod`. CLAUDE.md ligne "Go : 1.25.0+". Environnement CI utilise `go-version-file: go.mod` donc CI tournera sur 1.24.3 alors que le contrat dit 1.25+. Pas de directive `toolchain`. Aucune coherence entre la directive de projet et la realite. Risque : developpement local sur Go 1.26.2 (env actuel) qui pourrait introduire des features Go 1.26 silencieusement. Ajouter `toolchain go1.25.0` permettrait a Go de telecharger automatiquement le toolchain bon dans CI.
- **Patch propose** :
```diff
 module github.com/agbru/fibcalc

-go 1.24.3
+go 1.25.0
+
+toolchain go1.25.0
```
- **Effort** : S (mais necessite de revalider tous les tests sur 1.25)

---

### F-F9 : Fichiers d'erreurs/sortie de tests laisses a la racine

- **Severite** : P1
- **Fichier:ligne** : racine (`build_err.txt:0o`, `e2e_rich_out.txt:94o`, `test_err.txt:14o`, `test_out.txt:21o`)
- **Diagnostic** : Quatre fichiers `.txt` orphelins a la racine. `.gitignore` couvre `*.out`, `*.prof`, `coverage.*`, `test_results.txt` — mais **pas le pattern generique `*_err.txt` / `*_out.txt`**. Ces fichiers vont etre committes accidentellement (ils font deja partie du listing `git status` avant cleanup). Ils ne devraient pas exister a la racine — ils sont visiblement des stdout/stderr de runs locaux.
- **Patch propose** :
```diff
 # env file
 .env
+
+# Local test/build run artifacts at repo root
+/build_err.txt
+/test_err.txt
+/test_out.txt
+/e2e_rich_out.txt
+*_err.txt
+*_out.txt
```
+ supprimer manuellement les 4 fichiers (hors scope read-only).
- **Effort** : S

---

### F-F10 : Cibles `bench-versioned` et `pgo-profile` non portables Windows (`tee`, `date -u`)

- **Severite** : P2
- **Fichier:ligne** : `Makefile:172-182`
- **Diagnostic** : `tee` n'est pas dispo sous `cmd.exe` natif. `date -u +%Y-%m-%dT%H:%M:%SZ` echoue sous PowerShell. Sous Git Bash le bloc fonctionne. Pour un projet multi-plateforme, ces cibles devraient soit etre documentees comme "POSIX shell only" soit refactorees.
- **Patch propose** : ajouter dans le `help` ou `README` une mention "make targets requiring a POSIX shell (Git Bash on Windows): `bench-versioned`, `pgo-*`, `clean`, `bench`...". Alternative plus lourde : reecrire en script Go (`scripts/bench.go`).
- **Effort** : M

---

### F-F11 : `gosec` declare actif mais 21 findings G115 ignores

- **Severite** : P1
- **Fichier:ligne** : `.golangci.yml:42`, `bench/baseline/lint.txt` (21 occurrences `G115`)
- **Diagnostic** : `gosec` est dans la liste `linters.enable`. `linters-settings.gosec.excludes: [G104]` (qui est duplique d'errcheck). **G115 (integer overflow conversion)** n'est pas dans `excludes` — donc 21 findings sont remontes par lint mais ignores en pratique (pas de CI lint). `internal/bigfft/` est massivement concerne (10 findings) car les conversions `int <-> uint <-> uint64` sont omnipresentes en arithmetique FFT. Trois options : (a) corriger les 21, (b) `excludes: [G104, G115]` documente, (c) `//nolint:gosec` cible avec justification.
- **Patch propose** :
```diff
   gosec:
     excludes:
       - G104 # Duplicate of errcheck (ineffectual assignment)
+      - G115 # Many intentional conversions in FFT/bigint arithmetic; reviewed by Team B
```
+ alternative : ajouter `//nolint:gosec // FFT word arithmetic` ligne par ligne (preferable mais effort L).
- **Effort** : S (option b) / L (option c)

---

### F-F12 : Sortie `go.mod` declare un `require` separement pour `x/sync` (cosmetique)

- **Severite** : P2
- **Fichier:ligne** : `go.mod:5`
- **Diagnostic** : `golang.org/x/sync` est dans un bloc `require` solitaire (ligne 5), separe du bloc principal (lignes 7-17). C'est valide mais inhabituel ; `go mod tidy` rangerait normalement tout dans un bloc unique. Indique probablement un `go get` manuel non suivi de `tidy`.
- **Patch propose** : `go mod tidy` regroupera automatiquement.
- **Effort** : S

---

### F-F13 : Dependances obsoletes — 6 directes, 8 indirectes

- **Severite** : **P0**
- **Fichier:ligne** : `go.mod:5-48`
- **Diagnostic** : Voir annexe en bas. **1 mise a jour majeure** : `charmbracelet/bubbles v0.21.1 -> v1.0.0` (rupture API potentielle, le projet utilise `bubbles` pour TUI dans `internal/tui/`). 5 mises a jour mineures/patch directes. 8 mises a jour indirectes. `golang.org/x/term v0.36.0 -> v0.42.0` (6 versions de retard) et `golang.org/x/text v0.8.0 -> v0.36.0` (28 versions de retard !) sont particulierement preoccupantes pour les modules systemes/security.
- **Patch propose** : voir annexe. Workflow : `go get -u <dep>@<version>` + `go mod tidy` + tests.
- **Effort** : M (patch/minor) / L (`bubbles v1` rupture)

---

### F-F14 : `actions/setup-go@v5` — pas de `check-latest`

- **Severite** : P2
- **Fichier:ligne** : `.github/workflows/ci.yml:23-25`
- **Diagnostic** : `go-version-file: go.mod` est OK. Mais sans `check-latest: true`, l'action utilise le cache GitHub qui peut etre en retard de plusieurs jours/semaines pour les patch versions (1.24.3 -> 1.24.5). Pour un projet axe performance (PGO, FFT), bien recevoir les patch perf de Go est important.
- **Patch propose** :
```diff
       - uses: actions/setup-go@v5
         with:
           go-version-file: go.mod
           cache: true
+          check-latest: true
```
- **Effort** : S

---

### F-F15 : `linters-settings.govet.enable-all: true` mais `disable: [fieldalignment, shadow]`

- **Severite** : P2
- **Fichier:ligne** : `.golangci.yml:48-52`
- **Diagnostic** : Choix raisonne et documente. Cependant `bench/baseline/lint.txt` ne contient **aucun** finding `(govet)` — alors qu'avec `enable-all` on s'attendrait au moins a quelques `printf`, `composite`, etc. Probablement parce que le code est deja propre. **Cosmetique / point positif** : laisser tel quel. Pas de patch.
- **Effort** : N/A (informatif)

---

### F-F16 : Coherence avec CLAUDE.md — seuils respectes mais 5 violations live

- **Severite** : P2
- **Fichier:ligne** : `.golangci.yml:54-62` vs `bench/baseline/lint.txt`
- **Diagnostic** : CLAUDE.md exige `cyclomatic ≤ 15`, `cognitive ≤ 30`, `funlen ≤ 100 lignes / 50 statements`. `.golangci.yml` aligne **exactement** ces seuils (gocyclo:15, gocognit:30, funlen:100/50). **Coherence parfaite**. Cependant lint baseline montre 4 violations `gocyclo` (matrix_types.go:154 = 24, model.go:136 = 20, sparkline.go:126 = 17, calculate.go:39 = 16) et 1 `gocognit` (completion.go:96 = 39). Ces dettes existent et ne devraient pas s'aggraver.
- **Patch propose** : aucun (config OK). Recommandation : enforcer en CI (cf. F-F2).
- **Effort** : N/A

---

## Annexe A : dependances `go list -u -m all` (extrait — directes uniquement)

### Mises a jour MAJEURES (rupture API potentielle)

| Module | Actuelle | Disponible | Risque rupture |
|--------|----------|------------|----------------|
| `github.com/charmbracelet/bubbles` | v0.21.1 | **v1.0.0** | **Eleve** — release v1.0 inclut souvent breaking changes (signature de `tea.Model.Update`, types de spinner/progress). Code a auditer dans `internal/tui/`. |

### Mises a jour MINEURES (compatibles, recommandees)

| Module | Actuelle | Disponible | Risque |
|--------|----------|------------|--------|
| `golang.org/x/term` | v0.36.0 | v0.42.0 | Faible (extensions API surface) |
| `golang.org/x/text` | v0.8.0 | **v0.36.0** | Faible mais retard *enorme* — gros gap, valider |
| `golang.org/x/sync` | v0.17.0 | v0.20.0 | Faible (errgroup stable) |
| `golang.org/x/sys` | v0.40.0 | v0.43.0 | Faible (syscall wrappers) |

### Mises a jour PATCH (sans risque)

| Module | Actuelle | Disponible |
|--------|----------|------------|
| `github.com/rs/zerolog` | v1.34.0 | v1.35.0 |
| `github.com/shirou/gopsutil/v4` | v4.26.1 | v4.26.3 |

### Indirectes obsoletes

| Module | Actuelle | Disponible | Type |
|--------|----------|------------|------|
| `github.com/charmbracelet/colorprofile` | v0.4.1 | v0.4.3 | patch |
| `github.com/charmbracelet/x/ansi` | v0.11.5 | v0.11.7 | patch |
| `github.com/clipperhouse/displaywidth` | v0.9.0 | v0.11.0 | minor |
| `github.com/clipperhouse/uax29/v2` | v2.5.0 | v2.7.0 | minor |
| `github.com/ebitengine/purego` | v0.9.1 | v0.10.0 | minor |
| `github.com/fatih/color` | v1.18.0 | v1.19.0 | minor |
| `github.com/go-ole/go-ole` | v1.2.6 | v1.3.0 | minor |
| `github.com/lucasb-eyer/go-colorful` | v1.3.0 | v1.4.0 | minor |
| `github.com/lufia/plan9stats` | (2021) | (2026) | pseudo |
| `github.com/mattn/go-isatty` | v0.0.20 | v0.0.21 | patch |
| `github.com/mattn/go-runewidth` | v0.0.19 | v0.0.23 | patch |

### Indirectes a jour

`aymanbagabas/go-osc52`, `briandowns/spinner`, `bubbletea`, `lipgloss`, `x/cellbuf`, `x/term`, `clipperhouse/stringish`, `erikgeiser/coninput`, `mattn/go-colorable`, `mattn/go-localereader`, `muesli/ansi`, `muesli/cancelreader`, `muesli/termenv`, `power-devops/perfstat`, `rivo/uniseg`, `tklauser/go-sysconf`, `tklauser/numcpus`, `xo/terminfo`, `yusufpapurcu/wmi`, `leanovate/gopter`, `ncw/gmp`.

### Total transitive : 200 modules (graph compact pour un projet de cette taille)

> Note : `go list -u -m all` renvoie aussi des modules de tres longue chaine (`cloud.google.com/go`, `hashicorp/*`, `etcd`, `prometheus`, `kubernetes`...) qui sont des **transitives indirectes via gopsutil ou bubbletea**. Aucune n'est requise dans `go.mod` direct, donc pas d'action necessaire — `go mod tidy` les filtre du graphe build.

---

## Annexe B : findings linter par categorie (synthese baseline)

| Linter | Findings | Severite | Action recommandee |
|--------|----------|----------|--------------------|
| gocritic | 66 | P2 | Triage, beaucoup de style |
| revive | 42 | P2 | Triage, exported docs |
| errcheck | 29 | P1 | Verifier ignorables (recover, rand.Read tests) |
| gosec | 21 | P1 | G115 — voir F-F11 |
| misspell | 17 | P2 | Quick fix |
| staticcheck | 11 | P1 | SA6002, SA9003 a regarder |
| unparam | 5 | P2 | Cleanup parametres morts |
| prealloc | 4 | P2 | Optims slice |
| gocyclo | 3 | P1 | Refactor matrix_types/model.go |
| gocognit | 1 | P1 | completion.go:96 |
| funlen | 1 | P1 | Une fonction > 100 LOC |
| whitespace | 1 | P2 | Quick fix |
| **Total** | **201** | | |

---

## Conclusion

Le projet FibGo dispose d'un Makefile riche (34 cibles) et d'une configuration golangci-lint coherente avec CLAUDE.md (seuils alignes, 22 linters actives). Les **trois angles morts critiques** sont :

1. **Couverture CI catastrophique** (F-F2) : 1 seul package teste sur 16, sans race detector ni lint. Toute la promesse "race detector active par defaut dans CI" de CLAUDE.md est non tenue.
2. **PGO inutile en pratique** (F-F5) : la branche PGO du `Makefile:build` est morte car `default.pgo` n'est ni present ni autorise par `.gitignore`.
3. **Dette de versions** (F-F8 + F-F13) : `go.mod` declare 1.24.3 vs 1.25+ exige par CLAUDE.md, et 14 dependances (dont une majeure `bubbles v1.0`) sont obsoletes.

Les findings P1 (race CI, trimpath, gosec G115, residus root) sont rapidement corrigeables. Les P2 sont surtout cosmetiques/portabilite. Aucun findings ne necessite une refonte architecturale.
