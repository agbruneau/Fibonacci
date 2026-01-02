//! Configuration constants and tuning parameters for Fibonacci algorithms.
//!
//! This module centralizes all algorithm thresholds to facilitate tuning
//! and maintain consistency across the codebase.
//!
//! # Threshold Justifications
//!
//! These values are derived from empirical benchmarks on modern hardware
//! (24-core Ryzen 9, Windows) and represent crossover points where
//! algorithm performance characteristics change.

/// Algorithm selection thresholds for `fibonacci_adaptive`.
pub mod thresholds {
    /// Threshold for switching from Fast Doubling to Parallel Fast Doubling.
    ///
    /// For $n < 40,000$, sequential Fast Doubling is optimal because:
    /// - Thread pool overhead dominates for small inputs
    /// - Single-core performance is sufficient
    ///
    /// Empirically measured crossover point.
    pub const PARALLEL_CROSSOVER: u64 = 40_000;

    /// Threshold for switching from Parallel Fast Doubling to FFT-based multiplication.
    ///
    /// For $n \ge 200,000$, FFT multiplication wins because:
    /// - $O(n \log n)$ FFT beats $O(n^{1.585})$ Karatsuba for huge integers
    /// - Numbers exceed ~140k bits where FFT becomes asymptotically faster
    ///
    /// Empirically measured crossover point.
    pub const FFT_CROSSOVER: u64 = 200_000;

    /// Bit-length threshold for using FFT vs standard multiplication within an algorithm.
    ///
    /// When computing intermediate products, if operands exceed this bit length,
    /// FFT-based multiplication is used instead of the default library multiplication.
    ///
    /// Set conservatively to ensure FFT overhead is amortized.
    pub const FFT_BIT_THRESHOLD: usize = 50_000;
}

/// Memory and safety limits.
pub mod limits {
    /// Maximum safe input value before memory estimation kicks in.
    ///
    /// Beyond this, we require explicit memory checks.
    /// Set to 1 trillion ($10^{12}$) as a conservative upper bound.
    pub const MAX_SAFE_N: u64 = 1_000_000_000_000;

    /// Maximum memory allocation (in bytes) before rejecting a request.
    ///
    /// Default: 8 GB. Prevents out-of-memory crashes on extreme inputs.
    pub const SAFE_MEMORY_BYTES: u64 = 8 * 1024 * 1024 * 1024;
}

/// FFT-specific tuning parameters.
pub mod fft {
    /// Number of bits per digit for base conversion in FFT multiplication.
    ///
    /// # Precision Constraint
    ///
    /// Must satisfy: $2 \times \text{BASE\_BITS} + \log_2(\text{fft\_size}) < 53$
    ///
    /// For massive inputs ($n > 2 \times 10^9$), FFT size can reach $2^{28}$.
    /// With `BASE_BITS = 13`: $2 \times 13 + 28 = 54 > 53$ (unsafe)
    /// With `BASE_BITS = 12`: $2 \times 12 + 28 = 52 < 53$ (safe)
    ///
    /// Default uses 13 for most inputs, 12 for massive inputs (>100M bits).
    pub const BASE_BITS_DEFAULT: usize = 13;

    /// Safer base bits for massive numbers (>100M bits combined).
    pub const BASE_BITS_MASSIVE: usize = 12;

    /// Bit threshold for switching to the more conservative base.
    pub const MASSIVE_THRESHOLD: usize = 100_000_000;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn thresholds_are_ordered() {
        assert!(
            thresholds::PARALLEL_CROSSOVER < thresholds::FFT_CROSSOVER,
            "PARALLEL_CROSSOVER must be less than FFT_CROSSOVER"
        );
    }

    #[test]
    fn fft_precision_constraint() {
        // Verify BASE_BITS values satisfy precision constraint
        // 2*BASE_BITS + log2(max_fft_size) < 53
        // max_fft_size ≈ 2^28 for n=2e9
        let max_fft_log2 = 28;

        let precision_default = 2 * fft::BASE_BITS_DEFAULT + max_fft_log2;
        let precision_massive = 2 * fft::BASE_BITS_MASSIVE + max_fft_log2;

        // BASE_BITS_DEFAULT (13) is unsafe for massive inputs but ok for normal
        assert!(
            precision_massive < 53,
            "BASE_BITS_MASSIVE must satisfy precision constraint"
        );
        // Note: DEFAULT may exceed 53 for massive inputs, which is why we switch
    }

    #[test]
    fn limits_are_reasonable() {
        // MAX_SAFE_N should be at least 1 billion
        assert!(limits::MAX_SAFE_N >= 1_000_000_000);
        // SAFE_MEMORY_BYTES should be at least 1 GB
        assert!(limits::SAFE_MEMORY_BYTES >= 1024 * 1024 * 1024);
    }
}
