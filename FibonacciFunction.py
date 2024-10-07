# Programme Python : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
#
# Description :
# Ce programme en Python calcule les nombres de Fibonacci en utilisant la méthode du doublement, qui est une approche
# efficace basée sur la division et la conquête. L'algorithme utilise une technique itérative pour calculer
# rapidement les valeurs de Fibonacci pour de très grands nombres. Pour améliorer la performance, une stratégie
# de mémoïsation avec un cache LRU (Least Recently Used) est utilisée afin de mettre en cache les résultats des calculs
# précédents. Cela permet de réutiliser les valeurs déjà calculées et de réduire le temps de calcul des appels
# futurs. Le programme est également conçu pour mesurer les performances à l'aide de benchmarks, enregistrant
# les temps d'exécution pour diverses valeurs de Fibonacci.
#
# Algorithme de Doublement :
# L'algorithme de doublement repose sur les propriétés suivantes des nombres de Fibonacci :
# - F(2k) = F(k) * [2 * F(k+1) - F(k)]
# - F(2k + 1) = F(k)^2 + F(k+1)^2
# Ces formules permettent de calculer des valeurs de Fibonacci en utilisant une approche binaire sur les bits
# de l'indice n, rendant l'algorithme très performant pour de grands nombres.
#
# Le programme effectue également des tests de performance (benchmark) sur des valeurs élevées de Fibonacci
# et affiche le temps moyen d'exécution pour chaque valeur, en utilisant des répétitions multiples pour
# une meilleure précision. Les résultats des benchmarks sont loggés pour permettre une analyse plus approfondie
# des performances.

import time
from collections import OrderedDict, deque
from functools import lru_cache
from threading import RLock
import logging

# Configurer le logging pour capturer les informations de performance
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

MAX_FIB_VALUE = 100000001  # Valeur maximale de n qui peut être calculée
two = 2  # Valeur constante 2 pour les calculs

# Cache LRU optimisé avec une meilleure gestion de la concurrence et une approche plus évolutive
class LRUCache:
    def __init__(self, capacity):
        self.capacity = capacity
        self.cache = {}  # Utiliser un dictionnaire pour le stockage des données
        self.order = deque()  # Utiliser une deque pour maintenir l'ordre des accès
        self.lock = RLock()  # Verrou pour la gestion de la concurrence

    def get(self, key):
        with self.lock:
            if key in self.cache:
                # Déplacer l'élément utilisé récemment à la fin pour montrer qu'il a été accédé
                self.order.remove(key)
                self.order.append(key)
                return self.cache[key]
            return None

    def put(self, key, value):
        with self.lock:
            if key in self.cache:
                # Supprimer l'élément existant pour le rafraîchir
                self.order.remove(key)
            elif len(self.cache) >= self.capacity:
                # Supprimer l'élément le moins récemment utilisé (le plus ancien)
                oldest_key = self.order.popleft()
                del self.cache[oldest_key]
            # Ajouter l'élément avec la clé et la valeur
            self.cache[key] = value
            self.order.append(key)

# Initialiser le cache LRU avec une taille ajustable
lru_cache = LRUCache(1000)

# Fonction qui calcule le nième nombre de Fibonacci en utilisant la méthode de doublement
def fib_doubling(n):
    if n < 2:
        return n
    elif n > MAX_FIB_VALUE:
        raise ValueError("n est trop grand pour cette implémentation")

    # Récupérer la valeur du cache si elle existe
    cached_result = lru_cache.get(n)
    if cached_result is not None:
        return cached_result

    # Initialiser les valeurs de base F(0) = 0, F(1) = 1
    a, b = 0, 1

    # Calculer F(n) à l'aide de la méthode de doublement
    bit_length = n.bit_length()
    for i in reversed(range(bit_length)):
        # Utiliser les formules de doublement
        # Calcul temporaire pour rendre le code plus lisible et éviter des erreurs
        temp1 = 2 * b  # temp1 = 2 * F(k+1)
        temp2 = temp1 - a  # temp2 = 2 * F(k+1) - F(k)
        c = a * temp2  # c = F(2k) = F(k) * [2 * F(k+1) - F(k)]
        d = a * a + b * b  # d = F(2k + 1) = F(k)^2 + F(k+1)^2
        if (n >> i) & 1 == 0:
            # Si le bit est 0, mettre à jour a et b
            a, b = c, d
        else:
            # Si le bit est 1, mettre à jour a et b
            a, b = d, c + d

    # Mettre en cache le résultat
    lru_cache.put(n, a)
    return a

# Fonction pour effacer la mémoïsation
# Cette fonction permet de recréer une instance du cache LRU pour effacer toutes les entrées
# et garantir que les calculs futurs ne soient pas influencés par des résultats précédents
def clear_memoization():
    global lru_cache
    lru_cache = LRUCache(1000)  # Créer une nouvelle instance pour effacer le cache

# Fonction de benchmark pour tester les performances de calcul de Fibonacci
# Cette fonction calcule le temps nécessaire pour calculer plusieurs valeurs de Fibonacci
# et enregistre les résultats de chaque répétition
def benchmark_fib(n_values, repetitions):
    clear_memoization()  # Effacer le cache pour garantir des résultats cohérents
    
    for n in n_values:
        total_exec_time = 0
        individual_times = []  # Stocker les temps d'exécution individuels pour chaque répétition
        for _ in range(repetitions):
            start_time = time.perf_counter()  # Utiliser time.perf_counter() pour une meilleure précision
            try:
                fib_doubling(n)
            except ValueError as e:
                logging.error(f"fibDoubling({n}): {e}")
                continue
            exec_time = time.perf_counter() - start_time
            total_exec_time += exec_time
            individual_times.append(exec_time)

        avg_exec_time = total_exec_time / repetitions  # Calculer le temps d'exécution moyen
        logging.info(f"fibDoubling({n}) averaged over {repetitions} runs: {avg_exec_time:.6f} seconds")
        logging.info(f"Individual execution times for {n}: {individual_times}")

# Fonction principale
# Cette fonction lance le benchmark pour tester les performances du calcul de Fibonacci
if __name__ == "__main__":
    n_values = [1000000, 10000000, 100000000]  # Valeurs à tester
    repetitions = 3  # Nombre de répétitions pour plus de précision
    benchmark_fib(n_values, repetitions)
