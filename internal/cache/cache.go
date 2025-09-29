// MODULE : CACHE
//
// OBJECTIF PÉDAGOGIQUE :
// Ce module encapsule la logique de mise en cache des résultats de Fibonacci en utilisant
// une base de données clé-valeur embarquée (BoltDB). Il illustre :
//  1. L'ABSTRACTION : Il fournit une interface simple (`Get`, `Set`, `Close`) qui cache
//     complètement les détails d'implémentation de la base de données sous-jacente.
//  2. LA GESTION DU CYCLE DE VIE : Le constructeur `New` et la méthode `Close` permettent
//     de gérer proprement les ressources (le fichier de base de données).
//  3. LA SÉRIALISATION : Il montre comment sérialiser des types de données complexes
//     comme `math/big.Int` pour les stocker sur disque.

package cache

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.etcd.io/bbolt"
)

// bucketName est le nom du "bucket" (similaire à une table) dans la base de données BoltDB.
var bucketName = []byte("fibonacciCache")

// Cache gère l'accès à une base de données BoltDB pour la mise en cache.
type Cache struct {
	db *bbolt.DB
}

// New initialise et ouvre une nouvelle base de données de cache au chemin spécifié.
// Il crée le répertoire et le bucket nécessaires s'ils n'existent pas.
func New(dbPath string) (*Cache, error) {
	// S'assurer que le répertoire de la base de données existe.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("impossible de créer le répertoire du cache : %w", err)
	}

	// Ouvre la base de données BoltDB. Elle sera créée si elle n'existe pas.
	// Un timeout est défini pour éviter de bloquer indéfiniment si un autre processus détient le verrou.
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir la base de données du cache (%s) : %w", dbPath, err)
	}

	// Crée le bucket s'il n'existe pas déjà dans une transaction en écriture.
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return fmt.Errorf("impossible de créer le bucket du cache : %w", err)
		}
		return nil
	})
	if err != nil {
		// Si la création du bucket échoue, fermer la DB pour éviter les fuites de ressources.
		db.Close()
		return nil, err
	}

	return &Cache{db: db}, nil
}

// Close ferme proprement la connexion à la base de données.
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Get récupère une valeur `*big.Int` pour un indice `n` donné depuis le cache.
// Retourne la valeur, un booléen `true` si trouvée, et une erreur si un problème survient.
func (c *Cache) Get(n uint64) (*big.Int, bool, error) {
	var valueBytes []byte
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		key := []byte(strconv.FormatUint(n, 10))

		// Récupère la valeur. `v` est un slice qui n'est valide que pendant la durée de la transaction.
		v := b.Get(key)
		if v == nil {
			return nil // Non trouvé, ce n'est pas une erreur.
		}

		// Il est crucial de copier les bytes, car le slice `v` pointe vers une mémoire
		// qui ne sera plus valide après la fin de la transaction.
		valueBytes = make([]byte, len(v))
		copy(valueBytes, v)
		return nil
	})

	if err != nil {
		return nil, false, fmt.Errorf("erreur lors de la lecture du cache : %w", err)
	}

	if valueBytes == nil {
		return nil, false, nil // La clé n'a pas été trouvée dans le cache.
	}

	// Désérialise les bytes (texte) en un `*big.Int`.
	value := new(big.Int)
	err = value.UnmarshalText(valueBytes)
	if err != nil {
		// Si les données sont corrompues, on le signale.
		return nil, false, fmt.Errorf("erreur lors de la désérialisation de la valeur du cache : %w", err)
	}

	return value, true, nil
}

// Set stocke une paire (n, value) dans le cache.
func (c *Cache) Set(n uint64, value *big.Int) error {
	err := c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		key := []byte(strconv.FormatUint(n, 10))

		// Sérialise la valeur `*big.Int` en sa représentation textuelle (bytes).
		valueBytes, err := value.MarshalText()
		if err != nil {
			return fmt.Errorf("erreur lors de la sérialisation de la valeur pour le cache : %w", err)
		}

		// Stocke la paire clé-valeur dans le bucket.
		return b.Put(key, valueBytes)
	})

	if err != nil {
		return fmt.Errorf("erreur lors de l'écriture dans le cache : %w", err)
	}
	return nil
}