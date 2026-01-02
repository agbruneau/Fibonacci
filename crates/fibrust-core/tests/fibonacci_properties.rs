//! Property-based tests for Fibonacci algorithms.
//!
//! These tests verify mathematical invariants of the Fibonacci sequence
//! across all implemented algorithms using proptest.

use fibrust_core::{
    fibonacci_adaptive, fibonacci_fast_doubling, fibonacci_fft, fibonacci_parallel, FibIter,
    FibRange,
};
use ibig::UBig;
use proptest::prelude::*;

// ============================================================================
// Property: F(n) + F(n+1) = F(n+2) (Recurrence relation)
// ============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn recurrence_relation_fast_doubling(n in 0u64..10_000) {
        let f_n = fibonacci_fast_doubling(n);
        let f_n1 = fibonacci_fast_doubling(n + 1);
        let f_n2 = fibonacci_fast_doubling(n + 2);

        prop_assert_eq!(&f_n + &f_n1, f_n2, "F({}) + F({}) should equal F({})", n, n+1, n+2);
    }

    #[test]
    fn recurrence_relation_parallel(n in 0u64..1_000) {
        let f_n = fibonacci_parallel(n);
        let f_n1 = fibonacci_parallel(n + 1);
        let f_n2 = fibonacci_parallel(n + 2);

        prop_assert_eq!(&f_n + &f_n1, f_n2);
    }
}

// ============================================================================
// Property: All algorithms produce identical results
// ============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(50))]

    #[test]
    fn algorithms_consistent(n in 0u64..50_000) {
        let fd = fibonacci_fast_doubling(n);
        let par = fibonacci_parallel(n);

        prop_assert_eq!(&fd, &par, "Fast Doubling and Parallel differ at n={}", n);
    }

    #[test]
    fn algorithms_consistent_with_fft(n in 0u64..10_000) {
        let fd = fibonacci_fast_doubling(n);
        let fft = fibonacci_fft(n);

        prop_assert_eq!(&fd, &fft, "Fast Doubling and FFT differ at n={}", n);
    }

    #[test]
    fn adaptive_consistent_with_fast_doubling(n in 0u64..50_000) {
        let adaptive = fibonacci_adaptive(n);
        let fd = fibonacci_fast_doubling(n);

        prop_assert_eq!(&adaptive, &fd, "Adaptive and Fast Doubling differ at n={}", n);
    }
}

// ============================================================================
// Property: F(n) is monotonically increasing for n >= 1
// ============================================================================

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn monotonic_increasing(n in 1u64..10_000) {
        let f_n = fibonacci_fast_doubling(n);
        let f_n1 = fibonacci_fast_doubling(n + 1);

        prop_assert!(f_n1 > f_n, "F({}) should be greater than F({})", n+1, n);
    }
}

// ============================================================================
// Base cases: F(0) = 0, F(1) = 1, F(2) = 1
// ============================================================================

#[test]
fn base_cases_fast_doubling() {
    assert_eq!(fibonacci_fast_doubling(0), UBig::from(0u32));
    assert_eq!(fibonacci_fast_doubling(1), UBig::from(1u32));
    assert_eq!(fibonacci_fast_doubling(2), UBig::from(1u32));
}

#[test]
fn base_cases_parallel() {
    assert_eq!(fibonacci_parallel(0), UBig::from(0u32));
    assert_eq!(fibonacci_parallel(1), UBig::from(1u32));
    assert_eq!(fibonacci_parallel(2), UBig::from(1u32));
}

#[test]
fn base_cases_fft() {
    assert_eq!(fibonacci_fft(0), UBig::from(0u32));
    assert_eq!(fibonacci_fft(1), UBig::from(1u32));
    assert_eq!(fibonacci_fft(2), UBig::from(1u32));
}

// ============================================================================
// Known values (regression tests)
// ============================================================================

#[test]
fn known_values() {
    // F(10) = 55
    assert_eq!(fibonacci_fast_doubling(10), UBig::from(55u32));

    // F(20) = 6765
    assert_eq!(fibonacci_fast_doubling(20), UBig::from(6765u32));

    // F(50) = 12586269025
    assert_eq!(fibonacci_fast_doubling(50), UBig::from(12586269025u64));

    // F(100) - known value
    let f100 = fibonacci_fast_doubling(100);
    assert_eq!(f100.to_string(), "354224848179261915075");
}

// ============================================================================
// Iterator tests
// ============================================================================

#[test]
fn fib_range_produces_correct_sequence() {
    let range: Vec<UBig> = FibRange::new(0, 10).collect();

    assert_eq!(range.len(), 10);
    assert_eq!(range[0], UBig::from(0u32)); // F(0)
    assert_eq!(range[1], UBig::from(1u32)); // F(1)
    assert_eq!(range[2], UBig::from(1u32)); // F(2)
    assert_eq!(range[3], UBig::from(2u32)); // F(3)
    assert_eq!(range[4], UBig::from(3u32)); // F(4)
    assert_eq!(range[5], UBig::from(5u32)); // F(5)
    assert_eq!(range[9], UBig::from(34u32)); // F(9)
}

#[test]
fn fib_range_starting_midway() {
    let range: Vec<UBig> = FibRange::new(10, 15).collect();

    assert_eq!(range.len(), 5);
    assert_eq!(range[0], fibonacci_fast_doubling(10));
    assert_eq!(range[4], fibonacci_fast_doubling(14));
}

#[test]
fn fib_iter_infinite() {
    let first_10: Vec<UBig> = FibIter::new().take(10).collect();

    assert_eq!(first_10.len(), 10);
    assert_eq!(first_10[0], UBig::from(0u32));
    assert_eq!(first_10[9], UBig::from(34u32));
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(20))]

    #[test]
    fn fib_range_matches_individual_calculations(start in 0u64..1000, len in 1u64..100) {
        let end = start + len;
        let range: Vec<UBig> = FibRange::new(start, end).collect();

        for (i, fib) in range.iter().enumerate() {
            let expected = fibonacci_fast_doubling(start + i as u64);
            prop_assert_eq!(fib, &expected, "FibRange mismatch at index {}", start + i as u64);
        }
    }
}

// ============================================================================
// Edge cases
// ============================================================================

#[test]
fn empty_range() {
    let range: Vec<UBig> = FibRange::new(10, 10).collect();
    assert!(range.is_empty());

    let range2: Vec<UBig> = FibRange::new(100, 50).collect();
    assert!(range2.is_empty());
}

#[test]
fn large_index_consistency() {
    // Test at F(100,000) - all algorithms should match
    let n = 100_000;
    let fd = fibonacci_fast_doubling(n);
    let par = fibonacci_parallel(n);
    let fft = fibonacci_fft(n);

    assert_eq!(fd, par, "Fast Doubling and Parallel differ at n={}", n);
    assert_eq!(fd, fft, "Fast Doubling and FFT differ at n={}", n);
}
