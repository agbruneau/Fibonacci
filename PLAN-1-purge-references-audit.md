# PLAN-1 — Purger les références mortes vers `audit.md` / `auditPlan.md`

**Rang levier : 1/5 (commencer ici).** Effort : ~15 min. Impact : chaque session IA future lit CLAUDE.md/README en premier ; elles pointent aujourd'hui vers deux fichiers **supprimés** (commit `d10299b` « purge audit », 2026-07-06), ce qui induit en erreur tout agent et casse 6 liens Markdown rendus sur GitHub.

## Objectif

Zéro lien Markdown `[...](audit.md)` ou `[...](auditPlan.md)` (et variantes `../../audit.md`) dans le dépôt. La traçabilité de l'audit 2026-07 pointe désormais vers [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md), le CHANGELOG et le commit de purge `d10299b`.

## Fichiers à modifier (exactement 5, aucun autre)

1. `CLAUDE.md` — lignes 18 et 206
2. `README.md` — lignes 24 et 216
3. `CHANGELOG.md` — lignes 12–13 (section `[Unreleased]` uniquement)
4. `docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md` — lignes 5–8

## Étapes

### Étape 1 — CLAUDE.md ligne 18 (table « Projet », rangée Audit)

Remplacer le fragment :

```
Rapport [`audit.md`](audit.md), plan d'exécution [`auditPlan.md`](auditPlan.md)
```

par :

```
Rapport `audit.md` et plan `auditPlan.md` purgés post-exécution (commit `d10299b`) ; traçabilité : ADR-0009 + [CHANGELOG](CHANGELOG.md)
```

**Vérifier** : la rangée de table reste sur une seule ligne (les tables Markdown cassent sur un retour ligne).

### Étape 2 — CLAUDE.md ligne 206 (section Références)

Remplacer la puce :

```
- [`audit.md`](audit.md) / [`auditPlan.md`](auditPlan.md) — audit exhaustif 2026-07 et plan d'exécution des correctifs.
```

par :

```
- Audit exhaustif 2026-07 — exécuté puis purgé (`d10299b`) ; décisions résiduelles dans [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) et [`CHANGELOG.md`](CHANGELOG.md).
```

### Étape 3 — README.md ligne 24 (table « Historique des audits », rangée 2026-07)

En fin de cellule, remplacer :

```
— [`audit.md`](audit.md) / [`auditPlan.md`](auditPlan.md) / [`CHANGELOG.md`](CHANGELOG.md)
```

par :

```
— [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) / [`CHANGELOG.md`](CHANGELOG.md)
```

**Vérifier** : rangée toujours sur une seule ligne.

### Étape 4 — README.md ligne 216 (section Développement et tests)

Remplacer :

```
  Dernier audit : [`audit.md`](audit.md) / [`auditPlan.md`](auditPlan.md).
```

par :

```
  Dernier audit (2026-07) : exécuté puis purgé — voir [ADR-0009](docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) et [`CHANGELOG.md`](CHANGELOG.md).
```

### Étape 5 — CHANGELOG.md lignes 12–13 (intro de la section audit 2026-07, dans `[Unreleased]`)

Remplacer :

```
Audit exhaustif de toute la base ([`audit.md`](audit.md)) exécuté via
[`auditPlan.md`](auditPlan.md) en orchestration multi-agents (modèle exécuteur
```

par :

```
Audit exhaustif de toute la base (rapport `audit.md`, purgé post-exécution au
commit `d10299b`) exécuté via son plan `auditPlan.md` (idem) en orchestration
multi-agents (modèle exécuteur
```

### Étape 6 — ADR-0009 lignes 5–8

Remplacer :

```
- **Context source**: audit exhaustif 2026-07 ([`audit.md`](../../audit.md)),
  exécuté via le plan [`auditPlan.md`](../../auditPlan.md) (orchestration
```

par :

```
- **Context source**: audit exhaustif 2026-07 (rapport `audit.md`, exécuté via
  le plan `auditPlan.md` ; les deux fichiers ont été purgés post-exécution au
  commit `d10299b`, cf. CHANGELOG) — (orchestration
```

**Attention** : ne modifier QUE ces lignes d'en-tête. Le corps de l'ADR (Decision/Consequences) est un document accepté : ne pas le réécrire.

### Étape 7 — Vérification globale + gate + commit

```powershell
# 1. Plus aucun LIEN vers les fichiers purgés (les mentions en texte brut sont OK) :
Select-String -Path (Get-ChildItem -Recurse -Filter *.md -Exclude PLAN-*.md | Where-Object FullName -notmatch '\\docs\\dashboard\\').FullName -Pattern '\]\((\.\./)*(audit|auditPlan)\.md\)'
# Attendu : AUCUN résultat.

# 2. Diff limité aux 4 fichiers :
git diff --stat
# Attendu : CLAUDE.md, README.md, CHANGELOG.md, docs/adr/0009-... et rien d'autre.

# 3. Gate (docs seulement, mais obligatoire avant commit — Directive #8) :
pwsh scripts/check.ps1

# 4. Commit + push (pratique du dépôt : trunk-based, push direct sur main) :
git add CLAUDE.md README.md CHANGELOG.md docs/adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md
git commit -m "docs: repoint purged audit.md/auditPlan.md links to ADR-0009 + purge commit"
git push origin main
```

## Cas limites (à NE PAS « corriger »)

- **Les commentaires Go citant `audit.md APP-07`, `audit.md SEC-03`, etc.** (`internal/app/app.go:164`, `internal/app/calculate.go:132`, plusieurs `*_test.go`) sont des **identifiants historiques de findings**, pas des liens. Ne pas y toucher — Directive #5 (chirurgie).
- **CHANGELOG lignes ~557 et ~574** mentionnent aussi `audit.md` : ce sont des entrées **historiques** (audit A de 2026 antérieur, et l'entrée qui documente justement la suppression des fichiers d'audit). Un changelog ne se réécrit pas rétroactivement — ne toucher que les lignes 12–13 de `[Unreleased]`.
- **`docs/dashboard/`** est un artefact généré (interdit d'édition manuelle, cf. CLAUDE.md). S'il contient des références, elles disparaîtront à la prochaine régénération — hors périmètre.
- CLAUDE.md ligne 18 et README ligne 24 sont des **cellules de table** : tout retour à la ligne inséré casse le rendu de toute la table.
- Le commit de purge est bien `d10299b` (`git show -s d10299b` pour confirmer avant de le citer).

## Critères d'acceptation

1. La commande `Select-String` de l'étape 7 retourne **zéro** occurrence de lien vers `audit.md`/`auditPlan.md` (hors `docs/dashboard/` et fichiers `PLAN-*.md`).
2. `git diff` ne touche que les 4 fichiers listés ; les commentaires Go et les entrées historiques du CHANGELOG sont intacts (`git diff -- internal/` vide).
3. Les deux tables (CLAUDE.md « Projet », README « Historique des audits ») se rendent correctement (une ligne par rangée).
4. `pwsh scripts/check.ps1` : **PASS**.
5. Commit poussé sur `main`.
