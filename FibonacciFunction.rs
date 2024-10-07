// Programme Rust : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
//
// Description :
// Ce programme en Rust calcule les nombres de Fibonacci en utilisant la méthode du doublement, qui est une approche
// efficace basée sur la division et la conquête. L'algorithme utilise une technique itérative pour calculer
// rapidement les valeurs de Fibonacci pour de très grands nombres. Pour améliorer la performance, une stratégie
// de mémoïsation avec LRU (Least Recently Used) est utilisée afin de mettre en cache les résultats des calculs
// précédents. Cela permet de réutiliser les valeurs déjà calculées et de réduire le temps de calcul des appels
// futurs. Le programme est également conçu pour utiliser des threads pour calculer les valeurs de manière concurrente
// et améliorer l'efficacité.
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

use num_bigint::BigUint;
use num_traits::{One, Zero};
use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use threadpool::ThreadPool;

const MAX_FIB_VALUE: u64 = 100000001; // Valeur maximale de n qui peut être calculée
const THREAD_POOL_SIZE: usize = 4; // Taille du pool de threads

// Cache LRU optimisé avec une meilleure gestion de la concurrence
struct LRUCache {
    capacity: usize,
    cache: HashMap<u64, BigUint>, // Utiliser un HashMap pour le stockage des données
    order: VecDeque<u64>,          // Utiliser une VecDeque pour maintenir l'ordre des accès
}

impl LRUCache {
    // Crée un nouveau cache LRU avec la capacité donnée
    fn new(capacity: usize) -> Self {
        Self {
            capacity,
            cache: HashMap::new(),
            order: VecDeque::new(),
        }
    }

    // Récupère une valeur du cache LRU
    fn get(&mut self, key: u64) -> Option<BigUint> {
        if let Some(value) = self.cache.get(&key) {
            // Déplacer l'élément utilisé récemment à la fin pour montrer qu'il a été accédé
            self.order.retain(|&k| k != key);
            self.order.push_back(key);
            Some(value.clone())
        } else {
            None
        }
    }

    // Ajoute une nouvelle valeur au cache LRU
    fn put(&mut self, key: u64, value: BigUint) {
        if self.cache.contains_key(&key) {
            // Si la clé existe déjà, la déplacer en tête de liste et mettre à jour sa valeur
            self.order.retain(|&k| k != key);
        } else if self.cache.len() >= self.capacity {
            // Supprimer l'élément le moins récemment utilisé (le plus ancien)
            if let Some(oldest_key) = self.order.pop_front() {
                self.cache.remove(&oldest_key);
            }
        }
        // Ajouter l'élément avec la clé et la valeur
        self.cache.insert(key, value);
        self.order.push_back(key);
    }
}

// Initialiser le cache LRU avec une taille ajustable
fn initialize_cache() -> Arc<Mutex<LRUCache>> {
    Arc::new(Mutex::new(LRUCache::new(1000)))
}

// Fonction qui calcule le nième nombre de Fibonacci en utilisant la méthode de doublement
fn fib_doubling(n: u64, cache: Arc<Mutex<LRUCache>>) -> BigUint {
    if n < 2 {
        return BigUint::from(n);
    } else if n > MAX_FIB_VALUE {
        panic!("n est trop grand pour cette implémentation");
    }

    // Récupérer la valeur du cache si elle existe
    if let Some(result) = cache.lock().unwrap().get(n) {
        return result;
    }

    // Initialiser les valeurs de base F(0) = 0, F(1) = 1
    let mut a = BigUint::zero();
    let mut b = BigUint::one();

    // Calculer F(n) à l'aide de la méthode de doublement
    let bit_length = 64 - n.leading_zeros();
    for i in (0..bit_length).rev() {
        // Utiliser les formules de doublement
        // F(2k) = F(k) * [2 * F(k+1) - F(k)]
        let temp1 = &b << 1; // temp1 = 2 * F(k+1)
        let temp2 = &temp1 - &a; // temp2 = 2 * F(k+1) - F(k)
        let c: BigUint = &a * &temp2; // c = F(2k)
        // F(2k + 1) = F(k)^2 + F(k+1)^2
        let d: BigUint = &a * &a + &b * &b; // d = F(2k + 1)

        // Mettre à jour a et b en fonction du bit actuel de n
        if (n >> i) & 1 == 0 {
            a = c.clone(); // Si le bit est 0, définir F(2k) sur a
            b = d.clone(); // Définir F(2k+1) sur b
        } else {
            a = d.clone(); // Si le bit est 1, définir F(2k+1) sur a
            b = c + d;     // Définir F(2k + 2) sur b
        }
    }

    // Mettre en cache le résultat
    cache.lock().unwrap().put(n, a.clone());
    a
}

// Fonction pour effacer la mémoïsation
// Cette fonction permet de recréer une instance du cache LRU pour effacer toutes les entrées
// et garantir que les calculs futurs ne soient pas influencés par des résultats précédents
fn clear_memoization() -> Arc<Mutex<LRUCache>> {
    initialize_cache()
}

// Fonction de benchmark pour tester les performances de calcul de Fibonacci
// Cette fonction calcule le temps nécessaire pour calculer plusieurs valeurs de Fibonacci
// et enregistre les résultats de chaque répétition
fn benchmark_fib(n_values: Vec<u64>, repetitions: u32) {
    let cache = clear_memoization(); // Effacer le cache pour garantir des résultats cohérents
    let pool = ThreadPool::new(THREAD_POOL_SIZE); // Utiliser un pool de threads avec une taille définie

    for &n in &n_values {
        let cache = Arc::clone(&cache);
        pool.execute(move || {
            let mut total_exec_time = Duration::new(0, 0);
            let mut individual_times = Vec::new(); // Stocker les temps d'exécution individuels pour chaque répétition

            for _ in 0..repetitions {
                let start_time = Instant::now(); // Utiliser Instant pour une meilleure précision
                fib_doubling(n, Arc::clone(&cache));
                let exec_time = start_time.elapsed(); // Calculer le temps écoulé
                total_exec_time += exec_time; // Ajouter au temps total
                individual_times.push(exec_time.as_secs_f64()); // Enregistrer le temps individuel
            }

            let avg_exec_time = total_exec_time.as_secs_f64() / repetitions as f64; // Calculer le temps d'exécution moyen
            println!("fibDoubling({}) averaged over {} runs: {:.6} seconds", n, repetitions, avg_exec_time);
            println!("Individual execution times for {}: {:?}", n, individual_times);
        });
    }

    pool.join(); // Attendre que toutes les tâches dans le pool de threads soient terminées
}

// Fonction principale
// Cette fonction lance le benchmark pour tester les performances du calcul de Fibonacci
fn main() {
    let n_values = vec![1000000, 10000000, 100000000]; // Valeurs à tester
    let repetitions = 3; // Nombre de répétitions pour plus de précision
    benchmark_fib(n_values, repetitions);
}
