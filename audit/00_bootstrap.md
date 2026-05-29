# 00 — Bootstrap de l'audit (journal de l'orchestrateur)

> **Conclusion-clé** — L'environnement permet un audit **statique complet + dynamique partiel**. Le code **compile proprement** (`go build ./...`, exit 0) et `go vet`, `gosec`, `govulncheck`, les benchmarks, les fuzz et les golden tests sont tous exécutables. **Deux limites bloquantes** pour l'outillage : (1) le **race detector est indisponible** (`CGO_ENABLED=0`, aucun `gcc`/`clang` sous Windows) — la concurrence sera auditée **statiquement** ; (2) `staticcheck` et `golangci-lint` préinstallés **refusent de tourner** car compilés avec go1.25.x alors que le module cible **go 1.26.0** — contournés par recompilation locale avec go1.26.3 (voir §5).

- **Dépôt** : `C:\Users\agbru\OneDrive\Documents\GitHub\FibGo` (déjà cloné — pas de `git clone`).
- **Module** : `github.com/agbru/fibcalc`
- **Commit audité (HEAD)** : `866b8cdcdde5256bd78db260ed5434e1837d86ec` — daté **2026-05-24 19:22:47 -0400** — *« docs(dashboard): refresh knowledge graph at f322906 »*
- **Working tree** : **propre** (`git status --short` vide) [confirmé]
- **Date de l'audit** : 2026-05-28
- **Hôte** : Windows 11 Pro (26220), `windows/amd64`, shell PowerShell 7 (+ Bash via Git Bash).

---

## 1) Cible Go & écarts de version [confirmé]

| Source | Version déclarée | Observation |
|---|---|---|
| `go.mod` directive `go` | **`go 1.26.0`** | Cible réelle du module |
| `go.mod` `toolchain` | *(absente)* | `GOTOOLCHAIN=auto` |
| Toolchain installée | **`go1.26.3 windows/amd64`** | Dernier correctif (1.26.3, publié ~7 mai 2026) |
| `README` / `CLAUDE.md` | « Go 1.25.0+ (toolchain **1.26.2**) » | **Dérive doc** : la doc annonce 1.25+ / toolchain 1.26.2, la réalité est `go 1.26.0` + toolchain **1.26.3** |

→ **Constat de bootstrap** : la documentation projet (`CLAUDE.md`, README) sous-déclare la version Go minimale (1.25 vs 1.26.0 réel) et fige une toolchain (1.26.2) désormais dépassée (1.26.3). À reclasser par le sous-agent 5 (structure/doc). [confirmé]

---

## 2) Détecteur de races [confirmé — bloquant]

| Vérification | Résultat |
|---|---|
| `go env CGO_ENABLED` | **`0`** |
| `gcc --version` | **introuvable** |
| `clang --version` | **introuvable** |

→ **`go test -race` est INDISPONIBLE sur cet hôte** (CGO + compilateur C requis). Conforme à `CLAUDE.md` (« sous Windows la validation `-race` se fait via WSL ou un autre poste Linux/macOS »). Le sous-agent 2 (concurrence) procédera par **analyse statique** (`go vet`, lecture de code, inspection des `atomic`/`sync.Pool`/sémaphores) et marquera les conclusions sensibles aux races **[à vérifier]** (à confirmer sous `-race` Linux). [confirmé]

---

## 3) Inventaire du dépôt [confirmé]

| Métrique | Valeur |
|---|---|
| Fichiers `.go` (total) | **248** |
| Fichiers `.go` non-test | **135** |
| Fichiers `*_test.go` | **113** |
| Packages (`go list ./...`) | **25** |
| Cibles `Benchmark*` | **52** |
| Cibles `Fuzz*` | **7** (5 dans `internal/fibonacci/fibonacci_fuzz_test.go` + **2 dans `internal/bigfft/fft_fuzz_test.go`** : `FuzzMul`, `FuzzSqr`) |
| Test de propriété (gopter) | `internal/fibonacci/fibonacci_property_test.go` |
| Golden file | `internal/fibonacci/testdata/fibonacci_golden.json` (80 595 octets) + corpus `fuzz/` |
| Tests e2e | `test/e2e/cli_e2e_test.go`, `extended_e2e_test.go` |
| `go mod verify` | **all modules verified** |
| `.github/workflows/` | **ABSENT** (aucune CI distante) [confirmé] |

**Cibles fuzz détectées** : `FuzzFastDoublingConsistency`, `FuzzFFTBasedConsistency`, `FuzzFibonacciIdentities`, `FuzzFastDoublingMod`, `FuzzProgressMonotonicity` (fibonacci) ; `FuzzMul`, `FuzzSqr` (bigfft).
*Nuance vs énoncé* : l'énoncé annonçait « 5 cibles fuzz » — exact pour `fibonacci`, mais **7 au total** avec `bigfft`.

**Packages** : `cmd/fibcalc`, `cmd/generate-golden`, `internal` (+ `app`, `bigfft`, `calibration`, `cli`, `cli/completion`, `config`, `errors`, `fibonacci`, `fibonacci/fibonaccitest`, `fibonacci/memory`, `fibonacci/threshold`, `format`, `metrics`, `metrics/system`, `orchestration`, `parallel`, `progress`, `testutil`, `tui`, `tui/component`, `ui`), `test/e2e`.

---

## 4) Build & dépendances [confirmé]

- **`go build ./...` → exit 0** (aucune erreur). Le tree compile intégralement. [confirmé]
- **`go mod download` → OK** ; **`go mod verify` → all modules verified**.
- Dépendances directes notables : `golang.org/x/sync v0.20.0` (errgroup), `github.com/leanovate/gopter v0.2.11` (property testing), `github.com/ncw/gmp v1.0.5` (backend GMP, build tag), `charmbracelet/bubbletea v1.3.10` (TUI), `rs/zerolog v1.35.0`, `shirou/gopsutil/v4 v4.26.3`.

---

## 5) Matrice outillage → disponibilité → plan de repli [confirmé]

| Outil | Version installée | Disponible ? | Plan de repli |
|---|---|---|---|
| `go build` / `go test` | go1.26.3 | **OUI** | — |
| `go vet ./...` | go1.26.3 (toolchain locale) | **OUI** | — (smoke OK sur `internal/format`) |
| `go test -race` | — | **NON** (CGO_ENABLED=0, pas de C compiler) | Analyse statique ; reconfirmer sous WSL/Linux → **[à vérifier]** |
| `go test -bench` | go1.26.3 | **OUI** | `-benchtime=2x`, N modérés (énoncé) |
| `go test -fuzz` | go1.26.3 | **OUI** | `-fuzztime` court (30 s/cible) |
| `gofmt` | go1.26.3 | **OUI (avec piège)** | ⚠ **`core.autocrlf=true`** : le working tree est en **CRLF**, l'index en **LF**. `gofmt -l .` liste **tous** les fichiers → **faux positifs purs de fin de ligne** (le `gofmt -d` ne montre que des diffs CRLF→LF, zéro changement de contenu). Le code **commité (index) est LF et gofmt-propre**. Vérifier le formatage réel via la version indexée, pas le working tree. |
| `staticcheck` | 2025.1.1 (build go1.25.4) | **NON tel quel** → recompilé | Refus : « module requires at least go1.26.0, but Staticcheck was built with go1.25.4 ». **Recompilation locale** `go install honnef.co/go/tools/cmd/staticcheck@latest` avec go1.26.3 (en cours). Si échec : analyse via `go vet` + lecture. |
| `golangci-lint` | v1.64.8 (build go1.25) | **NON tel quel** → recompilé | Même refus de version. `.golangci.yml` est en **schéma v1** (`disable-all`, `gosimple`, `gofmt`) — **incompatible** avec golangci-lint v2 (current). Recompilation **v1.64.8** avec go1.26.3 (préserve la compat de config). Si échec : `go vet` + `staticcheck` + `gosec`. |
| `gosec` | dev | **OUI** | Smoke OK (exit 0 sur `internal/format`). |
| `govulncheck` | v1.1.4 (Go 1.26.3, DB 2026-05-26) | **OUI** | Scan de vulnérabilités des dépendances + appels atteignables. |

> **Note tooling = constat d'audit** : l'écosystème d'analyse statique préinstallé (`staticcheck`, `golangci-lint`) n'avait **pas été reconstruit pour go1.26** et refusait d'analyser le module. De plus `.golangci.yml` reste en schéma v1 alors que golangci-lint v2 est la branche courante. À traiter par le sous-agent 4 (méthodologie) et le sous-agent 5 (outillage/CI). [confirmé]

---

## 6) Stratégie d'orchestration retenue

- **Vagues de 2–3 agents** (consigne énoncé + doc agent-teams : « 2–3 focused teammates outperform larger teams »).
- **Isolation du sous-agent Performance** : ses benchmarks tournent dans une vague séparée, sans contention CPU concurrente, pour des `ns/op` / `B/op` exploitables.
- **Partage d'état par fichiers** : chaque sous-agent écrit **un seul** fichier (`audit/0X_*.md`) → zéro conflit d'écriture. Lecture seule du code source ; **aucune modification** de `./` ni de `testdata/`.
- **Contexte autonome** : chaque sous-agent reçoit chemin du dépôt, SHA, `CONVENTIONS.md`, résumé bootstrap, et son mandat.
