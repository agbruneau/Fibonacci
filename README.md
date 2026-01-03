# 🧮 Code — Portfolio de Projets Haute Performance

<div align="center">

![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)
![Rust](https://img.shields.io/badge/Rust-1.75%2B-000000?style=for-the-badge&logo=rust)
![Kafka](https://img.shields.io/badge/Apache_Kafka-3.7-231F20?style=for-the-badge&logo=apache-kafka)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)

**Collection de projets démontrant des patterns d'ingénierie logicielle avancés, l'optimisation des performances et les architectures distribuées.**

[FibGo](#-fibgo) • [FibRust](#-fibrust) • [PubSubKafka](#-pubsubkafka)

</div>

---

## 📋 Aperçu

Ce repository contient trois projets indépendants qui explorent différentes facettes du développement logiciel haute performance :

| Projet | Langage | Description | Licence |
|--------|---------|-------------|---------|
| [**FibGo**](./FibGo) | Go 1.25+ | Calculateur Fibonacci ultra-performant avec API REST | Apache 2.0 |
| [**FibRust**](./FibRust) | Rust 1.75+ | Calculateur Fibonacci parallèle avec NTT | MIT |
| [**PubSubKafka**](./PubSubKafka) | Go 1.24+ | Architecture événementielle avec Apache Kafka | MIT |

---

## 🔢 FibGo

<img src="https://img.shields.io/badge/Coverage-80%25-green?style=flat-square" alt="Coverage"> <img src="https://img.shields.io/badge/Status-Production--Ready-success?style=flat-square" alt="Status">

**FibCalc** est un calculateur de nombres de Fibonacci de pointe, capable de calculer $F(250\,000\,000)$ en quelques minutes.

### ✨ Caractéristiques Clés

- **Algorithmes Avancés**
  - 🚀 **Fast Doubling** — $O(\log n)$, méthode par défaut
  - 📐 **Exponentiation Matricielle** avec algorithme de Strassen
  - 🎵 **Multiplication FFT** pour les très grands nombres

- **Performance Extrême**
  - Pool de mémoire zéro-allocation (`sync.Pool`)
  - Parallélisme adaptatif multi-cœurs
  - Auto-calibration matérielle

- **Production Ready**
  - API REST avec métriques Prometheus
  - Mode REPL interactif
  - Support Docker & Kubernetes

### 🚀 Démarrage Rapide

```bash
cd FibGo

# Calculer F(10,000,000)
go run ./cmd/fibcalc -n 10000000

# Lancer le serveur API
go run ./cmd/fibcalc --server --port 8080

# Mode interactif
go run ./cmd/fibcalc --interactive
```

### 📊 Benchmarks

| Index (N) | Fast Doubling | Matrix | FFT | Chiffres |
|-----------|---------------|--------|-----|----------|
| 1,000,000 | 85ms | 110ms | 95ms | 208,988 |
| 100,000,000 | 45s | 62s | 48s | 20,898,764 |
| 250,000,000 | 3m 12s | 4m 25s | 3m 28s | 52,246,909 |

📖 [Documentation complète →](./FibGo/README.md)

---

## 🦀 FibRust

<img src="https://img.shields.io/badge/Rust-1.75%2B-orange?style=flat-square" alt="Rust"> <img src="https://img.shields.io/badge/License-MIT-yellow?style=flat-square" alt="MIT">

Implémentation Rust haute performance utilisant **Rayon** pour le parallélisme et des **Transformées de Fourier Numériques (NTT)** pour la multiplication de très grands entiers.

### ✨ Caractéristiques Clés

- **Performance Extrême** — $F(100\,000\,000)$ en **~1.2s**
- **Sélection Adaptative** — Choix automatique de l'algorithme optimal
- **Workspace Cargo** avec 3 crates modulaires

### 📦 Structure du Projet

```
FibRust/
├── crates/
│   ├── fibrust-core/     # Algorithmes (ibig, rustfft, rayon)
│   ├── fibrust-server/   # API HTTP (Axum)
│   └── fibrust-cli/      # Interface CLI (clap)
```

### 🚀 Démarrage Rapide

```bash
cd FibRust

# Compiler en mode release (LTO activé)
cargo build --workspace --release

# Calculer F(10,000,000)
cargo run -p fibrust-cli --release -- 10000000

# Comparer tous les algorithmes
cargo run -p fibrust-cli --release -- 10000000 -a all

# Lancer le serveur HTTP
cargo run -p fibrust-server --release -- --port 3000
```

### 📊 Benchmarks

| Index (n) | Fast Doubling | Parallel | FFT |
|-----------|---------------|----------|-----|
| 100K | 0.9 ms | 2.1 ms | 1.5 ms |
| 1M | 11 ms | 26 ms | 15 ms |
| 10M | 240 ms | 86 ms | **64 ms** |
| 100M | 7.13 s | 4.77 s | **1.15 s** |

📖 [Documentation complète →](./FibRust/README.md)

---

## 📨 PubSubKafka

<img src="https://img.shields.io/badge/Apache_Kafka-3.7.0-white?style=flat-square&logo=apache-kafka" alt="Kafka"> <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square" alt="Go">

Démonstration d'une **Architecture Événementielle (EDA)** enterprise-grade utilisant **Go** et **Apache Kafka**. Simule un cycle de vie complet de commandes e-commerce avec monitoring temps réel.

### 🏗 Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  📦 Producer │────▶│  📊 Kafka   │────▶│  ⚙️ Tracker │
│   (Orders)   │     │   Topic     │     │  (Consumer) │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  📊 Monitor │
                    │    (TUI)    │
                    └─────────────┘
```

### ✨ Caractéristiques Clés

- **Event-Driven Architecture (EDA)** — Découplage complet via messagerie asynchrone
- **Garantie de Livraison** — ACKs Kafka pour l'intégrité des données
- **Double Observabilité** — Logs techniques + Audit métier
- **Graceful Shutdown** — Zéro perte de données sur `SIGTERM`/`SIGINT`
- **TUI Dashboard** — Monitoring temps réel du débit et des taux de succès

### 🚀 Démarrage Rapide

```bash
cd PubSubKafka

# Déploiement automatisé (Linux/macOS)
make run

# OU déploiement manuel
make docker-up                              # Lancer Kafka
go run -tags kafka cmd/producer/main.go     # Terminal 1
go run -tags kafka cmd/tracker/main.go      # Terminal 2

# Monitoring TUI
make run-monitor
```

### ⌨️ Commandes Makefile

| Commande | Description |
|----------|-------------|
| `make build` | Compiler tous les binaires |
| `make run` | Déployer Kafka + services |
| `make stop` | Arrêt gracieux |
| `make test-cover` | Tests + rapport de couverture |

📖 [Documentation complète →](./PubSubKafka/README.md)

---

## 🛠 Technologies Utilisées

### Langages & Runtimes

| Technologie | Version | Projets |
|-------------|---------|---------|
| **Go** | 1.24+ / 1.25+ | FibGo, PubSubKafka |
| **Rust** | 1.75+ | FibRust |

### Frameworks & Bibliothèques

| Catégorie | Go | Rust |
|-----------|-----|------|
| **HTTP** | net/http | Axum |
| **CLI** | cobra | clap |
| **Observabilité** | zerolog, Prometheus | — |
| **Parallélisme** | goroutines | Rayon |
| **Big Integers** | math/big, GMP | ibig |
| **FFT** | Custom bigfft | rustfft |
| **Kafka** | confluent-kafka-go | — |

### Infrastructure

- **Docker** & **Docker Compose**
- **Kubernetes** (manifests pour FibGo)
- **Apache Kafka** (via Confluent)

---

## 📚 Structure du Repository

```
Code/
├── FibGo/                    # Calculateur Fibonacci en Go
│   ├── cmd/                  # Points d'entrée
│   ├── internal/             # Code applicatif privé
│   │   ├── fibonacci/        # Algorithmes de calcul
│   │   ├── bigfft/           # Arithmétique FFT
│   │   ├── server/           # API REST
│   │   └── ...
│   ├── Docs/                 # Documentation détaillée
│   └── Makefile
│
├── FibRust/                  # Calculateur Fibonacci en Rust
│   ├── crates/
│   │   ├── fibrust-core/     # Bibliothèque d'algorithmes
│   │   ├── fibrust-server/   # Serveur HTTP
│   │   └── fibrust-cli/      # Interface CLI
│   └── Cargo.toml
│
├── PubSubKafka/              # Architecture événementielle Kafka
│   ├── cmd/                  # Services (producer, tracker, monitor)
│   ├── pkg/                  # Bibliothèques partagées
│   ├── docker-compose.yaml
│   └── Makefile
│
└── README.md                 # Ce fichier
```

---

## 🎯 Points d'Apprentissage

Ces projets illustrent plusieurs concepts avancés :

### Algorithmique
- Exponentiation rapide et **Fast Doubling**
- **FFT/NTT** pour la multiplication de grands entiers
- Analyse de complexité $O(\log n)$ vs $O(n \log n)$

### Architecture Logicielle
- **Clean Architecture** avec séparation stricte des responsabilités
- **Event-Driven Architecture** avec Kafka
- **Microservices** découplés

### Performance
- Gestion mémoire **zéro-allocation** avec pools
- **Parallélisme adaptatif** selon la charge
- **Auto-calibration** matérielle
- **LTO** et optimisations de compilation

### Observabilité
- Métriques **Prometheus**
- Logging structuré (**zerolog**)
- Dashboards **TUI** temps réel

---

## 📄 Licences

| Projet | Licence |
|--------|---------|
| FibGo | [Apache License 2.0](./FibGo/LICENSE) |
| FibRust | MIT |
| PubSubKafka | [MIT](./PubSubKafka/LICENSE) |

---

## 👤 Auteur

**agbruneau**

- GitHub: [@agbruneau](https://github.com/agbruneau)

---

<div align="center">

**⭐ N'hésitez pas à star ce repository si vous le trouvez utile !**

</div>
