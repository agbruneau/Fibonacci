# Fib

Une même CLI Fibonacci académique en Go, Rust, TypeScript et Python, qui sert de terrain d'essai aux skills de Matt Pocock (plugin `mattpocock-skills`). Exigences, contrat de la CLI et démarche : `PRD.md`. Le temps de réponse prime, et chaque optimisation porte son pourquoi en commentaire d'une ligne (ADR-0004).

## Rôles

- **Orchestrateur** : la session principale, Opus 5.5 en `ultracode` (`.claude/settings.json`). Il conduit les skills de planification et de revue, délègue chaque ticket, intègre et vérifie. Le code d'implémentation revient à `fib-dev`.
- **`fib-dev`** : sous-agent Sonnet 5.5 (`.claude/agents/fib-dev.md`), un ticket et un langage par invocation, dans son propre worktree.

Ce dépôt demande la délégation : la règle globale « ne lance un sous-agent que si je le demande » est levée ici pour `fib-dev` et pour les sous-agents de `/code-review`.

## Boucle de livraison

1. Prendre la frontière : les tickets `ready-for-agent` marqués `waiting`, dont tous les bloqueurs sont fusionnés dans `main` et qui n'attendent aucun prérequis externe.
2. Passer le ticket de `waiting` à `in-progress` (`docs/agents/issue-tracker.md`), puis lancer un `fib-dev` par ticket, en parallèle quand les langages diffèrent. Dans un workflow : `agent(prompt, { agentType: 'fib-dev', isolation: 'worktree' })`. Le prompt donne le numéro du ticket et le langage. Les worktrees partent du `HEAD` local (`worktree.baseRef: "head"` dans `.claude/settings.json`), donc le travail fusionné sur `main` y est visible sans push. Pour un travail de performance, mesurer d'abord `main` avec `npm run bench`, et fixer dans le prompt le nom des tests de garde `PERF-<ID>-<n>` (`docs/cas-de-test.md` § 6). Pour un travail de démarrage (imports, chargement, point d'entrée), c'est l'A/B apparié qui tranche, non deux passes du banc : `npm run bench -- <id> --contre <révision d'avant le travail>` (ADR-0004).
3. Au retour, fusionner la branche dans `main` (conflit : skill `resolving-merge-conflicts`), puis lancer `npm run check:<id>` et `npm run contract`, et `npm run bench` pour un travail de performance, ou `npm run bench -- <id> --contre <commit d'avant la fusion>` pour un travail de démarrage, machine au repos (100 paires par défaut, toujours en nombre pair; ne pas descendre sous 100 pour trancher un écart de quelques pour cent).
4. `/code-review` sur la plage `<commit d'avant la fusion>..<branche>`, jamais sur un SHA seul, qui ne revoit que ce commit (journal du 2026-10-12). Un constat à corriger repart chez un `fib-dev` avec le rapport.
5. Quand `npm run check:<id>` et `npm run contract` sont verts sur `main`, retirer l'étiquette de statut et fermer le ticket. S'il manque un prérequis externe, repasser le ticket à `waiting`, avec un commentaire qui le nomme.
6. Ajouter à `docs/exploration.md` le constat sur chaque skill utilisée (format : PRD § 9).

## Vérification

- `npm run check:<go|rust|ts|py>` : un langage, avec son contrat.
- `npm run contract` : le contrat des implémentations présentes.
- `npm run check` : les quatre langages, vert depuis le 2026-10-10.
- `npm run differentiel [-- go|rust|ts|py …] [--dimension …] [--jetons K] [--revision REF]` : la boucle différentielle, un outil de recherche (ADR-0001), hors de `check` et de `contract`; code 1 s'il y a un écart, qu'on tranche ensuite par le PRD § 4 et `diagnosing-bugs`.
- `npm run bench [-- go|rust|ts|py …] [--runs N] [--contre REF] | --nettoyer` : le banc d'essai, médianes du temps de réponse (ADR-0004). Toute mesure de performance passe par lui. `--contre REF` en fait un A/B apparié de REF contre l'arbre de travail, la mesure d'un travail de démarrage; la révision s'extrait sous `.build/revisions/`, à vider par `npm run bench -- --nettoyer`, jamais par `Remove-Item -Recurse`.

Les scripts vivent dans `package.json`, créé par le ticket #2 (PRD § 9). Sous Windows, Rust passe par une override rustup GNU propre à ce dossier (PRD § 5). L'oracle se régénère avec `python contracts/generer_oracle.py` (`contracts/README.md`).

## Agent skills

### Issue tracker

Issues GitHub de `agbruneau/Fib`, via le CLI `gh`. Voir `docs/agents/issue-tracker.md`.

### Triage labels

Les cinq étiquettes canoniques, chacune égale à son nom (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). Voir `docs/agents/triage-labels.md`.

### Domain docs

Single-context : `CONTEXT.md` et `docs/adr/` à la racine. Voir `docs/agents/domain.md`.
