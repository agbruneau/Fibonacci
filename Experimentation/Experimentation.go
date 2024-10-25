package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/cloudkj/cuda-go"
)

const ptxSource = `
.version 7.0
.target sm_70
.address_size 64

.visible .entry fibonacciKernel(
    .param .u64 a,
    .param .u64 b,
    .param .u32 n
)
{
    .reg .u64 	%rd<5>;
    .reg .u32 	%r<4>;
    .reg .u64   %temp;
    .reg .pred 	%p1;

    ld.param.u64 	%rd1, [a];
    ld.param.u64 	%rd2, [b];
    ld.param.u32 	%r1, [n];
    
    mov.u32 	%r2, %ctaid.x;
    mov.u32 	%r3, %ntid.x;
    mad.lo.u32 	%r3, %r2, %r3, %tid.x;
    setp.ge.u32 	%p1, %r3, %r1;
    @%p1 bra 	BB0_2;

    // Calcul de l'offset pour accéder aux tableaux
    mul.wide.u32 	%rd3, %r3, 8;
    add.u64 	%rd4, %rd1, %rd3;
    
    // Chargement des valeurs initiales
    ld.global.u64 	%rd1, [%rd4];
    ld.global.u64 	%rd2, [%rd4+8];
    
    // Calcul de Fibonacci
    mov.u64     %temp, %rd2;
    add.u64     %rd2, %rd1, %rd2;
    mov.u64     %rd1, %temp;
    
    // Sauvegarde des résultats
    st.global.u64 	[%rd4], %rd1;
    st.global.u64 	[%rd4+8], %rd2;

BB0_2:
    ret;
}
`

type GPUFibonacci struct {
	context *cuda.Context
	module  *cuda.Module
	kernel  *cuda.Function
	device  *cuda.Device
	stream  *cuda.Stream
}

func NewGPUFibonacci() (*GPUFibonacci, error) {
	// Initialisation de CUDA
	device, err := cuda.GetDevice(0)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'obtention du device CUDA: %v", err)
	}

	// Création du contexte
	ctx, err := cuda.CreateContext(device)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du contexte: %v", err)
	}

	// Compilation du kernel PTX
	module, err := ctx.LoadModulePTX(ptxSource)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du chargement du module PTX: %v", err)
	}

	// Obtention du kernel
	kernel, err := module.GetFunction("fibonacciKernel")
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'obtention du kernel: %v", err)
	}

	// Création du stream
	stream, err := ctx.CreateStream()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du stream: %v", err)
	}

	return &GPUFibonacci{
		context: ctx,
		module:  module,
		kernel:  kernel,
		device:  device,
		stream:  stream,
	}, nil
}

func (gf *GPUFibonacci) Calculate(start, end int, results chan<- *big.Int) error {
	n := end - start + 1
	size := int64(n * 8) // 8 bytes par nombre

	// Allocation de la mémoire sur le GPU
	d_a, err := gf.context.AllocateMemory(size)
	if err != nil {
		return fmt.Errorf("erreur lors de l'allocation de la mémoire GPU (a): %v", err)
	}
	defer d_a.Free()

	d_b, err := gf.context.AllocateMemory(size)
	if err != nil {
		return fmt.Errorf("erreur lors de l'allocation de la mémoire GPU (b): %v", err)
	}
	defer d_b.Free()

	// Préparation des données initiales
	h_a := make([]uint64, n)
	h_b := make([]uint64, n)
	for i := 0; i < n; i++ {
		h_a[i] = 0 // F(0)
		h_b[i] = 1 // F(1)
	}

	// Copie des données vers le GPU
	if err := gf.context.CopyToDevice(d_a, h_a); err != nil {
		return fmt.Errorf("erreur lors de la copie vers le GPU (a): %v", err)
	}
	if err := gf.context.CopyToDevice(d_b, h_b); err != nil {
		return fmt.Errorf("erreur lors de la copie vers le GPU (b): %v", err)
	}

	// Configuration de la grille et des blocs
	blockSize := 256
	gridSize := (n + blockSize - 1) / blockSize

	// Lancement du kernel
	params := []interface{}{d_a, d_b, uint32(n)}
	if err := gf.kernel.Launch(gridSize, 1, 1, blockSize, 1, 1, 0, gf.stream, params...); err != nil {
		return fmt.Errorf("erreur lors du lancement du kernel: %v", err)
	}

	// Synchronisation
	if err := gf.stream.Synchronize(); err != nil {
		return fmt.Errorf("erreur lors de la synchronisation: %v", err)
	}

	// Récupération des résultats
	if err := gf.context.CopyFromDevice(h_b, d_b); err != nil {
		return fmt.Errorf("erreur lors de la copie depuis le GPU: %v", err)
	}

	// Envoi des résultats
	for i := 0; i < n; i++ {
		results <- new(big.Int).SetUint64(h_b[i])
	}

	return nil
}

func main() {
	n := 100000000       // Nombre de termes à calculer
	batchSize := 1000000 // Taille des lots pour le traitement GPU
	results := make(chan *big.Int, batchSize)
	var wg sync.WaitGroup

	// Initialisation du GPU
	gpu, err := NewGPUFibonacci()
	if err != nil {
		log.Fatalf("Erreur lors de l'initialisation du GPU: %v", err)
	}

	startTime := time.Now()

	// Traitement par lots
	for start := 0; start < n; start += batchSize {
		end := start + batchSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			if err := gpu.Calculate(s, e, results); err != nil {
				log.Printf("Erreur lors du calcul GPU pour le lot [%d-%d]: %v", s, e, err)
			}
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
		sumFib.Add(sumFib, result)
		numCalculations++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	// Écriture des résultats
	file, err := os.Create("fibonacci_result_gpu.txt")
	if err != nil {
		log.Fatalf("Erreur lors de la création du fichier: %v", err)
	}
	defer file.Close()

	writeResult := func(format string, args ...interface{}) {
		if _, err := fmt.Fprintf(file, format, args...); err != nil {
			log.Printf("Erreur lors de l'écriture dans le fichier: %v", err)
		}
	}

	writeResult("Somme des Fib(%d) = %s\n", n, sumFib.String())
	writeResult("Nombre de calculs: %d\n", numCalculations)
	writeResult("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	writeResult("Temps d'exécution: %s\n", executionTime)
	writeResult("Calcul effectué sur GPU NVIDIA 4070\n")

	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Nombre de calculs: %d\n", numCalculations)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Println("Résultats écrits dans 'fibonacci_result_gpu.txt'")
}
