# Fibonacci Benchmark (fibbench) - Production Ready

## Description

Ce projet fournit un outil CLI robuste, performant et observable pour comparer l'exécution concurrente de divers algorithmes de calcul de la suite de Fibonacci (F(N)) pour de très grands nombres entiers.

## Résumé des modifications (Refactorisation v3.0)

La version initiale (v2.0) était un prototype fonctionnel monolithique. Cette refactorisation (v3.0) a porté le code à un niveau de qualité production en appliquant les meilleures pratiques d'architecture logicielle et d'ingénierie de la fiabilité (SRE) en Go.

### Améliorations Architecturales
- **Structure Modulaire** : Le code a été restructuré selon le standard `cmd/internal/`. Les préoccupations sont séparées en paquets distincts (`config`, `fibonacci`, `runner`, `metrics`).
- **Gestion de la Concurrence Robuste** : Utilisation de `golang.org/x/sync/errgroup` pour gérer le cycle de vie des goroutines et la propagation de l'annulation, simplifiant l'orchestration par rapport à la gestion manuelle des `WaitGroup`.
- **Extensibilité** : Utilisation d'un patron de registre et d'une interface (`Calculator`) dans `internal/fibonacci`, permettant l'ajout facile de nouveaux algorithmes via `func init()`.

### Nouvelles Fonctionnalités et Bonifications
- **Journalisation Structurée** : Intégration de `log/slog` pour une journalisation performante et configurable (`-loglevel`).
- **Observabilité (Métriques Prometheus)** : Instrumentation du code pour suivre la durée d'exécution et les erreurs. Les métriques sont exposées via un endpoint HTTP optionnel (`-metrics-port`).
- **Tests Unitaires** : Introduction d'une suite de tests couvrant la correction des algorithmes et le respect de l'annulation de contexte.
- **Arrêt Gracieux (Graceful Shutdown)** : Gestion des signaux d'interruption (SIGINT, SIGTERM) pour annuler proprement les calculs et arrêter le serveur de métriques.

### Optimisations
- Les optimisations de performance critiques de la v2.0 (algorithmes O(log N), `sync.Pool` et `workspace`) ont été préservées et intégrées proprement dans la nouvelle architecture.

## Justifications Architecturales

1.  **Structure `cmd/internal/`** : Standard de l'industrie pour séparer le point d'entrée de la logique métier interne, garantissant l'encapsulation.
2.  **Utilisation de `errgroup`** : Simplifie la gestion de la concurrence, réduit le risque d'erreurs de synchronisation ou de fuites de goroutines, tout en gérant efficacement l'annulation propagée du contexte.
3.  **Métriques Prometheus et `slog`** : Standards de facto pour l'observabilité dans les environnements de production modernes.
4.  **Patron de Registre (via `init()`)** : Permet aux implémentations d'algorithmes de s'enregistrer automatiquement, favorisant un couplage faible avec le `Runner`.

## Guide de démarrage

### Prérequis
- Go 1.21+

### Compilation

```bash
# Installation des dépendances (Prometheus, errgroup)
go mod tidy

# Compilation de l'exécutable
go build -o fibbench ./cmd/fibbench