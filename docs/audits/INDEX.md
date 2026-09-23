# Index des identifiants d'audit

Ce fichier existe pour une raison simple : le code de production cite environ
**350 identifiants d'audit** dans ses commentaires (`FIB-02`, `OVR-10`, `M-01`,
`A2-04`, `R4.2`…), et jusqu'au 2026-09-07 aucun d'eux ne renvoyait à quoi que ce
soit de présent dans l'arbre. Les rapports d'audit qui les définissaient étaient
**supprimés après exécution** ; il ne restait que le CHANGELOG, les ADR, et un
lecteur obligé de faire `git log -S` pour comprendre une note de trois mots.

C'est le constat DOC-01 de l'audit 2026-09-07. La correction n'est pas de retirer
les identifiants — ils accompagnent le plus souvent une justification déjà écrite
en clair, et les retirer coûterait un diff énorme pour rien — mais de **les rendre
résolubles**.

## Règle

Un identifiant d'audit dans un commentaire est un **renvoi**, jamais l'explication.
Le commentaire doit rester compréhensible si on efface l'identifiant. Voir
`CONTRIBUTING.md`, section *Comments*.

Un identifiant n'est admis que si son préfixe figure dans le tableau ci-dessous.

## Où chaque famille est définie

| Préfixes | Campagne | Où lire le détail |
|---|---|---|
| `PRO-`, `STR-`, `TYP-`, `API-`, `TST-`, `MEM-`, `CON-`, `CFG-`, `ARC-`, `DEP-`, `OBS-`, `DOC-`, `SEC-` | Audit 2026-09-07 « livre » (grille *Building Enterprise Projects with Go*) | [`audit-2026-09-livre.md`](audit-2026-09-livre.md), décisions dans [ADR-0012](../adr/0012-audit-2026-09-livre-decisions.md) |
| `H-`, `M-`, `L-`, `FIB-`, `GATE-` | Audit exhaustif 2026-09 (sévérités Haute / Moyenne / Basse) | `CHANGELOG.md`, section « Audit exhaustif 2026-09 » ; décisions dans [ADR-0010](../adr/0010-audit-2026-09-decisions.md) |
| `DEAD-`, `ERR-`, `CONC-`, `CAL-`, `TEST-` | Vague post-v4.0.0, audit Fable5 (2026-07-07 → 2026-07-11) | `CHANGELOG.md`, section « Vague post-v4.0.0 — audit Fable5 » |
| `APP-`, `SEC-`, `FFT-`, `OVR-` | Audit exhaustif 2026-07 (`OVR-` = *over-engineering*) | `CHANGELOG.md`, section « Audit exhaustif 2026-07 » ; candidats rejetés dans [ADR-0008](../adr/0008-audit-2026-06-rejected-candidates.md) |
| `R1.`–`R4.` | Candidats **rejetés** de l'audit 2026-07, avec la mesure qui les rejette | [ADR-0009](../adr/0009-audit-2026-07-cleanup-and-rejected-fib05.md) |
| `A1-`, `A2-`, `A3-`, `A5-`, `A-` | Audit loop (2026-06-10) et remediation (mai 2026), lots A1 à A5 | `CHANGELOG.md`, sections « Audit loop (2026-06-10) » et « Audit remediation (May 2026) » |
| `F-` | Correctifs de fonctionnement relevés par l'audit loop (2026-06-10), p. ex. `F-012` | `CHANGELOG.md`, section « Audit loop (2026-06-10) » |
| `ARCH-` | Constats de couches, campagne Fable5 (2026-07) | `CHANGELOG.md`, section « Vague post-v4.0.0 — audit Fable5 » |
| `P0-`, `P1-`, `P2-`, `P3-` | Audit remediation (mai 2026), lots de priorité 0 à 3 | `CHANGELOG.md`, section « Audit remediation (May 2026) » |
| `EVAL-` | Évaluation académique 2026-09-15 (grille universitaire C1–C10), tâches du plan d'exécution | [`evaluation-academique-2026-09-15.md`](evaluation-academique-2026-09-15.md), plan [`plan-evaluation-2026-09-15.md`](plan-evaluation-2026-09-15.md), décisions dans [ADR-0013](../adr/0013-evaluation-2026-09-decisions.md) |
| `ADR-00NN` | Décision d'architecture | [`docs/adr/`](../adr/) |

Une décision notée `D1`…`D5` sans autre contexte appartient à l'ADR cité dans la
même phrase.

## Artefacts de mesure conservés ici

Contrairement aux rapports, les mesures sont gardées : ce sont elles qui
permettent de rejouer une comparaison.

| Fichier | Contenu |
|---|---|
| [`bench-baseline.txt`](bench-baseline.txt) | Référence de non-régression comparée par `benchstat` au seuil de 5 % |
| [`bench-dtm-2026-09.txt`](bench-dtm-2026-09.txt) | Seuils dynamiques, mesure ayant motivé ADR-0001 |
| [`bench-fftcache-2026-09.txt`](bench-fftcache-2026-09.txt) | Plafond de la cache FFT, mesure du rejet ADR-0010 R1 |
| [`bench-poolclear-2026-09.txt`](bench-poolclear-2026-09.txt) | Vidage de pool |
| [`mem-baseline-2026-09.txt`](mem-baseline-2026-09.txt) | Empreinte mémoire mesurée, base de l'estimateur |
| [`microbench-stability-2026-09.txt`](microbench-stability-2026-09.txt) | Stabilité des micro-benchmarks de calibration |

## Conserver les rapports

Le rapport d'un audit est archivé ici après exécution, il n'est plus supprimé.
C'est ce qui manquait : les campagnes 2026-05 à 2026-09-03 ont laissé des
identifiants sans définition parce que leur rapport a été effacé de l'arbre une
fois le travail fait.
