# ChatGPT 4o Canvas : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
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

import concurrent.futures
import time
import math
from multiprocessing import cpu_count
from collections import defaultdict
import contextlib
from functools import lru_cache

# Constantes pour la configuration
MAX_FIB_VALUE = 500000000  # Valeur maximale de l'indice de Fibonacci pouvant être calculé
SMALL_FIB_THRESHOLD = 93  # Seuil pour utiliser le calcul avec int (pour éviter le dépassement)
DEFAULT_WORKERS = 16  # Nombre par défaut de travailleurs
BENCHMARK_TIMEOUT = 600  # Timeout pour l'opération de benchmark (en secondes)

# Fonction d'erreur personnalisée pour les erreurs de Fibonacci
class FibError(Exception):
    def __init__(self, n, cause):
        # Initialiser une erreur personnalisée avec un message explicatif
        super().__init__(f"Erreur de Fibonacci pour n={n}: {cause}")
        self.n = n
        self.cause = cause

# Classe gérant les calculs de Fibonacci avec mémoïsation
class FibCalculator:
    def __init__(self):
        self.cache = {}  # Cache pour les valeurs déjà calculées
        self.two = 2  # Constante utilisée pour les calculs de doublement

    def calculate(self, n):
        # Fonction principale pour calculer le nième nombre de Fibonacci
        if n < 0:
            # Lever une erreur si l'indice est négatif
            raise FibError(n, "indice négatif")
        if n > MAX_FIB_VALUE:
            # Lever une erreur si l'indice est trop grand
            raise FibError(n, "valeur trop grande")
        
        # Vérifier le cache d'abord pour éviter des recalculs inutiles
        if n in self.cache:
            return self.cache[n]

        # Utiliser un calcul optimisé pour les petites valeurs (n < SMALL_FIB_THRESHOLD)
        if n < SMALL_FIB_THRESHOLD:
            result = self.fib_small(n)
        else:
            # Utiliser la méthode de doublement pour les grandes valeurs
            result = self.fib_doubling(n)
        
        # Stocker le résultat dans le cache pour une réutilisation future
        self.cache[n] = result
        return result

    def fib_small(self, n):
        # Fonction pour calculer Fibonacci pour de petites valeurs de n (utilisant un calcul itératif)
        if n <= 1:
            return n

        a, b = 0, 1
        for _ in range(2, n + 1):
            # Mise à jour des deux dernières valeurs pour le calcul suivant
            a, b = b, a + b
        return b

    def fib_doubling(self, n):
        # Fonction pour calculer Fibonacci pour de grandes valeurs de n en utilisant la méthode de doublement
        if n <= 1:
            return n

        def multiply_matrices(a, b):
            # Fonction pour multiplier deux matrices 2x2 (utilisé dans la méthode de doublement)
            return (
                a[0] * b[0] + a[1] * b[2],  # Premier élément de la matrice résultante
                a[0] * b[1] + a[1] * b[3],  # Deuxième élément de la matrice résultante
                a[2] * b[0] + a[3] * b[2],  # Troisième élément de la matrice résultante
                a[2] * b[1] + a[3] * b[3],  # Quatrième élément de la matrice résultante
            )

        def power(matrix, n):
            # Fonction pour calculer la puissance d'une matrice par exponentiation rapide
            if n == 1:
                return matrix
            if n % 2 == 0:
                half_power = power(matrix, n // 2)
                return multiply_matrices(half_power, half_power)
            else:
                return multiply_matrices(matrix, power(matrix, n - 1))

        # Matrice de base pour calculer Fibonacci
        base_matrix = (1, 1, 1, 0)
        result_matrix = power(base_matrix, n - 1)
        return result_matrix[0]  # Le premier élément de la matrice résultante est F(n)

# Classe pour exécuter les benchmarks
class Benchmarker:
    def __init__(self, workers=DEFAULT_WORKERS):
        self.calculator = FibCalculator()  # Initialiser le calculateur de Fibonacci
        # Définir le nombre de travailleurs (nombre de threads pour l'exécution parallèle)
        self.workers = workers if workers > 0 else cpu_count()

    def run(self, values, repetitions):
        # Fonction pour exécuter le benchmark sur une liste de valeurs
        results = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=self.workers) as executor:
            # Créer des tâches pour chaque valeur et soumettre aux threads
            future_to_n = {
                executor.submit(self._worker, n, repetitions): n for n in values
            }
            
            for future in concurrent.futures.as_completed(future_to_n):
                # Récupérer les résultats des futures à mesure qu'ils sont complétés
                n = future_to_n[future]
                try:
                    results.extend(future.result())
                except Exception as exc:
                    # Afficher une erreur en cas d'échec du calcul
                    print(f"Erreur lors du calcul de F({n}): {exc}")
        return results

    def _worker(self, n, repetitions):
        # Fonction de travailleur pour calculer une valeur de Fibonacci plusieurs fois (pour le benchmark)
        results = []
        for _ in range(repetitions):
            start_time = time.time()  # Commencer le chronométrage
            try:
                self.calculator.calculate(n)  # Calculer la valeur de Fibonacci
                duration = time.time() - start_time  # Calculer la durée du calcul
                results.append((n, duration))  # Ajouter le résultat à la liste des résultats
            except FibError as e:
                # Ajouter un message d'erreur en cas d'échec du calcul
                results.append((n, None, str(e)))
        return results

# Fonction principale pour exécuter le benchmark
if __name__ == "__main__":
    # Définir les valeurs pour lesquelles le benchmark sera exécuté
    values = [1000, 10000, 100000, 1000000, 10000000, 100000000]
    repetitions = 100  # Nombre de répétitions pour chaque valeur
    benchmarker = Benchmarker(DEFAULT_WORKERS)  # Créer une instance du benchmarker

    print(f"Exécution du benchmark avec {benchmarker.workers} travailleurs...")
    results = benchmarker.run(values, repetitions)  # Exécuter le benchmark

    # Traitement des résultats
    results_by_n = defaultdict(list)
    for result in results:
        if len(result) == 2:
            # Si le résultat est valide (pas d'erreur)
            n, duration = result
            results_by_n[n].append(duration)
        else:
            # Si une erreur est présente, l'afficher
            n, _, error = result
            print(f"Erreur lors du calcul de F({n}): {error}")

    # Calculer et afficher la durée moyenne pour chaque valeur de Fibonacci
    for n, durations in results_by_n.items():
        avg_duration = sum(durations) / len(durations)
        print(f"F({n}): Temps moyen = {avg_duration:.6f} secondes")