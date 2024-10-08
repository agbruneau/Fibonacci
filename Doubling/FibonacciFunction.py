# Programme Python : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
#
# Description :
# Ce programme en Python calcule les nombres de Fibonacci en utilisant la méthode du doublement, qui est une approche
# efficace basée sur la division et la conquête. L'algorithme utilise une technique itérative pour calculer
# rapidement les valeurs de Fibonacci pour de très grands nombres. Pour améliorer la performance, une stratégie
# de mémoïsation avec LRU (Least Recently Used) est utilisée afin de mettre en cache les résultats des calculs
# précédents. Cela permet de réutiliser les valeurs déjà calculées et de réduire le temps de calcul des appels
# futurs. De plus, le programme est conçu pour utiliser la parallélisation afin d'améliorer l'efficacité.
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
# une meilleure précision.

import time
from functools import lru_cache
from concurrent.futures import ProcessPoolExecutor
from multiprocessing import cpu_count

MAX_FIB_VALUE = 500000001  # Valeur maximale de n qui peut être calculée

# Utiliser lru_cache pour mémoïser les résultats afin d'éviter de recalculer des valeurs déjà connues
@lru_cache(maxsize=5000)  # Augmenter la taille du cache ou la rendre configurable
def fib_doubling(n):
    # Si n est inférieur à 2, les valeurs sont directement 0 ou 1
    if n < 2:
        return n
    # Si n dépasse la valeur maximale, lever une erreur
    elif n > MAX_FIB_VALUE:
        raise ValueError("n est trop grand pour cette implémentation")
    # Pour les petites valeurs, utiliser une approche plus rapide
    elif n < 93:
        return fib_int64(n)
    # Pour les grandes valeurs, utiliser la méthode de doublement
    else:
        return fib_doubling_helper_iterative(n)

# Fonction qui calcule Fibonacci pour les petites valeurs
@lru_cache(maxsize=500)  # Ajouter la mémoïsation pour améliorer les performances pour les petites valeurs
def fib_int64(n):
    a, b = 0, 1
    # Boucle pour calculer le nième nombre de Fibonacci
    for _ in range(n):
        a, b = b, a + b
    return a

# Fonction itérative pour calculer les nombres de Fibonacci avec la méthode de doublement
@lru_cache(maxsize=1000)  # Ajouter la mémoïsation pour éviter les recalculs redondants
def fib_doubling_helper_iterative(n):
    # Initialiser les valeurs de base de Fibonacci
    a, b = 0, 1
    # Déterminer le nombre de bits nécessaires pour représenter n
    bit_length = n.bit_length()

    # Itérer sur chaque bit du plus significatif au moins significatif
    for i in range(bit_length - 1, -1, -1):
        # Calculer F(2k) et F(2k + 1) en utilisant les formules de doublement
        c = a * ((2 * b) - a)  # c = F(k) * [2 * F(k+1) - F(k)]
        d = a * a + b * b  # d = F(k)^2 + F(k+1)^2

        # Mettre à jour les valeurs de a et b en fonction du bit actuel de n
        if (n >> i) & 1 == 0:
            a, b = c, d  # Si le bit est 0, a devient F(2k) et b devient F(2k+1)
        else:
            a, b = d, c + d  # Si le bit est 1, a devient F(2k+1) et b devient F(2k+2)

    return a

# Fonction pour effacer le cache LRU, utile avant les benchmarks pour avoir des mesures précises
def clear_memoization():
    fib_doubling.cache_clear()
    fib_int64.cache_clear()
    fib_doubling_helper_iterative.cache_clear()

# Fonction de benchmark pour tester la performance des calculs de Fibonacci
def benchmark_fib(n_values, repetitions, worker_count):
    clear_memoization()  # Effacer le cache pour des benchmarks précis
    # Utiliser ProcessPoolExecutor pour paralléliser le calcul
    with ProcessPoolExecutor(max_workers=worker_count) as executor:
        # Soumettre les tâches pour chaque valeur de n
        futures = [executor.submit(benchmark_worker, n, repetitions) for n in n_values]
        # Attendre les résultats et les afficher
        for future in futures:
            n, avg_exec_time = future.result()
            print(f"fib_doubling({n}) averaged over {repetitions} runs: {avg_exec_time:.5f} seconds")

# Worker pour effectuer les benchmarks individuels
def benchmark_worker(n, repetitions):
    total_exec_time = 0.0
    # Effectuer le calcul plusieurs fois pour obtenir un temps moyen
    for _ in range(repetitions):
        start_time = time.perf_counter()  # Utiliser perf_counter pour une meilleure précision
        try:
            fib_doubling(n)  # Calculer Fibonacci pour la valeur donnée
        except ValueError as e:
            # Si n est trop grand, afficher un message d'erreur et retourner un temps infini
            print(f"fib_doubling({n}): {e}")
            return n, float('inf')
        total_exec_time += time.perf_counter() - start_time
    # Calculer le temps d'exécution moyen
    avg_exec_time = total_exec_time / repetitions
    return n, avg_exec_time

# Fonction principale pour exécuter les tests de performance
def main():
    # Liste des valeurs pour lesquelles effectuer les benchmarks
    n_values = [100000, 500000, 1000000, 5000000, 10000000, 50000000, 100000000, 500000000]
    repetitions = 100  # Nombre de répétitions pour obtenir un temps moyen
    worker_count = min(cpu_count(), 16)  # Nombre de workers à utiliser, limité au nombre de CPU disponibles
    benchmark_fib(n_values, repetitions, worker_count)  # Exécuter le benchmark

if __name__ == "__main__":
    main()