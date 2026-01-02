use crate::FibNumber;
#[allow(unused_imports)]
use crate::FibOps;

use rayon::iter::{
    IndexedParallelIterator, IntoParallelRefIterator, IntoParallelRefMutIterator, ParallelIterator,
};
use rustfft::{num_complex::Complex64, FftPlanner};
use std::cell::RefCell;

use crate::config::{fft as fft_config, thresholds};

thread_local! {
    /// Thread-local FFT planner to reuse scratch space and precomputed roots of unity.
    static FFT_PLANNER: RefCell<FftPlanner<f64>> = RefCell::new(FftPlanner::new());
}

/// Helper to initialize thread-local planner (used by pre-warming).
///
/// Ensures that the FFT planner is initialized on the current thread, preventing
/// lazy initialization overhead during the first calculation.
///
/// # Usage
/// This is typically called by `prewarm_system` on all worker threads.
pub fn prewarm_fft_planner() {
    FFT_PLANNER.with(|p| {
        let _ = p.borrow_mut();
    });
}

/// Workspace for FFT operations to reuse memory buffers.
struct FftWorkspace {
    a_complex: Vec<Complex64>,
    b_complex: Vec<Complex64>,
    c_complex: Vec<Complex64>,
    d_complex: Vec<Complex64>,
}

impl FftWorkspace {
    fn new() -> Self {
        Self {
            a_complex: Vec::new(),
            b_complex: Vec::new(),
            c_complex: Vec::new(),
            d_complex: Vec::new(),
        }
    }

    /// Resizes all vectors to the given size, initializing new elements with zero.
    ///
    /// # Optimization
    /// Instead of zeroing the entire vector (O(N)), we only zero the tail starting from `data_len`.
    /// The head `0..data_len` will be overwritten by `copy_to_complex` anyway.
    fn prepare(&mut self, size: usize, data_len_a: usize, data_len_b: usize) {
        // Ensure capacity and correct size
        if self.a_complex.len() != size {
            self.a_complex.resize(size, Complex64::new(0.0, 0.0));
            self.b_complex.resize(size, Complex64::new(0.0, 0.0));
            self.c_complex.resize(size, Complex64::new(0.0, 0.0));
            self.d_complex.resize(size, Complex64::new(0.0, 0.0));
        }

        // Only zero the padded region (tail), not the whole vector.
        // We assume the caller will overwrite 0..data_len with valid data.
        // For 'a' and 'b', we zero from data_len to end.
        if data_len_a < size {
            self.a_complex[data_len_a..].fill(Complex64::new(0.0, 0.0));
        }
        if data_len_b < size {
            self.b_complex[data_len_b..].fill(Complex64::new(0.0, 0.0));
        }

        // Output buffers 'c' and 'd' are fully written by FFT process/pointwise mul,
        // but since they are used as scratch by FFT, their initial state might matter if "process" assumes something?
        // RustFFT's process takes input and produces output in-place.
        // However, we use `c` and `d` to store results of pointwise mul.
        // Then we run IFFT on them.
        // So we don't need to zero them at all, because we overwrite them fully during pointwise mul loop?
        // Wait, pointwise mul iterates 0..size. So yes, we overwrite fully.
        // So zeroing c/d is unnecessary overhead.
    }
}

/// Computes the nth Fibonacci number using FFT-accelerated Fast Doubling.
///
/// This function utilizes **Fast Fourier Transform (FFT)** for large integer multiplication.
///
/// # When to use
///
/// This algorithm is optimal for **very large inputs** (typically $n > 200,000$,
/// corresponding to results with millions of bits). At this scale, the $O(n \log n \log \log n)$
/// complexity of Schönhage-Strassen (FFT) multiplication outperforms the $O(n^{1.585})$
/// complexity of Karatsuba used by standard libraries.
///
/// # Algorithm
///
/// It modifies the standard Fast Doubling algorithm by replacing the scalar BigInt multiplications
/// with FFT-based polynomial multiplication:
/// 1.  Convert large integers into polynomials (vectors of digits in a specific base).
/// 2.  Apply Forward FFT to transform polynomials to the frequency domain.
/// 3.  Perform point-wise multiplication in the frequency domain.
/// 4.  Apply Inverse FFT to transform back to the time domain.
/// 5.  Apply carry propagation to reconstruct the resulting large integer.
///
/// A "Unified FFT Step" is used to compute $(F(2k), F(2k+1))$ simultaneously,
/// reducing the total number of transforms required from 7 to 4.
///
/// # Arguments
/// * `n` - The index of the Fibonacci number.
#[inline]
pub fn fibonacci_fft(n: u64) -> FibNumber {
    if n == 0 {
        return FibNumber::from(0u32);
    }
    if n == 1 {
        return FibNumber::from(1u32);
    }

    let highest_bit = 63 - n.leading_zeros() as usize;

    let mut a = FibNumber::from(0u32);
    let mut b = FibNumber::from(1u32);

    // FFT becomes beneficial when numbers have many bits.
    // Threshold from config::thresholds::FFT_BIT_THRESHOLD

    // Workspace for reusing large vectors across iterations
    let mut workspace = FftWorkspace::new();

    for i in (0..=highest_bit).rev() {
        let a_bits = a.bit_len();
        let b_bits = b.bit_len();

        let (c, d) =
            if a_bits > thresholds::FFT_BIT_THRESHOLD || b_bits > thresholds::FFT_BIT_THRESHOLD {
                // Use unified FFT step to compute (F(2k), F(2k+1)) with minimal transforms
                unified_fft_step(&a, &b, &mut workspace)
            } else {
                // Standard multiplication for smaller numbers
                let two_b = &b << 1;
                let diff = &two_b - &a;
                let c = &a * &diff;
                let a_sq = &a * &a;
                let b_sq = &b * &b;
                (c, &a_sq + &b_sq)
            };

        if (n >> i) & 1 == 0 {
            a = c;
            b = d;
        } else {
            b = c + &d;
            a = d;
        }
    }

    a
}

/// Unified FFT step for doubling.
///
/// Computes $(F(2k), F(2k+1)) = (a(2b - a), a^2 + b^2)$
/// by reusing FFT representations of $a$ and $b$.
///
/// # Complexity
/// 2 Forward FFTs + 2 Inverse FFTs = **4 Transforms** (compared to 7 naive multiplications).
#[inline]
fn unified_fft_step(
    a: &FibNumber,
    b: &FibNumber,
    workspace: &mut FftWorkspace,
) -> (FibNumber, FibNumber) {
    if a.bit_len() == 0 {
        return (FibNumber::from(0u32), b.pow(2));
    }

    // Determine optimal base bits to ensure precision
    // Constraint: 2*BASE_BITS + log2(fft_size) < 53
    // For n=2e9, bits ~ 1.4e9.
    // With Base 12: Digits ~ 1.4e9 / 12 ~ 116M. FFT Size 2^27 or 2^28.
    // Max FFT Size for 2e9 is 2^28 (268M).
    // 2*12 + 28 = 24 + 28 = 52 < 53. Safe.
    // Base 13: 2*13 + 28 = 54 > 53. Unsafe.
    // So Base 12 is optimal for massive numbers (better than 11).
    //
    // Heuristic: If we expect massive numbers, use safer (smaller) base.
    // Threshold: If bits > 100M, use base 12. Otherwise base 13.
    let approx_bits = a.bit_len() + b.bit_len();
    let base_bits = if approx_bits > fft_config::MASSIVE_THRESHOLD {
        fft_config::BASE_BITS_MASSIVE
    } else {
        fft_config::BASE_BITS_DEFAULT
    };

    let base = 1u64 << base_bits;

    // Parallelize base conversion for massive inputs.
    // This halves the time spent in the "preparation" phase for inputs > 1GB,
    // where base conversion is a significant serial bottleneck.
    let (a_digits, b_digits) = rayon::join(
        || ubig_to_digits(a, base_bits),
        || ubig_to_digits(b, base_bits),
    );

    let result_len = a_digits.len() + b_digits.len();
    let fft_size = result_len.next_power_of_two();

    // Resize workspace vectors and zero ONLY tail.
    workspace.prepare(fft_size, a_digits.len(), b_digits.len());

    FFT_PLANNER.with(|planner| {
        let mut planner = planner.borrow_mut();
        let fft = planner.plan_fft_forward(fft_size);
        let ifft = planner.plan_fft_inverse(fft_size);

        // Copy digits to complex vectors
        // Note: prepare() already zero-filled them, so we just copy the valid data.
        // Parallelizing this copy offers minor speedup for huge arrays.
        let copy_to_complex = |dest: &mut [Complex64], src: &[u32]| {
            for (i, &d) in src.iter().enumerate() {
                dest[i] = Complex64::new(d as f64, 0.0);
            }
        };

        rayon::join(
            || copy_to_complex(&mut workspace.a_complex[0..a_digits.len()], &a_digits),
            || copy_to_complex(&mut workspace.b_complex[0..b_digits.len()], &b_digits),
        );

        // Forward FFT(a) & FFT(b)
        // We can run these in parallel if planner allows, but FftPlanner is RefCell.
        // However, the `fft` instance (Arc<dyn Fft>) is thread-safe.
        // The issue is `process` needs `&mut [Complex64]`.
        // We have distinct mutable references to a_complex and b_complex.
        // So we can parallelize.
        // Scratch space: `process` allocates its own scratch.
        let a_complex = &mut workspace.a_complex;
        let b_complex = &mut workspace.b_complex;

        rayon::join(|| fft.process(a_complex), || fft.process(b_complex));

        // Compute frequencies
        // c = a * (2b - a)
        // d = a^2 + b^2
        let c_complex = &mut workspace.c_complex;
        let d_complex = &mut workspace.d_complex;

        // Process in parallel chunks
        c_complex
            .par_iter_mut()
            .zip(d_complex.par_iter_mut())
            .zip(a_complex.par_iter())
            .zip(b_complex.par_iter())
            .for_each(|(((cc, dc), &ac), &bc)| {
                // c = a * (2b - a)
                let diff = (bc * 2.0) - ac;
                *cc = ac * diff;

                // d = a^2 + b^2
                *dc = (ac * ac) + (bc * bc);
            });

        // Inverse FFTs
        rayon::join(|| ifft.process(c_complex), || ifft.process(d_complex));

        let scale = fft_size as f64;
        let base_i64 = base as i64;
        let base_mask = base_i64 - 1;

        // Closure to process IFFT results back to UBig
        // Optimized to use parallelism for the expensive rounding step
        let process_result = |complex_data: &[Complex64]| -> FibNumber {
            // We can't easily reuse result_digits vector without passing it in or putting it in workspace.
            // But UBig creation from digits consumes the vector usually.
            // Allocating result digits (Vec<u64>) is relatively cheap (1.6GB for 2B input) compared to Complex64.
            // Let's keep it local for now to avoid complexity with UBig internals.

            let mut result_digits: Vec<u64> = vec![0; result_len + 2];

            // Optimization: Parallelize the rounding step.
            // (c.re / scale).round() involves floating point div and round, which is expensive.
            // We can do this in parallel into a temporary buffer, then do carry propagation sequentially.
            // This transforms the main loop from Serial(Float + Carry) to Parallel(Float) + Serial(Carry).

            // Using a temporary buffer for rounded values (i64)
            // Note: Parallel iteration requires random access or collect.
            // We can iterate complex_data in parallel and write to a pre-allocated buffer.
            // But wait, carrying needs the previous value.
            // We split into:
            // 1. Parallel Rounding -> Vec<i64>
            // 2. Sequential Carry -> Vec<u64> (result_digits)

            // Step 1: Parallel Rounding
            let rounded_values: Vec<i64> = complex_data[..result_len]
                .par_iter()
                .map(|c| (c.re / scale).round() as i64)
                .collect();

            // Step 2: Sequential Carry Propagation (very fast integer ops)
            let mut carry: i64 = 0;
            for (i, &val_rounded) in rounded_values.iter().enumerate() {
                let val = val_rounded + carry;
                let digit = val & base_mask;
                carry = val >> base_bits;
                result_digits[i] = digit as u64;
            }

            let mut idx = result_len;
            while carry != 0 {
                let val = carry;
                let digit = val & base_mask;
                carry = val >> base_bits;

                if idx < result_digits.len() {
                    result_digits[idx] = digit as u64;
                } else {
                    result_digits.push(digit as u64);
                }
                idx += 1;
            }

            digits_to_ubig(&result_digits, base_bits)
        };

        // Reconstruct both results
        rayon::join(|| process_result(c_complex), || process_result(d_complex))
    })
}

/// Converts a FibNumber into a vector of digits in the specified base bits.
///
/// # Arguments
/// * `n` - The number to convert.
/// * `base_bits` - The number of bits per digit (e.g., 14).
fn ubig_to_digits(n: &FibNumber, base_bits: usize) -> Vec<u32> {
    let bytes = n.to_le_bytes();
    ubig_to_digits_sequential(&bytes, base_bits)
}

/// Sequential conversion of bytes to digits (used for smaller inputs).
#[inline]
fn ubig_to_digits_sequential(bytes: &[u8], base_bits: usize) -> Vec<u32> {
    let mut result = Vec::with_capacity((bytes.len() * 8).div_ceil(base_bits));

    let mask = (1u64 << base_bits) - 1;
    let mut bits: u64 = 0;
    let mut bit_count = 0;

    for &byte in bytes {
        bits |= (byte as u64) << bit_count;
        bit_count += 8;

        while bit_count >= base_bits {
            result.push((bits & mask) as u32);
            bits >>= base_bits;
            bit_count -= base_bits;
        }
    }

    if bit_count > 0 || result.is_empty() {
        result.push(bits as u32);
    }

    // Trim leading zeros
    while result.len() > 1 && result.last() == Some(&0) {
        result.pop();
    }

    result
}

/// Reconstructs a FibNumber from a vector of digits.
///
/// # Arguments
/// * `digits` - The vector of digits.
/// * `base_bits` - The number of bits per digit.
fn digits_to_ubig(digits: &[u64], base_bits: usize) -> FibNumber {
    if digits.is_empty() {
        return FibNumber::from(0u32);
    }
    digits_to_ubig_sequential(digits, base_bits)
}

/// Sequential conversion of digits to FibNumber (used for smaller inputs).
#[inline]
fn digits_to_ubig_sequential(digits: &[u64], base_bits: usize) -> FibNumber {
    let mut bytes: Vec<u8> = Vec::with_capacity(digits.len() * base_bits / 8 + 1);
    let mut bits: u64 = 0;
    let mut bit_count = 0;

    for &d in digits {
        bits |= d << bit_count;
        bit_count += base_bits;

        while bit_count >= 8 {
            bytes.push((bits & 0xFF) as u8);
            bits >>= 8;
            bit_count -= 8;
        }
    }

    if bit_count > 0 {
        bytes.push(bits as u8);
    }

    FibNumber::from_le_bytes(&bytes)
}

#[cfg(test)]
mod tests {
    use super::*;

    // ========================================================================
    // Tests for prewarm_fft_planner
    // ========================================================================

    #[test]
    fn prewarm_fft_planner_no_panic() {
        // Should initialize without panic
        prewarm_fft_planner();
        // Call again to verify idempotency
        prewarm_fft_planner();
    }

    // ========================================================================
    // Tests for ubig_to_digits
    // ========================================================================

    #[test]
    fn ubig_to_digits_zero() {
        let zero = FibNumber::from(0u32);
        let digits = ubig_to_digits(&zero, 14);
        // Zero should have at least one digit (0)
        assert_eq!(digits, vec![0]);
    }

    #[test]
    fn ubig_to_digits_small_values() {
        // Test with small values that fit in one digit
        let one = FibNumber::from(1u32);
        let digits = ubig_to_digits(&one, 14);
        assert_eq!(digits, vec![1]);

        let hundred = FibNumber::from(100u32);
        let digits = ubig_to_digits(&hundred, 14);
        assert_eq!(digits, vec![100]);
    }

    #[test]
    fn ubig_to_digits_large_value() {
        // Test with a value that spans multiple digits
        // 2^14 = 16384 should split into two 14-bit digits
        let val = FibNumber::from(16384u32);
        let digits = ubig_to_digits(&val, 14);
        assert_eq!(digits, vec![0, 1]); // 16384 = 0 + 1*2^14
    }

    // ========================================================================
    // Tests for digits_to_ubig
    // ========================================================================

    #[test]
    fn digits_to_ubig_empty() {
        let digits: Vec<u64> = vec![];
        let result = digits_to_ubig(&digits, 14);
        assert_eq!(result, FibNumber::from(0u32));
    }

    #[test]
    fn digits_to_ubig_single() {
        let digits = vec![42u64];
        let result = digits_to_ubig(&digits, 14);
        assert_eq!(result, FibNumber::from(42u32));
    }

    #[test]
    fn digits_to_ubig_multiple() {
        // 0 + 1*2^14 = 16384
        let digits = vec![0u64, 1];
        let result = digits_to_ubig(&digits, 14);
        assert_eq!(result, FibNumber::from(16384u32));
    }

    #[test]
    fn ubig_digits_round_trip() {
        // Round-trip test: UBig -> digits -> UBig
        let test_values = [
            FibNumber::from(0u32),
            FibNumber::from(1u32),
            FibNumber::from(12345u32),
            FibNumber::from(u64::MAX),
            FibNumber::from(u128::MAX),
        ];

        for val in test_values {
            let digits = ubig_to_digits(&val, 14);
            let digits_u64: Vec<u64> = digits.iter().map(|&d| d as u64).collect();
            let recovered = digits_to_ubig(&digits_u64, 14);
            assert_eq!(recovered, val, "Round-trip failed for {:?}", val);
        }
    }

    // ========================================================================
    // Tests for unified_fft_step
    // ========================================================================

    #[test]
    fn unified_fft_step_zero_a() {
        // When a=0: F(2k) = 0*(2b-0) = 0, F(2k+1) = 0 + b² = b²
        let a = FibNumber::from(0u32);
        let b = FibNumber::from(5u32);
        let mut workspace = FftWorkspace::new();
        let (c, d) = unified_fft_step(&a, &b, &mut workspace);
        assert_eq!(c, FibNumber::from(0u32));
        assert_eq!(d, FibNumber::from(25u32)); // 5²
    }

    #[test]
    fn unified_fft_step_known_values() {
        // F(0) = 0, F(1) = 1
        // unified_fft_step computes (F(2k), F(2k+1)) from (F(k), F(k+1))
        // For k=0: a=F(0)=0, b=F(1)=1 -> F(0)=0, F(1)=1
        let a = FibNumber::from(0u32);
        let b = FibNumber::from(1u32);
        let mut workspace = FftWorkspace::new();
        let (c, d) = unified_fft_step(&a, &b, &mut workspace);
        // c = a*(2b-a) = 0*(2-0) = 0 = F(0)
        // d = a² + b² = 0 + 1 = 1 = F(1)
        assert_eq!(c, FibNumber::from(0u32));
        assert_eq!(d, FibNumber::from(1u32));
    }

    // ========================================================================
    // Tests for fibonacci_fft
    // ========================================================================

    #[test]
    fn fibonacci_fft_base_cases() {
        assert_eq!(fibonacci_fft(0), FibNumber::from(0u32));
        assert_eq!(fibonacci_fft(1), FibNumber::from(1u32));
        assert_eq!(fibonacci_fft(2), FibNumber::from(1u32));
    }

    #[test]
    fn fibonacci_fft_known_values() {
        assert_eq!(fibonacci_fft(10), FibNumber::from(55u32));
        assert_eq!(fibonacci_fft(20), FibNumber::from(6765u32));
        assert_eq!(fibonacci_fft(50), FibNumber::from(12586269025u64));
    }

    #[test]
    fn fibonacci_fft_consistency_with_fast_doubling() {
        use crate::algo::fast_doubling::fibonacci_fast_doubling;

        // Test various values to ensure FFT produces same results
        for n in [0, 1, 10, 100, 500, 1000] {
            let fft_result = fibonacci_fft(n);
            let fd_result = fibonacci_fast_doubling(n);
            assert_eq!(fft_result, fd_result, "Mismatch at n={}", n);
        }
    }

    #[test]
    fn fibonacci_fft_medium_value() {
        // Test a medium value that exercises more of the algorithm
        let f1000 = fibonacci_fft(1000);
        // Verify digit count (F(1000) has 209 digits)
        assert_eq!(f1000.to_string().len(), 209);
    }
}
