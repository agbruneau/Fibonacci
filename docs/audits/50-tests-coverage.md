# 50 — Couverture & golden tests

## Couverture globale
- **Total : 88.9 %** (3 722 / 4 185 statements couverts)
- Méthode : `coverage.out` (mode `set`) parsé directement par awk — agrégation par package depuis les chemins `github.com/agbru/fibcalc/<pkg>/<file>`.
- Note : `go tool cover -func=coverage.out` échoue car `internal/fibonacci/generator_iterative.go` est référencé dans le profil mais absent du WC (profil obsolète). À régénérer via `make coverage`.

## Top 5 packages (couverture %)
| Package | Couverture |
|---|---|
| `internal/fibonacci/fibonaccitest` | 100.0 % |
| `internal/parallel` | 100.0 % |
| `internal/sysmon` | 100.0 % |
| `internal/testutil` | 100.0 % |
| `internal/metrics` | 96.1 % |

## Bottom 5 packages
| Package | Couverture | Justification possible |
|---|---|---|
| `cmd/generate-golden` | 29.4 % | Outil one-shot, pas de tests dédiés (oracle indépendant). Acceptable. |
| `cmd/fibcalc` | 75.0 % | Entrypoint mince, l'essentiel est testé via `internal/app` + e2e. |
| `internal/format` | 79.8 % | Branches d'affichage rares (très grandes ETA, edge cases unités). |
| `internal/fibonacci/memory` | 82.2 % | Chemins de pression mémoire/fallback difficiles à exercer. |
| `internal/orchestration` | 85.7 % | Branches d'erreur de cancel/ctx non couvertes. |

Aucun package <50 %. Seul `cmd/generate-golden` (oracle) est sous 75 %.

## Fichiers à 0 %
Top hotspots de blocs non couverts (≠ 0 % global, mais branches mortes/erreurs) :
- `internal/tui/sparkline.go` (23 blocs), `internal/tui/model.go` (10) — rendering TUI difficile à harnacher.
- `internal/bigfft/fft_cache.go` (23), `fft_poly.go` (18), `pool.go` (16), `fft_recursion.go` (12) — chemins d'éviction LRU et fallback.
- `internal/fibonacci/doubling_framework.go` (17), `fft.go` (16) — branches d'erreur multiplieur.
- Aucun fichier entièrement à 0 % détecté.

## Golden tests
- **Fichier** : `internal/fibonacci/testdata/fibonacci_golden.json` (7 414 octets)
- **# entrées : 23** ; **N min = 0**, **N max = 10 000**
- **Cibles** : 0–5, 10, 20, 50, 92–94 (autour du débordement uint64 natif), 100, puissances de 2 (128…8192), puissances de 10 (1000, 10000), 2000, 5000.
- **Consommé par** : `internal/fibonacci/fibonacci_golden_test.go::TestCalculatorsAgainstGoldenFile` — exerce `FastDoubling`, `MatrixExp`, `FFTBased` via `t.Parallel()` × N.
- **Robustesse** : `t.Fatalf` sur fichier absent (message guide vers `cmd/generate-golden`) ET sur erreur de décodage JSON. Pas de `SetString` error-check (entrée corrompue → big.Int = 0, mismatch silencieux possible mais détecté par `Cmp`).

## Générateur golden
- `cmd/generate-golden/main.go` : oracle itératif `fibBig` indépendant (P2-04, doc.go interdit l'unification avec `calculateSmall`). Flag `-out`, ciblage codé en dur (23 valeurs).
- **Reproductible** : oui, déterministe.
- **CI** : aucun workflow GitHub Actions (`.github/` absent). Aucune cible Makefile pour `generate-golden` ni `fuzz`/`e2e`. Régénération manuelle uniquement.

## Tests E2E
| Scénario | Localisation |
|---|---|
| Modes CLI (algos, parallèle, JSON, raw) | `test/e2e/cli_e2e_test.go::TestCLI_E2E` |
| Flags invalides / erreurs CLI | `cli_e2e_test.go::TestCLI_InvalidFlags`, `extended_e2e_test.go::TestCLI_ErrorCases` |
| Sortie fichier (`-output`) | `TestCLI_OutputFile` |
| Timeout sur grand N | `TestCLI_TimeoutLargeN` |
| Auto-complétion shell | `TestCLI_Completion` |
| Derniers chiffres (`-last-digits`) | `TestCLI_LastDigits` |
| Mode comparaison multi-algos | `TestCLI_CompareMode` |
| Détails version | `TestCLI_VersionDetails` |
| Valeurs golden via CLI | `extended_e2e_test.go::TestCLI_GoldenValues` |
| Combinaison de modes | `TestCLI_ModesCombination` |
| Formatage sortie | `TestCLI_Formatting` |

## Fuzz tests
- **5 cibles** dans `internal/fibonacci/fibonacci_fuzz_test.go` : `FuzzFastDoublingConsistency`, `FuzzFFTBasedConsistency`, `FuzzFibonacciIdentities`, `FuzzFastDoublingMod`, `FuzzProgressMonotonicity`.
- **Inputs minimisés** : 15 seeds checkés-in (`testdata/fuzz/<Cible>/seed-*`), couvrant zero/boundary/large/petit/edge.
- Pertinence : haute — vérifie cross-algos (FastDoubling vs FFT), identités F(n+m), modulo, monotonie progress.

## Examples
- 5 exemples dans `internal/fibonacci/example_test.go` + `fibonacci_test.go` (`MustNewCalculator`, `DefaultFactory`, `CalculateWithObservers`, `Example_smallValues`, `ExampleCalculator_Calculate`). Servent de godoc exécutable.

## Synthèse
**Forces** :
1. Couverture globale 88.9 % très solide pour un projet de ~37k LOC ; 4 packages à 100 %.
2. Golden test cross-algos (3 calculateurs) avec oracle indépendant intentionnel — design exemplaire (P2-04).
3. 11 scénarios E2E + 5 cibles fuzz avec corpus minimisé checké-in.

**Faiblesses** :
1. **Aucune CI** (`.github/` absent) — golden, fuzz, e2e dépendent de l'exécution locale.
2. `coverage.out` obsolète (référence `generator_iterative.go` introuvable) — `go tool cover -func` plante.
3. Borne supérieure golden N = 10 000 — ne stresse pas les chemins FFT (seuil ~500k bits ≈ N≈100k).

**Top 3 actions** :
1. Ajouter `.github/workflows/ci.yml` exécutant `make test`, fuzz court (`-fuzztime=30s`), golden, e2e à chaque PR.
2. Régénérer `coverage.out` (`make coverage`) et l'ignorer du repo (artefact).
3. Étendre `cmd/generate-golden` avec quelques N ≥ 200 000 pour couvrir le palier FFT dans `TestCalculatorsAgainstGoldenFile`.
