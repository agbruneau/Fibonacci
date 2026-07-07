# PLAN-3 — Régénérer le profil PGO (`cmd/fibcalc/default.pgo`)

**Rang levier : 3/5.** Effort : ~30 min (surtout du temps machine). Impact : le profil PGO committé date du **2026-04-18** (commit `4ff7f83`, P1-22). Depuis, le hot path a été remodelé par **trois vagues** : hardening mai 2026, audit 2026-06 (−12 % geomean, unification des sémaphores, `executeParallel3`…) et audit 2026-07 (~40 correctifs, ~500 LOC purgées dont `fftState`/`scan.go` dans `bigfft`). Un profil qui échantillonne des symboles disparus ou des chemins remodelés ne guide plus l'inlining : le binaire « PGO » livré par `make build` (qui utilise ce profil automatiquement dès qu'il existe) n'a plus le gain attendu.

## Objectif

`cmd/fibcalc/default.pgo` régénéré depuis les benchmarks actuels, validé (symboles du code courant présents), build PGO vert, committé.

## Fichiers à modifier

1. `cmd/fibcalc/default.pgo` — **régénéré, jamais édité** (fichier binaire pprof).
2. Aucun fichier source.

**Interdit** : `docs/audits/bench-baseline.txt` ne doit **pas** être régénéré ici — c'est l'étalon du gate perf (Directive #1), le régénérer sans cause affaiblirait le gate.

## Contexte d'exécution

Le Makefile est POSIX → tout passe par **WSL** (`go1.26.0` et `make` vérifiés présents dans WSL le 2026-07-07). Chemin du dépôt côté WSL : `/mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo`.

## Étapes

### Étape 1 — Machine au calme

Fermer les applications lourdes (navigateur, indexeurs). Sur portable : branché secteur, mode d'alimentation performance. Un profil bruité = optimisations mal ciblées.

### Étape 2 — Générer le profil

```powershell
wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && make pgo-profile"
```

Cible Makefile exécutée : `go test -cpuprofile=build/cpu.prof -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchtime=5s -count=3 -run='^$' ./internal/fibonacci/` puis `mv` vers `cmd/fibcalc/default.pgo`. Durée attendue : 5–15 min.

**Vérifier** : le message `Profile generated: cmd/fibcalc/default.pgo` s'affiche et `git status` montre `modified: cmd/fibcalc/default.pgo`.

### Étape 3 — Valider le contenu du profil

```powershell
go tool pprof -top -nodecount=25 cmd/fibcalc/default.pgo
```

**Vérifier** (critère dur) : le top contient au moins un symbole de `github.com/agbruneau/FibGo/internal/bigfft` **et** un de `internal/fibonacci` (plus, normalement, beaucoup de `math/big`). Vérifier aussi que la taille du fichier est plausible (> 5 Ko) :

```powershell
(Get-Item cmd/fibcalc/default.pgo).Length
```

### Étape 4 — Build PGO

```powershell
wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && make build-pgo"
```

**Vérifier** : `PGO Build complete: build/fibcalc` sans erreur (`pgo-check` passe, le compilateur accepte le profil).

### Étape 5 — Contrôle de non-régression (spot check, optionnel mais recommandé)

Le profil n'affecte que le binaire compilé, pas les tests. Contrôle rapide de bon sens sur le binaire produit :

```powershell
wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && ./build/fibcalc -n 10000000 -algo fast -quiet"
```

**Vérifier** : le calcul aboutit dans un temps du même ordre que la référence documentée (~30 ms de calcul pur ; le temps total processus inclut le démarrage). Un ordre de grandeur au-dessus = investiguer avant de committer.

### Étape 6 — Gate + commit

```powershell
pwsh scripts/check.ps1
git add cmd/fibcalc/default.pgo
git commit -m "perf(pgo): regenerate default.pgo profile post-audit 2026-07 (stale since 2026-04-18)"
git push origin main
```

## Cas limites

- **Ne PAS générer le profil avec `-tags gmp`** : le binaire livré par défaut n'inclut pas ce backend ; un profil échantillonnant du CGO guiderait mal l'inlining du chemin par défaut.
- **`git diff` est inutile sur ce fichier** (binaire) : la preuve de fraîcheur est `git status` + l'inspection pprof de l'étape 3, pas le diff.
- **Le profil est lié à la machine qui l'a produit** (fréquences relatives des fonctions). C'est acceptable et c'était déjà le cas du profil d'avril — le générer sur la machine de dev principale est le comportement voulu. Ne pas le générer dans un conteneur throttlé.
- **`make build` utilise le profil silencieusement dès qu'il existe** (branche `if [ -f $(PGO_PROFILE) ]`) : un profil corrompu casserait tous les builds futurs — c'est exactement ce que l'étape 4 vérifie avant commit.
- I/O WSL sur `/mnt/c` est lent, mais les benchmarks sont CPU-bound : sans impact sur la qualité du profil.
- Si `make pgo-profile` échoue en cours de bench (OOM, etc.) : le `mv` n'a pas lieu et l'ancien `default.pgo` reste intact — pas d'état à moitié cassé ; relancer.
- Ne pas confondre avec `make bench-baseline` : cible différente, artefact différent, à ne pas toucher.

## Critères d'acceptation

1. `git log -1 --format=%ad -- cmd/fibcalc/default.pgo` = date du jour (le profil n'est plus celui du 2026-04-18).
2. `go tool pprof -top -nodecount=25 cmd/fibcalc/default.pgo` liste ≥ 1 symbole `internal/bigfft` et ≥ 1 symbole `internal/fibonacci`.
3. `make build-pgo` (WSL) : succès.
4. Spot check étape 5 : F(10M) calcule sans anomalie d'ordre de grandeur.
5. `pwsh scripts/check.ps1` : PASS ; `docs/audits/bench-baseline.txt` **non modifié** (`git status`).
6. Commit poussé sur `main`.
