# Gabarit d'évaluation académique — projet logiciel de recherche ou de génie

Grille conçue pour évaluer un dépôt logiciel présenté comme travail universitaire
(projet de fin d'études, projet de cycle supérieur, laboratoire d'expérimentation
algorithmique). Elle est appliquée dans
[`evaluation-academique-2026-09-15.md`](evaluation-academique-2026-09-15.md).

## 1. Principes

1. **Preuve rejouable.** Chaque constat porte une commande, un fichier ou un
   extrait que l'évaluateur a exécuté ou lu. Un chiffre non réexécuté est cité
   comme tel, avec sa date et son hôte d'origine.
2. **Régime de preuve explicite.** Chaque constat est marqué :
   `[E]` exécuté par l'évaluateur, `[L]` lu dans le dépôt ou une source externe,
   `[D]` déduit sans exécution. Une déduction ne prend jamais le ton d'une
   vérification.
3. **Tout est rapporté.** Les constats mineurs figurent au même titre que les
   majeurs, avec une sévérité qui dit lesquels comptent.
4. **Le score n'absorbe pas la recevabilité.** Certaines exigences sont des
   conditions préalables (§ 3) ; un manquement y est signalé à part, quel que soit
   le score.

## 2. Critères et pondération

| # | Critère | Poids | Ce qu'on regarde | Preuves attendues |
|---|---|---|---|---|
| C1 | Problème, objectifs, positionnement | 5 | Énoncé du problème, hypothèses, ce qui est mesurable, situation par rapport à l'existant | README, docs d'intention, références à des implémentations comparables |
| C2 | Fondements théoriques et algorithmiques | 10 | Exactitude des identités, preuves, analyse de complexité, justesse des ordres de grandeur, bibliographie | docs d'algorithmes, commentaires de code, citations |
| C3 | Conception et architecture | 15 | Découpage, sens des dépendances, interfaces, état global, cohérence entre la doc d'architecture et le code | graphe d'imports vérifié, tests d'architecture, ADR |
| C4 | Implémentation | 15 | Idiomes du langage, gestion des erreurs, concurrence, mémoire, lisibilité, dette locale (`nolint`, `TODO`, `panic`) | build, vet, lint, race, lecture des chemins critiques |
| C5 | Vérification et validation | 15 | Étendue et indépendance des oracles, tests de propriétés, fuzzing, couverture mesurée **et** gardée, e2e | exécution de la suite, profil de couverture, lecture des oracles |
| C6 | Performance et méthode expérimentale | 10 | Protocole (répétitions, bruit, ordre), artefacts archivés, cohérence des chiffres publiés avec les mesures, hypothèses non testées | baselines, `benchstat`, réexécution ponctuelle |
| C7 | Documentation et communication | 10 | Exactitude, datation, absence de contradictions internes, accessibilité pour un lecteur neuf, règle de langue appliquée | lecture croisée README / CHANGELOG / docs / code |
| C8 | Processus d'ingénierie et outillage | 10 | Gate local et CI, épinglage des outils, versionnement, changelog, sécurité de la chaîne de build, revue | scripts, workflows, tags, historique git |
| C9 | Intégrité, attribution, licences | 5 | Sources tierces nommées et licences respectées, divulgation de l'aide (IA, collaborateurs), références | grep des notices, comparaison avec l'amont, en-têtes de licence |
| C10 | Maintenabilité, dette, perspectives | 5 | Surface de code par rapport au problème, dette consignée, fragilités connues, pistes crédibles | LOC par composant, ADR « à revoir si », dépendances |

Total : 100.

## 3. Conditions de recevabilité (hors score)

| Condition | Vérification |
|---|---|
| R1 — Le dépôt se construit et sa suite de tests passe sur un hôte propre | `go build ./...`, `go test ./...` (ou équivalent) |
| R2 — Les affirmations chiffrées du README sont datées et rejouables, ou signalées comme non rejouées | lecture du README, réexécution d'au moins deux commandes annotées |
| R3 — Le code tiers est attribué et sa licence respectée | recherche des notices, comparaison de fichiers suspects avec leur amont |
| R4 — L'aide extérieure (personnes, outils d'IA) est divulguée | README, historique des commits |

Verdict de recevabilité : *recevable*, *recevable sous réserve* (une condition
manque, corrigeable sans refonte), *non recevable*.

## 4. Échelle par critère

| Bande | Descripteur | Ce que ça veut dire |
|---|---|---|
| 90–100 | Exemplaire | Rien à corriger de substantiel ; pourrait servir de référence à d'autres projets |
| 80–89 | Solide | Quelques écarts identifiés, aucun ne compromet la valeur du travail |
| 70–79 | Adéquat | Écarts réels qui demandent une passe de correction avant réutilisation |
| 60–69 | Fragile | Manques qui affaiblissent la confiance dans les résultats ou la structure |
| < 60 | Insuffisant | Le critère n'est pas satisfait ; correction obligatoire |

Note globale = Σ (score du critère × poids) / 100. Correspondance lettrée
(convention des universités québécoises) : A+ ≥ 90, A 85–89, A- 80–84,
B+ 77–79, B 73–76, B- 70–72, C+ 65–69, C 60–64, C- 55–59, D 50–54, E < 50.

## 5. Structure de l'évaluation rendue

1. Fiche (objet, commit, date, hôte, évaluateur, régime de preuve).
2. Verdict : note globale, lettre, tableau des dix critères, recevabilité, trois
   phrases de synthèse.
3. Méthode : ce qui a été lu, exécuté, non vérifié.
4. Portrait chiffré du dépôt.
5. Évaluation critère par critère : constats forts, réserves, preuve, note.
6. Constats prioritaires transversaux, classés par sévérité
   (*bloquant* : touche une condition de recevabilité ; *haute* : erreur de fond
   observable ; *moyenne* : écart structurel sans incident ; *basse* : hygiène ;
   *info* : choix documenté, sans action).
7. Recommandations, ordonnées par rapport effort/valeur.
8. Annexes : preuves brutes (sorties de commandes, comparaisons de sources).
