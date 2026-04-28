# 02 — Dépendances

## go.mod
- Module : `github.com/agbru/fibcalc`
- Go : `1.25.0`
- Toolchain : `go1.26.2`

## Dépendances directes

| Module | Version | Utilisé par (packages internes) |
|---|---|---|
| `golang.org/x/sync` | v0.20.0 | `internal/orchestration`, `internal/fibonacci` (tests) |
| `github.com/briandowns/spinner` | v1.23.2 | `internal/cli` (ui, ui_display) |
| `github.com/charmbracelet/bubbles` | v0.21.1 | `internal/tui` (model, keymap, logs) |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | `internal/tui` (bridge, model, logs) |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | `internal/tui` (model, header, metrics, chart, styles, footer), `internal/ui` (themes) |
| `github.com/leanovate/gopter` | v0.2.11 | `internal/fibonacci` (property tests) |
| `github.com/ncw/gmp` | v1.0.5 | `internal/fibonacci` (calculator_gmp, build tag `gmp`) |
| `github.com/rs/zerolog` | v1.35.0 | `internal/app`, `internal/fibonacci` (registry, common, threshold, memory), `internal/bigfft` (fft_cache), `internal/progress` |
| `github.com/shirou/gopsutil/v4` | v4.26.3 | `internal/sysmon` |
| `golang.org/x/sys` | v0.43.0 | `internal/bigfft` (cpu_amd64), `internal/config` (hardware) |

## Dépendances indirectes
- Total : 26
- Liste : `aymanbagabas/go-osc52/v2 v2.0.1`, `charmbracelet/colorprofile v0.4.1`, `charmbracelet/x/ansi v0.11.5`, `charmbracelet/x/cellbuf v0.0.15`, `charmbracelet/x/term v0.2.2`, `clipperhouse/displaywidth v0.9.0`, `clipperhouse/stringish v0.1.1`, `clipperhouse/uax29/v2 v2.5.0`, `ebitengine/purego v0.10.0`, `erikgeiser/coninput v0.0.0-20211004153227`, `fatih/color v1.18.0`, `go-ole/go-ole v1.2.6`, `lucasb-eyer/go-colorful v1.3.0`, `lufia/plan9stats v0.0.0-20211012122336`, `mattn/go-colorable v0.1.14`, `mattn/go-isatty v0.0.20`, `mattn/go-localereader v0.0.1`, `mattn/go-runewidth v0.0.19`, `muesli/ansi v0.0.0-20230316100256`, `muesli/cancelreader v0.2.2`, `muesli/termenv v0.16.0`, `power-devops/perfstat v0.0.0-20240221224432`, `rivo/uniseg v0.4.7`, `tklauser/go-sysconf v0.3.16`, `tklauser/numcpus v0.11.0`, `xo/terminfo v0.0.0-20220910002029`, `yusufpapurcu/wmi v1.2.4`, `golang.org/x/term v0.42.0`, `golang.org/x/text v0.36.0`.

## go.sum
- Présent : oui
- Taille : 68 225 octets (692 lignes)

## Build tags
- `gmp` → `internal/fibonacci/calculator_gmp.go`, `internal/fibonacci/calculator_gmp_test.go` (backend GNU Multiple Precision optionnel)
- `amd64` → `internal/bigfft/arith_amd64.go`, `internal/bigfft/arith_amd64_test.go`, `internal/bigfft/cpu_amd64.go`, `internal/bigfft/cpu_amd64_extended_test.go` (intrinsics CPU FFT)
- `!amd64` → `internal/bigfft/arith_generic.go` (fallback portable)
- Aucun ancien `// +build` détecté ; uniquement la syntaxe moderne `//go:build`.

## Observations
- **Versions globalement récentes** : toutes les dépendances directes sont sur des versions majeures à jour (charmbracelet bubbletea v1.3.x, lipgloss v1.1, zerolog v1.35, gopsutil v4). Aucun module retired ni archivé identifié dans le bloc `require`.
- **Pseudo-versions indirectes** : `erikgeiser/coninput`, `lufia/plan9stats`, `muesli/ansi`, `power-devops/perfstat`, `xo/terminfo` sont sur des commits datés (2021–2024). C'est typique pour ces utilitaires bas-niveau (TTY, stats OS) et reflète le choix amont des auteurs (charmbracelet, gopsutil) ; aucune action requise.
- **`go-ole v1.2.6`** : la version la plus ancienne du graphe (tirée par gopsutil pour Windows WMI) ; stable et toujours maintenue côté amont.
- **Cohérence toolchain** : `go 1.25.0` + `toolchain go1.26.2` correspond à une configuration moderne sans dette de version.
- **Build tag `gmp`** correctement isolé : aucune fuite de l'import `ncw/gmp` hors des fichiers protégés.
