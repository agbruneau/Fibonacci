package bigfft

import (
	"math/big"
)

// FromDecimalString converts the base 10 string
// representation of a natural (non-negative) number
// into a *big.Int.
// Its asymptotic complexity is less than quadratic.
func FromDecimalString(s string) (*big.Int, error) {
	var sc scanner
	z := new(big.Int)
	if err := sc.scan(z, s); err != nil {
		return nil, err
	}
	return z, nil
}

type scanner struct {
	// powers[i] is 10^(2^i * quadraticScanThreshold).
	powers []*big.Int
}

func (s *scanner) chunkSize(size int) (int, *big.Int) {
	if size <= quadraticScanThreshold {
		panic("size < quadraticScanThreshold")
	}
	pow := uint(0)
	for n := size; n > quadraticScanThreshold; n /= 2 {
		pow++
	}
	// threshold * 2^(pow-1) <= size < threshold * 2^pow
	return quadraticScanThreshold << (pow - 1), s.power(pow - 1)
}

func (s *scanner) power(k uint) *big.Int {
	for i := len(s.powers); i <= int(k); i++ {
		z := new(big.Int)
		if i == 0 {
			if quadraticScanThreshold%14 != 0 {
				panic("quadraticScanThreshold % 14 != 0")
			}
			z.Exp(big.NewInt(1e14), big.NewInt(quadraticScanThreshold/14), nil)
		} else {
			z.Mul(s.powers[i-1], s.powers[i-1])
		}
		s.powers = append(s.powers, z)
	}
	return s.powers[k]
}

func (s *scanner) scan(z *big.Int, str string) error {
	return s.scanWithTemp(z, str, new(big.Int))
}

// scanWithTemp performs the recursive scan while reusing a temporary big.Int
// to reduce allocations during the divide-and-conquer parsing.
func (s *scanner) scanWithTemp(z *big.Int, str string, temp *big.Int) error {
	if len(str) <= quadraticScanThreshold {
		z.SetString(str, 10)
		return nil
	}
	sz, pow := s.chunkSize(len(str))
	// Scan the left half.
	if err := s.scanWithTemp(z, str[:len(str)-sz], temp); err != nil {
		return err
	}
	// Multiply left half by power of 10, reusing temp to avoid allocation.
	left, err := Mul(z, pow)
	if err != nil {
		return err
	}
	// Scan the right half into temp, then add to avoid overwriting z prematurely.
	if err := s.scanWithTemp(temp, str[len(str)-sz:], new(big.Int)); err != nil {
		return err
	}
	z.Add(left, temp)
	return nil
}

// quadraticScanThreshold is the number of digits
// below which big.Int.SetString is more efficient
// than subquadratic algorithms.
// 1232 digits fit in 4096 bits.
const quadraticScanThreshold = 1232
