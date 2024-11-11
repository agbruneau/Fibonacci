# Ce programme calcule la somme des n premiers nombres de Fibonacci de manière parallélisée.
# Il utilise des techniques avancées de concurrence en Python et gère les grands nombres.

import concurrent.futures
import time
from functools import lru_cache
from threading import Lock
import multiprocessing
from math import log10

# Configuration centralise tous les paramètres configurables du programme.
class Configuration:
    def __init__(self, m=100000, num_workers=None, segment_size=1000, timeout=5 * 60):
        # m définit la limite supérieure (exclue) du calcul de Fibonacci.
        self.m = m
        # num_workers définit le nombre de workers parallèles à utiliser.
        # Si aucun n'est spécifié, on utilise le nombre de cœurs disponibles sur la machine.
        self.num_workers = num_workers if num_workers else multiprocessing.cpu_count()
        # segment_size est la taille des segments de calcul assignés à chaque worker.
        self.segment_size = segment_size
        # timeout est la durée maximale autorisée pour le calcul complet, en secondes.
        self.timeout = timeout

# Metrics garde trace des métriques de performance pendant l'exécution.
class Metrics:
    def __init__(self):
        # start_time enregistre l'heure de début du calcul.
        self.start_time = time.time()
        # end_time enregistrera l'heure de fin du calcul.
        self.end_time = None
        # total_calculations est le nombre total de calculs effectués.
        self.total_calculations = 0
        # lock est utilisé pour assurer la sécurité des threads lors de l'incrémentation des calculs.
        self.lock = Lock()

    # increment_calculations incrémente le nombre total de calculs de manière thread-safe.
    def increment_calculations(self, count):
        with self.lock:
            self.total_calculations += count

# Mémoïsation pour améliorer la performance des calculs de Fibonacci.
@lru_cache(maxsize=None)
def fibonacci(n):
    # Validation des entrées
    if n < 0:
        raise ValueError("n doit être non-négatif")
    # Cas de base pour F(0)
    elif n == 0:
        return 0
    # Cas de base pour F(1)
    elif n == 1:
        return 1
    # Calcul récursif pour F(n) lorsque n > 1
    else:
        return fibonacci(n - 1) + fibonacci(n - 2)

# compute_segment calcule la somme des nombres de Fibonacci pour un segment donné.
def compute_segment(start, end, metrics, timeout):
    partial_sum = 0
    segment_size = end - start + 1
    start_time = time.time()

    for i in range(start, end + 1):
        # Vérifie si le timeout est atteint pour éviter des calculs trop longs.
        if time.time() - start_time > timeout:
            return None, "Timeout atteint pendant le calcul du segment"
        
        # Calcule F(i) et l'ajoute à la somme partielle.
        partial_sum += fibonacci(i)

    # Incrémente le nombre total de calculs dans les métriques.
    metrics.increment_calculations(segment_size)
    return partial_sum, None

# format_big_int_sci formate un grand nombre en notation scientifique.
def format_big_int_sci(n):
    if n == 0:
        return "0"
    # Calcul de l'exposant pour la notation scientifique.
    exponent = int(log10(n))
    # Calcul de la partie significative.
    significand = n / 10**exponent
    return f"{significand:.5g}e{exponent}"

# La fonction main orchestre tout le processus de calcul.
def main():
    # Initialisation de la configuration et des métriques.
    config = Configuration()
    metrics = Metrics()

    # Définition de la limite supérieure pour le calcul.
    n = config.m - 1
    results = []
    has_errors = False

    # Création d'un pool d'exécuteurs pour les workers parallèles.
    with concurrent.futures.ThreadPoolExecutor(max_workers=config.num_workers) as executor:
        futures = []
        # Assignation des segments à calculer à chaque worker.
        for start in range(0, n, config.segment_size):
            end = min(start + config.segment_size - 1, n - 1)
            futures.append(executor.submit(compute_segment, start, end, metrics, config.timeout))
        
        # Collecte et agrégation des résultats au fur et à mesure de l'achèvement des futures.
        for future in concurrent.futures.as_completed(futures):
            try:
                result, error = future.result()
                if error:
                    print(f"Erreur durant le calcul: {error}")
                    has_errors = True
                else:
                    results.append(result)
            except Exception as e:
                print(f"Erreur inattendue: {e}")
                has_errors = True

    # Calcul de la somme totale des segments.
    sum_fib = sum(results)

    # Calcul des métriques finales.
    metrics.end_time = time.time()
    duration = metrics.end_time - metrics.start_time
    avg_time = duration / metrics.total_calculations if metrics.total_calculations > 0 else float('inf')

    # Affichage des résultats de la configuration.
    print("\nConfiguration:")
    print(f"  Nombre de workers: {config.num_workers}")
    print(f"  Taille des segments: {config.segment_size}")
    print(f"  Valeur de m: {config.m}")

    # Affichage des performances du programme.
    print("\nPerformance:")
    print(f"  Temps total d'exécution: {duration:.2f} secondes")
    print(f"  Nombre de calculs: {metrics.total_calculations}")
    print(f"  Temps moyen par calcul: {avg_time:.6f} secondes")

    # Affichage du résultat de la somme des nombres de Fibonacci.
    print("\nRésultat:")
    print(f"  Somme des Fibonacci(0..{config.m}): {format_big_int_sci(sum_fib)}")

if __name__ == "__main__":
    main()
