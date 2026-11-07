# Journal d'exploration des skills

Une entrée par usage de skill (PRD § 9, objectif O2).

## 2026-10-10 — `/setup-matt-pocock-skills` (étape 0)

- **A aidé** : en quatre fichiers, le tracker GitHub, les étiquettes de triage et la disposition des docs du domaine étaient réglés, et toutes les skills d'ingénierie y trouvent leur configuration.
- **Friction** : les gabarits sont en anglais. La skill ne crée pas les étiquettes sur GitHub, alors que `/to-spec` en a besoin, d'où une création manuelle de quatre étiquettes. Elle pose une question par section, même quand l'exploration a déjà tranché.
- **À changer** : créer les étiquettes manquantes dans le même passage.

## 2026-10-10 — `/grill-with-docs` (`grilling` + `domain-modeling`, étape 1)

- **A aidé** :
  - Quatre sous-agents ont établi les faits en parallèle : sondes Node, sondes Python/Go/Rust, sources primaires et lecture de FibCalc.
  - Ils ont trouvé de vrais obstacles :
    - Rust ne lie pas avec MSVC sur cette machine;
    - Python écrit `\r\n` sous Windows;
    - argparse peut colorer ses erreurs;
    - tsc 7 exige `"type": "module"`;
    - `node --test` ne prend pas un dossier;
    - les conversions de `N` sont trop permissives dans les quatre langages.
  - Le grilling a forcé des trous à se combler :
    - une grammaire des arguments commune;
    - un texte d'usage identique;
    - le périmètre de `--time`;
    - les codes de sortie, alignés sur FibCalc;
    - une vérification indépendante de F(100 000) par les périodes de Pisano;
    - un découpage en un ticket par langage, puisque le contrat ne se découpe pas en tranches.
  - Le glossaire a fixé indice, terme, liste et comparaison.
- **Friction** : la skill attend un humain qui tranche à chaque ronde. Ici, l'utilisateur a délégué les décisions, donc l'agent a éprouvé et validé son propre PRD, et le regard extérieur que la skill doit apporter a manqué. Les sous-agents ont coûté environ 380 000 jetons.
- **À changer** : lancer la collecte de faits avant la première ronde, et garder au moins une ronde de décisions pour l'humain.

## 2026-10-10 — Audit après `/grill-with-docs` (sans skill)

- **Angle mort** : le grilling a éprouvé le contrat sur papier et par des sondes de bibliothèques, mais n'a jamais exécuté la commande de lancement du PRD. L'audit a mesuré que `go run` remplace le code de sortie par 1 et ajoute `exit status 2` sur stderr, ce qui aurait cassé le contrat dès le premier ticket Go. Il a aussi trouvé que les worktrees de sous-agents partent par défaut de `origin/main`, donc sans le travail fusionné localement.
- **À changer** : exécuter une tranche minimale par langage (skill `prototype`) avant de figer un PRD, plutôt que de sonder des bibliothèques isolées.

## 2026-10-10 — `/to-spec` (étape 4)

- **A aidé** : la synthèse sans entretien a été rapide, puisque le PRD, le glossaire et les ADR existaient. Le gabarit exige des récits utilisateur exhaustifs (46 ici), ce qui a forcé une relecture complète du contrat. La confirmation des seams a été la seule décision humaine.
- **Friction** : la spec reprend presque tout le PRD, ce qui fait deux sources à garder alignées. Le gabarit interdit les chemins de fichiers, alors que le contrat dépend de commandes de lancement précises.
- **À changer** : quand un PRD détaillé existe, faire de la spec un résumé qui renvoie au PRD plutôt qu'une copie.

## 2026-10-10 — `/to-tickets` (étape 5)

- **A aidé** : la règle du tracer bullet a révélé un défaut du découpage prévu par le PRD. Un ticket « contrat » seul livrait un harnais qu'aucune implémentation n'exerçait avant la mise en parallèle. Le premier ticket joint donc le contrat et l'implémentation TypeScript, dans le même langage que le test. Les dépendances natives de GitHub rendent la frontière visible : #2 bloque #3, #4 et #5.
- **Friction** : la réponse de l'API de dépendances renvoie l'issue bloquée, d'où une vérification à part pour confirmer le sens des arêtes. Le gabarit proscrit les chemins de fichiers, alors que les critères d'acceptation reposent sur des commandes précises (`npm run check:<id>`).
- **À changer** : prévoir dans le gabarit une ligne sur les fichiers partagés qu'un ticket peut toucher. Ici, chaque langage ne peut modifier que sa propre commande de lancement dans le test de contrat.

## 2026-10-10 — `tdd` (catalogue de cas de test)

- **A aidé** :
  - L'anti-pattern « tautologique » a imposé une source à chaque attendu : l'oracle, un fait mathématique ou le texte du contrat. D'où l'identité de Cassini et les périodes de Pisano au lieu de termes recalculés.
  - Les seams déjà convenues ont gardé tous les cas sur la bibliothèque ou la CLI.
  - Rédiger le catalogue a fait surgir des cas encore non couverts : la casse de `--algo FAST`, `-help` avec un seul tiret, `--list=5`, `--algo 5`, `--time abc`, la répétition de `--algo` (visible seulement avec `--time`), la locale et le forçage des couleurs. Il a aussi révélé une ambiguïté de la grammaire (`--algo --list 5`), tranchée dans le PRD 1.3.
- **Friction** : la demande d'un catalogue exhaustif écrit d'avance heurte l'anti-pattern du découpage horizontal. Je l'ai résolu en traitant le catalogue comme un carnet de cycles ordonnés, chaque test n'étant écrit qu'à son cycle. Pour Go, Rust et Python, en revanche, le test de contrat existe d'avance : c'est une boucle externe d'acceptation, choisie en connaissance de cause.
- **À changer** : rédiger le catalogue en même temps que le PRD. Ces trous seraient apparus pendant le grilling.

## 2026-10-10 — Implémentation par `fib-dev` (`tdd`, étape 6)

- **A aidé** :
  - Le catalogue et le contrat commun ont fait converger les quatre implémentations : 54 cas verts dans chacune, avec le harnais du ticket #2, réutilisé tel quel par Go, Rust et Python.
  - Les agents ont trouvé de vrais trous :
    - la répétition de `--algo` avec une valeur invalide (cas CLI-A16);
    - la discordance combinée à `--time`;
    - le `$` de Python, qui accepte un saut de ligne final (cas CLI-I16);
    - des indices négatifs qui font boucler `fast` en Go et `matrix` en Python.
  - `worktree.baseRef: "head"` fonctionne : les worktrees sont partis du `main` local fusionné.
- **Friction** :
  - Le téléchargement de l'oracle a été refusé par les permissions, même après accord dans le chat. Aucun ticket n'a donc pu être vérifié en entier, et les cycles qui dépendaient de l'oracle ont été guidés par des sondes jetables.
  - Les agents ne peuvent pas supprimer de fichiers : des sondes sont restées dans les worktrees Go et Rust.
  - Un agent a été interrompu, puis repris à partir de son travail partiel.
  - Le coût : environ un million de jetons et 40 minutes pour les trois langages en parallèle.
- **À changer** : obtenir les données externes (l'oracle) avant de lancer les agents, idéalement à l'étape de la spec, et dire d'emblée aux agents de garder leurs sondes hors du dépôt. C'est maintenant une règle de `fib-dev`.

## 2026-10-10 — Oracle généré et vérification finale

- **A aidé** :
  - La vérification ORA par la seule définition rend l'oracle contrôlable quelle que soit sa provenance. Une génération par la formule de Binet, indépendante des trois algorithmes, a donc suffi à débloquer le projet.
  - Avec l'oracle, les quatre implémentations sont passées au vert dès le premier lancement de `npm run check`. Les sondes jetables des agents avaient fait leur travail, et l'oracle réel n'a révélé aucun écart.
- **Friction** : exiger une copie conforme de l'OEIS était plus fort que nécessaire. Quand le téléchargement a été refusé, cette contrainte a bloqué les quatre tickets.
- **À changer** : exiger de l'oracle une propriété vérifiable (la définition de la suite) plutôt qu'une provenance.

## 2026-10-11 — Audit en workflow, refactorisation, puis `/code-review` (étape 8)

- **A aidé** :
  - L'audit a mis huit auditeurs en lecture seule : un par langage, le contrat, la cohérence entre langages, un test différentiel et la documentation. Un vérificateur sceptique suivait chacun d'eux. Sur 233 constats, 36 ont été réfutés. L'audit n'a trouvé aucune erreur de calcul et un seul écart au contrat : Python laissait `PYTHONIOENCODING` changer l'encodage (F10). Il est corrigé, avec le cas CLI-V03.
  - Le test différentiel a joué 121 vecteurs hors catalogue sur les quatre CLI, sans un seul écart de stdout ni de code de sortie.
  - Durcir le contrat avant de refactoriser a protégé la réécriture des messages d'erreur (CLI-A17, A18, H11, U04).
  - `/code-review`, sur ses deux axes, a trouvé ce que l'audit et les `fib-dev` avaient laissé passer :
    - le lecteur d'oracle Rust n'ignorait plus les lignes vides (PRD § 7);
    - un commentaire Python était faux;
    - des commentaires dépassaient 200 caractères.
- **Friction** :
  - L'audit a proposé 38 changements, et la synthèse mêlait l'essentiel et le cosmétique. Le tri s'est fait à la main; trois items et une partie des docs sont reportés.
  - Les briefs imposaient des phrases mot pour mot, ce qui a fait grossir le code de 15 à 33 lignes par langage, surtout en commentaires. Go et Rust dépassent la cible d'A3 (ADR-0003 amendé).
  - Les tickets #2 à #5 avaient été fermés sans `/code-review`, que prévoit pourtant l'étape 4 de la boucle de `CLAUDE.md`. Ce travail-ci n'avait pas de ticket : les briefs en tenaient lieu.
  - Coût : environ 2,5 millions de jetons pour l'audit, et 1,25 million pour la refactorisation, la revue et les corrections.
- **À changer** :
  - Demander à la synthèse un plan court, classé par gain de lecture.
  - Fixer un plafond de lignes dans les briefs de refactorisation.
  - Lancer `/code-review` à chaque ticket.

## 2026-10-11 — Optimisation des performances (hors skill), puis `/code-review`

- **A aidé** :
  - Le banc d'essai commun, mesuré avant tout changement, a montré où partait le temps : le démarrage (TypeScript, Python) et la conversion décimale de la liste. Le calcul ne dominait que pour `iterative`. Sans lui, on aurait optimisé `fast`, qui prenait moins d'une milliseconde.
  - Deux décisions sont revenues au propriétaire, chiffres à l'appui : la liste calculée en décimal, et les commandes de lancement.
  - `/code-review` a trouvé que `matrix` en TypeScript refaisait pas à pas l'ancien `fast` : l'algorithme avait perdu son identité.
  - Gains, en temps total : liste 10 000 de ×2,9 à ×4,3, terme seul de −34 % à −45 % en TypeScript et en Python, iterative ×3,8 en Go.
- **Friction** :
  - Quatre agents qui mesurent en même temps se gênent. Leurs chiffres sont bruités, et une mesure seule après la fusion fait foi.
  - Un sous-agent a écrasé le banc d'essai de l'orchestrateur, qui se trouvait dans un dossier temporaire partagé. Il vit maintenant dans le dépôt (`npm run bench`).
  - Le code grossit nettement (TypeScript 232, Python 228, Rust 359, Go 376).
  - Le sandbox refuse la suppression des dossiers de worktrees.
- **À changer** :
  - Fixer dès le brief le nom des tests de garde.
  - Lancer les mesures en série, pas en parallèle.

## 2026-10-11 — Borne d'un terme à 1 000 000, puis `/code-review`

- **A aidé** :
  - Les faits attendus (208 988 chiffres, et les congruences mod 2048 et mod 1000 tirées des périodes de Pisano) ont été vérifiés avant d'écrire le cas CLI-T13. Le contrat est passé au rouge, puis les quatre langages au vert.
  - Le banc d'essai a dit où optimiser, et où ne pas le faire. Go, Rust et TypeScript convertissent déjà en temps sous-quadratique, et aucune piste maison n'y a passé le seuil des 3 %. En Python, deux seuils mesurés font passer F(1 000 000) de 288 à 38 ms.
  - `/code-review` : l'axe Spec a joué 152 sondes, sans un seul écart, jusqu'à F(1 000 000) par `iterative`. L'axe Standards a vu que le chemin `Decimal` contournait la seam, et que deux gardes avaient un attendu tautologique ou pris d'un autre algorithme.
- **Friction** :
  - Un brief a supposé que `num-bigint` convertissait en temps quadratique, ce qui était faux. L'agent l'a vérifié dans la source avant d'écrire du code.
  - Les gains restants passent tous par le parallélisme, que le PRD § 8 exclut.
- **À changer** : mesurer la bibliothèque à la nouvelle taille avant de rédiger les pistes du brief.

## 2026-10-11 — Exécution et épreuve des cas de test, en workflow `ultracode` (hors skill)

- **Résultats** :
  - `npm run check:<id>` est vert dans les quatre langages. Le contrat passe 96 sur 96 partout (ORA-01 à 06 et les 90 cas CLI, tous joués). Tests unitaires : Go 11 (6 028 sous-tests), Rust 20, TypeScript 28, Python 30. BIB-01 à 14 sont portés partout, sauf BIB-14 en Rust, exclu comme prévu.
  - L'oracle a été recalculé par addition, sans la formule de Binet du générateur. Ses 52 vérifications concordent : la table à l'octet, les valeurs et restes du § 2, les périodes de Pisano, les nombres de chiffres et Cassini.
  - Différentiel : `100000`, `1000000`, `--list 10000`, `--algo all 1000000` et `--algo iterative 1000000` ont la même empreinte SHA-256 dans les quatre langages. Cela couvre les chiffres du milieu, que T12, T13 et L06 ne vérifient pas.
  - Mutation, avec 16 mutants par langage dans des worktrees jetables : 57 tués sur 64. Le code d'origine est correct partout; deux survivants montrent des cas absents du contrat :
    - en Go, un mutant qui rend `--algo` insensible à la casse passe, parce que la bibliothèque refuse encore `FAST`. `--algo FAST --algo fast 5`, par exemple, le distingue;
    - en TypeScript, un mutant qui ne cite en ASCII que le premier caractère non ASCII passe. `--bögüs 5` le tuerait.

    Vérifié sur les quatre CLI d'origine : `--algo FAST --algo fast 5` sort en erreur, et `--bögüs 5` est cité en ASCII. Les autres survivants : l'écriture tamponnée, sans garde en Go comme en Rust; la retenue Go à exactement 10^18, inatteignable par la CLI, et testable seulement une fois l'additionneur extrait de `FibList` (fait par #8 pour PERF-GO-5, pendant de PERF-RS-4); un limbe de marge inutile dans le tampon TypeScript; le mutant M-py-5, qui fait accepter `bool` par la bibliothèque Python. Le code d'origine le refuse (`TypeError`), mais BIB-14 ne nommait pas `True` : le catalogue n'arbitrait pas ce cas, ajouté par #14. (Corrigé le 2026-10-12 : cette ligne prêtait au vrai code le comportement du mutant.)
  - Fidélité au catalogue : sur 100 constats bruts, 61 ont été maintenus par au moins deux vérificateurs sur trois (1 majeur, 35 mineurs, 25 cosmétiques).
    - Le majeur, rejoué en mutant : PERF-PY-14 ne garde pas le passage à `str()` au-delà de 200 000 bits. Sans cette branche, les 30 tests Python restent verts.
    - Mineurs récurrents : des tests PERF qui ne gardent pas ce qu'ils annoncent (écriture par tranches, absence d'allocation en Rust, seuils Python lus dans les constantes, démarrage Go), des balayages BIB-05 à 07 non paramétrés, une notation des espaces ambiguë (I15, U02), une traçabilité périmée (T12 encore sous F8), et V02 et V03 neutralisés par `python -I`.
  - Banc d'essai sur 15 exécutions, machine au repos : les médianes tombent à 1 ms près des valeurs finales du README, donc aucune régression.
- **A aidé** :
  - La mutation mesure le pouvoir des tests, ce qu'un « tout vert » ne dit pas : elle a trouvé deux cas qui manquent au contrat, malgré 384 cas verts.
  - La vérification adverse a écarté 39 constats sur 100.
  - Le critique de complétude a relevé qu'une douzaine de constats maintenus n'étaient que des déductions. Le majeur a été exécuté avant d'être consigné.
- **Friction** :
  - Coût : 26 millions de jetons pour l'épreuve (327 agents, dont 300 vérificateurs), contre 0,35 million pour la simple exécution des quatre suites.
  - Un vérificateur maintient une déduction aussi volontiers qu'un fait exécuté. Le vote ne remplace pas l'exécution.
  - Les agents actifs faussent le banc d'essai : un premier passage dépassait le README de 5 à 20 %.
  - Pour quatre survivants sur sept, les deux juges ont rendu des classes différentes, ou une classe que le critique conteste.
  - L'orchestrateur a d'abord décrit le comportement de deux mutants comme celui du vrai code, et l'a consigné ainsi. Il a corrigé après exécution sur les CLI d'origine.
- **À changer** :
  - Trancher par l'exécution, mutant ou commande, plutôt que par le vote, et réserver les trois voix aux constats majeurs.
  - Lancer le banc seul, sur une machine au repos.
  - Décider des suites : les cas `--algo FAST --algo fast 5` et `--bögüs 5`; des gardes pour PERF-PY-14, pour l'écriture tamponnée et pour l'absence d'allocation en Rust; la notation des espaces et la traçabilité du catalogue.

## 2026-10-12 — Tickets #6 à #10 et #12 à #14 par `fib-dev`, puis `/code-review` (session infonuagique)

- **Contexte** : la session locale avait pris #6 à #10, puis s'était arrêtée parce que OneDrive avait passé des fichiers de `.git` en « nuage seulement ». Rien n'était poussé. Une session infonuagique (Linux, 4 cœurs) a tout repris depuis `bf684e5`. La branche `claude/determined-fermi-8r5kmg` a tenu lieu de `main` pour l'intégration, avec la PR agbruneau/Fib#15 vers `main`.
- **Résultats** :
  - Huit `fib-dev` ont livré les huit tickets en trois vagues, selon les bloqueurs : #6 à #10 en parallèle, puis #12 et #14, puis #13. Trois autres ont appliqué les corrections de revue. Les onze fusions se sont faites sans conflit.
  - `npm run check` est vert sur les quatre langages, et `npm run contract` passe 398 sur 398 (6 ORA + 4 × 98 cas CLI). Chaque garde nouvelle a été vue rouge sur le mutant de son ticket, puis verte sur le code d'origine.
  - Banc A/B au repos, binaires de `bf684e5` et de la branche lancés en alternance, 61 exécutions chacun. Les écarts restent sous le seuil de 3 % de #8 et #9 : Go `--list 10000` à +0,0 % au total (+1,5 % au calcul), Rust `--list 10000` à −1,0 %, Rust `iterative 100000` à +0,5 %.
- **A aidé** :
  - Les `fib-dev` ont corrigé ou nuancé quatre prototypes des tickets, en les exécutant plutôt qu'en les recopiant :
    - PERF-PY-21, tel que prototypé, laissait survivre la fuite d'`Emax`. PERF-PY-10 passe avant lui dans l'ordre alphabétique et laissait déjà `Emax` modifié. La garde tourne maintenant dans un contexte neuf.
    - PERF-TS-1 ne garde que la fin paire de la boucle de `fibList`; PERF-TS-5 garde la fin impaire.
    - `Math.round` à la place de `Math.ceil` est un mutant équivalent tant que le limbe d'assurance reste.
    - Sous M-py-5, quatre des sous-tests rouges sont dans BIB-14 et deux dans PERF-PY-16, qui partagent `REFUSED`.
  - `/code-review` a trouvé quatre défauts que les agents n'avaient pas vus :
    - CLI-U05 et CLI-A20, écrits par #6 en parallèle de #12, promettaient la citation que #12 retirait de U04 et A18;
    - V02 vidait `NO_COLOR` au lieu de le retirer, alors qu'un `NO_COLOR` vide fait avertir Node 22 sur stderr;
    - PERF-RS-8 tirait sa borne de la constante qu'il garde;
    - PERF-GO-7 rougissait sous `go test -cover`.
- **Friction** :
  - La skill `tdd` (plugin `mattpocock-skills`) n'est pas installée dans la session infonuagique : chaque `fib-dev` a suivi son cycle à la main.
  - `gh issue view` échoue, parce que GraphQL est refusé; `gh api repos/…/issues/<n>` passe.
  - Les références du banc de #8 et #9 venaient de la machine Windows du propriétaire : la comparaison ne vaut que sur une même machine, d'où l'A/B en alternance.
  - Deux `fib-dev` ont écrit dans le même dossier temporaire, et l'un a écrasé un fichier de l'autre.
  - `/code-review bf684e5` a revu le seul commit `bf684e5`, et non la plage voulue. Ce passage a quand même trouvé trois erreurs du journal du 2026-10-11, corrigées ci-dessus.
  - Huit constats de la revue sont écartés :
    - six contestent un choix que le ticket imposait : la marge nulle des allocations Go, PERF-PY-18 qui fige les seuils, les 6 003 sous-tests TypeScript, les trois passes de T13, l'environnement sans `PYTHON*` de PERF-PY-20, les commentaires de deux lignes;
    - deux reposent sur un environnement improbable ici : `go` absent du PATH, `PYTHONHOME` requis.
- **À changer** :
  - Donner à chaque `fib-dev` un sous-dossier temporaire à lui.
  - Passer à `/code-review` une cible explicite (la branche), jamais un SHA seul.
  - Installer `mattpocock-skills` dans l'environnement infonuagique, ou dire d'emblée dans le prompt que `tdd` manque.
  - Quand des tickets parallèles touchent la même règle, comme #6 et #12 pour les citations, donner la règle aux deux.

## 2026-10-12 — `diagnosing-bugs` (étape 9, #16)

- **Contexte** : aucun écart entre langages n'était connu. Le différentiel de 121 vecteurs du 2026-10-11 n'en avait trouvé aucun, et l'écart `PYTHONIOENCODING` était déjà corrigé. La skill part d'un bug signalé ; il a donc fallu en trouver un d'abord. La skill n'est pas installée dans la session infonuagique : son `SKILL.md` a été lu dans `mattpocock/skills` et suivi à la main.
- **Recherche** : un workflow de cinq chercheurs a construit chacun une boucle différentielle, le n° 9 de la phase 1 de la skill. Les dimensions sont la grammaire, les valeurs, les durées, l'environnement et les arguments extrêmes, soit environ 170 000 entrées sur les quatre CLI. Un vérificateur sceptique par dimension a ensuite reproduit, minimisé et classé chaque candidat face au PRD § 4.
  - La grammaire (83 319 suites, dont toutes celles de 1 à 3 jetons sur 41) et les valeurs (14 793 entrées, autour de chaque seuil d'optimisation) n'ont donné aucun candidat.
  - L'écart retenu a été trouvé trois fois, de façon indépendante : avec le fd 2 fermé, Python sort en code 1, stdout vide.
  - D'autres comportements divergent là où le contrat ne tranchait pas. Ils sont listés plus bas, et #18 les a écrits hors contrat.
- **Phases** :
  1. La boucle de rétroaction est un script Python qui lance les quatre CLI avec `os.close(2)`. Elle rougit sur 6 entrées, Python seul, en 0,46 s, de façon déterministe sur 4 passes.
  2. Minimisation : `python python/fib.py 2>&-` suffit. Ni l'argument, ni `-I -S` ne sont nécessaires.
  3. Hypothèses : 5, classées.
  4. Sondes : une par prédiction. H1 est confirmée : avec le fd 2 fermé, CPython met `sys.stderr` à `None`, et `reconfigure` plante avant `main`. Les quatre autres hypothèses sont réfutées. Un prototype jetable a montré que garder le seul `reconfigure` ne suffisait pas : les écritures de `main` tombaient ensuite sur le même `None`.
  5. Correctif et non-régression, test d'abord, par un `fib-dev`. CLI-V04 a été vu rouge pour Python seul, puis vert partout. Le correctif donne un puits `/dev/null` à un flux `None`, avec un `import os` paresseux. Le banc A/B reste à +0,2 à +0,7 % sur `petit (10)`.
  6. Nettoyage : le prototype est supprimé, et le dépôt ne garde aucune instrumentation.
- **`/code-review`** : 9 constats, 7 retenus.
  - V04 ne prouvait pas que le fd 2 était fermé pour la CLI elle-même : un mutant du harnais le laissait vert.
  - Rien ne gardait le correctif sous Windows, où V04 se saute : ajout de PERF-PY-23, portable.
  - L'`import os` paresseux n'avait pas de garde : ajout de PERF-PY-24. Celle-ci compare au démarrage nu de l'interpréteur, pour ne pas rougir à tort sur une plateforme qui charge `os` au démarrage.
  - Le commentaire du correctif était faux pour le fd 1.
  - Le PRD § 4 était muet sur un stderr fermé.
  - Retouches mineures : le contrôle du harnais tournait quatre fois, et les nouveaux cas n'étaient pas comptés au README.
  
  Écarté : un stderr ouvert mais qui refuse l'écriture, qui relève des comportements à arbitrer ci-dessous.
- **A aidé** :
  - La boucle différentielle a trouvé un vrai écart là où 121 vecteurs et 398 cas verts n'en voyaient aucun. Elle a pris la dimension que personne n'avait jouée : l'état des descripteurs au lancement.
  - Le vérificateur sceptique a ramené 15 candidats à 4 bugs et 10 lacunes, chacun avec sa commande minimale.
  - La phase 4 a évité un correctif à moitié : le seul garde de `reconfigure` aurait laissé `--time` et les erreurs d'usage en code 1.
- **Friction** :
  - La skill suppose un bug signalé. Sans écart connu, la recherche a coûté plus que le diagnostic : 1,1 million de jetons et 88 minutes pour le workflow, contre quelques minutes pour les phases 1 à 4.
  - Le contrat ne sait pas fermer un descripteur sous Windows : V04 s'y saute, et une garde Python prend le relais.
  - `/code-review` a dû rattraper deux trous dans un test écrit test d'abord : un harnais qui pouvait passer à vide, et une plateforme non couverte.
- **À arbitrer** : des cas que le contrat ne tranchait pas, où les CLI divergent, tous reproduits. Le propriétaire les a déclarés hors contrat le 2026-10-12, et #18 les a écrits au PRD § 4 et au catalogue § 6 :
  - stdout en tube non bloquant : Rust tronque en silence avec le code 0, Python plante, Go et TS réessaient.
  - stdout ou stderr qui refuse l'écriture (`/dev/full`, `ulimit -f`, tube sans lecteur) : quatre codes différents, et un succès annoncé à tort par Go et Rust (`/dev/full`) ou par Go et TS (`ulimit -f`).
  - Traces de plantage de TS et de Python : elles contiennent des séquences ANSI quand `FORCE_COLOR=1`.
  - `2>&1` sur un tube : TS place la ligne de durée au milieu d'un terme dès 64 Kio de sortie.
  - `ulimit -v` : Go sort en code 2, le code d'une erreur d'usage.
- **À changer** :
  - Garder la boucle différentielle comme outil de recherche, avant `diagnosing-bugs`, quand aucun bug n'est signalé.
  - Faire écrire au `fib-dev`, dans tout test qui manipule le harnais, le mutant du harnais qui doit le faire rougir.

## 2026-10-13 — `/improve-codebase-architecture` et `codebase-design` (étape 10, #20)

- **Contexte** :
  - Les deux skills manquent à la session infonuagique : leurs `SKILL.md` (et `HTML-REPORT.md`, `DEEPENING.md`, `DESIGN-IT-TWICE.md`, `grilling`) ont été lus dans `mattpocock/skills` et suivis à la main.
  - Le dépôt n'a pas de `GLOSSARY.md` : `CONTEXT.md` en tient lieu.
  - Les points chauds de l'historique sont `contracts/cli.test.ts`, Python et `rust/src/main.rs`.
- **Exploration** : un workflow de six explorateurs, un par zone (contrat, Python, Go, Rust, TypeScript, transverse), puis six vérificateurs sceptiques. Chaque vérificateur applique le test de suppression, confronte le candidat aux ADR et relit les lignes citées. Résultat :
  - 19 candidats, dont 8 gardés et fondus en 5 :
    - A, le lancement d'une implémentation en un seul module de `contracts/` (Strong) ;
    - B, la CLI TypeScript appelable dans le processus (Strong) ;
    - C, un seam pour tester la discordance dans chaque langage (Worth exploring) ;
    - D, le texte décimal de Python (Worth exploring) ;
    - E, le point d'entrée de Python (Speculative).
  - Le rapport HTML (Tailwind, Mermaid) est écrit hors du dépôt, son rendu vérifié par un navigateur sans tête, puis envoyé au propriétaire.
- ***Design it twice*** : le propriétaire a retenu B. Quatre conceptions ont été prototypées et mesurées hors du dépôt : minimale, flexible, appelant courant, ports et adaptateurs.
  - Les quatre tuent M1 à M4. Le démarrage ne bouge qu'au-delà d'un module interne, `import.meta`, pour environ 0,2 ms.
  - La conception minimale a été retenue : `run(args, io?)` rend le code, avec un `Io` à quatre membres.
- ***Grilling*** : deux tours, sept décisions.
  - La conception 1.
  - Le statut des tests de garde plutôt qu'un troisième seam.
  - Le `fib` discordant, réservé au test de F3.
  - Node 24.2 avec `import.meta.main`.
  - Les noms PERF-TS-6 à 8, avec le glossaire élargi.
  - TypeScript seulement.
  - `performance.now()` gardé, et la piste `process.hrtime` versée à #21.
- **Livraison (#20)** :
  - un `fib-dev`, test d'abord. M1 à M4 font rougir PERF-TS-6 à 8, alors que le contrat reste à 106 sur 106 sous chacun d'eux ;
  - le seul module ajouté au démarrage est `initialize_import_meta` ;
  - banc apparié au repos sur `petit (10)` : +0,4 % à +2,9 % (+0,16 à +1,29 ms), sous le seuil de 3 % ;
  - `/code-review` : 7 constats, tous traités (voir la PR).
- **A aidé** :
  - Le vérificateur sceptique a rabattu 6 forces sur 8, corrigé des citations fausses et écarté 11 candidats que la suppression ne concentrait pas.
  - Les prototypes mesurés ont tranché les questions au lieu de les débattre : le coût de `import.meta` contre celui d'un second fichier, les mutants vus rouges sur chaque conception.
  - L'exploration a sorti deux constats hors approfondissement :
    - une garde manque au préchargement de `decimal` en Python : le mutant reste vert, pour 5,5 → 9,4 ms ;
    - `performance` coûte 11 modules internes au démarrage de TypeScript, ce qui a donné #21.
- **Friction** :
  - Les skills ne sont pas installées. Le rapport « ouvert dans le navigateur » ne s'applique pas à une session infonuagique : il a été envoyé comme fichier.
  - Le vocabulaire imposé par `codebase-design` (interface, adaptateur, port) heurte les « _Avoid_ » de `CONTEXT.md` (« Contrat : _Avoid_ interface », « Implémentation : _Avoid_ port »). Le rapport a dû tenir deux lexiques à la fois.
  - La skill s'arrête au *grilling* ; la livraison (ticket, `fib-dev`, revue) vient de la boucle du dépôt.
  - Coût : environ 1,8 million de jetons pour l'exploration et 0,6 million pour les quatre prototypes.
- **À changer** :
  - Distinguer, dans `CONTEXT.md`, l'interface au sens de `codebase-design` du contrat, pour que les deux vocabulaires cohabitent.
  - Garder les candidats A, C et D comme pistes d'architecture, et décider de la garde manquante du préchargement de `decimal`.

## 2026-10-13 — `/handoff` (étape 11)

- **Contexte** : la skill, lue dans `mattpocock/skills` faute de plugin, porte `disable-model-invocation: true`. Elle se lance donc par le propriétaire, ici par sa consigne « passe à l'étape 11 avec /handoff après #20 ». Elle demande quatre choses :
  - un document hors du dépôt, dans le dossier temporaire du système ;
  - une section de skills suggérées ;
  - des renvois aux artefacts au lieu de redites ;
  - le caviardage des données sensibles.
  
  Sans argument, le document vise la suite du dépôt, #21 en tête.
- **Produit** : un document de reprise d'environ 900 mots, envoyé au propriétaire. Il renvoie au README, au PRD § 9, au journal, aux tickets et aux PR. En propre, il ne garde que ce qu'aucun artefact versionné ne contient :
  - les pistes A, C, D et E du rapport de l'étape 10, et ses petits constats hors approfondissement ;
  - les pièges de l'environnement infonuagique ;
  - l'étendue de l'autorisation de fusion.
- **A aidé** :
  - La règle « renvoyer, ne pas redire » a forcé un inventaire. Il a montré ce qui ne vivait que dans la session : les pistes d'un rapport resté hors dépôt, et le savoir opératoire (GraphQL refusé, SHA complet pour fusionner, `/code-review` sur une plage et non un SHA).
  - Relire chaque renvoi avant l'envoi a corrigé deux constats périmés :
    - `stderrClosed` était déjà nommé dans `run` ; seul un appel passe encore `{}, true` ;
    - #21 prévoit une fermeture en `wontfix`.
- **Friction** :
  - « Hors du dépôt, dans le dossier temporaire » se heurte à la session infonuagique : `/tmp` disparaît avec le conteneur. Le document ne survit que parce qu'il a été envoyé au propriétaire, et une session future ne le trouvera pas sans lui.
  - Le PRD § 9 nomme le document de reprise comme livrable de l'étape, mais la skill l'exclut du dépôt. Le livrable n'est donc pas versionné ; seule sa trace l'est, ici.
  - Le document est un instantané. Rédigé juste avant #21, il est déjà en partie dépassé quand la même session traite #21 ; il le dit en ouverture.
- **À changer** :
  - En session infonuagique, consigner les pistes orphelines dans un artefact durable (ticket de suivi ou page publiée) dès l'étape qui les produit, pour que la reprise n'en dépende pas.
  - Faire de la section « Pièges de l'environnement » une page de `docs/agents/`, puisqu'elle vaut pour toute session.

## 2026-10-13 — Horloge `process.hrtime` en TypeScript (#21), puis `/code-review`

- **Contexte** : #21 est né de l'étape 10. Le premier accès au global `performance` charge des modules internes, et `timed` lit l'horloge même sans `--time`. Le gain annoncé, environ 2,5 ms, avait été mesuré sur une machine partagée.
- **Mesure** : `npm run bench -- ts --runs 15`, trois tours au repos, sur `petit (10)`.
  - `main` donne de 46,3 à 55,8 ms selon le tour, et la branche de 44,9 à 47,8 ms. Les plages se recouvrent : le banc non apparié ne tranche pas un gain de 2 ms sur 50.
  - L'A/B apparié tranche : 6 tours de 201 lancements de `cli.js 10` et `--time 10` donnent de −1,8 à −2,6 ms (−4,0 à −5,7 %), tous dans le même sens. Le changement est retenu.
- **`/code-review`** : 13 constats, 9 retenus en tout ou en partie.
  - L'unité de l'horloge réelle n'était gardée par rien. `performance.now()` rendait des millisecondes nativement ; la conversion des nanosecondes est un risque créé par le changement lui-même. Un test borne maintenant une vraie durée entre le vingtième du temps mural et ce temps.
  - La garde interdisait tout `internal/perf/…` sauf trois exceptions mesurées sous un seul Node et un seul système. Elle n'interdit plus que les deux modules que seul le global charge.
  - La garde compilait la CLI à chaque `node --test`. Elle lance maintenant la source, dont Node retire les types.
  - Le décompte du commentaire était faux : 11 modules avec stdout vers un fichier ou un terminal, 9 dans un tube, où `net` en charge 2 de toute façon. L'orchestrateur a départagé l'auteur et le réviseur par une mesure.
  - Retenus aussi : `--list` et `--algo all` couverts ; `NODE_OPTIONS` retiré de l'environnement des sous-processus, avec un délai maximal ; la mesure en ms dans le commentaire d'une ligne ; le catalogue § 6.
  - Écartés : sauter le chronomètre sans `--time` (décision de #20), retirer l'origine de l'horloge (plus exacte pour un coût égal), et couvrir l'aide et les erreurs, qui ne lisent pas l'horloge.
  - Corrigé par l'orchestrateur : le compte de lignes TypeScript du README, périmé depuis #20 (234 → 266).
- **A aidé** :
  - L'A/B apparié, avec les deux builds côte à côte, a rendu mesurable un gain que le banc du dépôt noyait dans le bruit.
  - La revue a vu que le changement retirait une propriété gratuite, l'unité native de `performance.now()`, sans la remplacer par une garde.
- **Friction** :
  - La première garde codait l'état d'une plateforme, une liste d'exceptions, au lieu de la propriété voulue, l'absence de `performance`.
  - La skill `tdd` reste introuvable chez le `fib-dev` (`Unknown skill: tdd`), qui suit son cycle à la main.
- **À changer** :
  - Ajouter au banc un mode apparié entre deux builds, puisqu'un gain de démarrage de quelques pour cent est sous son bruit.
  - Demander au `fib-dev`, pour toute garde de modules chargés, un prédicat positif sur ce que l'optimisation évite.

## 2026-10-13 — Garde PERF-PY-25 du préchargement de `decimal` (#25), en workflow, puis `/code-review`

- **Contexte** : l'étape 10 avait relevé qu'une optimisation de l'ADR-0004 n'avait pas de garde. `compute` importe `decimal` hors du chronomètre, et sans cet import tous les tests restaient verts. Le propriétaire a demandé la garde ; l'orchestrateur a écrit le ticket.
- **Démarche** : un workflow `ultracode`.
  - Un `fib-dev` écrit la garde, puis trois vérificateurs adverses, chacun dans son worktree, l'éprouvent par exécution : mutants, robustesse et spec.
  - Deux tours de correction suivent.
  - Le premier lancement s'est arrêté au bout d'une minute, quand un message du propriétaire a interrompu le tour en cours. Le même script a été relancé.
- **Résultat** : un interpréteur neuf espionne `time.perf_counter`. Avant chaque chronomètre, `decimal` doit être chargé, et `sys.modules` ne doit pas grandir dans la fenêtre. Sans préchargement, la ligne `--time` de `fast` à 170 000 passe de 5,8 à 10,4 ms (`python -I -S`, médiane de 21 lancements).
  - Tour 1 : 3 défauts. La garde ne voyait ni un module paresseux (`LazyLoader`), ni une condition juste en un seul point, et ne passait jamais `--time`.
  - Tour 2 : les points testés étaient tous pairs.
  - Tour 3 : le vérificateur trouve encore une condition en résidu ou en bande qui survit entre les points. C'est une limite de tout test par échantillons ; l'orchestrateur l'a écartée.
- **`/code-review`** : 10 constats, tous retenus en tout ou en partie.
  - Le plus sérieux : l'autre côté de la condition n'était pas gardé. Un préchargement trop large (`n > 100_000`, ou `iterative` non exclu) passait les 39 tests et coûtait environ 5 ms de démarrage à des entrées qui n'utilisent pas `decimal`. Deux vérificateurs l'avaient vu, mais classé « mineur, hors mandat ».
  - La docstring avait gonflé à 23 lignes, chaque tour y empilant ses mutants ; elle revient à 8.
  - Corrigés aussi :
    - la réserve sur l'accélérateur C `_decimal` ;
    - un message d'échec qui donne le rapport brut ;
    - un lanceur partagé avec PERF-PY-8 ;
    - un témoin à trois fenêtres ;
    - le code mort `env`.
  - Au README, PERF-PY-20 et 23 sont désormais présentées comme gardes de correctifs.
  - Au total, 23 mutants sont rouges sous Python 3.12 et 3.13.
- **A aidé** :
  - La vérification par exécution a trouvé ce qu'une relecture laisse passer : un module paresseux qui trompe un `in sys.modules`.
  - Le témoin prouve chaque moitié du prédicat, et pas seulement que la garde tourne.
- **Friction** :
  - Le vérificateur de mutants ne converge pas sur une garde par échantillons : chaque tour invente une condition plus tordue. Il a fallu trancher.
  - Le workflow a coûté environ 1,06 million de jetons et 59 minutes, puis une revue et un tour de corrections.
  - Un message du propriétaire pendant un workflow en arrière-plan l'a interrompu, sans avertissement visible.
- **À changer** :
  - Donner au vérificateur de mutants un critère d'arrêt : un mutant compte s'il est une modification plausible du code, pas une condition arbitraire.
  - Demander dès le ticket les deux côtés d'une condition d'optimisation : là où elle doit agir, et là où elle ne doit rien coûter.

## 2026-10-13 — `codebase-design` et `grilling` (pistes A, C, D et E de l'étape 10, #27 à #36), livraison en trois vagues, puis `/code-review`

- **Contexte** : le rapport de l'étape 10 gardait quatre pistes en plus de B, livré par #20. Le propriétaire a demandé de traiter les quatre : A (lancement commun dans `contracts/`), C (discordance testée en Go, Rust et Python), D (texte décimal de Python) et E (entrée Python appelable sans processus).
- **Conception** : un workflow `ultracode` de 11 agents (environ 3 millions de jetons, 2 h 23).
  - Chaque piste a eu deux concepteurs indépendants (un seul pour E), qui prototypaient chacun dans un worktree. Ils ne mesuraient que les modules chargés et les instructions, jamais de temps mural.
  - Un critique adverse par piste a ensuite tout rejoué par exécution. Il a dit si la piste valait son coût, rédigé les décisions du propriétaire et écrit les tickets.
  - Deux critiques ont conclu que la forme du rapport ne valait pas son coût :
    - **D** : ni un module privé ni une fonction regroupée ne tuent un mutant de plus qu'une simple garde. Le module coûte une ligne `sys.path` (`-I` implique `-P`), 17 gardes à recibler et +0,55 M d'instructions par lancement pour la fonction.
    - **E** : une entrée appelable ferait perdre la garde du vrai bloc `__main__` sous Windows. Le seul vrai trou est la fin de ligne de F10, dont le mutant restait vert sous Linux.
  - Les critiques ont aussi réfuté des affirmations du rapport, chacune par mutant ou par mesure :
    - « sept noms » connus de `compute` : il y en a quatre ;
    - « le code 3 n'est observé nulle part » : faux depuis PERF-TS-7 ;
    - « deux lignes par langage » : il en faut 6 ou 7 en production, et 50 à 80 de gardes ;
    - « PERF-PY-22 en mémoire » : elle l'est déjà.
  - Pour C, la critique a montré qu'une relance du binaire de test tuerait aussi G9 et G10 en Go (une écriture directe sur `os.Stderr` ou `os.Stdout` depuis `compute`), ce que la variable de paquet ne faisait pas, mais au prix d'un mode caché du binaire de test : le paramètre retenu les laisse à la revue. Elle a aussi trouvé un trou commun aux quatre langages : une durée glissée sur la ligne du message de discordance (T2 en TypeScript, #32). Une assertion qui refuse ` ms` dans ce message le ferme dans PERF-TS-7, PERF-GO-9, PERF-RS-11 et PERF-PY-26.
- **Décisions** : une seule série de quatre questions, une par piste, avec les options et la recommandation de chaque critique. Le propriétaire a suivi les quatre recommandations :
  - A en trois tickets ;
  - C dans l'idiome de chaque langage, sur le modèle d'`Io.fib` en TypeScript : un paramètre `termFunc` en Go, un paramètre générique `impl Fn` en Rust, une substitution dans `_ALGOS` par `mock.patch.dict` en Python, avec les espions de l'algorithme exécuté ;
  - D réduite à une garde ;
  - E réduite à un ticket de test.
  
  Dix tickets en sont sortis, #27 à #36. Le lanceur Python minimal (−22,5 % d'instructions) n'est pas devenu un ticket : seul un banc au repos sous Windows peut le trancher.
- **Livraison** : trois vagues. Dans chacune, un workflow lance un `fib-dev` par ticket dans son worktree, puis un vérificateur adverse par exécution, limité aux mutants plausibles (leçon de #25), puis un tour de correction. Viennent ensuite la fusion, la vérification, `/code-review` et une PR par vague.
  - Vague 1 (#27, #30 à #33, PR #37) : 11 agents, 1,46 million de jetons, 44 minutes.
    - Les vérificateurs n'ont rien trouvé à corriger dans le code. `/code-review` a rendu 8 constats, tous corrigés, dont un HAR-1 qui cherchait l'absence d'un shell par liste noire au lieu de vérifier la commande exacte.
    - Le démarrage de Go et de Rust reste neutre : ±0,03 ms à l'A/B apparié, 3 tours de 401 lancements.
  - Vague 2 (#28, #34) : 6 agents, 0,88 million de jetons, 29 minutes.
    - Le vérificateur de D a trouvé un survivant réel : un terme calculé en int puis converti en Decimal.
    - `/code-review` a rendu 11 constats :
      - deux conversions encore invisibles à PERF-PY-28 ;
      - une médiane haute sur un nombre pair de paires ;
      - `versus` non testé ;
      - une jonction vers `node_modules` que le nettoyage manuel sous Windows peut traverser ;
      - un `fib-dev` invité à mesurer pendant que d'autres tournent.
    - Il a aussi relevé que #29 visait un HAR-5 déjà pris.
  - Vague 3 (#29, #35, #36, et les corrections de #28 et #34), à la demande du propriétaire de « tout faire en parallèle ».
    - Un premier workflow les portait toutes, mais il plafonne à 2 agents simultanés sur ce conteneur à 4 cœurs : min(16, CPU − 2). Il a été arrêté puis remplacé par cinq workflows, un par chantier, chacun avec son propre plafond. Les deux agents interrompus ont coûté environ 15 minutes ; le travail partiel de #35 a été repris par son successeur.
    - Les vérificateurs ont trouvé :
      - pour #35, la trace d'une exception perdue quand un flux hostile est remplacé par `None` ;
      - pour #36, une lacune réelle : une coquille dans la table des algorithmes (`"iterative": _fast`) passe toutes les gardes. Elle est documentée et reste ouverte ;
      - pour la correction de #28, la racine de `versus` non gardée ;
      - pour #29, la fin de ligne de stderr et la fermeture transitive des imports.
    - `npm run differentiel` joue 575 entrées sans écart sur l'arbre, et retrouve les 7 écarts de l'étape 9 sur la révision 5df6241.
    - `/code-review` a rendu 9 constats, dont un biais réel : 101 paires donnent une alternance déséquilibrée (51 contre 50), et la médiane garde l'effet d'ordre. Il fallait un nombre pair de paires.
    - Il a aussi relevé que l'orchestrateur avait corrigé lui-même un commentaire de `python/fib.py`, hors du circuit `fib-dev`.
    - La fusion de #29 a conflité dans cinq fichiers, parce que #29 et la correction de #28 réécrivaient `extract` en même temps. La résolution est revenue à un `fib-dev`.
    - Une dernière `/code-review` sur cette fusion a rendu 14 constats, dont 13 retenus et corrigés :
      - HAR-6 ne voyait ni les tests lancés par un glob, ni `node --run`, ni `npm test`, ni une URL relative sans `./` ;
      - la boucle sortait en 0 sans rien chercher sous Windows ;
      - la parité des paires n'était imposée que dans `main`.
    - Son vérificateur a aligné le marqueur de la boucle sur X-4 (`[^\x20-\x7e]`). 136 mutants ont été joués, et le catalogue compte désormais 53 mutants qui rougissent le témoin de la boucle (46 avant), sept de plus sur la projection.
    - Le lancement parallèle des quatre CLI dans la boucle a été écarté : c'est un outil de recherche, et ses 62 s par défaut ne sont pas un enjeu.
- **A aidé** :
  - Le critique adverse par piste a évité deux refactorisations sans gain, D et E, sur preuve par mutants et par instructions, non par opinion.
  - Une seule série de questions a suffi, parce que chaque option portait son coût mesuré.
  - En vague, une PR couvre plusieurs tickets indépendants sans perdre la vérification par ticket.
- **Friction** :
  - Les numéros de garde se télescopent quand des tickets écrits ensemble sont livrés dans un autre ordre : HAR-4 et HAR-5 visés deux fois, PERF-PY-26 visé par trois conceptions.
  - Les tickets de documentation sont livrés avec des numéros de ligne déjà périmés.
  - Les fichiers partagés (le catalogue, trois fois, puis `PRD.md`, `CONTEXT.md`, `contracts/README.md`, `contracts/bench.ts` et `contracts/cli.test.ts`) produisent des conflits dès que des tickets parallèles les touchent : un en vague 2, sept en vague 3.
  - Le plafond de concurrence des workflows dépend du nombre de cœurs : sur 4 cœurs, un workflow ne lance que 2 agents à la fois. « Tout en parallèle » demande un workflow par ticket.
  - L'orchestrateur a été pris à corriger lui-même du code (un commentaire). La revue l'a vu, mais la règle de `CLAUDE.md` se perd facilement en fin de vague.
- **À changer** :
  - Réserver les numéros de garde au moment de publier les tickets, dans l'ordre de livraison prévu.
  - Citer les passages par leur texte, non par leur ligne.
  - Garder les compteurs du README à l'orchestrateur, ce qui a évité les conflits sur la ligne la plus disputée.
  - Ne pas paralléliser deux tickets qui réécrivent la même fonction (ici `extract`) : les enchaîner, ou les confier au même `fib-dev`.
  - Lancer un workflow par ticket quand le nombre de cœurs plafonne la concurrence.

## 2026-10-13 — Revue finale de cohérence en workflow (hors skill), gardes d'identité de l'aiguillage (#39 à #43), puis `/code-review`, avant la version v1.0.0

- **Contexte** : avant la version finale, le propriétaire a demandé une revue de cohérence de tout le dépôt et sa fermeture. Il a ensuite demandé de lancer en parallèle la garde d'identité de la table des algorithmes, une lacune que la vérification de #36 avait trouvée.
- **Démarche** : quatre workflows de revue, un par angle (chiffres et commandes, cohérence entre documents, tickets et traçabilité, code et gardes). Chacun a un réviseur qui vérifie par exécution, puis un vérificateur adverse qui tente de réfuter chaque constat.
  - 46 constats en sont sortis : 40 confirmés et 6 réfutés.
  - L'orchestrateur a corrigé les constats de documentation. Ceux de code ont fait le ticket #43, livré par un `fib-dev`.
  - Les gardes d'identité ont fait quatre tickets, un par langage (#39 à #42), dans quatre workflows parallèles.
- **Ce que la revue a trouvé** :
  - des chiffres périmés : A3 du PRD au 2026-10-11, « 402 cas verts » au lieu de 398, « cinq chantiers » ;
  - une ADR qui décrivait une refonte que le propriétaire avait annulée ;
  - un PRD qui disait le harnais facultatif ;
  - un piège `process.exit()` déclaré « à vérifier », pourtant reproductible sous Linux ;
  - une procédure de tickets fondée sur des commandes `gh issue`, refusées en session infonuagique ;
  - deux optimisations sans garde : les accesseurs paresseux de stdout en TypeScript, devenus PERF-TS-11, et le contenu de la table de lancement, devenu HAR-7 ;
  - trois lacunes non nommées au catalogue : le code de sortie, le `fib` de `processIo` et le `forget` de Rust.
- **Gardes d'identité** : chaque langage a sa forme.
  - Go (PERF-GO-11) : une empreinte d'allocations, sans changer la production (4, 67 et 52 allocations à n = 1023).
  - Rust (PERF-RS-13) : des adresses de fonction, par une fonction privée `algorithm`.
  - TypeScript (PERF-TS-10) : le nom et des espions, sur la table exportée pour les seules gardes.
  - Python (PERF-PY-30) : l'identité de chaque entrée de `_ALGOS`.
  
  Les six échanges d'aiguillage les rougissent toutes.
- **Mesure** : le code généré par Rust change (un appel indirect au lieu de l'inlining), et TypeScript gagne un export. `npm run bench -- rust ts --contre de075bf`, au repos :
  - TypeScript : de −2,5 à +0,8 %, avec des signes alternés.
  - Rust : un premier tour de 100 paires a montré +4,7 % sur `petit (10)`. Deux tours de 200 paires l'ont démenti (−1,9 %, puis +1,0 %), sans qu'aucun scénario ne garde son signe : c'est du bruit.
- **A aidé** :
  - Le vérificateur adverse a réfuté 6 constats sur 46, dont deux affirmations fausses sur l'A/B et sur A3, et a resserré plusieurs correctifs avant leur application.
  - L'A/B apparié du dépôt a servi sur la mesure même qui l'avait motivé, et sa répétition a départagé un écart que la première passe faisait croire réel.
- **Friction** :
  - L'orchestrateur a deux fois mal lancé un workflow : un SHA de base inventé, puis un chemin de script provisoire envoyé en parallèle de sa création. Le premier a dû être arrêté et relancé.
  - Les tickets écrits ensemble se recoupent encore : #43 visait 113 tests par langage, alors que HAR-7, à un test par commande, en donne 116.
- **`/code-review` du lot final** : 12 constats, tous traités.
  - Huit de documentation :
    - le catalogue se contredisait sur `_ALGOS` ;
    - les survivants des gardes d'identité n'étaient pas nommés ;
    - les commandes de build échappent à HAR-7 ;
    - la traçabilité de F6 était incomplète ;
    - la procédure REST ne filtrait pas la frontière ;
    - la phrase du README sur les revues était fausse.
  - Quatre de code, chez deux `fib-dev` :
    - `algos` exportée en lecture seule par le type ;
    - un seul lanceur pour PERF-TS-9 et 11 ;
    - HAR-7 bouclé sur `IDS` ;
    - `hasGap`.
- **À changer** :
  - Toujours lire la tête (`git rev-parse`) au moment de lancer un workflow, jamais la recopier de mémoire.
  - Respecter la règle « un ticket et un langage par `fib-dev` » jusque dans les retouches. La dernière `/code-review` a relevé deux invocations qui mêlaient #41 et #43, ou ts/, contracts/ et python/.
  - Écrire dans un ticket le compte attendu par la formule, par exemple 6 ORA, plus le groupe du harnais, plus 99 cas, plutôt que par un nombre.

## 2026-10-13 — Seconde `/handoff`, file d'instructions et fermeture pour la version v1.0.0

- **Contexte** : après #25, le propriétaire a mis en file une quinzaine de consignes enchaînées par « ensuite » :
  - les pistes ;
  - le journal, le README et le PRD ;
  - la fermeture des PR et des tickets ;
  - le handoff et le résumé ;
  - le nettoyage et `CLAUDE.md` ;
  - la revue finale, la version v1.0.0, puis la poussée sur `main`.
  
  Quatre demandes « en parallèle » sont venues s'y ajouter pendant l'exécution. L'orchestrateur les a suivies dans l'ordre, sauf sur deux points : l'étiquette `v1.0.0` vient en dernier, après la poussée et le nettoyage, et le résumé suit l'étiquette, pour que l'étiquette marque le dernier commit et que le résumé la cite.
- **Seconde `/handoff`** : le document de l'étape 11 a été réécrit, pas complété, pour la fermeture du dépôt. Il garde la même règle, « renvoyer, ne pas redire » :
  - l'état vérifié sur `main` ;
  - les points ouverts sans ticket, chacun avec l'endroit où il est consigné et ce qu'il faudrait ;
  - les pièges de l'environnement ;
  - les skills suggérées.
  
  Rédigé avant le lot final, il a dû être repris après : 409 tests au lieu de 413, la garde d'identité encore ouverte, Windows non signalé. C'est la friction de l'étape 11, un instantané vite périmé, que la fermeture rend plus visible encore.
- **Parallélisme** :
  - un workflow par ticket, à cause du plafond de deux agents par workflow sur quatre cœurs ;
  - des `fib-dev` en arrière-plan, chacun dans son worktree, pour les retouches de la dernière `/code-review`.
  
  Pendant ce temps, l'orchestrateur tenait les documents, et la session a été résumée au moins une fois. Les brouillons laissés dans le dossier de travail de la session ont porté l'état d'un contexte à l'autre : le handoff, les notes de version et l'entrée du journal des pistes.
- **A aidé** :
  - L'ordre explicite de la file, qui a permis de repousser l'étiquette et le résumé sans redemander.
  - Les brouillons hors dépôt, relus au lieu d'être réécrits de mémoire.
- **Friction** :
  - Une consigne envoyée pendant un tour peut interrompre un workflow en arrière-plan (#25), et chaque « en parallèle » est arrivé pendant qu'un lot tournait.
  - Le handoff et les notes de version dépendent des derniers comptes. Les écrire avant le dernier lot oblige à les reprendre.
  - L'étiquette `v1.0.0` n'a pas pu être publiée. La session infonuagique ne pousse que des branches : `git push origin v1.0.0` et l'écriture des objets git par l'API (`git/tags`, `git/refs`) rendent HTTP 403, et aucun outil MCP ne crée de release. L'étiquette et la release reviennent donc au propriétaire, sur le commit qui porte cette entrée, avec les notes de version préparées par l'orchestrateur.
- **À changer** :
  - Recevoir une file d'instructions comme une liste de tâches, posée dès la réception.
  - Écrire le handoff et les notes de version en dernier, après la vérification finale de `main`, et y laisser des marqueurs pour les comptes plutôt que des chiffres provisoires.
  - Vérifier dès le début de la file que la session peut publier ce qu'on lui demande en dernier (une étiquette, une release), pour annoncer tôt ce qui reviendra au propriétaire.

## 2026-10-14 — Audits par langage : vérification en workflow, `fib-dev`, `/code-review` ([agbruneau/Fib#45](https://github.com/agbruneau/Fib/pull/45))

- **Contexte** : quatre `audit.md`, un par dossier de langage, écrits par un autre modèle (commit a92a808), étaient à implémenter. Aucun ticket : la demande du propriétaire en tenait lieu, et chaque prompt de `fib-dev` aussi.
- **Démarche** :
  - un workflow de vérification avant tout code, trois agents par audit : exactitude (chaque affirmation contre le code et par l'exécution), tri (chaque constat : à implémenter, déjà documenté, refusé ou faux, mutants joués dans des copies), critique adverse du tri ;
  - un `fib-dev` par langage dans son worktree, suivi d'un vérificateur adverse par branche, puis deux rondes de correction, la seconde avec les constats de `/code-review` sur la plage `a92a808..1a4c6d2` ;
  - la documentation (catalogue, README, PRD, suivi de chaque audit) écrite par l'orchestrateur seul, après les fusions.
- **A aidé** :
  - La vérification avant l'implémentation. Une quarantaine d'affirmations des audits étaient inexactes, dont six changeaient une décision : un mutant cité qui rougissait déjà PERF-TS-9, deux lacunes dites « documentées » absentes du catalogue, une garde (HAR-6) créditée d'une exigence qu'elle ne vérifie pas. Les implémenter à la lettre aurait ajouté des gardes inutiles et recopié ces erreurs dans le catalogue. Chaque audit garde désormais un § 7 de suivi et d'errata.
  - Un invariant commun à Go, Rust et Python pour « la durée est mesurée » : l'espion chronomètre son propre sommeil avec l'horloge de la production, à l'intérieur de la fenêtre. La borne est exacte sans dépendre de la précision du sommeil, donc sans faux rouge sous charge ni sous Windows (390 exécutions sous charge en Go, 0 échec).
  - Le vérificateur par branche, à chaque ronde : des commentaires faux que les `fib-dev` laissaient (« seuls ces trois tests », un survivant donné pour borné), et une famille de survivants de plus.
- **Friction** :
  - Les mutants d'une garde d'aiguillage forment une famille sans fin : un seuil sur n, une redéclaration, des parenthèses, un pointeur, une fonction auxiliaire, une copie en ligne. Chaque ronde en trouvait une. La règle qui a permis de s'arrêter : fermer ce qui l'est de façon déterministe et à coût nul, écrire le reste comme survivant, et ne jamais présenter une liste de survivants comme complète.
  - Le défaut Windows de PERF-PY-23 (#35) avait passé `/code-review` le 2026-10-13 sans être vu : la session tournait sous Linux, et le témoin écrivait sur les flux bruts de CPython. Seul un audit fait sous Windows l'a trouvé. Le correctif n'est éprouvé que par émulation.
  - `tdd` reste absente de la session infonuagique (`Unknown skill: tdd`) : les `fib-dev` ont suivi le cycle à la main.
  - Un agent de tri, en éprouvant `rustup override`, a installé une chaîne Rust de plus (retirée), et un `fib-dev` a tué par `pkill yes` les boucles de charge d'un autre.
- **Décision du propriétaire** : le dernier bit impair de `fast` en Go, en un produit comme en Rust et en Python, laissé ouvert par le tri, a été fait ensuite par un `fib-dev`. Le banc n'avait que des indices pairs : `/code-review` a rappelé que toute mesure passe par lui (CLAUDE.md), et un huitième scénario, impair (999 999), l'y fait entrer pour les quatre langages. Le banc ordinaire, avant et après, ne sépare pas le gain de la dérive de la machine (calcul 6,9 → 6,0 ms à 999 999, mais 6,5 → 5,9 ms au pair inchangé), et son A/B n'apparie que le temps total, où l'effet se perd dans le bruit. Seul un A/B apparié des binaires sur `--time`, hors banc, le chiffre : −15,6 % à 999 999. Apparier la durée de calcul dans l'A/B du banc est la suite à donner. PERF-GO-13 garde la forme par le texte, un produit par branche : l'empreinte d'allocations ne la distingue que d'une unité (51 contre 52 à n = 1023), trop peu pour une garde robuste.
- **À changer** :
  - Lancer `npm run check` sous Windows avant de fermer un chantier qui touche un flux standard ou une fin de ligne.
  - Dire au vérificateur, dès la première ronde, de ne retenir que des défauts de portée réelle, et à chaque prompt de ne tuer que ses propres processus.

## Bilan au 2026-10-14

| Skill | Usage |
|---|---|
| `/setup-matt-pocock-skills` | utilisée : tracker GitHub, étiquettes, docs du domaine |
| `/grill-with-docs` (`grilling`, `domain-modeling`) | utilisée : glossaire, ADR-0001 et ADR-0002, PRD éprouvé |
| `/to-spec` | utilisée : spec #1 |
| `/to-tickets` | utilisée : tickets #2 à #5, avec dépendances natives |
| `tdd` | utilisée : catalogue de cas, puis cycles des quatre `fib-dev`; absente de la session infonuagique des 2026-10-12, 2026-10-13 et 2026-10-14 (`Unknown skill: tdd`), où les `fib-dev` ont suivi son cycle à la main |
| `/implement` | remplacée par l'agent `fib-dev`, qui applique `tdd` directement |
| `resolving-merge-conflicts` | introuvable dans la session infonuagique : les conflits des vagues parallèles du 2026-10-13 (huit, dans six fichiers) ont été résolus à la main : par un `fib-dev` pour la fusion de #29 (cinq fichiers, code et documentation), par l'orchestrateur pour les trois autres (`docs/cas-de-test.md` deux fois, `CONTEXT.md`) |
| `research`, `prototype` | non utilisées; un `prototype` aurait détecté plus tôt le piège de `go run` |
| `/code-review` | utilisée trois fois le 2026-10-11, axes Standards et Spec : après la refactorisation, après l'optimisation, après la borne à 1 000 000; puis le 2026-10-12, sur les tickets #6 à #14 et sur #16; puis le 2026-10-13, sur #20, #21 et #25, puis quatre fois sur les pistes A, C, D et E (#27 à #36), une par vague et une dernière sur la fusion de #29, puis une sur le lot final (#39 à #43), soit huit passes ce jour-là; puis le 2026-10-14, sur les gardes des audits par langage (#45) |
| `diagnosing-bugs` | utilisée à l'étape 9 (#16), lue dans `mattpocock/skills` faute de plugin; précédée d'une boucle différentielle pour trouver l'écart |
| `/improve-codebase-architecture`, `codebase-design`, `grilling` | utilisées à l'étape 10 (#20), lues dans `mattpocock/skills` : rapport de 5 candidats, *design it twice* à 4 prototypes, 2 tours de *grilling*; puis pour les pistes A, C, D et E : deux conceptions prototypées par piste, un critique adverse, une série de questions |
| `/handoff` | utilisée deux fois, lue dans `mattpocock/skills` : à l'étape 11, puis à la fermeture du dépôt pour la version v1.0.0; chaque fois un document de reprise hors dépôt, envoyé au propriétaire |

Hors skill : l'audit du PRD (2026-10-10), l'audit en workflow du 2026-10-11 (auditeurs en lecture seule), puis l'optimisation des performances et la borne à 1 000 000, confiées à des sous-agents `fib-dev`, l'épreuve des cas de test en workflow, et, le 2026-10-13, les workflows `ultracode` de livraison : un `fib-dev`, un vérificateur adverse par exécution et un tour de correction par ticket; le 2026-10-14, la vérification en workflow des audits par langage, avant leur implémentation.

Six leçons :

1. **Un catalogue de cas avant le code** a fait converger quatre implémentations indépendantes. Elles sont passées au vert d'un seul coup quand l'oracle est arrivé.
2. **Les sondes par sous-agents valent le coût quand elles exécutent**, plus que quand elles lisent de la doc. Le piège de `go run`, la toolchain MSVC et les fins de ligne CRLF ne sont apparus qu'à l'exécution.
3. **Les données externes sont un prérequis à traiter avant les agents.** L'oracle bloqué a immobilisé les quatre tickets.
4. **Mesurer avant d'optimiser.** Le banc d'essai a montré que le temps partait au démarrage et dans la conversion décimale, pas dans `fast`, qui calculait F(100 000) en moins d'une milliseconde. À 1 000 000, il a aussi dit où ne rien changer : Go, Rust et TypeScript convertissaient déjà en temps sous-quadratique.
5. **La revue trouve ce que les agents laissent passer.** À chacun de ses trois passages du 2026-10-11, `/code-review` a trouvé un défaut que les `fib-dev` n'avaient pas vu : le lecteur d'oracle Rust qui n'ignorait plus les lignes vides (PRD § 7), `matrix` en TypeScript qui refaisait l'ancien `fast`, le chemin `Decimal` de Python qui contournait la seam.
6. **Un critique adverse qui exécute vaut mieux qu'un rapport qui raisonne.** Il a réfuté, par mutants et par instructions, quatre affirmations du rapport de l'étape 10 et deux refactorisations sans gain (D et E). À la livraison, un vérificateur par ticket a trouvé, avant la revue, des survivants réels : un module paresseux, un int converti en Decimal, une trace d'exception perdue.
