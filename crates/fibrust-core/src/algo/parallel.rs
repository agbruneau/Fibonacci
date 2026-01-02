use crate::FibNumber;
#[allow(unused_imports)]
use crate::FibOps;

use std::sync::OnceLock;
use std::time::Instant;

use super::fast_doubling::fibonacci_fast_doubling;

/// Global threshold for switching to parallel execution.
///
/// This value is lazily initialized by `calibrate_parallel_threshold`.
static PARALLEL_THRESHOLD: OnceLock<usize> = OnceLock::new();

/// Calibrates the parallel threshold based on system performance.
///
/// Runs a micro-benchmark to estimate single-core performance and combines it with
/// the number of available cores to determine an optimal bit length threshold.
/// Below this threshold, sequential execution is preferred to avoid synchronization overhead.
///
/// # Returns
/// * `usize` - The threshold in bits (approximate).
pub fn calibrate_parallel_threshold() -> usize {
    // Micro-benchmark: Measure single-core performance
    // Calculate F(10,000) using Fast Doubling (iterative)
    // This is large enough to measure but small enough to be fast (< 1ms on modern CPUs)
    let start = Instant::now();
    let _ = fibonacci_fast_doubling(10_000);
    let duration = start.elapsed();

    let micros = duration.as_micros();

    // Heuristic:
    // If CPU is very fast (< 200us), we can afford to stay serial longer to avoid overhead.
    // If CPU is slower (> 500us), parallelism might help earlier (or overhead is relatively smaller).
    // Also factor in core count.

    let cores = rayon::current_num_threads();

    // Base threshold based on core count
    let base_threshold: usize = if cores >= 8 {
        25_000
    } else if cores >= 4 {
        40_000
    } else {
        60_000
    };

    // Adjust based on single-thread performance
    // If single thread is super fast, increase threshold (overhead is expensive relative to compute)
    if micros < 200 {
        base_threshold + 10_000
    } else if micros > 1000 {
        // If single thread is slow, stick to base or slightly lower.
        base_threshold.saturating_sub(5_000)
    } else {
        base_threshold
    }
}

/// Adaptive parallelism threshold - lazily calibrated on first use.
///
/// # Returns
/// * `usize` - The parallel threshold in bits.
pub fn get_parallel_threshold() -> usize {
    *PARALLEL_THRESHOLD.get_or_init(calibrate_parallel_threshold)
}

/// Computes the nth Fibonacci number using Parallel Fast Doubling.
///
/// This is **NOT** classic matrix exponentiation (which uses 8 multiplications per step
/// for the $\begin{pmatrix} 1 & 1 \\ 1 & 0 \end{pmatrix}^n$ matrix). Instead, this is an optimized **Parallel Fast Doubling**
/// implementation that uses Rayon to parallelize the three independent multiplications:
///
/// - `c = a * (2b - a)`   (Thread 1)
/// - `a² = a.pow(2)`      (Thread 2)
/// - `b² = b.pow(2)`      (Thread 3)
///
/// This achieves $O(\log n)$ time complexity with parallelized big-integer multiplications,
/// making it faster than sequential Fast Doubling for large numbers ($>50,000$ bits).
///
/// # Algorithm
///
/// Uses the Fast Doubling identities:
/// - $F(2k) = F(k) * [2F(k+1) - F(k)]$
/// - $F(2k+1) = F(k)^2 + F(k+1)^2$
///
/// With 3-way parallelization for optimal load balancing on multi-core CPUs.
///
/// # Arguments
/// * `n` - The index of the Fibonacci number.
///
/// # Example
/// ```
/// use fibrust_core::fibonacci_parallel;
/// let f = fibonacci_parallel(10000);
/// assert_eq!(f.to_string().len(), 2090); // F(10000) has 2090 digits
/// ```
#[inline]
pub fn fibonacci_parallel(n: u64) -> FibNumber {
    if n == 0 {
        return FibNumber::from(0u32);
    }
    if n == 1 {
        return FibNumber::from(1u32);
    }
    if n == 2 {
        return FibNumber::from(1u32);
    }

    let highest_bit = 63 - n.leading_zeros() as usize;
    let parallel_threshold = get_parallel_threshold();

    let mut a = FibNumber::from(0u32);
    let mut b = FibNumber::from(1u32);

    for i in (0..=highest_bit).rev() {
        let a_bits = a.bit_len();

        let (c, d) = if a_bits > parallel_threshold {
            // Parallel computation for large numbers using 3-way parallelization
            // for optimal load balancing:
            //   Thread 1: c = a * (2b - a)    [1 multiplication]
            //   Thread 2: a² = a.pow(2)       [1 multiplication]
            //   Thread 3: b² = b.pow(2)       [1 multiplication]
            //
            // This reduces critical path from 2 muls to 1 mul when 3+ cores available.
            // Uses nested rayon::join: join(c, join(a², b²))

            let (c, (a_sq, b_sq)) = rayon::join(
                || {
                    // Thread 1: Compute c = a * (2b - a)
                    let two_b = &b << 1;
                    let diff = &two_b - &a;
                    &a * &diff
                },
                || {
                    // Threads 2 & 3: Compute a² and b² in parallel
                    rayon::join(
                        || a.pow(2), // Thread 2: a²
                        || b.pow(2), // Thread 3: b²
                    )
                },
            );
            (c, &a_sq + &b_sq)
        } else {
            // Sequential for smaller numbers
            let two_b = &b << 1;
            let diff = &two_b - &a;
            let c = &a * &diff;
            let a_sq = a.pow(2);
            let b_sq = b.pow(2);
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

/// Deprecated alias for backward compatibility.
///
/// Note: This function is misnamed - it does NOT use classic matrix exponentiation.
/// It actually implements Parallel Fast Doubling. Use [`fibonacci_parallel`] instead.
#[deprecated(since = "0.2.0", note = "Renamed to fibonacci_parallel for accuracy")]
#[inline]
pub fn fibonacci_matrix(n: u64) -> FibNumber {
    fibonacci_parallel(n)
}

#[cfg(test)]
mod tests {
    use super::*;

    // ========================================================================
    // Tests for calibrate_parallel_threshold
    // ========================================================================

    #[test]
    fn calibrate_parallel_threshold_returns_valid_value() {
        let threshold = calibrate_parallel_threshold();
        // Should be a reasonable value based on heuristics (20k-70k range)
        assert!(threshold >= 20_000, "Threshold {} too low", threshold);
        assert!(threshold <= 80_000, "Threshold {} too high", threshold);
    }

    // ========================================================================
    // Tests for get_parallel_threshold
    // ========================================================================

    #[test]
    fn get_parallel_threshold_memoized() {
        // First call initializes
        let threshold1 = get_parallel_threshold();
        // Second call should return same value (memoized)
        let threshold2 = get_parallel_threshold();
        assert_eq!(threshold1, threshold2, "Threshold not memoized");
    }

    #[test]
    fn get_parallel_threshold_consistent() {
        // Multiple calls should always return the same value
        let values: Vec<usize> = (0..5).map(|_| get_parallel_threshold()).collect();
        let first = values[0];
        assert!(
            values.iter().all(|&v| v == first),
            "Threshold values inconsistent"
        );
    }

    // ========================================================================
    // Tests for fibonacci_parallel
    // ========================================================================

    #[test]
    fn fibonacci_parallel_base_cases() {
        assert_eq!(fibonacci_parallel(0), FibNumber::from(0u32));
        assert_eq!(fibonacci_parallel(1), FibNumber::from(1u32));
        assert_eq!(fibonacci_parallel(2), FibNumber::from(1u32));
    }

    #[test]
    fn fibonacci_parallel_known_values() {
        assert_eq!(fibonacci_parallel(10), FibNumber::from(55u32));
        assert_eq!(fibonacci_parallel(20), FibNumber::from(6765u32));
        assert_eq!(fibonacci_parallel(50), FibNumber::from(12586269025u64));
    }

    #[test]
    fn fibonacci_parallel_consistency_with_fast_doubling() {
        // Parallel should produce same results as Fast Doubling
        for n in [0, 1, 2, 10, 100, 500, 1000] {
            let parallel_result = fibonacci_parallel(n);
            let fd_result = fibonacci_fast_doubling(n);
            assert_eq!(parallel_result, fd_result, "Mismatch at n={}", n);
        }
    }

    #[test]
    fn fibonacci_parallel_recurrence() {
        // Verify F(n) + F(n+1) = F(n+2)
        for n in [0, 1, 10, 100, 500] {
            let fn_val = fibonacci_parallel(n);
            let fn1_val = fibonacci_parallel(n + 1);
            let fn2_val = fibonacci_parallel(n + 2);
            assert_eq!(&fn_val + &fn1_val, fn2_val, "Recurrence failed at n={}", n);
        }
    }

    #[test]
    fn fibonacci_parallel_medium_value() {
        // Test a medium value with known digit count
        let f1000 = fibonacci_parallel(1000);
        assert_eq!(f1000.to_string().len(), 209);
    }

    // ========================================================================
    // Tests for deprecated fibonacci_matrix alias
    // ========================================================================

    #[test]
    #[allow(deprecated)]
    fn fibonacci_matrix_alias_works() {
        // Deprecated alias should still work
        assert_eq!(fibonacci_matrix(10), fibonacci_parallel(10));
        assert_eq!(fibonacci_matrix(100), fibonacci_parallel(100));
    }
}
