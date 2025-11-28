# Changelog

Toutes les modifications notables de ce projet seront documentées dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2023-10-27

### Ajouté
- Implémentation initiale du Calculateur Fibonacci Haute Performance.
- Algorithmes : Fast Doubling, Exponentiation Matricielle, FFT-Based Doubling.
- Support des grands nombres avec `math/big`.
- Optimisations : `sync.Pool` pour le zéro-allocation, parallélisme multi-cœur.
- Interface en ligne de commande (CLI) avec drapeaux de configuration.
- Mode Serveur HTTP avec API REST.
- Calibration automatique et manuelle des performances.
- Documentation complète (README, API, code GoDoc).
- Support Docker et Makefile.
- Tests unitaires, benchmarks et tests de propriétés.
