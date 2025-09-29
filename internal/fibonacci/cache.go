// MODULE ACADÉMIQUE : INTÉGRATION DU CACHE
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier sert de "colle" entre le module de calcul `fibonacci` et le
// module de cache générique. Il illustre plusieurs principes clés :
//  1. INITIALISATION AU DÉMARRAGE (`init`) : Utilisation de la fonction `init`
//     de Go pour initialiser de manière transparente une ressource partagée
//     (le cache) au démarrage de l'application. C'est une méthode idiomatique
//     pour gérer des singletons ou des états globaux nécessaires.
//  2. GESTION DES ERREURS AU DÉMARRAGE : Démonstration de la manière de gérer
//     les erreurs critiques qui peuvent survenir lors de l'initialisation.
//     Ici, une erreur de création du cache est considérée comme fatale, et
//     l'application s'arrête avec `log.Fatalf`.
//  3. ENCAPSULATION ET EXPOSITION CONTRÔLÉE : Le cache lui-même (`diskCache`)
//     est une variable non exportée (privée au package). Seules les fonctions
//     nécessaires (`SaveCache`) sont exportées, offrant une API propre et
//     limitant l'accès direct à l'état interne.
//  4. SINGLETON GÉRÉ PAR LE PACKAGE : Le `diskCache` agit comme un singleton
//     au sein du package. Tous les composants du package `fibonacci` peuvent
//     y accéder sans avoir besoin de le passer en paramètre, simplifiant
//     l'injection de cette dépendance transversale.
package fibonacci

import (
	"log"

	"example.com/fibcalc/internal/cache"
)

const (
	// Le nom de notre application, utilisé pour le sous-répertoire du cache.
	AppName = "fibcalc"
	// Le nom du fichier de cache.
	CacheFilename = "fibonacci_cache.json"
)

// diskCache est l'instance unique (singleton) du cache pour toute l'application.
// Elle est initialisée une seule fois au démarrage du programme.
var diskCache *cache.Cache

// init est appelée automatiquement par le runtime Go avant l'exécution de `main`.
// C'est l'endroit idéal pour initialiser des dépendances globales comme le cache.
func init() {
	var err error
	// Créer une nouvelle instance du cache.
	diskCache, err = cache.New(AppName, CacheFilename)
	if err != nil {
		// Si le cache ne peut pas être initialisé, c'est une erreur fatale.
		// L'application ne peut pas fonctionner comme prévu. On logue l'erreur
		// et on arrête le programme.
		log.Fatalf("Erreur critique: impossible d'initialiser le cache sur disque: %v", err)
	}
}

// SaveCache est une fonction exportée qui déclenche la sauvegarde du cache sur le disque.
// Elle sera appelée à la fin de l'exécution du programme pour persister les nouvelles
// entrées calculées pendant la session.
func SaveCache() {
	if diskCache == nil {
		return
	}
	// On logue l'événement pour la visibilité, mais une erreur de sauvegarde n'est
	// pas considérée comme fatale. Le pire qui puisse arriver est que les nouveaux
	// résultats ne soient pas mis en cache pour la prochaine exécution.
	log.Println("Tentative de sauvegarde du cache sur le disque...")
	if err := diskCache.Save(); err != nil {
		log.Printf("Avertissement: n'a pas pu sauvegarder le cache sur le disque: %v", err)
	}
}