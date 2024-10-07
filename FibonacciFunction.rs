// Programme Rust : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
//
// Description :
// Ce programme en Rust calcule les nombres de Fibonacci en utilisant la méthode du doublement, qui est une approche
// efficace basée sur la division et la conquête. L'algorithme utilise une technique itérative pour calculer
// rapidement les valeurs de Fibonacci pour de très grands nombres. Pour améliorer la performance, une stratégie
// de mémoïsation avec LRU (Least Recently Used) est utilisée afin de mettre en cache les résultats des calculs
// précédents. Cela permet de réutiliser les valeurs déjà calculées et de réduire le temps de calcul des appels
// futurs. De plus, le programme est conçu pour utiliser la concurrence, ce qui permet un calcul concurrent et
// améliore l'efficacité en utilisant plusieurs threads.
//
// Algorithme de Doublement :
// L'algorithme de doublement repose sur les propriétés suivantes des nombres de Fibonacci :
// - F(2k) = F(k) * [2 * F(k+1) - F(k)]
// - F(2k + 1) = F(k)^2 + F(k+1)^2
// Ces formules permettent de calculer des valeurs de Fibonacci en utilisant une approche binaire sur les bits
// de l'indice n, rendant l'algorithme très performant pour de grands nombres.
//
// Le programme effectue également des tests de performance (benchmark) sur des valeurs élevées de Fibonacci
// et affiche le temps moyen d'exécution pour chaque valeur, en utilisant des répétitions multiples pour
// une meilleure précision.

use std::collections::HashMap;
use std::sync::{Arc, Mutex, PoisonError};
use std::time::{Duration, Instant};
use std::thread;
use std::sync::mpsc;

const MAX_FIB_VALUE: u64 = 500_000_001; // Valeur maximale de n qui peut être calculée

// Initialiser le cache LRU avec une Mutex pour garantir la sécurité concurrente
struct LruCache {
    cache: Mutex<HashMap<u64, num::BigUint>>, // Utilisation d'un Mutex pour la sécurité des threads
}

impl LruCache {
    // Crée une nouvelle instance de LruCache
    fn new() -> LruCache {
        LruCache {
            cache: Mutex::new(HashMap::new()),
        }
    }

    // Récupère une valeur du cache si elle existe
    fn get(&self, n: u64) -> Option<num::BigUint> {
        match self.cache.lock() {
            Ok(cache) => cache.get(&n).cloned(),
            Err(PoisonError { .. }) => {
                // Gérer un Mutex empoisonné en affichant un avertissement
                eprintln!("Warning: Mutex was poisoned while attempting to get value for n = {}", n);
                None
            }
        }
    }

    // Ajoute une valeur au cache
    fn add(&self, n: u64, value: num::BigUint) {
        match self.cache.lock() {
            Ok(mut cache) => {
                cache.insert(n, value);
            }
            Err(PoisonError { .. }) => {
                // Gérer un Mutex empoisonné en affichant un avertissement
                eprintln!("Warning: Mutex was poisoned while attempting to add value for n = {}", n);
            }
        }
    }
}

lazy_static::lazy_static! {
    // Utilisation de lazy_static pour initialiser un cache global LRU partagé entre les threads
    static ref LRU_CACHE: Arc<LruCache> = Arc::new(LruCache::new());
}

// Fonction qui calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
fn fib_doubling(n: u64) -> Result<num::BigUint, &'static str> {
    if n < 2 {
        return Ok(num::BigUint::from(n)); // Retourner directement n si n est inférieur à 2
    } else if n > MAX_FIB_VALUE {
        return Err("n est trop grand pour cette implémentation"); // Limiter les calculs à une valeur maximale raisonnable
    }
    // Vérifier si la valeur est déjà dans le cache
    if let Some(value) = LRU_CACHE.get(n) {
        return Ok(value);
    }
    // Calculer le résultat si non mis en cache
    let result = fib_doubling_helper_iterative(n);
    // Ajouter le résultat au cache
    LRU_CACHE.add(n, result.clone());
    Ok(result)
}

// Fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
fn fib_doubling_helper_iterative(n: u64) -> num::BigUint {
    use num::BigUint;
    let mut a = BigUint::from(0u64); // F(0)
    let mut b = BigUint::from(1u64); // F(1)
    let mut c; // Variable temporaire pour les calculs intermédiaires
    let mut d; // Variable temporaire pour les calculs intermédiaires
    
    let two = BigUint::from(2u64); // Constante 2 pour les calculs

    let bit_length = 64 - n.leading_zeros(); // Déterminer le nombre de bits significatifs de n

    // Itérer sur chaque bit de `n` en partant du plus significatif.
    // Cela permet de déterminer les valeurs de Fibonacci en utilisant une approche binaire (doublement).
    // L'algorithme exploite les bits de l'indice pour construire efficacement la séquence de Fibonacci.
    for i in (0..bit_length).rev() {
        // Calculer F(2k) et F(2k + 1) en utilisant les formules de doublage
        c = &b * &two - &a; // c = 2 * F(k+1) - F(k)
        c = &a * &c; // c = F(k) * (2 * F(k+1) - F(k))
        d = &a * &a + &b * &b; // d = F(k)^2 + F(k+1)^2

        if (n >> i) & 1 == 0 {
            // Si le bit est 0, mettre à jour a et b pour F(2k) et F(2k+1)
            a = c.clone();
            b = d.clone();
        } else {
            // Si le bit est 1, mettre à jour a et b pour F(2k+1) et F(2k+2)
            a = d.clone();
            b = c + d;
        }
    }
    a // Retourner la valeur de F(n)
}

// Fonction pour afficher un message d'erreur dans un format cohérent
fn print_error(n: u64, err: &str) {
    println!("fib_doubling({}): {}", n, err); // Afficher le message d'erreur avec la valeur de n
}

// Fonction pour effectuer des tests de performance sur les calculs de Fibonacci pour une liste de valeurs
fn benchmark_fib_with_worker_pool(n_values: Vec<u64>, repetitions: u32, worker_count: usize) {
    let (tx, rx) = mpsc::channel(); // Canal pour envoyer les résultats des threads
    let n_values = Arc::new(n_values); // Partager la liste des valeurs entre les threads

    // Lancer les workers en parallèle
    for _ in 0..worker_count {
        let tx = tx.clone();
        let n_values = Arc::clone(&n_values);
        thread::spawn(move || {
            for &n in n_values.iter() {
                let mut total_exec_time = Duration::default(); // Initialiser la durée totale à zéro
                for _ in 0..repetitions {
                    let start = Instant::now(); // Démarrer le chronomètre
                    match fib_doubling(n) {
                        Ok(_) => {
                            total_exec_time += start.elapsed(); // Ajouter le temps écoulé
                        }
                        Err(err) => {
                            print_error(n, err); // Afficher l'erreur si le calcul échoue
                            return;
                        }
                    }
                }
                let avg_exec_time = total_exec_time / repetitions; // Calculer le temps d'exécution moyen
                if let Err(err) = tx.send((n, avg_exec_time)) {
                    // Gérer les erreurs potentielles lors de l'envoi des résultats
                    eprintln!("Warning: Failed to send result for n = {}: {}", n, err);
                }
            }
        });
    }

    // Recevoir et afficher les résultats
    drop(tx); // Fermer le canal pour indiquer qu'aucun autre résultat ne sera envoyé
    for (n, avg_exec_time) in rx.iter() {
        // Afficher les résultats des calculs de Fibonacci
        println!("fib_doubling({}) averaged over {} runs: {:.2?}", n, repetitions, avg_exec_time);
    }
}

// Fonction principale pour exécuter les tests de performance
fn main() {
    // Définir la liste des valeurs pour lesquelles effectuer les tests de performance
    let n_values = vec![100_000, 500_000, 1_000_000, 5_000_000, 10_000_000, 50_000_000, 100_000_000, 500_000_000];
    let repetitions = 100; // Nombre de répétitions pour calculer le temps moyen
    let worker_count = 16; // Nombre de threads concurrents

    // Exécuter le benchmark
    benchmark_fib_with_worker_pool(n_values, repetitions, worker_count);
}

extern crate lazy_static;
extern crate num;

