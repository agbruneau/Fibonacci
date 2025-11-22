# Changelog

Tous les changements notables de ce projet seront documentés dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/fr/1.0.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-11-22

### Ajouté
- **Serveur HTTP amélioré**:
  - Graceful shutdown avec gestion des signaux SIGTERM/SIGINT
  - Timeout configurable pour les requêtes (5 minutes par défaut)
  - Middleware de logging pour toutes les requêtes
  - Endpoint `/health` pour vérifier l'état du serveur
  - Endpoint `/algorithms` pour lister les algorithmes disponibles
  - Timeouts HTTP configurés (Read, Write, Idle)
  - Utilisation des configurations CLI (threshold, fft-threshold, etc.)
  
- **Gestion des erreurs centralisée**:
  - Package `apperrors` avec codes d'exit standardisés
  - Nouvelle structure `ServerError` pour les erreurs serveur
  - Messages d'erreur JSON structurés pour l'API
  
- **Infrastructure de développement**:
  - `Makefile` complet avec toutes les commandes utiles
  - `Dockerfile` multi-stage pour images optimisées
  - Configuration GitHub Actions pour CI/CD
  - `.dockerignore` pour builds Docker optimisés
  
- **Documentation**:
  - `API.md` avec documentation complète de l'API REST
  - Exemples d'intégration en Python, JavaScript et Go
  - Guide de déploiement Docker et Docker Compose
  - Instructions de monitoring et benchmarking

### Modifié
- **Architecture du serveur**: Refactorisation complète avec pattern constructeur
- **Tests du serveur**: Ajout de tests pour tous les endpoints et le middleware
- **README**: Mise à jour avec liens vers la documentation API
- **Imports**: Utilisation cohérente du package `apperrors` dans tout le projet

### Amélioré
- **Performance du serveur**: Timeouts appropriés et gestion des ressources
- **Sécurité**: Validation améliorée des paramètres d'entrée
- **Observabilité**: Logging structuré de toutes les requêtes
- **Testabilité**: Couverture de tests améliorée pour le serveur

## [1.0.0] - 2024

### Ajouté
- Implémentation de 3 algorithmes de Fibonacci optimisés:
  - Fast Doubling avec parallélisation
  - Matrix Exponentiation avec algorithme de Strassen
  - FFT-Based Calculator pour les très grands nombres
- Support des calculs haute performance avec `sync.Pool`
- Système de calibration automatique des seuils
- Mode serveur HTTP basique
- Interface CLI complète avec progress bars
- Internationalisation (i18n)
- Tests unitaires et property-based testing
- Documentation complète

### Performance
- Algorithmes en O(log n) multiplications
- Zero-allocation strategy avec object pooling
- Parallélisation multi-cœur intelligente
- Multiplication FFT adaptative
- Cache de lookup table pour petites valeurs

## Notes de migration

### De 1.0.0 à 1.1.0

#### Serveur HTTP
Si vous utilisiez le serveur HTTP, notez les changements suivants :

**Avant (1.0.0):**
```go
srv := &server.Server{Registry: calculatorRegistry}
srv.Start(cfg.Port)
```

**Après (1.1.0):**
```go
srv := server.NewServer(calculatorRegistry, cfg)
srv.Start()
```

#### Imports
Les codes d'exit ont été déplacés :

**Avant:**
```go
const ExitSuccess = 0
```

**Après:**
```go
import apperrors "example.com/fibcalc/internal/errors"
// ...
os.Exit(apperrors.ExitSuccess)
```

#### API REST
- La réponse JSON inclut maintenant un champ `algorithm`
- Les erreurs sont retournées dans un format JSON structuré
- Nouveaux endpoints `/health` et `/algorithms`

## Roadmap

### [1.2.0] - Prévu
- [ ] Support de Prometheus metrics
- [ ] API GraphQL optionnelle
- [ ] Cache Redis pour résultats fréquents
- [ ] Rate limiting configurable
- [ ] Support WebSocket pour calculs longs avec progress streaming

### [2.0.0] - Future
- [ ] Support de calculs distribués
- [ ] Persistence des résultats
- [ ] Interface Web interactive
- [ ] API RESTful complète avec versioning
- [ ] Support Kubernetes avec health checks
