import contextlib
import time
import threading
from functools import lru_cache

# Valeur maximale de n qui peut être calculée
MAX_FIB_VALUE = 500000001

# Créer un cache LRU avec une taille maximale
@lru_cache(maxsize=1000)
def fib_matrix_power(n):
    """
    Calcule le nième nombre de Fibonacci en utilisant la méthode de la matrice de puissance
    """
    # Si n est inférieur à 2, retourner directement le résultat correspondant
    if n < 2:
        return n
    elif n > MAX_FIB_VALUE:
        raise ValueError("n est trop grand pour cette implémentation")

    # Initialiser les matrices de base
    F = [[1, 1], [1, 0]]
    result = matrix_power(F, n - 1)

    # La valeur de Fibonacci est dans la case [0][0] de la matrice résultante
    return result[0][0]


def matrix_power(matrix, n):
    """
    Calcule la puissance d'une matrice à l'exposant n en utilisant l'exponentiation rapide
    """
    # Initialiser la matrice identité (matrice neutre pour la multiplication)
    result = [[1, 0], [0, 1]]

    # Utiliser l'exponentiation rapide pour calculer la puissance de la matrice
    base = matrix
    while n > 0:
        # Si le bit courant est 1, multiplier le résultat par la base
        if n % 2 == 1:
            result = matrix_multiply(result, base)
        # Multiplier la base par elle-même pour l'exponentiation rapide
        base = matrix_multiply(base, base)
        n //= 2

    return result


def matrix_multiply(a, b):
    """
    Multiplie deux matrices 2x2 et retourne le résultat
    """
    return [
        [
            # Calculer la valeur en [0][0]
            a[0][0] * b[0][0] + a[0][1] * b[1][0],
            # Calculer la valeur en [0][1]
            a[0][0] * b[0][1] + a[0][1] * b[1][1]
        ],
        [
            # Calculer la valeur en [1][0]
            a[1][0] * b[0][0] + a[1][1] * b[1][0],
            # Calculer la valeur en [1][1]
            a[1][0] * b[0][1] + a[1][1] * b[1][1]
        ],
    ]


def benchmark_fib_with_worker_pool(n_values, repetitions, worker_count):
    """
    Effectue des tests de performance sur les calculs de Fibonacci pour une liste de valeurs en utilisant un pool de threads
    """
    # Effacer le cache LRU avant de commencer le benchmark
    fib_matrix_power.cache_clear()

    # Fonction de travail pour les threads
    def worker(worker_id, jobs):
        for n in jobs:
            total_exec_time = 0
            for _ in range(repetitions):
                start_time = time.time()
                try:
                    # Calculer le nième nombre de Fibonacci
                    fib_matrix_power(n)
                except ValueError as e:
                    # Afficher l'erreur si n est trop grand
                    print(f"Worker {worker_id}: {e}")
                    continue
                # Calculer le temps écoulé pour ce calcul
                total_exec_time += (time.time() - start_time)
            # Calculer le temps moyen d'exécution
            avg_exec_time = total_exec_time / repetitions
            print(f"Worker {worker_id}: fib_matrix_power({n}) averaged over {repetitions} runs: {avg_exec_time:.6f} seconds")

    # Diviser les travaux entre les workers
    jobs_per_worker = len(n_values) // worker_count
    threads = []
    for i in range(worker_count):
        # Déterminer les indices de début et de fin des travaux pour chaque worker
        start_idx = i * jobs_per_worker
        end_idx = (i + 1) * jobs_per_worker if i != worker_count - 1 else len(n_values)
        # Créer un thread pour chaque worker
        thread = threading.Thread(target=worker, args=(i, n_values[start_idx:end_idx]))
        threads.append(thread)
        thread.start()

    # Attendre que tous les threads se terminent
    for thread in threads:
        thread.join()


# Fonction principale pour exécuter les tests de performance
def main():
    # Liste des valeurs pour lesquelles effectuer les tests de performance
    n_values = [100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000]
    repetitions = 250  # Nombre de répétitions pour calculer le temps moyen
    worker_count = 4  # Nombre de threads concurrents

    # Exécuter le benchmark
    benchmark_fib_with_worker_pool(n_values, repetitions, worker_count)


if __name__ == "__main__":
    main()