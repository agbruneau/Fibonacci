# Documentation de l'API REST

Ce document décrit les endpoints disponibles dans l'API REST du Calculateur Fibonacci.

## Endpoints

### 1. Calculer un nombre de Fibonacci

Effectue le calcul du Nième nombre de Fibonacci en utilisant l'algorithme spécifié.

**URL** : `/calculate`
**Méthode** : `GET`

#### Paramètres de Requête (Query Parameters)

| Paramètre | Type   | Requis | Description |
|-----------|--------|--------|-------------|
| `n`       | int    | Oui    | L'index du nombre de Fibonacci à calculer (doit être positif). |
| `algo`    | string | Non    | L'algorithme à utiliser. Défaut: `fast`. Valeurs possibles : `fast`, `matrix`, `fft`. |

#### Réponse de Succès (200 OK)

```json
{
  "n": 100,
  "result": 354224848179261915075,
  "duration": "125.5µs",
  "algorithm": "fast"
}
```

#### Réponse d'Erreur (400 Bad Request)

```json
{
  "error": "Bad Request",
  "message": "Invalid 'n' parameter: must be a positive integer"
}
```

---

### 2. Vérification de Santé (Health Check)

Vérifie si le serveur est en ligne et fonctionnel.

**URL** : `/health`
**Méthode** : `GET`

#### Réponse de Succès (200 OK)

```json
{
  "status": "healthy",
  "timestamp": 1678886400
}
```

---

### 3. Lister les Algorithmes

Renvoie la liste des algorithmes de calcul disponibles sur le serveur.

**URL** : `/algorithms`
**Méthode** : `GET`

#### Réponse de Succès (200 OK)

```json
{
  "algorithms": [
    "fast",
    "fft",
    "matrix"
  ]
}
```
