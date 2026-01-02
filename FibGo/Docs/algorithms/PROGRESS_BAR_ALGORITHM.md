# Algorithme de Barre de Progression pour Algorithmes O(log n)

## Description

Cet algorithme implémente un système de suivi de progression précis pour des algorithmes de complexité temporelle O(log n), spécifiquement conçu pour des algorithmes qui itèrent sur les bits d'un nombre (comme Fast Doubling, Matrix Exponentiation). Il modélise le travail effectué comme une série géométrique où chaque étape demande approximativement 4 fois plus de travail que la précédente.

## Contexte d'Utilisation

- **Algorithme cible** : Algorithmes O(log n) qui itèrent sur les bits d'un nombre
- **Exemples** : Fast Doubling pour Fibonacci, Matrix Exponentiation
- **Caractéristique clé** : Le travail effectué augmente exponentiellement à mesure que l'algorithme progresse vers les bits de poids faible

## Modèle Mathématique

### Série Géométrique du Travail

L'algorithme modélise le travail total comme une série géométrique :

```
TotalWork = 4^0 + 4^1 + 4^2 + ... + 4^(n-1) = (4^n - 1) / 3
```

Où `n` est le nombre de bits du nombre d'entrée.

### Justification

Les algorithmes O(log n) pour calculer F(n) :
- Commencent par les bits de poids fort (MSB) où les valeurs sont petites
- Progressent vers les bits de poids faible (LSB) où les valeurs deviennent très grandes
- Le travail de multiplication/cálculation quadruple approximativement à chaque étape

**Exemple** : Pour un nombre avec 20 bits (par ex. n = 1,000,000) :
- Bit 19 (MSB) : travail ≈ 4^0 = 1 unité
- Bit 10 : travail ≈ 4^9 = 262,144 unités
- Bit 0 (LSB) : travail ≈ 4^19 = 274,877,906,944 unités

## Composants de l'Algorithme

### 1. Calcul du Travail Total

**Fonction** : `CalcTotalWork(numBits int) float64`

```go
func CalcTotalWork(numBits int) float64 {
    if numBits == 0 {
        return 0
    }
    // Geometric sum: 4^0 + 4^1 + ... + 4^(n-1) = (4^n - 1) / 3
    return (math.Pow(4, float64(numBits)) - 1) / 3
}
```

**Paramètres** :
- `numBits` : Nombre de bits dans le nombre d'entrée

**Retour** :
- Valeur représentant le travail total estimé en unités

**Note** : Retourne 0 si `numBits == 0`

### 2. Précalcul des Puissances de 4

**Fonction** : `PrecomputePowers4(numBits int) []float64`

```go
func PrecomputePowers4(numBits int) []float64 {
    if numBits <= 0 {
        return nil
    }
    powers := make([]float64, numBits)
    powers[0] = 1.0
    for i := 1; i < numBits; i++ {
        powers[i] = powers[i-1] * 4.0
    }
    return powers
}
```

**Optimisation** : Évite les appels répétés à `math.Pow(4, x)` pendant la boucle de calcul, fournissant un accès O(1) au lieu de calculs d'exponentiation coûteux.

**Retour** :
- Slice où `powers[i] = 4^i` pour i de 0 à numBits-1

### 3. Rapport de Progression par Étape

**Fonction** : `ReportStepProgress(...) float64`

**Signature** :
```go
func ReportStepProgress(
    progressReporter ProgressReporter,
    lastReported *float64,
    totalWork float64,
    workDone float64,
    i int,           // Index du bit actuel (numBits-1 vers 0)
    numBits int,
    powers []float64,
) float64
```

**Logique** :

1. **Calcul de l'index de l'étape** :
   ```go
   stepIndex = numBits - 1 - i
   ```
   - Pour `i = numBits - 1` (premier bit, MSB) → `stepIndex = 0` (travail minimal)
   - Pour `i = 0` (dernier bit, LSB) → `stepIndex = numBits - 1` (travail maximal)

2. **Calcul du travail de l'étape** :
   ```go
   workOfStep = powers[stepIndex]  // O(1) lookup
   ```

3. **Calcul du travail cumulé** :
   ```go
   currentTotalDone = workDone + workOfStep
   ```

4. **Calcul de la progression** :
   ```go
   currentProgress = currentTotalDone / totalWork
   ```

5. **Rapport conditionnel** :
   ```go
   if currentProgress - *lastReported >= ProgressReportThreshold || 
      i == 0 || i == numBits - 1 {
       progressReporter(currentProgress)
       *lastReported = currentProgress
   }
   ```

**Seuil de Rapport** : `ProgressReportThreshold = 0.01` (1%)
- Évite les mises à jour excessives
- Rapporte toujours au début (i == numBits-1) et à la fin (i == 0)

**Retour** : Le travail cumulé mis à jour

### 4. Type de Callback

```go
type ProgressReporter func(progress float64)
```

- `progress` : Valeur normalisée de 0.0 à 1.0

## Intégration dans la Boucle de Calcul

### Exemple d'Utilisation

```go
func ExecuteCalculation(ctx context.Context, reporter ProgressReporter, n uint64) (*big.Int, error) {
    numBits := bits.Len64(n)
    
    // Initialisation
    totalWork := CalcTotalWork(numBits)
    powers := PrecomputePowers4(numBits)
    workDone := 0.0
    lastReportedProgress := -1.0  // -1 pour forcer le premier rapport
    
    // Boucle principale : itération sur les bits de numBits-1 vers 0
    for i := numBits - 1; i >= 0; i-- {
        // Vérification d'annulation
        if err := ctx.Err(); err != nil {
            return nil, err
        }
        
        // ... Effectuer le calcul de l'étape (doubling, addition, etc.) ...
        
        // Rapport de progression
        workDone = ReportStepProgress(
            reporter,
            &lastReportedProgress,
            totalWork,
            workDone,
            i,
            numBits,
            powers,
        )
    }
    
    // ... Retourner le résultat ...
}
```

## Propriétés Garanties

1. **Monotonie** : La progression est toujours croissante (ou stable), jamais décroissante
2. **Plage valide** : Les valeurs de progression sont toujours dans [0.0, 1.0]
3. **Finalisation** : La progression finale est toujours proche de 1.0 (≥ 0.99)
4. **Performance** : Pas de calculs d'exponentiation dans la boucle (précalculé)

## Comportement de la Progression

### Caractéristiques

- **Progression lente au début** : Les premières étapes (bits de poids fort) représentent peu de travail
- **Accélération vers la fin** : Les dernières étapes (bits de poids faible) représentent la majorité du travail
- **Distribution** : Pour 20 bits, environ 50% du travail est fait dans les 2-3 dernières étapes

### Exemple Numérique

Pour `numBits = 10` :
- TotalWork ≈ 1,398,101 unités
- Première étape (i=9) : 4^0 = 1 unité → ~0.00007% du total
- Étape médiane (i=5) : 4^4 = 256 unités → ~0.018% du total
- Dernière étape (i=0) : 4^9 = 262,144 unités → ~18.8% du total

## Cas Limites et Validation

### Cas à Gérer

1. **numBits = 0** :
   - `CalcTotalWork(0)` → 0
   - `PrecomputePowers4(0)` → nil

2. **totalWork = 0** :
   - `ReportStepProgress` doit éviter la division par zéro
   - Ne pas rapporter de progression si `totalWork <= 0`

3. **Première et dernière itération** :
   - Toujours rapporter, même si le changement < seuil

### Tests Recommandés

```go
// Test 1: Travail total augmente avec le nombre de bits
func TestCalcTotalWorkMonotonic(t *testing.T) {
    prev := CalcTotalWork(1)
    for bits := 2; bits <= 20; bits++ {
        current := CalcTotalWork(bits)
        assert.True(current > prev)
        prev = current
    }
}

// Test 2: Progression monotone
func TestProgressMonotonic(t *testing.T) {
    numBits := 20
    totalWork := CalcTotalWork(numBits)
    powers := PrecomputePowers4(numBits)
    
    var lastReported float64
    var prevProgress float64
    
    reporter := func(progress float64) {
        assert.True(progress >= prevProgress)
        prevProgress = progress
    }
    
    workDone := 0.0
    for i := numBits - 1; i >= 0; i-- {
        workDone = ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
    }
    
    assert.True(prevProgress >= 0.99)  // Final progress should be close to 1.0
}

// Test 3: Progression initiale faible
func TestProgressStartsSlow(t *testing.T) {
    // Le premier rapport devrait représenter < 25% du travail total
    // car les premières étapes sont très rapides
}
```

## Optimisations

### Performance

1. **Précalcul des puissances** : Évite `math.Pow(4, x)` dans la boucle (coût O(1) vs O(log x))
2. **Seuil de rapport** : Réduit le nombre de callbacks (moins de surcharge d'E/S)
3. **Lookup O(1)** : Utilisation d'un slice pré-calculé au lieu de calculs répétés

### Complexité

- **Temps** : O(numBits) pour le précalcul, O(1) par itération
- **Espace** : O(numBits) pour le tableau des puissances

## Adaptation pour d'Autres Algorithmes

### Modifications Possibles

1. **Facteur de croissance** : Si le travail triple par étape au lieu de quadrupler, utiliser 3 au lieu de 4
2. **Formule alternative** : Pour des algorithmes avec une croissance différente, adapter la formule géométrique
3. **Pondération** : Si certaines étapes prennent plus/moins de temps, ajuster `workOfStep`

### Exemple : Facteur de 3

```go
func CalcTotalWork3(numBits int) float64 {
    if numBits == 0 {
        return 0
    }
    // Geometric sum: 3^0 + 3^1 + ... + 3^(n-1) = (3^n - 1) / 2
    return (math.Pow(3, float64(numBits)) - 1) / 2
}
```

## Interface de Rapport de Progression

### Définition

```go
// Type de callback pour le rapport de progression
type ProgressReporter func(progress float64)
```

### Utilisation dans le Calcul

```go
// Option 1: Callback simple
reporter := func(progress float64) {
    fmt.Printf("Progress: %.2f%%\n", progress*100)
}

// Option 2: Envoi sur un canal (pour UI asynchrone)
progressChan := make(chan ProgressUpdate, 10)
reporter := func(progress float64) {
    select {
    case progressChan <- ProgressUpdate{Value: progress}:
    default:
        // Canal plein, ignorer pour éviter le blocage
    }
}
```

## Constantes Recommandées

```go
const (
    // Seuil minimum de changement de progression avant rapport (1%)
    ProgressReportThreshold = 0.01
    
    // Taux de rafraîchissement de l'affichage (200ms)
    ProgressRefreshRate = 200 * time.Millisecond
    
    // Largeur de la barre de progression en caractères
    ProgressBarWidth = 40
)
```

## Exemple Complet d'Implémentation

Voir `internal/fibonacci/progress.go` et `internal/fibonacci/doubling_framework.go` pour une implémentation complète de référence.

## Résumé des Équations Clés

1. **Travail total** : `TotalWork = (4^numBits - 1) / 3`
2. **Travail par étape** : `WorkOfStep(i) = 4^(numBits - 1 - i)`
3. **Progression** : `Progress = WorkDone / TotalWork`
4. **Condition de rapport** : `currentProgress - lastReported >= 0.01 || i == 0 || i == numBits-1`

## Notes d'Implémentation

- Utiliser des `float64` pour la précision des calculs
- Initialiser `lastReported` à `-1.0` pour forcer le premier rapport
- Valider que `totalWork > 0` avant la division
- Clamper les valeurs de progression dans [0.0, 1.0] si nécessaire
- Gérer les cas où `numBits == 0` ou très petit

---

*Ce document décrit l'algorithme utilisé dans le projet FibGo pour suivre la progression des calculs de nombres de Fibonacci avec une précision mathématique garantie.*