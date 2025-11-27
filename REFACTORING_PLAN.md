# Plan de Refactorisation du Code Go

Ce document décrit le plan de refactorisation du code Go dans ce dépôt. L'objectif principal est d'améliorer la maintenabilité, la lisibilité, les performances et la testabilité du code, tout en s'assurant que les fonctionnalités existantes restent intactes.

## Objectifs Généraux

*   **Améliorer la clarté et la lisibilité :** Simplifier les structures complexes, renommer les variables et fonctions pour une meilleure compréhension, et s'assurer que le code est auto-documenté.
*   **Réduire la dette technique :** Identifier et résoudre les problèmes de conception, les duplications de code et les mauvaises pratiques.
*   **Optimiser les performances :** Examiner les goulots d'étranglement potentiels, en particulier dans les algorithmes de calcul de Fibonacci, et appliquer des optimisations sans sacrifier la lisibilité.
*   **Renforcer la testabilité :** S'assurer que chaque composant peut être testé de manière isolée, facilitant ainsi l'ajout de nouvelles fonctionnalités et la détection de régressions.
*   **Adhérer aux meilleures pratiques Go :** Appliquer les conventions de nommage, la gestion des erreurs, la concurrence et les structures de projet idiomatiques.

## Axes de Refactorisation Prioritaires

### 1. Module `internal/fibonacci`

Ce module contient les implémentations des algorithmes de calcul de Fibonacci.
*   **Revue des algorithmes :** Vérifier l'efficacité et la robustesse des implémentations (`fastdoubling.go`, `fft_based.go`, `matrix.go`).
*   **Gestion des grands nombres :** S'assurer que la manipulation des très grands nombres est optimisée et correcte.
*   **Interface unifiée :** Évaluer si une interface plus générique ou un pattern de fabrique pourrait simplifier la sélection et l'intégration de nouveaux algorithmes.
*   **Tests unitaires :** Renforcer la couverture des tests, y compris les cas limites et les tests de performance.

### 2. Module `internal/server`

Ce module gère l'API HTTP et la logique du serveur.
*   **Gestion des requêtes :** Simplifier le traitement des requêtes et la validation des paramètres.
*   **Gestion des erreurs :** Uniformiser la manière dont les erreurs sont capturées, loggées et renvoyées aux clients. Utiliser des types d'erreurs personnalisés si nécessaire.
*   **Injection de dépendances :** Faciliter l'injection des dépendances (par exemple, les calculateurs de Fibonacci) pour améliorer la testabilité et la flexibilité.
*   **Middleware :** Standardiser l'utilisation des middlewares pour le logging, l'authentification (si applicable), et la gestion des timeouts.

### 3. Module `cmd/fibcalc` (Point d'entrée)

Le point d'entrée de l'application.
*   **Initialisation :** Clarifier la séquence d'initialisation du serveur, de la configuration et des autres composants.
*   **Gestion des flags CLI :** S'assurer que la configuration via les flags CLI est robuste et bien documentée.

### 4. Gestion des Erreurs Globale

*   **Centralisation :** Mettre en place une stratégie cohérente pour la gestion des erreurs à travers toute l'application.
*   **Contexte d'erreur :** Enrichir les erreurs avec des informations contextuelles pour faciliter le débogage.

### 5. Logging et Monitoring

*   **Structure des logs :** Assurer un format de log cohérent et exploitable.
*   **Configuration :** Faciliter la configuration du niveau de logging et des sorties.

## Approche

1.  **Analyse d'impact :** Avant chaque modification majeure, évaluer l'impact potentiel sur les performances et la stabilité.
2.  **Petits pas :** Effectuer des refactorisations par petites étapes, en s'assurant que le code reste fonctionnel après chaque changement.
3.  **Tests :** S'appuyer fortement sur les tests existants et en ajouter de nouveaux pour chaque portion de code refactorisée.
4.  **Revue de code :** Chaque refactorisation significative devrait faire l'objet d'une revue par un pair.

Ce plan sera mis à jour au fur et à mesure de l'avancement de la refactorisation.
