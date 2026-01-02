//! # FibRust Core
//!
//! High-performance Fibonacci algorithms library for Rust.
//!
//! This crate provides a suite of optimized algorithms for computing Fibonacci numbers,
//! ranging from efficient scalar implementations to massively parallelized and FFT-based approaches.
//!
//! ## Algorithms
//!
//! - **Fast Doubling**: $O(\log n)$ time complexity. Uses the identity $F(2k) = F(k)(2F(k+1) - F(k))$ and $F(2k+1) = F(k)^2 + F(k+1)^2$. Ideal for $n < 40,000$.
//! - **Parallel Fast Doubling**: Parallelizes the large integer multiplications in the Fast Doubling algorithm. Ideal for $40,000 \le n < 200,000$.
//! - **FFT-based**: Uses Fast Fourier Transform for integer multiplication. Ideal for extremely large $n$ (e.g., $n \ge 200,000$).
//! - **Adaptive**: Automatically selects the best algorithm based on the input size $n$ and available system resources.
//!
//! ## Usage
//!
//! Add this to your `Cargo.toml`:
//!
//! ```toml
//! [dependencies]
//! fibrust-core = "0.1"
//! ```
//!
//! ### Basic Example
//!
//! ```rust
//! use fibrust_core::fibonacci_adaptive;
//!
//! let n = 1000;
//! let result = fibonacci_adaptive(n);
//! println!("F({}) = {}", n, result);
//! ```
//!
//! ### Parallel Range Calculation
//!
//! ```rust
//! use fibrust_core::fib_range_parallel;
//!
//! // Calculate F(1000) to F(1009) in parallel
//! let start = 1000;
//! let end = 1010;
//! let results = fib_range_parallel(start, end, 0); // 0 = auto chunk size
//!
//! for (i, fib) in results.iter().enumerate() {
//!     println!("F({}) = {}", start + i as u64, fib);
//! }
//! ```

pub mod algo;
pub mod config;
pub mod iterators;
pub mod types;

// Re-export types
pub use types::{Algorithm, FibError, FibNumber, FibOps};

// Re-export algorithms
pub use algo::{
    fibonacci, fibonacci_adaptive, fibonacci_fast_doubling, fibonacci_fft, fibonacci_parallel,
    try_fibonacci_adaptive,
};

// Re-export deprecated alias for backward compatibility
#[allow(deprecated)]
pub use algo::fibonacci_matrix;

pub use iterators::{FibIter, FibRange};

// Re-export helper for initializing system
pub use algo::parallel::get_parallel_threshold;

use rayon::prelude::*;

/// Estimates the memory usage (in bytes) required to store the result of F(n).
///
/// This uses the approximation that F(n) has approximately $n \times \log_{10}(\phi) \approx 0.2089 \cdot n$ decimal digits,
/// or $n \times \log_2(\phi) \approx 0.6942 \cdot n$ bits.
///
/// The estimation adds a 10% safety margin for object overhead.
///
/// # Arguments
/// * `n` - The index of the Fibonacci number.
///
/// # Returns
/// Estimated size in bytes.
pub fn estimate_memory_bytes(n: u64) -> u64 {
    if n == 0 {
        return 0;
    }
    // bits = n * log2(phi) ~= n * 0.6942419
    // bytes = bits / 8
    // bytes ~= n * 0.08678
    // With safety margin (x1.1): n * 0.095458
    // Integer approximation: n / 10 is decent (0.1), slightly overestimating is safer.
    // Let's use floating point for precision if possible, or high prec integer math.
    // 0.08678 * 1.1 ~= 0.095
    // n * 95 / 1000
    // Check overflow? n is u64.
    // If n is huge, this calc might overflow if not careful.
    let n_u128 = n as u128;
    let bytes = (n_u128 * 95) / 1000;
    bytes as u64
}

/// Pre-warms the system by calibrating thresholds and initializing thread-local resources.
///
/// This function is designed to prevent first-call latency spikes in production environments,
/// such as API servers or CLI tools. It performs the following actions:
/// 1.  **Calibration**: Runs a micro-benchmark to determine the optimal threshold for switching to parallel algorithms (`algo::parallel::get_parallel_threshold`).
/// 2.  **Thread Pool Initialization**: Wakes up the Rayon thread pool.
/// 3.  **FFT Planner Initialization**: Initializes thread-local FFT planners on all worker threads to avoid lazy initialization overhead during the first FFT-based calculation.
///
/// # Usage
///
/// Call this function once at the start of your application (e.g., in `main`).
///
/// ```rust
/// fn main() {
///     fibrust_core::prewarm_system();
///     // ... application logic ...
/// }
/// ```
pub fn prewarm_system() {
    // 1. Force calibration
    algo::parallel::get_parallel_threshold();

    // 2. Pre-warm Rayon thread pool and FFT planners
    let _ = rayon::join(
        || {
            algo::fft::prewarm_fft_planner();
        },
        || {
            let threads = rayon::current_num_threads();
            if threads > 1 {
                (0..threads).into_par_iter().for_each(|_| {
                    algo::fft::prewarm_fft_planner();
                });
            }
        },
    );
}

/// Runs all available algorithms in parallel for a given `n` and returns their results.
///
/// This function is primarily used for benchmarking, testing, or verifying consistency across
/// different algorithm implementations. It runs:
/// - Fast Doubling
/// - Parallel Fast Doubling
/// - FFT-based Doubling
///
/// # Arguments
///
/// * `n` - The index of the Fibonacci number to compute.
///
/// # Returns
///
/// A vector of tuples, where each tuple contains:
/// - `String`: The name of the algorithm.
/// - `Duration`: The time taken to compute the result.
/// - `UBig`: The computed Fibonacci number.
///
/// # Example
///
/// ```rust
/// use fibrust_core::run_all_parallel;
///
/// let n = 1000;
/// let results = run_all_parallel(n);
///
/// for (name, duration, result) in results {
///     println!("Algorithm: {}, Time: {:?}, Result bits: {}", name, duration, result.bit_len());
/// }
/// ```
pub fn run_all_parallel(n: u64) -> Vec<(String, std::time::Duration, FibNumber)> {
    let results: std::sync::Mutex<Vec<(String, std::time::Duration, FibNumber)>> =
        std::sync::Mutex::new(Vec::new());

    rayon::scope(|s| {
        s.spawn(|_| {
            let start = std::time::Instant::now();
            let res = algo::fibonacci_fast_doubling(n);
            let duration = start.elapsed();
            results
                .lock()
                .unwrap()
                .push(("Fast Doubling".to_string(), duration, res));
        });
        s.spawn(|_| {
            let start = std::time::Instant::now();
            let res = algo::fibonacci_parallel(n);
            let duration = start.elapsed();
            results
                .lock()
                .unwrap()
                .push(("Parallel".to_string(), duration, res));
        });
        s.spawn(|_| {
            let start = std::time::Instant::now();
            let res = algo::fibonacci_fft(n);
            let duration = start.elapsed();
            results
                .lock()
                .unwrap()
                .push(("FFT".to_string(), duration, res));
        });
    });

    results.into_inner().unwrap()
}

/// Computes a range of Fibonacci numbers $[F(\text{start}), \dots, F(\text{end}-1)]$ in parallel.
///
/// This function splits the range into chunks and processes them in parallel using Rayon.
/// It utilizes the `FibRange` iterator which allows for $O(\log k)$ initialization for each chunk start,
/// followed by $O(1)$ sequential iteration.
///
/// # Arguments
///
/// * `start` - The starting index (inclusive).
/// * `end` - The ending index (exclusive).
/// * `_chunk_size` - **Ignored**. Rayon automatically handles load balancing and chunking strategies. Kept for API compatibility.
///
/// # Returns
///
/// A `Vec<UBig>` containing the Fibonacci numbers in the specified range.
///
/// # Example
///
/// ```rust
/// use fibrust_core::fib_range_parallel;
///
/// let results = fib_range_parallel(10, 15, 0);
/// assert_eq!(results.len(), 5);
/// assert_eq!(results[0], 55u32.into()); // F(10)
/// ```
pub fn fib_range_parallel(start: u64, end: u64, _chunk_size: usize) -> Vec<FibNumber> {
    // The _chunk_size parameter is currently ignored as Rayon handles splitting intelligently,
    // but kept for API compatibility.
    if start >= end {
        return Vec::new();
    }

    FibRange::new(start, end).into_par_iter().collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    // ========================================================================
    // Tests for prewarm_system
    // ========================================================================

    #[test]
    fn prewarm_system_no_panic() {
        // Should complete without panicking
        prewarm_system();
    }

    #[test]
    fn prewarm_system_idempotent() {
        // Multiple calls should not cause issues
        prewarm_system();
        prewarm_system();
        prewarm_system();
    }

    // ========================================================================
    // Tests for run_all_parallel
    // ========================================================================

    #[test]
    fn run_all_parallel_returns_three_results() {
        let results = run_all_parallel(10);
        assert_eq!(results.len(), 3, "Should return 3 algorithm results");
    }

    #[test]
    fn run_all_parallel_consistent_results() {
        let results = run_all_parallel(100);

        // All algorithms should produce the same Fibonacci number
        let first_result = &results[0].2;
        for (name, _, result) in &results {
            assert_eq!(
                result, first_result,
                "Algorithm {} produced different result",
                name
            );
        }
    }

    #[test]
    fn run_all_parallel_has_all_algorithms() {
        let results = run_all_parallel(50);
        let names: Vec<&str> = results.iter().map(|(n, _, _)| n.as_str()).collect();

        assert!(names
            .iter()
            .any(|n| n.contains("Fast") || n.contains("Doubling")));
        assert!(names.iter().any(|n| n.contains("Parallel")));
        assert!(names.iter().any(|n| n.contains("FFT")));
    }

    #[test]
    fn run_all_parallel_known_value() {
        let results = run_all_parallel(10);

        // F(10) = 55
        let expected = FibNumber::from(55u32);
        for (name, _, result) in &results {
            assert_eq!(result, &expected, "Algorithm {} produced wrong value", name);
        }
    }

    // ========================================================================
    // Tests for fib_range_parallel
    // ========================================================================

    #[test]
    fn fib_range_parallel_empty_when_start_equals_end() {
        let results = fib_range_parallel(10, 10, 0);
        assert!(results.is_empty());
    }

    #[test]
    fn fib_range_parallel_empty_when_start_greater() {
        let results = fib_range_parallel(100, 50, 0);
        assert!(results.is_empty());
    }

    #[test]
    fn fib_range_parallel_single_element() {
        let results = fib_range_parallel(10, 11, 0);
        assert_eq!(results.len(), 1);
        assert_eq!(results[0], FibNumber::from(55u32)); // F(10)
    }

    #[test]
    fn fib_range_parallel_auto_chunk_size() {
        // chunk_size = 0 means auto-calculate
        let results = fib_range_parallel(0, 100, 0);
        assert_eq!(results.len(), 100);
        assert_eq!(results[0], FibNumber::from(0u32)); // F(0)
        assert_eq!(results[10], FibNumber::from(55u32)); // F(10)
    }

    #[test]
    fn fib_range_parallel_explicit_chunk_size() {
        let results = fib_range_parallel(0, 50, 10);
        assert_eq!(results.len(), 50);
    }

    #[test]
    fn fib_range_parallel_matches_sequential() {
        let parallel_results = fib_range_parallel(0, 50, 0);
        let sequential_results: Vec<FibNumber> = FibRange::new(0, 50).collect();

        assert_eq!(parallel_results.len(), sequential_results.len());
        for (i, (par, seq)) in parallel_results
            .iter()
            .zip(sequential_results.iter())
            .enumerate()
        {
            assert_eq!(par, seq, "Mismatch at index {}", i);
        }
    }

    #[test]
    fn fib_range_parallel_larger_range() {
        let results = fib_range_parallel(1000, 1100, 0);
        assert_eq!(results.len(), 100);

        // Verify first and last values
        assert_eq!(results[0], algo::fibonacci_fast_doubling(1000));
        assert_eq!(results[99], algo::fibonacci_fast_doubling(1099));
    }
}
