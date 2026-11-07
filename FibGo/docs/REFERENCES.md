# References

Numbered bibliography for the algorithm documentation. Other pages cite an entry as
`[n]`, linking to its anchor here (for example [`REFERENCES.md#ref-1`](#ref-1)).

Every identifier below was resolved while this list was written, on 2026-09-23:
each DOI answered `https://doi.org/<doi>` with `302 Found` to the publisher named
in the entry, and the bibliographic fields were checked against the DOI's Crossref
record (`https://api.crossref.org/works/<doi>`) or, where there is no DOI, against
the page linked. Section numbers inside books are the one exception: they were not
re-checked against a copy of the text, and say so.

## Integer multiplication

<a id="ref-1"></a>**[1]** A. Schönhage and V. Strassen, "Schnelle Multiplikation großer Zahlen",
*Computing* 7(3–4), 1971, pp. 281–292. DOI [10.1007/BF02242355](https://doi.org/10.1007/BF02242355)
(→ link.springer.com).
The Fermat-ring FFT multiplication that `internal/bigfft` is built on, and the
O(n log n log log n) bound — a bound for the *recursive* algorithm, which this repository
does not implement ([FFT.md § Complexity Analysis](algorithms/FFT.md#complexity-analysis)).

<a id="ref-2"></a>**[2]** D. Harvey and J. van der Hoeven, "Integer multiplication in time O(n log n)",
*Annals of Mathematics* 193(2), 2021, pp. 563–617. DOI
[10.4007/annals.2021.193.2.4](https://doi.org/10.4007/annals.2021.193.2.4) (→ projecteuclid.org).
The O(n log n) bound Schönhage and Strassen conjectured. Not implemented here; GMP 6.3.0 [11]
also tops out at a Schönhage–Strassen FFT "at large to very large sizes". Cited only so that "O(n log n)" is attached to the algorithm that achieves it.

<a id="ref-3"></a>**[3]** A. Karatsuba and Yu. Ofman, "Multiplication of many-digital numbers by automatic
computers", *Doklady Akademii Nauk SSSR* 145(2), 1962, pp. 293–294 (in Russian). No DOI;
Math-Net.Ru record <http://mi.mathnet.ru/dan26729> (302 → mathnet.ru).
The O(n^log2 3) ≈ O(n^1.585) method `math/big` uses [13]. The English translation in
*Soviet Physics Doklady* is not listed because its volume and pages could not be verified.

<a id="ref-4"></a>**[4]** J. W. Cooley and J. W. Tukey, "An algorithm for the machine calculation of complex
Fourier series", *Mathematics of Computation* 19(90), 1965, pp. 297–301. DOI
[10.1090/S0025-5718-1965-0178586-1](https://doi.org/10.1090/S0025-5718-1965-0178586-1) (→ ams.org).
The radix-2 transform structure; `internal/bigfft` runs it over Z/(2^n'+1), where the
twiddle factors are powers of 2 and every butterfly is a shift, an add and a subtract.

## Matrix multiplication

<a id="ref-5"></a>**[5]** V. Strassen, "Gaussian elimination is not optimal", *Numerische Mathematik* 13(4),
1969, pp. 354–356. DOI [10.1007/BF02165411](https://doi.org/10.1007/BF02165411) (→ link.springer.com).
Seven products for a 2×2 matrix product instead of eight.

<a id="ref-6"></a>**[6]** S. Winograd, "On multiplication of 2 × 2 matrices", *Linear Algebra and its
Applications* 4(4), 1971, pp. 381–388. DOI
[10.1016/0024-3795(71)90009-7](https://doi.org/10.1016/0024-3795(71)90009-7) (→ linkinghub.elsevier.com).
Seven multiplications with 15 additions/subtractions — the variant in
`multiplyMatrixStrassen` (`internal/fibonacci/matrix_ops.go`).

## Fibonacci numbers

<a id="ref-7"></a>**[7]** D. Takahashi, "A fast algorithm for computing large Fibonacci numbers",
*Information Processing Letters* 75(6), 2000, pp. 243–246. DOI
[10.1016/S0020-0190(00)00112-5](https://doi.org/10.1016/S0020-0190(00)00112-5) (→ linkinghub.elsevier.com).
Doubling through F(k) and the Lucas number L(k) with **two squarings per bit** and one
final product; under the paper's FFT cost model (a square ≈ 2/3 of a product, three
transforms against two) the total is at most (7/6)·M(γn), γ = log2 φ ≈ 0.69424. The
comparison with this repository's loop is in
[FAST_DOUBLING.md § Complexity Analysis](algorithms/FAST_DOUBLING.md#complexity-analysis).

<a id="ref-8"></a>**[8]** D. E. Knuth, *The Art of Computer Programming*, Vol. 1: *Fundamental Algorithms*,
3rd ed., Addison-Wesley, Reading, MA, 1997. ISBN 0-201-89683-4 (edition and ISBN checked on
<https://www-cs-faculty.stanford.edu/~knuth/taocp.html>). §1.2.8 (Fibonacci numbers):
Q-matrix powers, the addition formula behind the doubling identities, and Cassini's
identity F(n+1)F(n−1) − F(n)² = (−1)^n. *Section number not re-checked against the text.*

<a id="ref-9"></a>**[9]** D. E. Knuth, *The Art of Computer Programming*, Vol. 2: *Seminumerical Algorithms*,
3rd ed., Addison-Wesley, Reading, MA, 1997. ISBN 0-201-89684-2 (same page). §4.3.3
(fast multiplication: Karatsuba, Toom–Cook, Schönhage–Strassen) and §4.6.3 (evaluation of
powers, i.e. binary exponentiation). *Section numbers not re-checked against the text.*

## Surveys and reference implementations

<a id="ref-10"></a>**[10]** R. P. Brent and P. Zimmermann, *Modern Computer Arithmetic*, Cambridge
Monographs on Applied and Computational Mathematics 18, Cambridge University Press, 2010.
DOI [10.1017/CBO9780511921698](https://doi.org/10.1017/CBO9780511921698) (→ cambridge.org).
Chapter 2, "Modular arithmetic and the FFT" (pp. 47–78), for Schönhage–Strassen in the
Fermat-ring form (chapter title and pages checked on the publisher page; the section
inside it was not).

<a id="ref-11"></a>**[11]** *GNU MP: The GNU Multiple Precision
Arithmetic Library*, manual for version 6.3.0. §15.1.6 "FFT Multiplication",
<https://gmplib.org/manual/FFT-Multiplication>; §15.7.4 "Fibonacci Numbers",
<https://gmplib.org/manual/Fibonacci-Numbers-Algorithm> (both pages read on 2026-09-23 and
labelled 6.3.0). The benchmark implementation: a recursive Fermat FFT whose pointwise products
recurse when that is faster, and `mpz_fib_ui` at two squarings per bit.

<a id="ref-12"></a>**[12]** R. Oudompheng, `bigfft` — "a toy proof-of-concept implementation of the well-known
Schonhage-Strassen method for multiplying integers" (its README), BSD-3-Clause.
<https://github.com/remyoudompheng/bigfft>. The upstream of `internal/bigfft`.

<a id="ref-13"></a>**[13]** The Go Authors, package `math/big`, `src/math/big/natmul.go` at `go1.26.1`
(the `go` directive of this module's `go.mod`),
<https://github.com/golang/go/blob/go1.26.1/src/math/big/natmul.go>. Schoolbook
multiplication below `karatsubaThreshold = 40` words, Karatsuba [3] above; no Toom–Cook and
no FFT. Both the `math/big` tier and the pointwise products of `internal/bigfft` end here.

<a id="ref-14"></a>**[14]** Project Nayuki, "Fast Fibonacci algorithms",
<https://www.nayuki.io/page/fast-fibonacci-algorithms> (page dated 2023-01-22). States the
doubling pair F(2k) = F(k)·[2F(k+1) − F(k)], F(2k+1) = F(k+1)² + F(k)² in the form this
repository uses.

## Software engineering

<a id="ref-15"></a>**[15]** S. Shahsavan, *Building Enterprise Projects with Go: Clarity at Scale in
Production-Grade Go Systems*, Apress, 2026. ISBN 979-8-8688-2369-5 (print),
979-8-8688-2370-1 (e-book). DOI [10.1007/979-8-8688-2370-1](https://doi.org/10.1007/979-8-8688-2370-1)
(→ link.springer.com). The yardstick of the 2026-09 audit
([`audits/audit-2026-09-livre.md`](audits/audit-2026-09-livre.md)); not an algorithmic source.
