// Package fib calcule les termes de la suite de Fibonacci en précision arbitraire (math/big).
package fib

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Fib rend F(n) par l'algorithme algo (iterative, matrix ou fast); elle refuse un indice négatif et tout autre nom.
func Fib(n int, algo string) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice negatif : %d", n)
	}
	switch algo {
	case "iterative":
		return iterative(n), nil
	case "matrix":
		return matrix(n), nil
	case "fast":
		return fast(n), nil
	}
	return nil, fmt.Errorf("algorithme inconnu : %+q", algo)
}

// base : un limbe décimal tient 18 chiffres. 10^18 laisse la place d'une retenue dans un uint64; 10^19 déborderait.
const (
	base       = 1_000_000_000_000_000_000
	baseDigits = 18
)

// digitsPerMille : chiffres par millier d'indices, car F(k) compte moins de 0,209·k + 1 chiffres (log10 φ = 0,20899…).
const digitsPerMille = 209

// FibList rend F(0) … F(n) en chaînes décimales, toujours par l'algorithme iterative; elle refuse un indice négatif.
// Les termes sont additionnés directement en base 10^18 : aucune conversion binaire → décimal.
func FibList(n int) ([]string, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice negatif : %d", n)
	}
	quadsOnce.Do(fillQuads)
	// Une réserve pour tout le texte, et des limbes pour F(n + 1).
	var text strings.Builder
	text.Grow(n*(n+1)/2*digitsPerMille/1000 + 2*n + 2)
	size := (n+2)*digitsPerMille/1000/baseDigits + 3
	a, b := make([]uint64, size), make([]uint64, size) // F(k) et F(k + 1), limbes de poids faible d'abord
	b[0] = 1
	la, lb := 1, 1 // limbes utilisés; les suivants de a restent à 0
	ends := make([]int, n+1)
	digits := make([]byte, 0, size*baseDigits) // le texte d'un terme, réservé pour le plus long
	for k := 0; ; k++ {
		digits = appendDecimal(digits[:0], a[:la])
		text.Write(digits)
		ends[k] = text.Len()
		if k == n {
			break
		}
		// a, b = b, a + b : a reçoit la somme sur place, puis les rôles s'échangent.
		carry := addLimbs(a, b[:lb]) // a[lb] est à 0 : a n'a jamais plus de limbes que b
		a, b, la, lb = b, a, lb, lb+int(carry)
	}
	// String() ne copie pas : les n + 1 termes sont des sous-chaînes du même tampon, sans allocation chacun.
	all, list := text.String(), make([]string, n+1)
	for k, start := 0, 0; k <= n; k++ {
		list[k], start = all[start:ends[k]], ends[k]
	}
	return list, nil
}

// addLimbs ajoute b à a sur place, limbe par limbe, et range la retenue finale (0 ou 1) dans a[len(b)], qu'elle rend aussi.
// À part pour que PERF-GO-5 en vise la retenue à 10^18, et assez courte pour rester inlinée (go build -gcflags=-m).
func addLimbs(a, b []uint64) (carry uint64) {
	for i, y := range b {
		s := a[i] + y + carry
		carry = 0
		if s >= base {
			s -= base
			carry = 1
		}
		a[i] = s
	}
	a[len(b)] += carry
	return carry
}

// quads donne les quatre chiffres ASCII de 0000 à 9999, un par entrée : une recherche par quatre chiffres au lieu de
// quatre divisions. fillQuads le remplit au premier FibList, pour que les autres commandes ne paient rien au démarrage.
var (
	quads     [10_000]uint32
	quadsOnce sync.Once
)

func fillQuads() {
	for i := range quads {
		quads[i] = uint32('0'+i/1000)<<24 | uint32('0'+i/100%10)<<16 | uint32('0'+i/10%10)<<8 | uint32('0'+i%10)
	}
}

// appendDecimal ajoute à dst le terme que représentent les limbes limbs (poids faible d'abord) : le limbe de tête tel
// quel, les autres sur 18 chiffres.
func appendDecimal(dst []byte, limbs []uint64) []byte {
	top := len(limbs) - 1
	dst = strconv.AppendUint(dst, limbs[top], 10)
	start := len(dst)
	// Les autres limbes s'écrivent sur place, dans une réserve prise d'un coup : pas d'append par limbe.
	dst = slices.Grow(dst, top*baseDigits)[:start+top*baseDigits]
	for i := top - 1; i >= 0; i-- {
		t := (*[baseDigits]byte)(dst[start:])
		x := limbs[i]
		for j := baseDigits - 4; j >= 2; j -= 4 {
			q := x / 10_000
			binary.BigEndian.PutUint32(t[j:], quads[x-q*10_000])
			x = q
		}
		pair := quads[x] // x < 100 : les deux derniers octets de « 00xy » sont les deux chiffres de tête
		t[0], t[1] = byte(pair>>8), byte(pair)
		start += baseDigits
	}
	return dst
}

// iterative applique la définition indice par indice, avec deux termes consécutifs en mémoire.
// L'addition se fait sur place dans des tampons réservés d'emblée : aucune allocation par pas.
func iterative(n int) *big.Int {
	// F(n) compte moins de 0,7·n bits (0,694·n); un mot de plus pour la retenue de Add.
	words := n*7/10/bits.UintSize + 3
	a := new(big.Int).SetBits(make([]big.Word, 0, words))
	b := new(big.Int).SetBits(make([]big.Word, 0, words)).SetInt64(1)
	for range n {
		a.Add(a, b) // a = F(k) + F(k + 1) = F(k + 2), puis les rôles s'échangent
		a, b = b, a
	}
	return a
}

// matrix : [[1,1],[1,0]]^n = [[F(n+1), F(n)], [F(n), F(n-1)]], par carrés successifs, des bits de poids fort à ceux
// de poids faible. Chaque puissance est symétrique : trois coefficients (a, b, c) = (F(k+1), F(k), F(k-1)) suffisent.
func matrix(n int) *big.Int {
	if n < 2 {
		return big.NewInt(int64(n))
	}
	a, b, c := big.NewInt(1), big.NewInt(1), new(big.Int) // Q¹, le bit de poids fort de n
	for i := bits.Len(uint(n)) - 2; i >= 1; i-- {
		// [[a,b],[b,c]]² = [[a²+b², b(a+c)], [b(a+c), b²+c²]] : trois carrés; b′ = a′ − c′ = a² − c² = b(a+c), car a − c = b.
		// bb sert à a′ et à c′ avant de recevoir b′.
		aa, bb, cc := new(big.Int).Mul(a, a), new(big.Int).Mul(b, b), new(big.Int).Mul(c, c)
		a, c = aa.Add(aa, bb), cc.Add(cc, bb)
		b = bb.Sub(a, c)
		if n>>i&1 == 1 {
			// fois Q : [[a,b],[b,c]]·[[1,1],[1,0]] = [[a+b, a], [a, b]], des additions seulement
			a, b, c = new(big.Int).Add(a, b), a, b
		}
	}
	// Dernier bit : seul F(n), le coefficient b, est utile; ni c ni le reste de la matrice ne sont calculés.
	if n&1 == 0 {
		return new(big.Int).Mul(new(big.Int).Add(a, c), b) // b(a+c)
	}
	aa, bb := new(big.Int).Mul(a, a), new(big.Int).Mul(b, b)
	return aa.Add(aa, bb) // a²+b², le coefficient a du carré, que fois Q ramène en b
}

// fast est le doublement rapide, lu de gauche à droite sur les bits de n : à chaque bit, (F(k), F(k-1)) devient
// (F(2k), F(2k-1)) pour un bit à 0, ou (F(2k+1), F(2k)) pour un bit à 1. Deux carrés par bit, sauf au dernier :
// il ne calcule que F(n), en un produit.
func fast(n int) *big.Int {
	if n < 2 {
		return big.NewInt(int64(n))
	}
	a, c := big.NewInt(1), new(big.Int) // F(1), F(0) : k = 1, le bit de poids fort de n
	two := big.NewInt(2)
	for i := bits.Len(uint(n)) - 2; i >= 1; i-- {
		// Formules de GMP : F(2k-1) = F(k)² + F(k-1)², F(2k+1) = 4F(k)² - F(k-1)² + 2(-1)^k, F(2k) = F(2k+1) - F(2k-1).
		sa, sc := new(big.Int).Mul(a, a), new(big.Int).Mul(c, c)
		odd := new(big.Int).Add(sa, sc) // F(2k-1)
		mid := sa.Lsh(sa, 2)            // 4F(k)², sur place
		mid.Sub(mid, sc)
		if n>>(i+1)&1 == 0 { // k est pair
			mid.Add(mid, two)
		} else {
			mid.Sub(mid, two)
		}
		even := sc.Sub(mid, odd) // F(2k), sur place
		if n>>i&1 == 1 {
			a, c = mid, even
		} else {
			a, c = even, odd
		}
	}
	// Dernier bit : seul F(n) est utile. F(2k) = F(k)(F(k) + 2F(k-1)); F(2k+1) = (2F(k) - F(k-1))(2F(k) + F(k-1)) + 2(-1)^k,
	// soit un produit au lieu de deux carrés : environ 30 % de moins pour cette étape à n = 999 999 (garde : PERF-GO-13).
	if n&1 == 0 {
		return new(big.Int).Mul(a, new(big.Int).Add(a, new(big.Int).Lsh(c, 1)))
	}
	// Sur place dans c, mort ici, et dans t : une allocation de moins (rendre c ferait passer son new(big.Int) au tas, à tout n).
	t := new(big.Int).Lsh(a, 1) // 2F(k)
	c.Sub(t, c)                 // 2F(k) - F(k-1) : jamais négatif, car F(k) >= F(k-1)
	t.Lsh(t, 1).Sub(t, c)       // 4F(k) - (2F(k) - F(k-1)) = 2F(k) + F(k-1)
	odd := new(big.Int).Mul(c, t)
	if n>>1&1 == 0 { // k = n >> 1 est pair
		return odd.Add(odd, two)
	}
	return odd.Sub(odd, two)
}
