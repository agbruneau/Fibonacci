# PLAN-2 — Couper la release v4.0.0 (CHANGELOG + tag)

**Rang levier : 2/5.** Effort : ~30 min. Impact : le CHANGELOG affiche « Keep a Changelog + SemVer » mais `[Unreleased]` accumule **~570 lignes** couvrant six vagues d'audit depuis 1.0.0 (2025-12-22), et le tag `v3.0.0` (2026-04-18, PR #17) n'a **aucune** entrée `## [3.0.0]` correspondante. Couper une release fige l'état post-audit 2026-07 (gate vert, couverture 95,2 %) et rend le versionnage de nouveau honnête.

**Prérequis : exécuter PLAN-1 d'abord** (la section `[Unreleased]` contient des liens morts qui seraient sinon figés dans la release).

## Objectif

- `CHANGELOG.md` : la section `[Unreleased]` actuelle devient `## [4.0.0] - <date du jour>` ; une nouvelle section `[Unreleased]` vide est créée au-dessus.
- Tag git annoté `v4.0.0` sur le commit de release, poussé sur `origin`.

## Décision de version (déjà prise — ne pas re-débattre)

**v4.0.0** (majeure), car depuis `v3.0.0` le comportement observable du CLI a changé de façon incompatible pour des scripts appelants :

- codes de sortie modifiés : timeout → 2, SIGINT → 130 (APP-04/06), divergence `--algo all --quiet` → 3 ;
- rejet nouveau de combinaisons de flags (`--tui` + `--last-digits`/`--output`, APP-07) ;
- exigence toolchain relevée (`go.mod` : go 1.26.0) ;
- symboles/thèmes retirés (~500 LOC, OVR-*).

SemVer : changement incompatible du contrat observable ⇒ incrément **majeur**.

## Fichiers à modifier

1. `CHANGELOG.md` — en-tête de section ligne 8 + insertion d'une note.
2. Aucun autre fichier (le README n'affiche pas de badge de version — vérifié).

## Étapes

### Étape 1 — Gate complet avant de figer

```powershell
pwsh scripts/check.ps1
```

**Vérifier** : `Overall: PASS`. Une release ne se coupe pas sur un arbre rouge. Vérifier aussi `git status` propre (tout committé).

### Étape 2 — Éditer CHANGELOG.md

À la ligne 8, remplacer :

```
## [Unreleased]
```

par :

```
## [Unreleased]

_Rien pour l'instant._

## [4.0.0] - 2026-07-07

> Note de versionnage : les tags `v2.x`/`v3.0.0` (2026-04-18) n'ont jamais reçu
> d'entrée de version dédiée dans ce fichier ; cette entrée 4.0.0 regroupe donc
> tout le travail livré depuis 1.0.0 (vagues d'audit 2026-04 → 2026-07). Bump
> majeur : codes de sortie CLI modifiés (timeout 2, SIGINT 130, divergence 3),
> nouveaux rejets de combinaisons de flags, exigence Go 1.26.
```

(Adapter `2026-07-07` à la date réelle du jour, format `AAAA-MM-JJ`.)

**Ne rien changer d'autre** : toutes les sous-sections existantes (`### Audit exhaustif 2026-07`, `### Audit Go exhaustif (2026-06-24)`, etc., jusqu'à la ligne précédant `## [1.0.0]`) restent telles quelles — elles se retrouvent simplement sous `[4.0.0]`.

**Vérifier** :

```powershell
Select-String -Path CHANGELOG.md -Pattern '^## '
```

Attendu, dans l'ordre : `[Unreleased]`, `[4.0.0] - 2026-07-07`, `[1.0.0] - 2025-12-22`, `[0.1.0] - 2025-11-01`.

### Étape 3 — Commit de release

```powershell
git add CHANGELOG.md
git commit -m "chore(release): v4.0.0 — cut post-audit-2026-07 changelog"
```

### Étape 4 — Tag annoté + push

```powershell
git tag -a v4.0.0 -m "v4.0.0 — post-audit 2026-07 (couverture 95,2 %, ~40 findings corrigés, ~500 LOC purgées, gate vert)"
git push origin main v4.0.0
```

**Vérifier** : `git tag --list 'v*'` contient `v4.0.0` ; `git ls-remote --tags origin | Select-String v4.0.0` confirme le push.

### Étape 5 (optionnelle) — Release GitHub

Si `gh` est authentifié :

```powershell
gh release create v4.0.0 --title "v4.0.0" --notes "Voir CHANGELOG.md, section [4.0.0] - 2026-07-07."
```

Sinon, sauter — le tag suffit.

## Cas limites

- **Ne PAS reconstituer rétroactivement des entrées `[2.0.0]`/`[3.0.0]`** en découpant la section actuelle : la frontière exacte du contenu couvert par le tag `v3.0.0` dans le texte est ambiguë (les sous-sections non datées des lignes ~495–579 mélangent des vagues antérieures et postérieures au tag). Une mauvaise attribution serait pire que la note honnête de l'étape 2. C'est un choix assumé, documenté dans la note.
- **Pas de liens de comparaison** (`[4.0.0]: https://github.com/...compare/...`) : le fichier n'en a jamais eu ; ne pas introduire une convention nouvelle dans un commit de release.
- La sous-section `_Rien pour l'instant._` sous `[Unreleased]` est volontaire : un `[Unreleased]` totalement vide fait échouer certains parseurs Keep-a-Changelog et invite les futurs commits à écrire au bon endroit.
- Si `scripts/check.ps1` échoue à l'étape 1 : **s'arrêter et corriger d'abord** — ne jamais tagger en contournant le gate.
- `git push origin main v4.0.0` pousse la branche ET le tag en une commande ; un `git push` nu ne pousse pas les tags.

## Critères d'acceptation

1. `Select-String -Path CHANGELOG.md -Pattern '^## '` → exactement 4 en-têtes, dans l'ordre donné à l'étape 2.
2. Le contenu entre `[4.0.0]` et `[1.0.0]` est **octet pour octet** l'ancien contenu de `[Unreleased]` (aux ajouts de l'étape 2 près) : `git diff HEAD~1 -- CHANGELOG.md` ne montre que l'insertion d'en-têtes/note, aucune suppression de contenu.
3. `git ls-remote --tags origin` liste `refs/tags/v4.0.0`.
4. `pwsh scripts/check.ps1` : PASS sur le commit taggé.
