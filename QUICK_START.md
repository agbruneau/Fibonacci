# 🚀 Guide de Démarrage Rapide

## 📦 Installation (30 secondes)

```bash
# 1. Clone du projet
git clone <votre-repo>
cd fibcalc

# 2. Build
make build

# 3. Test rapide
./build/fibcalc -n 100 -algo fast
```

**Résultat attendu:**
```
F(100) = 354,224,848,179,261,915,075
Calculation time: 18µs
```

---

## 💻 Mode CLI

### Calcul simple
```bash
./build/fibcalc -n 1000 -algo fast
```

### Calcul détaillé
```bash
./build/fibcalc -n 1000 -algo fast -d
```

### Comparer tous les algorithmes
```bash
./build/fibcalc -n 10000000 -algo all
```

### Calibration
```bash
./build/fibcalc --calibrate
```

---

## 🌐 Mode Serveur

### Démarrage
```bash
# Option 1: Avec Make
make run-server

# Option 2: Direct
./build/fibcalc --server --port 8080
```

### Test des endpoints

**Calcul:**
```bash
curl "http://localhost:8080/calculate?n=1000&algo=fast"
```

**Health check:**
```bash
curl "http://localhost:8080/health"
```

**Liste des algorithmes:**
```bash
curl "http://localhost:8080/algorithms"
```

---

## 🐳 Mode Docker

### Build et Run
```bash
# Build
docker build -t fibcalc:latest .

# Run CLI
docker run --rm fibcalc:latest -n 1000 -algo fast

# Run Serveur
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080
```

### Test
```bash
curl "http://localhost:8080/calculate?n=1000"
```

---

## 🧪 Tests

### Tests complets
```bash
make test
```

### Avec couverture
```bash
make coverage
# Ouvre coverage.html dans le navigateur
```

### Benchmarks
```bash
make benchmark
```

---

## 🛠️ Commandes Essentielles

| Commande | Description |
|----------|-------------|
| `make build` | Compiler le projet |
| `make test` | Exécuter tous les tests |
| `make run-fast` | Test rapide (n=1000) |
| `make run-server` | Démarrer le serveur |
| `make coverage` | Rapport de couverture |
| `make clean` | Nettoyer les artifacts |
| `make help` | Liste toutes les commandes |

---

## 📚 Documentation

- **README.md** - Documentation principale
- **API.md** - Documentation API REST complète
- **IMPROVEMENTS.md** - Détails techniques
- **CHANGELOG.md** - Historique des versions
- **SUMMARY.md** - Résumé de la refactorisation

---

## 🎯 Exemples Pratiques

### 1. Calcul Simple
```bash
./build/fibcalc -n 50
# F(50) = 12,586,269,025
```

### 2. Calcul avec Détails
```bash
./build/fibcalc -n 1000 -d
# Affiche: temps, taille binaire, notation scientifique
```

### 3. Comparaison d'Algorithmes
```bash
./build/fibcalc -n 10000000 -algo all
# Compare: fast, matrix, fft
```

### 4. API REST
```bash
# Terminal 1: Démarrer le serveur
./build/fibcalc --server --port 8080

# Terminal 2: Tester l'API
curl "http://localhost:8080/calculate?n=100&algo=fast"
curl "http://localhost:8080/health"
curl "http://localhost:8080/algorithms"
```

### 5. Docker Compose
```yaml
# docker-compose.yml
version: '3.8'
services:
  fibcalc:
    build: .
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080"]
```

```bash
docker-compose up -d
curl "http://localhost:8080/calculate?n=1000"
```

---

## 🔥 Fonctionnalités Clés

### ✅ Serveur HTTP Production-Ready
- Graceful shutdown
- Timeouts configurés
- Logging de toutes les requêtes
- Health checks
- Validation stricte

### ✅ Algorithmes Optimisés
- Fast Doubling (O(log n))
- Matrix Exponentiation
- FFT-Based Calculator
- Parallélisation multi-core
- Zero-allocation strategy

### ✅ Infrastructure Moderne
- Makefile complet
- Docker optimisé
- CI/CD GitHub Actions
- Documentation exhaustive

---

## 🐛 Dépannage

### Le build échoue
```bash
# Vérifier Go version
go version  # Doit être >= 1.22

# Nettoyer et rebuild
make clean
go mod tidy
make build
```

### Les tests échouent
```bash
# Exécuter avec verbose
go test -v ./...

# Exécuter un test spécifique
go test -v -run TestHandleCalculate ./internal/server/
```

### Le serveur ne démarre pas
```bash
# Vérifier que le port est libre
lsof -i :8080

# Utiliser un autre port
./build/fibcalc --server --port 9090
```

---

## 📈 Performance

### Petits nombres (n < 1000)
```bash
./build/fibcalc -n 100 -algo fast
# ~20µs
```

### Moyens nombres (n = 1M)
```bash
./build/fibcalc -n 1000000 -algo fast
# ~100ms
```

### Grands nombres (n = 10M)
```bash
./build/fibcalc -n 10000000 -algo fast
# ~2-5s (selon CPU)
```

### Très grands nombres (n = 100M)
```bash
./build/fibcalc -n 100000000 -algo fft --timeout 10m
# ~1-2min (selon CPU)
```

---

## 🎓 Prochaines Étapes

1. **Lire la documentation**
   - README.md pour vue d'ensemble
   - API.md pour l'API REST
   - IMPROVEMENTS.md pour les détails techniques

2. **Explorer le code**
   - `internal/fibonacci/` - Algorithmes
   - `internal/server/` - Serveur HTTP
   - `cmd/fibcalc/` - Point d'entrée

3. **Contribuer**
   - Voir CONTRIBUTING.md
   - Créer une branche
   - Soumettre une PR

4. **Déployer**
   - Utiliser Docker
   - Configurer Kubernetes
   - Monitorer avec Prometheus

---

## 💡 Astuces

### Performance
```bash
# Calibrer pour votre machine
./build/fibcalc --calibrate

# Utiliser l'auto-calibration
./build/fibcalc -n 10000000 --auto-calibrate
```

### Développement
```bash
# Vérifier le code avant commit
make check

# Générer la couverture
make coverage
```

### Production
```bash
# Build optimisé pour production
make build

# Build pour toutes les plateformes
make build-all
```

---

## ✨ C'est parti !

Vous êtes maintenant prêt à utiliser le calculateur Fibonacci haute performance.

**Commande pour commencer:**
```bash
make build && ./build/fibcalc -n 100 -algo fast -d
```

**Pour le mode serveur:**
```bash
make run-server
# Puis: curl "http://localhost:8080/calculate?n=1000"
```

**Besoin d'aide ?**
```bash
./build/fibcalc --help
make help
```

---

**Bon calcul ! 🚀**
