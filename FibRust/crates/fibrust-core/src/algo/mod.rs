//! Algorithm module containing different Fibonacci implementation strategies.
//!
//! This module groups the various algorithms provided by `fibrust-core`. Each submodule
//! implements a specific strategy optimized for different scales of input.
//!
//! # Strategies
//!
//! - **Fast Doubling (`fast_doubling`)**: The baseline $O(\log n)$ algorithm.
//! - **Parallel Fast Doubling (`parallel`)**: Exploits multi-core parallelism for large inputs.
//! - **FFT-based (`fft`)**: Uses Fast Fourier Transform for massive inputs.
//! - **Adaptive**: A smart selector that chooses the best strategy.

use crate::config::{limits, thresholds};
use crate::{FibError, FibNumber};

pub mod fast_doubling;
pub mod fft;
pub mod parallel;
pub mod progress;

pub use fast_doubling::{fibonacci, fibonacci_fast_doubling};
pub use fft::fibonacci_fft;
pub use parallel::fibonacci_parallel;

// Re-export deprecated alias for backward compatibility
#[allow(deprecated)]
pub use parallel::fibonacci_matrix;

/// Adaptive algorithm selection for optimal Fibonacci calculation.
///
/// Automatically selects the best algorithm based on input size.
///
/// # Threshold Justifications
///
/// - **$n < 40,000$**: **Fast Doubling** (Sequential).
///   Benchmarks show that for small inputs, the overhead of Rayon's thread pool management
///   and task splitting in the parallel implementation outweighs the benefits of parallel
///   multiplication. The sequential $O(\log n)$ algorithm is most efficient here.
///
/// - **$40,000 \le n < 200,000$**: **Parallel Fast Doubling**.
///   In this range, the cost of arbitrary-precision integer multiplication grows significantly.
///   Parallelizing the three large multiplications in the doubling step ($F(k)^2$, $F(k+1)^2$, etc.)
///   across available cores provides a net speedup despite the synchronization overhead.
///
/// - **$n \ge 200,000$**: **FFT-based Multiplication**.
///   For massive numbers (millions of bits), the $O(n \log n)$ complexity of FFT-based multiplication
///   becomes superior to the $O(n^{1.585})$ Karatsuba/Toom-Cook algorithms used in `ibig`.
///   Our custom Schönhage-Strassen implementation minimizes asymptotic complexity.
///
/// # Arguments
/// * `n` - The index of the Fibonacci number.
///
/// # Example
/// ```
/// use fibrust_core::fibonacci_adaptive;
///
/// let result = fibonacci_adaptive(1000);
/// assert_eq!(result.to_string().len(), 209); // F(1000) has 209 digits
/// ```
#[inline]
pub fn fibonacci_adaptive(n: u64) -> FibNumber {
    try_fibonacci_adaptive(n).expect("Fibonacci calculation failed")
}

/// Adaptive algorithm selection with explicit error handling.
///
/// This function performs the same logic as `fibonacci_adaptive` but returns a `Result`
/// to handle potential errors, such as inputs exceeding safe memory limits.
///
/// # Arguments
/// * `n` - The index of the Fibonacci number.
///
/// # Errors
/// * `FibError::InputTooLarge` if `n` exceeds a safe default limit (currently set conservatively at $10^{12}$).
/// * `FibError::MemoryLimitExceeded` if the estimated memory requirement exceeds the default safe limit (8 GB).
///
/// # Example
/// ```
/// use fibrust_core::{try_fibonacci_adaptive, FibError};
///
/// match try_fibonacci_adaptive(1_000_000) {
///     Ok(val) => println!("Success!"),
///     Err(e) => eprintln!("Error: {}", e),
/// }
/// ```
pub fn try_fibonacci_adaptive(n: u64) -> Result<FibNumber, FibError> {
    if n > limits::MAX_SAFE_N {
        return Err(FibError::InputTooLarge {
            n,
            max: limits::MAX_SAFE_N,
        });
    }

    let estimated_bytes = crate::estimate_memory_bytes(n);
    if estimated_bytes > limits::SAFE_MEMORY_BYTES {
        return Err(FibError::MemoryLimitExceeded {
            required_bytes: estimated_bytes,
            limit_bytes: limits::SAFE_MEMORY_BYTES,
        });
    }

    Ok(if n < thresholds::PARALLEL_CROSSOVER {
        // n < 40,000: Fast Doubling (includes u128 fast path for n <= 186)
        fibonacci_fast_doubling(n)
    } else if n < thresholds::FFT_CROSSOVER {
        // 40,000 ≤ n < 200,000: Parallel Fast Doubling (multicore advantage)
        fibonacci_parallel(n)
    } else {
        // n ≥ 200,000: FFT-based multiplication
        fibonacci_fft(n)
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    // ========================================================================
    // Tests for fibonacci_adaptive
    // ========================================================================

    #[test]
    fn fibonacci_adaptive_base_cases() {
        assert_eq!(fibonacci_adaptive(0), FibNumber::from(0u32));
        assert_eq!(fibonacci_adaptive(1), FibNumber::from(1u32));
        assert_eq!(fibonacci_adaptive(2), FibNumber::from(1u32));
    }

    #[test]
    fn fibonacci_adaptive_known_values() {
        assert_eq!(fibonacci_adaptive(10), FibNumber::from(55u32));
        assert_eq!(fibonacci_adaptive(20), FibNumber::from(6765u32));
    }

    #[test]
    fn fibonacci_adaptive_fast_doubling_range() {
        // n < 40,000 should use Fast Doubling
        let n = 39_999;
        let result = fibonacci_adaptive(n);
        let expected = fibonacci_fast_doubling(n);
        assert_eq!(result, expected, "Fast Doubling range mismatch");
    }

    #[test]
    fn fibonacci_adaptive_parallel_range() {
        // 40,000 <= n < 200,000 should use Parallel Fast Doubling
        let n = 40_000;
        let adaptive_result = fibonacci_adaptive(n);
        let parallel_result = fibonacci_parallel(n);
        assert_eq!(
            adaptive_result, parallel_result,
            "Parallel crossover mismatch"
        );

        // Also test within the range
        let n2 = 100_000;
        let adaptive_result2 = fibonacci_adaptive(n2);
        let parallel_result2 = fibonacci_parallel(n2);
        assert_eq!(
            adaptive_result2, parallel_result2,
            "Parallel range mismatch"
        );
    }

    #[test]
    fn fibonacci_adaptive_fft_range() {
        // n >= 200,000 should use FFT
        let n = 200_000;
        let adaptive_result = fibonacci_adaptive(n);
        let fft_result = fibonacci_fft(n);
        assert_eq!(adaptive_result, fft_result, "FFT crossover mismatch");
    }

    #[test]
    fn fibonacci_adaptive_consistency() {
        // Adaptive should always produce correct results across all ranges
        // Compare against fast_doubling which is the reference implementation
        for n in [0, 1, 100, 1000, 10_000, 39_999] {
            let adaptive = fibonacci_adaptive(n);
            let fd = fibonacci_fast_doubling(n);
            assert_eq!(adaptive, fd, "Consistency check failed at n={}", n);
        }
    }
}
