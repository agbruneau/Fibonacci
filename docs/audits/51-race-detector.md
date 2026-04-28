# 51 — Race detector & tests parallèles

## Méthode
- Commande visée : `go test -race -timeout 600s ./...`
- Log brut : `docs/audits/_test_raw.log`
- Run de repli (CGO indisponible) : `CGO_ENABLED=0 go test -timeout 600s ./...` → `docs/audits/_test_norace.log`
- Run verbeux pour skips : `CGO_ENABLED=0 go test -v -timeout 600s ./...` → `docs/audits/_test_skips.log`
- Plateforme : Windows 11, `GOOS=windows`, `GOARCH=amd64`

## Statut global

**BLOQUANT — race detector non exécutable sur cette machine.**

Contenu intégral de `_test_raw.log` (1 ligne) :

```
go: -race requires cgo; enable cgo by setting CGO_ENABLED=0
```

Cause : Go détecte `CGO_ENABLED=0` (ou plus probablement absence de toolchain C). `which gcc` → introuvable ; `C:\msys64\ucrt64\bin\gcc.exe`, `mingw64`, `TDM-GCC` → tous absents. Le runtime `-race` requiert un linker C.

Statut sur le run de repli (sans `-race`) :

| Indicateur | Valeur |
|---|---|
| Packages testés | 22 |
| PASS | 22 |
| FAIL | 0 |
| SKIP (lignes) | 0 |
| Build failures | 0 |

Tous les packages passent en mode standard. La conformité fonctionnelle est confirmée ; **la conformité concurrence ne l'est pas sur cette plateforme**.

## Data races détectées

Indéterminé — l'instrumentation n'a pas tourné. Aucune ligne `DATA RACE` dans le log (et pour cause).

| Localisation | Contexte | Sévérité |
|---|---|---|
| n/a | Race detector non exécuté (CGO requis) | Inconnu |

## Tests skipped

Aucun `--- SKIP:` détecté dans le run verbeux (0 occurrence sur 22 packages). Notable : pas de garde `testing.Short()` actif sans `-short`, pas de skip OS-spécifique observé sur Windows.

| Pattern | Raison probable |
|---|---|
| (aucun) | Suite test entièrement active sur Windows/amd64 |

## Top packages par durée (sans race)

| Package | Durée | Notes |
|---|---|---|
| `internal/tui` | 6.34 s | Bubble Tea, rendu/animations |
| `cmd/generate-golden` | 4.36 s | Génération données de test |
| `test/e2e` | 3.99 s | Bout-en-bout CLI |
| `internal/calibration` | 2.08 s | Micro-benchmarks calibration |
| `cmd/fibcalc` | 2.05 s | Tests CLI principal |
| `internal/fibonacci` | 0.93 s | Cœur algorithmique (étonnamment court sans -race) |
| `internal/bigfft` | 0.52 s | FFT |

Estimation `-race` : facteur 2–10× → `tui` ~30–60 s, `fibonacci`/`bigfft` ~5–10 s. Bien sous le timeout 600 s.

## Build failures

Aucun. Aucun `[build failed]` ni `cannot find package` dans les logs.

## Couplage avec t.Parallel()

- Audit 3.3 : ~94 % des `Test*` invoquent `t.Parallel()` → la suite expose effectivement de la concurrence inter-tests.
- Audit 5.3 : `Makefile:162` active `-race`, mais `make test-short` (ligne 165–167) ne l'active pas → fenêtre de régression entre dev (test-short) et CI (test).
- **Confiance race-free sur cette exécution : NULLE.** Le run a échoué avant l'instrumentation. La couverture race historique (CI Linux supposé) reste la seule garantie effective.

## Synthèse

- **Score : Avec issues — non-exécution sur dev Windows.** Tests fonctionnels OK (22/22), mais l'objectif de l'audit (race detector) n'est pas vérifiable localement.
- **Cause racine** : absence de toolchain C sur la machine dev. `make test` est silencieusement non-conforme à sa propre cible sur Windows sans MSYS2/TDM-GCC.

### Top actions

1. **Documenter la dépendance CGO** dans le README/CLAUDE.md : `make test` requiert gcc (MSYS2 UCRT64 ou équivalent) sur Windows.
2. **Vérifier que la CI exécute bien `-race`** (Linux a CGO par défaut, mais confirmer via `.github/workflows/`).
3. **Ajouter une cible `make test-race-check`** qui échoue explicitement si CGO indisponible, plutôt que de laisser le warning Go se perdre.
4. **Re-exécuter cet audit** sur Linux ou Windows+MSYS2 pour valider l'absence réelle de data races sur les ~94 % de tests parallèles.
