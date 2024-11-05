# Calculateur de Fibonacci Parallélisé

![Diagramme de Séquence](SequenceDiagram.jpeg)

## Description
Ce projet est une implémentation hautement optimisée d'un calculateur de nombres de Fibonacci en Go, utilisant une approche matricielle parallélisée. Il est capable de calculer efficacement de très grands nombres de la suite de Fibonacci grâce à plusieurs optimisations :

- Utilisation de la méthode matricielle (complexité O(log n))
- Parallélisation des calculs
- Gestion de grands nombres avec `math/big`
- Pool de workers réutilisables
- Métriques de performance détaillées

## Caractéristiques Principales
- ⚡ Calcul rapide grâce à l'algorithme d'exponentiation matricielle
- 🔄 Parallélisation automatique sur tous les cœurs disponibles
- 📊 Métriques de performance détaillées
- ⏱️ Gestion des timeouts
- 🔒 Thread-safe
- 💾 Gestion efficace de la mémoire
- 📈 Support des très grands nombres

## Prérequis
- Go 1.16 ou supérieur
- Package `github.com/pkg/errors`

## Installation

```bash
# Cloner le repository
git clone https://github.com/votre-username/fibonacci-calculator
cd fibonacci-calculator

# Installer les dépendances
go mod tidy
```

## Utilisation

### Compilation et Exécution
```bash
# Compiler le programme
go build -o fib-calc

# Exécuter avec la configuration par défaut
./fib-calc
```

### Configuration
Le programme peut être configuré en modifiant les valeurs dans `DefaultConfig()` :

```go
func DefaultConfig() Configuration {
    return Configuration{
        M:           100000,           // Nombre maximum de termes
        NumWorkers:  runtime.NumCPU(), // Nombre de workers
        SegmentSize: 1000,             // Taille des segments
        Timeout:     5 * time.Minute,  // Timeout global
    }
}
```

### Sortie
Le programme affiche :
- La configuration utilisée
- Les métriques de performance
- La somme des nombres de Fibonacci calculés (en notation scientifique)

Exemple de sortie :
```
Configuration:
  Nombre de workers: 8
  Taille des segments: 1000
  Valeur de m: 100000

Performance:
  Temps total d'exécution: 2m15s
  Nombre de calculs: 100000
  Temps moyen par calcul: 1.35ms

Résultat:
  Somme des Fibonacci(0..100000): 1.234e20089
```

## Architecture Technique

### Composants Principaux

1. **Matrix2x2**
   - Représente une matrice 2x2 pour le calcul matriciel
   - Utilise `big.Int` pour la précision infinie

2. **FibCalculator**
   - Implémente l'algorithme d'exponentiation matricielle
   - Thread-safe avec mutex intégré
   - Réutilise les matrices pour optimiser la mémoire

3. **WorkerPool**
   - Gère un pool de calculateurs réutilisables
   - Distribution round-robin des calculateurs
   - Évite la création/destruction excessive d'objets

4. **Metrics**
   - Collecte les métriques de performance
   - Thread-safe pour les accès concurrents
   - Calcule les statistiques d'exécution

### Algorithme Matriciel
La méthode utilise la propriété suivante :
```
[1 1]^n = [F(n+1) F(n)  ]
[1 0]    [F(n)   F(n-1)]
```

L'exponentiation rapide permet d'obtenir une complexité de O(log n).

## Performance

Les performances dépendent de plusieurs facteurs :
- Nombre de cœurs CPU disponibles
- Taille des segments de calcul
- Nombres de Fibonacci à calculer
- Mémoire système disponible

Optimisations clés :
- Réutilisation des objets `big.Int`
- Parallélisation automatique
- Algorithme d'exponentiation rapide
- Pool de workers

## Limitations

- Limite pratique sur n ≈ 1,000,000 pour des raisons de performance
- Consommation mémoire proportionnelle à la taille des nombres
- Précision limitée par la mémoire disponible

## Dépannage

### Erreurs Communes

1. **Timeout**
   - Augmenter la valeur de `Timeout` dans la configuration
   - Réduire la valeur de `M`
   - Augmenter la taille des segments

2. **Mémoire Insuffisante**
   - Réduire le nombre de workers
   - Diminuer la taille des segments
   - Réduire la valeur de `M`

3. **Performance Faible**
   - Vérifier la charge CPU
   - Ajuster la taille des segments
   - Optimiser le nombre de workers

## Contribution

Les contributions sont bienvenues ! Voici comment contribuer :

1. Forker le projet
2. Créer une branche pour votre fonctionnalité
3. Commiter vos changements
4. Pousser vers la branche
5. Créer une Pull Request

## Licence
MIT License

## Contact
Pour toute question ou suggestion, n'hésitez pas à ouvrir une issue sur GitHub.

## Changelog

### v1.0.0
- Implémentation initiale avec méthode matricielle
- Support de la parallélisation
- Métriques de performance
- Documentation complète