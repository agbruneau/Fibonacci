//! Progress tracking for O(log n) algorithms.
//!
//! This module implements a progress estimation system based on the geometric work
//! progression of algorithms that iterate over bits (like Fast Doubling or Matrix Exponentiation).
//!
//! # Mathematical Model
//!
//! The work done at each step $i$ (iterating from `num_bits-1` down to 0) is proportional to $4^i$.
//! This reflects that operations on larger numbers (which occur at lower bit indices as we build up the number)
//! are exponentially more expensive than operations on smaller numbers (higher bit indices).
//!
//! - **Total Work**: The sum of the geometric series $\sum_{i=0}^{n-1} 4^i = \frac{4^n - 1}{3}$.
//! - **Step Work**: The work for step $i$ is $4^{n-1-i}$.
//!
//! This model provides a much more accurate progress bar than linear interpolation, as the final steps
//! of the calculation take significantly longer than the initial steps.

/// Function type for reporting progress updates.
pub type ProgressReporter = Box<dyn Fn(f64) + Send + Sync>;

/// Calculates the estimated total work units for an algorithm operating on `num_bits`.
///
/// Based on geometric series sum: $\frac{4^n - 1}{3}$.
#[inline]
pub fn calc_total_work(num_bits: u32) -> f64 {
    if num_bits == 0 {
        return 0.0;
    }
    // (4^n - 1) / 3
    (4_f64.powi(num_bits as i32) - 1.0) / 3.0
}

/// Pre-computes powers of 4 to avoid repeated expensive exponentiation calls.
///
/// Returns a slice where `index` corresponds to $4^{index}$.
/// We need powers up to `num_bits - 1`.
pub fn precompute_powers_4(num_bits: u32) -> Vec<f64> {
    if num_bits == 0 {
        return Vec::new();
    }

    let mut powers = Vec::with_capacity(num_bits as usize);
    let mut current = 1.0;

    powers.push(current);
    for _ in 1..num_bits {
        current *= 4.0;
        powers.push(current);
    }

    powers
}

/// Reports progress for a single step of the algorithm.
///
/// # Arguments
///
/// * `reporter` - Callback to invoke with progress (0.0 to 1.0).
/// * `last_reported` - Mutable reference to the last reported progress value.
/// * `total_work` - Total work calculated by `calc_total_work`.
/// * `work_done` - Accumulated work done so far.
/// * `bit_index` - Current bit index being processed (iterating from `num_bits-1` down to 0).
/// * `num_bits` - Total number of bits.
/// * `powers` - Precomputed powers of 4.
///
/// # Returns
///
/// The updated `work_done` value.
pub fn report_step_progress(
    reporter: &Option<ProgressReporter>,
    last_reported: &mut f64,
    total_work: f64,
    work_done: f64,
    bit_index: u32,
    num_bits: u32,
    powers: &[f64],
) -> f64 {
    // If no reporter or invalid state, just return work_done without updates
    if reporter.is_none() || total_work <= 0.0 || num_bits == 0 {
        return work_done;
    }

    // Calculate work for this step: 4^(num_bits - 1 - bit_index)
    // Note: bit_index goes from (num_bits-1) -> 0
    // So exponent goes from 0 -> (num_bits-1)
    let power_idx = (num_bits - 1 - bit_index) as usize;

    // Safety check for bounds
    let step_work = if power_idx < powers.len() {
        powers[power_idx]
    } else {
        // Fallback if precompute was insufficient (should not happen in correct usage)
        4_f64.powi(power_idx as i32)
    };

    let current_total_done = work_done + step_work;
    let mut current_progress = current_total_done / total_work;

    // Clamp to valid range [0.0, 1.0]
    if current_progress > 1.0 {
        current_progress = 1.0;
    }

    // Report if:
    // 1. It's the very first step (bit_index == num_bits - 1)
    // 2. It's the very last step (bit_index == 0)
    // 3. Progress has increased by at least 1% since last report
    let threshold = 0.01;
    let is_start = bit_index == num_bits - 1;
    let is_end = bit_index == 0;
    let significant_change = (current_progress - *last_reported) >= threshold;

    if is_start || is_end || significant_change {
        if let Some(rpt) = reporter {
            rpt(current_progress);
        }
        *last_reported = current_progress;
    }

    current_total_done
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Arc, Mutex};

    #[test]
    fn test_calc_total_work() {
        assert_eq!(calc_total_work(0), 0.0);
        assert_eq!(calc_total_work(1), 1.0); // (4^1 - 1)/3 = 1
        assert_eq!(calc_total_work(2), 5.0); // (4^2 - 1)/3 = 15/3 = 5
        assert_eq!(calc_total_work(3), 21.0); // (4^3 - 1)/3 = 63/3 = 21
    }

    #[test]
    fn test_precompute_powers() {
        assert_eq!(precompute_powers_4(0), Vec::<f64>::new());
        assert_eq!(precompute_powers_4(1), vec![1.0]);
        assert_eq!(precompute_powers_4(3), vec![1.0, 4.0, 16.0]);
    }

    #[test]
    fn test_progress_monotonicity_and_bounds() {
        let num_bits = 10;
        let total_work = calc_total_work(num_bits);
        let powers = precompute_powers_4(num_bits);

        let last_progress = Arc::new(Mutex::new(-1.0));
        let last_progress_clone = last_progress.clone();

        // Custom reporter that asserts properties
        let reporter: Option<ProgressReporter> = Some(Box::new(move |p| {
            let mut last = last_progress_clone.lock().unwrap();

            // Check bounds
            assert!(p >= 0.0, "Progress below 0: {}", p);
            assert!(p <= 1.0, "Progress above 1: {}", p);

            // Check monotonicity (strict increase or equal if first report)
            assert!(p >= *last, "Progress decreased: {} -> {}", *last, p);

            *last = p;
        }));

        let mut work_done = 0.0;
        let mut last_reported = -1.0;

        // Simulate loop from MSB to LSB
        for i in (0..num_bits).rev() {
            work_done = report_step_progress(
                &reporter,
                &mut last_reported,
                total_work,
                work_done,
                i,
                num_bits,
                &powers,
            );
        }

        // Final check
        let final_progress = *last_progress.lock().unwrap();
        assert!(
            final_progress >= 0.99,
            "Did not reach completion: {}",
            final_progress
        );
    }
}
