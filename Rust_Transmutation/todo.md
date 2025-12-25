# Todo - Rust Transmutation

## 🔴 Priorité Haute (Phase 1 - Fondations)

- [ ] Créer la structure du Workspace Cargo (`fibcalc-core`, `fibcalc-cli`, `fibcalc-server`)
- [ ] Définir les types partagés (`Options`, `Error`) et l'interface (Trait) `Calculator`
- [ ] Mettre en place le CI/CD de base (fmt, clippy, test)

## 🟠 Priorité Moyenne (Phase 2 - Core Algorithmique)

- [ ] Implémenter le wrapper `num-bigint` et les utilitaires mathématiques de base
- [ ] Implémenter l'algorithme `Fast Doubling` (portage direct de la logique Go)
- [ ] Implémenter l'algorithme `Matrix Exponentiation` avec optimisation des carrés symétriques
- [ ] Implémenter l'algorithme `Strassen` pour la multiplication matricielle récursive
- [ ] Implémenter la multiplication `FFT` (Schönhage-Strassen ou optimisation `num-bigint`)
- [ ] Ajouter la logique de sélection dynamique des algorithmes (`DynamicThresholds`)

## 🟡 Priorité Normale (Phase 3 - CLI)

- [ ] Configurer `clap` pour reproduire exactement les flags Go (`-n`, `-a`, etc.)
- [ ] Implémenter l'affichage avec `indicatif` (Spinner) et la gestion des couleurs
- [ ] Implémenter le mode interactif (REPL)
- [ ] Ajouter la commande de calibration (`--calibrate`) et l'analyse des résultats

## 🟢 Priorité Standard (Phase 4 - Serveur API)

- [ ] Mettre en place `axum` et le routeur HTTP
- [ ] Implémenter les handlers (`calculate_handler`, `health_handler`)
- [ ] Intégrer les métriques Prometheus
- [ ] Ajouter le middleware de Rate Limiting

## 🔵 Phase 5 - Optimisation & Validation

- [ ] Écrire les tests de propriétés (`proptest`) pour vérifier l'identité de Cassini
- [ ] Configurer `criterion` pour les benchmarks comparatifs (Rust vs Go)
- [ ] Profiling et optimisation mémoire (réduire les clones `BigInt`)
- [ ] Validation croisée sur Linux et Windows 11
