// MODULE ACADÉMIQUE : CACHE PERSISTANT SUR DISQUE
//
// OBJECTIF PÉDAGOGIQUE :
// Ce module implémente un cache clé-valeur simple et persistant sur le disque
// sous forme de fichier JSON. Il illustre des concepts importants pour la
// robustesse et la performance des applications :
//  1. SÉPARATION DES PRÉOCCUPATIONS : La logique de cache est entièrement
//     encapsulée ici, découplée du reste de l'application (calculs, UI).
//  2. SÉCURITÉ EN CONCURRENCE (THREAD-SAFETY) : Utilisation de `sync.RWMutex`
//     pour protéger l'accès concurrentiel à la map du cache. Le `RWMutex` est
//     particulièrement performant ici, car il permet des lectures simultanées
//     illimitées (le cas le plus fréquent) et ne verrouille l'accès que pour
//     les écritures (plus rares).
//  3. GESTION DES ERREURS ROBUSTE : Les opérations de lecture/écriture sur le
//     système de fichiers sont gérées avec soin, propageant les erreurs de
//     manière claire.
//  4. ABSTRACTION DU SYSTÈME DE FICHIERS : Le chemin du cache est géré de
//     manière portable grâce au package `os`, en particulier `os.UserCacheDir`.
//  5. MARSHALLING/UNMARSHALLING DE DONNÉES : Utilisation de `encoding/json`
//     pour sérialiser et désérialiser la structure de données du cache.
package cache

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
)

// Cache est une structure de cache thread-safe qui persiste les résultats
// sur le disque.
type Cache struct {
	// RWMutex est utilisé pour protéger l'accès concurrentiel à la map `data`.
	// Il permet de multiples lecteurs simultanés (`RLock`) mais un seul
	// écrivain (`Lock`), ce qui est idéal pour un cache (beaucoup de lectures,
	// peu d'écritures).
	mu   sync.RWMutex
	data map[uint64]string // Stocke F(n) comme une chaîne pour une sérialisation JSON simple.
	path string            // Chemin complet vers le fichier de cache.
}

// New crée et initialise un nouveau cache à partir d'un fichier sur le disque.
// Si le fichier n'existe pas, un cache vide est créé.
// `appName` est utilisé pour créer un sous-répertoire spécifique à l'application
// dans le répertoire de cache de l'utilisateur.
func New(appName, cacheFile string) (*Cache, error) {
	// Obtenir le répertoire de cache standard de l'OS de manière portable.
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("impossible de trouver le répertoire de cache utilisateur: %w", err)
	}

	// Créer un sous-répertoire pour notre application pour éviter de polluer
	// le répertoire de cache racine.
	appCacheDir := filepath.Join(cacheDir, appName)
	if err := os.MkdirAll(appCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("impossible de créer le répertoire de cache de l'application: %w", err)
	}

	path := filepath.Join(appCacheDir, cacheFile)
	c := &Cache{
		data: make(map[uint64]string),
		path: path,
	}

	// Tenter de charger les données existantes. Si le fichier n'existe pas,
	// c'est normal, on continue avec un cache vide.
	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("échec du chargement du cache depuis %s: %w", path, err)
	}

	return c, nil
}

// load lit et décode le fichier de cache JSON.
func (c *Cache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(c.path)
	if err != nil {
		return err // Retourne l'erreur (ex: os.IsNotExist) pour être gérée par l'appelant.
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&c.data)
}

// Save écrit l'état actuel du cache dans le fichier JSON.
func (c *Cache) Save() error {
	c.mu.RLock() // Un verrou en lecture suffit car on ne modifie pas la map, on la lit seulement.
	defer c.mu.RUnlock()

	// Créer un fichier temporaire pour une écriture atomique.
	// Cela évite de corrompre le fichier de cache existant si l'écriture est interrompue.
	tempFile, err := os.CreateTemp(filepath.Dir(c.path), "cache-*.tmp")
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier de cache temporaire: %w", err)
	}
	// `closed` et `renamed` sont utilisés pour s'assurer que le nettoyage ne se produit
	// que si l'opération a échoué.
	closed := false
	renamed := false
	defer func() {
		if !closed {
			tempFile.Close()
		}
		if !renamed {
			os.Remove(tempFile.Name()) // Nettoyer le fichier temporaire en cas d'erreur.
		}
	}()

	// Utiliser un `io.MultiWriter` pour encoder le JSON à la fois dans le fichier
	// et dans un `json.Encoder` qui ne fait rien (pour la validation).
	// C'est une astuce pour utiliser `json.Encoder` avec un `os.File`.
	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ") // Pour un JSON lisible par l'homme.
	if err := encoder.Encode(c.data); err != nil {
		return fmt.Errorf("erreur d'encodage JSON: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("impossible de fermer le fichier de cache temporaire: %w", err)
	}
	closed = true

	// Remplacer l'ancien fichier de cache par le nouveau. C'est une opération atomique
	// sur la plupart des systèmes de fichiers POSIX.
	if err := os.Rename(tempFile.Name(), c.path); err != nil {
		return fmt.Errorf("impossible de remplacer le fichier de cache: %w", err)
	}
	renamed = true

	return nil
}

// Get récupère une valeur du cache pour la clé `n`.
// Il retourne la valeur, et un booléen indiquant si la clé a été trouvée.
func (c *Cache) Get(n uint64) (*big.Int, bool) {
	c.mu.RLock()
	valStr, ok := c.data[n]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Tenter de convertir la chaîne stockée en `*big.Int`.
	// Si la conversion échoue, cela signifie que le cache est corrompu pour cette entrée.
	// On traite cela comme un "cache miss" pour forcer le recalcul.
	val, success := new(big.Int).SetString(valStr, 10)
	if !success {
		return nil, false // Entrée de cache corrompue.
	}

	return val, true
}

// Set ajoute une nouvelle valeur au cache.
func (c *Cache) Set(n uint64, val *big.Int) {
	if val == nil {
		return
	}

	c.mu.Lock()
	c.data[n] = val.String()
	c.mu.Unlock()
}