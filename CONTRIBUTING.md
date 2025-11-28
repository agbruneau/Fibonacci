# Guide de Contribution

Merci de l'intérêt que vous portez à contribuer au Calculateur Fibonacci ! Nous accueillons toutes les formes de contribution, que ce soit pour corriger des bugs, ajouter des fonctionnalités, améliorer la documentation ou optimiser les performances.

## Comment Contribuer

1.  **Forker le projet** : Créez une copie du dépôt sur votre compte GitHub.
2.  **Créer une branche** : Travaillez sur une branche dédiée pour votre fonctionnalité ou correction (`git checkout -b feature/ma-nouvelle-fonctionnalite`).
3.  **Commiter vos changements** : Faites des commits atomiques et bien décrits.
4.  **Pousser vers votre fork** : Envoyez vos changements sur GitHub (`git push origin feature/ma-nouvelle-fonctionnalite`).
5.  **Ouvrir une Pull Request** : Proposez vos changements pour intégration dans la branche principale.

## Standards de Code

-   **Gofmt** : Tout le code doit être formaté avec `gofmt` (ou `goimports`).
-   **Linting** : Utilisez `golangci-lint` pour vérifier la qualité du code.
-   **Documentation** : Tout le code exporté doit avoir des commentaires GoDoc clairs.
-   **Tests** : Toute nouvelle fonctionnalité doit être accompagnée de tests unitaires. Assurez-vous que `make test` passe.

## Exécuter les Tests

Utilisez le Makefile fourni pour exécuter les tests :

```bash
make test
```

Pour les benchmarks :

```bash
make benchmark
```

Pour vérifier la couverture de code :

```bash
make coverage
```

## Signaler des Bugs

Si vous trouvez un bug, veuillez ouvrir une issue sur GitHub en décrivant :
1.  Les étapes pour reproduire le bug.
2.  Le comportement attendu.
3.  Le comportement observé.
4.  Votre environnement (OS, version de Go, architecture).

Merci de votre aide !
