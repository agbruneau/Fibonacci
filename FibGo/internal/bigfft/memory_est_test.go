package bigfft

import (
	"testing"
)

func TestEstimateMemoryNeeds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		n    uint64
		want MemoryEstimate
	}{
		{
			name: "Small n",
			n:    100,
			want: MemoryEstimate{
				MaxWordSliceSize:   4, // (100 * 0.69424 + 63)/64 * 2 = 2 * 2 = 4
				MaxFermatSize:      2048,
				MaxNatSliceSize:    2048,
				MaxFermatSliceSize: 2048,
			},
		},
		{
			name: "Medium n (wordLen > 10000)",
			n:    1000000, // bitLen ~ 694240, wordLen ~ 10848
			want: MemoryEstimate{
				MaxWordSliceSize:   21696,
				MaxFermatSize:      131072,
				MaxNatSliceSize:    2048,
				MaxFermatSliceSize: 2048,
			},
		},
		{
			name: "Large n (wordLen > 100000)",
			n:    10000000, // bitLen ~ 6942400, wordLen ~ 108475
			// k = bits.Len(108475) = 17
			// kVal = 1 << (17-3) = 16384
			want: MemoryEstimate{
				MaxWordSliceSize:   216950,
				MaxFermatSize:      524288,
				MaxNatSliceSize:    16384,
				MaxFermatSliceSize: 16384,
			},
		},
		{
			name: "Huge n (wordLen > 1000000)",
			n:    100000000, // bitLen ~ 69424000, wordLen ~ 1084750
			// k = bits.Len(1084750) = 21
			// kVal = 1 << (21-3) = 262144
			want: MemoryEstimate{
				MaxWordSliceSize:   2169500,
				MaxFermatSize:      2097152,
				MaxNatSliceSize:    262144,
				MaxFermatSliceSize: 262144,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EstimateMemoryNeeds(tt.n)
			if got.MaxWordSliceSize != tt.want.MaxWordSliceSize {
				t.Errorf("MaxWordSliceSize = %v, want %v", got.MaxWordSliceSize, tt.want.MaxWordSliceSize)
			}
			if got.MaxFermatSize != tt.want.MaxFermatSize {
				t.Errorf("MaxFermatSize = %v, want %v", got.MaxFermatSize, tt.want.MaxFermatSize)
			}
			if got.MaxNatSliceSize != tt.want.MaxNatSliceSize {
				t.Errorf("MaxNatSliceSize = %v, want %v", got.MaxNatSliceSize, tt.want.MaxNatSliceSize)
			}
			if got.MaxFermatSliceSize != tt.want.MaxFermatSliceSize {
				t.Errorf("MaxFermatSliceSize = %v, want %v", got.MaxFermatSliceSize, tt.want.MaxFermatSliceSize)
			}
		})
	}
}
