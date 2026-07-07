# PLAN-4 — Brancher le backend GMP (`-tags gmp`) au gate local

**Rang levier : 4/5.** Effort : ~1 h. Impact : le backend GMP (`internal/fibonacci/calculator_gmp.go`, build tag `gmp`, CGO + libgmp) a **zéro validation automatisée** sur cette machine : WSL n'a pas `libgmp-dev` (vérifié le 2026-07-07 : gcc présent, GMP absent) et aucun script du gate ne compile le tag. Conséquence déjà observée : le build `gmp` a **cassé silencieusement** (réparé pendant l'audit 2026-07 — « `globalFactory` déplacé dans `calculator_gmp.go` après la suppression de la fabrique globale »). Sans garde, la prochaine purge de symbole le recassera sans qu'aucun test ne rougisse.

## Objectif

1. `libgmp-dev` installé dans WSL ; la suite `gmp` passe sous `-race`.
2. `scripts/check.sh` compile et teste le tag `gmp` **quand libgmp est présent** (étape dure), et l'annonce comme SKIP sinon.
3. Les notes docs (« libgmp-dev absent de WSL ⇒ tag gmp non testable là ») mises à jour.

## Fichiers à modifier

1. `scripts/check.sh` — insertion d'une étape entre l'étape 3 (tests) et l'étape 4 (lint).
2. `CLAUDE.md` — section « Commandes essentielles », puce `-race` (phrase « `libgmp-dev` absent de WSL ⇒ tag `gmp` non testable là »).
3. `docs/TESTING.md` — ligne ~318 (note « Validate it on a Linux runner with `libgmp-dev` installed »).

**Ne PAS toucher** : `scripts/check.ps1` (Windows sans CGO — le tag `gmp` y est impossible par construction) ; `internal/fibonacci/calculator_gmp*.go` (aucun changement de code requis).

## Étapes

### Étape 1 — Installer libgmp dans WSL

```powershell
wsl -u root -e bash -lc "apt-get update && apt-get install -y libgmp-dev"
```

(Si `wsl -u root` est refusé par la config WSL, lancer à la place dans un terminal WSL interactif : `sudo apt-get update && sudo apt-get install -y libgmp-dev`.)

**Vérifier** :

```powershell
wsl -e bash -lc "dpkg -s libgmp-dev | grep Status"
```

Attendu : `Status: install ok installed`.

### Étape 2 — Passe GMP de référence (avant tout changement de script)

```powershell
wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && CGO_ENABLED=1 go test -tags gmp -race -count=1 ./internal/fibonacci/"
```

**Vérifier** : PASS. Cette passe exécute notamment `TestGMPCalculator_CalculateCore`, `_Cancel`, `_Name` et `TestGMPCalculator_CrossValidateFastDoubling` (validation croisée GMP ↔ Fast Doubling). **Ne pas ajouter `-short`** : le cross-val se skippe en mode short et la passe deviendrait creuse.

### Étape 3 — Ajouter l'étape au gate `check.sh`

Dans `scripts/check.sh`, insérer entre l'étape 3 (ligne 48, `echo "OK: go test -race"`) et l'étape 4 (ligne 50, commentaire lint) :

```bash
# 3b. GMP build tag (hard when libgmp is available, skipped otherwise).
# The gmp backend (CGO + libgmp) is not compiled by the default steps above;
# it has silently broken before (globalFactory, fixed in the 2026-07 audit).
step "gmp build tag (-tags gmp)"
if [ -f /usr/include/gmp.h ] || [ -f /usr/include/x86_64-linux-gnu/gmp.h ]; then
    go build -tags gmp ./...
    go vet -tags gmp ./internal/fibonacci/
    go test -tags gmp -race -count=1 ./internal/fibonacci/
    echo "OK: gmp build tag"
else
    echo "SKIP: gmp (libgmp headers not found; apt-get install libgmp-dev)"
fi
```

Mettre aussi à jour le bloc de commentaires d'en-tête du script (liste « Steps: » lignes 8–13) pour y insérer l'étape `3b`.

**Vérifier la syntaxe** avant exécution :

```powershell
wsl -e bash -lc "bash -n /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo/scripts/check.sh && echo SYNTAX-OK"
```

### Étape 4 — Exécuter le gate complet modifié

```powershell
wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && bash scripts/check.sh"
```

**Vérifier** : la sortie contient `==> gmp build tag (-tags gmp)` puis `OK: gmp build tag`, et `Overall: PASS`.

### Étape 5 — Vérifier la branche SKIP (le gate ne doit pas casser ailleurs)

Simuler l'absence de headers en inspectant la condition : la garde teste des chemins de fichiers, pas une commande — sur un hôte sans libgmp elle imprime `SKIP` et continue (pas de `exit`). Relire le diff pour confirmer qu'aucun chemin de la branche `else` ne peut faire échouer le script sous `set -euo pipefail` (un simple `echo` est sûr).

### Étape 6 — Mettre à jour les docs

- `CLAUDE.md`, puce `-race` : remplacer « `libgmp-dev` absent de WSL ⇒ tag `gmp` non testable là » par « `libgmp-dev` installé dans WSL (2026-07) ⇒ passes `gmp` : `wsl go test -tags gmp -race ./internal/fibonacci/` ; `check.sh` compile+teste le tag quand libgmp est présent ».
- `docs/TESTING.md` ~318 : compléter la note GMP : le tag est désormais validé localement via WSL + l'étape `3b` de `scripts/check.sh` (et reste couvert par `.devcontainer/` qui embarque libgmp).

### Étape 7 — Gate final + commit

```powershell
wsl -e bash -lc "cd /mnt/c/Users/agbru/OneDrive/Documents/GitHub/FibGo && bash scripts/check.sh"
git add scripts/check.sh CLAUDE.md docs/TESTING.md
git commit -m "test(gmp): wire -tags gmp build+race tests into check.sh gate (guarded by libgmp presence)"
git push origin main
```

## Cas limites

- **La garde par headers plutôt que `pkg-config`** : `libgmp-dev` sur Debian/Ubuntu installe `gmp.h` sous `/usr/include/x86_64-linux-gnu/` (multiarch) ou `/usr/include/` selon la version — tester **les deux** chemins ; `pkg-config --exists gmp` échoue sur certaines versions qui ne shippent pas `gmp.pc`.
- **`set -euo pipefail` est actif** : toute commande ajoutée qui peut échouer légitimement (la détection) doit être dans un `if`, jamais nue, sinon le gate meurt au lieu de skipper.
- **Ne pas mettre l'étape dans `check.ps1`** : Windows sans gcc ne peut ni compiler ni tester du CGO ; y ajouter un skip serait du bruit. Le gate Windows reste inchangé — c'est documenté à l'étape 6 côté CLAUDE.md.
- **`go test -tags gmp ./...` (tout le module) est inutilement long** : seul `internal/fibonacci` contient du code taggé `gmp` — cibler ce package. En revanche `go build -tags gmp ./...` reste sur tout le module : c'est lui qui aurait attrapé la casse `globalFactory` (erreur de compilation inter-fichiers).
- **`-count=1`** dans l'étape gmp : le cache de test Go ne tient pas compte de la présence de libgmp au moment du cache ; forcer l'exécution évite un faux vert après installation de la lib.
- La couverture du gate (plancher 80 %) est calculée sur la passe **sans** tag à l'étape 3 — l'étape `3b` n'écrit pas de `coverprofile` et ne doit pas écraser `coverage.out`.
- Alternative sans WSL : `.devcontainer/devcontainer.json` embarque déjà Go + CGO + libgmp — utilisable si WSL est indisponible, mais ne remplace pas l'étape gate.

## Critères d'acceptation

1. `wsl -e bash -lc "dpkg -s libgmp-dev | grep Status"` → `install ok installed`.
2. `wsl ... go test -tags gmp -race -count=1 ./internal/fibonacci/` → PASS (sans `-short`).
3. `wsl ... bash scripts/check.sh` → affiche `OK: gmp build tag` et `Overall: PASS`.
4. Relecture du diff `check.sh` : la branche « libgmp absent » ne peut pas faire échouer le script (aucune commande faillible hors `if`).
5. `CLAUDE.md` et `docs/TESTING.md` ne contiennent plus l'affirmation périmée « gmp non testable dans WSL » (`Select-String -Path CLAUDE.md,docs/TESTING.md -Pattern 'non testable'`).
6. `pwsh scripts/check.ps1` inchangé et toujours PASS ; commit poussé sur `main`.
