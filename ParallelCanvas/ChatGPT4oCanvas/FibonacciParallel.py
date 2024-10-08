import multiprocessing
import os
import time
from math import ceil
from multiprocessing import Pool

# Utilisation d'une approche itérative pour éviter la limite de récursion
# Mémoïsation pour stocker les valeurs intermédiaires de Fibonacci
fibonacci_memo = {0: 0, 1: 1}  # Dictionnaire pour mémoriser les valeurs de Fibonacci déjà calculées

def fibonacci(n):
    # Si la valeur est déjà calculée, la retourner
    if n in fibonacci_memo:
        return fibonacci_memo[n]
    
    # Calcul itératif de la séquence de Fibonacci
    a, b = 0, 1
    for _ in range(2, n + 1):
        a, b = b, a + b  # Calculer le prochain nombre de Fibonacci
    
    # Stocker le résultat dans le dictionnaire pour mémoïsation
    fibonacci_memo[n] = b
    return b

# calc_fibonacci calcule une portion de la liste de Fibonacci entre start et end
def calc_fibonacci_segment(segment):
    start, end = segment  # Segment de la suite à calculer
    partial_sum = 0  # Somme partielle pour ce segment
    for i in range(start, end + 1):
        partial_sum += fibonacci(i)  # Ajouter chaque nombre de Fibonacci au résultat partiel
    return partial_sum  # Retourner la somme partielle

def main():
    # Taille de la suite de Fibonacci que nous voulons calculer (n)
    n = 10000  # Par exemple, 10 000ème nombre de Fibonacci
    num_workers = 4  # Nombre de processus pour effectuer le travail

    # Déterminer la taille du segment pour chaque processus
    segment_size = ceil(n / num_workers)  # Taille de chaque segment calculé par un processus
    segments = [(i * segment_size, min((i + 1) * segment_size - 1, n)) for i in range(num_workers)]
    # Créer une liste de tuples indiquant les bornes (start, end) pour chaque segment

    # Mesurer le temps de début
    start_time = time.time()

    # Créer un pool de processus pour calculer les segments
    with Pool(processes=num_workers) as pool:
        partial_sums = pool.map(calc_fibonacci_segment, segments)  # Calculer chaque segment en parallèle

    # Récupérer et combiner les résultats partiels des calculs de Fibonacci
    total_sum = sum(partial_sums)  # Calculer la somme totale de tous les segments

    # Calculer le temps total écoulé
    execution_time = time.time() - start_time

    # Ouvrir (ou créer) un fichier pour y écrire le résultat
    with open("fibonacci_result.txt", "w") as file:
        file.write(f"Somme des Fib({n}) = {total_sum}\n")  # Écrire le résultat dans un fichier

    # Afficher uniquement le temps d'exécution dans le terminal
    print(f"Temps d'exécution: {execution_time:.2f} secondes")  # Afficher le temps total d'exécution
    print("Résultat et temps d'exécution écrits dans 'fibonacci_result.txt'.")  # Confirmation de l'écriture du fichier

if __name__ == "__main__":
    main()