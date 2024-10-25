package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"
	"unsafe"

	"gorgonia.org/cu"
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
    .reg .u64   %rd<5>;
    .reg .u32   %r<4>;
    .reg .u64   %temp;
    .reg .pred  %p1;

    ld.param.u64    %rd1, [a];
    ld.param.u64    %rd2, [b];
    ld.param.u32    %r1, [n];
    
    mov.u32     %r2, %ctaid.x;
    mov.u32     %r3, %ntid.x;
    mad.lo.u32  %r3, %r2, %r3, %tid.x;
    setp.ge.u32     %p1, %r3, %r1;
    @%p1 bra    BB0_2;

    mul.wide.u32    %rd3, %r3, 8;
    add.u64     %rd4, %rd1, %rd3;
    
    ld.global.u64   %rd1, [%rd4];
    ld.global.u64   %rd2, [%rd4+8];
    
    mov.u64     %temp, %rd2;
    add.u64     %rd2, %rd1, %rd2;
    mov.u64     %rd1, %temp;
    
    st.global.u64   [%rd4], %rd1;
    st.global.u64   [%rd4+8], %rd2;

BB0_2:
    ret;
}
`

type GPUFibonacci struct {
	context cu.CUContext
	module  cu.Module
	kernel  cu.Function
	device  cu.Device
	stream  cu.Stream
}

func NewGPUFibonacci() (*GPUFibonacci, error) {
	if err := cu.Init(cu.CTX_SCHED_AUTO); err != nil {
		return nil, fmt.Errorf("error initializing CUDA: %v", err)
	}

	devices, err := cu.NumDevices()
	if err != nil {
		return nil, fmt.Errorf("error counting devices: %v", err)
	}

	if devices == 0 {
		return nil, fmt.Errorf("no CUDA devices found")
	}

	device, err := cu.GetDevice(0)
	if err != nil {
		return nil, fmt.Errorf("error getting device: %v", err)
	}

	ctx, err := cu.CtxCreate(cu.CTX_SCHED_AUTO, device)
	if err != nil {
		return nil, fmt.Errorf("error creating context: %v", err)
	}

	module, err := cu.ModuleLoadData(ptxSource)
	if err != nil {
		return nil, fmt.Errorf("error loading PTX module: %v", err)
	}

	kernel, err := module.Function("fibonacciKernel")
	if err != nil {
		return nil, fmt.Errorf("error getting kernel: %v", err)
	}

	stream := cu.Stream(0)

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
	size := int64(n * 8)

	d_a, err := cu.MemAlloc(size)
	if err != nil {
		return fmt.Errorf("error allocating device memory for a: %v", err)
	}
	defer cu.MemFree(d_a)

	d_b, err := cu.MemAlloc(size)
	if err != nil {
		return fmt.Errorf("error allocating device memory for b: %v", err)
	}
	defer cu.MemFree(d_b)

	h_a := make([]uint64, n)
	h_b := make([]uint64, n)
	for i := 0; i < n; i++ {
		h_a[i] = 0
		h_b[i] = 1
	}

	if err := cu.MemcpyHtoD(d_a, unsafe.Pointer(&h_a[0]), size); err != nil {
		return fmt.Errorf("error copying h_a to device: %v", err)
	}
	if err := cu.MemcpyHtoD(d_b, unsafe.Pointer(&h_b[0]), size); err != nil {
		return fmt.Errorf("error copying h_b to device: %v", err)
	}

	blockSize := 256
	gridSize := (n + blockSize - 1) / blockSize

	args := []unsafe.Pointer{
		unsafe.Pointer(&d_a),
		unsafe.Pointer(&d_b),
		unsafe.Pointer(uintptr(n)),
	}

	err = cu.LaunchKernel(gf.kernel, gridSize, 1, 1, blockSize, 1, 1, 0, gf.stream, args)
	if err != nil {
		return fmt.Errorf("error launching kernel: %v", err)
	}

	if err := cu.StreamSynchronize(gf.stream); err != nil {
		return fmt.Errorf("error synchronizing stream: %v", err)
	}

	if err := cu.MemcpyDtoH(unsafe.Pointer(&h_b[0]), d_b, size); err != nil {
		return fmt.Errorf("error copying results back to host: %v", err)
	}

	for i := 0; i < n; i++ {
		results <- new(big.Int).SetUint64(h_b[i])
	}

	return nil
}

func (gf *GPUFibonacci) Cleanup() {
	cu.CtxDestroy(gf.context)
}

func main() {
	n := 100000000
	batchSize := 1000000
	results := make(chan *big.Int, batchSize)
	var wg sync.WaitGroup

	gpu, err := NewGPUFibonacci()
	if err != nil {
		log.Fatalf("Error initializing GPU: %v", err)
	}
	defer gpu.Cleanup()

	startTime := time.Now()

	for start := 0; start < n; start += batchSize {
		end := start + batchSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			if err := gpu.Calculate(s, e, results); err != nil {
				log.Printf("Error in GPU calculation [%d-%d]: %v", s, e, err)
			}
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	sumFib := new(big.Int)
	numCalculations := 0
	for result := range results {
		sumFib.Add(sumFib, result)
		numCalculations++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	file, err := os.Create("fibonacci_result_gpu.txt")
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	writeResult := func(format string, args ...interface{}) {
		if _, err := fmt.Fprintf(file, format, args...); err != nil {
			log.Printf("Error writing to file: %v", err)
		}
	}

	writeResult("Sum of Fib(%d) = %s\n", n, sumFib.String())
	writeResult("Number of calculations: %d\n", numCalculations)
	writeResult("Average time per calculation: %s\n", avgTimePerCalculation)
	writeResult("Total execution time: %s\n", executionTime)
	writeResult("Calculation performed on NVIDIA 4070\n")

	fmt.Printf("Execution time: %s\n", executionTime)
	fmt.Printf("Number of calculations: %d\n", numCalculations)
	fmt.Printf("Average time per calculation: %s\n", avgTimePerCalculation)
	fmt.Println("Results written to 'fibonacci_result_gpu.txt'")
}
