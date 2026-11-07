package completion

import (
	"io"
	"testing"
)

func BenchmarkGeneratePowerShellCompletion(b *testing.B) {
	algorithms := []string{"fast", "matrix", "fft", "strassen", "optimized", "karatsuba", "schonhage-strassen", "toom-cook"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Generate(io.Discard, "powershell", algorithms)
	}
}
