package main

import (
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/barnex/cuda5/cu"
)

// Code du kernel CUDA (à sauvegarder dans fibonacci_kernel.cu)
const kernelSource = `
extern "C" __global__ void fibonacciKernel(unsigned long long* a, unsigned long long* b, int n) {
    int idx = blockIdx.x * blockDim.x + threadIdx.x;
    if (idx < n) {
        // Utilisation d'unsigned long long pour gérer de plus grands nombres
        unsigned long long temp;
        unsigned long long prev = a[idx];
        unsigned long long curr = b[idx];
        
        // Calcul du nombre de Fibonacci pour cet indice
        for(int i = 0; i < idx; i++) {
            temp = curr;
            curr = prev + curr;
            prev = temp;
        }
        
        a[idx] = prev;
        b[idx] = curr;
    }
}
`

func init() {
	// Initialisation de CUDA
	err := cu.Init(0)
	if err != nil {
		panic(fmt.Sprintf("Impossible d'initialiser CUDA: %v", err))
	}

	// Sélection du premier dispositif CUDA disponible
	dev := cu.DeviceGet(0)
	ctx := cu.CtxCreate(cu.CTX_SCHED_AUTO, dev)
	if ctx == nil {
		panic("Impossible de créer le contexte CUDA")
	}
}

// Structure pour stocker les résultats partiels
type FibResult struct {
	value *big.Int
	index int
}

// Fonction principale de calcul utilisant le GPU
func calculateFibonacciGPU(start, end int, results chan<- FibResult) {
	// Allocation de la mémoire sur le GPU
	n := end - start + 1
	size := n * 8 // 8 bytes pour unsigned long long

	// Allocation de la mémoire sur le device (GPU)
	d_a := cu.MemAlloc(size)
	d_b := cu.MemAlloc(size)

	// Allocation de la mémoire sur l'host (CPU)
	h_a := make([]uint64, n)
	h_b := make([]uint64, n)

	// Initialisation des valeurs
	for i := 0; i < n; i++ {
		h_a[i] = 0 // F(0)
		h_b[i] = 1 // F(1)
	}

	// Copie des données vers le GPU
	cu.MemcpyHtoD(d_a, cu.Malloc(h_a), size)
	cu.MemcpyHtoD(d_b, cu.Malloc(h_b), size)

	// Configuration de la grille et des blocks pour le kernel
	blockSize := 256
	gridSize := (n + blockSize - 1) / blockSize

	// Chargement et exécution du kernel
	module := cu.ModuleLoadData(kernelSource)
	kernel := module.GetFunction("fibonacciKernel")
	kernel.Launch(
		cu.Grid{X: gridSize, Y: 1, Z: 1},
		cu.Block{X: blockSize, Y: 1, Z: 1},
		0, // Shared memory
		cu.Stream(0),
		d_a,
		d_b,
		n,
	)

	// Récupération des résultats
	cu.MemcpyDtoH(cu.Malloc(h_a), d_a, size)
	cu.MemcpyDtoH(cu.Malloc(h_b), d_b, size)

	// Conversion et envoi des résultats
	for i := 0; i < n; i++ {
		result := new(big.Int).SetUint64(h_b[i])
		results <- FibResult{
			value: result,
			index: start + i,
		}
	}

	// Libération de la mémoire GPU
	d_a.Free()
	d_b.Free()
}

func main() {
	n := 100000000       // Nombre de termes à calculer
	batchSize := 1000000 // Taille des lots pour le traitement GPU
	results := make(chan FibResult, batchSize)
	var wg sync.WaitGroup

	startTime := time.Now()

	// Traitement par lots pour éviter de surcharger la mémoire GPU
	for start := 0; start < n; start += batchSize {
		end := start + batchSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			calculateFibonacciGPU(s, e, results)
		}(start, end)
	}

	// Fermeture du canal une fois tous les calculs terminés
	go func() {
		wg.Wait()
		close(results)
	}()

	// Agrégation des résultats
	sumFib := new(big.Int)
	numCalculations := 0
	for result := range results {
		sumFib.Add(sumFib, result.value)
		numCalculations++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	// Écriture des résultats
	file, err := os.Create("fibonacci_result_gpu.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("Somme des Fib(%d) = %s\n", n, sumFib.String()))
	file.WriteString(fmt.Sprintf("Nombre de calculs: %d\n", numCalculations))
	file.WriteString(fmt.Sprintf("Temps moyen par calcul: %s\n", avgTimePerCalculation))
	file.WriteString(fmt.Sprintf("Temps d'exécution: %s\n", executionTime))
	file.WriteString(fmt.Sprintf("Calcul effectué sur GPU NVIDIA\n"))

	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Nombre de calculs: %d\n", numCalculations)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Println("Résultats écrits dans 'fibonacci_result_gpu.txt'")
}
