# Changelog

Toutes les modifications notables de ce projet seront documentées dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-11-29

### Ajouté
- **Mode Interactif (REPL)** : Nouvelle session interactive pour effectuer plusieurs calculs (`--interactive`).
  - Commandes : `calc`, `algo`, `compare`, `list`, `hex`, `status`, `help`, `exit`.
  - Changement d'algorithme à la volée.
  - Comparaison de tous les algorithmes sur une même valeur.
- **Nouvelles options CLI** :
  - `-o, --output` : Export du résultat vers un fichier.
  - `-q, --quiet` : Mode silencieux pour scripts.
  - `--hex` : Affichage hexadécimal du résultat.
  - `--completion` : Génération de scripts d'autocomplétion (bash, zsh, fish, powershell).
  - `--no-color` : Désactivation des couleurs (respecte aussi `NO_COLOR`).
  - `--version, -V` : Affichage de la version.
  - `--calibration-profile` : Chemin personnalisé pour le profil de calibration.
- **Endpoint `/metrics`** : Métriques de performance du serveur HTTP.
- **Support i18n étendu** : Nouvelles langues (ES, DE, JA, ZH).
- **Thèmes de couleur CLI** : Support des thèmes dark, light, et none.
- **Barre de progression avec ETA** : Estimation du temps restant pour les calculs longs.
- **Calibration persistante** : Sauvegarde et chargement des profils de calibration.

### Amélioré
- Documentation complète mise à jour (README, API, Architecture).
- Tests de fuzzing pour la cohérence des algorithmes.
- Tests de charge/stress pour le serveur HTTP.
- Pools d'objets FFT pour réduction des allocations mémoire.
- Factory pattern pour le registre des calculateurs.

### Corrigé
- Gestion améliorée des signaux d'arrêt gracieux.
- Validation plus stricte des entrées utilisateur.

## [1.0.0] - 2023-10-27

### Ajouté
- Implémentation initiale du Calculateur Fibonacci Haute Performance.
- **Algorithmes** :
  - Fast Doubling : O(log n), parallélisme, zéro-allocation.
  - Exponentiation Matricielle : O(log n), algorithme de Strassen.
  - FFT-Based Doubling : Multiplication FFT forcée.
- Support des grands nombres avec `math/big`.
- **Optimisations** :
  - `sync.Pool` pour le recyclage des objets.
  - Parallélisme multi-cœur adaptatif.
  - Multiplication adaptative (Karatsuba/FFT).
  - Algorithme de Strassen pour matrices.
  - Mise au carré de matrices symétriques.
- **Modes d'exécution** :
  - CLI avec drapeaux de configuration complets.
  - Mode Serveur HTTP avec API REST.
- **API REST** :
  - Endpoint `/calculate` pour les calculs.
  - Endpoint `/health` pour le health check.
  - Endpoint `/algorithms` pour lister les algorithmes.
  - Rate limiting et headers de sécurité.
- Calibration automatique et manuelle des performances.
- Système d'internationalisation (i18n) avec support FR/EN.
- Documentation complète :
  - README avec guide de démarrage.
  - API.md pour l'API REST.
  - Documentation des algorithmes.
  - Guide de performance.
  - Politique de sécurité.
- Support Docker avec multi-stage build.
- Makefile complet pour le développement.
- Tests unitaires, benchmarks et tests de propriétés.
- Support des formats de sortie JSON et texte.

### Infrastructure
- Architecture Clean Architecture / Hexagonale.
- Structure de projet Go standard (`cmd/`, `internal/`).
- CI/CD avec GitHub Actions (lint, test, build, docker).
- Manifests Kubernetes pour le déploiement.

---

## Types de changements

- **Ajouté** pour les nouvelles fonctionnalités.
- **Modifié** pour les changements dans les fonctionnalités existantes.
- **Déprécié** pour les fonctionnalités qui seront supprimées prochainement.
- **Supprimé** pour les fonctionnalités maintenant supprimées.
- **Corrigé** pour les corrections de bugs.
- **Sécurité** en cas de vulnérabilités.
