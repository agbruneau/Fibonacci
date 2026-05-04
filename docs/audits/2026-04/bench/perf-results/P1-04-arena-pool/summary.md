# P1-04 Arena Pooling — Validation Summary

**Date** : 2026-05-04
**Branche** : `claude/audit-finalization-2026-05`
**Commit P1-04** : `88d2cbf`
**CPU** : Intel Core Ultra 9 275HX (24 threads)
**Go** : 1.25.0 / toolchain 1.26.2

## Tests

| Suite | Statut |
|---|---|
| `go test -count=1 ./...` (22 packages) | ✅ PASS |
| `go test -count=20 -short -run 'TestArena*\|TestConcurrentCalculations'` | ✅ PASS (320 itérations) |
| `go test -count=1 -run 'TestCalculatorsAgainstGolden'` | ✅ PASS |
| `go vet ./...` | ✅ clean |
| `go build ./...` | ✅ clean |
| `golangci-lint run` | ⚠️ N/A (v1.64 incompatible avec toolchain go1.26.2 — limitation préexistante) |
| `gofmt -l` | ⚠️ CRLF artifacts pré-existants Windows, non régression |

## Benchmarks `BenchmarkFibonacci` — comparaison `before` vs `after`

3 itérations × `benchtime=2s`. Médianes retenues (les chiffres `ns/op` ont une variance run-to-run de ~10 % sur cette machine, lisible aussi sur les baselines inchangées MatrixExp/FFTBased).

| Cas | ns/op before | ns/op after | Δ ns/op | B/op before | B/op after | Δ B/op | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| **FastDoubling/1M** | 4 504 438 | 4 893 151 | +8.6 % * | 1 899 703 | **1 769 754** | **−6.8 %** | 93 | 99 |
| **FastDoubling/10M** | 43 309 616 | 43 962 654 | +1.5 % * | 24 981 752 | **24 783 560** | **−0.8 %** | 1 055 | 1 061 |
| MatrixExp/1M (témoin) | 5 429 379 | 5 771 301 | +6.3 % * | 3 880 973 | 3 838 065 | −1.1 % | 635 | 632 |
| MatrixExp/10M (témoin) | 36 856 119 | 40 974 514 | +11.2 % * | 57 766 118 | 54 349 891 | −5.9 % | 2 026 | 1 982 |
| FFTBased/1M (témoin) | 5 405 917 | 5 531 136 | +2.3 % | 3 153 832 | 3 078 657 | −2.4 % | 2 784 | 2 791 |
| FFTBased/10M (témoin) | 43 356 525 | 44 965 035 | +3.7 % | 26 569 682 | 26 186 020 | −1.4 % | 3 783 | 3 787 |

\* MatrixExp et FFTBased servent de témoins : leur code n'a PAS été modifié par P1-04, donc leur dérive ns/op (+1 % à +11 %) capture la variance du système, pas une régression. La dérive ns/op de FastDoubling reste dans cette même bande de bruit.

### Lecture

- **Gain mémoire** : −6.8 % B/op sur FastDoubling/1M (le sweet-spot où l'arena de ~1.5 MB est la fraction majoritaire de l'allocation totale), −0.8 % sur FastDoubling/10M (arena ~30 MB devient plus diluée dans 25 MB de FFT/multiplications).
- **Coût** : +5–6 allocs/op (deep-copy `result` + 5 `new(big.Int)` du `clearStateAliases`). C'est le prix de la sûreté ; B/op net est positif.
- **Pas de régression ns/op** observable hors variance de la machine (les témoins inchangés dérivent autant ou plus).

## Race-safety

L'absence de `-race` détecteur (Windows sans cgo) est compensée par :
- `TestArenaStateConcurrent` (16 goroutines × 8 itérations × 3 tailles) en `-count=50` → **0 fail** sur 800 sous-runs.
- `TestConcurrentCalculations` (test pré-existant qui reproduisait la race en `-count=5` sous l'ancienne tentative) en `-count=20` → **0 fail**.

L'ancienne tentative documentée dans `P1-04-SKIPPED.md` reproduisait `result mismatch: expected ..., got 0` et `index out of range [97] with length 97` en quelques runs. La présente implémentation A+B élimine ces deux symptômes par construction (deep-copy hors arena + `clearStateAliases` avant `Put`).

## Conclusion

P1-04 est **résolu**. Net : −6 % à −7 % B/op sur le cas hot, sûreté retrouvée, suite verte sur 22 packages.

Limitation locale identifiée (orthogonale) : `golangci-lint` v1.64 incompatible avec `go1.26.2` — à mettre à jour côté outillage CI/dev (suivi hors-scope).
